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
)

// MwebBalance is spendable MWEB balance reported by the MLN sidecar (GET /v1/balance).
type MwebBalance struct {
	AvailableSat uint64 `json:"availableSat"`
	SpendableSat uint64 `json:"spendableSat"`
	Detail       string `json:"detail,omitempty"`
}

type balanceResponseJSON struct {
	Ok           bool    `json:"ok"`
	AvailableSat uint64  `json:"availableSat"`
	SpendableSat *uint64 `json:"spendableSat,omitempty"`
	Error        string  `json:"error,omitempty"`
	Detail       string  `json:"detail,omitempty"`
}

// BalanceURL derives the balance GET URL from the POST /v1/swap sidecar URL by replacing a trailing /swap with /balance.
func BalanceURL(swapURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(swapURL))
	if err != nil {
		return "", fmt.Errorf("forger: parse sidecar URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("forger: sidecar URL must include scheme and host")
	}
	p := strings.TrimSuffix(u.Path, "/")
	if strings.HasSuffix(p, "/swap") {
		u.Path = strings.TrimSuffix(p, "/swap") + "/balance"
	} else {
		u.Path = p + "/balance"
	}
	return u.String(), nil
}

// FetchMwebBalance performs GET on the balance endpoint derived from swapURL.
func FetchMwebBalance(ctx context.Context, swapURL string, httpClient *http.Client) (*MwebBalance, error) {
	balURL, err := BalanceURL(swapURL)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, balURL, nil)
	if err != nil {
		return nil, fmt.Errorf("forger: balance request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("forger: balance http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("forger: balance read body: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("forger: sidecar has no GET balance endpoint (expected %s); add it on your coinswapd MLN fork", balURL)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("forger: balance HTTP %d: %s", resp.StatusCode, string(bytes.TrimSpace(body)))
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, fmt.Errorf("forger: balance empty body")
	}
	var parsed balanceResponseJSON
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("forger: balance JSON: %w", err)
	}
	if !parsed.Ok {
		msg := strings.TrimSpace(parsed.Error)
		if msg == "" {
			msg = "sidecar returned ok=false for balance"
		}
		if d := strings.TrimSpace(parsed.Detail); d != "" {
			return nil, fmt.Errorf("forger: %s (%s)", msg, d)
		}
		return nil, fmt.Errorf("forger: %s", msg)
	}
	out := &MwebBalance{
		AvailableSat: parsed.AvailableSat,
		Detail:       strings.TrimSpace(parsed.Detail),
	}
	if parsed.SpendableSat != nil {
		out.SpendableSat = *parsed.SpendableSat
	} else {
		out.SpendableSat = parsed.AvailableSat
	}
	return out, nil
}
