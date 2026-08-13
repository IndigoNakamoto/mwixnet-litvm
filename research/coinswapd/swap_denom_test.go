package main

import "testing"

func TestRequireSameDenomination_okSame(t *testing.T) {
	t.Parallel()
	amts := []onionAmt{{value: 10_000_000, ok: true}, {value: 10_000_000, ok: true}}
	if err := requireSameDenomination(amts); err != nil {
		t.Fatal(err)
	}
}

func TestRequireSameDenomination_rejectMixed(t *testing.T) {
	t.Parallel()
	amts := []onionAmt{{value: 100_000_000, ok: true}, {value: 500_000_000, ok: true}}
	if err := requireSameDenomination(amts); err == nil {
		t.Fatal("expected mixed-denom error")
	}
}

func TestRequireSameDenomination_vanillaOnlySkipped(t *testing.T) {
	t.Parallel()
	amts := []onionAmt{{}, {}}
	if err := requireSameDenomination(amts); err != nil {
		t.Fatal(err)
	}
}

func TestRequireSameDenomination_partialRecordedRejected(t *testing.T) {
	t.Parallel()
	amts := []onionAmt{{value: 10_000_000, ok: true}, {}}
	if err := requireSameDenomination(amts); err == nil {
		t.Fatal("expected error")
	}
}

func TestRequireSameDenomination_empty(t *testing.T) {
	t.Parallel()
	if err := requireSameDenomination(nil); err != nil {
		t.Fatal(err)
	}
}
