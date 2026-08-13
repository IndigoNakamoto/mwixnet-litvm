package mixnetdir

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
)

func examplePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	return filepath.Join(root, "deploy", "mixnet.directory.example.json")
}

func TestLoad_exampleDirectory(t *testing.T) {
	t.Parallel()
	d, err := Load(examplePath(t))
	if err != nil {
		t.Fatal(err)
	}
	if d.AmountSat != 10_000_000 {
		t.Fatalf("amountSat %d", d.AmountSat)
	}
	route, err := ToPathfindRoute(d)
	if err != nil {
		t.Fatal(err)
	}
	if route.FeeSumSat != 30_000 {
		t.Fatalf("feeSum %d", route.FeeSumSat)
	}
	payload, err := ToForgerPayload(d, "ltcmweb1qq")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if payload.EpochID != "" || payload.Accuser != "" || payload.SwapID != "" {
		t.Fatalf("litvm on payload: %+v", payload)
	}
	for _, h := range payload.Route {
		if h.Operator != "" {
			t.Fatalf("operator on hop: %+v", h)
		}
	}
	for _, k := range []string{"epochId", "accuser", "swapId", `"operator"`} {
		if strings.Contains(s, k) {
			t.Fatalf("marshaled payload has %s: %s", k, s)
		}
	}
	routeJSON, err := MarshalRouteJSON(d)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(routeJSON), "operator") || strings.Contains(string(routeJSON), "epochId") {
		t.Fatalf("route.json has LitVM fields: %s", routeJSON)
	}
	var parsed pathfind.Route
	if err := json.Unmarshal(routeJSON, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Hops[0].Tor == "" || parsed.Hops[0].FeeMinSat != 10000 {
		t.Fatalf("forger unmarshal %+v", parsed)
	}
}

func TestValidate_rejectsMixedHopCount(t *testing.T) {
	t.Parallel()
	d := &Directory{AmountSat: 100, Hops: []Hop{{Tor: "http://a", SwapX25519PubHex: hex64(), FeeMinSat: 1}}}
	if err := Validate(d); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_missingFile(t *testing.T) {
	t.Parallel()
	if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("expected error")
	}
}

func TestToForgerPayload_requiresDest(t *testing.T) {
	t.Parallel()
	d, err := Load(examplePath(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ToForgerPayload(d, "  "); err == nil {
		t.Fatal("expected dest error")
	}
}

func hex64() string {
	return "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}
