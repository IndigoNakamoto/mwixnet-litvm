package mweb

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// SwapOutcome is returned by HandleSwap (detail + optional LitVM receipt wiring).
type SwapOutcome struct {
	Detail  string
	SwapID  string
	Receipt json.RawMessage
}

// BatchOutcome is returned by HandleRunBatch when RPC includes receipt metadata.
type BatchOutcome struct {
	Detail  string
	SwapID  string
	Receipt json.RawMessage
}

// Bridge forwards MLN HTTP contract semantics to either a mock or a coinswapd JSON-RPC endpoint.
type Bridge interface {
	HandleSwap(ctx context.Context, req *SwapRequest) (*SwapOutcome, error)
	HandleBalance(ctx context.Context) (availableSat, spendableSat uint64, detail string, err error)
	// HandleRouteStatus polls mweb_getRouteStatus (RPC mode); mock returns zeros.
	HandleRouteStatus(ctx context.Context) (*RouteStatus, error)
	HandleRunBatch(ctx context.Context) (*BatchOutcome, error)
}

// RouteStatus mirrors research/coinswapd mweb_getRouteStatus for HTTP JSON.
type RouteStatus struct {
	PendingOnions          int `json:"pendingOnions"`
	MlnRouteHops           int `json:"mlnRouteHops"`
	NodeIndex              int `json:"nodeIndex"`
	NeutrinoConnectedPeers int `json:"neutrinoConnectedPeers"`
}

// InvalidSwapRequest marks validation failures that the HTTP layer should map to 400.
type InvalidSwapRequest struct {
	Err error
}

func (e *InvalidSwapRequest) Error() string { return e.Err.Error() }
func (e *InvalidSwapRequest) Unwrap() error { return e.Err }

// MockBridge preserves Phase 12 E2E behavior (no external RPC).
type MockBridge struct{}

// NewMockBridge returns a bridge that logs a mock onion and serves fixed balances.
func NewMockBridge() *MockBridge {
	return &MockBridge{}
}

// HandleSwap validates the request and simulates onion construction.
func (b *MockBridge) HandleSwap(_ context.Context, req *SwapRequest) (*SwapOutcome, error) {
	NormalizeSwapRequestHops(req)
	if err := ValidateSwapRequest(req); err != nil {
		return nil, &InvalidSwapRequest{Err: err}
	}
	_ = BuildMockOnion(req)
	out := &SwapOutcome{Detail: "Mock onion successfully injected into coinswapd queue"}
	// With LitVM metadata, emit a golden receipt for local E2E (forger vault + grievance dry-run).
	if strings.TrimSpace(req.EpochID) != "" {
		acc := strings.TrimSpace(req.Accuser)
		if !strings.HasPrefix(acc, "0x") && !strings.HasPrefix(acc, "0X") {
			acc = "0x" + acc
		}
		rw := map[string]interface{}{
			"epochId":               strings.TrimSpace(req.EpochID),
			"accuser":               acc,
			"accusedMaker":          "0x0000000000000000000000000000000000000001",
			"hopIndex":              0,
			"peeledCommitment":      "0x1111111111111111111111111111111111111111111111111111111111111111",
			"forwardCiphertextHash": "0x2222222222222222222222222222222222222222222222222222222222222222",
			"nextHopPubkey":         "mock-bridge-next-hop",
			"signature":             "mock-bridge-signature",
			"swapId":                strings.TrimSpace(req.SwapID),
		}
		raw, err := json.Marshal(rw)
		if err != nil {
			return nil, err
		}
		out.SwapID = strings.TrimSpace(req.SwapID)
		out.Receipt = raw
		out.Detail = "Mock onion successfully injected into coinswapd queue (golden receipt with LitVM metadata)"
	}
	return out, nil
}

// HandleBalance returns the same hardcoded values as the original sidecar handlers.
func (b *MockBridge) HandleBalance(_ context.Context) (availableSat, spendableSat uint64, detail string, err error) {
	return 125_000_000, 120_000_000, "Mock balance for E2E", nil
}

// HandleRouteStatus returns an empty queue in mock mode.
func (b *MockBridge) HandleRouteStatus(_ context.Context) (*RouteStatus, error) {
	return &RouteStatus{}, nil
}

// HandleRunBatch is a no-op in mock mode.
func (b *MockBridge) HandleRunBatch(_ context.Context) (*BatchOutcome, error) {
	return &BatchOutcome{Detail: "mock: no batch RPC"}, nil
}

// IsInvalidSwapRequest reports whether err is (or wraps) *InvalidSwapRequest.
func IsInvalidSwapRequest(err error) bool {
	var inv *InvalidSwapRequest
	return errors.As(err, &inv)
}
