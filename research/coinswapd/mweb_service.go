package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

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

	rawKeys, err := mlnroute.ResolveX25519PubKeys(&req, m.pubkeyMap)
	if err != nil {
		return nil, mlnroute.InvalidParams("swap keys required: "+err.Error())
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

	if err := m.ss.acceptOnionAndSetMLNRoute(o, mlNodesFromRequest(&req, rawKeys)); err != nil {
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
	PendingOnions          int  `json:"pendingOnions"`
	MlnRouteHops           int  `json:"mlnRouteHops"`
	NodeIndex              int  `json:"nodeIndex"`
	NeutrinoConnectedPeers int  `json:"neutrinoConnectedPeers"`
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
	return map[string]interface{}{
		"triggered": true,
		"detail":    "performSwap finished its synchronous steps; P2P swap_forward/swap_backward may still be in flight",
	}, nil
}

func mlNodesFromRequest(req *mlnroute.Request, rawKeys [][]byte) []config.Node {
	nodes := make([]config.Node, len(req.Route))
	for i, h := range req.Route {
		nodes[i] = config.NewNode(strings.TrimSpace(h.Tor), hex.EncodeToString(rawKeys[i]))
	}
	return nodes
}

// acceptOnionAndSetMLNRoute validates, persists, and pins routing peers for the next forward/backward pass.
func (s *swapService) acceptOnionAndSetMLNRoute(o *onion.Onion, peers []config.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(peers) == mlnroute.ExpectedHops {
		s.mlnPeers = peers
	}
	return s.acceptOnionLocked(o)
}
