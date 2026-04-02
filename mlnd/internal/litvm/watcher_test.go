package litvm

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestGrievanceOpenedEventSigGolden(t *testing.T) {
	// Golden: cast keccak "GrievanceOpened(bytes32,address,address,uint256,bytes32,uint256)"
	want := common.HexToHash("0xa9ad9c45148151beaae5b9b1dddf35e5c196382578e67feded31a6ea25a0e010")
	if GrievanceOpenedEventSig != want {
		t.Fatalf("event topic0: got %s want %s", GrievanceOpenedEventSig.Hex(), want.Hex())
	}
}

func TestParseGrievanceLog_roundTrip(t *testing.T) {
	grievanceID := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	accuser := common.HexToAddress("0x0000000000000000000000000000000000000001")
	accused := common.HexToAddress("0x000000000000000000000000000000000000cafe")
	epochID := big.NewInt(42)
	evidenceHash := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	deadline := big.NewInt(1700000000)

	data := make([]byte, 0, 96)
	data = append(data, common.LeftPadBytes(epochID.Bytes(), 32)...)
	data = append(data, evidenceHash.Bytes()...)
	data = append(data, common.LeftPadBytes(deadline.Bytes(), 32)...)

	vLog := types.Log{
		Topics: []common.Hash{
			GrievanceOpenedEventSig,
			grievanceID,
			common.BytesToHash(common.LeftPadBytes(accuser.Bytes(), 32)),
			common.BytesToHash(common.LeftPadBytes(accused.Bytes(), 32)),
		},
		Data: data,
	}

	ev, err := ParseGrievanceLog(vLog)
	if err != nil {
		t.Fatal(err)
	}
	if ev.GrievanceID != grievanceID {
		t.Fatalf("grievanceId: got %s want %s", ev.GrievanceID.Hex(), grievanceID.Hex())
	}
	if ev.Accuser != accuser {
		t.Fatalf("accuser: got %s want %s", ev.Accuser.Hex(), accuser.Hex())
	}
	if ev.Accused != accused {
		t.Fatalf("accused: got %s want %s", ev.Accused.Hex(), accused.Hex())
	}
	if ev.EpochID.Cmp(epochID) != 0 {
		t.Fatalf("epochId: got %s want %s", ev.EpochID.String(), epochID.String())
	}
	if ev.EvidenceHash != evidenceHash {
		t.Fatalf("evidenceHash: got %s want %s", ev.EvidenceHash.Hex(), evidenceHash.Hex())
	}
	if ev.Deadline.Cmp(deadline) != 0 {
		t.Fatalf("deadline: got %s want %s", ev.Deadline.String(), deadline.String())
	}
}

func TestParseGrievanceLog_wrongTopicCount(t *testing.T) {
	_, err := ParseGrievanceLog(types.Log{Topics: []common.Hash{{}, {}, {}}, Data: make([]byte, 96)})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseGrievanceLog_shortData(t *testing.T) {
	topics := []common.Hash{
		GrievanceOpenedEventSig,
		{},
		common.BytesToHash(common.LeftPadBytes(common.HexToAddress("0x1").Bytes(), 32)),
		common.BytesToHash(common.LeftPadBytes(common.HexToAddress("0x2").Bytes(), 32)),
	}
	_, err := ParseGrievanceLog(types.Log{Topics: topics, Data: make([]byte, 95)})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestParseGrievanceLog_hexFixture decodes a log built from explicit hex (relay/export style).
func TestParseGrievanceLog_hexFixture(t *testing.T) {
	// topics: sig + grievanceId + accuser + accused (32-byte words)
	topic1 := "2222222222222222222222222222222222222222222222222222222222222222"
	accuser := common.HexToAddress("0x3e8")
	accused := common.HexToAddress("0x7b")
	topics := []common.Hash{
		GrievanceOpenedEventSig,
		mustHexHash(t, topic1),
		common.BytesToHash(common.LeftPadBytes(accuser.Bytes(), 32)),
		common.BytesToHash(common.LeftPadBytes(accused.Bytes(), 32)),
	}
	// data: epoch=7, evidence=0xbb..bb, deadline=9
	epoch := common.LeftPadBytes(big.NewInt(7).Bytes(), 32)
	evid := common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Bytes()
	deadline := common.LeftPadBytes(big.NewInt(9).Bytes(), 32)
	data := append(append(append([]byte{}, epoch...), evid...), deadline...)

	ev, err := ParseGrievanceLog(types.Log{Topics: topics, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if ev.EpochID.Cmp(big.NewInt(7)) != 0 || ev.Deadline.Cmp(big.NewInt(9)) != 0 {
		t.Fatalf("epoch/deadline: %+v", ev)
	}
	if ev.Accuser != common.HexToAddress("0x3e8") || ev.Accused != common.HexToAddress("0x7b") {
		t.Fatalf("addresses: accuser=%s accused=%s", ev.Accuser.Hex(), ev.Accused.Hex())
	}
}

func mustHexHash(t *testing.T, s string) common.Hash {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	var h common.Hash
	copy(h[:], b)
	return h
}
