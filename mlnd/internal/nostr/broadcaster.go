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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/opslog"
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/makerad"
	"github.com/ethereum/go-ethereum/common"
	gnostr "github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
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
	// SwapX25519PubHex is optional 64 lowercase hex digits (32-byte Curve25519 pubkey).
	SwapX25519PubHex string
	// AuthEnabled enables NIP-42 AUTH after relay connect. When true, broadcaster
	// will fail fast (backoff) if AUTH is rejected by the relay.
	AuthEnabled bool
}

// errRelayBackoff means this relay URL is in exponential backoff; skip until next tick.
var errRelayBackoff = errors.New("nostr relay in backoff")

// RelayPublishLine is one relay outcome from the last Publish round.
type RelayPublishLine struct {
	URL    string `json:"url"`
	Status string `json:"status"` // ok | skipped | error
	Detail string `json:"detail,omitempty"`
}

// LastPublishSnapshot is the most recent maker-ad publish attempt.
type LastPublishSnapshot struct {
	At     time.Time          `json:"at"`
	Relays []RelayPublishLine `json:"relays"`
}

// Broadcaster republishes replaceable maker ads on an interval.
type Broadcaster struct {
	cfg      BroadcasterConfig
	relays   []string
	secHex   string // 64-char hex private key for gnostr.Event.Sign
	pubHex   string // x-only public key hex (derived from secHex)
	interval time.Duration
	log      *log.Logger
	Ops      *opslog.Log

	relayMu        sync.Mutex
	relayByURL     map[string]*gnostr.Relay
	relayFailCount map[string]int
	relayNextTry   map[string]time.Time

	lastPubMu sync.RWMutex
	lastPub   LastPublishSnapshot
}

// NewBroadcaster returns a configured broadcaster (e.g. for tests). relays may be empty if only BuildMakerAdEvent is used.
func NewBroadcaster(cfg BroadcasterConfig, relays []string, secHex string, interval time.Duration, lg *log.Logger) *Broadcaster {
	if lg == nil {
		lg = log.Default()
	}
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	pubHex, _ := gnostr.GetPublicKey(secHex)
	return &Broadcaster{
		cfg:      cfg,
		relays:   relays,
		secHex:   secHex,
		pubHex:   pubHex,
		interval: interval,
		log:      lg,
	}
}

// LoadBroadcasterFromEnv returns nil if MLND_NOSTR_RELAYS is unset (broadcaster disabled).
// If relays are set but MLND_NOSTR_NSEC is empty, returns (nil, nil): no publishing, but the
// dashboard may still use the same relay list for read-only ad self-check.
func LoadBroadcasterFromEnv() (*Broadcaster, error) {
	relaysRaw := strings.TrimSpace(os.Getenv("MLND_NOSTR_RELAYS"))
	if relaysRaw == "" {
		return nil, nil
	}

	nsec := strings.TrimSpace(os.Getenv("MLND_NOSTR_NSEC"))
	if nsec == "" {
		log.Printf("mlnd: MLND_NOSTR_RELAYS is set but MLND_NOSTR_NSEC is empty — maker-ad broadcaster disabled (set MLND_NOSTR_NSEC to publish)")
		return nil, nil
	}

	secHex, err := parseSigningKey(nsec)
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
	swapHex := strings.TrimSpace(strings.ToLower(os.Getenv("MLND_SWAP_X25519_PUB_HEX")))
	if swapHex != "" {
		swapHex = strings.TrimPrefix(strings.TrimPrefix(swapHex, "0x"), "0X")
		if len(swapHex) != 64 {
			return nil, fmt.Errorf("MLND_SWAP_X25519_PUB_HEX: want 64 hex digits (optional 0x prefix), got length %d", len(swapHex))
		}
		for _, c := range swapHex {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return nil, fmt.Errorf("MLND_SWAP_X25519_PUB_HEX: non-hex character")
			}
		}
	}
	var authEnabled bool
	if s := strings.TrimSpace(os.Getenv("MLND_NOSTR_AUTH")); s != "" {
		v, err := strconv.ParseBool(s)
		if err != nil {
			return nil, fmt.Errorf("MLND_NOSTR_AUTH: %w", err)
		}
		authEnabled = v
	}

	cfg := BroadcasterConfig{
		ChainID:          chainID,
		Registry:         regNorm,
		GrievanceCourt:   courtNorm,
		Operator:         opNorm,
		TorOnion:         torNorm,
		FeeMinSat:        feeMin,
		FeeMaxSat:        feeMax,
		Capabilities:     []string{"mweb-coinswap-v0"},
		ClientName:       "mlnd",
		ClientVersion:    "0",
		SwapX25519PubHex: swapHex,
		AuthEnabled:      authEnabled,
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
	return makerad.DTag(chainID, operatorLower)
}

// BuildMakerAdEvent builds and signs a kind-31250 replaceable maker ad at now.
func (b *Broadcaster) BuildMakerAdEvent(now time.Time) (*gnostr.Event, error) {
	body := makerad.Content{
		V: 1,
		Litvm: makerad.LitVM{
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
		body.Fees = &makerad.Fees{
			Unit: "sat_per_hop",
			Min:  *b.cfg.FeeMinSat,
			Max:  *b.cfg.FeeMaxSat,
		}
	}
	if b.cfg.SwapX25519PubHex != "" {
		body.SwapX25519PubHex = b.cfg.SwapX25519PubHex
	}
	contentBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	d := DTag(b.cfg.ChainID, b.cfg.Operator)
	tags := gnostr.Tags{
		{"d", d},
		{"t", makerad.TagTMakerAd},
	}
	if b.cfg.ClientName != "" {
		tags = append(tags, gnostr.Tag{"client", b.cfg.ClientName, b.cfg.ClientVersion})

	}

	ev := &gnostr.Event{
		CreatedAt: gnostr.Timestamp(now.Unix()),
		Kind:      makerad.KindMakerAd,
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

// Config returns a copy of the static broadcaster configuration (for dashboard “intended ad”).
func (b *Broadcaster) Config() BroadcasterConfig {
	if b == nil {
		return BroadcasterConfig{}
	}
	return b.cfg
}

// AuthKeys returns the NIP-42 signing material when AUTH is enabled.
// Returns empty strings if AUTH is off or broadcaster is nil.
func (b *Broadcaster) AuthKeys() (secHex, pubHex string) {
	if b == nil || !b.cfg.AuthEnabled {
		return "", ""
	}
	return b.secHex, b.pubHex
}

// LastPublish returns the most recent publish round summary.
func (b *Broadcaster) LastPublish() LastPublishSnapshot {
	if b == nil {
		return LastPublishSnapshot{}
	}
	b.lastPubMu.RLock()
	defer b.lastPubMu.RUnlock()
	out := b.lastPub
	if len(out.Relays) > 0 {
		out.Relays = append([]RelayPublishLine(nil), out.Relays...)
	}
	return out
}

// Publish sends the event to each relay; errors are logged per relay only.
// Relays are reused across ticks; failed publishes trigger close, exponential backoff, and reconnect.
func (b *Broadcaster) Publish(ctx context.Context, ev gnostr.Event) {
	pubCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	lines := make([]RelayPublishLine, 0, len(b.relays))
	for _, relayURL := range b.relays {
		r, err := b.ensureRelay(pubCtx, relayURL)
		if errors.Is(err, errRelayBackoff) {
			lines = append(lines, RelayPublishLine{URL: relayURL, Status: "skipped", Detail: "relay backoff"})
			if b.Ops != nil {
				b.Ops.Append(opslog.Warn, "nostr_publish_skipped", "Relay in backoff; skipped publish", map[string]string{"relay": relayURL})
			}
			continue
		}
		if err != nil {
			b.log.Printf("mlnd nostr: connect %s: %v", relayURL, err)
			lines = append(lines, RelayPublishLine{URL: relayURL, Status: "error", Detail: err.Error()})
			if b.Ops != nil {
				b.Ops.Append(opslog.Warn, "nostr_connect_failed", "Could not connect to Nostr relay", map[string]string{"relay": relayURL, "detail": err.Error()})
			}
			continue
		}
		if err := r.Publish(pubCtx, ev); err != nil {
			b.log.Printf("mlnd nostr: publish %s: %v", relayURL, err)
			b.forgetRelayAfterFailure(relayURL)
			lines = append(lines, RelayPublishLine{URL: relayURL, Status: "error", Detail: err.Error()})
			if b.Ops != nil {
				b.Ops.Append(opslog.Warn, "nostr_publish_failed", "Publish to relay failed", map[string]string{"relay": relayURL, "detail": err.Error()})
			}
			continue
		}
		b.markRelayOK(relayURL)
		b.log.Printf("mlnd nostr: published kind=%d id=%s… to %s", ev.Kind, trimID(ev.ID), relayURL)
		lines = append(lines, RelayPublishLine{URL: relayURL, Status: "ok"})
		if b.Ops != nil {
			b.Ops.Append(opslog.Info, "nostr_published", "Maker ad published to relay", map[string]string{"relay": relayURL, "eventId": ev.ID})
		}
	}
	b.lastPubMu.Lock()
	b.lastPub = LastPublishSnapshot{At: time.Now().UTC(), Relays: lines}
	b.lastPubMu.Unlock()
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

	if b.cfg.AuthEnabled {
		secHex, pubHex := b.secHex, b.pubHex
		if err := r.Auth(connCtx, func(ev *gnostr.Event) error {
			ev.PubKey = pubHex
			return ev.Sign(secHex)
		}); err != nil {
			_ = r.Close()
			b.forgetRelayAfterFailure(relayURL)
			return nil, fmt.Errorf("NIP-42 AUTH rejected: %w", err)
		}
		b.log.Printf("mlnd nostr: AUTH OK on %s", relayURL)
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
