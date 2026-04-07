package forger

import (
	"fmt"
	"strings"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/litvmevidence"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/receiptstore"
)

// VaultOptions configures SQLite receipt persistence after sidecar responses.
type VaultOptions struct {
	DBPath  string
	EpochID string
	Accuser string
	SwapID  string // if empty, caller must generate before SubmitRoute
}

// PersistReceiptFromResponse parses receipt JSON from a swap response and saves via evidenceHash (idempotent).
// Returns canonical evidenceHash hex and whether a new row was inserted.
func PersistReceiptFromResponse(dbPath string, payload *ResponsePayload) (evidenceHashHex string, inserted bool, err error) {
	if strings.TrimSpace(dbPath) == "" || payload == nil || len(payload.Receipt) == 0 {
		return "", false, nil
	}
	rec, err := receiptstore.ParseReceiptNDJSON(payload.Receipt)
	if err != nil {
		return "", false, fmt.Errorf("vault: parse receipt: %w", err)
	}
	if sid := strings.TrimSpace(payload.SwapID); sid != "" {
		rec.SwapID = sid
	}
	st, err := receiptstore.NewStore(dbPath)
	if err != nil {
		return "", false, fmt.Errorf("vault: open db: %w", err)
	}
	defer st.Close()
	inserted, err = st.SaveReceipt(rec)
	if err != nil {
		return "", false, fmt.Errorf("vault: SaveReceipt: %w", err)
	}
	pre := litvmevidence.EvidencePreimage{
		EpochID:               rec.EpochID,
		Accuser:               rec.Accuser,
		AccusedMaker:          rec.AccusedMaker,
		HopIndex:              rec.HopIndex,
		PeeledCommitment:      rec.PeeledCommitment,
		ForwardCiphertextHash: rec.ForwardCiphertextHash,
	}
	ev := litvmevidence.ComputeEvidenceHash(pre)
	return ev.Hex(), inserted, nil
}

// PersistBatchReceiptFromResponse saves a receipt returned on POST /v1/route/batch when present.
func PersistBatchReceiptFromResponse(dbPath string, payload *BatchPayload) (evidenceHashHex string, inserted bool, err error) {
	if strings.TrimSpace(dbPath) == "" || payload == nil || len(payload.Receipt) == 0 {
		return "", false, nil
	}
	rp := &ResponsePayload{
		Ok:      payload.Ok,
		Detail:  payload.Detail,
		Error:   payload.Error,
		SwapID:  payload.SwapID,
		Receipt: payload.Receipt,
	}
	return PersistReceiptFromResponse(dbPath, rp)
}

// PersistLastReceiptHTTP saves a receipt from GET /v1/route/receipt after async swap_forward failure.
func PersistLastReceiptHTTP(dbPath string, hr *LastReceiptHTTP) (evidenceHashHex string, inserted bool, err error) {
	if strings.TrimSpace(dbPath) == "" || hr == nil || len(hr.Receipt) == 0 {
		return "", false, nil
	}
	rec, err := receiptstore.ParseReceiptNDJSON(hr.Receipt)
	if err != nil {
		return "", false, fmt.Errorf("vault: parse receipt from poll: %w", err)
	}
	if sid := strings.TrimSpace(hr.SwapID); sid != "" && strings.TrimSpace(rec.SwapID) == "" {
		rec.SwapID = sid
	}
	st, err := receiptstore.NewStore(dbPath)
	if err != nil {
		return "", false, fmt.Errorf("vault: open db: %w", err)
	}
	defer st.Close()
	inserted, err = st.SaveReceipt(rec)
	if err != nil {
		return "", false, fmt.Errorf("vault: SaveReceipt: %w", err)
	}
	pre := litvmevidence.EvidencePreimage{
		EpochID:               rec.EpochID,
		Accuser:               rec.Accuser,
		AccusedMaker:          rec.AccusedMaker,
		HopIndex:              rec.HopIndex,
		PeeledCommitment:      rec.PeeledCommitment,
		ForwardCiphertextHash: rec.ForwardCiphertextHash,
	}
	ev := litvmevidence.ComputeEvidenceHash(pre)
	return ev.Hex(), inserted, nil
}
