package makerad

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ComputeNostrKeyHash returns keccak256(P) for the 32-byte x-only secp256k1 pubkey (hex, with or without 0x).
func ComputeNostrKeyHash(pubkeyHex string) (common.Hash, error) {
	pubkeyHex = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(pubkeyHex, "0x"), "0X"))
	if len(pubkeyHex) != 64 {
		return common.Hash{}, fmt.Errorf("makerad: pubkey must be 64 hex chars, got %d", len(pubkeyHex))
	}
	pubkeyBytes := common.FromHex("0x" + pubkeyHex)
	if len(pubkeyBytes) != 32 {
		return common.Hash{}, fmt.Errorf("makerad: invalid pubkey hex")
	}
	return crypto.Keccak256Hash(pubkeyBytes), nil
}
