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
	"strings"
	"sync"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

// submitRouteBody matches the single param object mln-sidecar sends to mweb_submitRoute (parity with research/coinswapd/mlnroute).
type submitRouteBody struct {
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

func writeRPC(w http.ResponseWriter, id json.RawMessage, result interface{}, rpcErr *struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}) {
	w.Header().Set("Content-Type", "application/json")
	var out map[string]interface{}
	if rpcErr != nil {
		out = map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(id),
			"error":   rpcErr,
		}
	} else {
		out = map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(id),
			"result":  result,
		}
	}
	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Printf("encode: %v", err)
	}
}

func main() {
	addr := flag.String("addr", ":8546", "listen address (e.g. :8546)")
	flag.Parse()

	var stubMu sync.Mutex
	stubPending := 0

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
		switch req.Method {
		case "mweb_submitRoute":
			var params []json.RawMessage
			if err := json.Unmarshal(req.Params, &params); err != nil {
				writeRPC(w, req.ID, nil, &struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{Code: -32602, Message: "invalid params: " + err.Error()})
				return
			}
			if len(params) != 1 {
				writeRPC(w, req.ID, nil, &struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{Code: -32602, Message: fmt.Sprintf("mweb_submitRoute expects 1 param object, got %d", len(params))})
				return
			}
			var sr submitRouteBody
			if err := json.Unmarshal(params[0], &sr); err != nil {
				writeRPC(w, req.ID, nil, &struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{Code: -32602, Message: "route body: " + err.Error()})
				return
			}
			if len(sr.Route) != 3 || sr.Destination == "" || sr.Amount == 0 {
				writeRPC(w, req.ID, nil, &struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{Code: -32602, Message: "route must have 3 hops, non-empty destination, positive amount"})
				return
			}
			for i, h := range sr.Route {
				if h.Tor == "" {
					writeRPC(w, req.ID, nil, &struct {
						Code    int    `json:"code"`
						Message string `json:"message"`
					}{Code: -32602, Message: fmt.Sprintf("hop %d: tor required", i)})
					return
				}
			}
			log.Printf("mweb_submitRoute ok (accepted; destination and amount not logged)")
			stubMu.Lock()
			stubPending++
			stubMu.Unlock()

			ep := strings.TrimSpace(sr.EpochID)
			ac := strings.TrimSpace(sr.Accuser)
			sw := strings.TrimSpace(sr.SwapID)
			if ep != "" && ac != "" && sw != "" {
				acc := ac
				if !strings.HasPrefix(acc, "0x") && !strings.HasPrefix(acc, "0X") {
					acc = "0x" + acc
				}
				receipt := map[string]interface{}{
					"epochId":               ep,
					"accuser":               acc,
					"accusedMaker":          goldenReceiptAccusedMaker(sr),
					"hopIndex":              0,
					"peeledCommitment":      "0x1111111111111111111111111111111111111111111111111111111111111111",
					"forwardCiphertextHash": "0x2222222222222222222222222222222222222222222222222222222222222222",
					"nextHopPubkey":         "mw-rpc-stub-next-hop",
					"signature":             "mw-rpc-stub-signature",
					"swapId":                sw,
				}
				writeRPC(w, req.ID, map[string]interface{}{
					"accepted": true,
					"swapId":   sw,
					"detail":   "mw-rpc-stub: golden receipt (LitVM metadata present)",
					"receipt":  receipt,
				}, nil)
				return
			}
			writeRPC(w, req.ID, map[string]interface{}{"accepted": true}, nil)
		case "mweb_getBalance":
			writeRPC(w, req.ID, map[string]interface{}{
				"availableSat": uint64(10),
				"spendableSat": uint64(9),
				"detail":       "mw-rpc-stub",
			}, nil)
		case "mweb_getRouteStatus":
			stubMu.Lock()
			n := stubPending
			stubMu.Unlock()
			writeRPC(w, req.ID, map[string]interface{}{
				"pendingOnions":          n,
				"mlnRouteHops":           0,
				"nodeIndex":              0,
				"neutrinoConnectedPeers": 0,
			}, nil)
		case "mweb_runBatch":
			stubMu.Lock()
			stubPending = 0
			stubMu.Unlock()
			writeRPC(w, req.ID, map[string]interface{}{
				"triggered": true,
				"detail":    "mw-rpc-stub: cleared virtual pending queue",
			}, nil)
		default:
			writeRPC(w, req.ID, nil, &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32601, Message: "method not found: " + req.Method})
		}
	})

	log.Printf("mw-rpc-stub listening on %s (mweb_getBalance, mweb_submitRoute, mweb_getRouteStatus, mweb_runBatch)", *addr)
	server := &http.Server{Addr: *addr, Handler: mux}
	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
