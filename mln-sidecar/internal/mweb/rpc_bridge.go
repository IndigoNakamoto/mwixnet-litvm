package mweb

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"
)

const (
	rpcMethodSubmitRoute   = "mweb_submitRoute"
	rpcMethodGetBalance    = "mweb_getBalance"
	rpcMethodGetRouteStatus = "mweb_getRouteStatus"
	rpcMethodRunBatch      = "mweb_runBatch"
)

// rpcBalanceResult matches the expected JSON shape from mweb_getBalance (fork contract).
type rpcBalanceResult struct {
	AvailableSat uint64 `json:"availableSat"`
	SpendableSat uint64 `json:"spendableSat"`
	Detail       string `json:"detail,omitempty"`
}

// RPCBridge calls mweb_submitRoute and mweb_getBalance on a coinswapd-compatible JSON-RPC server.
type RPCBridge struct {
	URL string
}

// normalizeRPCURL trims whitespace and a trailing slash so Dial matches coinswapd / stub listeners.
func normalizeRPCURL(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, "/")
	return s
}

// NewRPCBridge dials nothing at construction; each call uses a short-lived client.
func NewRPCBridge(rawURL string) *RPCBridge {
	return &RPCBridge{URL: normalizeRPCURL(rawURL)}
}

// HandleSwap validates locally, then forwards the MLN payload to mweb_submitRoute.
func (b *RPCBridge) HandleSwap(ctx context.Context, req *SwapRequest) (string, error) {
	NormalizeSwapRequestHops(req)
	if err := ValidateSwapRequest(req); err != nil {
		return "", &InvalidSwapRequest{Err: err}
	}
	c, err := rpc.DialContext(ctx, b.URL)
	if err != nil {
		return "", fmt.Errorf("mweb rpc dial: %w", err)
	}
	defer c.Close()

	var result interface{}
	if err := c.CallContext(ctx, &result, rpcMethodSubmitRoute, req); err != nil {
		return "", fmt.Errorf("mweb_submitRoute: %w", err)
	}
	return "Route submitted to MWEB RPC (mweb_submitRoute)", nil
}

// HandleBalance calls mweb_getBalance (no parameters).
func (b *RPCBridge) HandleBalance(ctx context.Context) (availableSat, spendableSat uint64, detail string, err error) {
	c, err := rpc.DialContext(ctx, b.URL)
	if err != nil {
		return 0, 0, "", fmt.Errorf("mweb rpc dial: %w", err)
	}
	defer c.Close()

	var out rpcBalanceResult
	if err := c.CallContext(ctx, &out, rpcMethodGetBalance); err != nil {
		return 0, 0, "", fmt.Errorf("mweb_getBalance: %w", err)
	}
	return out.AvailableSat, out.SpendableSat, out.Detail, nil
}

// HandleRouteStatus calls mweb_getRouteStatus (no params).
func (b *RPCBridge) HandleRouteStatus(ctx context.Context) (*RouteStatus, error) {
	c, err := rpc.DialContext(ctx, b.URL)
	if err != nil {
		return nil, fmt.Errorf("mweb rpc dial: %w", err)
	}
	defer c.Close()

	var st RouteStatus
	if err := c.CallContext(ctx, &st, rpcMethodGetRouteStatus); err != nil {
		return nil, fmt.Errorf("mweb_getRouteStatus: %w", err)
	}
	return &st, nil
}

// HandleRunBatch calls mweb_runBatch (no params).
func (b *RPCBridge) HandleRunBatch(ctx context.Context) (string, error) {
	c, err := rpc.DialContext(ctx, b.URL)
	if err != nil {
		return "", fmt.Errorf("mweb rpc dial: %w", err)
	}
	defer c.Close()

	var result map[string]interface{}
	if err := c.CallContext(ctx, &result, rpcMethodRunBatch); err != nil {
		return "", fmt.Errorf("mweb_runBatch: %w", err)
	}
	if d, _ := result["detail"].(string); d != "" {
		return d, nil
	}
	return "mweb_runBatch ok", nil
}
