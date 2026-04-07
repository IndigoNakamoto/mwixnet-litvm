// Package mlnroute holds the MLN HTTP / JSON-RPC route body (parity with mln-sidecar SwapRequest).
package mlnroute

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const ExpectedHops = 3

var swapX25519PubHexRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Hop is one hop from MLN route JSON.
type Hop struct {
	Tor              string `json:"tor"`
	FeeMinSat        uint64 `json:"feeMinSat"`
	SwapX25519PubHex string `json:"swapX25519PubHex,omitempty"`
	// Operator is the hop maker's LitVM registry address (0x). Required on every hop when epochId/accuser/swapId are set.
	Operator string `json:"operator,omitempty"`
}

// Request is the mweb_submitRoute parameter object.
type Request struct {
	Route       []Hop  `json:"route"`
	Destination string `json:"destination"`
	Amount      uint64 `json:"amount"`
	// LitVM coordination (optional together; when any is set, all must be valid — used for receipt / grievance threading).
	EpochID string `json:"epochId,omitempty"`
	Accuser string `json:"accuser,omitempty"`
	SwapID  string `json:"swapId,omitempty"`
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
	return validateLitVMMetadata(req)
}

func validateLitVMMetadata(req *Request) error {
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
	for i, h := range req.Route {
		addr, err := normalizeHexAddress(strings.TrimSpace(h.Operator))
		if err != nil {
			return fmt.Errorf("hop %d: operator: %w", i, err)
		}
		if addr == (common.Address{}) {
			return fmt.Errorf("hop %d: operator is required when LitVM route metadata is set", i)
		}
	}
	return nil
}

func normalizeHexAddress(s string) (common.Address, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return common.Address{}, fmt.Errorf("empty address")
	}
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		s = "0x" + s
	}
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("invalid hex address %q", s)
	}
	return common.HexToAddress(s), nil
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

// PeerOperatorsFromRequest returns per-hop LitVM operators when LitVM metadata is present; otherwise zero addresses.
func PeerOperatorsFromRequest(req *Request) ([3]common.Address, error) {
	var out [3]common.Address
	if strings.TrimSpace(req.EpochID) == "" {
		return out, nil
	}
	for i, h := range req.Route {
		addr, err := normalizeHexAddress(strings.TrimSpace(h.Operator))
		if err != nil {
			return out, fmt.Errorf("hop %d: operator: %w", i, err)
		}
		out[i] = addr
	}
	return out, nil
}
