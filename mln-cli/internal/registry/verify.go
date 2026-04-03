package registry

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/makerad"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Minimal ABI for MwixnetRegistry view functions used by the Scout.
const registryABIJSON = `[
  {"name":"makerNostrKeyHash","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"bytes32"}]},
  {"name":"stake","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"uint256"}]},
  {"name":"minStake","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256"}]},
  {"name":"stakeFrozen","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"bool"}]}
]`

var parsedABI abi.ABI

func init() {
	a, err := abi.JSON(strings.NewReader(registryABIJSON))
	if err != nil {
		panic(err)
	}
	parsedABI = a
}

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

	v := &Verification{
		NostrKeyHashMatch: match,
		Stake:             new(big.Int).Set(stake),
		MinStake:          new(big.Int).Set(minStake),
		Frozen:            frozen,
	}

	switch {
	case !match:
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
