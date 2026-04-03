package makerad

import (
	"fmt"

	gnostr "github.com/nbd-wtf/go-nostr"
)

// VerifySignature checks the Nostr event Schnorr signature (NIP-01).
func VerifySignature(ev *gnostr.Event) error {
	if ev == nil {
		return fmt.Errorf("makerad: nil event")
	}
	ok, err := ev.CheckSignature()
	if err != nil {
		return fmt.Errorf("makerad: signature check: %w", err)
	}
	if !ok {
		return fmt.Errorf("makerad: invalid signature")
	}
	return nil
}
