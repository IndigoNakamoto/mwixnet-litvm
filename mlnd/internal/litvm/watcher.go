package litvm

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/opslog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// GrievanceOpenedEventSig is keccak256("GrievanceOpened(bytes32,address,address,uint256,bytes32,uint256)").
var GrievanceOpenedEventSig = crypto.Keccak256Hash([]byte("GrievanceOpened(bytes32,address,address,uint256,bytes32,uint256)"))

// GrievanceEvent is a decoded GrievanceOpened log.
type GrievanceEvent struct {
	GrievanceID  common.Hash
	Accuser      common.Address
	Accused      common.Address
	EpochID      *big.Int
	EvidenceHash common.Hash
	Deadline     *big.Int
}

// Watcher subscribes to GrievanceOpened logs for a fixed accused (operator) address.
type Watcher struct {
	client       *ethclient.Client
	courtAddr    common.Address
	operatorAddr common.Address
	receipts     ReceiptLookup
	defender     *Defender
	ops          *opslog.Log
}

// NewWatcher returns a watcher using an existing JSON-RPC client (caller typically dials once for ChainID / defend).
// The caller owns client lifecycle (close on shutdown); ops may be nil.
func NewWatcher(client *ethclient.Client, courtHex, operatorHex string, receipts ReceiptLookup, defender *Defender, ops *opslog.Log) (*Watcher, error) {
	if client == nil {
		return nil, fmt.Errorf("nil eth client")
	}
	return &Watcher{
		client:       client,
		courtAddr:    common.HexToAddress(courtHex),
		operatorAddr: common.HexToAddress(operatorHex),
		receipts:     receipts,
		defender:     defender,
		ops:          ops,
	}, nil
}

// Start blocks until ctx is cancelled or the subscription errors. It logs each matching grievance.
func (w *Watcher) Start(ctx context.Context) error {
	operatorTopic := common.BytesToHash(common.LeftPadBytes(w.operatorAddr.Bytes(), 32))
	query := ethereum.FilterQuery{
		Addresses: []common.Address{w.courtAddr},
		Topics: [][]common.Hash{
			{GrievanceOpenedEventSig},
			nil,
			nil,
			{operatorTopic},
		},
	}

	logsCh := make(chan types.Log)
	sub, err := w.client.SubscribeFilterLogs(ctx, query, logsCh)
	if err != nil {
		return fmt.Errorf("subscribe logs: %w", err)
	}
	defer sub.Unsubscribe()

	log.Printf("mlnd: watching GrievanceOpened accused=%s court=%s", w.operatorAddr.Hex(), w.courtAddr.Hex())

	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-sub.Err():
			if !ok {
				return nil
			}
			if err != nil {
				return fmt.Errorf("subscription: %w", err)
			}
		case vLog := <-logsCh:
			ev, err := ParseGrievanceLog(vLog)
			if err != nil {
				log.Printf("mlnd: skip log: %v", err)
				continue
			}
			log.Printf("mlnd: GrievanceOpened grievanceId=%s accuser=%s accused=%s epochId=%s evidenceHash=%s deadline=%s",
				ev.GrievanceID.Hex(), ev.Accuser.Hex(), ev.Accused.Hex(), ev.EpochID.String(), ev.EvidenceHash.Hex(), ev.Deadline.String())
			w.opsAppend(opslog.Info, "grievance_opened", "Grievance opened against this operator (LitVM)", map[string]string{
				"grievanceId":  ev.GrievanceID.Hex(),
				"evidenceHash": ev.EvidenceHash.Hex(),
				"deadline":     ev.Deadline.String(),
				"epochId":      ev.EpochID.String(),
			})

			if w.receipts != nil {
				w.opsAppend(opslog.Info, "receipt_lookup", "Looking up hop receipt in SQLite vault", map[string]string{
					"evidenceHash": ev.EvidenceHash.Hex(),
				})
				receipt, err := w.receipts.GetByEvidenceHash(ev.EvidenceHash)
				if err != nil {
					log.Printf("mlnd: [CRITICAL] cannot defend grievance %s: %v", ev.GrievanceID.Hex(), err)
					w.opsAppend(opslog.Critical, "receipt_missing", "No receipt in vault for evidenceHash — cannot auto-defend or build defenseData", map[string]string{
						"grievanceId":  ev.GrievanceID.Hex(),
						"evidenceHash": ev.EvidenceHash.Hex(),
					})
					continue
				}
				w.handleReceiptFound(ctx, ev, receipt)
			}
		}
	}
}

func (w *Watcher) handleReceiptFound(ctx context.Context, ev *GrievanceEvent, receipt *ReceiptForDefense) {
	if err := ValidateReceiptForGrievance(ev, receipt, w.operatorAddr); err != nil {
		log.Printf("mlnd: receipt validation failed for grievance %s: %v", ev.GrievanceID.Hex(), err)
		w.opsAppend(opslog.Warn, "receipt_validation_failed", "Receipt in vault does not match grievance correlators", map[string]string{
			"grievanceId": ev.GrievanceID.Hex(),
			"detail":      err.Error(),
		})
		return
	}
	w.opsAppend(opslog.Info, "receipt_validated", "Receipt matches grievance; computing evidenceHash / defense payload", map[string]string{
		"grievanceId":  ev.GrievanceID.Hex(),
		"evidenceHash": ev.EvidenceHash.Hex(),
	})
	if w.defender == nil {
		log.Printf("mlnd: validated receipt for grievance %s (auto-defend off; set MLND_DEFEND_AUTO=1 and MLND_OPERATOR_PRIVATE_KEY)", ev.GrievanceID.Hex())
		w.opsAppend(opslog.Critical, "manual_defense_required", "Auto-defend is OFF — submit defendGrievance manually before the deadline", map[string]string{
			"grievanceId":  ev.GrievanceID.Hex(),
			"evidenceHash": ev.EvidenceHash.Hex(),
			"deadline":     ev.Deadline.String(),
		})
		return
	}

	if w.client == nil {
		return
	}
	head, err := w.client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Printf("mlnd: chain header for deadline check: %v", err)
		w.opsAppend(opslog.Error, "deadline_check_failed", "Could not load chain head for defense deadline check", map[string]string{
			"detail": err.Error(),
		})
		return
	}
	if !ChainTimeBeforeDeadline(head, ev.Deadline) {
		log.Printf("mlnd: skip defend %s: chain time %d >= deadline %s", ev.GrievanceID.Hex(), head.Time, ev.Deadline.String())
		w.opsAppend(opslog.Warn, "defend_skipped_deadline", "Chain time is past grievance defense deadline", map[string]string{
			"grievanceId": ev.GrievanceID.Hex(),
		})
		return
	}

	defenseData, err := BuildDefenseData(receipt)
	if err != nil {
		log.Printf("mlnd: build defenseData for %s: %v", ev.GrievanceID.Hex(), err)
		w.opsAppend(opslog.Error, "defense_build_failed", "Failed to ABI-encode defenseData", map[string]string{
			"grievanceId": ev.GrievanceID.Hex(),
			"detail":      err.Error(),
		})
		return
	}

	if w.defender.IsDryRun() {
		log.Printf("mlnd: DRY-RUN defendGrievance grievanceId=%s defenseData (%d bytes)=%x", ev.GrievanceID.Hex(), len(defenseData), defenseData)
		w.opsAppend(opslog.Info, "defend_dry_run", "Would submit defendGrievance (MLND_DEFEND_DRY_RUN)", map[string]string{
			"grievanceId":  ev.GrievanceID.Hex(),
			"defenseBytes": fmt.Sprintf("%d", len(defenseData)),
		})
		return
	}

	tx, err := w.defender.SubmitDefend(ctx, w.client, ev.GrievanceID, defenseData)
	if err != nil {
		log.Printf("mlnd: defendGrievance failed for %s: %v", ev.GrievanceID.Hex(), err)
		w.opsAppend(opslog.Error, "defend_submit_failed", "defendGrievance transaction failed", map[string]string{
			"grievanceId": ev.GrievanceID.Hex(),
			"detail":      err.Error(),
		})
		return
	}
	log.Printf("mlnd: submitted defendGrievance tx=%s grievanceId=%s", tx.Hash().Hex(), ev.GrievanceID.Hex())
	w.opsAppend(opslog.Info, "defend_submitted", "defendGrievance transaction submitted — awaiting resolution", map[string]string{
		"grievanceId": ev.GrievanceID.Hex(),
		"txHash":      tx.Hash().Hex(),
	})
}

func (w *Watcher) opsAppend(level opslog.Level, code, msg string, data map[string]string) {
	if w == nil || w.ops == nil {
		return
	}
	w.ops.Append(level, code, msg, data)
}

// ParseGrievanceLog decodes a GrievanceOpened log (four topics, 96 bytes data).
func ParseGrievanceLog(vLog types.Log) (*GrievanceEvent, error) {
	if len(vLog.Topics) != 4 {
		return nil, fmt.Errorf("expected 4 topics, got %d", len(vLog.Topics))
	}
	if len(vLog.Data) < 96 {
		return nil, fmt.Errorf("expected at least 96 bytes data, got %d", len(vLog.Data))
	}

	return &GrievanceEvent{
		GrievanceID:  vLog.Topics[1],
		Accuser:      common.BytesToAddress(vLog.Topics[2].Bytes()),
		Accused:      common.BytesToAddress(vLog.Topics[3].Bytes()),
		EpochID:      new(big.Int).SetBytes(vLog.Data[0:32]),
		EvidenceHash: common.BytesToHash(vLog.Data[32:64]),
		Deadline:     new(big.Int).SetBytes(vLog.Data[64:96]),
	}, nil
}
