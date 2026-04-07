package config

import (
	"testing"
)

func TestScoutFromEnv_singleRelayURLFallback(t *testing.T) {
	t.Setenv("MLN_NOSTR_RELAYS", "")
	t.Setenv("MLN_NOSTR_RELAY_URL", "wss://relay.example/nostr")
	t.Setenv("MLN_LITVM_CHAIN_ID", "31337")
	t.Setenv("MLN_LITVM_HTTP_URL", "http://127.0.0.1:8545")
	t.Setenv("MLN_REGISTRY_ADDR", "0x0000000000000000000000000000000000000001")

	relays, chainID, rpcURL, reg, court, _, err := ScoutFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if len(relays) != 1 || relays[0] != "wss://relay.example/nostr" {
		t.Fatalf("relays = %v", relays)
	}
	if chainID != "31337" || rpcURL != "http://127.0.0.1:8545" || reg != "0x0000000000000000000000000000000000000001" {
		t.Fatalf("chain/rpc/reg = %s %s %s", chainID, rpcURL, reg)
	}
	_ = court
}

func TestScoutFromEnv_relaysPreferredOverRelayURL(t *testing.T) {
	t.Setenv("MLN_NOSTR_RELAYS", "wss://a,wss://b")
	t.Setenv("MLN_NOSTR_RELAY_URL", "wss://ignored")
	t.Setenv("MLN_LITVM_CHAIN_ID", "1")
	t.Setenv("MLN_LITVM_HTTP_URL", "http://x")
	t.Setenv("MLN_REGISTRY_ADDR", "0x0000000000000000000000000000000000000002")

	relays, _, _, _, _, _, err := ScoutFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if len(relays) != 2 || relays[0] != "wss://a" || relays[1] != "wss://b" {
		t.Fatalf("relays = %v", relays)
	}
}
