package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ScoutFromEnv loads scout-related settings (see PHASE_10_TAKER_CLI.md).
func ScoutFromEnv() (relays []string, chainID, rpcURL, registry string, court string, timeout time.Duration, err error) {
	raw := strings.TrimSpace(os.Getenv("MLN_NOSTR_RELAYS"))
	if raw == "" {
		return nil, "", "", "", "", 0, fmt.Errorf("MLN_NOSTR_RELAYS is required")
	}
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			relays = append(relays, p)
		}
	}
	chainID = strings.TrimSpace(os.Getenv("MLN_LITVM_CHAIN_ID"))
	if chainID == "" {
		return nil, "", "", "", "", 0, fmt.Errorf("MLN_LITVM_CHAIN_ID is required")
	}
	rpcURL = strings.TrimSpace(os.Getenv("MLN_LITVM_HTTP_URL"))
	registry = strings.TrimSpace(os.Getenv("MLN_REGISTRY_ADDR"))
	if registry == "" {
		return nil, "", "", "", "", 0, fmt.Errorf("MLN_REGISTRY_ADDR is required")
	}
	court = strings.TrimSpace(os.Getenv("MLN_GRIEVANCE_COURT_ADDR"))
	timeout = 30 * time.Second
	if s := strings.TrimSpace(os.Getenv("MLN_SCOUT_TIMEOUT")); s != "" {
		d, e := time.ParseDuration(s)
		if e != nil {
			return nil, "", "", "", "", 0, fmt.Errorf("MLN_SCOUT_TIMEOUT: %w", e)
		}
		timeout = d
	}
	return relays, chainID, rpcURL, registry, court, timeout, nil
}

// ParseRegistryAddr normalizes a 0x address.
func ParseRegistryAddr(s string) (common.Address, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		s = "0x" + s
	}
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("invalid registry address")
	}
	return common.HexToAddress(s), nil
}
