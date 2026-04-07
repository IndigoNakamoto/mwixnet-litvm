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
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/dashboard"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/litvm"
	mlnnostr "github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/nostr"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/opslog"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/receiptstore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/sync/errgroup"
)

func main() {
	wsURL := os.Getenv("MLND_WS_URL")
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:8545"
	}

	courtAddr := requireEVMAddr("MLND_COURT_ADDR", os.Getenv("MLND_COURT_ADDR"))
	operatorAddr := requireEVMAddr("MLND_OPERATOR_ADDR", os.Getenv("MLND_OPERATOR_ADDR"))

	dbPath := os.Getenv("MLND_DB_PATH")
	if dbPath == "" {
		dbPath = "mlnd.db"
	}
	dbStore, err := receiptstore.NewStore(dbPath)
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
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	olog := opslog.New(256)

	defender, err := litvm.LoadDefenderFromEnv(ctx, client, courtAddr, operatorAddr)
	if err != nil {
		log.Fatalf("defender config: %v", err)
	}
	if defender != nil {
		if defender.IsDryRun() {
			log.Println("mlnd: auto-defend enabled (DRY-RUN — no transactions)")
		} else {
			log.Println("mlnd: auto-defend enabled (will submit defendGrievance)")
		}
	}

	accuserAddrEnv := strings.TrimSpace(os.Getenv("MLND_ACCUSER_ADDR"))
	accuserResolver, derivedAccuser, err := litvm.LoadAccuserResolverFromEnv(ctx, client, courtAddr, accuserAddrEnv)
	if err != nil {
		log.Fatalf("accuser resolver config: %v", err)
	}
	var accuserWatcher *litvm.AccuserWatcher
	if accuserResolver != nil {
		if accuserResolver.IsDryRun() {
			log.Println("mlnd: accuser auto-resolve enabled (DRY-RUN — no resolve txs)")
		} else {
			log.Println("mlnd: accuser auto-resolve enabled (will submit resolveGrievance after deadline if Open)")
		}
		accuserWatcher, err = litvm.NewAccuserWatcher(client, courtAddr, derivedAccuser, accuserResolver, olog)
		if err != nil {
			log.Fatalf("init accuser watcher: %v", err)
		}
	}

	watcher, err := litvm.NewWatcher(client, courtAddr, operatorAddr, dbStore, defender, olog)
	if err != nil {
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
		log.Fatalf("nostr broadcaster config: %v", err)
	}
	if bc != nil {
		bc.Ops = olog
		log.Println("mlnd: Nostr maker-ad broadcaster enabled")
	}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return watcher.Start(gctx)
	})
	if accuserWatcher != nil {
		g.Go(func() error {
			return accuserWatcher.Start(gctx)
		})
	}
	if bc != nil {
		g.Go(func() error {
			return bc.Run(gctx)
		})
	}
	if bridgeCoinswapdEnabled() {
		bridgeDir := strings.TrimSpace(os.Getenv("MLND_BRIDGE_RECEIPTS_DIR"))
		if bridgeDir == "" {
			log.Fatal("MLND_BRIDGE_COINSWAPD is set but MLND_BRIDGE_RECEIPTS_DIR is empty")
		}
		poll := 2 * time.Second
		if s := strings.TrimSpace(os.Getenv("MLND_BRIDGE_POLL_INTERVAL")); s != "" {
			d, err := time.ParseDuration(s)
			if err != nil {
				log.Fatalf("MLND_BRIDGE_POLL_INTERVAL: %v", err)
			}
			if d > 0 {
				poll = d
			}
		}
		br := bridge.NewCoinswapd(log.Default(), olog, dbStore, bridgeDir, poll)
		g.Go(func() error {
			return br.Run(gctx)
		})
	}

	if dashAddr := strings.TrimSpace(os.Getenv("MLND_DASHBOARD_ADDR")); dashAddr != "" {
		reg := requireEVMAddr("MLND_REGISTRY_ADDR", os.Getenv("MLND_REGISTRY_ADDR"))
		chainID := strings.TrimSpace(os.Getenv("MLND_LITVM_CHAIN_ID"))
		relays := splitRelays(strings.TrimSpace(os.Getenv("MLND_NOSTR_RELAYS")))
		deps := dashboard.StatusDeps{
			EthClient:        client,
			RegistryAddr:     reg,
			CourtAddr:        courtAddr,
			OperatorAddr:     operatorAddr,
			ChainID:          chainID,
			Relays:           relays,
			Broadcaster:      bc,
			BridgeEnabled:    bridgeCoinswapdEnabled(),
			BridgeDir:        strings.TrimSpace(os.Getenv("MLND_BRIDGE_RECEIPTS_DIR")),
			Store:            dbStore,
			AutoDefend:       envTruthy("MLND_DEFEND_AUTO"),
			AutoDefendDryRun: envTruthy("MLND_DEFEND_DRY_RUN"),
			Ops:              olog,
			LitVMRPCURL:      wsURL,
		}
		srv, err := dashboard.NewServer(dashAddr, strings.TrimSpace(os.Getenv("MLND_HTTP_TOKEN")), os.Getenv("MLND_DASHBOARD_ALLOW_LAN"), olog, deps, log.Default())
		if err != nil {
			log.Fatalf("dashboard: %v", err)
		}
		g.Go(func() error {
			return srv.Run(gctx)
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

func envTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

func splitRelays(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// requireEVMAddr normalizes a 20-byte LitVM/EVM address from env and rejects placeholders, invalid hex, and the zero address.
func requireEVMAddr(envName, raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		log.Fatalf("%s is required", envName)
	}
	if strings.Contains(strings.ToUpper(s), "YOUR") {
		log.Fatalf("%s looks like an unreplaced README placeholder (%q); set your real 0x address from deployment", envName, raw)
	}
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		s = "0x" + s
	}
	if !common.IsHexAddress(s) {
		log.Fatalf("%s must be 0x followed by 40 hex digits (check for a bad paste: each export needs its own line in the shell)", envName)
	}
	a := common.HexToAddress(s)
	if a == (common.Address{}) {
		log.Fatalf("%s must not be the zero address", envName)
	}
	return strings.ToLower(a.Hex())
}
