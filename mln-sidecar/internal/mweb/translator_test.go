package mweb

import "testing"

func TestValidateSwapRequest_ok(t *testing.T) {
	t.Parallel()
	req := &SwapRequest{
		Route: []HopInput{
			{Tor: "http://a", FeeMinSat: 1},
			{Tor: "http://b", FeeMinSat: 2},
			{Tor: "http://c", FeeMinSat: 3},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	if err := ValidateSwapRequest(req); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSwapRequest_wrongHopCount(t *testing.T) {
	t.Parallel()
	req := &SwapRequest{
		Route:       []HopInput{{Tor: "x"}, {Tor: "y"}},
		Destination: "mweb1",
		Amount:      10,
	}
	if err := ValidateSwapRequest(req); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateSwapRequest_feesExceedAmount(t *testing.T) {
	t.Parallel()
	req := &SwapRequest{
		Route: []HopInput{
			{Tor: "a", FeeMinSat: 50},
			{Tor: "b", FeeMinSat: 50},
			{Tor: "c", FeeMinSat: 50},
		},
		Destination: "mweb1",
		Amount:      100,
	}
	if err := ValidateSwapRequest(req); err == nil {
		t.Fatal("expected error")
	}
}
