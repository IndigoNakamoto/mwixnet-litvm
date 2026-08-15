package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ltcmweb/coinswapd/config"
	"github.com/ltcmweb/coinswapd/mlnroute"
	"github.com/ltcmweb/coinswapd/onion"
	"github.com/ltcmweb/ltcd/ltcutil/mweb/mw"
)

// BalanceResult is returned by mweb_getBalance (JSON field names match mln-sidecar).
type BalanceResult struct {
	AvailableSat uint64 `json:"availableSat"`
	SpendableSat uint64 `json:"spendableSat"`
	Detail       string `json:"detail,omitempty"`
}

// mwebService implements JSON-RPC namespace "mweb" → mweb_getBalance / mweb_submitRoute.
type mwebService struct {
	ss *swapService

	scanKey   *mw.SecretKey
	spendKey  *mw.SecretKey
	pubkeyMap map[string]string

	// When true, RunBatch deletes any onions still in the DB after performSwap (no finalize). See main flag.
	devClearPendingAfterBatch bool
	// submitRemote: after buildOnionFromMLNRoute, Dial hop 0 swap_Swap instead of saveOnion.
	submitRemote bool
}

func decodeSecretKeyHex(label, s string) (*mw.SecretKey, error) {
	if s == "" {
		return nil, nil
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if len(b) != 32 {
		return nil, fmt.Errorf("%s: want 32 bytes hex", label)
	}
	var k mw.SecretKey
	copy(k[:], b)
	return &k, nil
}

// GetBalance returns MWEB balances from MwebCoinDB + RewindOutput (scan key).
func (m *mwebService) GetBalance(ctx context.Context) (BalanceResult, error) {
	_ = ctx
	if m.scanKey == nil {
		return BalanceResult{
			Detail: "wallet scan key not configured (set -mweb-scan-secret)",
		}, nil
	}
	coins, err := listWalletCoins(cs, m.scanKey, m.spendKey)
	if err != nil {
		return BalanceResult{}, mlnroute.Internal(err.Error())
	}
	avail, spend := mwebBalanceTotals(coins)
	detail := ""
	if m.spendKey == nil {
		detail = "spendableSat is 0 without -mweb-spend-secret"
	}
	return BalanceResult{
		AvailableSat: avail,
		SpendableSat: spend,
		Detail:       detail,
	}, nil
}

// SubmitRoute accepts MLN route JSON (single object param) per COINSWAPD_MLN_FORK_SPEC.
func (m *mwebService) SubmitRoute(ctx context.Context, req mlnroute.Request) (interface{}, error) {
	_ = ctx
	if err := mlnroute.Validate(&req); err != nil {
		return nil, mlnroute.InvalidParams(err.Error())
	}

	ops, err := mlnroute.PeerOperatorsFromRequest(&req)
	if err != nil {
		return nil, mlnroute.InvalidParams(err.Error())
	}
	m.ss.setMLNPeerOperators(ops)
	// Thread LitVM coordination into swap state (appendix 13 receipts on swap_forward failure).
	m.ss.setMLNRouteMeta(strings.TrimSpace(req.EpochID), strings.TrimSpace(req.Accuser), strings.TrimSpace(req.SwapID))

	rawKeys, err := mlnroute.ResolveX25519PubKeys(&req, m.pubkeyMap)
	if err != nil {
		return nil, mlnroute.InvalidParams("swap keys required: " + err.Error())
	}
	peerPub, err := mlnroute.ECDHPublicKeys(rawKeys)
	if err != nil {
		return nil, mlnroute.InvalidParams(err.Error())
	}

	if m.scanKey == nil || m.spendKey == nil {
		return nil, mlnroute.InvalidParams("mweb_submitRoute requires -mweb-scan-secret and -mweb-spend-secret")
	}

	coins, err := listWalletCoins(cs, m.scanKey, m.spendKey)
	if err != nil {
		return nil, mlnroute.Internal(err.Error())
	}
	coin, err := pickCoinExactAmount(coins, req.Amount)
	if err != nil {
		return nil, err
	}

	o, err := buildOnionFromMLNRoute(&req, peerPub, coin)
	if err != nil {
		return nil, err
	}

	peers := mlNodesFromRequest(&req, rawKeys)
	if m.submitRemote {
		if err := m.ss.swapSwapRemote(o, peers); err != nil {
			return nil, mlnroute.Internal(err.Error())
		}
		return map[string]interface{}{
			"accepted": true,
			"detail":   "onion sent to hop 0 via swap_Swap (not queued on this taker)",
		}, nil
	}

	if err := m.ss.acceptOnionAndSetMLNRoute(o, peers, req.Amount); err != nil {
		if errors.Is(err, errNotNodeZero) {
			return nil, mlnroute.InvalidParams(err.Error())
		}
		if strings.HasPrefix(err.Error(), "save onion:") {
			return nil, mlnroute.Internal(err.Error())
		}
		return nil, mlnroute.OnionOrCrypto(err.Error())
	}

	return map[string]bool{"accepted": true}, nil
}

// RouteStatus is returned by mweb_getRouteStatus for operator polling after mweb_submitRoute.
type RouteStatus struct {
	PendingOnions          int `json:"pendingOnions"`
	MlnRouteHops           int `json:"mlnRouteHops"`
	NodeIndex              int `json:"nodeIndex"`
	NeutrinoConnectedPeers int `json:"neutrinoConnectedPeers"`
}

// GetRouteStatus reports how many onions are persisted locally and whether an MLN route is pinned.
func (m *mwebService) GetRouteStatus(ctx context.Context) (RouteStatus, error) {
	_ = ctx
	onions, err := loadOnions(db)
	if err != nil {
		return RouteStatus{}, mlnroute.Internal(err.Error())
	}
	m.ss.mu.Lock()
	mlnHops := len(m.ss.mlnPeers)
	nodeIdx := m.ss.nodeIndex
	m.ss.mu.Unlock()
	if mlnHops != mlnroute.ExpectedHops {
		mlnHops = 0
	}
	peers := 0
	if cs != nil {
		peers = int(cs.ConnectedCount())
	}
	return RouteStatus{
		PendingOnions:          len(onions),
		MlnRouteHops:           mlnHops,
		NodeIndex:              nodeIdx,
		NeutrinoConnectedPeers: peers,
	}, nil
}

// RunBatch invokes the same midnight batch entrypoint synchronously (validate → peel → forward/backward).
// swap_forward / swap_backward to peer makers still run asynchronously inside coinswapd when dialing succeeds.
func (m *mwebService) RunBatch(ctx context.Context) (map[string]interface{}, error) {
	_ = ctx
	if err := m.ss.performSwap(); err != nil {
		return nil, mlnroute.Internal(err.Error())
	}
	detail := "performSwap finished its synchronous steps; P2P swap_forward/swap_backward may still be in flight"
	if m.devClearPendingAfterBatch {
		onions, err := loadOnions(db)
		if err != nil {
			return nil, mlnroute.Internal(err.Error())
		}
		for _, o := range onions {
			if err := deleteOnion(db, o); err != nil {
				return nil, mlnroute.Internal(err.Error())
			}
		}
		if len(onions) > 0 {
			fmt.Printf("mweb-dev-clear-pending-after-batch: removed %d onion(s) from DB without chain finalize (DEV ONLY)\n", len(onions))
			detail = fmt.Sprintf("devClearPendingAfterBatch: cleared %d onion(s) from local DB (no broadcast)", len(onions))
		}
	}
	return map[string]interface{}{
		"triggered": true,
		"detail":    detail,
	}, nil
}

// LastReceiptResponse is returned by mweb_getLastReceipt when a hop failure receipt was recorded.
type LastReceiptResponse struct {
	Receipt             json.RawMessage `json:"receipt"`
	SwapID              string          `json:"swapId,omitempty"`
	ForwardFailureClass string          `json:"forwardFailureClass,omitempty"`
}

// GetLastReceipt returns the most recent appendix-13-style receipt from a failed swap_forward, or nil if none.
func (m *mwebService) GetLastReceipt(ctx context.Context) (*LastReceiptResponse, error) {
	_ = ctx
	raw, swapID, cls := m.ss.takeLastReceiptSnapshot()
	if len(raw) == 0 {
		return nil, nil
	}
	return &LastReceiptResponse{
		Receipt:             append(json.RawMessage(nil), raw...),
		SwapID:              swapID,
		ForwardFailureClass: cls,
	}, nil
}

// swapSwapRemote pins directory peers from the route and sends the onion to hop 0 (no local queue).
func (s *swapService) swapSwapRemote(o *onion.Onion, peers []config.Node) error {
	if o == nil {
		return fmt.Errorf("mln-submit-remote: nil onion")
	}
	if len(peers) != mlnroute.ExpectedHops {
		return fmt.Errorf("mln-submit-remote: need %d hops, got %d", mlnroute.ExpectedHops, len(peers))
	}
	hop0 := strings.TrimSpace(peers[0].Url)
	if hop0 == "" {
		return fmt.Errorf("mln-submit-remote: hop 0 tor URL is empty")
	}
	s.mu.Lock()
	s.mlnPeers = peers
	s.mu.Unlock()

	// Taker→hop0 JSON-RPC uses net/http DefaultTransport (HTTP_PROXY / socks5h for .onion).
	client, err := rpc.Dial(hop0)
	if err != nil {
		return fmt.Errorf("mln-submit-remote: rpc.Dial hop 0: %w", err)
	}
	defer client.Close()
	if err := client.Call(nil, "swap_swap", *o); err != nil {
		return fmt.Errorf("mln-submit-remote: swap_swap hop 0: %w", err)
	}
	return nil
}

func mlNodesFromRequest(req *mlnroute.Request, rawKeys [][]byte) []config.Node {
	nodes := make([]config.Node, len(req.Route))
	for i, h := range req.Route {
		nodes[i] = config.NewNode(strings.TrimSpace(h.Tor), hex.EncodeToString(rawKeys[i]))
	}
	return nodes
}

// acceptOnionAndSetMLNRoute validates, persists, pins routing peers, and records the route amount for same-value batches.
func (s *swapService) acceptOnionAndSetMLNRoute(o *onion.Onion, peers []config.Node, amountSat uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(peers) == mlnroute.ExpectedHops {
		s.mlnPeers = peers
	}
	if err := s.acceptOnionLocked(o); err != nil {
		return err
	}
	if err := saveOnionAmount(db, o, amountSat); err != nil {
		_ = deleteOnion(db, o)
		return fmt.Errorf("save onion amount: %w", err)
	}
	return nil
}
