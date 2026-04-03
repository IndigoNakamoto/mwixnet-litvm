package dashboard

import (
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/opslog"
)

func TestLatestGrievanceNarrative_latestWins(t *testing.T) {
	l := opslog.New(20)
	l.Append(opslog.Info, "nostr_published", "noise", nil)
	l.Append(opslog.Info, "grievance_opened", "opened", map[string]string{"grievanceId": "0xab"})
	l.Append(opslog.Info, "receipt_lookup", "lookup", nil)
	g := LatestGrievanceNarrative(l)
	if g == nil {
		t.Fatal("nil")
	}
	if g.Code != "receipt_lookup" {
		t.Fatalf("want receipt_lookup, got %q", g.Code)
	}
}

func TestLatestGrievanceNarrative_nilLog(t *testing.T) {
	if LatestGrievanceNarrative(nil) != nil {
		t.Fatal("expected nil")
	}
}
