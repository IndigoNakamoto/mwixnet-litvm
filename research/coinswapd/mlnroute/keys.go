package mlnroute

import (
	"crypto/ecdh"
	"encoding/hex"
	"fmt"
	"strings"
)

// NormalizeTor trims space and trailing slash for map lookups.
func NormalizeTor(tor string) string {
	s := strings.TrimSpace(tor)
	s = strings.TrimSuffix(s, "/")
	return s
}

// ResolveX25519PubKeys returns one 32-byte Curve25519 pubkey per hop (Approach A from payload, else Approach C map).
func ResolveX25519PubKeys(req *Request, pubkeyMap map[string]string) ([][]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	out := make([][]byte, len(req.Route))
	for i, h := range req.Route {
		hexKey := strings.TrimSpace(h.SwapX25519PubHex)
		if hexKey == "" && pubkeyMap != nil {
			hexKey = strings.TrimSpace(pubkeyMap[NormalizeTor(h.Tor)])
		}
		if hexKey == "" {
			return nil, fmt.Errorf("hop %d: missing swap key (swapX25519PubHex or pubkey map)", i)
		}
		if !swapX25519PubHexRE.MatchString(hexKey) {
			return nil, fmt.Errorf("hop %d: invalid swapX25519PubHex", i)
		}
		raw, err := hex.DecodeString(hexKey)
		if err != nil || len(raw) != 32 {
			return nil, fmt.Errorf("hop %d: invalid swap key bytes", i)
		}
		out[i] = raw
	}
	return out, nil
}

// ECDHPublicKeys parses raw keys into ecdh public keys.
func ECDHPublicKeys(raw [][]byte) ([]*ecdh.PublicKey, error) {
	out := make([]*ecdh.PublicKey, len(raw))
	for i, b := range raw {
		k, err := ecdh.X25519().NewPublicKey(b)
		if err != nil {
			return nil, fmt.Errorf("hop %d: %w", i, err)
		}
		out[i] = k
	}
	return out, nil
}
