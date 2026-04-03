package dashboard

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const registryABIJSON = `[
  {"name":"makerNostrKeyHash","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"bytes32"}]},
  {"name":"stake","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"uint256"}]},
  {"name":"minStake","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256"}]},
  {"name":"stakeFrozen","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"bool"}]},
  {"name":"exitUnlockTime","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"uint256"}]},
  {"name":"cooldownPeriod","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256"}]},
  {"name":"grievanceCourt","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"address"}]}
]`

const courtABIJSON = `[
  {"name":"openGrievanceCountAgainst","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"uint256"}]},
  {"name":"withdrawalLockUntil","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"uint256"}]}
]`

var regABI abi.ABI
var courtABI abi.ABI

func init() {
	var err error
	regABI, err = abi.JSON(strings.NewReader(registryABIJSON))
	if err != nil {
		panic(err)
	}
	courtABI, err = abi.JSON(strings.NewReader(courtABIJSON))
	if err != nil {
		panic(err)
	}
}

// ChainPillar is LitVM registry + grievance court views for the operator.
type ChainPillar struct {
	RegistryAddress       string `json:"registryAddress"`
	CourtAddress          string `json:"courtAddress"`
	CourtFromRegistry     string `json:"courtFromRegistry,omitempty"`
	CourtAddressMismatch  bool   `json:"courtAddressMismatch,omitempty"`
	Operator              string `json:"operator"`
	Stake                 string `json:"stake"`
	MinStake              string `json:"minStake"`
	CooldownPeriodSeconds string `json:"cooldownPeriodSeconds"`
	ExitUnlockTime        string `json:"exitUnlockTime"`
	StakeFrozen           bool   `json:"stakeFrozen"`
	MakerNostrKeyHash     string `json:"makerNostrKeyHash"`
	OpenGrievanceCount    string `json:"openGrievanceCount"`
	WithdrawalLockUntil   string `json:"withdrawalLockUntil"`
	RPCError              string `json:"rpcError,omitempty"`
}

func readChainPillar(ctx context.Context, client *ethclient.Client, registryAddr, courtAddrHex string, maker common.Address) ChainPillar {
	out := ChainPillar{
		RegistryAddress: registryAddr,
		CourtAddress:    courtAddrHex,
		Operator:        maker.Hex(),
	}
	if client == nil {
		out.RPCError = "no eth client"
		return out
	}
	reg := common.HexToAddress(registryAddr)
	courtConfigured := common.HexToAddress(courtAddrHex)

	data, err := regABI.Pack("grievanceCourt")
	if err != nil {
		out.RPCError = err.Error()
		return out
	}
	res, err := client.CallContract(ctx, ethereum.CallMsg{To: &reg, Data: data}, nil)
	if err != nil {
		out.RPCError = fmt.Sprintf("grievanceCourt: %v", err)
		return out
	}
	vals, err := regABI.Unpack("grievanceCourt", res)
	if err != nil || len(vals) != 1 {
		out.RPCError = "unpack grievanceCourt"
		return out
	}
	courtOnChain := vals[0].(common.Address)
	out.CourtFromRegistry = courtOnChain.Hex()
	if courtOnChain != courtConfigured && courtOnChain != (common.Address{}) {
		out.CourtAddressMismatch = true
	}
	court := courtConfigured
	if courtOnChain != (common.Address{}) {
		court = courtOnChain
	}
	out.CourtAddress = court.Hex()

	callReg := func(method string, args ...interface{}) ([]interface{}, error) {
		data, err := regABI.Pack(method, args...)
		if err != nil {
			return nil, err
		}
		raw, err := client.CallContract(ctx, ethereum.CallMsg{To: &reg, Data: data}, nil)
		if err != nil {
			return nil, err
		}
		return regABI.Unpack(method, raw)
	}
	callCourt := func(method string, args ...interface{}) ([]interface{}, error) {
		data, err := courtABI.Pack(method, args...)
		if err != nil {
			return nil, err
		}
		raw, err := client.CallContract(ctx, ethereum.CallMsg{To: &court, Data: data}, nil)
		if err != nil {
			return nil, err
		}
		return courtABI.Unpack(method, raw)
	}

	if v, err := callReg("makerNostrKeyHash", maker); err == nil && len(v) == 1 {
		switch x := v[0].(type) {
		case common.Hash:
			out.MakerNostrKeyHash = x.Hex()
		case [32]byte:
			out.MakerNostrKeyHash = common.Hash(x).Hex()
		}
	} else if err != nil {
		out.RPCError = fmt.Sprintf("makerNostrKeyHash: %v", err)
		return out
	}

	if v, err := callReg("stake", maker); err == nil && len(v) == 1 {
		out.Stake = v[0].(*big.Int).String()
	} else if err != nil {
		out.RPCError = fmt.Sprintf("stake: %v", err)
		return out
	}

	if v, err := callReg("minStake"); err == nil && len(v) == 1 {
		out.MinStake = v[0].(*big.Int).String()
	} else if err != nil {
		out.RPCError = fmt.Sprintf("minStake: %v", err)
		return out
	}

	if v, err := callReg("cooldownPeriod"); err == nil && len(v) == 1 {
		out.CooldownPeriodSeconds = v[0].(*big.Int).String()
	} else if err != nil {
		out.RPCError = fmt.Sprintf("cooldownPeriod: %v", err)
		return out
	}

	if v, err := callReg("exitUnlockTime", maker); err == nil && len(v) == 1 {
		out.ExitUnlockTime = v[0].(*big.Int).String()
	} else if err != nil {
		out.RPCError = fmt.Sprintf("exitUnlockTime: %v", err)
		return out
	}

	if v, err := callReg("stakeFrozen", maker); err == nil && len(v) == 1 {
		out.StakeFrozen = v[0].(bool)
	} else if err != nil {
		out.RPCError = fmt.Sprintf("stakeFrozen: %v", err)
		return out
	}

	if v, err := callCourt("openGrievanceCountAgainst", maker); err == nil && len(v) == 1 {
		out.OpenGrievanceCount = v[0].(*big.Int).String()
	} else if err != nil {
		out.RPCError = fmt.Sprintf("openGrievanceCountAgainst: %v", err)
		return out
	}

	if v, err := callCourt("withdrawalLockUntil", maker); err == nil && len(v) == 1 {
		out.WithdrawalLockUntil = v[0].(*big.Int).String()
	} else if err != nil {
		out.RPCError = fmt.Sprintf("withdrawalLockUntil: %v", err)
		return out
	}

	return out
}
