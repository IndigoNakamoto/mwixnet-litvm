package identity

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestAddressFromHexPrivateKey_anvil0(t *testing.T) {
	t.Parallel()
	// Well-known Anvil account #0 (see Foundry docs).
	const hexKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	want := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	got, err := AddressFromHexPrivateKey(hexKey)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %s want %s", got.Hex(), want.Hex())
	}
	got2, err := AddressFromHexPrivateKey("0x" + hexKey)
	if err != nil {
		t.Fatal(err)
	}
	if got2 != want {
		t.Fatal()
	}
}

func TestAddressFromHexPrivateKey_badLength(t *testing.T) {
	t.Parallel()
	_, err := AddressFromHexPrivateKey("abcd")
	if err == nil {
		t.Fatal("expected error")
	}
}
