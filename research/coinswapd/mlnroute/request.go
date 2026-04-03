// Package mlnroute holds the MLN HTTP / JSON-RPC route body (parity with mln-sidecar SwapRequest).
package mlnroute

import (
	"fmt"
	"regexp"
	"strings"
)

const ExpectedHops = 3

var swapX25519PubHexRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Hop is one hop from MLN route JSON.
type Hop struct {
	Tor              string `json:"tor"`
	FeeMinSat        uint64 `json:"feeMinSat"`
	SwapX25519PubHex string `json:"swapX25519PubHex,omitempty"`
}

// Request is the mweb_submitRoute parameter object.
type Request struct {
	Route       []Hop  `json:"route"`
	Destination string `json:"destination"`
	Amount      uint64 `json:"amount"`
}

// Validate checks structural and fee rules (parity with mln-sidecar ValidateSwapRequest).
func Validate(req *Request) error {
	if req == nil {
		return fmt.Errorf("nil request")
	}
	if len(req.Route) != ExpectedHops {
		return fmt.Errorf("route must have exactly %d hops, got %d", ExpectedHops, len(req.Route))
	}
	if strings.TrimSpace(req.Destination) == "" {
		return fmt.Errorf("destination is required")
	}
	if req.Amount == 0 {
		return fmt.Errorf("amount must be positive")
	}
	var feeSum uint64
	for i, h := range req.Route {
		if strings.TrimSpace(h.Tor) == "" {
			return fmt.Errorf("hop %d: tor URL is required", i)
		}
		next, err := addFee(feeSum, h.FeeMinSat)
		if err != nil {
			return fmt.Errorf("hop %d: fee overflow", i)
		}
		feeSum = next
	}
	if feeSum > req.Amount {
		return fmt.Errorf("sum of feeMinSat (%d) exceeds amount (%d)", feeSum, req.Amount)
	}
	var withKey int
	for i, h := range req.Route {
		k := strings.TrimSpace(h.SwapX25519PubHex)
		if k == "" {
			continue
		}
		if !swapX25519PubHexRE.MatchString(k) {
			return fmt.Errorf("hop %d: swapX25519PubHex must be 64 lowercase hex digits", i)
		}
		withKey++
	}
	if withKey != 0 && withKey != len(req.Route) {
		return fmt.Errorf("swapX25519PubHex must be set on all hops or omitted on all")
	}
	return nil
}

func addFee(sum, add uint64) (uint64, error) {
	if add != 0 && sum > ^uint64(0)-add {
		return 0, fmt.Errorf("overflow")
	}
	return sum + add, nil
}

// FeeSum returns sum of per-hop fees (after Validate).
func FeeSum(req *Request) uint64 {
	var s uint64
	for _, h := range req.Route {
		s += h.FeeMinSat
	}
	return s
}
