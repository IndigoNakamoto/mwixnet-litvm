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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/config"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/forger"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/grievance"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/identity"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/nostridentity"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/registry"
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
	case "route":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "route: need subcommand (e.g. build)")
			usage()
			os.Exit(2)
		}
		switch os.Args[2] {
		case "build":
			runRouteBuild(os.Args[3:])
		default:
			fmt.Fprintf(os.Stderr, "route: unknown subcommand %q\n", os.Args[2])
			os.Exit(2)
		}
	case "forger":
		runForger(os.Args[2:])
	case "maker":
		runMaker(os.Args[2:])
	case "grievance":
		runGrievance(os.Args[2:])
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
  pathfind  Pick an ordered 3-hop route from verified makers with Tor endpoints (min fee hint, stake tie-break; -self-included for N2=self)
  route build  Run scout + pathfind and write route JSON (default route.json) for forger -route-json
  forger    Validate route (-dry-run) or POST route JSON to local coinswapd MLN sidecar
  maker onboard  Plan or execute MwixnetRegistry deposit + registerMaker (LitVM maker onboarding)
  grievance file Open a grievance on LitVM (openGrievance + bond; receipt from JSON or vault swap id)

Environment (scout & pathfind):
  MLN_NOSTR_RELAYS          comma-separated wss:// relays
  MLN_NOSTR_RELAY_URL       optional single relay if MLN_NOSTR_RELAYS is unset
  MLN_LITVM_CHAIN_ID        decimal chain id string (filter ads)
  MLN_LITVM_HTTP_URL        LitVM HTTP JSON-RPC for eth_call
  MLN_REGISTRY_ADDR         MwixnetRegistry address
  MLN_GRIEVANCE_COURT_ADDR  optional; if set, must match ad content
  MLN_SCOUT_TIMEOUT         optional subscription timeout (e.g. 45s)
  MLN_OPERATOR_ETH_KEY      optional; 64-hex LitVM maker private key — scout marks matching row "(local)";
                              required with pathfind -self-included (fixes N2 to that maker)

Environment (maker onboard):
  MLN_LITVM_HTTP_URL        LitVM HTTP JSON-RPC (required)
  MLN_REGISTRY_ADDR         MwixnetRegistry address (required)
  MLN_LITVM_CHAIN_ID        decimal chain id (required)
  MLN_OPERATOR_ETH_KEY      64-hex LitVM key that signs txs and is the maker address (required)
  MLN_NOSTR_PUBKEY_HEX      64-hex x-only Nostr pubkey, or set MLN_NOSTR_NSEC (hex or nsec1…)

Environment (grievance file):
  MLN_LITVM_HTTP_URL        LitVM HTTP JSON-RPC (required)
  MLN_GRIEVANCE_COURT_ADDR  GrievanceCourt contract (required)
  MLN_ACCUSER_ETH_KEY       64-hex key for msg.sender (must match receipt accuser); else MLN_OPERATOR_ETH_KEY
  MLN_GRIEVANCE_BOND_WEI    optional wei amount (decimal string); default 0.1 ether
  MLN_RECEIPT_VAULT_PATH    optional SQLite path (hop_receipts); use with positional swap_id

Environment (forger -vault / MLN_RECEIPT_VAULT_PATH):
  MLN_RECEIPT_EPOCH_ID      decimal LitVM epoch for threaded receipts (required with vault)
  MLN_ACCUSER_ETH_KEY       same as grievance (must match receipt accuser)

`)
}

func runGrievance(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "grievance: need subcommand (e.g. file)")
		os.Exit(2)
	}
	switch args[0] {
	case "file":
		runGrievanceFile(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "grievance: unknown subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func runGrievanceFile(args []string) {
	fs := flag.NewFlagSet("grievance file", flag.ExitOnError)
	receiptPath := fs.String("receipt-json", "", "path to receipt JSON (same schema as mlnd bridge NDJSON)")
	vaultPath := fs.String("vault", "", "SQLite vault path (default: MLN_RECEIPT_VAULT_PATH)")
	dryRun := fs.Bool("dry-run", false, "print hashes only; do not broadcast")
	_ = fs.Parse(args)
	pos := fs.Args()

	var src grievance.ReceiptSource
	switch {
	case *receiptPath != "":
		src = grievance.ReceiptJSONFile{Path: *receiptPath}
	case len(pos) >= 1:
		db := strings.TrimSpace(*vaultPath)
		if db == "" {
			db = strings.TrimSpace(os.Getenv("MLN_RECEIPT_VAULT_PATH"))
		}
		if db == "" {
			fmt.Fprintln(os.Stderr, "grievance file: need -receipt-json or (swap_id + -vault / MLN_RECEIPT_VAULT_PATH)")
			os.Exit(2)
		}
		src = grievance.VaultSwapLookup{DBPath: db, SwapID: strings.TrimSpace(pos[0])}
	default:
		fmt.Fprintln(os.Stderr, "grievance file: provide -receipt-json PATH or SWAP_ID with vault")
		os.Exit(2)
	}

	rpcURL, err := config.LitvmHTTPURLFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}
	courtStr := strings.TrimSpace(os.Getenv("MLN_GRIEVANCE_COURT_ADDR"))
	if courtStr == "" {
		fmt.Fprintln(os.Stderr, "MLN_GRIEVANCE_COURT_ADDR is required")
		os.Exit(2)
	}
	if !common.IsHexAddress(courtStr) && !strings.HasPrefix(courtStr, "0x") {
		fmt.Fprintln(os.Stderr, "invalid MLN_GRIEVANCE_COURT_ADDR")
		os.Exit(2)
	}
	if !strings.HasPrefix(courtStr, "0x") {
		courtStr = "0x" + courtStr
	}
	courtAddr := common.HexToAddress(courtStr)

	key, err := grievance.AccuserKeyFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "accuser key: %v\n", err)
		os.Exit(2)
	}

	ctx := context.Background()
	err = grievance.RunFile(ctx, grievance.FileOpts{
		RPCURL:      rpcURL,
		Court:       courtAddr,
		PrivateKey:  key,
		BondWei:     grievance.BondWeiFromEnv(),
		DryRun:      *dryRun,
		Out:         os.Stdout,
		SuggestFees: true,
	}, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "grievance file: %v\n", err)
		os.Exit(1)
	}
}

func runMaker(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "maker: need subcommand (e.g. onboard)")
		os.Exit(2)
	}
	switch args[0] {
	case "onboard":
		runMakerOnboard(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "maker: unknown subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func runMakerOnboard(args []string) {
	fs := flag.NewFlagSet("maker onboard", flag.ExitOnError)
	execute := fs.Bool("execute", false, "broadcast deposit and/or registerMaker (default is dry-run plan only)")
	forceReregister := fs.Bool("force-reregister", false, "allow registerMaker when on-chain makerNostrKeyHash differs (overwrites binding)")
	_ = fs.Parse(args)

	env, err := config.OnboardFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}
	pubHex, err := nostridentity.PubkeyHexFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "nostr identity: %v\n", err)
		os.Exit(2)
	}

	ctx := context.Background()
	err = registry.RunOnboard(ctx, registry.OnboardOpts{
		RPCHTTP:         env.RPCHTTP,
		Registry:        env.Registry,
		ChainID:         env.ChainID,
		PrivateKeyHex:   env.PrivateKeyHex,
		NostrPubHex:     pubHex,
		Execute:         *execute,
		ForceReregister: *forceReregister,
		Out:             os.Stdout,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "maker onboard: %v\n", err)
		os.Exit(1)
	}
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

	var localAddr common.Address
	var hasLocal bool
	if k := strings.TrimSpace(os.Getenv("MLN_OPERATOR_ETH_KEY")); k != "" {
		if a, err := identity.AddressFromHexPrivateKey(k); err == nil {
			localAddr = a
			hasLocal = true
		}
	}

	if *jsonOut {
		if hasLocal {
			for i := range res.Verified {
				res.Verified[i].Local = res.Verified[i].Operator == localAddr
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintf(os.Stderr, "json: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("%-42s %-20s %-24s %s\n", "OPERATOR", "STATUS", "STAKE", "TOR")
	fmt.Println(strings.Repeat("-", 130))
	for _, m := range res.Verified {
		tor := m.Tor
		if len(tor) > 40 {
			tor = tor[:37] + "..."
		}
		status := "verified"
		if hasLocal && m.Operator == localAddr {
			status = "verified (local)"
		}
		fmt.Printf("%-42s %-20s %-24s %s\n", m.Operator.Hex(), status, m.Stake, tor)
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

func loadScoutConfig() (scout.Config, error) {
	relays, chainID, rpcURL, regStr, court, timeout, err := config.ScoutFromEnv()
	if err != nil {
		return scout.Config{}, err
	}
	regAddr, err := config.ParseRegistryAddr(regStr)
	if err != nil {
		return scout.Config{}, err
	}
	return scout.Config{
		Relays:         relays,
		RPCHTTP:        rpcURL,
		ChainID:        chainID,
		RegistryAddr:   regAddr,
		GrievanceCourt: court,
		Timeout:        timeout,
	}, nil
}

func runPathfindPick(ctx context.Context, cfg scout.Config, selfIncluded bool, rng *rand.Rand) (*pathfind.Route, error) {
	res, err := scout.Run(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("scout: %w", err)
	}
	if selfIncluded {
		key := strings.TrimSpace(os.Getenv("MLN_OPERATOR_ETH_KEY"))
		if key == "" {
			return nil, fmt.Errorf("pathfind: -self-included requires MLN_OPERATOR_ETH_KEY")
		}
		addr, err := identity.AddressFromHexPrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("pathfind: MLN_OPERATOR_ETH_KEY: %w", err)
		}
		return pathfind.PickRouteSelfMiddle(res.Verified, addr, rng)
	}
	return pathfind.PickRoute(res.Verified, rng)
}

func runPathfind(args []string) {
	fs := flag.NewFlagSet("pathfind", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "print route JSON")
	selfIncluded := fs.Bool("self-included", false, "fix middle hop to maker from MLN_OPERATOR_ETH_KEY")
	_ = fs.Parse(args)

	cfg, err := loadScoutConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}

	ctx := context.Background()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	route, err := runPathfindPick(ctx, cfg, *selfIncluded, rng)
	if err != nil {
		msg := err.Error()
		fmt.Fprintf(os.Stderr, "%v\n", err)
		if strings.HasPrefix(msg, "pathfind:") && (strings.Contains(msg, "self-included requires") || strings.Contains(msg, "MLN_OPERATOR_ETH_KEY:")) {
			os.Exit(2)
		}
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

func runRouteBuild(args []string) {
	fs := flag.NewFlagSet("route build", flag.ExitOnError)
	outPath := fs.String("out", "route.json", "path to write pathfind.Route JSON (for forger -route-json)")
	selfIncluded := fs.Bool("self-included", false, "fix middle hop to maker from MLN_OPERATOR_ETH_KEY")
	_ = fs.Parse(args)

	cfg, err := loadScoutConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}

	ctx := context.Background()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	route, err := runPathfindPick(ctx, cfg, *selfIncluded, rng)
	if err != nil {
		msg := err.Error()
		fmt.Fprintf(os.Stderr, "%v\n", err)
		if strings.HasPrefix(msg, "pathfind:") && (strings.Contains(msg, "self-included requires") || strings.Contains(msg, "MLN_OPERATOR_ETH_KEY:")) {
			os.Exit(2)
		}
		os.Exit(1)
	}

	raw, err := json.MarshalIndent(route, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "route build: json: %v\n", err)
		os.Exit(1)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(*outPath, raw, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "route build: write %s: %v\n", *outPath, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "route build: wrote %s\n", *outPath)
}

func runForger(args []string) {
	fs := flag.NewFlagSet("forger", flag.ExitOnError)
	routePath := fs.String("route-json", "", "path to JSON route from `mln-cli pathfind -json` or `mln-cli route build`")
	dryRun := fs.Bool("dry-run", true, "validate route only (default); set false to POST to sidecar")
	dest := fs.String("dest", "", "destination MWEB address (required with -dry-run=false)")
	amount := fs.Uint64("amount", 0, "amount to swap in satoshis (required with -dry-run=false)")
	sidecarURL := fs.String("coinswapd-url", "http://127.0.0.1:8080/v1/swap", "MLN extension URL of local coinswapd sidecar")
	triggerBatch := fs.Bool("trigger-batch", false, "after submit, POST /v1/route/batch (forwards to mweb_runBatch on coinswapd)")
	waitBatch := fs.Bool("wait-batch", false, "poll GET /v1/route/status until pendingOnions==0 or -batch-timeout")
	batchPoll := fs.Duration("batch-poll", 2*time.Second, "poll interval for -wait-batch")
	batchTimeout := fs.Duration("batch-timeout", 2*time.Minute, "max wait for pendingOnions==0 with -wait-batch")
	vaultPath := fs.String("vault", "", "SQLite receipt vault path (default MLN_RECEIPT_EPOCH_ID + MLN_ACCUSER_ETH_KEY required)")
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
		if err := forger.DryRunCLI(&route, os.Stderr); err != nil {
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

	ctxTimeout := 10 * time.Second
	if *triggerBatch || *waitBatch {
		ctxTimeout = *batchTimeout + 45*time.Second
		if ctxTimeout < 45*time.Second {
			ctxTimeout = 45 * time.Second
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	var batch *forger.BatchOptions
	if *triggerBatch || *waitBatch {
		batch = &forger.BatchOptions{
			TriggerBatch:    *triggerBatch,
			WaitPendingZero: *waitBatch,
			PollInterval:    *batchPoll,
			Timeout:         *batchTimeout,
		}
	}

	db := strings.TrimSpace(*vaultPath)
	if db == "" {
		db = strings.TrimSpace(os.Getenv("MLN_RECEIPT_VAULT_PATH"))
	}
	var vault *forger.VaultOptions
	if db != "" {
		epoch := strings.TrimSpace(os.Getenv("MLN_RECEIPT_EPOCH_ID"))
		if epoch == "" {
			fmt.Fprintln(os.Stderr, "forger: -vault requires MLN_RECEIPT_EPOCH_ID")
			os.Exit(2)
		}
		key, err := grievance.AccuserKeyFromEnv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "forger: vault accuser key: %v\n", err)
			os.Exit(2)
		}
		addr := crypto.PubkeyToAddress(key.PublicKey)
		vault = &forger.VaultOptions{
			DBPath:  db,
			EpochID: epoch,
			Accuser: addr.Hex(),
		}
	}

	if err := forger.ExecuteCLIWithBatch(ctx, &route, *sidecarURL, *dest, *amount, batch, vault, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
