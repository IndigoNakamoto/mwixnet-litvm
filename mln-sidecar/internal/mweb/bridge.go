package mweb

import (
	"context"
	"errors"
)

// Bridge forwards MLN HTTP contract semantics to either a mock or a coinswapd JSON-RPC endpoint.
type Bridge interface {
	// HandleSwap validates and processes the swap; on success detail is the HTTP response detail field.
	HandleSwap(ctx context.Context, req *SwapRequest) (detail string, err error)
	HandleBalance(ctx context.Context) (availableSat, spendableSat uint64, detail string, err error)
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
func (b *MockBridge) HandleSwap(_ context.Context, req *SwapRequest) (string, error) {
	NormalizeSwapRequestHops(req)
	if err := ValidateSwapRequest(req); err != nil {
		return "", &InvalidSwapRequest{Err: err}
	}
	_ = BuildMockOnion(req)
	return "Mock onion successfully injected into coinswapd queue", nil
}

// HandleBalance returns the same hardcoded values as the original sidecar handlers.
func (b *MockBridge) HandleBalance(_ context.Context) (availableSat, spendableSat uint64, detail string, err error) {
	return 125_000_000, 120_000_000, "Mock balance for E2E", nil
}

// IsInvalidSwapRequest reports whether err is (or wraps) *InvalidSwapRequest.
func IsInvalidSwapRequest(err error) bool {
	var inv *InvalidSwapRequest
	return errors.As(err, &inv)
}
