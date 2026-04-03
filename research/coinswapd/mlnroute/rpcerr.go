package mlnroute

import "github.com/ethereum/go-ethereum/rpc"

// CustomRPCError is JSON-RPC application error (e.g. -32000 insufficient funds).
type CustomRPCError struct {
	Code    int
	Message string
}

func (e *CustomRPCError) Error() string { return e.Message }

func (e *CustomRPCError) ErrorCode() int { return e.Code }

var (
	_ rpc.Error = (*CustomRPCError)(nil)
)

const (
	CodeInsufficientFunds = -32000
	CodeOnionBuild        = -32000
	CodeInvalidParams     = -32602
	CodeInternal          = -32603
)

// InvalidParams is JSON-RPC -32602 (validation, swap keys, etc.).
func InvalidParams(msg string) error {
	return &CustomRPCError{Code: CodeInvalidParams, Message: msg}
}

func Internal(msg string) error {
	return &CustomRPCError{Code: CodeInternal, Message: msg}
}

func InsufficientFunds(msg string) error {
	if msg == "" {
		msg = "insufficient funds"
	}
	return &CustomRPCError{Code: CodeInsufficientFunds, Message: msg}
}

func OnionOrCrypto(msg string) error {
	return &CustomRPCError{Code: CodeOnionBuild, Message: msg}
}
