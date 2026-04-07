package mweb

import (
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const expectedHops = 3

// NormalizeMixEndpoint trims whitespace and, if there is no URI scheme, prefixes "http://"
// so values match what go-ethereum rpc.Dial and coinswapd use for hop RPC URLs (Tor ads often omit the scheme).
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

// NormalizeSwapRequestHops applies NormalizeMixEndpoint to each hop's Tor field in place.
func NormalizeSwapRequestHops(req *SwapRequest) {
	if req == nil {
		return
	}
	for i := range req.Route {
		req.Route[i].Tor = NormalizeMixEndpoint(req.Route[i].Tor)
	}
}

// swapX25519PubHexRE matches a 32-byte Curve25519 public key as 64 lowercase hex digits (see research/COINSWAPD_MLN_FORK_SPEC.md).
var swapX25519PubHexRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// HopInput is one hop from the MLN sidecar JSON (matches mln-cli RequestPayload.route).
type HopInput struct {
	Tor              string `json:"tor"`
	FeeMinSat        uint64 `json:"feeMinSat"`
	SwapX25519PubHex string `json:"swapX25519PubHex,omitempty"`
}

// SwapRequest is the POST /v1/swap body (matches mln-cli forger.RequestPayload).
type SwapRequest struct {
	Route       []HopInput `json:"route"`
	Destination string     `json:"destination"`
	Amount      uint64     `json:"amount"`
	EpochID     string     `json:"epochId,omitempty"`
	Accuser     string     `json:"accuser,omitempty"`
	SwapID      string     `json:"swapId,omitempty"`
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
	return validateSwapLitVMMetadata(req)
}

func validateSwapLitVMMetadata(req *SwapRequest) error {
	e := strings.TrimSpace(req.EpochID)
	a := strings.TrimSpace(req.Accuser)
	s := strings.TrimSpace(req.SwapID)
	if e == "" && a == "" && s == "" {
		return nil
	}
	if e == "" || a == "" || s == "" {
		return fmt.Errorf("epochId, accuser, and swapId must all be set together when providing LitVM route metadata")
	}
	epoch, ok := new(big.Int).SetString(e, 10)
	if !ok || epoch.Sign() < 0 {
		return fmt.Errorf("invalid epochId %q", req.EpochID)
	}
	h := strings.TrimSpace(a)
	if !strings.HasPrefix(h, "0x") && !strings.HasPrefix(h, "0X") {
		h = "0x" + h
	}
	if !common.IsHexAddress(h) {
		return fmt.Errorf("invalid accuser address %q", req.Accuser)
	}
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("swapId must be non-empty when providing LitVM route metadata")
	}
	return nil
}

func addFee(sum, add uint64) (uint64, error) {
	if add != 0 && sum > ^uint64(0)-add {
		return 0, fmt.Errorf("overflow")
	}
	return sum + add, nil
}

// BuildMockOnion maps a validated route into a mock onion. Logs omit hop endpoints (Tor URLs)
// so operator logs do not record maker infrastructure or route shape beyond hop count.
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
	log.Printf("[Sidecar] mock: built %d-hop mock onion (entry endpoint not logged)", o.HopCount)
	return o
}
