package registry

import (
	"math/big"
	"testing"
)

func TestDecideMakerVerified_exitQueueRejects(t *testing.T) {
	t.Parallel()
	stake := big.NewInt(100)
	min := big.NewInt(10)
	ok, reason := decideMakerVerified(true, false, stake, min, big.NewInt(1))
	if ok || reason != "in exit queue" {
		t.Fatalf("got ok=%v reason=%q", ok, reason)
	}
}

func TestDecideMakerVerified_exitQueueZeroOk(t *testing.T) {
	t.Parallel()
	stake := big.NewInt(100)
	min := big.NewInt(10)
	ok, reason := decideMakerVerified(true, false, stake, min, big.NewInt(0))
	if !ok || reason != "ok" {
		t.Fatalf("got ok=%v reason=%q", ok, reason)
	}
}

func TestDecideMakerVerified_precedenceHashBeforeExit(t *testing.T) {
	t.Parallel()
	ok, reason := decideMakerVerified(false, false, big.NewInt(100), big.NewInt(10), big.NewInt(99))
	if ok || reason != "nostrKeyHash mismatch" {
		t.Fatalf("got ok=%v reason=%q", ok, reason)
	}
}

func TestDecideMakerVerified_precedenceExitBeforeFrozen(t *testing.T) {
	t.Parallel()
	ok, reason := decideMakerVerified(true, true, big.NewInt(100), big.NewInt(10), big.NewInt(1))
	if ok || reason != "in exit queue" {
		t.Fatalf("got ok=%v reason=%q want in exit queue", ok, reason)
	}
}

func TestDecideMakerVerified_belowMinStake(t *testing.T) {
	t.Parallel()
	ok, reason := decideMakerVerified(true, false, big.NewInt(5), big.NewInt(10), big.NewInt(0))
	if ok || reason != "below minStake" {
		t.Fatalf("got ok=%v reason=%q", ok, reason)
	}
}
