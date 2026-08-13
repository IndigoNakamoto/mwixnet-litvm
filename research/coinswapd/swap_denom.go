package main

import "fmt"

// onionAmt is the recorded mweb_submitRoute amount for one queued onion.
type onionAmt struct {
	value uint64
	ok    bool
}

// requireSameDenomination enforces v1: one denomination per performSwap / aggregated tx.
// Vanilla swap_Swap onions have no recorded amount; if none are recorded, the gate is skipped.
// If any amount is recorded, every onion must share that exact value (mixed 1.0 + 5.0 is rejected).
func requireSameDenomination(amts []onionAmt) error {
	recorded := 0
	for _, a := range amts {
		if a.ok {
			recorded++
		}
	}
	if recorded == 0 {
		return nil
	}
	if recorded != len(amts) {
		return fmt.Errorf("performSwap: not all onions have a recorded amount; refuse mixed batch")
	}
	first := amts[0].value
	for _, a := range amts[1:] {
		if a.value != first {
			return fmt.Errorf("performSwap: mixed denominations %d and %d sat; one value per aggregated tx", first, a.value)
		}
	}
	return nil
}
