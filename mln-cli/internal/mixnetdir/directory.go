// Package mixnetdir loads a permissioned CoinSwap hop list (no Nostr, no LitVM).
package mixnetdir

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/forger"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/scout"
)

const expectedHops = 3

// Hop is one published mixnode (Tor JSON-RPC + onion X25519 + fee).
type Hop struct {
	Tor              string `json:"tor"`
	SwapX25519PubHex string `json:"swapX25519PubHex"`
	FeeMinSat        uint64 `json:"feeMinSat"`
}

// Directory is the committed/operator hop list for Phase 3-mix.
type Directory struct {
	Notes     string `json:"notes,omitempty"`
	AmountSat uint64 `json:"amountSat"`
	Hops      []Hop  `json:"hops"`
}

// Load reads and validates a hop directory JSON file.
func Load(path string) (*Directory, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("mixnetdir: read %s: %w", path, err)
	}
	var d Directory
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("mixnetdir: json: %w", err)
	}
	if err := Validate(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

// Validate checks hop count, keys, fees, and one-denomination amount.
func Validate(d *Directory) error {
	if d == nil {
		return fmt.Errorf("mixnetdir: nil directory")
	}
	if len(d.Hops) != expectedHops {
		return fmt.Errorf("mixnetdir: need exactly %d hops, got %d", expectedHops, len(d.Hops))
	}
	if d.AmountSat == 0 {
		return fmt.Errorf("mixnetdir: amountSat must be positive")
	}
	var feeSum uint64
	for i, h := range d.Hops {
		if strings.TrimSpace(h.Tor) == "" {
			return fmt.Errorf("mixnetdir: hop %d: tor is required", i)
		}
		k := strings.TrimSpace(h.SwapX25519PubHex)
		if len(k) != 64 {
			return fmt.Errorf("mixnetdir: hop %d: swapX25519PubHex must be 64 hex digits", i)
		}
		for _, c := range k {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return fmt.Errorf("mixnetdir: hop %d: swapX25519PubHex must be lowercase hex", i)
			}
		}
		if h.FeeMinSat == 0 {
			return fmt.Errorf("mixnetdir: hop %d: feeMinSat must be positive (cover coinswapd backward() floor)", i)
		}
		next := feeSum + h.FeeMinSat
		if next < feeSum {
			return fmt.Errorf("mixnetdir: hop %d: fee overflow", i)
		}
		feeSum = next
	}
	if feeSum >= d.AmountSat {
		return fmt.Errorf("mixnetdir: sum of feeMinSat (%d) must be less than amountSat (%d)", feeSum, d.AmountSat)
	}
	return nil
}

// ToPathfindRoute builds a forger-compatible route.json (no LitVM operator/stake).
func ToPathfindRoute(d *Directory) (*pathfind.Route, error) {
	if err := Validate(d); err != nil {
		return nil, err
	}
	var hops [3]scout.VerifiedMaker
	var feeSum uint64
	for i, h := range d.Hops {
		hops[i] = scout.VerifiedMaker{
			Tor:              strings.TrimSpace(h.Tor),
			FeeMinSat:        h.FeeMinSat,
			SwapX25519PubHex: strings.TrimSpace(h.SwapX25519PubHex),
			Stake:            "0",
		}
		feeSum += h.FeeMinSat
	}
	return &pathfind.Route{Hops: hops, FeeSumSat: feeSum}, nil
}

type slimHop struct {
	Tor              string `json:"tor"`
	FeeMinSat        uint64 `json:"feeMinSat"`
	SwapX25519PubHex string `json:"swapX25519PubHex"`
}

type slimRoute struct {
	Hops      [3]slimHop `json:"hops"`
	FeeSumSat uint64     `json:"feeSumSat"`
}

// MarshalRouteJSON writes forger-compatible route.json without LitVM operator/stake fields.
func MarshalRouteJSON(d *Directory) ([]byte, error) {
	route, err := ToPathfindRoute(d)
	if err != nil {
		return nil, err
	}
	var out slimRoute
	out.FeeSumSat = route.FeeSumSat
	for i, h := range route.Hops {
		out.Hops[i] = slimHop{
			Tor:              h.Tor,
			FeeMinSat:        h.FeeMinSat,
			SwapX25519PubHex: h.SwapX25519PubHex,
		}
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// ToForgerPayload emits POST /v1/swap JSON with no epochId/accuser/swapId/operator.
func ToForgerPayload(d *Directory, dest string) (*forger.RequestPayload, error) {
	if err := Validate(d); err != nil {
		return nil, err
	}
	if strings.TrimSpace(dest) == "" {
		return nil, fmt.Errorf("mixnetdir: destination is required for swap payload")
	}
	p := &forger.RequestPayload{
		Route:       make([]forger.HopRequest, 0, expectedHops),
		Destination: strings.TrimSpace(dest),
		Amount:      d.AmountSat,
	}
	for _, h := range d.Hops {
		p.Route = append(p.Route, forger.HopRequest{
			Tor:              strings.TrimSpace(h.Tor),
			FeeMinSat:        h.FeeMinSat,
			SwapX25519PubHex: strings.TrimSpace(h.SwapX25519PubHex),
		})
	}
	return p, nil
}
