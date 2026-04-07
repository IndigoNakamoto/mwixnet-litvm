package litvm

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/opslog"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/litvmevidence"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// AccuserWatcher subscribes to GrievanceOpened where accuser is the local victim address.
type AccuserWatcher struct {
	client      *ethclient.Client
	courtAddr   common.Address
	accuserAddr common.Address
	resolver    *AccuserResolver
	ops         *opslog.Log
}

// NewAccuserWatcher constructs a watcher; resolver must be non-nil for auto-resolve.
func NewAccuserWatcher(client *ethclient.Client, courtHex string, accuser common.Address, resolver *AccuserResolver, ops *opslog.Log) (*AccuserWatcher, error) {
	if client == nil {
		return nil, fmt.Errorf("nil eth client")
	}
	if resolver == nil {
		return nil, fmt.Errorf("nil accuser resolver")
	}
	return &AccuserWatcher{
		client:      client,
		courtAddr:   common.HexToAddress(courtHex),
		accuserAddr: accuser,
		resolver:    resolver,
		ops:         ops,
	}, nil
}

// Start blocks until ctx cancelled. For each matching grievance, waits for deadline then calls resolveGrievance if still Open.
func (w *AccuserWatcher) Start(ctx context.Context) error {
	accuserTopic := common.BytesToHash(common.LeftPadBytes(w.accuserAddr.Bytes(), 32))
	query := ethereum.FilterQuery{
		Addresses: []common.Address{w.courtAddr},
		Topics: [][]common.Hash{
			{GrievanceOpenedEventSig},
			nil,
			{accuserTopic},
			nil,
		},
	}

	logsCh := make(chan types.Log)
	sub, err := w.client.SubscribeFilterLogs(ctx, query, logsCh)
	if err != nil {
		return fmt.Errorf("accuser subscribe logs: %w", err)
	}
	defer sub.Unsubscribe()

	log.Printf("mlnd: watching GrievanceOpened accuser=%s court=%s (auto-resolve)", w.accuserAddr.Hex(), w.courtAddr.Hex())

	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-sub.Err():
			if !ok {
				return nil
			}
			if err != nil {
				return fmt.Errorf("accuser subscription: %w", err)
			}
		case vLog := <-logsCh:
			ev, err := ParseGrievanceLog(vLog)
			if err != nil {
				log.Printf("mlnd accuser watcher: skip log: %v", err)
				continue
			}
			log.Printf("mlnd: GrievanceOpened (as accuser) grievanceId=%s accused=%s deadline=%s", ev.GrievanceID.Hex(), ev.Accused.Hex(), ev.Deadline.String())
			w.opsAppend(opslog.Info, "grievance_opened_accuser", "Grievance you opened — will attempt resolve after deadline if still Open", map[string]string{
				"grievanceId": ev.GrievanceID.Hex(),
				"accused":     ev.Accused.Hex(),
				"deadline":    ev.Deadline.String(),
			})
			go w.waitAndResolve(ctx, ev)
		}
	}
}

func (w *AccuserWatcher) waitAndResolve(ctx context.Context, ev *litvmevidence.GrievanceOpened) {
	const poll = 12 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		head, err := w.client.HeaderByNumber(ctx, nil)
		if err != nil {
			log.Printf("mlnd accuser resolve: chain head: %v", err)
			time.Sleep(poll)
			continue
		}
		chainTs := new(big.Int).SetUint64(head.Time)
		if chainTs.Cmp(ev.Deadline) < 0 {
			time.Sleep(poll)
			continue
		}

		if w.resolver.IsDryRun() {
			log.Printf("mlnd: DRY-RUN resolveGrievance grievanceId=%s (MLND_ACCUSER_RESOLVE_DRY_RUN)", ev.GrievanceID.Hex())
			w.opsAppend(opslog.Info, "resolve_dry_run", "Would submit resolveGrievance (dry run)", map[string]string{
				"grievanceId": ev.GrievanceID.Hex(),
			})
			return
		}

		tx, err := w.submitResolveLoop(ctx, ev.GrievanceID, ev.Deadline)
		if err != nil {
			log.Printf("mlnd: resolveGrievance failed for %s: %v", ev.GrievanceID.Hex(), err)
			w.opsAppend(opslog.Error, "resolve_submit_failed", "resolveGrievance failed", map[string]string{
				"grievanceId": ev.GrievanceID.Hex(),
				"detail":      err.Error(),
			})
			return
		}
		if tx != nil {
			log.Printf("mlnd: submitted resolveGrievance tx=%s grievanceId=%s", tx.Hash().Hex(), ev.GrievanceID.Hex())
			w.opsAppend(opslog.Info, "resolve_submitted", "resolveGrievance submitted", map[string]string{
				"grievanceId": ev.GrievanceID.Hex(),
				"txHash":      tx.Hash().Hex(),
			})
		}
		return
	}
}

func (w *AccuserWatcher) submitResolveLoop(ctx context.Context, grievanceID common.Hash, deadline *big.Int) (*types.Transaction, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(200*attempt) * time.Millisecond)
		}
		tx, err := w.resolver.SubmitResolve(ctx, w.client, grievanceID, deadline)
		if err == nil {
			return tx, nil
		}
		lastErr = err
		if !isRetryableSubmitErr(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("after retries: %w", lastErr)
}

func (w *AccuserWatcher) opsAppend(level opslog.Level, code, msg string, data map[string]string) {
	if w == nil || w.ops == nil {
		return
	}
	w.ops.Append(level, code, msg, data)
}
