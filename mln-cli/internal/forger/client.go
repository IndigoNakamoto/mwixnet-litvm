package forger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
)

// RequestPayload is the MLN extension JSON body expected by a coinswapd sidecar (POST /v1/swap).
type RequestPayload struct {
	Route       []HopRequest `json:"route"`
	Destination string       `json:"destination"`
	Amount      uint64       `json:"amount"`
}

// HopRequest is one hop in the sidecar route (Tor mix API + fee hint from the maker ad).
type HopRequest struct {
	Tor              string `json:"tor"`
	FeeMinSat        uint64 `json:"feeMinSat"`
	SwapX25519PubHex string `json:"swapX25519PubHex,omitempty"`
}

// ResponsePayload is the generic JSON response from the sidecar.
type ResponsePayload struct {
	Ok     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

// SidecarClient POSTs route JSON to a local coinswapd MLN extension endpoint.
type SidecarClient struct {
	URL        string
	HTTPClient *http.Client
}

// NewSidecarClient returns a client with a bounded default HTTP timeout (caller may also use context).
func NewSidecarClient(url string) *SidecarClient {
	return &SidecarClient{
		URL: url,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SubmitRoute marshals the route and POSTs it to the sidecar URL.
func (c *SidecarClient) SubmitRoute(ctx context.Context, route *pathfind.Route, dest string, amount uint64) (*ResponsePayload, error) {
	if route == nil {
		return nil, fmt.Errorf("forger: nil route")
	}
	payload := RequestPayload{
		Route:       make([]HopRequest, 0, len(route.Hops)),
		Destination: dest,
		Amount:      amount,
	}
	for _, hop := range route.Hops {
		payload.Route = append(payload.Route, HopRequest{
			Tor:              hop.Tor,
			FeeMinSat:        hop.FeeMinSat,
			SwapX25519PubHex: hop.SwapX25519PubHex,
		})
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("forger: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("forger: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("forger: http post: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("forger: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("forger: sidecar HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	if len(bytes.TrimSpace(respBytes)) == 0 {
		return nil, fmt.Errorf("forger: sidecar returned empty body")
	}

	var parsed ResponsePayload
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("forger: parse sidecar JSON: %w", err)
	}
	return &parsed, nil
}
