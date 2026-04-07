package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-judge/internal/judge"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	ws := strings.TrimSpace(os.Getenv("JUDGE_LITVM_WS_URL"))
	if ws == "" {
		ws = "ws://127.0.0.1:8545"
	}
	courtRaw := strings.TrimSpace(os.Getenv("JUDGE_COURT_ADDR"))
	if courtRaw == "" || !common.IsHexAddress(with0x(courtRaw)) {
		log.Fatal("JUDGE_COURT_ADDR is required (0x + 40 hex)")
	}
	court := common.HexToAddress(with0x(courtRaw))

	key, err := judge.PrivateKeyFromEnv()
	if err != nil {
		log.Fatalf("key: %v", err)
	}

	dry := truthy(os.Getenv("JUDGE_DRY_RUN"))
	auto := truthy(os.Getenv("JUDGE_AUTO_ADJUDICATE"))
	verdict := strings.TrimSpace(strings.ToLower(os.Getenv("JUDGE_VERDICT")))
	exonerate := verdict == "exonerate"
	if auto && verdict != "exonerate" && verdict != "uphold" {
		log.Fatal("JUDGE_AUTO_ADJUDICATE=1 requires JUDGE_VERDICT=exonerate|uphold")
	}

	client, err := ethclient.Dial(ws)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sig := <-signalCh(syscall.SIGINT, syscall.SIGTERM)
		log.Printf("signal %v, shutting down", sig)
		cancel()
	}()

	svc := &judge.Service{
		Client:           client,
		Court:            court,
		PrivateKey:       key,
		DryRun:           dry,
		AutoAdjudicate:   auto,
		VerdictExonerate: exonerate,
	}
	if err := svc.Run(ctx); err != nil {
		log.Fatalf("run: %v", err)
	}
}

func with0x(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return s
	}
	return "0x" + s
}

func truthy(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "1" || s == "true" || s == "yes"
}

func signalCh(sig ...os.Signal) <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig...)
	return ch
}
