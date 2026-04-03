// Package forger is the Phase 10.3 hook for MWEB onion submission via Tor to coinswapd.
// Full execution requires a patched coinswapd and SOCKS proxy; see research/COINSWAPD_TEARDOWN.md.
package forger

import (
	"fmt"
	"os"
	"strings"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
)

// DryRun checks that each hop exposes a Tor mix API URL and prints the next integration steps.
func DryRun(route *pathfind.Route) error {
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

	fmt.Fprintln(os.Stderr, "Forger dry-run: route OK (3 hops with Tor endpoints).")
	fmt.Fprintln(os.Stderr, "MWEB onion build and POST to N1 are not implemented in this binary.")
	fmt.Fprintln(os.Stderr, "Use a patched coinswapd per research/COINSWAPD_TEARDOWN.md; route Tor URLs:")
	for i, h := range route.Hops {
		fmt.Fprintf(os.Stderr, "  N%d: %s\n", i+1, h.Tor)
	}
	return nil
}
