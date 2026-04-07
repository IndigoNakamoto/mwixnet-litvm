package main

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const goldenReceiptAccusedFallback = "0x0000000000000000000000000000000000000001"

// goldenReceiptAccusedMaker returns receipt.accusedMaker for LitVM golden receipts.
// When route hop 0 carries a valid hex operator (parity with MockBridge), use it;
// otherwise the legacy placeholder address for RPC tests that omit operator.
func goldenReceiptAccusedMaker(sr submitRouteBody) string {
	if len(sr.Route) == 0 {
		return goldenReceiptAccusedFallback
	}
	op := strings.TrimSpace(sr.Route[0].Operator)
	if op == "" {
		return goldenReceiptAccusedFallback
	}
	if !strings.HasPrefix(op, "0x") && !strings.HasPrefix(op, "0X") {
		op = "0x" + op
	}
	if !common.IsHexAddress(op) {
		return goldenReceiptAccusedFallback
	}
	return strings.ToLower(common.HexToAddress(op).Hex())
}
