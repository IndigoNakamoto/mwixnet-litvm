package forger

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/scout"
)

func testRoute() *pathfind.Route {
	return &pathfind.Route{
		Hops: [3]scout.VerifiedMaker{
			{Tor: "http://n1.onion", FeeMinSat: 100},
			{Tor: "http://n2.onion", FeeMinSat: 200},
			{Tor: "http://n3.onion", FeeMinSat: 300},
		},
		FeeSumSat: 600,
	}
}

func TestSubmitRoute_SuccessAndBody(t *testing.T) {
	t.Parallel()

	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"detail":"queued"}`))
	}))
	t.Cleanup(srv.Close)

	c := NewSidecarClient(srv.URL)
	resp, err := c.SubmitRoute(context.Background(), testRoute(), "mweb1qqdest", 99_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Ok || resp.Detail != "queued" {
		t.Fatalf("resp = %+v", resp)
	}

	var payload RequestPayload
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Destination != "mweb1qqdest" || payload.Amount != 99_000_000 {
		t.Fatalf("payload = %+v", payload)
	}
	if len(payload.Route) != 3 {
		t.Fatalf("len(route) = %d", len(payload.Route))
	}
	want := []HopRequest{
		{Tor: "http://n1.onion", FeeMinSat: 100},
		{Tor: "http://n2.onion", FeeMinSat: 200},
		{Tor: "http://n3.onion", FeeMinSat: 300},
	}
	for i := range want {
		if payload.Route[i] != want[i] {
			t.Fatalf("hop %d = %+v, want %+v", i, payload.Route[i], want[i])
		}
	}
}

func TestSubmitRoute_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	c := NewSidecarClient(srv.URL)
	_, err := c.SubmitRoute(context.Background(), testRoute(), "mweb1x", 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubmitRoute_OkFalse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"bad route","detail":"N2 down"}`))
	}))
	t.Cleanup(srv.Close)

	c := NewSidecarClient(srv.URL)
	resp, err := c.SubmitRoute(context.Background(), testRoute(), "mweb1x", 1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Ok || resp.Error != "bad route" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestSubmitRoute_EmptyBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := NewSidecarClient(srv.URL)
	_, err := c.SubmitRoute(context.Background(), testRoute(), "mweb1x", 1)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestExecute_RequiresDestAndAmount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	if _, err := Execute(ctx, testRoute(), "http://unused", "", 100); err == nil {
		t.Fatal("expected error for empty dest")
	}
	if _, err := Execute(ctx, testRoute(), "http://unused", "mweb1x", 0); err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestNormalizeMixEndpoint(t *testing.T) {
	t.Parallel()
	if got, want := NormalizeMixEndpoint("abc123.onion:8080"), "http://abc123.onion:8080"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got, want := NormalizeMixEndpoint("http://x"), "http://x"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDryRun_Result(t *testing.T) {
	t.Parallel()

	res, err := DryRun(testRoute())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hops) != 3 {
		t.Fatalf("hops = %d", len(res.Hops))
	}
	if res.Hops[0].Index != 1 || res.Hops[0].Tor != "http://n1.onion" {
		t.Fatalf("hop0 = %+v", res.Hops[0])
	}
}

func TestDryRun_prefixesBareOnionHost(t *testing.T) {
	t.Parallel()
	r := &pathfind.Route{
		Hops: [3]scout.VerifiedMaker{
			{Tor: "n1.onion:8080", FeeMinSat: 1},
			{Tor: "n2.onion:8080", FeeMinSat: 1},
			{Tor: "n3.onion:8080", FeeMinSat: 1},
		},
		FeeSumSat: 3,
	}
	res, err := DryRun(r)
	if err != nil {
		t.Fatal(err)
	}
	if res.Hops[0].Tor != "http://n1.onion:8080" {
		t.Fatalf("got %q", res.Hops[0].Tor)
	}
}

func TestSidecarBaseFromSwapURL(t *testing.T) {
	t.Parallel()
	got, err := SidecarBaseFromSwapURL("http://127.0.0.1:8080/v1/swap")
	if err != nil {
		t.Fatal(err)
	}
	if want := "http://127.0.0.1:8080"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExecuteWithBatchOptions_triggerAndWait(t *testing.T) {
	t.Parallel()

	var pending atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/swap", func(w http.ResponseWriter, r *http.Request) {
		pending.Store(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"detail":"submitted"}`))
	})
	mux.HandleFunc("/v1/route/batch", func(w http.ResponseWriter, r *http.Request) {
		pending.Store(0)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"detail":"batch ok"}`))
	})
	mux.HandleFunc("/v1/route/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n := pending.Load()
		_, _ = w.Write([]byte(`{"ok":true,"pendingOnions":` + strconv.FormatInt(int64(n), 10) + `,"mlnRouteHops":0,"nodeIndex":0,"neutrinoConnectedPeers":0}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	swapURL := srv.URL + "/v1/swap"
	ctx := context.Background()
	res, err := ExecuteWithBatchOptions(ctx, testRoute(), swapURL, "mweb1qq", 1_000_000, &BatchOptions{
		TriggerBatch:    true,
		WaitPendingZero: true,
		PollInterval:    50 * time.Millisecond,
		Timeout:         2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.PendingCleared {
		t.Fatal("expected pending cleared")
	}
}

func TestBalanceURL(t *testing.T) {
	t.Parallel()

	got, err := BalanceURL("http://127.0.0.1:8080/v1/swap")
	if err != nil {
		t.Fatal(err)
	}
	if want := "http://127.0.0.1:8080/v1/balance"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFetchMwebBalance_OK(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/balance", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method %q", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"availableSat":125000000,"detail":"mweb wallet"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	b, err := FetchMwebBalance(context.Background(), srv.URL+"/v1/swap", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if b.AvailableSat != 125_000_000 || b.SpendableSat != 125_000_000 {
		t.Fatalf("%+v", b)
	}
	if b.Detail != "mweb wallet" {
		t.Fatal(b.Detail)
	}
}

func TestFetchMwebBalance_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	_, err := FetchMwebBalance(context.Background(), srv.URL+"/v1/swap", srv.Client())
	if err == nil {
		t.Fatal("expected error")
	}
}
