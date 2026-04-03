//go:build wails

package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

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
