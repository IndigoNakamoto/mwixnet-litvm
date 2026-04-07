package judge

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

//go:embed abi/judge_court.json
var judgeCourtJSON []byte

var parsedJudgeABI abi.ABI

func init() {
	var err error
	parsedJudgeABI, err = abi.JSON(bytes.NewReader(judgeCourtJSON))
	if err != nil {
		panic(fmt.Sprintf("judge court abi: %v", err))
	}
}

func newBound(client *ethclient.Client, court common.Address) *bind.BoundContract {
	return bind.NewBoundContract(court, parsedJudgeABI, client, client, client)
}
