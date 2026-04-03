// Package pathfind selects an ordered 3-hop maker cascade (wallet PoC policy).
// See research/USER_STORIES_MLN.md — scoring is implementation-defined, not on-chain protocol.
package pathfind

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/scout"
	"github.com/ethereum/go-ethereum/common"
)

// Route is an ordered N1 → N2 → N3 list of verified makers.
type Route struct {
	Hops [3]scout.VerifiedMaker `json:"hops"`
	// FeeSumSat is the sum of per-hop fee hints (min sat_per_hop); 0 if a hop omitted fees.
	FeeSumSat uint64 `json:"feeSumSat"`
}

// PickRoute chooses distinct makers minimizing total fee hint (FeeMinSat), with random tie-break.
// Makers must have at least three entries; Tor is not required here (forger checks Tor).
func PickRoute(makers []scout.VerifiedMaker, rng *rand.Rand) (*Route, error) {
	if len(makers) < 3 {
		return nil, fmt.Errorf("pathfind: need at least 3 verified makers, got %d", len(makers))
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	n := len(makers)
	type pick struct {
		i, j, k int
		fee     uint64
		stake   *big.Int // sum of stake for tie-break (higher preferred)
	}
	bestFee := ^uint64(0) >> 1 // large
	var candidates []pick

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if j == i {
				continue
			}
			for k := 0; k < n; k++ {
				if k == i || k == j {
					continue
				}
				fee := makers[i].FeeMinSat + makers[j].FeeMinSat + makers[k].FeeMinSat
				si := strToStake(makers[i].Stake)
				sj := strToStake(makers[j].Stake)
				sk := strToStake(makers[k].Stake)
				sumStake := new(big.Int).Add(si, sj)
				sumStake.Add(sumStake, sk)
				if fee < bestFee {
					bestFee = fee
					candidates = nil
				}
				if fee != bestFee {
					continue
				}
				candidates = append(candidates, pick{i, j, k, fee, sumStake})
			}
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("pathfind: no route candidates")
	}

	// Among min fee, prefer larger total stake; then random tie-break.
	maxStake := candidates[0].stake
	for _, c := range candidates[1:] {
		if c.stake.Cmp(maxStake) > 0 {
			maxStake = c.stake
		}
	}
	var tier []pick
	for _, c := range candidates {
		if c.stake.Cmp(maxStake) == 0 {
			tier = append(tier, c)
		}
	}
	ch := tier[rng.Intn(len(tier))]
	return &Route{
		Hops:      [3]scout.VerifiedMaker{makers[ch.i], makers[ch.j], makers[ch.k]},
		FeeSumSat: ch.fee,
	}, nil
}

// PickRouteSelfMiddle builds N1 → N2(self) → N3 with the same fee/stake tie-break as PickRoute over valid triples.
func PickRouteSelfMiddle(makers []scout.VerifiedMaker, self common.Address, rng *rand.Rand) (*Route, error) {
	if len(makers) < 3 {
		return nil, fmt.Errorf("pathfind: need at least 3 verified makers for self-route, got %d", len(makers))
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	selfIdx := -1
	for i := range makers {
		if makers[i].Operator == self {
			selfIdx = i
			break
		}
	}
	if selfIdx < 0 {
		return nil, fmt.Errorf("pathfind: self operator %s not in verified maker set", self.Hex())
	}
	var ext []int
	for i := range makers {
		if i != selfIdx {
			ext = append(ext, i)
		}
	}
	if len(ext) < 2 {
		return nil, fmt.Errorf("pathfind: need at least 2 external makers besides self")
	}

	type pick struct {
		n1, n3 int
		fee    uint64
		stake  *big.Int
	}
	bestFee := ^uint64(0) >> 1
	var candidates []pick

	for _, i := range ext {
		for _, k := range ext {
			if i == k {
				continue
			}
			fee := makers[i].FeeMinSat + makers[selfIdx].FeeMinSat + makers[k].FeeMinSat
			si := strToStake(makers[i].Stake)
			sj := strToStake(makers[selfIdx].Stake)
			sk := strToStake(makers[k].Stake)
			sumStake := new(big.Int).Add(si, sj)
			sumStake.Add(sumStake, sk)
			if fee < bestFee {
				bestFee = fee
				candidates = nil
			}
			if fee != bestFee {
				continue
			}
			candidates = append(candidates, pick{i, k, fee, sumStake})
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("pathfind: no self-route candidates")
	}
	maxStake := candidates[0].stake
	for _, c := range candidates[1:] {
		if c.stake.Cmp(maxStake) > 0 {
			maxStake = c.stake
		}
	}
	var tier []pick
	for _, c := range candidates {
		if c.stake.Cmp(maxStake) == 0 {
			tier = append(tier, c)
		}
	}
	ch := tier[rng.Intn(len(tier))]
	return &Route{
		Hops:      [3]scout.VerifiedMaker{makers[ch.n1], makers[selfIdx], makers[ch.n3]},
		FeeSumSat: ch.fee,
	}, nil
}

func strToStake(s string) *big.Int {
	x, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return big.NewInt(0)
	}
	return x
}
