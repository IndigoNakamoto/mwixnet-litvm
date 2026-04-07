package forger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
)

// RequestPayload is the MLN extension JSON body expected by a coinswapd sidecar (POST /v1/swap).
type RequestPayload struct {
	Route       []HopRequest `json:"route"`
	Destination string       `json:"destination"`
	Amount      uint64       `json:"amount"`
	EpochID     string       `json:"epochId,omitempty"`
	Accuser     string       `json:"accuser,omitempty"`
	SwapID      string       `json:"swapId,omitempty"`
}

// RouteMeta is LitVM coordination threaded with the route (epoch + taker + attempt id).
type RouteMeta struct {
	EpochID string
	Accuser string
	SwapID  string
}

// HopRequest is one hop in the sidecar route (Tor mix API + fee hint from the maker ad).
type HopRequest struct {
	Tor              string `json:"tor"`
	FeeMinSat        uint64 `json:"feeMinSat"`
	SwapX25519PubHex string `json:"swapX25519PubHex,omitempty"`
}

// ResponsePayload is the generic JSON response from the sidecar.
type ResponsePayload struct {
	Ok      bool            `json:"ok"`
	Detail  string          `json:"detail,omitempty"`
	Error   string          `json:"error,omitempty"`
	SwapID  string          `json:"swapId,omitempty"`
	Receipt json.RawMessage `json:"receipt,omitempty"`
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

// SidecarBaseFromSwapURL returns the sidecar origin + path prefix without /v1/swap (for /v1/route/* helpers).
func SidecarBaseFromSwapURL(swapURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(swapURL))
	if err != nil {
		return "", fmt.Errorf("forger: parse sidecar URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("forger: sidecar URL must include scheme and host")
	}
	p := strings.TrimSuffix(strings.TrimSpace(u.Path), "/")
	if strings.HasSuffix(p, "/v1/swap") {
		p = strings.TrimSuffix(p, "/v1/swap")
	}
	u.Path = p
	if u.Path == "" {
		u.Path = "/"
	}
	u.RawQuery, u.Fragment = "", ""
	return strings.TrimRight(u.String(), "/"), nil
}

// RouteStatusPayload is GET /v1/route/status from mln-sidecar.
type RouteStatusPayload struct {
	Ok                     bool   `json:"ok"`
	PendingOnions          int    `json:"pendingOnions"`
	MlnRouteHops           int    `json:"mlnRouteHops"`
	NodeIndex              int    `json:"nodeIndex"`
	NeutrinoConnectedPeers int    `json:"neutrinoConnectedPeers"`
	Detail                 string `json:"detail,omitempty"`
	Error                  string `json:"error,omitempty"`
}

// BatchPayload is POST /v1/route/batch from mln-sidecar.
type BatchPayload struct {
	Ok      bool            `json:"ok"`
	Detail  string          `json:"detail,omitempty"`
	Error   string          `json:"error,omitempty"`
	SwapID  string          `json:"swapId,omitempty"`
	Receipt json.RawMessage `json:"receipt,omitempty"`
}

// GetRouteStatus calls GET /v1/route/status on the sidecar hosting swapURL.
func (c *SidecarClient) GetRouteStatus(ctx context.Context, swapURL string) (*RouteStatusPayload, error) {
	base, err := SidecarBaseFromSwapURL(swapURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/route/status", nil)
	if err != nil {
		return nil, fmt.Errorf("forger: status request: %w", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("forger: route status http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("forger: read status: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("forger: route status HTTP %d: %s", resp.StatusCode, string(body))
	}
	var out RouteStatusPayload
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("forger: route status JSON: %w", err)
	}
	if !out.Ok {
		msg := strings.TrimSpace(out.Error)
		if msg == "" {
			msg = "route status ok=false"
		}
		return nil, fmt.Errorf("forger: %s (%s)", msg, strings.TrimSpace(out.Detail))
	}
	return &out, nil
}

// RunBatch calls POST /v1/route/batch (triggers coinswapd performSwap via sidecar RPC).
func (c *SidecarClient) RunBatch(ctx context.Context, swapURL string) (*BatchPayload, error) {
	base, err := SidecarBaseFromSwapURL(swapURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/route/batch", nil)
	if err != nil {
		return nil, fmt.Errorf("forger: batch request: %w", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("forger: batch http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("forger: read batch: %w", err)
	}
	var out BatchPayload
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			if resp.StatusCode >= 300 {
				return nil, fmt.Errorf("forger: batch HTTP %d: %s", resp.StatusCode, string(body))
			}
			return nil, fmt.Errorf("forger: batch JSON: %w", err)
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(out.Receipt) > 0 {
			return &out, fmt.Errorf("forger: batch HTTP %d (%s)", resp.StatusCode, strings.TrimSpace(out.Error))
		}
		return nil, fmt.Errorf("forger: batch HTTP %d: %s", resp.StatusCode, string(body))
	}
	if !out.Ok {
		msg := strings.TrimSpace(out.Error)
		if msg == "" {
			msg = "batch ok=false"
		}
		if len(out.Receipt) > 0 {
			return &out, fmt.Errorf("forger: %s (%s)", msg, strings.TrimSpace(out.Detail))
		}
		return nil, fmt.Errorf("forger: %s (%s)", msg, strings.TrimSpace(out.Detail))
	}
	return &out, nil
}

// SubmitRoute marshals the route and POSTs it to the sidecar URL.
// meta may be nil; when set, epochId/accuser/swapId are included for LitVM receipt threading.
func (c *SidecarClient) SubmitRoute(ctx context.Context, route *pathfind.Route, dest string, amount uint64, meta *RouteMeta) (*ResponsePayload, error) {
	if route == nil {
		return nil, fmt.Errorf("forger: nil route")
	}
	payload := RequestPayload{
		Route:       make([]HopRequest, 0, len(route.Hops)),
		Destination: dest,
		Amount:      amount,
	}
	if meta != nil {
		payload.EpochID = strings.TrimSpace(meta.EpochID)
		payload.Accuser = strings.TrimSpace(meta.Accuser)
		payload.SwapID = strings.TrimSpace(meta.SwapID)
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

	if len(bytes.TrimSpace(respBytes)) == 0 {
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("forger: sidecar HTTP %d: empty body", resp.StatusCode)
		}
		return nil, fmt.Errorf("forger: sidecar returned empty body")
	}

	var parsed ResponsePayload
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("forger: sidecar HTTP %d: %s", resp.StatusCode, string(respBytes))
		}
		return nil, fmt.Errorf("forger: parse sidecar JSON: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(parsed.Receipt) > 0 {
			return &parsed, fmt.Errorf("forger: sidecar HTTP %d (%s)", resp.StatusCode, strings.TrimSpace(parsed.Error))
		}
		return nil, fmt.Errorf("forger: sidecar HTTP %d: %s", resp.StatusCode, string(respBytes))
	}
	return &parsed, nil
}
