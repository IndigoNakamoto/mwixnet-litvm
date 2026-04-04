package mweb

import (
	"testing"
)

func TestNormalizeMixEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"http://x.onion:8080", "http://x.onion:8080"},
		{"x.onion:8080", "http://x.onion:8080"},
		{"  https://y  ", "https://y"},
	}
	for _, tc := range tests {
		if got := NormalizeMixEndpoint(tc.in); got != tc.want {
			t.Fatalf("NormalizeMixEndpoint(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeSwapRequestHops(t *testing.T) {
	t.Parallel()
	req := &SwapRequest{
		Route: []HopInput{
			{Tor: "n1.onion:80", FeeMinSat: 1},
			{Tor: "http://n2", FeeMinSat: 1},
			{Tor: "n3", FeeMinSat: 1},
		},
		Destination: "mweb1qq",
		Amount:      10,
	}
	NormalizeSwapRequestHops(req)
	if req.Route[0].Tor != "http://n1.onion:80" || req.Route[1].Tor != "http://n2" || req.Route[2].Tor != "http://n3" {
		t.Fatalf("route tors: %+v", req.Route)
	}
}

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

func validKey64() string {
	return "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}

func TestValidateSwapRequest_swapKeysAllHops(t *testing.T) {
	t.Parallel()
	k := validKey64()
	req := &SwapRequest{
		Route: []HopInput{
			{Tor: "http://a", FeeMinSat: 1, SwapX25519PubHex: k},
			{Tor: "http://b", FeeMinSat: 2, SwapX25519PubHex: k},
			{Tor: "http://c", FeeMinSat: 3, SwapX25519PubHex: k},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	if err := ValidateSwapRequest(req); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSwapRequest_swapKeysPartialRejected(t *testing.T) {
	t.Parallel()
	k := validKey64()
	req := &SwapRequest{
		Route: []HopInput{
			{Tor: "http://a", FeeMinSat: 1, SwapX25519PubHex: k},
			{Tor: "http://b", FeeMinSat: 2},
			{Tor: "http://c", FeeMinSat: 3},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	if err := ValidateSwapRequest(req); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateSwapRequest_swapKeyInvalidHex(t *testing.T) {
	t.Parallel()
	req := &SwapRequest{
		Route: []HopInput{
			{Tor: "http://a", FeeMinSat: 1, SwapX25519PubHex: "nothex"},
			{Tor: "http://b", FeeMinSat: 2, SwapX25519PubHex: "nothex"},
			{Tor: "http://c", FeeMinSat: 3, SwapX25519PubHex: "nothex"},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	if err := ValidateSwapRequest(req); err == nil {
		t.Fatal("expected error")
	}
}
