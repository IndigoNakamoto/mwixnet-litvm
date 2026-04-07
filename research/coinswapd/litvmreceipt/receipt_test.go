package litvmreceipt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/ltcmweb/ltcd/ltcutil/mweb/mw"
	"golang.org/x/crypto/sha3"
)

func TestPeeledCommitmentHash_knownVector(t *testing.T) {
	t.Parallel()
	var c mw.Commitment
	c[0] = 8
	copy(c[1:], bytes.Repeat([]byte{0x03}, 32))
	got := PeeledCommitmentHash(c)
	want := sha256.Sum256(c[:])
	if got != want {
		t.Fatalf("PeeledCommitmentHash: got %x want %x", got, want)
	}
}

func TestForwardCiphertextHash_keccak(t *testing.T) {
	t.Parallel()
	p := []byte("mln-appendix-13-payload")
	got := ForwardCiphertextHash(p)
	h := sha3.NewLegacyKeccak256()
	h.Write(p)
	var want [32]byte
	copy(want[:], h.Sum(nil))
	if got != want {
		t.Fatalf("ForwardCiphertextHash: got %x want %x", got, want)
	}
}

func TestMarshalSwapForwardFailureReceipt_roundTrip(t *testing.T) {
	t.Parallel()
	var c mw.Commitment
	c[0] = 9
	copy(c[1:], bytes.Repeat([]byte{0xab}, 32))
	raw, err := MarshalSwapForwardFailureReceipt(
		"7",
		"0x1111111111111111111111111111111111111111",
		"swap-roundtrip",
		"0x000000000000000000000000000000000000dEaD",
		2,
		c,
		[]byte("P"),
		hex.EncodeToString(bytes.Repeat([]byte{0x01}, 32)),
		"rpc_application",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"signature":"unsigned-swap-forward-failure-v1"`)) {
		t.Fatalf("missing sentinel: %s", string(raw))
	}
	if !bytes.Contains(raw, []byte(`"forwardFailureClass":"rpc_application"`)) {
		t.Fatalf("missing class: %s", string(raw))
	}
}
