package nostridentity

import (
	"os"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func TestPubkeyHexFromEnv_npub(t *testing.T) {
	sk := nostr.GeneratePrivateKey()
	pk, err := nostr.GetPublicKey(sk)
	if err != nil {
		t.Fatal(err)
	}
	npub, err := nip19.EncodePublicKey(pk)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MLN_NOSTR_PUBKEY_HEX", npub)
	t.Setenv("MLN_NOSTR_NSEC", "")
	got, err := PubkeyHexFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got != pk {
		t.Fatalf("got %s want %s", got, pk)
	}
}

func TestPubkeyHexFromEnv_explicitPubkey(t *testing.T) {
	sk := nostr.GeneratePrivateKey()
	pk, err := nostr.GetPublicKey(sk)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MLN_NOSTR_PUBKEY_HEX", pk)
	t.Setenv("MLN_NOSTR_NSEC", "")
	got, err := PubkeyHexFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got != pk {
		t.Fatalf("got %s want %s", got, pk)
	}
}

func TestPubkeyHexFromEnv_nsec(t *testing.T) {
	sk := nostr.GeneratePrivateKey()
	nsec, err := nip19.EncodePrivateKey(sk)
	if err != nil {
		t.Fatal(err)
	}
	want, err := nostr.GetPublicKey(sk)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MLN_NOSTR_PUBKEY_HEX", "")
	t.Setenv("MLN_NOSTR_NSEC", nsec)
	got, err := PubkeyHexFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestPubkeyHexFromEnv_missing(t *testing.T) {
	_ = os.Unsetenv("MLN_NOSTR_PUBKEY_HEX")
	_ = os.Unsetenv("MLN_NOSTR_NSEC")
	if _, err := PubkeyHexFromEnv(); err == nil {
		t.Fatal("expected error")
	}
}
