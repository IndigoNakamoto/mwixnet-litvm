package registry

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestABI_packDeposit(t *testing.T) {
	data, err := parsedABI.Pack("deposit")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 4 {
		t.Fatal("short calldata")
	}
	// d0e30db0 = deposit()
	want := "d0e30db0"
	if got := hex.EncodeToString(data[:4]); got != want {
		t.Fatalf("selector = %s want %s", got, want)
	}
}

func TestABI_packRegisterMaker(t *testing.T) {
	h := common.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	data, err := parsedABI.Pack("registerMaker", h)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 4 {
		t.Fatal("short calldata")
	}
	// registerMaker(bytes32) — verify fixed args length (32 after selector)
	if len(data) != 4+32 {
		t.Fatalf("calldata len %d want %d", len(data), 4+32)
	}
}
