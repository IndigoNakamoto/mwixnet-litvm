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
}

// NewWatcher dials wsURL (WebSocket JSON-RPC) and returns a watcher for courtAddr / operatorAddr.
func NewWatcher(wsURL, courtHex, operatorHex string) (*Watcher, error) {
	client, err := ethclient.Dial(wsURL)
	if err != nil {
		return nil, fmt.Errorf("dial %q: %w", wsURL, err)
	}
	return &Watcher{
		client:       client,
		courtAddr:    common.HexToAddress(courtHex),
		operatorAddr: common.HexToAddress(operatorHex),
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
		}
	}
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
