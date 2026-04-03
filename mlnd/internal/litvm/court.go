package litvm

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

//go:embed abi/grievancecourt_defend.json
var grievanceCourtDefendJSON []byte

var parsedCourtDefendABI abi.ABI

func init() {
	var err error
	parsedCourtDefendABI, err = abi.JSON(bytes.NewReader(grievanceCourtDefendJSON))
	if err != nil {
		panic(fmt.Sprintf("litvm: parse embedded GrievanceCourt ABI: %v", err))
	}
}

// NewGrievanceCourtBound returns a bound contract with only defendGrievance (minimal ABI).
func NewGrievanceCourtBound(client *ethclient.Client, courtAddress common.Address) *bind.BoundContract {
	return bind.NewBoundContract(courtAddress, parsedCourtDefendABI, client, client, client)
}
