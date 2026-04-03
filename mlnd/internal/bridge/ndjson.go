package bridge

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/litvm"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/store"
	"github.com/ethereum/go-ethereum/common"
)

// ndjsonReceipt is one JSON object per line (PHASE_6_BRIDGE_INTEGRATION.md).
type ndjsonReceipt struct {
	EpochID               string `json:"epochId"`
	Accuser               string `json:"accuser"`
	AccusedMaker          string `json:"accusedMaker"`
	HopIndex              int    `json:"hopIndex"`
	PeeledCommitment      string `json:"peeledCommitment"`
	ForwardCiphertextHash string `json:"forwardCiphertextHash"`
	NextHopPubkey         string `json:"nextHopPubkey"`
	Signature             string `json:"signature"`
}

// ParseReceiptLine decodes one NDJSON line into a ReceiptRecord for SaveReceipt.
func ParseReceiptLine(line []byte) (store.ReceiptRecord, error) {
	line = []byte(strings.TrimSpace(string(line)))
	if len(line) == 0 {
		return store.ReceiptRecord{}, fmt.Errorf("empty line")
	}
	var raw ndjsonReceipt
	if err := json.Unmarshal(line, &raw); err != nil {
		return store.ReceiptRecord{}, fmt.Errorf("json: %w", err)
	}
	return raw.toRecord()
}

func (raw *ndjsonReceipt) toRecord() (store.ReceiptRecord, error) {
	epochID := strings.TrimSpace(raw.EpochID)
	if epochID == "" {
		return store.ReceiptRecord{}, fmt.Errorf("epochId required")
	}
	epoch, ok := new(big.Int).SetString(epochID, 10)
	if !ok || epoch.Sign() < 0 {
		return store.ReceiptRecord{}, fmt.Errorf("invalid epochId %q", raw.EpochID)
	}
	if raw.HopIndex < 0 || raw.HopIndex > 255 {
		return store.ReceiptRecord{}, fmt.Errorf("hopIndex out of range: %d", raw.HopIndex)
	}
	accuser := strings.TrimSpace(raw.Accuser)
	accused := strings.TrimSpace(raw.AccusedMaker)
	if !common.IsHexAddress(accuser) {
		return store.ReceiptRecord{}, fmt.Errorf("invalid accuser %q", accuser)
	}
	if !common.IsHexAddress(accused) {
		return store.ReceiptRecord{}, fmt.Errorf("invalid accusedMaker %q", accused)
	}
	peeled, err := parseBytes32Field("peeledCommitment", raw.PeeledCommitment)
	if err != nil {
		return store.ReceiptRecord{}, err
	}
	fwd, err := parseBytes32Field("forwardCiphertextHash", raw.ForwardCiphertextHash)
	if err != nil {
		return store.ReceiptRecord{}, err
	}
	nextHop := strings.TrimSpace(raw.NextHopPubkey)
	sig := strings.TrimSpace(raw.Signature)
	if nextHop == "" {
		return store.ReceiptRecord{}, fmt.Errorf("nextHopPubkey required")
	}
	if sig == "" {
		return store.ReceiptRecord{}, fmt.Errorf("signature required")
	}
	return store.ReceiptRecord{
		EvidencePreimage: litvm.EvidencePreimage{
			EpochID:               epoch,
			Accuser:               common.HexToAddress(accuser),
			AccusedMaker:          common.HexToAddress(accused),
			HopIndex:              uint8(raw.HopIndex),
			PeeledCommitment:      peeled,
			ForwardCiphertextHash: fwd,
		},
		NextHopPubkey: nextHop,
		Signature:     sig,
	}, nil
}

func parseBytes32Field(field, hexStr string) (common.Hash, error) {
	s := strings.TrimSpace(strings.ToLower(hexStr))
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return common.Hash{}, fmt.Errorf("%s required", field)
	}
	if len(s) != 64 {
		return common.Hash{}, fmt.Errorf("%s want 64 hex chars, got %d", field, len(s))
	}
	var out [32]byte
	if _, err := hex.Decode(out[:], []byte(s)); err != nil {
		return common.Hash{}, fmt.Errorf("%s: %w", field, err)
	}
	return common.BytesToHash(out[:]), nil
}
