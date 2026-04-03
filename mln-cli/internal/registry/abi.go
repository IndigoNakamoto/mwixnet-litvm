package registry

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Minimal ABI for MwixnetRegistry: views used by Scout/onboard + deposit/registerMaker.
const registryABIJSON = `[
  {"name":"makerNostrKeyHash","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"bytes32"}]},
  {"name":"stake","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"uint256"}]},
  {"name":"minStake","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256"}]},
  {"name":"stakeFrozen","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"bool"}]},
  {"name":"exitUnlockTime","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"uint256"}]},
  {"name":"deposit","type":"function","stateMutability":"payable","inputs":[],"outputs":[]},
  {"name":"registerMaker","type":"function","stateMutability":"nonpayable","inputs":[{"name":"nostrKeyHash","type":"bytes32"}],"outputs":[]}
]`

var parsedABI abi.ABI

func init() {
	a, err := abi.JSON(strings.NewReader(registryABIJSON))
	if err != nil {
		panic(err)
	}
	parsedABI = a
}
