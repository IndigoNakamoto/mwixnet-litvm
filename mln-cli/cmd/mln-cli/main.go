package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/config"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/forger"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/scout"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "scout":
		runScout(os.Args[2:])
	case "pathfind":
		runPathfind(os.Args[2:])
	case "forger":
		runForger(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `mln-cli — MLN taker client (Phase 10)

Commands:
  scout     Discover kind-31250 maker ads, verify LitVM registry state
  pathfind  Pick an ordered 3-hop route from verified makers (min fee hint, stake tie-break)
  forger    Validate route (-dry-run) or POST route JSON to local coinswapd MLN sidecar

Environment (scout & pathfind):
  MLN_NOSTR_RELAYS          comma-separated wss:// relays
  MLN_LITVM_CHAIN_ID        decimal chain id string (filter ads)
  MLN_LITVM_HTTP_URL        LitVM HTTP JSON-RPC for eth_call
  MLN_REGISTRY_ADDR         MwixnetRegistry address
  MLN_GRIEVANCE_COURT_ADDR  optional; if set, must match ad content
  MLN_SCOUT_TIMEOUT         optional subscription timeout (e.g. 45s)

`)
}

func runScout(args []string) {
	fs := flag.NewFlagSet("scout", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "print JSON result to stdout")
	quiet := fs.Bool("quiet", false, "omit rejection rows from JSON / stderr")
	_ = fs.Parse(args)

	relays, chainID, rpcURL, regStr, court, timeout, err := config.ScoutFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}
	regAddr, err := config.ParseRegistryAddr(regStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "registry: %v\n", err)
		os.Exit(2)
	}

	cfg := scout.Config{
		Relays:          relays,
		RPCHTTP:         rpcURL,
		ChainID:         chainID,
		RegistryAddr:    regAddr,
		GrievanceCourt:  court,
		Timeout:         timeout,
		QuietRejections: *quiet,
	}

	ctx := context.Background()
	res, err := scout.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scout: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintf(os.Stderr, "json: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("%-42s %-10s %-24s %s\n", "OPERATOR", "STATUS", "STAKE", "TOR")
	fmt.Println(strings.Repeat("-", 120))
	for _, m := range res.Verified {
		tor := m.Tor
		if len(tor) > 40 {
			tor = tor[:37] + "..."
		}
		fmt.Printf("%-42s %-10s %-24s %s\n", m.Operator.Hex(), "verified", m.Stake, tor)
	}
	if !*quiet {
		for _, r := range res.Rejected {
			fmt.Fprintf(os.Stderr, "rejected %s: %s\n", r.EventID, r.Reason)
		}
	}
	if len(res.Verified) == 0 {
		fmt.Fprintln(os.Stderr, "(no verified makers)")
	}
}

func runPathfind(args []string) {
	fs := flag.NewFlagSet("pathfind", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "print route JSON")
	_ = fs.Parse(args)

	relays, chainID, rpcURL, regStr, court, timeout, err := config.ScoutFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}
	regAddr, err := config.ParseRegistryAddr(regStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "registry: %v\n", err)
		os.Exit(2)
	}

	cfg := scout.Config{
		Relays:         relays,
		RPCHTTP:        rpcURL,
		ChainID:        chainID,
		RegistryAddr:   regAddr,
		GrievanceCourt: court,
		Timeout:        timeout,
	}

	ctx := context.Background()
	res, err := scout.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scout: %v\n", err)
		os.Exit(1)
	}

	route, err := pathfind.PickRoute(res.Verified, rand.New(rand.NewSource(time.Now().UnixNano())))
	if err != nil {
		fmt.Fprintf(os.Stderr, "pathfind: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(route); err != nil {
			fmt.Fprintf(os.Stderr, "json: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("fee_sum_sat_hint=%d\n", route.FeeSumSat)
	for i, h := range route.Hops {
		fmt.Printf("N%d operator=%s tor=%s stake=%s\n", i+1, h.Operator.Hex(), h.Tor, h.Stake)
	}
}

func runForger(args []string) {
	fs := flag.NewFlagSet("forger", flag.ExitOnError)
	routePath := fs.String("route-json", "", "path to JSON route from `mln-cli pathfind -json`")
	dryRun := fs.Bool("dry-run", true, "validate route only (default); set false to POST to sidecar")
	dest := fs.String("dest", "", "destination MWEB address (required with -dry-run=false)")
	amount := fs.Uint64("amount", 0, "amount to swap in satoshis (required with -dry-run=false)")
	sidecarURL := fs.String("coinswapd-url", "http://127.0.0.1:8080/v1/swap", "MLN extension URL of local coinswapd sidecar")
	_ = fs.Parse(args)

	if *routePath == "" {
		fmt.Fprintln(os.Stderr, "forger: -route-json is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*routePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "forger: read route: %v\n", err)
		os.Exit(1)
	}
	var route pathfind.Route
	if err := json.Unmarshal(raw, &route); err != nil {
		fmt.Fprintf(os.Stderr, "forger: json: %v\n", err)
		os.Exit(1)
	}

	if *dryRun {
		if err := forger.DryRun(&route); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		return
	}

	if strings.TrimSpace(*dest) == "" {
		fmt.Fprintln(os.Stderr, "forger: -dest is required when -dry-run=false")
		os.Exit(2)
	}
	if *amount == 0 {
		fmt.Fprintln(os.Stderr, "forger: -amount must be > 0 when -dry-run=false")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := forger.Execute(ctx, &route, *sidecarURL, *dest, *amount); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
