package config

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// OnboardEnv holds LitVM RPC, registry, chain id, and operator key hex for `maker onboard` (no Nostr relays).
type OnboardEnv struct {
	RPCHTTP       string
	Registry      common.Address
	ChainID       *big.Int
	PrivateKeyHex string // 64 lowercase hex (no 0x), for signing txs
}

// OnboardFromEnv loads MLN_LITVM_HTTP_URL, MLN_REGISTRY_ADDR, MLN_LITVM_CHAIN_ID, MLN_OPERATOR_ETH_KEY.
func OnboardFromEnv() (OnboardEnv, error) {
	rpcURL := strings.TrimSpace(os.Getenv("MLN_LITVM_HTTP_URL"))
	if rpcURL == "" {
		return OnboardEnv{}, fmt.Errorf("MLN_LITVM_HTTP_URL is required")
	}
	regStr := strings.TrimSpace(os.Getenv("MLN_REGISTRY_ADDR"))
	if regStr == "" {
		return OnboardEnv{}, fmt.Errorf("MLN_REGISTRY_ADDR is required")
	}
	chainStr := strings.TrimSpace(os.Getenv("MLN_LITVM_CHAIN_ID"))
	if chainStr == "" {
		return OnboardEnv{}, fmt.Errorf("MLN_LITVM_CHAIN_ID is required")
	}
	keyHex := strings.TrimSpace(os.Getenv("MLN_OPERATOR_ETH_KEY"))
	keyHex = strings.TrimPrefix(strings.TrimPrefix(keyHex, "0x"), "0X")
	if keyHex == "" {
		return OnboardEnv{}, fmt.Errorf("MLN_OPERATOR_ETH_KEY is required")
	}
	if len(keyHex) != 64 {
		return OnboardEnv{}, fmt.Errorf("MLN_OPERATOR_ETH_KEY: want 64 hex chars (optional 0x prefix)")
	}
	for _, c := range keyHex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return OnboardEnv{}, fmt.Errorf("MLN_OPERATOR_ETH_KEY: invalid hex")
		}
	}
	cid := new(big.Int)
	if _, ok := cid.SetString(chainStr, 10); !ok || cid.Sign() <= 0 {
		return OnboardEnv{}, fmt.Errorf("MLN_LITVM_CHAIN_ID: invalid positive decimal %q", chainStr)
	}
	reg, err := ParseRegistryAddr(regStr)
	if err != nil {
		return OnboardEnv{}, err
	}
	return OnboardEnv{
		RPCHTTP:       rpcURL,
		Registry:      reg,
		ChainID:       cid,
		PrivateKeyHex: strings.ToLower(keyHex),
	}, nil
}

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

// LitvmHTTPURLFromEnv returns MLN_LITVM_HTTP_URL (for commands that do not need Nostr/scout).
func LitvmHTTPURLFromEnv() (string, error) {
	rpcURL := strings.TrimSpace(os.Getenv("MLN_LITVM_HTTP_URL"))
	if rpcURL == "" {
		return "", fmt.Errorf("MLN_LITVM_HTTP_URL is required")
	}
	return rpcURL, nil
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
