// Package makerad holds MLN maker advertisement (Nostr kind 31250) types per research/NOSTR_MLN.md.
package makerad

const (
	KindMakerAd = 31250
	TagTMakerAd = "mln-maker-ad"
)

// Content is the decoded JSON object in the Nostr event content field (wire v1).
type Content struct {
	V            int      `json:"v"`
	Litvm        LitVM    `json:"litvm"`
	Fees         *Fees    `json:"fees,omitempty"`
	Tor          string   `json:"tor,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// LitVM carries deployment pointers; chainId is a decimal string (e.g. "31337").
type LitVM struct {
	ChainID        string `json:"chainId"`
	Registry       string `json:"registry"`
	GrievanceCourt string `json:"grievanceCourt"`
}

// Fees are optional hints; MWEB fees remain authoritative per PRODUCT_SPEC.
type Fees struct {
	Unit string `json:"unit"`
	Min  uint64 `json:"min"`
	Max  uint64 `json:"max"`
}
