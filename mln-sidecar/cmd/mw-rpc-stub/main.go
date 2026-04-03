// Command mw-rpc-stub is a minimal JSON-RPC server implementing mweb_getBalance and mweb_submitRoute
// for integration testing mln-sidecar -mode=rpc without running research/coinswapd (see PHASE_3_MWEB_HANDOFF_SLICE.md).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

func main() {
	addr := flag.String("addr", ":8546", "listen address (e.g. :8546)")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}
		var req rpcRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		var resp map[string]interface{}
		switch req.Method {
		case "mweb_submitRoute":
			log.Printf("mweb_submitRoute params_len=%d", len(req.Params))
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
					"detail":       "mw-rpc-stub",
				},
			}
		default:
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(req.ID),
				"error": map[string]interface{}{
					"code":    -32601,
					"message": "method not found: " + req.Method,
				},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("encode: %v", err)
		}
	})

	log.Printf("mw-rpc-stub listening on %s (mweb_getBalance, mweb_submitRoute)", *addr)
	server := &http.Server{Addr: *addr, Handler: mux}
	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
