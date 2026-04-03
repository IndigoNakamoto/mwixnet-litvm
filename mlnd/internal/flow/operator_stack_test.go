package flow

import (
	"encoding/json"
	"io"
	"log"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/litvm"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/nostr"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/store"
	"github.com/ethereum/go-ethereum/common"
)

// TestOperatorStack_SQLiteEvidenceAndMakerAd ties the receipt vault, canonical evidenceHash,
// and a signed kind-31250 maker ad (no chain RPC, no live relays).
func TestOperatorStack_SQLiteEvidenceAndMakerAd(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "op.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	epochID := big.NewInt(99)
	accuser := common.HexToAddress("0x0000000000000000000000000000000000000aa1")
	accused := common.HexToAddress("0x0000000000000000000000000000000000000bb2")
	peeled := common.BytesToHash(common.LeftPadBytes([]byte{0xca, 0xfe}, 32))
	fwd := common.BytesToHash(common.LeftPadBytes([]byte{0xbe, 0xef}, 32))

	rec := store.ReceiptRecord{
		EvidencePreimage: litvm.EvidencePreimage{
			EpochID:               epochID,
			Accuser:               accuser,
			AccusedMaker:          accused,
			HopIndex:              0,
			PeeledCommitment:      peeled,
			ForwardCiphertextHash: fwd,
		},
		NextHopPubkey: "npub1stacktest",
		Signature:     "sig-stacktest",
	}
	evHash := litvm.ComputeEvidenceHash(rec.EvidencePreimage)
	if err := s.SaveReceipt(rec); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetByEvidenceHash(evHash)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccusedMaker != accused || got.Accuser != accuser {
		t.Fatalf("lookup mismatch: %+v", got)
	}

	torURL := "http://v3abc123xyz567890123456789012345678901234567890abcdefgh.onion:18081"
	sec := strings.Repeat("3a", 32)
	lg := log.New(io.Discard, "", 0)
	bc := nostr.NewBroadcaster(nostr.BroadcasterConfig{
		ChainID:        "31337",
		Registry:       "0x5fbdb2315678afecb367f032d93f642f64180aa3",
		GrievanceCourt: "0xe7f1725e7734ce288f8367e1bb143e90bb3f0512",
		Operator:       "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
		TorOnion:       torURL,
		FeeMinSat:      uint64Ptr(1),
		FeeMaxSat:      uint64Ptr(10),
		Capabilities:   []string{"mweb-coinswap-v0"},
		ClientName:     "mlnd-flow-test",
		ClientVersion:  "0",
	}, nil, sec, time.Hour, lg)

	ev, err := bc.BuildMakerAdEvent(time.Unix(1700000000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != 31250 {
		t.Fatalf("kind %d", ev.Kind)
	}
	var payload struct {
		Tor   string `json:"tor"`
		Litvm struct {
			ChainID string `json:"chainId"`
		} `json:"litvm"`
	}
	if err := json.Unmarshal([]byte(ev.Content), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Tor != torURL {
		t.Fatalf("tor in content: %q", payload.Tor)
	}
	if payload.Litvm.ChainID != "31337" {
		t.Fatalf("chainId %q", payload.Litvm.ChainID)
	}
}

func uint64Ptr(u uint64) *uint64 {
	return &u
}

// TestOperatorStack_ReceiptValidateAndDefense builds a receipt and synthetic GrievanceOpened
// correlators, then runs validation and defense packing (no chain RPC, no relays).
func TestOperatorStack_ReceiptValidateAndDefense(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "def.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	epochID := big.NewInt(42)
	accuser := common.HexToAddress("0x0000000000000000000000000000000000000a01")
	accused := common.HexToAddress("0x0000000000000000000000000000000000000b02")
	peeled := common.BytesToHash(common.LeftPadBytes([]byte{0x11}, 32))
	fwd := common.BytesToHash(common.LeftPadBytes([]byte{0x22}, 32))

	rec := store.ReceiptRecord{
		EvidencePreimage: litvm.EvidencePreimage{
			EpochID:               epochID,
			Accuser:               accuser,
			AccusedMaker:          accused,
			HopIndex:              0,
			PeeledCommitment:      peeled,
			ForwardCiphertextHash: fwd,
		},
		NextHopPubkey: "npub1defense",
		Signature:     "sig-defense",
	}
	evidenceHash := litvm.ComputeEvidenceHash(rec.EvidencePreimage)
	if err := s.SaveReceipt(rec); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetByEvidenceHash(evidenceHash)
	if err != nil {
		t.Fatal(err)
	}

	grievanceID := litvm.ComputeGrievanceID(accuser, accused, epochID, evidenceHash)
	ev := &litvm.GrievanceEvent{
		GrievanceID:  grievanceID,
		Accuser:      accuser,
		Accused:      accused,
		EpochID:      epochID,
		EvidenceHash: evidenceHash,
		Deadline:     big.NewInt(1 << 62),
	}
	if err := litvm.ValidateReceiptForGrievance(ev, got, accused); err != nil {
		t.Fatal(err)
	}
	defense, err := litvm.BuildDefenseData(got)
	if err != nil {
		t.Fatal(err)
	}
	if len(defense) == 0 {
		t.Fatal("empty defense payload")
	}
}
