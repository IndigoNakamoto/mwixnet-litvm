package litvmevidence

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// GrievanceOpened is decoded GrievanceOpened log data (four topics + data fields).
type GrievanceOpened struct {
	GrievanceID  common.Hash
	Accuser      common.Address
	Accused      common.Address
	EpochID      *big.Int
	EvidenceHash common.Hash
	Deadline     *big.Int
}
