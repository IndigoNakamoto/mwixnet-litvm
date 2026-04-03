package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-sidecar/internal/mweb"
)

// BalanceResponse matches PHASE_10_TAKER_CLI.md and mln-cli forger balance parsing.
type BalanceResponse struct {
	Ok           bool   `json:"ok"`
	AvailableSat uint64 `json:"availableSat"`
	SpendableSat uint64 `json:"spendableSat"`
	Detail       string `json:"detail,omitempty"`
	Error        string `json:"error,omitempty"`
}

// SwapResponse matches mln-cli ResponsePayload for success paths.
type SwapResponse struct {
	Ok     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

// NewMux registers the MLN HTTP contract (GET /v1/balance, POST /v1/swap).
func NewMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/balance", methodOnly(http.MethodGet, handleBalance))
	mux.HandleFunc("/v1/swap", methodOnly(http.MethodPost, handleSwap))
	mux.HandleFunc("/healthz", methodOnly(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	return loggingMiddleware(mux)
}

func methodOnly(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

func handleBalance(w http.ResponseWriter, r *http.Request) {
	resp := BalanceResponse{
		Ok:           true,
		AvailableSat: 125_000_000,
		SpendableSat: 120_000_000,
		Detail:       "Mock balance for E2E",
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleSwap(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req mweb.SwapRequest
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SwapResponse{
			Ok:    false,
			Error: "invalid JSON",
			Detail: err.Error(),
		})
		return
	}
	if err := mweb.ValidateSwapRequest(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SwapResponse{
			Ok:    false,
			Error: "validation failed",
			Detail: err.Error(),
		})
		return
	}
	_ = mweb.BuildMockOnion(&req)
	writeJSON(w, http.StatusOK, SwapResponse{
		Ok:     true,
		Detail: "Mock onion successfully injected into coinswapd queue",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		log.Printf("[Sidecar] json encode: %v", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[Sidecar] %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
