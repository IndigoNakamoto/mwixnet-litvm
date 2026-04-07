package mlnroute

import (
	"encoding/json"
	"testing"
)

func TestValidate_litvmRequiresOperators(t *testing.T) {
	t.Parallel()
	raw := `{"route":[` +
		`{"tor":"http://a","feeMinSat":1},` +
		`{"tor":"http://b","feeMinSat":1},` +
		`{"tor":"http://c","feeMinSat":1}],` +
		`"destination":"mweb1qq","amount":100,"epochId":"1","accuser":"0x1111111111111111111111111111111111111111","swapId":"s"}`
	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	if err := Validate(&req); err == nil {
		t.Fatal("expected error without operators")
	}
}

func TestValidate_litvmWithOperators(t *testing.T) {
	t.Parallel()
	op := "0xdead000000000000000000000000000000000001"
	raw := `{"route":[` +
		`{"tor":"http://a","feeMinSat":1,"operator":"` + op + `"},` +
		`{"tor":"http://b","feeMinSat":1,"operator":"` + op + `"},` +
		`{"tor":"http://c","feeMinSat":1,"operator":"` + op + `"}],` +
		`"destination":"mweb1qq","amount":100,"epochId":"1","accuser":"0x1111111111111111111111111111111111111111","swapId":"s"}`
	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	if err := Validate(&req); err != nil {
		t.Fatal(err)
	}
	ops, err := PeerOperatorsFromRequest(&req)
	if err != nil {
		t.Fatal(err)
	}
	for i := range ops {
		if ops[i].Hex() != op {
			t.Fatalf("hop %d got %s", i, ops[i].Hex())
		}
	}
}
