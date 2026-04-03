// Package opslog holds a bounded, thread-safe log of operator milestones for the HTTP dashboard and SSE.
package opslog

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Level is a coarse severity for dashboard coloring.
type Level string

const (
	Info     Level = "info"
	Warn     Level = "warn"
	Error    Level = "error"
	Critical Level = "critical"
)

// Event is one dashboard-visible milestone (JSON-serializable).
type Event struct {
	ID      uint64            `json:"id"`
	TS      int64             `json:"ts"`
	Level   string            `json:"level"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data,omitempty"`
}

// Log is a ring buffer plus fan-out for live SSE subscribers.
type Log struct {
	mu       sync.RWMutex
	capacity int
	events   []Event
	seq      uint64

	subsMu sync.Mutex
	subs   map[uint64]chan Event
	subSeq uint64
}

// New returns a log that retains at most capacity entries (capacity < 1 defaults to 256).
func New(capacity int) *Log {
	if capacity < 1 {
		capacity = 256
	}
	return &Log{
		capacity: capacity,
		events:   make([]Event, 0, min(capacity, 64)),
		subs:     make(map[uint64]chan Event),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Append adds an event and notifies subscribers.
func (l *Log) Append(level Level, code, msg string, data map[string]string) {
	if l == nil {
		return
	}
	now := time.Now().Unix()
	l.mu.Lock()
	l.seq++
	id := l.seq
	ev := Event{
		ID:      id,
		TS:      now,
		Level:   string(level),
		Code:    code,
		Message: msg,
		Data:    data,
	}
	if len(l.events) >= l.capacity {
		copy(l.events, l.events[1:])
		l.events = l.events[:len(l.events)-1]
	}
	l.events = append(l.events, ev)
	l.mu.Unlock()

	l.subsMu.Lock()
	for _, ch := range l.subs {
		select {
		case ch <- ev:
		default:
		}
	}
	l.subsMu.Unlock()
}

// Snapshot returns a copy of retained events (oldest first).
func (l *Log) Snapshot() []Event {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Event, len(l.events))
	copy(out, l.events)
	return out
}

// Subscribe sends new events until ctx is done; the channel is closed afterward.
func (l *Log) Subscribe(ctx context.Context) <-chan Event {
	ch := make(chan Event, 32)
	if l == nil {
		close(ch)
		return ch
	}
	l.subsMu.Lock()
	id := atomic.AddUint64(&l.subSeq, 1)
	l.subs[id] = ch
	l.subsMu.Unlock()

	go func() {
		<-ctx.Done()
		l.subsMu.Lock()
		delete(l.subs, id)
		l.subsMu.Unlock()
		close(ch)
	}()
	return ch
}
