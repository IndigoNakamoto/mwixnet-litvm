package forger

// DryRunResult summarizes a successful route Tor validation (no sidecar I/O).
type DryRunResult struct {
	Hops []HopTorSummary `json:"hops"`
}

// HopTorSummary is one hop’s Tor endpoint for display or APIs.
type HopTorSummary struct {
	Index int    `json:"index"` // 1-based (N1, N2, N3)
	Tor   string `json:"tor"`
}

// ExecuteResult is returned when the sidecar accepts the route (HTTP OK and ok=true).
type ExecuteResult struct {
	Detail string `json:"detail,omitempty"`
}
