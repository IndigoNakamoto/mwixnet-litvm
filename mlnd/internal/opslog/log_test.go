package opslog

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestLog_ringBuffer(t *testing.T) {
	l := New(3)
	for i := 0; i < 5; i++ {
		l.Append(Info, "code", fmt.Sprintf("m%d", i), nil)
	}
	sn := l.Snapshot()
	if len(sn) != 3 {
		t.Fatalf("len=%d want 3", len(sn))
	}
	if sn[0].Message != "m2" || sn[2].Message != "m4" {
		t.Fatalf("messages: %+v", sn)
	}
}

func TestLog_subscribeReceives(t *testing.T) {
	l := New(10)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ch := l.Subscribe(ctx)
	l.Append(Warn, "x", "hello", map[string]string{"k": "v"})
	select {
	case ev := <-ch:
		if ev.Code != "x" || ev.Message != "hello" || ev.Data["k"] != "v" {
			t.Fatalf("%+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestLog_monotonicID(t *testing.T) {
	l := New(5)
	l.Append(Info, "a", "1", nil)
	l.Append(Info, "b", "2", nil)
	sn := l.Snapshot()
	if sn[0].ID >= sn[1].ID {
		t.Fatalf("ids %d %d", sn[0].ID, sn[1].ID)
	}
}
