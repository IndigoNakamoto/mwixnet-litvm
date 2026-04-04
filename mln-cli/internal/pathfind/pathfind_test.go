package pathfind

import (
	"math/rand"
	"testing"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/scout"
	"github.com/ethereum/go-ethereum/common"
)

func TestPickRoute_minFee(t *testing.T) {
	makers := []scout.VerifiedMaker{
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000001"), FeeMinSat: 10, Stake: "100", Tor: "http://n1"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000002"), FeeMinSat: 1, Stake: "100", Tor: "http://n2"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000003"), FeeMinSat: 1, Stake: "100", Tor: "http://n3"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000004"), FeeMinSat: 99, Stake: "900", Tor: "http://n4"},
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
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000001"), Tor: "http://a"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000002"), Tor: "http://b"},
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPickRoute_tooFewTor(t *testing.T) {
	_, err := PickRoute([]scout.VerifiedMaker{
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000001"), FeeMinSat: 1, Stake: "1"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000002"), FeeMinSat: 1, Stake: "1"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000003"), FeeMinSat: 1, Stake: "1"},
	}, nil)
	if err == nil {
		t.Fatal("expected error: no Tor on makers")
	}
}

func TestPickRouteSelfMiddle_fixesN2(t *testing.T) {
	t.Parallel()
	self := common.HexToAddress("0x00000000000000000000000000000000000000b2")
	makers := []scout.VerifiedMaker{
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000001"), FeeMinSat: 5, Stake: "100", Tor: "http://n1"},
		{Operator: self, FeeMinSat: 1, Stake: "100", Tor: "http://self"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000003"), FeeMinSat: 1, Stake: "100", Tor: "http://n3"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000004"), FeeMinSat: 99, Stake: "900", Tor: "http://n4"},
	}
	rng := rand.New(rand.NewSource(1))
	route, err := PickRouteSelfMiddle(makers, self, rng)
	if err != nil {
		t.Fatal(err)
	}
	if route.Hops[1].Operator != self {
		t.Fatalf("N2 = %s want self %s", route.Hops[1].Operator.Hex(), self.Hex())
	}
	if route.Hops[0].Operator == route.Hops[2].Operator {
		t.Fatal("N1 and N3 must differ")
	}
	// Cheapest externals are fee 1+1 with self fee 1 → 7; other pairs are higher (e.g. 5+1+1).
	if route.FeeSumSat != 7 {
		t.Fatalf("fee sum %d want 7", route.FeeSumSat)
	}
}

func TestPickRouteSelfMiddle_selfMissing(t *testing.T) {
	t.Parallel()
	self := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	makers := []scout.VerifiedMaker{
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000001"), Tor: "http://a"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000002"), Tor: "http://b"},
		{Operator: common.HexToAddress("0x0000000000000000000000000000000000000003"), Tor: "http://c"},
	}
	_, err := PickRouteSelfMiddle(makers, self, rand.New(rand.NewSource(1)))
	if err == nil {
		t.Fatal("expected error")
	}
}
