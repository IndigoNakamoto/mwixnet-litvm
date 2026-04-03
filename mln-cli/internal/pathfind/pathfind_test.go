package pathfind

import (
	"math/rand"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/scout"
	"github.com/ethereum/go-ethereum/common"
)

func TestPickRoute_minFee(t *testing.T) {
	makers := []scout.VerifiedMaker{
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000001"), FeeMinSat: 10, Stake: "100"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000002"), FeeMinSat: 1, Stake: "100"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000003"), FeeMinSat: 1, Stake: "100"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000004"), FeeMinSat: 99, Stake: "900"},
	}
	rng := rand.New(rand.NewSource(42))
	route, err := PickRoute(makers, rng)
	if err != nil {
		t.Fatal(err)
	}
	if route.FeeSumSat != 12 {
		t.Fatalf("fee sum %d want 12", route.FeeSumSat)
	}
	seen := make(map[common.Address]struct{})
	for _, h := range route.Hops {
		seen[h.Operator] = struct{}{}
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 distinct hops, got %d", len(seen))
	}
}

func TestPickRoute_tooFew(t *testing.T) {
	_, err := PickRoute([]scout.VerifiedMaker{
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000001")},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000002")},
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
