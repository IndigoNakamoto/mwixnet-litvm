package dashboard

import (
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/opslog"
)

// GrievanceNarrativeView is the latest grievance-related milestone in the ops buffer (operator-facing copy).
type GrievanceNarrativeView struct {
	Headline string            `json:"headline"`
	Detail   string            `json:"detail,omitempty"`
	Level    string            `json:"level"`
	Code     string            `json:"code"`
	TS       int64             `json:"ts"`
	Data     map[string]string `json:"data,omitempty"`
}

var grievanceMilestoneCodes = map[string]struct{}{
	"grievance_opened":          {},
	"receipt_lookup":            {},
	"receipt_missing":           {},
	"receipt_validation_failed": {},
	"manual_defense_required":   {},
	"deadline_check_failed":     {},
	"defend_skipped_deadline":   {},
	"defense_build_failed":      {},
	"defend_dry_run":            {},
	"defend_submitted":          {},
	"defend_submit_failed":      {},
}

// LatestGrievanceNarrative returns the most recent grievance-path event from the ops ring buffer, or nil.
func LatestGrievanceNarrative(l *opslog.Log) *GrievanceNarrativeView {
	if l == nil {
		return nil
	}
	evts := l.Snapshot()
	for i := len(evts) - 1; i >= 0; i-- {
		e := evts[i]
		if _, ok := grievanceMilestoneCodes[e.Code]; !ok {
			continue
		}
		return &GrievanceNarrativeView{
			Headline: grievanceHeadline(e.Code),
			Detail:   e.Message,
			Level:    e.Level,
			Code:     e.Code,
			TS:       e.TS,
			Data:     e.Data,
		}
	}
	return nil
}

func grievanceHeadline(code string) string {
	switch code {
	case "grievance_opened":
		return "Grievance filed on LitVM — your stake may be frozen until resolution"
	case "receipt_lookup":
		return "Looking up hop receipt in the local vault"
	case "receipt_missing":
		return "No matching receipt in vault — auto-defense not possible from this host"
	case "receipt_validation_failed":
		return "Receipt did not match grievance correlators"
	case "manual_defense_required":
		return "Manual defense required before the deadline"
	case "deadline_check_failed":
		return "Could not verify defense deadline against chain time"
	case "defend_skipped_deadline":
		return "Defense window appears closed on-chain"
	case "defense_build_failed":
		return "Could not build defense payload"
	case "defend_dry_run":
		return "Dry-run: would submit defense (no transaction broadcast)"
	case "defend_submitted":
		return "Defense submitted — case Contested; interim judge must adjudicate on-chain"
	case "defend_submit_failed":
		return "Defense transaction failed"
	default:
		return "Grievance workflow update"
	}
}
