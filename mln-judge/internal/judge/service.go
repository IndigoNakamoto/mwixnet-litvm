package judge

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/litvmevidence"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ContestedEventSig is topic0 for Contested(bytes32,address,bytes32).
var ContestedEventSig = crypto.Keccak256Hash([]byte("Contested(bytes32,address,bytes32)"))

// Service listens for Contested logs, replays defend calldata, and optionally calls adjudicateGrievance.
type Service struct {
	Client     *ethclient.Client
	Court      common.Address
	PrivateKey *ecdsa.PrivateKey
	DryRun     bool
	// AutoAdjudicate when true requires VerdictExonerate to be set.
	AutoAdjudicate   bool
	VerdictExonerate bool
}

// Run subscribes until ctx is cancelled.
func (s *Service) Run(ctx context.Context) error {
	judgeAddr := crypto.PubkeyToAddress(s.PrivateKey.PublicKey)
	log.Printf("mln-judge: judge address=%s court=%s dry_run=%v auto=%v exonerate=%v",
		judgeAddr.Hex(), s.Court.Hex(), s.DryRun, s.AutoAdjudicate, s.VerdictExonerate)

	query := ethereum.FilterQuery{
		Addresses: []common.Address{s.Court},
		Topics:    [][]common.Hash{{ContestedEventSig}},
	}
	ch := make(chan types.Log)
	sub, err := s.Client.SubscribeFilterLogs(ctx, query, ch)
	if err != nil {
		return fmt.Errorf("subscribe Contested: %w", err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			if err != nil {
				return err
			}
		case lg := <-ch:
			if err := s.handleContested(ctx, lg); err != nil {
				log.Printf("mln-judge: case %s: %v", lg.TxHash.Hex(), err)
			}
		}
	}
}

func (s *Service) handleContested(ctx context.Context, lg types.Log) error {
	if len(lg.Topics) != 3 || len(lg.Data) < 32 {
		return fmt.Errorf("unexpected Contested log shape")
	}
	grievanceID := lg.Topics[1]
	accused := common.BytesToAddress(lg.Topics[2].Bytes())
	var digest common.Hash
	copy(digest[:], lg.Data[:32])

	tx, _, err := s.Client.TransactionByHash(ctx, lg.TxHash)
	if err != nil {
		return fmt.Errorf("tx %s: %w", lg.TxHash.Hex(), err)
	}
	if tx.To() == nil || *tx.To() != s.Court {
		return fmt.Errorf("tx not to court")
	}
	payload := tx.Data()
	if len(payload) < 4 {
		return fmt.Errorf("short calldata")
	}
	method, err := parsedJudgeABI.MethodById(payload[:4])
	if err != nil || method == nil || method.Name != "defendGrievance" {
		return fmt.Errorf("tx calldata is not defendGrievance")
	}
	vals, err := method.Inputs.Unpack(payload[4:])
	if err != nil {
		return fmt.Errorf("unpack defend: %w", err)
	}
	if len(vals) != 2 {
		return fmt.Errorf("defend args len %d", len(vals))
	}
	var gotID common.Hash
	switch x := vals[0].(type) {
	case [32]byte:
		gotID = common.Hash(x)
	case common.Hash:
		gotID = x
	default:
		return fmt.Errorf("grievanceId arg type %T", vals[0])
	}
	if gotID != grievanceID {
		return fmt.Errorf("grievanceId mismatch log vs calldata")
	}
	defenseData, ok := vals[1].([]byte)
	if !ok {
		return fmt.Errorf("defenseData type")
	}
	sum := crypto.Keccak256Hash(defenseData)
	if sum != digest {
		return fmt.Errorf("defenseData digest mismatch (log %s computed %s)", digest.Hex(), sum.Hex())
	}

	receipt, err := litvmevidence.UnpackDefenseV1(defenseData)
	if err != nil {
		return fmt.Errorf("unpack defense v1: %w", err)
	}

	log.Printf("mln-judge: Contested grievanceId=%s accused=%s hop=%d nextHopPubkey_len=%d signature_len=%d",
		grievanceID.Hex(), accused.Hex(), receipt.HopIndex, len(receipt.NextHopPubkey), len(receipt.Signature))
	// v1: receipt signature / pubkey formats are not normatively verified on-chain; off-chain policy TBD (PRODUCT_SPEC §13.6).
	log.Printf("mln-judge: manual verify next-hop acknowledgment; automated signature verification not enabled in v1 stub")

	exonerate := s.VerdictExonerate
	if !s.AutoAdjudicate || s.DryRun {
		log.Printf("mln-judge: cast hint (adjudicate): cast send %s \"adjudicateGrievance(bytes32,bool)\" %s %t --rpc-url \"$RPC\" --private-key \"$JUDGE_KEY\"",
			s.Court.Hex(), grievanceID.Hex(), exonerate)
		return nil
	}

	chainID, err := s.Client.ChainID(ctx)
	if err != nil {
		return err
	}
	opts, err := bind.NewKeyedTransactorWithChainID(s.PrivateKey, chainID)
	if err != nil {
		return err
	}
	opts.Context = ctx
	if gp, err := s.Client.SuggestGasPrice(ctx); err == nil {
		opts.GasPrice = gp
	}

	bound := newBound(s.Client, s.Court)
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(200*attempt) * time.Millisecond)
		}
		_, err = bound.Transact(opts, "adjudicateGrievance", grievanceID, exonerate)
		if err == nil {
			log.Printf("mln-judge: adjudicateGrievance submitted grievanceId=%s exonerate=%v", grievanceID.Hex(), exonerate)
			return nil
		}
		lastErr = err
		if !retryable(err) {
			return err
		}
	}
	return fmt.Errorf("adjudicate: %w", lastErr)
}

func retryable(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "execution reverted") || strings.Contains(s, "insufficient funds") {
		return false
	}
	return true
}

// PrivateKeyFromEnv loads JUDGE_PRIVATE_KEY (64 hex, optional 0x).
func PrivateKeyFromEnv() (*ecdsa.PrivateKey, error) {
	h := strings.TrimSpace(os.Getenv("JUDGE_PRIVATE_KEY"))
	h = strings.TrimPrefix(h, "0x")
	h = strings.TrimPrefix(h, "0X")
	if len(h) != 64 {
		return nil, fmt.Errorf("JUDGE_PRIVATE_KEY must be 64 hex chars")
	}
	return crypto.HexToECDSA(h)
}
