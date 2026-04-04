package forger

import (
	"strings"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
)

// NormalizeMixEndpoint trims whitespace and adds "http://" when no URI scheme is present,
// matching mln-sidecar and coinswapd rpc.Dial expectations for maker Tor / mix API URLs.
func NormalizeMixEndpoint(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return s
	}
	if strings.Contains(s, "://") {
		return s
	}
	return "http://" + s
}

// NormalizeRouteTor applies NormalizeMixEndpoint to each hop in place before POST/dry-run.
func NormalizeRouteTor(route *pathfind.Route) {
	if route == nil {
		return
	}
	for i := range route.Hops {
		route.Hops[i].Tor = NormalizeMixEndpoint(route.Hops[i].Tor)
	}
}
