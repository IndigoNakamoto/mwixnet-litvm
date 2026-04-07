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
			write(map[string]interface{}{"accepted": true}, nil)
		case rpcMethodGetBalance:
			write(balanceResult, nil)
		case rpcMethodGetRouteStatus:
			write(map[string]interface{}{
				"pendingOnions":          0,
				"mlnRouteHops":           0,
				"nodeIndex":              0,
				"neutrinoConnectedPeers": 0,
			}, nil)
		case rpcMethodRunBatch:
			write(map[string]interface{}{"triggered": true, "detail": "stub"}, nil)
		case rpcMethodGetLastReceipt:
			write(nil, nil)
		default:
			write(nil, &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32601, Message: "method not found"})
		}
	}))
}

func TestRPCBridge_HandleSwap_extendedReceipt(t *testing.T) {
	t.Parallel()
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var env jsonRPCReq
		if err := json.Unmarshal(body, &env); err != nil {
			http.Error(w, "json", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		result := map[string]interface{}{
			"accepted": true,
			"swapId":   "ext-1",
			"detail":   "stub extended",
			"receipt": map[string]interface{}{
				"epochId": "1", "accuser": "0x1111111111111111111111111111111111111111",
				"accusedMaker": "0x0000000000000000000000000000000000000001", "hopIndex": 0,
				"peeledCommitment": "0x1111111111111111111111111111111111111111111111111111111111111111",
				"forwardCiphertextHash": "0x2222222222222222222222222222222222222222222222222222222222222222",
				"nextHopPubkey": "x", "signature": "y", "swapId": "ext-1",
			},
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0", "id": json.RawMessage(env.ID), "result": result,
		})
	}))
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
	out, err := b.HandleSwap(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if out.SwapID != "ext-1" || !strings.Contains(out.Detail, "stub extended") || len(out.Receipt) == 0 {
		t.Fatalf("got %+v", out)
	}
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
	out, err := b.HandleSwap(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Detail, "mweb_submitRoute") {
		t.Fatalf("detail: %q", out.Detail)
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

func TestRPCBridge_HandleRouteStatus_and_RunBatch(t *testing.T) {
	t.Parallel()
	stub := newMwebStubServer(t, "", rpcBalanceResult{AvailableSat: 1, SpendableSat: 1})
	t.Cleanup(stub.Close)

	b := NewRPCBridge(stub.URL)
	st, err := b.HandleRouteStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.PendingOnions != 0 {
		t.Fatalf("pending %d", st.PendingOnions)
	}
	bo, err := b.HandleRunBatch(context.Background())
	if err != nil || !strings.Contains(bo.Detail, "stub") {
		t.Fatalf("batch: %q %v", bo.Detail, err)
	}
	lr, err := b.HandleLastReceipt(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if lr != nil {
		t.Fatalf("want nil last receipt from stub, got %+v", lr)
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
		Operator         string `json:"operator,omitempty"`
	} `json:"route"`
	Destination string `json:"destination"`
	Amount      uint64 `json:"amount"`
	EpochID     string `json:"epochId,omitempty"`
	Accuser     string `json:"accuser,omitempty"`
	SwapID      string `json:"swapId,omitempty"`
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
	op := "0x4444444444444444444444444444444444444444"
	b := NewRPCBridge(stub.URL)
	req := &SwapRequest{
		Route: []HopInput{
			{Tor: "http://a", FeeMinSat: 1, SwapX25519PubHex: key, Operator: op},
			{Tor: "http://b", FeeMinSat: 2, SwapX25519PubHex: key, Operator: op},
			{Tor: "http://c", FeeMinSat: 3, SwapX25519PubHex: key, Operator: op},
		},
		Destination: "mweb1qq",
		Amount:      100,
		EpochID:     "42",
		Accuser:     "0x1111111111111111111111111111111111111111",
		SwapID:      "stub-swap-wire",
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
	if got.Route[0].Operator != op {
		t.Fatalf("want operator on hop 0, got %q", got.Route[0].Operator)
	}
	if got.Destination != "mweb1qq" || got.Amount != 100 {
		t.Fatalf("dest/amount: %+v", got)
	}
	if got.EpochID != "42" || got.Accuser != "0x1111111111111111111111111111111111111111" || got.SwapID != "stub-swap-wire" {
		t.Fatalf("litvm metadata: %+v", got)
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

func TestRPCBridge_HandleLastReceipt_populated(t *testing.T) {
	t.Parallel()
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "application/json")
		if req.Method != rpcMethodGetLastReceipt {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(req.ID),
				"error":   map[string]interface{}{"code": -32601, "message": "wrong"},
			})
			return
		}
		result := map[string]interface{}{
			"receipt":             json.RawMessage(`{"epochId":"1","accuser":"0x1111111111111111111111111111111111111111","accusedMaker":"0x2222222222222222222222222222222222222222","hopIndex":1,"peeledCommitment":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","forwardCiphertextHash":"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","nextHopPubkey":"0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f","signature":"unsigned-swap-forward-failure-v1"}`),
			"swapId":              "sw1",
			"forwardFailureClass": "rpc_application",
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(req.ID),
			"result":  result,
		})
	}))
	t.Cleanup(stub.Close)

	b := NewRPCBridge(stub.URL)
	lr, err := b.HandleLastReceipt(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if lr == nil || len(lr.Receipt) == 0 {
		t.Fatalf("receipt: %+v", lr)
	}
	if lr.SwapID != "sw1" || lr.ForwardFailureClass != "rpc_application" {
		t.Fatalf("fields: %+v", lr)
	}
}
