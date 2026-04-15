package scout

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/registry"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/makerad"
	"github.com/ethereum/go-ethereum/common"
	gnostr "github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

// Config drives Nostr fetch + LitVM verification.
type Config struct {
	Relays          []string
	RPCHTTP         string
	ChainID         string
	RegistryAddr    common.Address
	GrievanceCourt  string // optional; if set, must match content.litvm.grievanceCourt (lowercase)
	Timeout         time.Duration
	QuietRejections bool
	// AuthNsec is an optional Nostr private key (nsec1 or 64 hex) for NIP-42 relay AUTH.
	// When set, Scout authenticates to relays that require it before subscribing.
	AuthNsec string
}

// VerifiedMaker is a maker that passed signature, wire, deployment filter, and registry checks.
type VerifiedMaker struct {
	Operator  common.Address `json:"operator"`
	Tor       string         `json:"tor,omitempty"`
	Stake     string         `json:"stake"` // decimal string (wei-style units)
	MinStake  string         `json:"minStake"`
	EventID   string         `json:"eventId"`
	CreatedAt int64          `json:"createdAt"`
	RelayURL  string         `json:"relay,omitempty"`
	FeeMinSat uint64         `json:"feeMinSat,omitempty"`
	FeeMaxSat uint64         `json:"feeMaxSat,omitempty"`
	// SwapX25519PubHex is optional; when set on all hops in a route, takers can build real onions (see research/COINSWAPD_MLN_FORK_SPEC.md).
	SwapX25519PubHex string `json:"swapX25519PubHex,omitempty"`
	RegistryOK       bool   `json:"registryOk"`
	Local            bool   `json:"local,omitempty"` // set by takerflow when operator matches configured maker key
}

// Rejection records why an event was not listed as verified.
type Rejection struct {
	EventID string `json:"eventId,omitempty"`
	Reason  string `json:"reason"`
}

// Result is the full scout output.
type Result struct {
	Verified []VerifiedMaker `json:"verified"`
	Rejected []Rejection     `json:"rejected"`
}

func normAddr(a string) string {
	s := strings.TrimSpace(strings.ToLower(a))
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	return s
}

func parseAuthKey(raw string) (secHex, pubHex string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("empty auth key")
	}
	if strings.HasPrefix(raw, "nsec1") {
		prefix, val, err := nip19.Decode(raw)
		if err != nil {
			return "", "", err
		}
		if prefix != "nsec" {
			return "", "", fmt.Errorf("expected nsec, got %q", prefix)
		}
		sk, ok := val.(string)
		if !ok || len(sk) != 64 {
			return "", "", fmt.Errorf("invalid nsec decode")
		}
		secHex = sk
	} else {
		key := strings.TrimPrefix(strings.TrimPrefix(raw, "0x"), "0X")
		if len(key) != 64 {
			return "", "", fmt.Errorf("expected 64 hex chars or nsec1…")
		}
		secHex = strings.ToLower(key)
	}
	pubHex, err = gnostr.GetPublicKey(secHex)
	if err != nil {
		return "", "", fmt.Errorf("derive pubkey: %w", err)
	}
	return secHex, pubHex, nil
}

// Run collects maker ads from relays, dedupes by operator (latest created_at), verifies, and returns structured results.
func Run(ctx context.Context, cfg Config) (*Result, error) {
	if len(cfg.Relays) == 0 {
		return nil, fmt.Errorf("scout: no relays")
	}
	if strings.TrimSpace(cfg.RPCHTTP) == "" {
		return nil, fmt.Errorf("scout: MLN_LITVM_HTTP_URL is required for registry verification")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	subCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	filter := gnostr.Filter{
		Kinds: []int{makerad.KindMakerAd},
		Tags:  gnostr.TagMap{"t": []string{makerad.TagTMakerAd}},
	}

	var poolOpts []gnostr.PoolOption
	if cfg.AuthNsec != "" {
		secHex, pubHex, err := parseAuthKey(cfg.AuthNsec)
		if err != nil {
			return nil, fmt.Errorf("scout: auth key: %w", err)
		}
		poolOpts = append(poolOpts, gnostr.WithAuthHandler(func(ev *gnostr.Event) error {
			ev.PubKey = pubHex
			return ev.Sign(secHex)
		}))
	}
	pool := gnostr.NewSimplePool(subCtx, poolOpts...)
	ch := pool.SubManyEose(subCtx, cfg.Relays, gnostr.Filters{filter})

	// Latest event per operator hex (lowercase).
	latest := make(map[string]gnostr.IncomingEvent)
	rejections := make([]Rejection, 0)

	for ie := range ch {
		ev := ie.Event
		if ev == nil {
			continue
		}
		if err := makerad.VerifySignature(ev); err != nil {
			if !cfg.QuietRejections {
				rejections = append(rejections, Rejection{EventID: ev.ID, Reason: "signature: " + err.Error()})
			}
			continue
		}
		parsed, err := makerad.ParseAd(ev)
		if err != nil {
			if !cfg.QuietRejections {
				rejections = append(rejections, Rejection{EventID: ev.ID, Reason: "parse: " + err.Error()})
			}
			continue
		}
		if parsed.DTagChainID != cfg.ChainID {
			if !cfg.QuietRejections {
				rejections = append(rejections, Rejection{EventID: ev.ID, Reason: "chainId mismatch"})
			}
			continue
		}
		if normAddr(parsed.Content.Litvm.Registry) != normAddr(cfg.RegistryAddr.Hex()) {
			if !cfg.QuietRejections {
				rejections = append(rejections, Rejection{EventID: ev.ID, Reason: "registry address mismatch"})
			}
			continue
		}
		if cfg.GrievanceCourt != "" && normAddr(parsed.Content.Litvm.GrievanceCourt) != normAddr(cfg.GrievanceCourt) {
			if !cfg.QuietRejections {
				rejections = append(rejections, Rejection{EventID: ev.ID, Reason: "grievance court mismatch"})
			}
			continue
		}

		key := strings.ToLower(parsed.Operator.Hex())
		prev, ok := latest[key]
		if !ok || int64(ev.CreatedAt) > int64(prev.Event.CreatedAt) {
			latest[key] = ie
		}
	}

	out := &Result{Verified: nil, Rejected: rejections}

	for _, ie := range latest {
		ev := ie.Event
		parsed, err := makerad.ParseAd(ev)
		if err != nil {
			continue
		}
		v, err := registry.VerifyMaker(ctx, cfg.RPCHTTP, cfg.RegistryAddr, parsed.Operator, ev.PubKey)
		if err != nil {
			out.Rejected = append(out.Rejected, Rejection{EventID: ev.ID, Reason: "rpc: " + err.Error()})
			continue
		}
		if !v.OK {
			out.Rejected = append(out.Rejected, Rejection{EventID: ev.ID, Reason: "registry: " + v.Reason})
			continue
		}

		vm := VerifiedMaker{
			Operator:         parsed.Operator,
			Tor:              strings.TrimSpace(parsed.Content.Tor),
			SwapX25519PubHex: strings.TrimSpace(parsed.Content.SwapX25519PubHex),
			Stake:            v.Stake.String(),
			MinStake:         v.MinStake.String(),
			EventID:          ev.ID,
			CreatedAt:        int64(ev.CreatedAt),
			RegistryOK:       true,
		}
		if ie.Relay != nil {
			vm.RelayURL = ie.Relay.URL
		}
		if parsed.Content.Fees != nil {
			vm.FeeMinSat = parsed.Content.Fees.Min
			vm.FeeMaxSat = parsed.Content.Fees.Max
		}
		out.Verified = append(out.Verified, vm)
	}

	return out, nil
}
