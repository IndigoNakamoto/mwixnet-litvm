package litvm

import (
	"math/big"
	"reflect"
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
	out, err := defensePackArgs.Unpack(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("unpack len %d", len(out))
	}
	assertUnpackedDefenseV1(t, out[0], r)
}

// geth unpacks the tuple as a struct with same field order as the ABI tuple.
func assertUnpackedDefenseV1(t *testing.T, unpacked any, want *ReceiptForDefense) {
	t.Helper()
	rv := reflect.ValueOf(unpacked)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct || rv.NumField() < 9 {
		t.Fatalf("unexpected unpack shape kind=%s numField=%d", rv.Kind(), rv.NumField())
	}
	if uint8(rv.Field(0).Uint()) != 1 {
		t.Fatalf("version got %d", rv.Field(0).Uint())
	}
	epoch := rv.Field(1).Interface().(*big.Int)
	if epoch.Cmp(want.EpochID) != 0 {
		t.Fatalf("epochId %s", epoch.String())
	}
	if rv.Field(2).Interface().(common.Address) != want.Accuser {
		t.Fatal("accuser")
	}
	if rv.Field(3).Interface().(common.Address) != want.AccusedMaker {
		t.Fatal("accusedMaker")
	}
	if uint8(rv.Field(4).Uint()) != want.HopIndex {
		t.Fatal("hopIndex")
	}
	if hashFromABI(rv.Field(5).Interface()) != want.PeeledCommitment {
		t.Fatal("peeledCommitment")
	}
	if hashFromABI(rv.Field(6).Interface()) != want.ForwardCiphertextHash {
		t.Fatal("forwardCiphertextHash")
	}
	if string(rv.Field(7).Interface().([]byte)) != want.NextHopPubkey {
		t.Fatal("nextHopPubkey")
	}
	if string(rv.Field(8).Interface().([]byte)) != want.Signature {
		t.Fatal("signature")
	}
}

func hashFromABI(v any) common.Hash {
	switch x := v.(type) {
	case common.Hash:
		return x
	case [32]byte:
		return common.Hash(x)
	default:
		return common.Hash{}
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

	ev := &GrievanceEvent{
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
	ev := &GrievanceEvent{
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
