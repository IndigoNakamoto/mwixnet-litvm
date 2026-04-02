package litvm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// ReceiptForDefense is hop receipt data loaded for a grievance defense path.
type ReceiptForDefense struct {
	EpochID               *big.Int
	Accuser               common.Address
	AccusedMaker          common.Address
	HopIndex              uint8
	PeeledCommitment      common.Hash
	ForwardCiphertextHash common.Hash
	NextHopPubkey         string
	Signature             string
}

// ReceiptLookup is implemented by the evidence store; nil means log-only watcher behavior.
type ReceiptLookup interface {
	GetByEvidenceHash(hash common.Hash) (*ReceiptForDefense, error)
}
