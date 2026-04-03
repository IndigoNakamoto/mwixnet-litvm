package config

import "testing"

func TestValidateSelfInclusion(t *testing.T) {
	t.Parallel()
	base := NetworkSettings{
		NostrRelays:  []string{"wss://x"},
		LitvmChainID: "1",
		LitvmHTTPURL: "http://rpc",
		RegistryAddr: "0x0000000000000000000000000000000000000001",
	}
	if err := base.ValidateSelfInclusion(); err != nil {
		t.Fatal(err)
	}
	on := base
	on.SelfIncludedRouting = true
	if err := on.ValidateSelfInclusion(); err == nil {
		t.Fatal("expected error without key")
	}
	on.OperatorEthPrivateKeyHex = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	if err := on.ValidateSelfInclusion(); err != nil {
		t.Fatal(err)
	}
}
