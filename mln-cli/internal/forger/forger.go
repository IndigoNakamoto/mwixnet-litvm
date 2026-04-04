// Package forger is the Phase 10.3 hook for MWEB onion submission via Tor to coinswapd.
// Route submission uses an MLN HTTP extension on a local sidecar; see research/COINSWAPD_TEARDOWN.md.
package forger

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
)

func validateTorHops(route *pathfind.Route) error {
	if route == nil {
		return fmt.Errorf("forger: nil route")
	}
	NormalizeRouteTor(route)
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

// DryRun checks that each hop exposes a Tor mix API URL and returns a structured summary.
func DryRun(route *pathfind.Route) (*DryRunResult, error) {
	if err := validateTorHops(route); err != nil {
		return nil, err
	}
	out := &DryRunResult{Hops: make([]HopTorSummary, 0, len(route.Hops))}
	for i, h := range route.Hops {
		out.Hops = append(out.Hops, HopTorSummary{Index: i + 1, Tor: h.Tor})
	}
	return out, nil
}

// DryRunCLI prints human-oriented dry-run output to w (typically os.Stderr).
func DryRunCLI(route *pathfind.Route, w io.Writer) error {
	res, err := DryRun(route)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "Forger dry-run: route OK (3 hops with Tor endpoints).")
	fmt.Fprintln(w, "To POST this route to a local coinswapd MLN sidecar, use -dry-run=false with -dest and -amount (see COINSWAPD_TEARDOWN.md).")
	fmt.Fprintln(w, "Route Tor URLs:")
	for _, h := range res.Hops {
		fmt.Fprintf(w, "  N%d: %s\n", h.Index, h.Tor)
	}
	return nil
}

// Execute validates the route and POSTs it to the sidecar URL (destination MWEB address and amount in satoshis).
func Execute(ctx context.Context, route *pathfind.Route, sidecarURL, dest string, amount uint64) (*ExecuteResult, error) {
	return ExecuteWithBatchOptions(ctx, route, sidecarURL, dest, amount, nil)
}

// ExecuteWithBatchOptions POSTs the route, then optionally triggers mweb_runBatch and/or waits for pendingOnions==0.
func ExecuteWithBatchOptions(ctx context.Context, route *pathfind.Route, sidecarURL, dest string, amount uint64, batch *BatchOptions) (*ExecuteResult, error) {
	if strings.TrimSpace(dest) == "" {
		return nil, fmt.Errorf("forger: destination MWEB address is required (-dest)")
	}
	if amount == 0 {
		return nil, fmt.Errorf("forger: amount must be greater than 0 (-amount, satoshis)")
	}
	if err := validateTorHops(route); err != nil {
		return nil, err
	}

	client := NewSidecarClient(sidecarURL)
	resp, err := client.SubmitRoute(ctx, route, dest, amount)
	if err != nil {
		return nil, err
	}

	if !resp.Ok {
		msg := strings.TrimSpace(resp.Error)
		if msg == "" {
			msg = "sidecar returned ok=false"
		}
		if d := strings.TrimSpace(resp.Detail); d != "" {
			return nil, fmt.Errorf("forger: %s (%s)", msg, d)
		}
		return nil, fmt.Errorf("forger: %s", msg)
	}

	out := &ExecuteResult{Detail: strings.TrimSpace(resp.Detail)}
	if batch == nil {
		return out, nil
	}

	if batch.TriggerBatch {
		bresp, err := client.RunBatch(ctx, sidecarURL)
		if err != nil {
			return nil, err
		}
		if bresp != nil && strings.TrimSpace(bresp.Detail) != "" {
			if out.Detail != "" {
				out.Detail += "; "
			}
			out.Detail += strings.TrimSpace(bresp.Detail)
		}
	}

	if batch.WaitPendingZero {
		poll := batch.PollInterval
		if poll <= 0 {
			poll = 2 * time.Second
		}
		timeout := batch.Timeout
		if timeout <= 0 {
			timeout = 2 * time.Minute
		}
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			st, err := client.GetRouteStatus(ctx, sidecarURL)
			if err != nil {
				return nil, err
			}
			if st.PendingOnions == 0 {
				out.PendingCleared = true
				return out, nil
			}
			t := time.NewTimer(poll)
			select {
			case <-ctx.Done():
				t.Stop()
				return nil, ctx.Err()
			case <-t.C:
			}
			t.Stop()
		}
		return nil, fmt.Errorf("forger: -wait-batch timeout (%s) with pendingOnions still > 0 (try -trigger-batch or wait for coinswapd batch)", timeout)
	}

	return out, nil
}

// ExecuteCLI runs Execute and prints progress and outcome to w (typically os.Stderr).
func ExecuteCLI(ctx context.Context, route *pathfind.Route, sidecarURL, dest string, amount uint64, w io.Writer) error {
	return ExecuteCLIWithBatch(ctx, route, sidecarURL, dest, amount, nil, w)
}

// ExecuteCLIWithBatch runs ExecuteWithBatchOptions with optional batch / wait flags.
func ExecuteCLIWithBatch(ctx context.Context, route *pathfind.Route, sidecarURL, dest string, amount uint64, batch *BatchOptions, w io.Writer) error {
	fmt.Fprintf(w, "Submitting route to coinswapd sidecar at %s...\n", sidecarURL)
	res, err := ExecuteWithBatchOptions(ctx, route, sidecarURL, dest, amount, batch)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "[SUCCESS] Route accepted by local sidecar.")
	if res != nil && res.Detail != "" {
		fmt.Fprintf(w, "Detail: %s\n", res.Detail)
	}
	if batch != nil && batch.TriggerBatch {
		fmt.Fprintln(w, "Batch trigger sent (POST /v1/route/batch → mweb_runBatch).")
	}
	if batch != nil && batch.WaitPendingZero {
		if res != nil && res.PendingCleared {
			fmt.Fprintln(w, "[SUCCESS] Route status reports pendingOnions=0 (local DB queue cleared after broadcast or stub).")
		}
	}
	if batch == nil || !batch.WaitPendingZero {
		fmt.Fprintln(w, "Tip: use -trigger-batch / -wait-batch for POST /v1/route/batch and GET /v1/route/status; full P2P hops still need live maker RPCs.")
	}
	return nil
}
