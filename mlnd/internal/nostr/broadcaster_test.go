package nostr

import (
	"encoding/json"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/makerad"
)

func TestTorOnionWithOptionalPort(t *testing.T) {
	base := "http://v3abcdefghijklmnop1234567890abcdef1234567890abcdefgh.onion"
	got := torOnionWithOptionalPort(base, "18081")
	want := base + ":18081"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if torOnionWithOptionalPort(base+":18081", "9999") != base+":18081" {
		t.Fatal("expected existing port unchanged")
	}
	if torOnionWithOptionalPort("", "80") != "" {
		t.Fatal("empty base")
	}
	if torOnionWithOptionalPort(base, "") != base {
		t.Fatal("empty port env")
	}
}

func TestLoadBroadcasterFromEnv_relaysWithoutNsec(t *testing.T) {
	t.Setenv("MLND_NOSTR_RELAYS", "wss://example.com")
	t.Setenv("MLND_NOSTR_NSEC", "")
	// Clear related vars so we do not accidentally pick up host env from a full broadcaster config.
	t.Setenv("MLND_LITVM_CHAIN_ID", "")
	t.Setenv("MLND_REGISTRY_ADDR", "")
	bc, err := LoadBroadcasterFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if bc != nil {
		t.Fatal("expected nil broadcaster when MLND_NOSTR_NSEC is empty")
	}
}

func TestLoadBroadcasterFromEnv_noRelays(t *testing.T) {
	t.Setenv("MLND_NOSTR_RELAYS", "")
	bc, err := LoadBroadcasterFromEnv()
	if err != nil || bc != nil {
		t.Fatalf("got bc=%v err=%v", bc, err)
	}
}

func TestDTag(t *testing.T) {
	got := DTag("31337", "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	want := "mln:v1:31337:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
	if got != want {
		t.Fatalf("DTag: got %q want %q", got, want)
	}
}

func TestNormalizeETHAddr(t *testing.T) {
	got, err := normalizeETHAddr("F39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	if err != nil {
		t.Fatal(err)
	}
	want := "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildMakerAdEvent_shapeAndSignature(t *testing.T) {
	// Deterministic 32-byte secret (do not use in production).
	sec := strings.Repeat("3a", 32)
	b := &Broadcaster{
		cfg: BroadcasterConfig{
			ChainID:        "31337",
			Registry:       "0x5fbdb2315678afecb367f032d93f642f64180aa3",
			GrievanceCourt: "0xe7f1725e7734ce288f8367e1bb143e90bb3f0512",
			Operator:       "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
			TorOnion:       "http://abcdefghijklmnop123456789012345678901234567890abcdefgh.onion:18081",
			FeeMinSat:      uint64Ptr(1),
			FeeMaxSat:      uint64Ptr(99),
			Capabilities:   []string{"mweb-coinswap-v0"},
			ClientName:     "mlnd-test",
			ClientVersion:  "0",
		},
		secHex: sec,
		log:    log.New(io.Discard, "", 0),
	}

	ts := time.Unix(1700000000, 0).UTC()
	ev, err := b.BuildMakerAdEvent(ts)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != makerad.KindMakerAd {
		t.Fatalf("kind: got %d", ev.Kind)
	}
	if int64(ev.CreatedAt) != ts.Unix() {
		t.Fatalf("created_at: got %d want %d", ev.CreatedAt, ts.Unix())
	}

	var dVal, tVal string
	for _, tag := range ev.Tags {
		if len(tag) >= 2 && tag[0] == "d" {
			dVal = tag[1]
		}
		if len(tag) >= 2 && tag[0] == "t" {
			tVal = tag[1]
		}
	}
	if dVal != DTag("31337", "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266") {
		t.Fatalf("d tag: got %q", dVal)
	}
	if tVal != makerad.TagTMakerAd {
		t.Fatalf("t tag: got %q want %q", tVal, makerad.TagTMakerAd)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(ev.Content), &payload); err != nil {
		t.Fatal(err)
	}
	if string(payload["v"]) != "1" {
		t.Fatalf("content v: %s", payload["v"])
	}
	var lit map[string]string
	if err := json.Unmarshal(payload["litvm"], &lit); err != nil {
		t.Fatal(err)
	}
	if lit["chainId"] != "31337" {
		t.Fatalf("litvm.chainId: %q", lit["chainId"])
	}
	if lit["registry"] != "0x5fbdb2315678afecb367f032d93f642f64180aa3" {
		t.Fatalf("litvm.registry not lowercase: %q", lit["registry"])
	}
	var tor string
	if err := json.Unmarshal(payload["tor"], &tor); err != nil {
		t.Fatal(err)
	}
	wantTor := "http://abcdefghijklmnop123456789012345678901234567890abcdefgh.onion:18081"
	if tor != wantTor {
		t.Fatalf("content.tor: got %q want %q", tor, wantTor)
	}
}

func TestBuildMakerAdEvent_includesSwapX25519(t *testing.T) {
	sec := strings.Repeat("3b", 32)
	swap := strings.Repeat("ab", 32)
	b := &Broadcaster{
		cfg: BroadcasterConfig{
			ChainID:          "31337",
			Registry:         "0x5fbdb2315678afecb367f032d93f642f64180aa3",
			GrievanceCourt:   "0xe7f1725e7734ce288f8367e1bb143e90bb3f0512",
			Operator:         "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
			SwapX25519PubHex: swap,
		},
		secHex: sec,
		log:    log.New(io.Discard, "", 0),
	}
	ev, err := b.BuildMakerAdEvent(time.Unix(1700000001, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		SwapX25519PubHex string `json:"swapX25519PubHex"`
	}
	if err := json.Unmarshal([]byte(ev.Content), &body); err != nil {
		t.Fatal(err)
	}
	if body.SwapX25519PubHex != swap {
		t.Fatalf("swap key: got %q want %q", body.SwapX25519PubHex, swap)
	}
}

func TestNewBroadcaster_derivesPublicKey(t *testing.T) {
	sec := strings.Repeat("3a", 32)
	b := NewBroadcaster(BroadcasterConfig{AuthEnabled: true}, nil, sec, 0, log.New(io.Discard, "", 0))
	if b.pubHex == "" {
		t.Fatal("pubHex should be derived from secHex")
	}
	if len(b.pubHex) != 64 {
		t.Fatalf("pubHex length %d, want 64", len(b.pubHex))
	}
}

func TestAuthKeys_returnsMaterialWhenEnabled(t *testing.T) {
	sec := strings.Repeat("3a", 32)
	b := NewBroadcaster(BroadcasterConfig{AuthEnabled: true}, nil, sec, 0, log.New(io.Discard, "", 0))
	s, p := b.AuthKeys()
	if s != sec {
		t.Fatalf("secHex: got %q want %q", s, sec)
	}
	if p != b.pubHex {
		t.Fatalf("pubHex: got %q want %q", p, b.pubHex)
	}
}

func TestAuthKeys_returnsEmptyWhenDisabled(t *testing.T) {
	sec := strings.Repeat("3a", 32)
	b := NewBroadcaster(BroadcasterConfig{AuthEnabled: false}, nil, sec, 0, log.New(io.Discard, "", 0))
	s, p := b.AuthKeys()
	if s != "" || p != "" {
		t.Fatalf("expected empty keys when AUTH disabled, got sec=%q pub=%q", s, p)
	}
}

func TestAuthKeys_nilBroadcaster(t *testing.T) {
	var b *Broadcaster
	s, p := b.AuthKeys()
	if s != "" || p != "" {
		t.Fatalf("expected empty keys from nil broadcaster, got sec=%q pub=%q", s, p)
	}
}

func TestLoadBroadcasterFromEnv_authEnabled(t *testing.T) {
	sec := strings.Repeat("3a", 32)
	t.Setenv("MLND_NOSTR_RELAYS", "wss://example.com")
	t.Setenv("MLND_NOSTR_NSEC", sec)
	t.Setenv("MLND_LITVM_CHAIN_ID", "31337")
	t.Setenv("MLND_REGISTRY_ADDR", "0x5fbdb2315678afecb367f032d93f642f64180aa3")
	t.Setenv("MLND_COURT_ADDR", "0xe7f1725e7734ce288f8367e1bb143e90bb3f0512")
	t.Setenv("MLND_OPERATOR_ADDR", "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	t.Setenv("MLND_NOSTR_AUTH", "true")
	bc, err := LoadBroadcasterFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if bc == nil {
		t.Fatal("expected non-nil broadcaster")
	}
	if !bc.cfg.AuthEnabled {
		t.Fatal("expected AuthEnabled=true")
	}
	s, p := bc.AuthKeys()
	if s == "" || p == "" {
		t.Fatal("expected non-empty auth keys when AUTH is on")
	}
}

func TestLoadBroadcasterFromEnv_authInvalid(t *testing.T) {
	sec := strings.Repeat("3a", 32)
	t.Setenv("MLND_NOSTR_RELAYS", "wss://example.com")
	t.Setenv("MLND_NOSTR_NSEC", sec)
	t.Setenv("MLND_LITVM_CHAIN_ID", "31337")
	t.Setenv("MLND_REGISTRY_ADDR", "0x5fbdb2315678afecb367f032d93f642f64180aa3")
	t.Setenv("MLND_COURT_ADDR", "0xe7f1725e7734ce288f8367e1bb143e90bb3f0512")
	t.Setenv("MLND_OPERATOR_ADDR", "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	t.Setenv("MLND_NOSTR_AUTH", "notabool")
	_, err := LoadBroadcasterFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid MLND_NOSTR_AUTH")
	}
	if !strings.Contains(err.Error(), "MLND_NOSTR_AUTH") {
		t.Fatalf("error should mention MLND_NOSTR_AUTH: %v", err)
	}
}

func uint64Ptr(u uint64) *uint64 {
	return &u
}
