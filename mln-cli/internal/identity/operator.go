package identity

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// AddressFromHexPrivateKey derives the LitVM maker address from a secp256k1 private key (same as Ethereum EOA).
func AddressFromHexPrivateKey(hexKey string) (common.Address, error) {
	h := strings.TrimSpace(hexKey)
	h = strings.TrimPrefix(h, "0x")
	h = strings.TrimPrefix(h, "0X")
	if len(h) != 64 {
		return common.Address{}, fmt.Errorf("identity: expect 64 hex chars for secp256k1 private key, got %d", len(h))
	}
	key, err := crypto.HexToECDSA(h)
	if err != nil {
		return common.Address{}, fmt.Errorf("identity: parse private key: %w", err)
	}
	return crypto.PubkeyToAddress(key.PublicKey), nil
}
