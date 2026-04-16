//go:build wails

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/config"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/forger"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/takerflow"
)

// App is the Wails-bound API for the taker wallet.
type App struct {
	ctx context.Context

	mu    sync.Mutex
	store *settingsStore
}

// NewApp constructs the desktop app (settings loaded lazily).
func NewApp() (*App, error) {
	st, err := newSettingsStore()
	if err != nil {
		return nil, err
	}
	return &App{store: st}, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// LoadSettings reads persisted network settings (defaults if missing).
func (a *App) LoadSettings() (config.NetworkSettings, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.store.load()
}

// SaveSettings persists network settings for the next session.
func (a *App) SaveSettings(net config.NetworkSettings) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := net.Validate(); err != nil {
		return err
	}
	if err := net.ValidateSelfInclusion(); err != nil {
		return err
	}
	return a.store.save(net)
}

// Scout runs Nostr discovery and LitVM verification using saved settings.
func (a *App) Scout() (*takerflow.ScoutResult, error) {
	net, err := a.loadValidated()
	if err != nil {
		return nil, err
	}
	timeout, err := net.ScoutTimeoutDuration()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	defer cancel()
	return takerflow.Scout(ctx, net)
}

// BuildRoute discovers makers and picks a 3-hop route (same policy as mln-cli pathfind).
func (a *App) BuildRoute() (*takerflow.RouteResult, error) {
	net, err := a.loadValidated()
	if err != nil {
		return nil, err
	}
	timeout, err := net.ScoutTimeoutDuration()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	defer cancel()
	return takerflow.BuildRoute(ctx, net, nil)
}

// FetchMwebBalance queries the sidecar for MWEB balance available to coinswap (GET /v1/balance).
func (a *App) FetchMwebBalance(sidecarURL string) (*forger.MwebBalance, error) {
	a.mu.Lock()
	net, err := a.store.load()
	a.mu.Unlock()
	if err != nil {
		return nil, err
	}
	u := strings.TrimSpace(sidecarURL)
	if u == "" {
		u = net.DefaultSidecar()
	}
	if err := warnNonLoopback(u, net.DefaultSidecar()); err != nil {
		return nil, err
	}
	d, err := net.ForgerContextTimeout()
	if err != nil {
		d = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(a.ctx, d)
	defer cancel()
	cl := forger.NewSidecarClient(u).HTTPClient
	return forger.FetchMwebBalance(ctx, u, cl)
}

// DryRunRouteJSON validates Tor endpoints on a JSON route (from BuildRoute).
func (a *App) DryRunRouteJSON(routeJSON string) (*forger.DryRunResult, error) {
	route, err := takerflow.ParseRouteJSON(routeJSON)
	if err != nil {
		return nil, err
	}
	return takerflow.DryRunRoute(route)
}

// Send posts the route to the local coinswapd MLN sidecar.
func (a *App) Send(routeJSON string, dest string, amountSat uint64, sidecarURL string) (*takerflow.SendResult, error) {
	net, err := a.loadValidated()
	if err != nil {
		return nil, err
	}
	route, err := takerflow.ParseRouteJSON(routeJSON)
	if err != nil {
		return nil, err
	}
	if err := warnNonLoopback(sidecarURL, net.DefaultSidecar()); err != nil {
		return nil, err
	}
	forgerTimeout, err := net.ForgerContextTimeout()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, forgerTimeout)
	defer cancel()
	return takerflow.Send(ctx, net, route, takerflow.SendParams{
		Destination: strings.TrimSpace(dest),
		AmountSat:   amountSat,
		SidecarURL:  strings.TrimSpace(sidecarURL),
	})
}

// LocalLabResult summarizes a Tier 1 stub-lab invocation for the UI.
type LocalLabResult struct {
	ExitCode int    `json:"exitCode"`
	TailLog  string `json:"tailLog"`
	ScriptPath string `json:"scriptPath"`
}

// RunLocalLab executes the Tier 1 stub lab (scripts/e2e-mweb-handoff-stub.sh with E2E_MWEB_FULL=1)
// against the workspace rooted at repoRoot. If repoRoot is empty, the method looks for the script
// relative to the current working directory (typical when the wallet is launched from the repo).
//
// This is a convenience for first-time users: it drives the same path the `make e2e-tier1` target
// uses, but from inside the desktop app. It requires Docker and the same prerequisites as the
// script itself; errors are surfaced in the returned LocalLabResult.TailLog.
func (a *App) RunLocalLab(repoRoot string) (*LocalLabResult, error) {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		cwd, _ := os.Getwd()
		root = cwd
	}
	script := filepath.Join(root, "scripts", "e2e-mweb-handoff-stub.sh")
	if _, err := os.Stat(script); err != nil {
		return nil, fmt.Errorf("tier 1 script not found at %s — pass repoRoot or launch mln-wallet from the repo root", script)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", script)
	cmd.Env = append(os.Environ(), "E2E_MWEB_FULL=1")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()

	tail := string(out)
	if len(tail) > 4000 {
		tail = "…\n" + tail[len(tail)-4000:]
	}
	exit := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			return &LocalLabResult{ExitCode: -1, TailLog: tail + "\n" + err.Error(), ScriptPath: script}, nil
		}
	}
	return &LocalLabResult{ExitCode: exit, TailLog: tail, ScriptPath: script}, nil
}

func (a *App) loadValidated() (config.NetworkSettings, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	net, err := a.store.load()
	if err != nil {
		return net, err
	}
	if err := net.Validate(); err != nil {
		return net, fmt.Errorf("settings: %w (save network settings in the app)", err)
	}
	return net, nil
}

func warnNonLoopback(override, def string) error {
	u := strings.TrimSpace(override)
	if u == "" {
		u = def
	}
	lu := strings.ToLower(u)
	if strings.HasPrefix(lu, "http://") && !sidecarLikelyLocal(lu) {
		return fmt.Errorf("sidecar URL %q is not localhost or .onion; ensure you trust this endpoint", u)
	}
	return nil
}

func sidecarLikelyLocal(lu string) bool {
	return strings.Contains(lu, "127.0.0.1") || strings.Contains(lu, "localhost") || strings.Contains(lu, ".onion")
}
