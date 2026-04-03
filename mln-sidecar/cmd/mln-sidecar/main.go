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
	"syscall"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-sidecar/internal/api"
)

func main() {
	port := flag.Int("port", 8080, "HTTP listen port")
	flag.Parse()

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           api.NewMux(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("[Sidecar] listening on %s (GET /v1/balance, POST /v1/swap)", srv.Addr)
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
