package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/bridge"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/litvm"
	mlnnostr "github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/nostr"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/store"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/sync/errgroup"
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

	client, err := ethclient.Dial(wsURL)
	if err != nil {
		log.Fatalf("dial rpc: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defender, err := litvm.LoadDefenderFromEnv(ctx, client, courtAddr, operatorAddr)
	if err != nil {
		client.Close()
		log.Fatalf("defender config: %v", err)
	}
	if defender != nil {
		if defender.IsDryRun() {
			log.Println("mlnd: auto-defend enabled (DRY-RUN — no transactions)")
		} else {
			log.Println("mlnd: auto-defend enabled (will submit defendGrievance)")
		}
	}

	watcher, err := litvm.NewWatcher(client, courtAddr, operatorAddr, dbStore, defender)
	if err != nil {
		client.Close()
		log.Fatalf("init watcher: %v", err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("signal %s, shutting down", sig)
		cancel()
	}()

	bc, err := mlnnostr.LoadBroadcasterFromEnv()
	if err != nil {
		client.Close()
		log.Fatalf("nostr broadcaster config: %v", err)
	}
	if bc != nil {
		log.Println("mlnd: Nostr maker-ad broadcaster enabled")
	}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return watcher.Start(gctx)
	})
	if bc != nil {
		g.Go(func() error {
			return bc.Run(gctx)
		})
	}
	if bridgeCoinswapdEnabled() {
		bridgeDir := strings.TrimSpace(os.Getenv("MLND_BRIDGE_RECEIPTS_DIR"))
		if bridgeDir == "" {
			client.Close()
			log.Fatal("MLND_BRIDGE_COINSWAPD is set but MLND_BRIDGE_RECEIPTS_DIR is empty")
		}
		poll := 2 * time.Second
		if s := strings.TrimSpace(os.Getenv("MLND_BRIDGE_POLL_INTERVAL")); s != "" {
			d, err := time.ParseDuration(s)
			if err != nil {
				client.Close()
				log.Fatalf("MLND_BRIDGE_POLL_INTERVAL: %v", err)
			}
			if d > 0 {
				poll = d
			}
		}
		br := bridge.NewCoinswapd(log.Default(), dbStore, bridgeDir, poll)
		g.Go(func() error {
			return br.Run(gctx)
		})
	}

	if err := g.Wait(); err != nil {
		log.Fatalf("run: %v", err)
	}
	log.Println("shutdown complete")
}

func bridgeCoinswapdEnabled() bool {
	s := strings.ToLower(strings.TrimSpace(os.Getenv("MLND_BRIDGE_COINSWAPD")))
	return s == "1" || s == "true" || s == "yes"
}
