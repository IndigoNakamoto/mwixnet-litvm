package litvm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// defenseTupleType is one Solidity tuple for abi.encode (single off-chain decode).
var defenseTupleType, _ = abi.NewType("tuple", "", []abi.ArgumentMarshaling{
	{Name: "version", Type: "uint8"},
	{Name: "epochId", Type: "uint256"},
	{Name: "accuser", Type: "address"},
	{Name: "accusedMaker", Type: "address"},
	{Name: "hopIndex", Type: "uint8"},
	{Name: "peeledCommitment", Type: "bytes32"},
	{Name: "forwardCiphertextHash", Type: "bytes32"},
	{Name: "nextHopPubkeyUTF8", Type: "bytes"},
	{Name: "signatureUTF8", Type: "bytes"},
})

var defensePackArgs = abi.Arguments{{Type: defenseTupleType}}

// ValidateReceiptForGrievance checks the stored receipt matches the opened grievance correlators and hashes.
func ValidateReceiptForGrievance(ev *GrievanceEvent, r *ReceiptForDefense, operator common.Address) error {
	if ev == nil || r == nil {
		return fmt.Errorf("nil event or receipt")
	}
	if r.AccusedMaker != operator || ev.Accused != operator {
		return fmt.Errorf("accused/operator mismatch (ev.accused=%s receipt.accusedMaker=%s want operator=%s)",
			ev.Accused.Hex(), r.AccusedMaker.Hex(), operator.Hex())
	}
	if ev.Accuser != r.Accuser {
		return fmt.Errorf("accuser mismatch: log %s receipt %s", ev.Accuser.Hex(), r.Accuser.Hex())
	}
	if ev.EpochID == nil || r.EpochID == nil {
		return fmt.Errorf("epochId required")
	}
	if ev.EpochID.Cmp(r.EpochID) != 0 {
		return fmt.Errorf("epochId mismatch: log %s receipt %s", ev.EpochID.String(), r.EpochID.String())
	}
	pre := EvidencePreimage{
		EpochID:               r.EpochID,
		Accuser:               r.Accuser,
		AccusedMaker:          r.AccusedMaker,
		HopIndex:              r.HopIndex,
		PeeledCommitment:      r.PeeledCommitment,
		ForwardCiphertextHash: r.ForwardCiphertextHash,
	}
	got := ComputeEvidenceHash(pre)
	if got != ev.EvidenceHash {
		return fmt.Errorf("evidenceHash mismatch: computed %s log %s", got.Hex(), ev.EvidenceHash.Hex())
	}
	gid := ComputeGrievanceID(r.Accuser, r.AccusedMaker, r.EpochID, got)
	if gid != ev.GrievanceID {
		return fmt.Errorf("grievanceId mismatch: computed %s log %s", gid.Hex(), ev.GrievanceID.Hex())
	}
	return nil
}

// BuildDefenseData returns abi.encode(tuple(...)) for defenseData v1 (opaque on-chain).
func BuildDefenseData(r *ReceiptForDefense) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("nil receipt")
	}
	if r.EpochID == nil {
		return nil, fmt.Errorf("EpochID required")
	}
	tuple := struct {
		Version               uint8
		EpochID               *big.Int `abi:"epochId"`
		Accuser               common.Address
		AccusedMaker          common.Address
		HopIndex              uint8
		PeeledCommitment      common.Hash
		ForwardCiphertextHash common.Hash
		NextHopPubkeyUTF8     []byte
		SignatureUTF8         []byte
	}{
		Version:               1,
		EpochID:               new(big.Int).Set(r.EpochID),
		Accuser:               r.Accuser,
		AccusedMaker:          r.AccusedMaker,
		HopIndex:              r.HopIndex,
		PeeledCommitment:      r.PeeledCommitment,
		ForwardCiphertextHash: r.ForwardCiphertextHash,
		NextHopPubkeyUTF8:     []byte(r.NextHopPubkey),
		SignatureUTF8:         []byte(r.Signature),
	}
	return defensePackArgs.Pack(tuple)
}

// ChainTimeBeforeDeadline compares latest chain header time to grievance deadline (Unix seconds).
// Use HeaderByNumber(ctx, nil) for chain time — not the local wall clock.
func ChainTimeBeforeDeadline(header *types.Header, deadline *big.Int) bool {
	if header == nil || deadline == nil {
		return false
	}
	chainTs := new(big.Int).SetUint64(header.Time)
	return chainTs.Cmp(deadline) < 0
}
