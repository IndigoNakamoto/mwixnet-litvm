package store

import (
	"math/big"
	"path/filepath"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/litvm"
	"github.com/ethereum/go-ethereum/common"
)

func TestStore_saveAndGet_roundTrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	epochID := big.NewInt(42)
	accuser := common.HexToAddress("0x0000000000000000000000000000beef")
	accused := common.HexToAddress("0x0000000000000000000000000000cafe")
	peeled := common.BytesToHash(common.LeftPadBytes(big.NewInt(0x1111).Bytes(), 32))
	forward := common.BytesToHash(common.LeftPadBytes(big.NewInt(0x2222).Bytes(), 32))

	rec := ReceiptRecord{
		EvidencePreimage: litvm.EvidencePreimage{
			EpochID:               epochID,
			Accuser:               accuser,
			AccusedMaker:          accused,
			HopIndex:              2,
			PeeledCommitment:      peeled,
			ForwardCiphertextHash: forward,
		},
		NextHopPubkey: "npub1example",
		Signature:     "deadbeef",
	}

	ev := litvm.ComputeEvidenceHash(rec.EvidencePreimage)

	if ins, err := s.SaveReceipt(rec); err != nil || !ins {
		t.Fatalf("SaveReceipt: inserted=%v err=%v", ins, err)
	}

	got, err := s.GetByEvidenceHash(ev)
	if err != nil {
		t.Fatal(err)
	}
	if got.EpochID.Cmp(epochID) != 0 {
		t.Fatalf("epoch: got %s want %s", got.EpochID, epochID)
	}
	if got.Accuser != accuser || got.AccusedMaker != accused {
		t.Fatalf("addresses: %+v", got)
	}
	if got.HopIndex != 2 {
		t.Fatalf("hop: got %d", got.HopIndex)
	}
	if got.PeeledCommitment != peeled || got.ForwardCiphertextHash != forward {
		t.Fatalf("hashes: %+v", got)
	}
	if got.NextHopPubkey != "npub1example" || got.Signature != "deadbeef" {
		t.Fatalf("receipt fields: %+v", got)
	}

	// Idempotent second insert
	if ins, err := s.SaveReceipt(rec); err != nil || ins {
		t.Fatalf("second SaveReceipt: inserted=%v err=%v", ins, err)
	}
}

func TestStore_GetByEvidenceHash_missing(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(filepath.Join(dir, "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	_, err = s.GetByEvidenceHash(common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	if err == nil {
		t.Fatal("expected error")
	}
}
