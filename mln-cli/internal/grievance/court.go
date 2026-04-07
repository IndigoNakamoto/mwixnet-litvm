package grievance

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

//go:embed abi/grievancecourt_open.json
var grievanceCourtOpenJSON []byte

var parsedOpenABI abi.ABI

func init() {
	var err error
	parsedOpenABI, err = abi.JSON(bytes.NewReader(grievanceCourtOpenJSON))
	if err != nil {
		panic(fmt.Sprintf("grievance: parse embedded ABI: %v", err))
	}
}

// NewGrievanceCourtOpenBound returns a bound contract with openGrievance only.
func NewGrievanceCourtOpenBound(client *ethclient.Client, courtAddress common.Address) *bind.BoundContract {
	return bind.NewBoundContract(courtAddress, parsedOpenABI, client, client, client)
}
