package dashboard

import (
	"context"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/nostr"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/opslog"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/receiptstore"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/makerad"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EnginePillar is MWEB bridge + receipt vault surface.
type EnginePillar struct {
	BridgeEnabled  bool   `json:"bridgeEnabled"`
	BridgeDir      string `json:"bridgeDir,omitempty"`
	ReceiptCount   int64  `json:"receiptCount"`
	ReceiptCountOK bool   `json:"receiptCountOk"`
	ReceiptErr     string `json:"receiptErr,omitempty"`
}

// DaemonInfo is how mlnd is configured (no secrets).
type DaemonInfo struct {
	AutoDefend          bool     `json:"autoDefend"`
	AutoDefendDryRun    bool     `json:"autoDefendDryRun"`
	NostrBroadcaster    bool     `json:"nostrBroadcaster"`
	CoinswapdBridge     bool     `json:"coinswapdBridge"`
	AutoDefendWarning   string   `json:"autoDefendWarning,omitempty"`
	RelaysConfigured    []string `json:"relaysConfigured,omitempty"`
}

// NetworkStatus aggregates Nostr relay health and ad verification.
type NetworkStatus struct {
	NetworkPillarSelfCheck
	LastPublish       nostr.LastPublishSnapshot `json:"lastPublish,omitempty"`
	IntendedAdContent string                    `json:"intendedAdContent,omitempty"`
}

// ConnectionHints is trust UX: show which LitVM RPC the daemon uses (truncated).
type ConnectionHints struct {
	LitVMRPCLabel string `json:"litvmRpcLabel,omitempty"`
}

// Status is the full GET /api/v1/status payload.
type Status struct {
	Chain              ChainPillar              `json:"chain"`
	Network            NetworkStatus            `json:"network"`
	Engine             EnginePillar             `json:"engine"`
	Daemon             DaemonInfo               `json:"daemon"`
	Connection         ConnectionHints          `json:"connection"`
	GrievanceNarrative *GrievanceNarrativeView  `json:"grievanceNarrative,omitempty"`
}

// StatusDeps bundles inputs for BuildStatus.
type StatusDeps struct {
	EthClient *ethclient.Client

	RegistryAddr string
	CourtAddr    string
	OperatorAddr string
	ChainID      string
	Relays       []string

	Broadcaster *nostr.Broadcaster

	BridgeEnabled bool
	BridgeDir     string

	Store *receiptstore.Store

	AutoDefend       bool
	AutoDefendDryRun bool

	Ops *opslog.Log

	// LitVMRPCURL is MLND_WS_URL (or default); shown truncated in the UI for “is anything wired?” confidence.
	LitVMRPCURL string
}

// BuildStatus aggregates chain, network, engine, and daemon fields.
func BuildStatus(ctx context.Context, d StatusDeps) Status {
	out := Status{
		Daemon: DaemonInfo{
			AutoDefend:       d.AutoDefend,
			AutoDefendDryRun: d.AutoDefendDryRun,
			NostrBroadcaster: d.Broadcaster != nil,
			CoinswapdBridge:  d.BridgeEnabled,
			RelaysConfigured: append([]string(nil), d.Relays...),
		},
		Engine: EnginePillar{
			BridgeEnabled: d.BridgeEnabled,
			BridgeDir:     d.BridgeDir,
		},
	}

	if !d.AutoDefend {
		out.Daemon.AutoDefendWarning = "Auto-defend is OFF. If a grievance is filed, you must submit defendGrievance manually before the deadline or risk slashing."
	}

	if label := ShortenRPCDisplay(d.LitVMRPCURL); label != "" {
		out.Connection.LitVMRPCLabel = label
	}
	out.GrievanceNarrative = LatestGrievanceNarrative(d.Ops)

	maker := common.HexToAddress(d.OperatorAddr)
	out.Chain = readChainPillar(ctx, d.EthClient, d.RegistryAddr, d.CourtAddr, maker)

	if d.Store != nil {
		n, err := d.Store.CountReceipts()
		out.Engine.ReceiptCount = n
		out.Engine.ReceiptCountOK = err == nil
		if err != nil {
			out.Engine.ReceiptErr = err.Error()
		}
	}

	dTag := makerad.DTag(strings.TrimSpace(d.ChainID), strings.ToLower(maker.Hex()))
	out.Network.DTag = dTag
	out.Network.LocalSwapX25519Expected = ""
	if d.Broadcaster != nil {
		cfg := d.Broadcaster.Config()
		out.Network.LocalSwapX25519Expected = cfg.SwapX25519PubHex
		if ev, err := d.Broadcaster.BuildMakerAdEvent(time.Now().UTC()); err == nil {
			out.Network.IntendedAdContent = ev.Content
		}
		out.Network.LastPublish = d.Broadcaster.LastPublish()
	}

	if len(d.Relays) == 0 || strings.TrimSpace(d.ChainID) == "" {
		out.Network.Error = "Relay self-check skipped: set MLND_NOSTR_RELAYS and MLND_LITVM_CHAIN_ID to verify your maker ad is visible (optional for dashboard-only)."
		return out
	}

	ev, relayURL, err := FetchLatestMakerAdForDTag(ctx, d.Relays, dTag, d.ChainID, d.RegistryAddr, d.CourtAddr)
	if err != nil {
		out.Network.Error = err.Error()
		return out
	}

	out.Network.EventFound = true
	out.Network.EventID = ev.ID
	out.Network.CreatedAt = int64(ev.CreatedAt)
	out.Network.ContentJSON = ev.Content
	out.Network.RelayQueried = relayURL

	parsed, err := makerad.ParseAd(ev)
	if err != nil {
		out.Network.Error = "parse ad: " + err.Error()
		return out
	}
	if parsed.Operator != maker {
		out.Network.Error = "d-tag operator does not match MLND_OPERATOR_ADDR"
		return out
	}

	out.Network.SwapX25519FromRelay = strings.TrimSpace(strings.ToLower(parsed.Content.SwapX25519PubHex))
	if exp := strings.TrimSpace(strings.ToLower(out.Network.LocalSwapX25519Expected)); exp != "" && out.Network.SwapX25519FromRelay != exp {
		out.Network.SwapKeyDrift = true
	}

	regAddr := common.HexToAddress(d.RegistryAddr)
	vm, err := VerifyMakerOnChain(ctx, d.EthClient, regAddr, maker, ev.PubKey)
	if err != nil {
		out.Network.VerifyReason = err.Error()
		return out
	}
	out.Network.NostrKeyHashMatch = vm.NostrKeyHashMatch
	out.Network.RegistryOK = vm.OK
	out.Network.VerifyReason = vm.Reason
	return out
}
