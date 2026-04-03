// Package forger is the Phase 10.3 hook for MWEB onion submission via Tor to coinswapd.
// Route submission uses an MLN HTTP extension on a local sidecar; see research/COINSWAPD_TEARDOWN.md.
package forger

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
)

func validateTorHops(route *pathfind.Route) error {
	if route == nil {
		return fmt.Errorf("forger: nil route")
	}
	var missing []int
	for i, h := range route.Hops {
		if strings.TrimSpace(h.Tor) == "" {
			missing = append(missing, i+1)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("forger: hops %v missing tor URL in maker ad", missing)
	}
	return nil
}

// DryRun checks that each hop exposes a Tor mix API URL and prints route details.
func DryRun(route *pathfind.Route) error {
	if err := validateTorHops(route); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Forger dry-run: route OK (3 hops with Tor endpoints).")
	fmt.Fprintln(os.Stderr, "To POST this route to a local coinswapd MLN sidecar, use -dry-run=false with -dest and -amount (see COINSWAPD_TEARDOWN.md).")
	fmt.Fprintln(os.Stderr, "Route Tor URLs:")
	for i, h := range route.Hops {
		fmt.Fprintf(os.Stderr, "  N%d: %s\n", i+1, h.Tor)
	}
	return nil
}

// Execute validates the route and POSTs it to the sidecar URL (destination MWEB address and amount in satoshis).
func Execute(ctx context.Context, route *pathfind.Route, sidecarURL, dest string, amount uint64) error {
	if strings.TrimSpace(dest) == "" {
		return fmt.Errorf("forger: destination MWEB address is required (-dest)")
	}
	if amount == 0 {
		return fmt.Errorf("forger: amount must be greater than 0 (-amount, satoshis)")
	}
	if err := validateTorHops(route); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Submitting route to coinswapd sidecar at %s...\n", sidecarURL)

	client := NewSidecarClient(sidecarURL)
	resp, err := client.SubmitRoute(ctx, route, dest, amount)
	if err != nil {
		return err
	}

	if !resp.Ok {
		msg := strings.TrimSpace(resp.Error)
		if msg == "" {
			msg = "sidecar returned ok=false"
		}
		if d := strings.TrimSpace(resp.Detail); d != "" {
			return fmt.Errorf("forger: %s (%s)", msg, d)
		}
		return fmt.Errorf("forger: %s", msg)
	}

	fmt.Fprintln(os.Stderr, "[SUCCESS] Route accepted by local sidecar.")
	if d := strings.TrimSpace(resp.Detail); d != "" {
		fmt.Fprintf(os.Stderr, "Detail: %s\n", d)
	}
	fmt.Fprintln(os.Stderr, "Note: MWEB batch processing in coinswapd runs at epoch cutover (local midnight); there may be no immediate chain txid.")
	return nil
}
