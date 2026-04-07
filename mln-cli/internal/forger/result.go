package forger

import "time"

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
	Detail            string `json:"detail,omitempty"`
	PendingCleared    bool   `json:"pendingCleared,omitempty"` // wait-batch observed pendingOnions==0
	VaultSwapID       string `json:"vaultSwapId,omitempty"`
	VaultEvidenceHash string `json:"vaultEvidenceHash,omitempty"`
}

// BatchOptions controls optional POST /v1/route/batch and polling GET /v1/route/status after submit.
type BatchOptions struct {
	TriggerBatch    bool
	WaitPendingZero bool
	PollInterval    time.Duration
	Timeout         time.Duration
}
