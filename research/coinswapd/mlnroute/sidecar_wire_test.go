package mlnroute

import (
	"encoding/json"
	"testing"
)

// Golden bodies: mln-sidecar POST /v1/swap (e2e-mweb-handoff-stub.sh) and mln-sidecar → mweb_submitRoute use the same object.
const sidecarE2EPayload = `{"route":[{"tor":"http://n1","feeMinSat":1},{"tor":"http://n2","feeMinSat":2},{"tor":"http://n3","feeMinSat":3}],"destination":"mweb1x","amount":1000000}`

func TestValidate_sidecarE2EJSON(t *testing.T) {
	t.Parallel()
	var req Request
	if err := json.Unmarshal([]byte(sidecarE2EPayload), &req); err != nil {
		t.Fatal(err)
	}
	if err := Validate(&req); err != nil {
		t.Fatal(err)
	}
	if len(req.Route) != ExpectedHops {
		t.Fatalf("hops %d", len(req.Route))
	}
	if req.Route[0].Tor != "http://n1" || req.Amount != 1000000 {
		t.Fatalf("%+v", req)
	}
}

func TestValidate_sidecarJSONWithAllSwapKeys(t *testing.T) {
	t.Parallel()
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	raw := `{"route":[` +
		`{"tor":"http://a","feeMinSat":1,"swapX25519PubHex":"` + key + `"},` +
		`{"tor":"http://b","feeMinSat":1,"swapX25519PubHex":"` + key + `"},` +
		`{"tor":"http://c","feeMinSat":1,"swapX25519PubHex":"` + key + `"}],` +
		`"destination":"mweb1qq","amount":100}`
	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	if err := Validate(&req); err != nil {
		t.Fatal(err)
	}
}
