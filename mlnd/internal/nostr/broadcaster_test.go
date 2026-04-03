package nostr

import (
	"encoding/json"
	"io"
	"log"
	"strings"
	"testing"
	"time"
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
	if ev.Kind != kindMakerAd {
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
	if tVal != tagTMakerAd {
		t.Fatalf("t tag: got %q want %q", tVal, tagTMakerAd)
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

func uint64Ptr(u uint64) *uint64 {
	return &u
}
