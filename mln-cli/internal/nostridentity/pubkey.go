package nostridentity

import (
	"fmt"
	"os"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

// PubkeyHexFromEnv returns 64-char lowercase hex x-only secp256k1 pubkey.
// Prefer MLN_NOSTR_PUBKEY_HEX when set; otherwise MLN_NOSTR_NSEC (hex or nip19 nsec1…).
func PubkeyHexFromEnv() (string, error) {
	if p := strings.TrimSpace(os.Getenv("MLN_NOSTR_PUBKEY_HEX")); p != "" {
		return parsePublicKeyInput(p)
	}
	nsec := strings.TrimSpace(os.Getenv("MLN_NOSTR_NSEC"))
	if nsec == "" {
		return "", fmt.Errorf("set MLN_NOSTR_PUBKEY_HEX or MLN_NOSTR_NSEC")
	}
	skHex, err := parseSecretKeyHex(nsec)
	if err != nil {
		return "", err
	}
	pk, err := nostr.GetPublicKey(skHex)
	if err != nil {
		return "", fmt.Errorf("nostr pubkey from secret: %w", err)
	}
	return normalizePubkeyHex(pk)
}

// parsePublicKeyInput accepts 64-char hex (optional 0x) or nip19 npub1….
func parsePublicKeyInput(s string) (string, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "npub1") {
		prefix, val, err := nip19.Decode(s)
		if err != nil {
			return "", fmt.Errorf("MLN_NOSTR_PUBKEY_HEX: decode npub: %w", err)
		}
		if prefix != "npub" {
			return "", fmt.Errorf("MLN_NOSTR_PUBKEY_HEX: expected npub, got %q", prefix)
		}
		pk, ok := val.(string)
		if !ok || len(pk) != 64 {
			return "", fmt.Errorf("MLN_NOSTR_PUBKEY_HEX: invalid npub payload")
		}
		return normalizePubkeyHex(pk)
	}
	return normalizePubkeyHex(s)
}

func normalizePubkeyHex(p string) (string, error) {
	p = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(p, "0x"), "0X"))
	if len(p) != 64 {
		if strings.HasPrefix(strings.ToLower(p), "npub1") {
			return "", fmt.Errorf("nostr pubkey: MLN_NOSTR_PUBKEY_HEX is an npub (nip19), not hex — rebuild mln-cli from this repo (`make build-mln-cli`) or use 64-char hex / MLN_NOSTR_NSEC")
		}
		return "", fmt.Errorf("nostr pubkey: want 64 hex chars, got %d", len(p))
	}
	for _, c := range p {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", fmt.Errorf("nostr pubkey: invalid hex")
		}
	}
	if !nostr.IsValidPublicKey(strings.ToLower(p)) {
		return "", fmt.Errorf("nostr pubkey: not a valid schnorr public key")
	}
	return strings.ToLower(p), nil
}

func parseSecretKeyHex(s string) (string, error) {
	if strings.HasPrefix(s, "nsec1") {
		prefix, val, err := nip19.Decode(s)
		if err != nil {
			return "", fmt.Errorf("MLN_NOSTR_NSEC: %w", err)
		}
		if prefix != "nsec" {
			return "", fmt.Errorf("MLN_NOSTR_NSEC: expected nsec, got %q", prefix)
		}
		sk, ok := val.(string)
		if !ok || len(sk) != 64 {
			return "", fmt.Errorf("MLN_NOSTR_NSEC: invalid decode")
		}
		return strings.ToLower(sk), nil
	}
	key := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(s), "0x"), "0X")
	if len(key) != 64 {
		return "", fmt.Errorf("MLN_NOSTR_NSEC: want 64 hex chars or nsec1…")
	}
	for _, c := range key {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", fmt.Errorf("MLN_NOSTR_NSEC: invalid hex")
		}
	}
	return strings.ToLower(key), nil
}
