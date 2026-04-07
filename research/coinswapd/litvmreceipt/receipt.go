// Package litvmreceipt builds LitVM grievance receipt fields tied to MWEB swap_forward wire data (PRODUCT_SPEC appendix 13).
package litvmreceipt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/ltcmweb/ltcd/ltcutil/mweb/mw"
	"golang.org/x/crypto/sha3"
)

// UnsignedSwapForwardFailureV1 is the v1 testnet sentinel for signature (no maker-signed ack yet).
const UnsignedSwapForwardFailureV1 = "unsigned-swap-forward-failure-v1"

// PeeledCommitmentHash is sha256 over the 33-byte compressed Pedersen commitment (post-peel state) per appendix 13.3.
func PeeledCommitmentHash(commit2 mw.Commitment) [32]byte {
	return sha256.Sum256(commit2[:])
}

// ForwardCiphertextHash is keccak256 over P, the exact post-XOR bytes sent as swap_forward RPC payload (appendix 13.4).
func ForwardCiphertextHash(p []byte) [32]byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(p)
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func hash0x(h [32]byte) string {
	return "0x" + hex.EncodeToString(h[:])
}

// SwapForwardFailureReceipt is JSON-shaped like mlnd/pkg/receiptstore NDJSON plus optional debug class.
type SwapForwardFailureReceipt struct {
	EpochID               string `json:"epochId"`
	Accuser               string `json:"accuser"`
	AccusedMaker          string `json:"accusedMaker"`
	HopIndex              int    `json:"hopIndex"`
	PeeledCommitment      string `json:"peeledCommitment"`
	ForwardCiphertextHash string `json:"forwardCiphertextHash"`
	NextHopPubkey         string `json:"nextHopPubkey"`
	Signature             string `json:"signature"`
	SwapID                string `json:"swapId,omitempty"`
	ForwardFailureClass   string `json:"forwardFailureClass,omitempty"`
}

// MarshalSwapForwardFailureReceipt builds canonical JSON for the taker vault / mweb_getLastReceipt path.
// accusedHex must be a normalized 0x-prefixed checksummed or lowercase 20-byte hex address.
func MarshalSwapForwardFailureReceipt(
	epochID, accuser, swapID, accusedHex string,
	hopIndex int,
	commit2 mw.Commitment,
	forwardPayload []byte,
	nextHopPubHex string,
	failureClass string,
) ([]byte, error) {
	acc := strings.TrimSpace(accuser)
	if !strings.HasPrefix(acc, "0x") && !strings.HasPrefix(acc, "0X") {
		acc = "0x" + acc
	}
	a := strings.TrimSpace(accusedHex)
	if !strings.HasPrefix(a, "0x") && !strings.HasPrefix(a, "0X") {
		a = "0x" + a
	}
	a = strings.ToLower(a)
	r := SwapForwardFailureReceipt{
		EpochID:               strings.TrimSpace(epochID),
		Accuser:               acc,
		AccusedMaker:          a,
		HopIndex:              hopIndex,
		PeeledCommitment:      hash0x(PeeledCommitmentHash(commit2)),
		ForwardCiphertextHash: hash0x(ForwardCiphertextHash(forwardPayload)),
		NextHopPubkey:         nextHopPubHex,
		Signature:             UnsignedSwapForwardFailureV1,
		SwapID:                strings.TrimSpace(swapID),
		ForwardFailureClass:   strings.TrimSpace(failureClass),
	}
	return json.Marshal(r)
}
