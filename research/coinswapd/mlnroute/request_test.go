package mlnroute

import "testing"

func TestValidate_ok(t *testing.T) {
	t.Parallel()
	req := &Request{
		Route: []Hop{
			{Tor: "http://a", FeeMinSat: 1},
			{Tor: "http://b", FeeMinSat: 2},
			{Tor: "http://c", FeeMinSat: 3},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	if err := Validate(req); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_wrongHopCount(t *testing.T) {
	t.Parallel()
	req := &Request{
		Route:       []Hop{{Tor: "x"}, {Tor: "y"}},
		Destination: "mweb1",
		Amount:      10,
	}
	if err := Validate(req); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_feesExceedAmount(t *testing.T) {
	t.Parallel()
	req := &Request{
		Route: []Hop{
			{Tor: "a", FeeMinSat: 50},
			{Tor: "b", FeeMinSat: 50},
			{Tor: "c", FeeMinSat: 50},
		},
		Destination: "mweb1",
		Amount:      100,
	}
	if err := Validate(req); err == nil {
		t.Fatal("expected error")
	}
}

func validKey64() string {
	return "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}

func TestValidate_swapKeysAllHops(t *testing.T) {
	t.Parallel()
	k := validKey64()
	req := &Request{
		Route: []Hop{
			{Tor: "http://a", FeeMinSat: 1, SwapX25519PubHex: k},
			{Tor: "http://b", FeeMinSat: 2, SwapX25519PubHex: k},
			{Tor: "http://c", FeeMinSat: 3, SwapX25519PubHex: k},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	if err := Validate(req); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_swapKeysPartialRejected(t *testing.T) {
	t.Parallel()
	k := validKey64()
	req := &Request{
		Route: []Hop{
			{Tor: "http://a", FeeMinSat: 1, SwapX25519PubHex: k},
			{Tor: "http://b", FeeMinSat: 2},
			{Tor: "http://c", FeeMinSat: 3},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	if err := Validate(req); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_swapKeyInvalidHex(t *testing.T) {
	t.Parallel()
	req := &Request{
		Route: []Hop{
			{Tor: "http://a", FeeMinSat: 1, SwapX25519PubHex: "nothex"},
			{Tor: "http://b", FeeMinSat: 2, SwapX25519PubHex: "nothex"},
			{Tor: "http://c", FeeMinSat: 3, SwapX25519PubHex: "nothex"},
		},
		Destination: "mweb1qq",
		Amount:      100,
	}
	if err := Validate(req); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveKeys_withMap(t *testing.T) {
	req := &Request{
		Route: []Hop{
			{Tor: "http://a/", FeeMinSat: 1},
			{Tor: "http://b", FeeMinSat: 1},
			{Tor: "http://c", FeeMinSat: 1},
		},
		Destination: "mweb1",
		Amount:      10,
	}
	k := validKey64()
	m := map[string]string{
		"http://a": k,
		"http://b": k,
		"http://c": k,
	}
	raw, err := ResolveX25519PubKeys(req, m)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 3 {
		t.Fatalf("got %d keys", len(raw))
	}
}
