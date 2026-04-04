package mweb

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// jsonRPCReq is a minimal JSON-RPC 2.0 request shape (go-ethereum client compatible).
type jsonRPCReq struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

func newMwebStubServer(t *testing.T, swapErr string, balanceResult interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read", http.StatusBadRequest)
			return
		}
		var req jsonRPCReq
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "json", http.StatusBadRequest)
			return
		}
		write := func(result interface{}, rpcErr *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}) {
			w.Header().Set("Content-Type", "application/json")
			var out map[string]interface{}
			if rpcErr != nil {
				out = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      json.RawMessage(req.ID),
					"error":   rpcErr,
				}
			} else {
				out = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      json.RawMessage(req.ID),
					"result":  result,
				}
			}
			_ = json.NewEncoder(w).Encode(out)
		}

		switch req.Method {
		case rpcMethodSubmitRoute:
			if swapErr != "" {
				write(nil, &struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{Code: -32603, Message: swapErr})
				return
			}
			write(nil, nil)
		case rpcMethodGetBalance:
			write(balanceResult, nil)
		default:
			write(nil, &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32601, Message: "method not found"})
		}
	}))
}

func TestRPCBridge_HandleSwap_success(t *testing.T) {
	t.Parallel()
	stub := newMwebStubServer(t, "", nil)
	t.Cleanup(stub.Close)

	b := NewRPCBridge(stub.URL)
	req := &SwapRequest{
		Route: []HopInput{
			{Tor: "http://a", FeeMinSat: 1},
			{Tor: "http://b", FeeMinSat: 2},
			{Tor: "http://c", FeeMinSat: 3},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	detail, err := b.HandleSwap(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, "mweb_submitRoute") {
		t.Fatalf("detail: %q", detail)
	}
}

func TestRPCBridge_HandleSwap_rpcError(t *testing.T) {
	t.Parallel()
	stub := newMwebStubServer(t, "insufficient funds", nil)
	t.Cleanup(stub.Close)

	b := NewRPCBridge(stub.URL)
	req := &SwapRequest{
		Route: []HopInput{
			{Tor: "http://a", FeeMinSat: 1},
			{Tor: "http://b", FeeMinSat: 2},
			{Tor: "http://c", FeeMinSat: 3},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	_, err := b.HandleSwap(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "insufficient funds") {
		t.Fatalf("err: %v", err)
	}
}

func TestRPCBridge_HandleBalance_success(t *testing.T) {
	t.Parallel()
	stub := newMwebStubServer(t, "", rpcBalanceResult{
		AvailableSat: 50_000_000,
		SpendableSat: 49_000_000,
		Detail:       "from stub",
	})
	t.Cleanup(stub.Close)

	b := NewRPCBridge(stub.URL)
	avail, spend, detail, err := b.HandleBalance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if avail != 50_000_000 || spend != 49_000_000 {
		t.Fatalf("avail=%d spend=%d", avail, spend)
	}
	if detail != "from stub" {
		t.Fatalf("detail %q", detail)
	}
}

func TestNewRPCBridge_normalizesTrailingSlash(t *testing.T) {
	t.Parallel()
	stub := newMwebStubServer(t, "", rpcBalanceResult{AvailableSat: 1, SpendableSat: 2})
	t.Cleanup(stub.Close)

	b := NewRPCBridge(stub.URL + "/  ")
	avail, spend, _, err := b.HandleBalance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if avail != 1 || spend != 2 {
		t.Fatalf("avail=%d spend=%d", avail, spend)
	}
}

// forkWireRoute is the JSON shape expected by research/coinswapd/mlnroute.Request (single mweb_submitRoute param).
type forkWireRoute struct {
	Route []struct {
		Tor              string `json:"tor"`
		FeeMinSat        uint64 `json:"feeMinSat"`
		SwapX25519PubHex string `json:"swapX25519PubHex,omitempty"`
	} `json:"route"`
	Destination string `json:"destination"`
	Amount      uint64 `json:"amount"`
}

// TestRPCBridge_HandleSwap_JSONRPCParamsMatchForkWire asserts go-ethereum encodes mweb_submitRoute as one object param
// with field names matching mlnroute.Request / COINSWAPD_MLN_FORK_SPEC.
func TestRPCBridge_HandleSwap_JSONRPCParamsMatchForkWire(t *testing.T) {
	t.Parallel()
	var capturedBody string
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read", http.StatusBadRequest)
			return
		}
		capturedBody = string(body)
		var req jsonRPCReq
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "json", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if req.Method != rpcMethodSubmitRoute {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(req.ID),
				"error": map[string]interface{}{
					"code": -32601, "message": "wrong method",
				},
			})
			return
		}
		var params []json.RawMessage
		if err := json.Unmarshal(req.Params, &params); err != nil || len(params) != 1 {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(req.ID),
				"error": map[string]interface{}{
					"code": -32602, "message": "params",
				},
			})
			return
		}
		var wire forkWireRoute
		if err := json.Unmarshal(params[0], &wire); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(req.ID),
				"error": map[string]interface{}{
					"code": -32602, "message": err.Error(),
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(req.ID),
			"result":  nil,
		})
	}))
	t.Cleanup(stub.Close)

	key := "0000000000000000000000000000000000000000000000000000000000000001"
	b := NewRPCBridge(stub.URL)
	req := &SwapRequest{
		Route: []HopInput{
			{Tor: "http://a", FeeMinSat: 1, SwapX25519PubHex: key},
			{Tor: "http://b", FeeMinSat: 2, SwapX25519PubHex: key},
			{Tor: "http://c", FeeMinSat: 3, SwapX25519PubHex: key},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	_, err := b.HandleSwap(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if capturedBody == "" {
		t.Fatal("no request captured")
	}
	var env struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal([]byte(capturedBody), &env); err != nil {
		t.Fatalf("captured json: %v", err)
	}
	if env.Method != rpcMethodSubmitRoute {
		t.Fatalf("method %q", env.Method)
	}
	var params []json.RawMessage
	if err := json.Unmarshal(env.Params, &params); err != nil || len(params) != 1 {
		t.Fatalf("params: %s", env.Params)
	}
	var got forkWireRoute
	if err := json.Unmarshal(params[0], &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Route) != 3 || got.Route[0].Tor != "http://a" || got.Route[0].FeeMinSat != 1 {
		t.Fatalf("route[0]: %+v", got.Route)
	}
	if got.Route[0].SwapX25519PubHex != key {
		t.Fatalf("want swap key on hop 0, got %q", got.Route[0].SwapX25519PubHex)
	}
	if got.Destination != "mweb1qq" || got.Amount != 100 {
		t.Fatalf("dest/amount: %+v", got)
	}
}

func TestRPCBridge_HandleSwap_validationBeforeRPC(t *testing.T) {
	t.Parallel()
	stub := newMwebStubServer(t, "should not reach", nil)
	t.Cleanup(stub.Close)

	b := NewRPCBridge(stub.URL)
	req := &SwapRequest{Route: []HopInput{{Tor: "x"}}, Destination: "d", Amount: 1}
	_, err := b.HandleSwap(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !IsInvalidSwapRequest(err) {
		t.Fatalf("want InvalidSwapRequest, got %v", err)
	}
}
