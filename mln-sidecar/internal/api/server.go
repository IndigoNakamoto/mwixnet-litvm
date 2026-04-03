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
func NewMux(bridge mweb.Bridge) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/balance", methodOnly(http.MethodGet, handleBalance(bridge)))
	mux.HandleFunc("/v1/swap", methodOnly(http.MethodPost, handleSwap(bridge)))
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

func handleBalance(bridge mweb.Bridge) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		avail, spend, detail, err := bridge.HandleBalance(r.Context())
		if err != nil {
			// Upstream / RPC failures: 502 so clients distinguish from bad request.
			writeJSON(w, http.StatusBadGateway, BalanceResponse{
				Ok:     false,
				Error:  "balance unavailable",
				Detail: err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, BalanceResponse{
			Ok:           true,
			AvailableSat: avail,
			SpendableSat: spend,
			Detail:       detail,
		})
	}
}

func handleSwap(bridge mweb.Bridge) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var req mweb.SwapRequest
		if err := dec.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, SwapResponse{
				Ok:     false,
				Error:  "invalid JSON",
				Detail: err.Error(),
			})
			return
		}
		detail, err := bridge.HandleSwap(r.Context(), &req)
		if err != nil {
			if mweb.IsInvalidSwapRequest(err) {
				writeJSON(w, http.StatusBadRequest, SwapResponse{
					Ok:     false,
					Error:  "validation failed",
					Detail: err.Error(),
				})
				return
			}
			writeJSON(w, http.StatusBadGateway, SwapResponse{
				Ok:     false,
				Error:  "mweb rpc failed",
				Detail: err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, SwapResponse{
			Ok:     true,
			Detail: detail,
		})
	}
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
