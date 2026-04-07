package grievance

import (
	"bytes"
	"context"
	"io"
	"math/big"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/litvmevidence"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type staticReceipt struct {
	r *litvmevidence.ReceiptForDefense
}

func (s staticReceipt) Load() (*litvmevidence.ReceiptForDefense, error) {
	return s.r, nil
}

func TestRunFile_dryRun_ok(t *testing.T) {
	key, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	epoch := big.NewInt(42)
	peeled := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	fwd := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	rec := &litvmevidence.ReceiptForDefense{
		EpochID:               epoch,
		Accuser:               addr,
		AccusedMaker:          common.HexToAddress("0x0000000000000000000000000000000000000b02"),
		HopIndex:              0,
		PeeledCommitment:      peeled,
		ForwardCiphertextHash: fwd,
		NextHopPubkey:         "npub",
		Signature:             "sig",
	}
	var buf bytes.Buffer
	err = RunFile(context.Background(), FileOpts{
		RPCURL:     "http://unused",
		Court:      common.HexToAddress("0x0000000000000000000000000000000000000001"),
		PrivateKey: key,
		BondWei:    big.NewInt(1e17),
		DryRun:     true,
		Out:        &buf,
	}, staticReceipt{r: rec})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !containsAll(out, "evidenceHash=", "grievanceId=", "dry-run") {
		t.Fatalf("output: %q", out)
	}
}

func TestRunFile_accuserMismatch(t *testing.T) {
	key, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		t.Fatal(err)
	}
	rec := &litvmevidence.ReceiptForDefense{
		EpochID:               big.NewInt(1),
		Accuser:               common.HexToAddress("0x0000000000000000000000000000000000000001"),
		AccusedMaker:          common.HexToAddress("0x2"),
		HopIndex:              0,
		PeeledCommitment:      common.Hash{},
		ForwardCiphertextHash: common.Hash{},
	}
	err = RunFile(context.Background(), FileOpts{
		PrivateKey: key,
		BondWei:    big.NewInt(1),
		DryRun:     true,
		Out:        io.Discard,
	}, staticReceipt{r: rec})
	if err == nil {
		t.Fatal("expected accuser mismatch error")
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !bytes.Contains([]byte(s), []byte(sub)) {
			return false
		}
	}
	return true
}
