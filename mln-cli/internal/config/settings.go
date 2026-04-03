package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/scout"
)

// NetworkSettings holds the same fields as scout/pathfind environment variables (PHASE_10_TAKER_CLI.md).
type NetworkSettings struct {
	NostrRelays         []string `json:"nostrRelays"`
	LitvmChainID        string   `json:"litvmChainId"`
	LitvmHTTPURL        string   `json:"litvmHttpUrl"`
	RegistryAddr        string   `json:"registryAddr"`
	GrievanceCourtAddr  string   `json:"grievanceCourtAddr,omitempty"`
	ScoutTimeout        string   `json:"scoutTimeout,omitempty"` // duration string, e.g. "30s"
	DefaultSidecarURL   string   `json:"defaultSidecarUrl,omitempty"`
	ForgerHTTPTimeout   string   `json:"forgerHttpTimeout,omitempty"` // duration for sidecar POST context
}

// ScoutTimeoutDuration parses ScoutTimeout or returns default 30s.
func (s *NetworkSettings) ScoutTimeoutDuration() (time.Duration, error) {
	raw := strings.TrimSpace(s.ScoutTimeout)
	if raw == "" {
		return 30 * time.Second, nil
	}
	return time.ParseDuration(raw)
}

// ForgerContextTimeout returns duration for forger HTTP (default 10s).
func (s *NetworkSettings) ForgerContextTimeout() (time.Duration, error) {
	raw := strings.TrimSpace(s.ForgerHTTPTimeout)
	if raw == "" {
		return 10 * time.Second, nil
	}
	return time.ParseDuration(raw)
}

// Validate checks required network fields (mirrors ScoutFromEnv).
func (s *NetworkSettings) Validate() error {
	if len(s.NostrRelays) == 0 {
		return fmt.Errorf("nostr relays are required")
	}
	if strings.TrimSpace(s.LitvmChainID) == "" {
		return fmt.Errorf("litvm chain id is required")
	}
	if strings.TrimSpace(s.LitvmHTTPURL) == "" {
		return fmt.Errorf("litvm HTTP URL is required")
	}
	if strings.TrimSpace(s.RegistryAddr) == "" {
		return fmt.Errorf("registry address is required")
	}
	return nil
}

// ToScoutConfig maps settings to scout.Config.
func (s *NetworkSettings) ToScoutConfig() (scout.Config, error) {
	if err := s.Validate(); err != nil {
		return scout.Config{}, err
	}
	regAddr, err := ParseRegistryAddr(s.RegistryAddr)
	if err != nil {
		return scout.Config{}, fmt.Errorf("registry: %w", err)
	}
	timeout, err := s.ScoutTimeoutDuration()
	if err != nil {
		return scout.Config{}, fmt.Errorf("scout timeout: %w", err)
	}
	return scout.Config{
		Relays:         s.NostrRelays,
		RPCHTTP:        strings.TrimSpace(s.LitvmHTTPURL),
		ChainID:        strings.TrimSpace(s.LitvmChainID),
		RegistryAddr:   regAddr,
		GrievanceCourt: strings.TrimSpace(s.GrievanceCourtAddr),
		Timeout:        timeout,
	}, nil
}

// DefaultSidecar returns the sidecar URL for forger (setting or constant default).
func (s *NetworkSettings) DefaultSidecar() string {
	u := strings.TrimSpace(s.DefaultSidecarURL)
	if u != "" {
		return u
	}
	return "http://127.0.0.1:8080/v1/swap"
}

// ParseRelayList splits a comma-separated relay line into URLs (same rules as ScoutFromEnv).
func ParseRelayList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var relays []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			relays = append(relays, p)
		}
	}
	return relays
}

// NetworkSettingsFromEnv builds NetworkSettings from the same variables as ScoutFromEnv.
func NetworkSettingsFromEnv() (NetworkSettings, error) {
	relays, chainID, rpcURL, registry, court, timeout, err := ScoutFromEnv()
	if err != nil {
		return NetworkSettings{}, err
	}
	return NetworkSettings{
		NostrRelays:        relays,
		LitvmChainID:       chainID,
		LitvmHTTPURL:       rpcURL,
		RegistryAddr:       registry,
		GrievanceCourtAddr: court,
		ScoutTimeout:       timeout.String(),
	}, nil
}

// RegistryAddress parses RegistryAddr hex.
func (s *NetworkSettings) RegistryAddress() (common.Address, error) {
	return ParseRegistryAddr(s.RegistryAddr)
}
