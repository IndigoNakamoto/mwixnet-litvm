package litvm

import (
	"context"
	"math/big"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/litvmevidence"
	"github.com/ethereum/go-ethereum/common"
)

func TestHandleReceiptFound_nilDefender_skipsBeforeRPC(t *testing.T) {
	epoch := big.NewInt(7)
	accuser := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	accused := common.HexToAddress("0x00000000000000000000000000000000000000bb")
	peeled := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	fwd := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	pre := litvmevidence.EvidencePreimage{EpochID: epoch, Accuser: accuser, AccusedMaker: accused, HopIndex: 0, PeeledCommitment: peeled, ForwardCiphertextHash: fwd}
	evHash := litvmevidence.ComputeEvidenceHash(pre)
	gid := litvmevidence.ComputeGrievanceID(accuser, accused, epoch, evHash)
	ev := &litvmevidence.GrievanceOpened{
		GrievanceID:  gid,
		Accuser:      accuser,
		Accused:      accused,
		EpochID:      new(big.Int).Set(epoch),
		EvidenceHash: evHash,
		Deadline:     big.NewInt(9999999999),
	}
	r := &litvmevidence.ReceiptForDefense{
		EpochID:               new(big.Int).Set(epoch),
		Accuser:               accuser,
		AccusedMaker:          accused,
		HopIndex:              0,
		PeeledCommitment:      peeled,
		ForwardCiphertextHash: fwd,
		NextHopPubkey:         "k",
		Signature:             "s",
	}
	w := &Watcher{
		client:       nil,
		operatorAddr: accused,
		defender:     nil,
	}
	w.handleReceiptFound(context.Background(), ev, r)
}

func TestHandleReceiptFound_validationError(t *testing.T) {
	accused := common.HexToAddress("0x00000000000000000000000000000000000000bb")
	ev := &litvmevidence.GrievanceOpened{
		GrievanceID:  common.Hash{1},
		Accuser:      common.HexToAddress("0xaa"),
		Accused:      accused,
		EpochID:      big.NewInt(1),
		EvidenceHash: common.Hash{0xff},
		Deadline:     big.NewInt(9),
	}
	r := &litvmevidence.ReceiptForDefense{
		EpochID:               big.NewInt(1),
		Accuser:               common.HexToAddress("0xaa"),
		AccusedMaker:          accused,
		HopIndex:              0,
		PeeledCommitment:      common.Hash{},
		ForwardCiphertextHash: common.Hash{},
	}
	w := &Watcher{operatorAddr: accused, defender: nil}
	w.handleReceiptFound(context.Background(), ev, r)
}
