package receiptstore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/litvmevidence"
	"github.com/ethereum/go-ethereum/common"
)

// ndjsonReceipt is one JSON object per line or file (PHASE_6_BRIDGE_INTEGRATION.md).
type ndjsonReceipt struct {
	EpochID               string `json:"epochId"`
	Accuser               string `json:"accuser"`
	AccusedMaker          string `json:"accusedMaker"`
	HopIndex              int    `json:"hopIndex"`
	PeeledCommitment      string `json:"peeledCommitment"`
	ForwardCiphertextHash string `json:"forwardCiphertextHash"`
	NextHopPubkey         string `json:"nextHopPubkey"`
	Signature             string `json:"signature"`
	SwapID                string `json:"swapId,omitempty"`
}

// ParseReceiptNDJSON decodes one JSON object (NDJSON line or small file) into a ReceiptRecord.
func ParseReceiptNDJSON(raw []byte) (ReceiptRecord, error) {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return ReceiptRecord{}, fmt.Errorf("empty input")
	}
	var j ndjsonReceipt
	if err := json.Unmarshal(raw, &j); err != nil {
		return ReceiptRecord{}, fmt.Errorf("json: %w", err)
	}
	return j.toRecord()
}

func (raw *ndjsonReceipt) toRecord() (ReceiptRecord, error) {
	epochID := strings.TrimSpace(raw.EpochID)
	if epochID == "" {
		return ReceiptRecord{}, fmt.Errorf("epochId required")
	}
	epoch, ok := new(big.Int).SetString(epochID, 10)
	if !ok || epoch.Sign() < 0 {
		return ReceiptRecord{}, fmt.Errorf("invalid epochId %q", raw.EpochID)
	}
	if raw.HopIndex < 0 || raw.HopIndex > 255 {
		return ReceiptRecord{}, fmt.Errorf("hopIndex out of range: %d", raw.HopIndex)
	}
	accuser := strings.TrimSpace(raw.Accuser)
	accused := strings.TrimSpace(raw.AccusedMaker)
	if !common.IsHexAddress(accuser) {
		return ReceiptRecord{}, fmt.Errorf("invalid accuser %q", accuser)
	}
	if !common.IsHexAddress(accused) {
		return ReceiptRecord{}, fmt.Errorf("invalid accusedMaker %q", accused)
	}
	peeled, err := parseBytes32Field("peeledCommitment", raw.PeeledCommitment)
	if err != nil {
		return ReceiptRecord{}, err
	}
	fwd, err := parseBytes32Field("forwardCiphertextHash", raw.ForwardCiphertextHash)
	if err != nil {
		return ReceiptRecord{}, err
	}
	nextHop := strings.TrimSpace(raw.NextHopPubkey)
	sig := strings.TrimSpace(raw.Signature)
	if nextHop == "" {
		return ReceiptRecord{}, fmt.Errorf("nextHopPubkey required")
	}
	if sig == "" {
		return ReceiptRecord{}, fmt.Errorf("signature required")
	}
	return ReceiptRecord{
		EvidencePreimage: litvmevidence.EvidencePreimage{
			EpochID:               epoch,
			Accuser:               common.HexToAddress(accuser),
			AccusedMaker:          common.HexToAddress(accused),
			HopIndex:              uint8(raw.HopIndex),
			PeeledCommitment:      peeled,
			ForwardCiphertextHash: fwd,
		},
		NextHopPubkey: nextHop,
		Signature:     sig,
		SwapID:        strings.TrimSpace(raw.SwapID),
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
