package dashboard

import "strings"

// ShortenRPCDisplay trims long WebSocket/HTTP URLs for the dashboard header (no secrets expected; strips obvious query).
func ShortenRPCDisplay(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "?"); i >= 0 {
		s = s[:i]
	}
	const max = 52
	if len(s) <= max {
		return s
	}
	head, tail := 22, 22
	return s[:head] + "…" + s[len(s)-tail:]
}
