package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-sidecar/internal/api"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-sidecar/internal/mweb"
)

func main() {
	port := flag.Int("port", 8080, "HTTP listen port")
	mode := flag.String("mode", "mock", "mock (Phase 12 default) or rpc (forward to -rpc-url)")
	rpcURL := flag.String("rpc-url", "http://127.0.0.1:8546", "JSON-RPC URL for coinswapd fork (-mode=rpc only)")
	flag.Parse()

	var bridge mweb.Bridge
	switch strings.ToLower(strings.TrimSpace(*mode)) {
	case "mock":
		bridge = mweb.NewMockBridge()
	case "rpc":
		bridge = mweb.NewRPCBridge(strings.TrimSpace(*rpcURL))
	default:
		log.Fatalf("[Sidecar] invalid -mode %q (want mock or rpc)", *mode)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           api.NewMux(bridge),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("[Sidecar] mode=%s listening on %s (GET /v1/balance, POST /v1/swap)", *mode, srv.Addr)
		if *mode == "rpc" {
			log.Printf("[Sidecar] rpc-url=%s (mweb_submitRoute, mweb_getBalance)", strings.TrimSpace(*rpcURL))
		}
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[Sidecar] server: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Printf("[Sidecar] shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[Sidecar] shutdown: %v", err)
	}
}
