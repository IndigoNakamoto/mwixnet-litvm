package mlnroute

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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

// forgerJSONNoLitVM is the mixnet happy-path body: three hops + dest + amount + keys, no epochId/accuser/swapId/operator.
const forgerJSONNoLitVM = `{
  "route": [
    {"tor":"http://n1.onion:8334","feeMinSat":10000,"swapX25519PubHex":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
    {"tor":"http://n2.onion:8334","feeMinSat":10000,"swapX25519PubHex":"fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"},
    {"tor":"http://n3.onion:8334","feeMinSat":10000,"swapX25519PubHex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
  ],
  "destination":"ltcmweb1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqs2d0y0",
  "amount":10000000
}`

func TestValidate_forgerJSONOmitsLitVM_accepted(t *testing.T) {
	t.Parallel()
	var req Request
	if err := json.Unmarshal([]byte(forgerJSONNoLitVM), &req); err != nil {
		t.Fatal(err)
	}
	if err := Validate(&req); err != nil {
		t.Fatalf("mweb_submitRoute must accept mixnet JSON without LitVM fields: %v", err)
	}
	if req.EpochID != "" || req.Accuser != "" || req.SwapID != "" {
		t.Fatalf("unexpected LitVM fields: %+v", req)
	}
	for i, h := range req.Route {
		if h.Operator != "" {
			t.Fatalf("hop %d operator should be empty, got %q", i, h.Operator)
		}
	}
	ops, err := PeerOperatorsFromRequest(&req)
	if err != nil {
		t.Fatal(err)
	}
	var zero [3]common.Address
	if ops != zero {
		t.Fatalf("operators: %+v", ops)
	}
}

func TestValidate_sidecarJSONLitVMWithOperators(t *testing.T) {
	t.Parallel()
	op := "0xcafe000000000000000000000000000000000001"
	raw := `{"route":[` +
		`{"tor":"http://n1","feeMinSat":1,"operator":"` + op + `"},` +
		`{"tor":"http://n2","feeMinSat":2,"operator":"` + op + `"},` +
		`{"tor":"http://n3","feeMinSat":3,"operator":"` + op + `"}],` +
		`"destination":"mweb1x","amount":1000000,"epochId":"7","accuser":"0x1111111111111111111111111111111111111111","swapId":"sw"}`
	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	if err := Validate(&req); err != nil {
		t.Fatal(err)
	}
}
