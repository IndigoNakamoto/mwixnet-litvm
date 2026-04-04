package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-sidecar/internal/mweb"
)

// rpcWire is a minimal JSON-RPC envelope for httptest stubs (go-ethereum client).
type rpcWire struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

func newJSONRPCStub(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req rpcWire
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		var resp map[string]interface{}
		switch req.Method {
		case "mweb_submitRoute":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(req.ID),
				"result":  nil,
			}
		case "mweb_getBalance":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(req.ID),
				"result": map[string]interface{}{
					"availableSat": uint64(10),
					"spendableSat": uint64(9),
					"detail":       "stub",
				},
			}
		case "mweb_getRouteStatus":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(req.ID),
				"result": map[string]interface{}{
					"pendingOnions":          0,
					"mlnRouteHops":           0,
					"nodeIndex":              0,
					"neutrinoConnectedPeers": 0,
				},
			}
		case "mweb_runBatch":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(req.ID),
				"result": map[string]interface{}{
					"triggered": true,
					"detail":    "stub",
				},
			}
		default:
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(req.ID),
				"error": map[string]interface{}{
					"code":    -32601,
					"message": "not found",
				},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestHTTP_RPC_swapAndBalance(t *testing.T) {
	t.Parallel()
	stub := newJSONRPCStub(t)
	t.Cleanup(stub.Close)

	bridge := mweb.NewRPCBridge(stub.URL)
	srv := httptest.NewServer(NewMux(bridge))
	t.Cleanup(srv.Close)

	payload := `{"route":[
		{"tor":"http://n1","feeMinSat":1},
		{"tor":"http://n2","feeMinSat":2},
		{"tor":"http://n3","feeMinSat":3}
	],"destination":"mweb1x","amount":1000000}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v1/swap", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("swap status %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("swap body: %s", body)
	}

	balResp, err := srv.Client().Get(srv.URL + "/v1/balance")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = balResp.Body.Close() })
	balBody, _ := io.ReadAll(balResp.Body)
	if balResp.StatusCode != http.StatusOK {
		t.Fatalf("balance status %d: %s", balResp.StatusCode, balBody)
	}
	if !strings.Contains(string(balBody), `"ok":true`) || !strings.Contains(string(balBody), `"availableSat":10`) {
		t.Fatalf("balance body: %s", balBody)
	}

	stResp, err := srv.Client().Get(srv.URL + "/v1/route/status")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stResp.Body.Close() })
	stBody, _ := io.ReadAll(stResp.Body)
	if stResp.StatusCode != http.StatusOK || !strings.Contains(string(stBody), `"pendingOnions":0`) {
		t.Fatalf("route status: %d %s", stResp.StatusCode, stBody)
	}

	bReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v1/route/batch", nil)
	if err != nil {
		t.Fatal(err)
	}
	batchResp, err := srv.Client().Do(bReq)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = batchResp.Body.Close() })
	batchBody, _ := io.ReadAll(batchResp.Body)
	if batchResp.StatusCode != http.StatusOK || !strings.Contains(string(batchBody), `"ok":true`) {
		t.Fatalf("batch: %d %s", batchResp.StatusCode, batchBody)
	}
}

func TestHTTP_RPC_swapUpstreamError(t *testing.T) {
	t.Parallel()
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcWire
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(req.ID),
			"error": map[string]interface{}{
				"code":    -32000,
				"message": "engine offline",
			},
		})
	}))
	t.Cleanup(stub.Close)

	srv := httptest.NewServer(NewMux(mweb.NewRPCBridge(stub.URL)))
	t.Cleanup(srv.Close)

	payload := `{"route":[
		{"tor":"http://n1","feeMinSat":1},
		{"tor":"http://n2","feeMinSat":2},
		{"tor":"http://n3","feeMinSat":3}
	],"destination":"mweb1x","amount":1000000}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/swap", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("want 502, got %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "engine offline") {
		t.Fatalf("body: %s", body)
	}
}

func TestHTTP_RPC_balanceUpstreamError(t *testing.T) {
	t.Parallel()
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcWire
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(req.ID),
			"error": map[string]interface{}{
				"code":    -32000,
				"message": "balance engine down",
			},
		})
	}))
	t.Cleanup(stub.Close)

	srv := httptest.NewServer(NewMux(mweb.NewRPCBridge(stub.URL)))
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL + "/v1/balance")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("want 502, got %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "balance unavailable") || !strings.Contains(string(body), "balance engine down") {
		t.Fatalf("body: %s", body)
	}
}

func TestHTTP_RPC_swapWithAllHopX25519Keys(t *testing.T) {
	t.Parallel()
	stub := newJSONRPCStub(t)
	t.Cleanup(stub.Close)

	srv := httptest.NewServer(NewMux(mweb.NewRPCBridge(stub.URL)))
	t.Cleanup(srv.Close)

	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	payload := `{"route":[` +
		`{"tor":"http://n1","feeMinSat":1,"swapX25519PubHex":"` + key + `"},` +
		`{"tor":"http://n2","feeMinSat":2,"swapX25519PubHex":"` + key + `"},` +
		`{"tor":"http://n3","feeMinSat":3,"swapX25519PubHex":"` + key + `"}],` +
		`"destination":"mweb1x","amount":1000000}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v1/swap", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("swap status %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("swap body: %s", body)
	}
}
