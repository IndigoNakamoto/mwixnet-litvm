package registry

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/makerad"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// MakerRegistryState is on-chain view used for onboard planning.
type MakerRegistryState struct {
	Stake             *big.Int
	MinStake          *big.Int
	MakerNostrKeyHash common.Hash
	StakeFrozen       bool
	ExitUnlockTime    *big.Int // unix seconds; 0 if not in exit queue
}

// ReadMakerState returns registry fields for maker.
func ReadMakerState(ctx context.Context, rpcURL string, registryAddr, maker common.Address) (*MakerRegistryState, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("eth client: %w", err)
	}
	defer client.Close()

	st := &MakerRegistryState{Stake: new(big.Int), MinStake: new(big.Int), ExitUnlockTime: new(big.Int)}

	data, err := parsedABI.Pack("stake", maker)
	if err != nil {
		return nil, err
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("stake: %w", err)
	}
	vals, err := parsedABI.Unpack("stake", out)
	if err != nil || len(vals) != 1 {
		return nil, fmt.Errorf("unpack stake")
	}
	st.Stake = new(big.Int).Set(vals[0].(*big.Int))

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
	st.MinStake = new(big.Int).Set(vals[0].(*big.Int))

	data, err = parsedABI.Pack("makerNostrKeyHash", maker)
	if err != nil {
		return nil, err
	}
	out, err = client.CallContract(ctx, ethereum.CallMsg{To: &registryAddr, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("makerNostrKeyHash: %w", err)
	}
	vals, err = parsedABI.Unpack("makerNostrKeyHash", out)
	if err != nil || len(vals) != 1 {
		return nil, fmt.Errorf("unpack makerNostrKeyHash")
	}
	switch x := vals[0].(type) {
	case [32]byte:
		st.MakerNostrKeyHash = x
	case []byte:
		if len(x) != 32 {
			return nil, fmt.Errorf("makerNostrKeyHash: wrong len")
		}
		st.MakerNostrKeyHash = common.BytesToHash(x)
	default:
		return nil, fmt.Errorf("makerNostrKeyHash: unexpected type %T", vals[0])
	}

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
	st.StakeFrozen = vals[0].(bool)

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
	st.ExitUnlockTime = new(big.Int).Set(vals[0].(*big.Int))

	return st, nil
}

// OnboardOpts configures maker onboard.
type OnboardOpts struct {
	RPCHTTP       string
	Registry      common.Address
	ChainID       *big.Int
	PrivateKeyHex string // 64 hex
	NostrPubHex   string // 64 hex x-only pubkey
	Execute       bool   // broadcast txs
	ForceReregister bool // allow registerMaker when on-chain hash differs from derived
	Out           io.Writer
}

// RunOnboard plans or executes deposit + registerMaker. Default is dry-run (prints plan).
func RunOnboard(ctx context.Context, o OnboardOpts) error {
	out := o.Out
	if out == nil {
		out = io.Discard
	}

	wantHash, err := makerad.ComputeNostrKeyHash(o.NostrPubHex)
	if err != nil {
		return err
	}

	key, err := crypto.HexToECDSA(o.PrivateKeyHex)
	if err != nil {
		return fmt.Errorf("operator key: %w", err)
	}
	from := crypto.PubkeyToAddress(key.PublicKey)

	st, err := ReadMakerState(ctx, o.RPCHTTP, o.Registry, from)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "operator=%s\n", from.Hex())
	fmt.Fprintf(out, "registry=%s\n", o.Registry.Hex())
	fmt.Fprintf(out, "chainId=%s\n", o.ChainID.String())
	fmt.Fprintf(out, "stake=%s wei\n", st.Stake.String())
	fmt.Fprintf(out, "minStake=%s wei\n", st.MinStake.String())
	fmt.Fprintf(out, "makerNostrKeyHash(on-chain)=%s\n", st.MakerNostrKeyHash.Hex())
	fmt.Fprintf(out, "nostrKeyHash(want)=%s\n", wantHash.Hex())
	fmt.Fprintf(out, "stakeFrozen=%v\n", st.StakeFrozen)
	fmt.Fprintf(out, "exitUnlockTime=%s\n", st.ExitUnlockTime.String())

	if st.ExitUnlockTime.Sign() > 0 {
		return fmt.Errorf("in exit queue (exitUnlockTime=%s); cannot registerMaker until cleared", st.ExitUnlockTime.String())
	}
	if st.StakeFrozen {
		return fmt.Errorf("stake is frozen; cannot registerMaker")
	}

	already := st.MakerNostrKeyHash == wantHash && st.Stake.Cmp(st.MinStake) >= 0
	if already {
		fmt.Fprintln(out, "\nAlready onboarded: stake >= minStake and nostrKeyHash matches.")
		return nil
	}

	onChainSet := st.MakerNostrKeyHash != (common.Hash{})
	if onChainSet && st.MakerNostrKeyHash != wantHash && !o.ForceReregister {
		return fmt.Errorf("on-chain makerNostrKeyHash differs from derived hash; re-register would overwrite — pass -force-reregister if intentional")
	}

	var depositWei *big.Int
	if st.Stake.Cmp(st.MinStake) < 0 {
		depositWei = new(big.Int).Sub(st.MinStake, st.Stake)
		fmt.Fprintf(out, "\nPlan: deposit %s wei then registerMaker(%s)\n", depositWei.String(), wantHash.Hex())
	} else {
		depositWei = big.NewInt(0)
		fmt.Fprintf(out, "\nPlan: registerMaker(%s) only (stake already >= minStake)\n", wantHash.Hex())
	}

	if !o.Execute {
		fmt.Fprintln(out, "\nDry-run only (no txs). Set -execute to broadcast.")
		return nil
	}

	client, err := ethclient.DialContext(ctx, o.RPCHTTP)
	if err != nil {
		return fmt.Errorf("eth client: %w", err)
	}
	defer client.Close()

	if err := ensureBalance(ctx, client, from, depositWei); err != nil {
		return err
	}

	if depositWei.Sign() > 0 {
		depositTx, err := sendDeposit(ctx, client, o.ChainID, o.Registry, key, depositWei)
		if err != nil {
			return fmt.Errorf("deposit: %w", err)
		}
		fmt.Fprintf(out, "deposit tx %s\n", depositTx.Hash().Hex())
		if _, err := bind.WaitMined(ctx, client, depositTx); err != nil {
			return fmt.Errorf("deposit wait: %w", err)
		}
	}

	regTx, err := sendRegisterMaker(ctx, client, o.ChainID, o.Registry, key, wantHash)
	if err != nil {
		return fmt.Errorf("registerMaker: %w", err)
	}
	fmt.Fprintf(out, "registerMaker tx %s\n", regTx.Hash().Hex())
	if _, err := bind.WaitMined(ctx, client, regTx); err != nil {
		return fmt.Errorf("registerMaker wait: %w", err)
	}

	st2, err := ReadMakerState(ctx, o.RPCHTTP, o.Registry, from)
	if err != nil {
		return fmt.Errorf("re-read state: %w", err)
	}
	fmt.Fprintf(out, "\nDone. makerNostrKeyHash=%s stake=%s\n", st2.MakerNostrKeyHash.Hex(), st2.Stake.String())
	return nil
}

func ensureBalance(ctx context.Context, client *ethclient.Client, from common.Address, depositWei *big.Int) error {
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("gas price: %w", err)
	}
	// Upper-bound gas for deposit + register (register may be ~100k+).
	const gasCeil uint64 = 500_000
	needGas := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasCeil)))
	need := new(big.Int).Add(depositWei, needGas)

	bal, err := client.BalanceAt(ctx, from, nil)
	if err != nil {
		return fmt.Errorf("balance: %w", err)
	}
	if bal.Cmp(need) < 0 {
		return fmt.Errorf("insufficient balance: have %s wei, need at least %s wei (deposit %s + gas buffer)",
			bal.String(), need.String(), depositWei.String())
	}
	return nil
}

func sendDeposit(ctx context.Context, client *ethclient.Client, chainID *big.Int, registry common.Address, key *ecdsa.PrivateKey, value *big.Int) (*types.Transaction, error) {
	from := crypto.PubkeyToAddress(key.PublicKey)
	data, err := parsedABI.Pack("deposit")
	if err != nil {
		return nil, err
	}
	return sendLegacyTx(ctx, client, chainID, registry, key, from, value, data)
}

func sendRegisterMaker(ctx context.Context, client *ethclient.Client, chainID *big.Int, registry common.Address, key *ecdsa.PrivateKey, nostrHash common.Hash) (*types.Transaction, error) {
	from := crypto.PubkeyToAddress(key.PublicKey)
	data, err := parsedABI.Pack("registerMaker", nostrHash)
	if err != nil {
		return nil, err
	}
	return sendLegacyTx(ctx, client, chainID, registry, key, from, big.NewInt(0), data)
}

func sendLegacyTx(ctx context.Context, client *ethclient.Client, chainID *big.Int, to common.Address, key *ecdsa.PrivateKey, from common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, err
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{From: from, To: &to, GasPrice: gasPrice, Value: value, Data: data}
	gasLimit, err := client.EstimateGas(ctx, msg)
	if err != nil {
		gasLimit = 200_000
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &to,
		Value:    value,
		Data:     data,
	})
	signer := types.NewEIP155Signer(chainID)
	signed, err := types.SignTx(tx, signer, key)
	if err != nil {
		return nil, err
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		if strings.Contains(err.Error(), "nonce") {
			time.Sleep(200 * time.Millisecond)
			nonce, nerr := client.PendingNonceAt(ctx, from)
			if nerr == nil && nonce != tx.Nonce() {
				tx2 := types.NewTx(&types.LegacyTx{
					Nonce:    nonce,
					GasPrice: gasPrice,
					Gas:      gasLimit,
					To:       &to,
					Value:    value,
					Data:     data,
				})
				signed, err = types.SignTx(tx2, signer, key)
				if err != nil {
					return nil, err
				}
				if err := client.SendTransaction(ctx, signed); err != nil {
					return nil, err
				}
				return signed, nil
			}
		}
		return nil, err
	}
	return signed, nil
}
