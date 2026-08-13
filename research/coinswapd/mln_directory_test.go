package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMLNDirectoryPeers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixnet.directory.json")
	privs := make([]*ecdh.PrivateKey, 3)
	pubs := make([]string, 3)
	for i := range privs {
		k, err := ecdh.X25519().GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		privs[i] = k
		pubs[i] = hex.EncodeToString(k.PublicKey().Bytes())
	}
	body := `{
  "amountSat": 10000000,
  "hops": [
    {"tor": "http://a.onion:8334", "swapX25519PubHex": "` + pubs[0] + `", "feeMinSat": 10000},
    {"tor": "http://b.onion:8335", "swapX25519PubHex": "` + pubs[1] + `", "feeMinSat": 10000},
    {"tor": "http://c.onion:8336", "swapX25519PubHex": "` + pubs[2] + `", "feeMinSat": 10000}
  ]
}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	peers, err := loadMLNDirectoryPeers(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 3 {
		t.Fatalf("len=%d", len(peers))
	}
	if peers[1].Url != "http://b.onion:8335" {
		t.Fatalf("url=%s", peers[1].Url)
	}

	ss := &swapService{}
	if err := applyMLNDirectory(ss, path, 1, false, privs[1].PublicKey()); err != nil {
		t.Fatal(err)
	}
	if ss.nodeIndex != 1 || len(ss.mlnPeers) != 3 {
		t.Fatalf("nodeIndex=%d peers=%d", ss.nodeIndex, len(ss.mlnPeers))
	}
	if err := applyMLNDirectory(ss, path, 1, false, privs[0].PublicKey()); err == nil {
		t.Fatal("expected mismatch on -k")
	}
	if err := applyMLNDirectory(ss, path, 2, true, privs[2].PublicKey()); err == nil {
		t.Fatal("expected local-taker vs hop-index conflict")
	}
}

func TestLoadMLNDirectoryPeers_rejectMixedCaseHex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	body := `{
  "amountSat": 10000000,
  "hops": [
    {"tor": "http://a.onion:8334", "swapX25519PubHex": "` + strings.Repeat("A", 64) + `", "feeMinSat": 10000},
    {"tor": "http://b.onion:8335", "swapX25519PubHex": "` + strings.Repeat("b", 64) + `", "feeMinSat": 10000},
    {"tor": "http://c.onion:8336", "swapX25519PubHex": "` + strings.Repeat("c", 64) + `", "feeMinSat": 10000}
  ]
}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadMLNDirectoryPeers(path); err == nil {
		t.Fatal("expected uppercase hex reject")
	}
}
