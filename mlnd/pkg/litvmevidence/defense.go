package litvmevidence

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ReceiptForDefense is hop receipt data loaded for a grievance defense path.
type ReceiptForDefense struct {
	EpochID               *big.Int
	Accuser               common.Address
	AccusedMaker          common.Address
	HopIndex              uint8
	PeeledCommitment      common.Hash
	ForwardCiphertextHash common.Hash
	NextHopPubkey         string
	Signature             string
}

// ReceiptLookup is implemented by the evidence store; nil means log-only watcher behavior.
type ReceiptLookup interface {
	GetByEvidenceHash(hash common.Hash) (*ReceiptForDefense, error)
}

// defenseTupleType is one Solidity tuple for abi.encode (single off-chain decode).
var defenseTupleType, _ = abi.NewType("tuple", "", []abi.ArgumentMarshaling{
	{Name: "version", Type: "uint8"},
	{Name: "epochId", Type: "uint256"},
	{Name: "accuser", Type: "address"},
	{Name: "accusedMaker", Type: "address"},
	{Name: "hopIndex", Type: "uint8"},
	{Name: "peeledCommitment", Type: "bytes32"},
	{Name: "forwardCiphertextHash", Type: "bytes32"},
	{Name: "nextHopPubkeyUTF8", Type: "bytes"},
	{Name: "signatureUTF8", Type: "bytes"},
})

var defensePackArgs = abi.Arguments{{Type: defenseTupleType}}

// ValidateReceiptForGrievance checks the stored receipt matches the opened grievance correlators and hashes.
func ValidateReceiptForGrievance(ev *GrievanceOpened, r *ReceiptForDefense, operator common.Address) error {
	if ev == nil || r == nil {
		return fmt.Errorf("nil event or receipt")
	}
	if r.AccusedMaker != operator || ev.Accused != operator {
		return fmt.Errorf("accused/operator mismatch (ev.accused=%s receipt.accusedMaker=%s want operator=%s)",
			ev.Accused.Hex(), r.AccusedMaker.Hex(), operator.Hex())
	}
	if ev.Accuser != r.Accuser {
		return fmt.Errorf("accuser mismatch: log %s receipt %s", ev.Accuser.Hex(), r.Accuser.Hex())
	}
	if ev.EpochID == nil || r.EpochID == nil {
		return fmt.Errorf("epochId required")
	}
	if ev.EpochID.Cmp(r.EpochID) != 0 {
		return fmt.Errorf("epochId mismatch: log %s receipt %s", ev.EpochID.String(), r.EpochID.String())
	}
	pre := EvidencePreimage{
		EpochID:               r.EpochID,
		Accuser:               r.Accuser,
		AccusedMaker:          r.AccusedMaker,
		HopIndex:              r.HopIndex,
		PeeledCommitment:      r.PeeledCommitment,
		ForwardCiphertextHash: r.ForwardCiphertextHash,
	}
	got := ComputeEvidenceHash(pre)
	if got != ev.EvidenceHash {
		return fmt.Errorf("evidenceHash mismatch: computed %s log %s", got.Hex(), ev.EvidenceHash.Hex())
	}
	gid := ComputeGrievanceID(r.Accuser, r.AccusedMaker, r.EpochID, got)
	if gid != ev.GrievanceID {
		return fmt.Errorf("grievanceId mismatch: computed %s log %s", gid.Hex(), ev.GrievanceID.Hex())
	}
	return nil
}

// BuildDefenseData returns abi.encode(tuple(...)) for defenseData v1 (opaque on-chain).
func BuildDefenseData(r *ReceiptForDefense) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("nil receipt")
	}
	if r.EpochID == nil {
		return nil, fmt.Errorf("EpochID required")
	}
	tuple := struct {
		Version               uint8
		EpochID               *big.Int `abi:"epochId"`
		Accuser               common.Address
		AccusedMaker          common.Address
		HopIndex              uint8
		PeeledCommitment      common.Hash
		ForwardCiphertextHash common.Hash
		NextHopPubkeyUTF8     []byte
		SignatureUTF8         []byte
	}{
		Version:               1,
		EpochID:               new(big.Int).Set(r.EpochID),
		Accuser:               r.Accuser,
		AccusedMaker:          r.AccusedMaker,
		HopIndex:              r.HopIndex,
		PeeledCommitment:      r.PeeledCommitment,
		ForwardCiphertextHash: r.ForwardCiphertextHash,
		NextHopPubkeyUTF8:     []byte(r.NextHopPubkey),
		SignatureUTF8:         []byte(r.Signature),
	}
	return defensePackArgs.Pack(tuple)
}

// UnpackDefenseV1 decodes v1 defenseData bytes from defendGrievance calldata.
func UnpackDefenseV1(data []byte) (*ReceiptForDefense, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty defenseData")
	}
	out, err := defensePackArgs.Unpack(data)
	if err != nil {
		return nil, err
	}
	if len(out) != 1 {
		return nil, fmt.Errorf("unpack: want 1 field, got %d", len(out))
	}
	return unpackDefenseReflect(out[0])
}

func unpackDefenseReflect(unpacked any) (*ReceiptForDefense, error) {
	rv := reflect.ValueOf(unpacked)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct || rv.NumField() < 9 {
		return nil, fmt.Errorf("unexpected unpack shape kind=%s numField=%d", rv.Kind(), rv.NumField())
	}
	ver := uint8(rv.Field(0).Uint())
	if ver != 1 {
		return nil, fmt.Errorf("defense version: got %d want 1", ver)
	}
	epoch, ok := rv.Field(1).Interface().(*big.Int)
	if !ok || epoch == nil {
		return nil, fmt.Errorf("epochId field")
	}
	accuser, ok := rv.Field(2).Interface().(common.Address)
	if !ok {
		return nil, fmt.Errorf("accuser field")
	}
	accused, ok := rv.Field(3).Interface().(common.Address)
	if !ok {
		return nil, fmt.Errorf("accusedMaker field")
	}
	hop := uint8(rv.Field(4).Uint())
	peeled := hashFromABIField(rv.Field(5).Interface())
	fwd := hashFromABIField(rv.Field(6).Interface())
	nextPub, ok := rv.Field(7).Interface().([]byte)
	if !ok {
		return nil, fmt.Errorf("nextHopPubkeyUTF8 field")
	}
	sig, ok := rv.Field(8).Interface().([]byte)
	if !ok {
		return nil, fmt.Errorf("signatureUTF8 field")
	}
	return &ReceiptForDefense{
		EpochID:               new(big.Int).Set(epoch),
		Accuser:               accuser,
		AccusedMaker:          accused,
		HopIndex:              hop,
		PeeledCommitment:      peeled,
		ForwardCiphertextHash: fwd,
		NextHopPubkey:         string(nextPub),
		Signature:             string(sig),
	}, nil
}

func hashFromABIField(v any) common.Hash {
	switch x := v.(type) {
	case common.Hash:
		return x
	case [32]byte:
		return common.Hash(x)
	default:
		return common.Hash{}
	}
}

// ChainTimeBeforeDeadline compares latest chain header time to grievance deadline (Unix seconds).
func ChainTimeBeforeDeadline(header *types.Header, deadline *big.Int) bool {
	if header == nil || deadline == nil {
		return false
	}
	chainTs := new(big.Int).SetUint64(header.Time)
	return chainTs.Cmp(deadline) < 0
}
