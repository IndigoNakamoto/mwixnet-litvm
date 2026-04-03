package litvm

import (
	"context"
	"fmt"
	"log"
	"math/big"

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
}

// NewWatcher returns a watcher using an existing JSON-RPC client (caller typically dials once for ChainID / defend).
func NewWatcher(client *ethclient.Client, courtHex, operatorHex string, receipts ReceiptLookup, defender *Defender) (*Watcher, error) {
	if client == nil {
		return nil, fmt.Errorf("nil eth client")
	}
	return &Watcher{
		client:       client,
		courtAddr:    common.HexToAddress(courtHex),
		operatorAddr: common.HexToAddress(operatorHex),
		receipts:     receipts,
		defender:     defender,
	}, nil
}

// Start blocks until ctx is cancelled or the subscription errors. It logs each matching grievance.
func (w *Watcher) Start(ctx context.Context) error {
	defer w.client.Close()

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

			if w.receipts != nil {
				receipt, err := w.receipts.GetByEvidenceHash(ev.EvidenceHash)
				if err != nil {
					log.Printf("mlnd: [CRITICAL] cannot defend grievance %s: %v", ev.GrievanceID.Hex(), err)
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
		return
	}
	if w.defender == nil {
		log.Printf("mlnd: validated receipt for grievance %s (auto-defend off; set MLND_DEFEND_AUTO=1 and MLND_OPERATOR_PRIVATE_KEY)", ev.GrievanceID.Hex())
		return
	}

	head, err := w.client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Printf("mlnd: chain header for deadline check: %v", err)
		return
	}
	if !ChainTimeBeforeDeadline(head, ev.Deadline) {
		log.Printf("mlnd: skip defend %s: chain time %d >= deadline %s", ev.GrievanceID.Hex(), head.Time, ev.Deadline.String())
		return
	}

	defenseData, err := BuildDefenseData(receipt)
	if err != nil {
		log.Printf("mlnd: build defenseData for %s: %v", ev.GrievanceID.Hex(), err)
		return
	}

	if w.defender.IsDryRun() {
		log.Printf("mlnd: DRY-RUN defendGrievance grievanceId=%s defenseData (%d bytes)=%x", ev.GrievanceID.Hex(), len(defenseData), defenseData)
		return
	}

	tx, err := w.defender.SubmitDefend(ctx, w.client, ev.GrievanceID, defenseData)
	if err != nil {
		log.Printf("mlnd: defendGrievance failed for %s: %v", ev.GrievanceID.Hex(), err)
		return
	}
	log.Printf("mlnd: submitted defendGrievance tx=%s grievanceId=%s", tx.Hash().Hex(), ev.GrievanceID.Hex())
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
