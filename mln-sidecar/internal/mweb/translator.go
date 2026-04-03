package mweb

import (
	"fmt"
	"log"
	"strings"
)

const expectedHops = 3

// HopInput is one hop from the MLN sidecar JSON (matches mln-cli RequestPayload.route).
type HopInput struct {
	Tor       string `json:"tor"`
	FeeMinSat uint64 `json:"feeMinSat"`
}

// SwapRequest is the POST /v1/swap body (matches mln-cli forger.RequestPayload).
type SwapRequest struct {
	Route       []HopInput `json:"route"`
	Destination string     `json:"destination"`
	Amount      uint64     `json:"amount"`
}

// MockOnion is a stand-in for coinswapd's onion.Onion until the real engine is wired.
type MockOnion struct {
	HopCount  int
	EntryTor  string
	HopTors   []string
	Dest      string
	AmountSat uint64
	FeeSumSat uint64
}

// ValidateSwapRequest checks structural and fee-budget constraints for the mock engine.
func ValidateSwapRequest(req *SwapRequest) error {
	if req == nil {
		return fmt.Errorf("nil request")
	}
	if len(req.Route) != expectedHops {
		return fmt.Errorf("route must have exactly %d hops, got %d", expectedHops, len(req.Route))
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
		fee, err := addFee(feeSum, h.FeeMinSat)
		if err != nil {
			return fmt.Errorf("hop %d: fee overflow", i)
		}
		feeSum = fee
	}
	if feeSum > req.Amount {
		return fmt.Errorf("sum of feeMinSat (%d) exceeds amount (%d)", feeSum, req.Amount)
	}
	return nil
}

func addFee(sum, add uint64) (uint64, error) {
	if add != 0 && sum > ^uint64(0)-add {
		return 0, fmt.Errorf("overflow")
	}
	return sum + add, nil
}

// BuildMockOnion maps a validated route into a mock onion and logs the simulated handoff.
func BuildMockOnion(req *SwapRequest) MockOnion {
	tors := make([]string, len(req.Route))
	for i := range req.Route {
		tors[i] = req.Route[i].Tor
	}
	var feeSum uint64
	for _, h := range req.Route {
		feeSum += h.FeeMinSat
	}
	o := MockOnion{
		HopCount:  len(req.Route),
		EntryTor:  tors[0],
		HopTors:   tors,
		Dest:      req.Destination,
		AmountSat: req.Amount,
		FeeSumSat: feeSum,
	}
	log.Printf("[Sidecar] Translated %d-hop route into MWEB Onion. Target N1: %s", o.HopCount, o.EntryTor)
	return o
}
