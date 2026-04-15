package makerad

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	gnostr "github.com/nbd-wtf/go-nostr"
)

func TestParseDTag(t *testing.T) {
	chain, op, err := ParseDTag("mln:v1:31337:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	if err != nil {
		t.Fatal(err)
	}
	if chain != "31337" {
		t.Fatalf("chain %q", chain)
	}
	want := "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
	if strings.ToLower(op.Hex()) != want {
		t.Fatalf("op %s want %s", op.Hex(), want)
	}
}

func TestFixtureMakerAdJSON(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Caller")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	p := filepath.Join(root, "nostr", "fixtures", "valid", "maker_ad.json")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Skipf("fixture %s: %v", p, err)
	}
	var wrap struct {
		Kind    int         `json:"kind"`
		Tags    gnostr.Tags `json:"tags"`
		Content string      `json:"content"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		t.Fatal(err)
	}
	ev := &gnostr.Event{
		Kind:    wrap.Kind,
		Tags:    wrap.Tags,
		Content: wrap.Content,
	}
	parsed, err := ParseAd(ev)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Content.Litvm.ChainID != "31337" {
		t.Fatalf("chainId %q", parsed.Content.Litvm.ChainID)
	}
	if parsed.Content.Litvm.Registry != "0x5fbdb2315678afecb367f032d93f642f64180aa3" {
		t.Fatalf("registry %q", parsed.Content.Litvm.Registry)
	}
	if parsed.Content.Fees == nil || parsed.Content.Fees.Min != 1 || parsed.Content.Fees.Max != 99 {
		t.Fatalf("fees %+v", parsed.Content.Fees)
	}
}

func TestFixtureMakerAdV2SealedJSON(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Caller")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	p := filepath.Join(root, "nostr", "fixtures", "valid", "maker_ad_v2_sealed.json")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Skipf("fixture %s: %v", p, err)
	}
	var wrap struct {
		Kind    int         `json:"kind"`
		Tags    gnostr.Tags `json:"tags"`
		Content string      `json:"content"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		t.Fatal(err)
	}
	ev := &gnostr.Event{
		Kind:    wrap.Kind,
		Tags:    wrap.Tags,
		Content: wrap.Content,
	}
	parsed, err := ParseAd(ev)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Content.V != 2 {
		t.Fatalf("v %d", parsed.Content.V)
	}
	if parsed.Content.Reachability == nil || parsed.Content.Reachability.Scheme != "nip44-v2" {
		t.Fatalf("reachability %+v", parsed.Content.Reachability)
	}
	if strings.TrimSpace(parsed.Content.Tor) != "" || strings.TrimSpace(parsed.Content.SwapX25519PubHex) != "" {
		t.Fatalf("expected no cleartext tor/swap in sealed v2 fixture")
	}
}

func TestParseAdV2SealedRejectsCleartextMix(t *testing.T) {
	ev := &gnostr.Event{
		Kind: KindMakerAd,
		Tags: gnostr.Tags{
			{"d", "mln:v1:31337:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"},
			{"t", TagTMakerAd},
		},
		Content: `{"v":2,"litvm":{"chainId":"31337","registry":"0x5fbdb2315678afecb367f032d93f642f64180aa3","grievanceCourt":"0xe7f1725e7734ce288f8367e1bb143e90bb3f0512"},"tor":"http://x.onion","reachability":{"scheme":"nip44-v2","ciphertext":"abcdefgh"}}`,
	}
	if _, err := ParseAd(ev); err == nil {
		t.Fatal("expected error when mixing sealed reachability with cleartext tor")
	}
}
