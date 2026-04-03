// Package nostr publishes MLN maker advertisements (kind 31250) per research/NOSTR_MLN.md.
package nostr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gnostr "github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

const (
	kindMakerAd = 31250
	tagTMakerAd = "mln-maker-ad"
)

// BroadcasterConfig holds static fields for the maker ad (all EVM addresses lowercase 0x per spec).
type BroadcasterConfig struct {
	ChainID        string
	Registry       string
	GrievanceCourt string
	Operator       string // maker LitVM address -> d tag + binding
	TorOnion       string // optional, e.g. http://....onion
	FeeMinSat      *uint64
	FeeMaxSat      *uint64
	Capabilities   []string
	ClientName     string
	ClientVersion  string
}

// errRelayBackoff means this relay URL is in exponential backoff; skip until next tick.
var errRelayBackoff = errors.New("nostr relay in backoff")

// Broadcaster republishes replaceable maker ads on an interval.
type Broadcaster struct {
	cfg      BroadcasterConfig
	relays   []string
	secHex   string // 64-char hex private key for gnostr.Event.Sign
	interval time.Duration
	log      *log.Logger

	relayMu        sync.Mutex
	relayByURL     map[string]*gnostr.Relay
	relayFailCount map[string]int
	relayNextTry   map[string]time.Time
}

// NewBroadcaster returns a configured broadcaster (e.g. for tests). relays may be empty if only BuildMakerAdEvent is used.
func NewBroadcaster(cfg BroadcasterConfig, relays []string, secHex string, interval time.Duration, lg *log.Logger) *Broadcaster {
	if lg == nil {
		lg = log.Default()
	}
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &Broadcaster{
		cfg:      cfg,
		relays:   relays,
		secHex:   secHex,
		interval: interval,
		log:      lg,
	}
}

// LoadBroadcasterFromEnv returns nil if MLND_NOSTR_RELAYS is unset (broadcaster disabled).
func LoadBroadcasterFromEnv() (*Broadcaster, error) {
	relaysRaw := strings.TrimSpace(os.Getenv("MLND_NOSTR_RELAYS"))
	if relaysRaw == "" {
		return nil, nil
	}

	secHex, err := parseSigningKey(strings.TrimSpace(os.Getenv("MLND_NOSTR_NSEC")))
	if err != nil {
		return nil, fmt.Errorf("MLND_NOSTR_NSEC: %w", err)
	}

	chainID := strings.TrimSpace(os.Getenv("MLND_LITVM_CHAIN_ID"))
	if chainID == "" {
		return nil, fmt.Errorf("MLND_LITVM_CHAIN_ID is required when MLND_NOSTR_RELAYS is set")
	}
	reg := strings.TrimSpace(os.Getenv("MLND_REGISTRY_ADDR"))
	if reg == "" {
		return nil, fmt.Errorf("MLND_REGISTRY_ADDR is required when MLND_NOSTR_RELAYS is set")
	}
	court := strings.TrimSpace(os.Getenv("MLND_COURT_ADDR"))
	if court == "" {
		return nil, fmt.Errorf("MLND_COURT_ADDR is required when MLND_NOSTR_RELAYS is set")
	}
	op := strings.TrimSpace(os.Getenv("MLND_OPERATOR_ADDR"))
	if op == "" {
		return nil, fmt.Errorf("MLND_OPERATOR_ADDR is required when MLND_NOSTR_RELAYS is set")
	}

	regNorm, err := normalizeETHAddr(reg)
	if err != nil {
		return nil, fmt.Errorf("MLND_REGISTRY_ADDR: %w", err)
	}
	courtNorm, err := normalizeETHAddr(court)
	if err != nil {
		return nil, fmt.Errorf("MLND_COURT_ADDR: %w", err)
	}
	opNorm, err := normalizeETHAddr(op)
	if err != nil {
		return nil, fmt.Errorf("MLND_OPERATOR_ADDR: %w", err)
	}

	interval := 30 * time.Minute
	if s := strings.TrimSpace(os.Getenv("MLND_NOSTR_INTERVAL")); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("MLND_NOSTR_INTERVAL: %w", err)
		}
		interval = d
	}

	var feeMin, feeMax *uint64
	if s := strings.TrimSpace(os.Getenv("MLND_FEE_MIN_SAT")); s != "" {
		var v uint64
		if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
			return nil, fmt.Errorf("MLND_FEE_MIN_SAT: %w", err)
		}
		feeMin = &v
	}
	if s := strings.TrimSpace(os.Getenv("MLND_FEE_MAX_SAT")); s != "" {
		var v uint64
		if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
			return nil, fmt.Errorf("MLND_FEE_MAX_SAT: %w", err)
		}
		feeMax = &v
	}

	relays := splitRelays(relaysRaw)
	torRaw := strings.TrimSpace(os.Getenv("MLND_TOR_ONION"))
	torNorm := torOnionWithOptionalPort(torRaw, strings.TrimSpace(os.Getenv("MLND_TOR_PORT")))
	cfg := BroadcasterConfig{
		ChainID:        chainID,
		Registry:       regNorm,
		GrievanceCourt: courtNorm,
		Operator:       opNorm,
		TorOnion:       torNorm,
		FeeMinSat:      feeMin,
		FeeMaxSat:      feeMax,
		Capabilities:   []string{"mweb-coinswap-v0"},
		ClientName:     "mlnd",
		ClientVersion:  "0",
	}

	return NewBroadcaster(cfg, relays, secHex, interval, log.Default()), nil
}

// torOnionWithOptionalPort returns baseTor unchanged if portStr is empty or baseTor already has a host port.
// If baseTor parses as a URL with a host but no port, net.JoinHostPort is applied (see MLND_TOR_PORT in README).
func torOnionWithOptionalPort(baseTor, portStr string) string {
	if baseTor == "" || portStr == "" {
		return baseTor
	}
	u, err := url.Parse(baseTor)
	if err != nil || u.Host == "" {
		return baseTor
	}
	if u.Port() != "" {
		return baseTor
	}
	host := u.Hostname()
	if host == "" {
		return baseTor
	}
	u.Host = net.JoinHostPort(host, portStr)
	return u.String()
}

func splitRelays(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseSigningKey(s string) (hex64 string, err error) {
	if s == "" {
		return "", fmt.Errorf("empty signing key")
	}
	if strings.HasPrefix(s, "nsec1") {
		prefix, val, err := nip19.Decode(s)
		if err != nil {
			return "", err
		}
		if prefix != "nsec" {
			return "", fmt.Errorf("expected nsec, got %q", prefix)
		}
		sk, ok := val.(string)
		if !ok || len(sk) != 64 {
			return "", fmt.Errorf("invalid nsec decode")
		}
		return sk, nil
	}
	key := strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if len(key) != 64 {
		return "", fmt.Errorf("expected 64 hex chars or nsec1…")
	}
	for _, c := range key {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", fmt.Errorf("private key must be hex")
		}
	}
	return strings.ToLower(key), nil
}

func normalizeETHAddr(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		s = "0x" + s
	}
	if !common.IsHexAddress(s) {
		return "", fmt.Errorf("invalid EVM address %q", s)
	}
	addr := common.HexToAddress(s)
	return strings.ToLower(addr.Hex()), nil
}

// DTag returns the NIP-33 d tag mln:v1:<chainId>:<operatorLower>.
func DTag(chainID, operatorLower string) string {
	return fmt.Sprintf("mln:v1:%s:%s", chainID, operatorLower)
}

type makerAdContent struct {
	V            int       `json:"v"`
	Litvm        litvmJSON `json:"litvm"`
	Fees         *feesJSON `json:"fees,omitempty"`
	Tor          string    `json:"tor,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
}

type litvmJSON struct {
	ChainID        string `json:"chainId"`
	Registry       string `json:"registry"`
	GrievanceCourt string `json:"grievanceCourt"`
}

type feesJSON struct {
	Unit string `json:"unit"`
	Min  uint64 `json:"min"`
	Max  uint64 `json:"max"`
}

// BuildMakerAdEvent builds and signs a kind-31250 replaceable maker ad at now.
func (b *Broadcaster) BuildMakerAdEvent(now time.Time) (*gnostr.Event, error) {
	body := makerAdContent{
		V: 1,
		Litvm: litvmJSON{
			ChainID:        b.cfg.ChainID,
			Registry:       b.cfg.Registry,
			GrievanceCourt: b.cfg.GrievanceCourt,
		},
		Capabilities: b.cfg.Capabilities,
	}
	if b.cfg.TorOnion != "" {
		body.Tor = b.cfg.TorOnion
	}
	if b.cfg.FeeMinSat != nil && b.cfg.FeeMaxSat != nil {
		body.Fees = &feesJSON{
			Unit: "sat_per_hop",
			Min:  *b.cfg.FeeMinSat,
			Max:  *b.cfg.FeeMaxSat,
		}
	}
	contentBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	d := DTag(b.cfg.ChainID, b.cfg.Operator)
	tags := gnostr.Tags{
		{"d", d},
		{"t", tagTMakerAd},
	}
	if b.cfg.ClientName != "" {
		tags = append(tags, gnostr.Tag{"client", b.cfg.ClientName, b.cfg.ClientVersion})

	}

	ev := &gnostr.Event{
		CreatedAt: gnostr.Timestamp(now.Unix()),
		Kind:      kindMakerAd,
		Tags:      tags,
		Content:   string(contentBytes),
	}
	if err := ev.Sign(b.secHex); err != nil {
		return nil, err
	}
	ok, err := ev.CheckSignature()
	if err != nil || !ok {
		return nil, fmt.Errorf("signature check failed: ok=%v err=%v", ok, err)
	}
	return ev, nil
}

// Publish sends the event to each relay; errors are logged per relay only.
// Relays are reused across ticks; failed publishes trigger close, exponential backoff, and reconnect.
func (b *Broadcaster) Publish(ctx context.Context, ev gnostr.Event) {
	pubCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	for _, relayURL := range b.relays {
		r, err := b.ensureRelay(pubCtx, relayURL)
		if errors.Is(err, errRelayBackoff) {
			continue
		}
		if err != nil {
			b.log.Printf("mlnd nostr: connect %s: %v", relayURL, err)
			continue
		}
		if err := r.Publish(pubCtx, ev); err != nil {
			b.log.Printf("mlnd nostr: publish %s: %v", relayURL, err)
			b.forgetRelayAfterFailure(relayURL)
			continue
		}
		b.markRelayOK(relayURL)
		b.log.Printf("mlnd nostr: published kind=%d id=%s… to %s", ev.Kind, trimID(ev.ID), relayURL)
	}
}

func (b *Broadcaster) ensureRelay(ctx context.Context, relayURL string) (*gnostr.Relay, error) {
	b.relayMu.Lock()
	if t, ok := b.relayNextTry[relayURL]; ok && time.Now().Before(t) {
		b.relayMu.Unlock()
		return nil, errRelayBackoff
	}
	if r := b.relayByURL[relayURL]; r != nil && r.IsConnected() {
		b.relayMu.Unlock()
		return r, nil
	}
	if r := b.relayByURL[relayURL]; r != nil {
		_ = r.Close()
		delete(b.relayByURL, relayURL)
	}
	b.relayMu.Unlock()

	connCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	r := gnostr.NewRelay(ctx, relayURL)
	if err := r.Connect(connCtx); err != nil {
		b.forgetRelayAfterFailure(relayURL)
		return nil, err
	}

	b.relayMu.Lock()
	if b.relayByURL == nil {
		b.relayByURL = make(map[string]*gnostr.Relay)
	}
	b.relayByURL[relayURL] = r
	if b.relayFailCount != nil {
		delete(b.relayFailCount, relayURL)
	}
	if b.relayNextTry != nil {
		delete(b.relayNextTry, relayURL)
	}
	b.relayMu.Unlock()
	return r, nil
}

func (b *Broadcaster) forgetRelayAfterFailure(relayURL string) {
	b.relayMu.Lock()
	defer b.relayMu.Unlock()
	if r := b.relayByURL[relayURL]; r != nil {
		_ = r.Close()
		delete(b.relayByURL, relayURL)
	}
	if b.relayFailCount == nil {
		b.relayFailCount = make(map[string]int)
	}
	n := b.relayFailCount[relayURL] + 1
	b.relayFailCount[relayURL] = n
	shift := min(n, 6)
	backoff := time.Duration(1<<shift) * time.Second
	const maxBackoff = 60 * time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	if b.relayNextTry == nil {
		b.relayNextTry = make(map[string]time.Time)
	}
	b.relayNextTry[relayURL] = time.Now().Add(backoff)
}

func (b *Broadcaster) markRelayOK(relayURL string) {
	b.relayMu.Lock()
	defer b.relayMu.Unlock()
	if b.relayFailCount != nil {
		delete(b.relayFailCount, relayURL)
	}
	if b.relayNextTry != nil {
		delete(b.relayNextTry, relayURL)
	}
}

func (b *Broadcaster) closeAllRelays() {
	b.relayMu.Lock()
	defer b.relayMu.Unlock()
	for u, r := range b.relayByURL {
		if r != nil {
			_ = r.Close()
		}
		delete(b.relayByURL, u)
	}
	b.relayFailCount = nil
	b.relayNextTry = nil
}

func trimID(id string) string {
	if len(id) <= 16 {
		return id
	}
	return id[:16]
}

// Run publishes immediately, then every interval until ctx is done.
func (b *Broadcaster) Run(ctx context.Context) error {
	defer b.closeAllRelays()

	b.log.Printf("mlnd nostr: broadcaster running, interval=%s relays=%d", b.interval, len(b.relays))

	ev, err := b.BuildMakerAdEvent(time.Now())
	if err != nil {
		return fmt.Errorf("build maker ad: %w", err)
	}
	b.Publish(ctx, *ev)

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			ev, err := b.BuildMakerAdEvent(time.Now())
			if err != nil {
				b.log.Printf("mlnd nostr: build event: %v", err)
				continue
			}
			b.Publish(ctx, *ev)
		}
	}
}
