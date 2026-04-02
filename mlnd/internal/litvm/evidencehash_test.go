package litvm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Golden vectors from contracts/test/EvidenceGoldenVectors.t.sol
var (
	goldenEvidenceHash = common.HexToHash("0x2d4d7ae96f39e2d5037f21782bc831874261ffe22743f74bbf865a39ec4df112")
	goldenGrievanceID  = common.HexToHash("0x5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3")
)

func TestComputeEvidenceHash_goldenVectors(t *testing.T) {
	epochID := big.NewInt(42)
	accuser := common.HexToAddress("0x0000000000000000000000000000beef")
	accused := common.HexToAddress("0x0000000000000000000000000000cafe")
	peeled := common.BytesToHash(common.LeftPadBytes(big.NewInt(0x1111).Bytes(), 32))
	forward := common.BytesToHash(common.LeftPadBytes(big.NewInt(0x2222).Bytes(), 32))

	p := EvidencePreimage{
		EpochID:               epochID,
		Accuser:               accuser,
		AccusedMaker:          accused,
		HopIndex:              2,
		PeeledCommitment:      peeled,
		ForwardCiphertextHash: forward,
	}

	got := ComputeEvidenceHash(p)
	if got != goldenEvidenceHash {
		t.Fatalf("evidenceHash: got %s want %s", got.Hex(), goldenEvidenceHash.Hex())
	}

	// Preimage length sanity (Solidity test asserts 137)
	var buf []byte
	buf = append(buf, common.LeftPadBytes(epochID.Bytes(), 32)...)
	buf = append(buf, accuser.Bytes()...)
	buf = append(buf, accused.Bytes()...)
	buf = append(buf, 2)
	buf = append(buf, peeled.Bytes()...)
	buf = append(buf, forward.Bytes()...)
	if len(buf) != 137 {
		t.Fatalf("packed preimage length: got %d want 137", len(buf))
	}

	gid := ComputeGrievanceID(accuser, accused, epochID, got)
	if gid != goldenGrievanceID {
		t.Fatalf("grievanceId: got %s want %s", gid.Hex(), goldenGrievanceID.Hex())
	}
}

func TestComputeEvidenceHash_largeEpochID(t *testing.T) {
	// Full uint256-ish epoch: 2^255 + 1 (fits in big.Int, must left-pad to 32 bytes)
	epochID, _ := new(big.Int).SetString("57896044618658097711785492504343953926634992332820282019728792003956564819969", 10)
	p := EvidencePreimage{
		EpochID:               epochID,
		Accuser:               common.HexToAddress("0x1"),
		AccusedMaker:          common.HexToAddress("0x2"),
		HopIndex:              0,
		PeeledCommitment:      common.Hash{},
		ForwardCiphertextHash: common.Hash{},
	}
	h := ComputeEvidenceHash(p)
	if h == (common.Hash{}) {
		t.Fatal("expected non-zero hash")
	}
	// Recompute with same epoch should match
	if ComputeEvidenceHash(p) != h {
		t.Fatal("deterministic hash expected")
	}
}
