package litvmevidence

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestBuildDefenseData_roundTrip(t *testing.T) {
	r := &ReceiptForDefense{
		EpochID:               big.NewInt(99),
		Accuser:               common.HexToAddress("0x1000000000000000000000000000000000000001"),
		AccusedMaker:          common.HexToAddress("0x2000000000000000000000000000000000000002"),
		HopIndex:              3,
		PeeledCommitment:      common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		ForwardCiphertextHash: common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		NextHopPubkey:         "npub1test",
		Signature:             "sig-deadbeef",
	}
	data, err := BuildDefenseData(r)
	if err != nil {
		t.Fatal(err)
	}
	data2, err := BuildDefenseData(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != len(data2) || string(data) != string(data2) {
		t.Fatal("BuildDefenseData not deterministic")
	}
	got, err := UnpackDefenseV1(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.EpochID.Cmp(r.EpochID) != 0 || got.Accuser != r.Accuser || got.AccusedMaker != r.AccusedMaker {
		t.Fatalf("round trip addresses/epoch %+v", got)
	}
	if got.HopIndex != r.HopIndex {
		t.Fatal("hopIndex")
	}
	if got.PeeledCommitment != r.PeeledCommitment || got.ForwardCiphertextHash != r.ForwardCiphertextHash {
		t.Fatal("hashes")
	}
	if got.NextHopPubkey != r.NextHopPubkey || got.Signature != r.Signature {
		t.Fatal("utf8 fields")
	}
}

func TestValidateReceiptForGrievance_ok(t *testing.T) {
	epoch := big.NewInt(42)
	accuser := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	accused := common.HexToAddress("0x00000000000000000000000000000000000000bb")
	hop := uint8(0)
	peeled := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	fwd := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	pre := EvidencePreimage{EpochID: epoch, Accuser: accuser, AccusedMaker: accused, HopIndex: hop, PeeledCommitment: peeled, ForwardCiphertextHash: fwd}
	evHash := ComputeEvidenceHash(pre)
	gid := ComputeGrievanceID(accuser, accused, epoch, evHash)

	ev := &GrievanceOpened{
		GrievanceID:  gid,
		Accuser:      accuser,
		Accused:      accused,
		EpochID:      new(big.Int).Set(epoch),
		EvidenceHash: evHash,
		Deadline:     big.NewInt(9999999999),
	}
	r := &ReceiptForDefense{
		EpochID:               new(big.Int).Set(epoch),
		Accuser:               accuser,
		AccusedMaker:          accused,
		HopIndex:              hop,
		PeeledCommitment:      peeled,
		ForwardCiphertextHash: fwd,
		NextHopPubkey:         "k",
		Signature:             "s",
	}
	if err := ValidateReceiptForGrievance(ev, r, accused); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReceiptForGrievance_badHash(t *testing.T) {
	epoch := big.NewInt(1)
	accuser := common.HexToAddress("0x1")
	accused := common.HexToAddress("0x2")
	ev := &GrievanceOpened{
		GrievanceID:  common.Hash{1},
		Accuser:      accuser,
		Accused:      accused,
		EpochID:      epoch,
		EvidenceHash: common.Hash{9},
		Deadline:     big.NewInt(9),
	}
	r := &ReceiptForDefense{
		EpochID:               epoch,
		Accuser:               accuser,
		AccusedMaker:          accused,
		HopIndex:              0,
		PeeledCommitment:      common.Hash{},
		ForwardCiphertextHash: common.Hash{},
	}
	if err := ValidateReceiptForGrievance(ev, r, accused); err == nil {
		t.Fatal("expected error")
	}
}

func TestChainTimeBeforeDeadline(t *testing.T) {
	h := &types.Header{Time: 100}
	if !ChainTimeBeforeDeadline(h, big.NewInt(101)) {
		t.Fatal("want before")
	}
	if ChainTimeBeforeDeadline(h, big.NewInt(100)) {
		t.Fatal("want not before at equality")
	}
	if ChainTimeBeforeDeadline(h, big.NewInt(99)) {
		t.Fatal("want after deadline")
	}
}
