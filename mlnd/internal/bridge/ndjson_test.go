package bridge

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/litvm"
	"github.com/ethereum/go-ethereum/common"
)

func TestParseReceiptLine_ok(t *testing.T) {
	m := map[string]any{
		"epochId":               "99",
		"accuser":               "0x0000000000000000000000000000000000000aa1",
		"accusedMaker":          "0x0000000000000000000000000000000000000bb2",
		"hopIndex":              0,
		"peeledCommitment":      "0x" + string(bytesRepeat('c', 64)),
		"forwardCiphertextHash": "0x" + string(bytesRepeat('d', 64)),
		"nextHopPubkey":         "npub1test",
		"signature":             "sig1",
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := ParseReceiptLine(raw)
	if err != nil {
		t.Fatal(err)
	}
	if rec.EpochID.Cmp(big.NewInt(99)) != 0 {
		t.Fatalf("epoch %v", rec.EpochID)
	}
	wantA := common.HexToAddress("0x0000000000000000000000000000000000000aa1")
	if rec.Accuser != wantA {
		t.Fatalf("accuser %s", rec.Accuser.Hex())
	}
	if rec.HopIndex != 0 || rec.NextHopPubkey != "npub1test" || rec.Signature != "sig1" {
		t.Fatalf("fields %+v", rec)
	}
	ev := litvm.ComputeEvidenceHash(rec.EvidencePreimage)
	if ev == (common.Hash{}) {
		t.Fatal("zero evidence hash")
	}
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func TestParseReceiptLine_empty(t *testing.T) {
	_, err := ParseReceiptLine([]byte("   "))
	if err == nil {
		t.Fatal("want error")
	}
}

func TestParseReceiptLine_badHop(t *testing.T) {
	m := map[string]any{
		"epochId":               "1",
		"accuser":               "0x0000000000000000000000000000000000000001",
		"accusedMaker":          "0x0000000000000000000000000000000000000002",
		"hopIndex":              256,
		"peeledCommitment":      "0x" + string(bytesRepeat('1', 64)),
		"forwardCiphertextHash": "0x" + string(bytesRepeat('2', 64)),
		"nextHopPubkey":         "k",
		"signature":             "s",
	}
	raw, _ := json.Marshal(m)
	_, err := ParseReceiptLine(raw)
	if err == nil {
		t.Fatal("want error")
	}
}
