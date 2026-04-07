package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-sidecar/internal/mweb"
)

func TestBalance(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(NewMux(mweb.NewMockBridge()))
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL + "/v1/balance")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"ok":true`) || !strings.Contains(string(body), "125000000") {
		t.Fatalf("body: %s", body)
	}
}

func TestSwap_success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(NewMux(mweb.NewMockBridge()))
	t.Cleanup(srv.Close)

	payload := `{"route":[
		{"tor":"http://n1","feeMinSat":1},
		{"tor":"http://n2","feeMinSat":2},
		{"tor":"http://n3","feeMinSat":3}
	],"destination":"mweb1x","amount":1000000}`
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/swap", strings.NewReader(payload))
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
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("body: %s", body)
	}
	if !strings.Contains(string(body), "Mock onion successfully injected into coinswapd queue") {
		t.Fatalf("expected mock success detail, body: %s", body)
	}
}

func TestSwap_mockWithLitVMMetadata_hasReceipt(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(NewMux(mweb.NewMockBridge()))
	t.Cleanup(srv.Close)

	payload := `{"route":[
		{"tor":"http://n1","feeMinSat":1},
		{"tor":"http://n2","feeMinSat":2},
		{"tor":"http://n3","feeMinSat":3}
	],"destination":"mweb1x","amount":1000000,"epochId":"5","accuser":"0x3333333333333333333333333333333333333333","swapId":"srv-test-swap"}`
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/swap", strings.NewReader(payload))
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
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	var wrap struct {
		Ok      bool            `json:"ok"`
		SwapID  string          `json:"swapId"`
		Receipt json.RawMessage `json:"receipt"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatal(err)
	}
	if !wrap.Ok || wrap.SwapID != "srv-test-swap" || len(wrap.Receipt) == 0 {
		t.Fatalf("body: %s", body)
	}
}

func TestSwap_validationError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(NewMux(mweb.NewMockBridge()))
	t.Cleanup(srv.Close)

	payload := `{"route":[{"tor":"only-one"}],"destination":"mweb1","amount":1}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/swap", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}
