package bridge

import (
	"context"
	"io"
	"log"
	"testing"
	"time"
)

func TestCoinswapdRun_contextCancel(t *testing.T) {
	l := log.New(io.Discard, "", 0)
	b := NewCoinswapd(l)
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
