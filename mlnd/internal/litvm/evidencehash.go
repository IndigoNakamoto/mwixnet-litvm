package litvm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EvidencePreimage is the preimage for appendix 13 evidenceHash (Solidity abi.encodePacked then keccak256).
type EvidencePreimage struct {
	EpochID               *big.Int
	Accuser               common.Address
	AccusedMaker          common.Address
	HopIndex              uint8
	PeeledCommitment      common.Hash
	ForwardCiphertextHash common.Hash
}

// ComputeEvidenceHash matches EvidenceLib.evidenceHash (137-byte packed preimage).
func ComputeEvidenceHash(p EvidencePreimage) common.Hash {
	epoch := p.EpochID
	if epoch == nil {
		epoch = big.NewInt(0)
	}
	packed := make([]byte, 0, 137)
	packed = append(packed, common.LeftPadBytes(epoch.Bytes(), 32)...)
	packed = append(packed, p.Accuser.Bytes()...)
	packed = append(packed, p.AccusedMaker.Bytes()...)
	packed = append(packed, p.HopIndex)
	packed = append(packed, p.PeeledCommitment.Bytes()...)
	packed = append(packed, p.ForwardCiphertextHash.Bytes()...)
	return crypto.Keccak256Hash(packed)
}

// ComputeGrievanceID matches EvidenceLib.grievanceId (encodePacked accuser, accused, epochId, evidenceHash).
func ComputeGrievanceID(accuser, accused common.Address, epochID *big.Int, evidenceHash common.Hash) common.Hash {
	epoch := epochID
	if epoch == nil {
		epoch = big.NewInt(0)
	}
	packed := make([]byte, 0, 104)
	packed = append(packed, accuser.Bytes()...)
	packed = append(packed, accused.Bytes()...)
	packed = append(packed, common.LeftPadBytes(epoch.Bytes(), 32)...)
	packed = append(packed, evidenceHash.Bytes()...)
	return crypto.Keccak256Hash(packed)
}
