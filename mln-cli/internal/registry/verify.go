package registry

import (
	"context"
	"fmt"
	"math/big"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/makerad"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Verification is the outcome of on-chain checks for one maker.
type Verification struct {
	NostrKeyHashMatch bool
	Stake             *big.Int
	MinStake          *big.Int
	Frozen            bool
	OK                bool
	Reason            string
}

// VerifyMaker checks registry binding and stake rules for the maker advertising with pubkeyHex.
func VerifyMaker(ctx context.Context, rpcURL string, registryAddr common.Address, maker common.Address, pubkeyHex string) (*Verification, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("eth client: %w", err)
	}

	wantHash, err := makerad.ComputeNostrKeyHash(pubkeyHex)
	if err != nil {
		return nil, err
	}

	data, err := parsedABI.Pack("makerNostrKeyHash", maker)
	if err != nil {
		return nil, err
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("makerNostrKeyHash: %w", err)
	}
	vals, err := parsedABI.Unpack("makerNostrKeyHash", out)
	if err != nil || len(vals) != 1 {
		return nil, fmt.Errorf("unpack makerNostrKeyHash")
	}
	var onChain common.Hash
	switch x := vals[0].(type) {
	case [32]byte:
		onChain = x
	case []byte:
		if len(x) != 32 {
			return nil, fmt.Errorf("makerNostrKeyHash: wrong bytes len %d", len(x))
		}
		onChain = common.BytesToHash(x)
	default:
		return nil, fmt.Errorf("makerNostrKeyHash: unexpected type %T", vals[0])
	}
	match := onChain == wantHash

	data, err = parsedABI.Pack("stake", maker)
	if err != nil {
		return nil, err
	}
	out, err = client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("stake: %w", err)
	}
	vals, err = parsedABI.Unpack("stake", out)
	if err != nil || len(vals) != 1 {
		return nil, fmt.Errorf("unpack stake")
	}
	stake := vals[0].(*big.Int)

	data, err = parsedABI.Pack("minStake")
	if err != nil {
		return nil, err
	}
	out, err = client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("minStake: %w", err)
	}
	vals, err = parsedABI.Unpack("minStake", out)
	if err != nil || len(vals) != 1 {
		return nil, fmt.Errorf("unpack minStake")
	}
	minStake := vals[0].(*big.Int)

	data, err = parsedABI.Pack("stakeFrozen", maker)
	if err != nil {
		return nil, err
	}
	out, err = client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("stakeFrozen: %w", err)
	}
	vals, err = parsedABI.Unpack("stakeFrozen", out)
	if err != nil || len(vals) != 1 {
		return nil, fmt.Errorf("unpack stakeFrozen")
	}
	frozen := vals[0].(bool)

	data, err = parsedABI.Pack("exitUnlockTime", maker)
	if err != nil {
		return nil, err
	}
	out, err = client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("exitUnlockTime: %w", err)
	}
	vals, err = parsedABI.Unpack("exitUnlockTime", out)
	if err != nil || len(vals) != 1 {
		return nil, fmt.Errorf("unpack exitUnlockTime")
	}
	exitUnlock := vals[0].(*big.Int)

	v := &Verification{
		NostrKeyHashMatch: match,
		Stake:             new(big.Int).Set(stake),
		MinStake:          new(big.Int).Set(minStake),
		Frozen:            frozen,
	}

	ok, reason := decideMakerVerified(match, frozen, stake, minStake, exitUnlock)
	v.OK = ok
	v.Reason = reason
	return v, nil
}

// decideMakerVerified encodes scout routing policy after on-chain reads (also unit-tested).
func decideMakerVerified(match, frozen bool, stake, minStake, exitUnlock *big.Int) (ok bool, reason string) {
	switch {
	case !match:
		return false, "nostrKeyHash mismatch"
	case exitUnlock != nil && exitUnlock.Sign() != 0:
		return false, "in exit queue"
	case frozen:
		return false, "stake frozen"
	case stake.Cmp(minStake) < 0:
		return false, "below minStake"
	default:
		return true, "ok"
	}
}
