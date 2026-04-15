package makerad

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	gnostr "github.com/nbd-wtf/go-nostr"
)

func validateV2Content(c *Content) error {
	if c.Reachability != nil {
		if strings.TrimSpace(c.Tor) != "" || strings.TrimSpace(c.SwapX25519PubHex) != "" {
			return fmt.Errorf("makerad: v2 sealed reachability forbids cleartext tor/swapX25519PubHex")
		}
		if c.Reachability.Scheme != "nip44-v2" {
			return fmt.Errorf("makerad: unsupported reachability.scheme %q", c.Reachability.Scheme)
		}
		if len(strings.TrimSpace(c.Reachability.Ciphertext)) < 8 {
			return fmt.Errorf("makerad: reachability.ciphertext too short")
		}
		return nil
	}
	if strings.TrimSpace(c.Tor) == "" && strings.TrimSpace(c.SwapX25519PubHex) == "" {
		return fmt.Errorf("makerad: v2 requires reachability or non-empty tor/swapX25519PubHex")
	}
	return nil
}

// DTag returns the NIP-33 d tag mln:v1:<chainId>:<operatorLower>.
func DTag(chainID, operatorLower string) string {
	return fmt.Sprintf("mln:v1:%s:%s", chainID, operatorLower)
}

// ParseDTag parses mln:v1:<decimal_chain_id>:<0x_address>.
func ParseDTag(d string) (chainID string, operator common.Address, err error) {
	d = strings.TrimSpace(d)
	parts := strings.Split(d, ":")
	if len(parts) != 4 || parts[0] != "mln" || parts[1] != "v1" {
		return "", common.Address{}, fmt.Errorf("makerad: invalid d-tag format")
	}
	chainID = strings.TrimSpace(parts[2])
	if chainID == "" {
		return "", common.Address{}, fmt.Errorf("makerad: empty chainId in d-tag")
	}
	addrStr := strings.TrimSpace(parts[3])
	if !strings.HasPrefix(addrStr, "0x") && !strings.HasPrefix(addrStr, "0X") {
		addrStr = "0x" + addrStr
	}
	if !common.IsHexAddress(addrStr) {
		return "", common.Address{}, fmt.Errorf("makerad: invalid operator address in d-tag")
	}
	return chainID, common.HexToAddress(addrStr), nil
}

// ParsedAd is a validated maker ad payload plus the operator from the d tag.
type ParsedAd struct {
	Content *Content
	// Operator is the maker LitVM address from the d tag (canonicalized checksummed form from HexToAddress).
	Operator common.Address
	// DTagChainID is the chain id segment from the d tag (decimal string).
	DTagChainID string
}

// ParseAd checks kind, required tags, unmarshals content, and ensures d-tag chain id matches content.litvm.chainId.
func ParseAd(ev *gnostr.Event) (*ParsedAd, error) {
	if ev == nil {
		return nil, fmt.Errorf("makerad: nil event")
	}
	if ev.Kind != KindMakerAd {
		return nil, fmt.Errorf("makerad: wrong event kind %d", ev.Kind)
	}

	var hasT bool
	var dVal string
	for _, t := range ev.Tags {
		if len(t) < 2 {
			continue
		}
		switch t[0] {
		case "t":
			if t[1] == TagTMakerAd {
				hasT = true
			}
		case "d":
			dVal = t[1]
		}
	}
	if !hasT {
		return nil, fmt.Errorf("makerad: missing #t tag %q", TagTMakerAd)
	}
	if strings.TrimSpace(dVal) == "" {
		return nil, fmt.Errorf("makerad: missing d tag")
	}

	dChain, operator, err := ParseDTag(dVal)
	if err != nil {
		return nil, err
	}

	var content Content
	if err := json.Unmarshal([]byte(ev.Content), &content); err != nil {
		return nil, fmt.Errorf("makerad: content JSON: %w", err)
	}
	switch content.V {
	case 1:
		// ok
	case 2:
		if err := validateV2Content(&content); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("makerad: unsupported content version %d", content.V)
	}

	if strings.TrimSpace(content.Litvm.ChainID) != dChain {
		return nil, fmt.Errorf("makerad: litvm.chainId %q does not match d-tag chainId %q", content.Litvm.ChainID, dChain)
	}

	return &ParsedAd{
		Content:     &content,
		Operator:    operator,
		DTagChainID: dChain,
	}, nil
}
