package bridge

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/litvm"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/store"
	"github.com/ethereum/go-ethereum/common"
)

func TestCoinswapd_tailFile_SaveReceipt(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "b.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	line := []byte(`{"epochId":"7","accuser":"0x0000000000000000000000000000000000000cc3","accusedMaker":"0x0000000000000000000000000000000000000dd4","hopIndex":1,"peeledCommitment":"0x` +
		string(bytesRepeat('e', 64)) + `","forwardCiphertextHash":"0x` + string(bytesRepeat('f', 64)) +
		`","nextHopPubkey":"npub-bridge","signature":"sig-bridge"}` + "\n")
	path := filepath.Join(dir, "r.ndjson")
	if err := os.WriteFile(path, line, 0o644); err != nil {
		t.Fatal(err)
	}

	lg := log.New(io.Discard, "", 0)
	c := NewCoinswapd(lg, s, dir, defaultPollInterval)
	if err := c.tailFile(path); err != nil {
		t.Fatal(err)
	}

	rec, err := ParseReceiptLine(line)
	if err != nil {
		t.Fatal(err)
	}
	ev := litvm.ComputeEvidenceHash(rec.EvidencePreimage)
	got, err := s.GetByEvidenceHash(ev)
	if err != nil {
		t.Fatal(err)
	}
	if got.HopIndex != 1 || got.NextHopPubkey != "npub-bridge" {
		t.Fatalf("got %+v", got)
	}
	if got.Accuser != common.HexToAddress("0x0000000000000000000000000000000000000cc3") {
		t.Fatalf("accuser %s", got.Accuser.Hex())
	}
}
