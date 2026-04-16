package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/config"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/scout"
)

// runDoctor performs environment reachability checks before users try scout/pathfind.
// It aggregates the separate probes previously spread across tor-preflight, phase3-operator-preflight,
// e2e-status.sh, and manual curl invocations. Non-zero exit only when a critical check fails.
func runDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	sidecarURL := fs.String("sidecar-url", "", "optional mln-sidecar base URL (e.g. http://127.0.0.1:8080)")
	skipScout := fs.Bool("skip-scout", false, "skip live Nostr scout call (faster; still checks RPC + relay reachability)")
	_ = fs.Parse(args)

	ok, failed := 0, 0
	mark := func(b bool, line string) {
		if b {
			fmt.Printf("  [ OK ] %s\n", line)
			ok++
		} else {
			fmt.Printf("  [FAIL] %s\n", line)
			failed++
		}
	}

	fmt.Println("mln-cli doctor — environment preflight")
	fmt.Println()

	fmt.Println("1) Env vars")
	relays, chainID, rpcURL, regStr, court, _, err := config.ScoutFromEnv()
	if err != nil {
		fmt.Printf("  [FAIL] scout env: %v\n", err)
		fmt.Println("\nRequired: MLN_NOSTR_RELAYS (or MLN_NOSTR_RELAY_URL), MLN_LITVM_HTTP_URL, MLN_REGISTRY_ADDR, MLN_LITVM_CHAIN_ID")
		os.Exit(2)
	}
	mark(len(relays) > 0, fmt.Sprintf("MLN_NOSTR_RELAYS = %s", strings.Join(relays, ",")))
	mark(rpcURL != "", fmt.Sprintf("MLN_LITVM_HTTP_URL = %s", rpcURL))
	mark(regStr != "", fmt.Sprintf("MLN_REGISTRY_ADDR = %s", regStr))
	mark(chainID != "", fmt.Sprintf("MLN_LITVM_CHAIN_ID = %s", chainID))
	if strings.TrimSpace(court) != "" {
		fmt.Printf("  [info] MLN_GRIEVANCE_COURT_ADDR = %s\n", court)
	}
	fmt.Println()

	fmt.Println("2) LitVM RPC (eth_chainId)")
	cid, err := rpcChainID(rpcURL)
	if err != nil {
		mark(false, fmt.Sprintf("eth_chainId: %v", err))
	} else {
		mark(true, fmt.Sprintf("eth_chainId = 0x%x (%d)", cid, cid))
		if chainID != "" && fmt.Sprintf("%d", cid) != chainID {
			mark(false, fmt.Sprintf("chain id mismatch: RPC=%d env=%s — scout will reject all ads", cid, chainID))
		}
	}
	fmt.Println()

	fmt.Println("3) Nostr relay reachability")
	for _, r := range relays {
		host, port := relayHostPort(r)
		if host == "" {
			mark(false, fmt.Sprintf("%s: could not parse host", r))
			continue
		}
		d := net.Dialer{Timeout: 3 * time.Second}
		conn, derr := d.Dial("tcp", net.JoinHostPort(host, port))
		if derr != nil {
			mark(false, fmt.Sprintf("%s: %v", r, derr))
			continue
		}
		conn.Close()
		mark(true, fmt.Sprintf("tcp %s:%s reachable", host, port))
	}
	fmt.Println()

	if s := strings.TrimSpace(*sidecarURL); s != "" {
		fmt.Println("4) mln-sidecar /v1/balance")
		base := strings.TrimRight(s, "/")
		req, _ := http.NewRequest("GET", base+"/v1/balance", nil)
		c := &http.Client{Timeout: 3 * time.Second}
		resp, serr := c.Do(req)
		if serr != nil {
			mark(false, fmt.Sprintf("GET %s/v1/balance: %v", base, serr))
		} else {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 {
				mark(true, fmt.Sprintf("%s/v1/balance: %s", base, truncate(string(body), 120)))
			} else {
				mark(false, fmt.Sprintf("%s/v1/balance: HTTP %d", base, resp.StatusCode))
			}
		}
		fmt.Println()
	}

	fmt.Println("5) Scout (live Nostr subscribe + LitVM verify)")
	if *skipScout {
		fmt.Println("  [info] -skip-scout set; skipping")
	} else {
		cfg, cerr := loadScoutConfig()
		if cerr != nil {
			mark(false, fmt.Sprintf("config: %v", cerr))
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			res, serr := scout.Run(ctx, cfg)
			cancel()
			if serr != nil {
				mark(false, fmt.Sprintf("scout: %v", serr))
			} else {
				verified := len(res.Verified)
				rejected := len(res.Rejected)
				withTor := 0
				for _, m := range res.Verified {
					if strings.TrimSpace(m.Tor) != "" {
						withTor++
					}
				}
				mark(verified >= 3, fmt.Sprintf("verified=%d (need >=3 for pathfind), with tor=%d, rejected=%d", verified, withTor, rejected))
				if verified > 0 && withTor < verified {
					fmt.Printf("  [info] %d verified maker(s) have empty `tor` — pathfind will skip them\n", verified-withTor)
				}
			}
		}
	}
	fmt.Println()

	fmt.Printf("Summary: %d ok, %d failed.\n", ok, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func rpcChainID(rpcURL string) (uint64, error) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`)
	req, _ := http.NewRequest("POST", rpcURL, bytes.NewReader(body))
	req.Header.Set("content-type", "application/json")
	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, fmt.Errorf("decode: %w (body=%s)", err, truncate(string(raw), 80))
	}
	if out.Error != nil {
		return 0, fmt.Errorf("%s", out.Error.Message)
	}
	if !strings.HasPrefix(out.Result, "0x") {
		return 0, fmt.Errorf("unexpected result %q", out.Result)
	}
	var n uint64
	if _, err := fmt.Sscanf(out.Result, "0x%x", &n); err != nil {
		return 0, fmt.Errorf("parse: %w", err)
	}
	return n, nil
}

func relayHostPort(raw string) (string, string) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", ""
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "wss" || u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return host, port
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
