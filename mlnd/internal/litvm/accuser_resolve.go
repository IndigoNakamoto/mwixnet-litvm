package litvm

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

//go:embed abi/grievancecourt_resolve.json
var grievanceCourtResolveJSON []byte

var parsedCourtResolveABI abi.ABI

func init() {
	var err error
	parsedCourtResolveABI, err = abi.JSON(bytes.NewReader(grievanceCourtResolveJSON))
	if err != nil {
		panic(fmt.Sprintf("litvm: parse embedded GrievanceCourt resolve ABI: %v", err))
	}
}

func newGrievanceCourtResolveBound(client *ethclient.Client, courtAddress common.Address) *bind.BoundContract {
	return bind.NewBoundContract(courtAddress, parsedCourtResolveABI, client, client, client)
}

// AccuserResolver submits resolveGrievance when the accused stayed silent past the deadline.
type AccuserResolver struct {
	contract   *bind.BoundContract
	courtAddr  common.Address
	opts       *bind.TransactOpts
	dryRun     bool
}

// IsDryRun is true when MLND_ACCUSER_RESOLVE_DRY_RUN is set.
func (a *AccuserResolver) IsDryRun() bool {
	return a != nil && a.dryRun
}

// LoadAccuserResolverFromEnv returns nil when auto-resolve is off or key missing.
func LoadAccuserResolverFromEnv(ctx context.Context, client *ethclient.Client, courtAddrHex, accuserAddrHex string) (*AccuserResolver, common.Address, error) {
	if !envTruthy("MLND_GRIEVANCE_RESOLVE_AUTO") {
		return nil, common.Address{}, nil
	}
	keyHex := strings.TrimSpace(os.Getenv("MLND_ACCUSER_PRIVATE_KEY"))
	if keyHex == "" {
		return nil, common.Address{}, fmt.Errorf("MLND_GRIEVANCE_RESOLVE_AUTO is set but MLND_ACCUSER_PRIVATE_KEY is empty")
	}
	key, err := parsePrivateKeyHex(keyHex)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("MLND_ACCUSER_PRIVATE_KEY: %w", err)
	}
	gotAddr := crypto.PubkeyToAddress(key.PublicKey)
	want := strings.TrimSpace(accuserAddrHex)
	if want != "" {
		if !common.IsHexAddress(want) {
			return nil, common.Address{}, fmt.Errorf("MLND_ACCUSER_ADDR invalid")
		}
		if common.HexToAddress(want) != gotAddr {
			return nil, common.Address{}, fmt.Errorf("accuser key derives %s but MLND_ACCUSER_ADDR is %s", gotAddr.Hex(), want)
		}
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("chain id: %w", err)
	}
	opts, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		return nil, common.Address{}, err
	}
	opts.Context = ctx

	court := common.HexToAddress(courtAddrHex)
	return &AccuserResolver{
		contract:  newGrievanceCourtResolveBound(client, court),
		courtAddr: court,
		opts:      opts,
		dryRun:    envTruthy("MLND_ACCUSER_RESOLVE_DRY_RUN"),
	}, gotAddr, nil
}

// SubmitResolve sends resolveGrievance after validating phase Open and past deadline via chain head.
func (a *AccuserResolver) SubmitResolve(ctx context.Context, client *ethclient.Client, grievanceID common.Hash, deadline *big.Int) (*types.Transaction, error) {
	if a == nil {
		return nil, fmt.Errorf("nil accuser resolver")
	}
	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	chainTs := new(big.Int).SetUint64(head.Time)
	if chainTs.Cmp(deadline) < 0 {
		return nil, fmt.Errorf("deadline not reached (chain %s deadline %s)", chainTs.String(), deadline.String())
	}

	data, err := parsedCourtResolveABI.Pack("grievances", grievanceID)
	if err != nil {
		return nil, err
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &a.courtAddr, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("grievances call: %w", err)
	}
	vals, err := parsedCourtResolveABI.Unpack("grievances", out)
	if err != nil || len(vals) != 8 {
		return nil, fmt.Errorf("unpack grievances")
	}
	var phase uint8
	switch p := vals[6].(type) {
	case uint8:
		phase = p
	case *big.Int:
		phase = uint8(p.Uint64())
	default:
		return nil, fmt.Errorf("unexpected phase type %T", vals[6])
	}
	// GrievancePhase.Open == 1
	if phase != 1 {
		return nil, fmt.Errorf("grievance phase %d not Open (cannot resolveGrievance)", phase)
	}

	if a.dryRun {
		return nil, nil
	}
	a.opts.Context = ctx
	return a.contract.Transact(a.opts, "resolveGrievance", grievanceID)
}
