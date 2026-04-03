package config

import (
	"os"
	"testing"
)

func TestOnboardFromEnv_ok(t *testing.T) {
	t.Setenv("MLN_LITVM_HTTP_URL", "http://127.0.0.1:8545")
	t.Setenv("MLN_REGISTRY_ADDR", "0x0000000000000000000000000000000000000001")
	t.Setenv("MLN_LITVM_CHAIN_ID", "31337")
	t.Setenv("MLN_OPERATOR_ETH_KEY", "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	env, err := OnboardFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if env.RPCHTTP != "http://127.0.0.1:8545" {
		t.Fatal("rpc")
	}
	if env.ChainID.String() != "31337" {
		t.Fatal("chain id")
	}
	if len(env.PrivateKeyHex) != 64 {
		t.Fatal("key hex")
	}
}

func TestOnboardFromEnv_missingRPC(t *testing.T) {
	_ = os.Unsetenv("MLN_LITVM_HTTP_URL")
	t.Setenv("MLN_REGISTRY_ADDR", "0x0000000000000000000000000000000000000001")
	t.Setenv("MLN_LITVM_CHAIN_ID", "1")
	t.Setenv("MLN_OPERATOR_ETH_KEY", "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if _, err := OnboardFromEnv(); err == nil {
		t.Fatal("expected error")
	}
}
