// Package bridge ingests hop receipts from coinswapd-side emitters into the SQLite vault.
// v1: NDJSON / JSONL files under MLND_BRIDGE_RECEIPTS_DIR (see PHASE_6_BRIDGE_INTEGRATION.md).
package bridge

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/store"
)

const defaultPollInterval = 2 * time.Second

type fileTailState struct {
	off     int64
	partial []byte
}

// Coinswapd tails *.ndjson and *.jsonl in a directory and calls SaveReceipt per complete line.
type Coinswapd struct {
	log    *log.Logger
	store  *store.Store
	dir    string
	poll   time.Duration
	states map[string]*fileTailState
}

// NewCoinswapd returns a bridge that reads receipt lines from dir into store.
// poll must be positive; otherwise defaultPollInterval is used.
func NewCoinswapd(l *log.Logger, st *store.Store, dir string, poll time.Duration) *Coinswapd {
	if l == nil {
		l = log.Default()
	}
	if poll <= 0 {
		poll = defaultPollInterval
	}
	return &Coinswapd{
		log:    l,
		store:  st,
		dir:    dir,
		poll:   poll,
		states: make(map[string]*fileTailState),
	}
}

// Run polls the receipts directory until ctx is cancelled.
func (c *Coinswapd) Run(ctx context.Context) error {
	c.log.Printf("mlnd bridge: watching %q for *.ndjson / *.jsonl (interval %s)", c.dir, c.poll)
	t := time.NewTicker(c.poll)
	defer t.Stop()
	if err := c.scanDir(); err != nil {
		c.log.Printf("mlnd bridge: initial scan: %v", err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := c.scanDir(); err != nil {
				c.log.Printf("mlnd bridge: scan: %v", err)
			}
		}
	}
}

func (c *Coinswapd) scanDir() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ln := strings.ToLower(name)
		if !strings.HasSuffix(ln, ".ndjson") && !strings.HasSuffix(ln, ".jsonl") {
			continue
		}
		path := filepath.Join(c.dir, name)
		if err := c.tailFile(path); err != nil {
			c.log.Printf("mlnd bridge: %s: %v", path, err)
		}
	}
	return nil
}

func (c *Coinswapd) tailFile(path string) error {
	st, ok := c.states[path]
	if !ok {
		st = &fileTailState{}
		c.states[path] = st
	}
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Size() < st.off {
		st.off = 0
		st.partial = nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(st.off, io.SeekStart); err != nil {
		return err
	}
	chunk, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if len(chunk) == 0 {
		return nil
	}
	buf := append(st.partial, chunk...)
	st.off += int64(len(chunk))
	lineStart := 0
	for i := range buf {
		if buf[i] != '\n' {
			continue
		}
		line := buf[lineStart:i]
		lineStart = i + 1
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		rec, err := ParseReceiptLine(line)
		if err != nil {
			c.log.Printf("mlnd bridge: parse line in %s: %v", path, err)
			continue
		}
		if err := c.store.SaveReceipt(rec); err != nil {
			c.log.Printf("mlnd bridge: SaveReceipt: %v", err)
		}
	}
	st.partial = append([]byte(nil), buf[lineStart:]...)
	return nil
}
