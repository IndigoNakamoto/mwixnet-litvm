package takerflow

import (
	"encoding/json"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/scout"
)

func TestParseRouteJSON_RoundTrip(t *testing.T) {
	t.Parallel()

	r := &pathfind.Route{
		Hops: [3]scout.VerifiedMaker{
			{Tor: "http://a.onion", FeeMinSat: 1},
			{Tor: "http://b.onion", FeeMinSat: 1},
			{Tor: "http://c.onion", FeeMinSat: 1},
		},
		FeeSumSat: 3,
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseRouteJSON(string(b))
	if err != nil {
		t.Fatal(err)
	}
	if got.FeeSumSat != r.FeeSumSat || got.Hops[0].Tor != r.Hops[0].Tor {
		t.Fatalf("got %+v", got)
	}
}

func TestSidecarURL(t *testing.T) {
	t.Parallel()
	if sidecarURL("  http://x  ", "http://def") != "http://x" {
		t.Fatal(sidecarURL("  http://x  ", "http://def"))
	}
	if sidecarURL("", "http://def") != "http://def" {
		t.Fatal()
	}
}
