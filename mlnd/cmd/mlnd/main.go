package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/litvm"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/store"
)

func main() {
	wsURL := os.Getenv("MLND_WS_URL")
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:8545"
	}

	courtAddr := os.Getenv("MLND_COURT_ADDR")
	if courtAddr == "" {
		log.Fatal("MLND_COURT_ADDR is required")
	}

	operatorAddr := os.Getenv("MLND_OPERATOR_ADDR")
	if operatorAddr == "" {
		log.Fatal("MLND_OPERATOR_ADDR is required")
	}

	dbPath := os.Getenv("MLND_DB_PATH")
	if dbPath == "" {
		dbPath = "mlnd.db"
	}
	dbStore, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer func() {
		if err := dbStore.Close(); err != nil {
			log.Printf("database close: %v", err)
		}
	}()

	watcher, err := litvm.NewWatcher(wsURL, courtAddr, operatorAddr, dbStore)
	if err != nil {
		log.Fatalf("init watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("signal %s, shutting down", sig)
		cancel()
	}()

	if err := watcher.Start(ctx); err != nil {
		log.Fatalf("watcher: %v", err)
	}
	log.Println("shutdown complete")
}
