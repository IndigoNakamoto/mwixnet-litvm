package forger

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/receiptstore"
	"github.com/ethereum/go-ethereum/common"
)

func TestPersistLastReceiptHTTP(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "vault.db")
	receipt := map[string]interface{}{
		"epochId":               "42",
		"accuser":               "0x1111111111111111111111111111111111111111",
		"accusedMaker":          "0x2222222222222222222222222222222222222222",
		"hopIndex":              1,
		"peeledCommitment":      "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"forwardCiphertextHash": "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"nextHopPubkey":         "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		"signature":             "unsigned-swap-forward-failure-v1",
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	hr := &LastReceiptHTTP{
		Ok:      true,
		Receipt: raw,
		SwapID:  "poll-swap-1",
	}
	ev, ins, err := PersistLastReceiptHTTP(dbPath, hr)
	if err != nil || ev == "" || !ins {
		t.Fatalf("persist: ev=%q ins=%v err=%v", ev, ins, err)
	}
	st, err := receiptstore.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	def, err := st.GetBySwapID("poll-swap-1")
	if err != nil {
		t.Fatal(err)
	}
	if def.AccusedMaker != common.HexToAddress("0x2222222222222222222222222222222222222222") {
		t.Fatalf("row accused: %+v", def)
	}
}
