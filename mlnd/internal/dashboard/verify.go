package dashboard

import (
	"context"
	"fmt"
	"math/big"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/makerad"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// VerifyMakerOnChain checks registry binding and stake rules (same semantics as mln-cli scout).
func VerifyMakerOnChain(ctx context.Context, client *ethclient.Client, registryAddr, maker common.Address, nostrPubKeyHex string) (v struct {
	NostrKeyHashMatch bool
	OK                bool
	Reason            string
	OnChainHash       common.Hash
}, err error) {
	if client == nil {
		return v, fmt.Errorf("nil client")
	}
	wantHash, err := makerad.ComputeNostrKeyHash(nostrPubKeyHex)
	if err != nil {
		return v, err
	}

	data, err := regABI.Pack("makerNostrKeyHash", maker)
	if err != nil {
		return v, err
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return v, fmt.Errorf("makerNostrKeyHash: %w", err)
	}
	vals, err := regABI.Unpack("makerNostrKeyHash", out)
	if err != nil || len(vals) != 1 {
		return v, fmt.Errorf("unpack makerNostrKeyHash")
	}
	var onChain common.Hash
	switch x := vals[0].(type) {
	case [32]byte:
		onChain = x
	case []byte:
		if len(x) != 32 {
			return v, fmt.Errorf("makerNostrKeyHash: wrong bytes len %d", len(x))
		}
		onChain = common.BytesToHash(x)
	default:
		return v, fmt.Errorf("makerNostrKeyHash: unexpected type %T", vals[0])
	}
	v.OnChainHash = onChain
	v.NostrKeyHashMatch = onChain == wantHash

	data, err = regABI.Pack("stake", maker)
	if err != nil {
		return v, err
	}
	out, err = client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return v, fmt.Errorf("stake: %w", err)
	}
	vals, err = regABI.Unpack("stake", out)
	if err != nil || len(vals) != 1 {
		return v, fmt.Errorf("unpack stake")
	}
	stake := vals[0].(*big.Int)

	data, err = regABI.Pack("minStake")
	if err != nil {
		return v, err
	}
	out, err = client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return v, fmt.Errorf("minStake: %w", err)
	}
	vals, err = regABI.Unpack("minStake", out)
	if err != nil || len(vals) != 1 {
		return v, fmt.Errorf("unpack minStake")
	}
	minStake := vals[0].(*big.Int)

	data, err = regABI.Pack("stakeFrozen", maker)
	if err != nil {
		return v, err
	}
	out, err = client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return v, fmt.Errorf("stakeFrozen: %w", err)
	}
	vals, err = regABI.Unpack("stakeFrozen", out)
	if err != nil || len(vals) != 1 {
		return v, fmt.Errorf("unpack stakeFrozen")
	}
	frozen := vals[0].(bool)

	switch {
	case !v.NostrKeyHashMatch:
		v.Reason = "nostrKeyHash mismatch"
	case frozen:
		v.Reason = "stake frozen"
	case stake.Cmp(minStake) < 0:
		v.Reason = "below minStake"
	default:
		v.OK = true
		v.Reason = "ok"
	}
	return v, nil
}
