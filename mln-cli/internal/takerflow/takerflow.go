// Package takerflow orchestrates scout → pathfind → forger for the taker wallet and APIs.
package takerflow

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/config"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/forger"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/scout"
)

// ScoutResult is JSON-friendly output after Nostr + LitVM verification.
type ScoutResult struct {
	Verified []scout.VerifiedMaker `json:"verified"`
	Rejected []scout.Rejection     `json:"rejected"`
}

// RouteResult is a picked 3-hop route plus scout snapshot metadata.
type RouteResult struct {
	Route          *pathfind.Route `json:"route"`
	VerifiedCount  int             `json:"verifiedCount"`
	RejectedCount  int             `json:"rejectedCount"`
	FeeSumSat      uint64          `json:"feeSumSat"`
}

// SendParams carries forger execution inputs (sidecar URL may override settings default).
type SendParams struct {
	Destination string `json:"destination"`
	AmountSat   uint64 `json:"amountSat"`
	SidecarURL  string `json:"sidecarUrl,omitempty"`
}

// SendResult is returned when the sidecar accepts the payload.
type SendResult struct {
	Detail    string `json:"detail,omitempty"`
	EpochNote string `json:"epochNote"`
}

const epochNote = "MWEB batch processing in coinswapd runs at epoch cutover (local midnight); there may be no immediate chain txid."

// Scout runs discovery with the given network settings.
func Scout(ctx context.Context, net config.NetworkSettings) (*ScoutResult, error) {
	scfg, err := net.ToScoutConfig()
	if err != nil {
		return nil, err
	}
	res, err := scout.Run(ctx, scfg)
	if err != nil {
		return nil, err
	}
	return &ScoutResult{Verified: res.Verified, Rejected: res.Rejected}, nil
}

// BuildRoute runs scout then pathfind.PickRoute.
func BuildRoute(ctx context.Context, net config.NetworkSettings, rng *rand.Rand) (*RouteResult, error) {
	sr, err := Scout(ctx, net)
	if err != nil {
		return nil, err
	}
	if len(sr.Verified) < 3 {
		return nil, fmt.Errorf("takerflow: need at least 3 verified makers, got %d", len(sr.Verified))
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	route, err := pathfind.PickRoute(sr.Verified, rng)
	if err != nil {
		return nil, err
	}
	return &RouteResult{
		Route:         route,
		VerifiedCount: len(sr.Verified),
		RejectedCount: len(sr.Rejected),
		FeeSumSat:     route.FeeSumSat,
	}, nil
}

// DryRunRoute validates Tor endpoints on a route (no network besides already-built route).
func DryRunRoute(route *pathfind.Route) (*forger.DryRunResult, error) {
	return forger.DryRun(route)
}

// Send submits route to the local sidecar. Context should carry the desired deadline (e.g. 10s).
func Send(ctx context.Context, net config.NetworkSettings, route *pathfind.Route, p SendParams) (*SendResult, error) {
	if route == nil {
		return nil, fmt.Errorf("takerflow: nil route")
	}
	sidecar := sidecarURL(p.SidecarURL, net.DefaultSidecar())
	execRes, err := forger.Execute(ctx, route, sidecar, p.Destination, p.AmountSat)
	if err != nil {
		return nil, err
	}
	out := &SendResult{EpochNote: epochNote}
	if execRes != nil {
		out.Detail = execRes.Detail
	}
	return out, nil
}

func sidecarURL(override, def string) string {
	if s := strings.TrimSpace(override); s != "" {
		return s
	}
	return def
}
