package mweb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"
)

const (
	rpcMethodSubmitRoute    = "mweb_submitRoute"
	rpcMethodGetBalance     = "mweb_getBalance"
	rpcMethodGetRouteStatus = "mweb_getRouteStatus"
	rpcMethodRunBatch       = "mweb_runBatch"
	rpcMethodGetLastReceipt = "mweb_getLastReceipt"
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
func (b *RPCBridge) HandleSwap(ctx context.Context, req *SwapRequest) (*SwapOutcome, error) {
	NormalizeSwapRequestHops(req)
	if err := ValidateSwapRequest(req); err != nil {
		return nil, &InvalidSwapRequest{Err: err}
	}
	c, err := rpc.DialContext(ctx, b.URL)
	if err != nil {
		return nil, fmt.Errorf("mweb rpc dial: %w", err)
	}
	defer c.Close()

	var rpcResult interface{}
	if err := c.CallContext(ctx, &rpcResult, rpcMethodSubmitRoute, req); err != nil {
		return nil, fmt.Errorf("mweb_submitRoute: %w", err)
	}
	return decodeSubmitRouteRPCResult(rpcResult), nil
}

func decodeSubmitRouteRPCResult(raw interface{}) *SwapOutcome {
	if raw == nil {
		return &SwapOutcome{Detail: "Route submitted to MWEB RPC (mweb_submitRoute)"}
	}
	rb, err := json.Marshal(raw)
	if err != nil {
		return &SwapOutcome{Detail: "Route submitted to MWEB RPC (mweb_submitRoute)"}
	}
	var aux struct {
		Accepted bool            `json:"accepted"`
		SwapID   string          `json:"swapId"`
		Receipt  json.RawMessage `json:"receipt"`
		Detail   string          `json:"detail"`
	}
	if err := json.Unmarshal(rb, &aux); err != nil {
		return &SwapOutcome{Detail: "Route submitted to MWEB RPC (mweb_submitRoute)"}
	}
	out := &SwapOutcome{
		Detail:  strings.TrimSpace(aux.Detail),
		SwapID:  strings.TrimSpace(aux.SwapID),
		Receipt: aux.Receipt,
	}
	if out.Detail == "" {
		out.Detail = "Route submitted to MWEB RPC (mweb_submitRoute)"
	}
	_ = aux.Accepted // legacy payloads only set accepted:true
	return out
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
func (b *RPCBridge) HandleRunBatch(ctx context.Context) (*BatchOutcome, error) {
	c, err := rpc.DialContext(ctx, b.URL)
	if err != nil {
		return nil, fmt.Errorf("mweb rpc dial: %w", err)
	}
	defer c.Close()

	var result map[string]interface{}
	if err := c.CallContext(ctx, &result, rpcMethodRunBatch); err != nil {
		return nil, fmt.Errorf("mweb_runBatch: %w", err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return &BatchOutcome{Detail: "mweb_runBatch ok"}, nil
	}
	var aux struct {
		Detail  string          `json:"detail"`
		SwapID  string          `json:"swapId"`
		Receipt json.RawMessage `json:"receipt"`
	}
	if err := json.Unmarshal(raw, &aux); err != nil {
		return &BatchOutcome{Detail: "mweb_runBatch ok"}, nil
	}
	bo := &BatchOutcome{
		Detail:  strings.TrimSpace(aux.Detail),
		SwapID:  strings.TrimSpace(aux.SwapID),
		Receipt: aux.Receipt,
	}
	if bo.Detail == "" {
		if d, _ := result["detail"].(string); strings.TrimSpace(d) != "" {
			bo.Detail = strings.TrimSpace(d)
		} else {
			bo.Detail = "mweb_runBatch ok"
		}
	}
	return bo, nil
}

// HandleLastReceipt calls mweb_getLastReceipt (no params).
func (b *RPCBridge) HandleLastReceipt(ctx context.Context) (*RouteLastReceipt, error) {
	c, err := rpc.DialContext(ctx, b.URL)
	if err != nil {
		return nil, fmt.Errorf("mweb rpc dial: %w", err)
	}
	defer c.Close()

	var out *RouteLastReceipt
	if err := c.CallContext(ctx, &out, rpcMethodGetLastReceipt); err != nil {
		return nil, fmt.Errorf("mweb_getLastReceipt: %w", err)
	}
	if out == nil {
		return nil, nil
	}
	return out, nil
}
