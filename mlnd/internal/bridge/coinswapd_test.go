package bridge

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"testing"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/receiptstore"
)

func TestCoinswapdRun_contextCancel(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cancel.db")
	s, err := receiptstore.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	l := log.New(io.Discard, "", 0)
	// Long poll so Run blocks on ctx.Done(), not the ticker.
	b := NewCoinswapd(l, nil, s, dir, 24*time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- b.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}
