// Package bridge will connect coinswapd (JSON-RPC swap_* per research/COINSWAPD_TEARDOWN.md)
// to the SQLite receipt vault. v0 is a no-op Run loop for lifecycle wiring only.
package bridge

import (
	"context"
	"log"
)

// Coinswapd is a placeholder receipt bridge until the mix-completion RPC or log format is fixed.
type Coinswapd struct {
	log *log.Logger
}

// NewCoinswapd returns a bridge that logs once and blocks until ctx is cancelled.
func NewCoinswapd(l *log.Logger) *Coinswapd {
	if l == nil {
		l = log.Default()
	}
	return &Coinswapd{log: l}
}

// Run logs that the bridge is active (no SaveReceipt yet) and waits for shutdown.
func (c *Coinswapd) Run(ctx context.Context) error {
	c.log.Println("mlnd bridge: coinswapd receipt bridge enabled (no-op until JSON-RPC/log surface is finalized); see research/COINSWAPD_TEARDOWN.md")
	<-ctx.Done()
	return nil
}
