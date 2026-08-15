package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ltcmweb/coinswapd/config"
	"github.com/ltcmweb/coinswapd/onion"
)

func TestClearVanillaSwapMeta_keepsDirectoryPeers(t *testing.T) {
	t.Parallel()
	peers := []config.Node{
		config.NewNode("http://a.onion:8334", "aa"),
		config.NewNode("http://b.onion:8335", "bb"),
		config.NewNode("http://c.onion:8336", "cc"),
	}
	ss := &swapService{
		mlnPeers:   peers,
		mlnEpochID: "should-clear",
		mlnAccuser: "0x1",
		mlnSwapID:  "swap",
	}
	ss.clearVanillaSwapMetaLocked()
	if len(ss.mlnPeers) != 3 {
		t.Fatalf("directory peers cleared: %d", len(ss.mlnPeers))
	}
	if ss.mlnEpochID != "" || ss.mlnAccuser != "" || ss.mlnSwapID != "" {
		t.Fatalf("litvm meta still set: %+v", ss)
	}

	vanilla := &swapService{mlnPeers: []config.Node{config.NewNode("http://x", "00")}}
	vanilla.clearVanillaSwapMetaLocked()
	if vanilla.mlnPeers != nil {
		t.Fatalf("non-directory peers should clear: %d", len(vanilla.mlnPeers))
	}
}

type swapCapture struct {
	mu  sync.Mutex
	got *onion.Onion
}

func (s *swapCapture) Swap(_ context.Context, o onion.Onion) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := o
	s.got = &cp
	return nil
}

func TestSwapSwapRemote_dialsHop0(t *testing.T) {
	t.Parallel()
	cap := &swapCapture{}
	srv := rpc.NewServer()
	if err := srv.RegisterName("swap", cap); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(srv.ServeHTTP))
	t.Cleanup(ts.Close)

	ss := &swapService{submitRemote: true}
	o := &onion.Onion{}
	o.PubKey = []byte{1, 2, 3}
	peers := []config.Node{
		config.NewNode(ts.URL, "aa"),
		config.NewNode("http://b.onion:8335", "bb"),
		config.NewNode("http://c.onion:8336", "cc"),
	}
	if err := ss.swapSwapRemote(o, peers); err != nil {
		t.Fatal(err)
	}
	cap.mu.Lock()
	defer cap.mu.Unlock()
	if cap.got == nil || len(cap.got.PubKey) != 3 {
		t.Fatalf("hop 0 did not receive onion: %+v", cap.got)
	}
	if len(ss.mlnPeers) != 3 {
		t.Fatalf("taker mlnPeers=%d", len(ss.mlnPeers))
	}
}

func TestPerformSwap_submitRemoteNoOp(t *testing.T) {
	t.Parallel()
	ss := &swapService{submitRemote: true, nodeIndex: 0}
	if err := ss.performSwap(); err != nil {
		t.Fatal(err)
	}
}
