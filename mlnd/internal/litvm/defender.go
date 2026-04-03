package litvm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Defender submits defendGrievance when AutoDefend is enabled (see LoadDefenderFromEnv).
type Defender struct {
	contract *bind.BoundContract
	opts     *bind.TransactOpts
	dryRun   bool
}

// IsDryRun is true when MLND_DEFEND_DRY_RUN is set (no transaction broadcast).
func (d *Defender) IsDryRun() bool {
	return d != nil && d.dryRun
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

// parsePrivateKeyHex accepts 64 hex chars with optional 0x prefix.
func parsePrivateKeyHex(s string) (*ecdsa.PrivateKey, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	if len(s) != 64 {
		return nil, fmt.Errorf("want 64 hex chars (optionally 0x-prefixed), got length %d", len(s))
	}
	return crypto.HexToECDSA(s)
}

// LoadDefenderFromEnv returns nil when auto-defend is off or no private key is set.
// Requires MLND_DEFEND_AUTO truthy and MLND_OPERATOR_PRIVATE_KEY when defending.
// Validates derived address equals operatorAddrHex.
func LoadDefenderFromEnv(ctx context.Context, client *ethclient.Client, courtAddrHex, operatorAddrHex string) (*Defender, error) {
	if !envTruthy("MLND_DEFEND_AUTO") {
		return nil, nil
	}
	keyHex := strings.TrimSpace(os.Getenv("MLND_OPERATOR_PRIVATE_KEY"))
	if keyHex == "" {
		return nil, fmt.Errorf("MLND_DEFEND_AUTO is set but MLND_OPERATOR_PRIVATE_KEY is empty")
	}
	key, err := parsePrivateKeyHex(keyHex)
	if err != nil {
		return nil, fmt.Errorf("MLND_OPERATOR_PRIVATE_KEY: %w", err)
	}
	wantOp := common.HexToAddress(operatorAddrHex)
	gotAddr := crypto.PubkeyToAddress(key.PublicKey)
	if gotAddr != wantOp {
		return nil, fmt.Errorf("operator key derives %s but MLND_OPERATOR_ADDR is %s", gotAddr.Hex(), wantOp.Hex())
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
	opts, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		return nil, err
	}
	opts.Context = ctx

	court := common.HexToAddress(courtAddrHex)
	return &Defender{
		contract: NewGrievanceCourtBound(client, court),
		opts:     opts,
		dryRun:   envTruthy("MLND_DEFEND_DRY_RUN"),
	}, nil
}

// SubmitDefend sends defendGrievance with retries on transport errors (not on revert).
func (d *Defender) SubmitDefend(ctx context.Context, client *ethclient.Client, grievanceID common.Hash, defenseData []byte) (*types.Transaction, error) {
	if d == nil {
		return nil, fmt.Errorf("nil defender")
	}
	if d.dryRun {
		return nil, nil
	}
	d.opts.Context = ctx

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(200*attempt) * time.Millisecond)
		}
		gp, err := client.SuggestGasPrice(ctx)
		if err == nil {
			d.opts.GasPrice = gp
		}
		tx, err := d.contract.Transact(d.opts, "defendGrievance", grievanceID, defenseData)
		if err == nil {
			return tx, nil
		}
		lastErr = err
		if !isRetryableSubmitErr(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("after retries: %w", lastErr)
}

func isRetryableSubmitErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "execution reverted") || strings.Contains(s, "insufficient funds") {
		return false
	}
	if strings.Contains(s, "revert") {
		return false
	}
	return true
}
