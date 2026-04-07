package grievance

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/litvmevidence"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/receiptstore"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// FileOpts configures openGrievance submission.
type FileOpts struct {
	RPCURL      string
	Court       common.Address
	PrivateKey  *ecdsa.PrivateKey
	BondWei     *big.Int
	DryRun      bool
	Out         io.Writer
	SuggestFees bool
}

// ReceiptSource loads a ReceiptForDefense for filing.
type ReceiptSource interface {
	Load() (*litvmevidence.ReceiptForDefense, error)
}

// ReceiptJSONFile loads from a path (NDJSON / JSON line or file).
type ReceiptJSONFile struct {
	Path string
}

func (r ReceiptJSONFile) Load() (*litvmevidence.ReceiptForDefense, error) {
	raw, err := os.ReadFile(r.Path)
	if err != nil {
		return nil, err
	}
	rec, err := receiptstore.ParseReceiptNDJSON(raw)
	if err != nil {
		return nil, err
	}
	return receiptToDefense(&rec), nil
}

func receiptToDefense(r *receiptstore.ReceiptRecord) *litvmevidence.ReceiptForDefense {
	return &litvmevidence.ReceiptForDefense{
		EpochID:               r.EpochID,
		Accuser:               r.Accuser,
		AccusedMaker:          r.AccusedMaker,
		HopIndex:              r.HopIndex,
		PeeledCommitment:      r.PeeledCommitment,
		ForwardCiphertextHash: r.ForwardCiphertextHash,
		NextHopPubkey:         r.NextHopPubkey,
		Signature:             r.Signature,
	}
}

// VaultSwapLookup loads by swap_id from SQLite vault (MLN_RECEIPT_VAULT_PATH / mlnd hop_receipts schema).
type VaultSwapLookup struct {
	DBPath string
	SwapID string
}

func (v VaultSwapLookup) Load() (*litvmevidence.ReceiptForDefense, error) {
	st, err := receiptstore.NewStore(v.DBPath)
	if err != nil {
		return nil, err
	}
	defer st.Close()
	return st.GetBySwapID(v.SwapID)
}

// RunFile computes evidenceHash and broadcasts openGrievance(accused, epochId, evidenceHash) with bond.
func RunFile(ctx context.Context, opts FileOpts, src ReceiptSource) error {
	if opts.Out == nil {
		opts.Out = io.Discard
	}
	rec, err := src.Load()
	if err != nil {
		return err
	}
	if rec.EpochID == nil {
		return fmt.Errorf("receipt: epochId required")
	}
	addr := crypto.PubkeyToAddress(opts.PrivateKey.PublicKey)
	if addr != rec.Accuser {
		return fmt.Errorf("signing address differs from receipt accuser (want accuser=%s got key=%s)", rec.Accuser.Hex(), addr.Hex())
	}

	pre := litvmevidence.EvidencePreimage{
		EpochID:               rec.EpochID,
		Accuser:               rec.Accuser,
		AccusedMaker:          rec.AccusedMaker,
		HopIndex:              rec.HopIndex,
		PeeledCommitment:      rec.PeeledCommitment,
		ForwardCiphertextHash: rec.ForwardCiphertextHash,
	}
	evidenceHash := litvmevidence.ComputeEvidenceHash(pre)
	grievanceID := litvmevidence.ComputeGrievanceID(rec.Accuser, rec.AccusedMaker, rec.EpochID, evidenceHash)

	fmt.Fprintf(opts.Out, "evidenceHash=%s\n", evidenceHash.Hex())
	fmt.Fprintf(opts.Out, "grievanceId=%s\n", grievanceID.Hex())
	fmt.Fprintf(opts.Out, "accused=%s epochId=%s bondWei=%s\n", rec.AccusedMaker.Hex(), rec.EpochID.String(), opts.BondWei.String())

	if opts.DryRun {
		fmt.Fprintln(opts.Out, "dry-run: not broadcasting")
		return nil
	}

	client, err := ethclient.DialContext(ctx, opts.RPCURL)
	if err != nil {
		return fmt.Errorf("rpc: %w", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("chain id: %w", err)
	}
	transactOpts, err := bind.NewKeyedTransactorWithChainID(opts.PrivateKey, chainID)
	if err != nil {
		return err
	}
	transactOpts.Context = ctx
	transactOpts.Value = new(big.Int).Set(opts.BondWei)
	if opts.SuggestFees {
		if gp, err := client.SuggestGasPrice(ctx); err == nil {
			transactOpts.GasPrice = gp
		}
	}

	court := NewGrievanceCourtOpenBound(client, opts.Court)
	tx, err := court.Transact(transactOpts, "openGrievance", rec.AccusedMaker, rec.EpochID, evidenceHash)
	if err != nil {
		return fmt.Errorf("openGrievance: %w", err)
	}
	fmt.Fprintf(opts.Out, "txHash=%s\n", tx.Hash().Hex())

	ctxWait, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	rc, err := bind.WaitMined(ctxWait, client, tx)
	if err != nil {
		return fmt.Errorf("wait mined: %w", err)
	}
	if rc.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("transaction failed (status=%d)", rc.Status)
	}
	return nil
}

// AccuserKeyFromEnv returns MLN_ACCUSER_ETH_KEY or falls back to MLN_OPERATOR_ETH_KEY.
func AccuserKeyFromEnv() (*ecdsa.PrivateKey, error) {
	keyHex := strings.TrimSpace(os.Getenv("MLN_ACCUSER_ETH_KEY"))
	if keyHex == "" {
		keyHex = strings.TrimSpace(os.Getenv("MLN_OPERATOR_ETH_KEY"))
	}
	if keyHex == "" {
		return nil, fmt.Errorf("MLN_ACCUSER_ETH_KEY or MLN_OPERATOR_ETH_KEY is required")
	}
	return parseAccuserKey(keyHex)
}

func parseAccuserKey(hexKey string) (*ecdsa.PrivateKey, error) {
	h := strings.TrimSpace(hexKey)
	h = strings.TrimPrefix(h, "0x")
	h = strings.TrimPrefix(h, "0X")
	if len(h) != 64 {
		return nil, fmt.Errorf("expect 64 hex chars for private key, got %d", len(h))
	}
	return crypto.HexToECDSA(h)
}

// BondWeiFromEnv parses MLN_GRIEVANCE_BOND_WEI (decimal wei string); default 0.1 ether if unset.
func BondWeiFromEnv() *big.Int {
	s := strings.TrimSpace(os.Getenv("MLN_GRIEVANCE_BOND_WEI"))
	if s == "" {
		return big.NewInt(0).Mul(big.NewInt(1e17), big.NewInt(10)) // 1e18 / 10 = 0.1 ether
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok || v.Sign() <= 0 {
		return big.NewInt(0).Mul(big.NewInt(1e17), big.NewInt(10))
	}
	return v
}
