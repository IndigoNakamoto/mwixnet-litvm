package dashboard

import "testing"

func TestShortenRPCDisplay(t *testing.T) {
	if got := ShortenRPCDisplay("ws://127.0.0.1:8545"); got != "ws://127.0.0.1:8545" {
		t.Fatalf("%q", got)
	}
	// Must exceed ShortenRPCDisplay max (52) so truncation path runs.
	long := "wss://very-long-subdomain.example.com:8545/some/longer/path"
	got := ShortenRPCDisplay(long)
	if len(got) >= len(long) {
		t.Fatalf("expected shorter: %q", got)
	}
	if got := ShortenRPCDisplay("wss://x.com?token=secret"); got != "wss://x.com" {
		t.Fatalf("strip query: %q", got)
	}
}
