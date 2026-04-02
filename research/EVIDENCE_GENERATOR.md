# Off-chain `evidenceHash` and `grievanceId` (implementer note)

Canonical definitions live in **`PRODUCT_SPEC.md` Appendix 13** and **`contracts/src/EvidenceLib.sol`**.  
This note gives **Python** and **Go** snippets that match **`abi.encodePacked`** on LitVM, **golden test vectors** checked in **`contracts/test/EvidenceGoldenVectors.t.sol`**, and how to re-emit values from **`contracts/script/PrintEvidenceVectors.s.sol`**.

See **`research/COINSWAPD_TEARDOWN.md`** for where **`commit2`** (post-peel) and the exact forward payload come from in the reference **`coinswapd`** implementation.

---

## 1. `evidenceHash` preimage (137 bytes)

Packed order (tight `abi.encodePacked`, no padding between fields):

| Order | Field | EVM type | Bytes | Notes |
| ----- | ----- | -------- | ----- | ----- |
| 1 | `epochId` | `uint256` | 32 | Big-endian |
| 2 | `accuser` | `address` | 20 | — |
| 3 | `accusedMaker` | `address` | 20 | Registry-bound identity of the accused hop (spec §13.2) |
| 4 | `hopIndex` | `uint8` | 1 | 0-based index in the peeled route |
| 5 | `peeledCommitment` | `bytes32` | 32 | `sha256(ltcd 33-byte compressed commit2)` (§13.3) |
| 6 | `forwardCiphertextHash` | `bytes32` | 32 | `keccak256(P)` where `P` = exact **post-XOR** `swap_forward` payload (§13.4) |

```solidity
// contracts/src/EvidenceLib.sol
function evidenceHash(
    uint256 epochId,
    address accuser,
    address accusedMaker,
    uint8 hopIndex,
    bytes32 peeledCommitment,
    bytes32 forwardCiphertextHash
) internal pure returns (bytes32) {
    return keccak256(
        abi.encodePacked(
            epochId, accuser, accusedMaker, hopIndex, peeledCommitment, forwardCiphertextHash
        )
    );
}
```

---

## 2. `grievanceId` preimage (104 bytes)

Storage key in **`GrievanceCourt`**; same packing as **`EvidenceLib.grievanceId`**:

| Order | Field | Bytes | Notes |
| ----- | ----- | ----- | ----- |
| 1 | `accuser` | 20 | Must match **`msg.sender`** at **`openGrievance`** |
| 2 | `accused` | 20 | Same 20 bytes as **`accusedMaker`** in the evidence preimage |
| 3 | `epochId` | 32 | Big-endian `uint256` |
| 4 | `evidenceHash` | 32 | Output of §1 |

```solidity
// contracts/src/EvidenceLib.sol
function grievanceId(address accuser, address accused, uint256 epochId, bytes32 evidenceHash_)
    internal pure
    returns (bytes32)
{
    return keccak256(abi.encodePacked(accuser, accused, epochId, evidenceHash_));
}
```

Off-chain clients must use the **same** `accuser` address in this preimage as the wallet that will call **`openGrievance`**.

---

## 3. Python (real Keccak-256)

Use **original Keccak-256** (Ethereum). **`hashlib.sha3_256`** is **FIPS SHA-3**, not EVM Keccak — do not use it for these hashes.

```bash
pip install pycryptodome
```

```python
from typing import Tuple

def keccak256(data: bytes) -> bytes:
    """Ethereum Keccak-256 (original Keccak, not FIPS SHA3)."""
    from Crypto.Hash import keccak
    return keccak.new(digest_bits=256, data=data).digest()


def evidence_hash_and_id(
    epoch_id: int,
    accuser: str,
    accused_maker: str,
    hop_index: int,
    peeled_commitment: bytes,
    forward_ciphertext: bytes,
) -> Tuple[bytes, bytes, str]:
    """Returns (evidenceHash, grievanceId, evidence_preimage_hex)."""
    if epoch_id < 0 or epoch_id.bit_length() > 256:
        raise ValueError("epoch_id must fit uint256")
    if not (0 <= hop_index <= 255):
        raise ValueError("hop_index must be uint8")
    if len(peeled_commitment) != 32:
        raise ValueError("peeled_commitment must be 32 bytes")

    def _addr(s: str) -> bytes:
        s = s.lower()
        return bytes.fromhex(s[2:] if s.startswith("0x") else s)

    accuser_b = _addr(accuser)
    accused_b = _addr(accused_maker)
    if len(accuser_b) != 20 or len(accused_b) != 20:
        raise ValueError("addresses must be 20 bytes")

    forward_ct_hash = keccak256(forward_ciphertext)

    preimage = bytearray()
    preimage.extend(epoch_id.to_bytes(32, "big"))
    preimage.extend(accuser_b)
    preimage.extend(accused_b)
    preimage.extend(bytes([hop_index & 0xFF]))
    preimage.extend(peeled_commitment)
    preimage.extend(forward_ct_hash)

    assert len(preimage) == 137
    ev_hash = keccak256(preimage)

    gid_preimage = bytearray()
    gid_preimage.extend(accuser_b)
    gid_preimage.extend(accused_b)
    gid_preimage.extend(epoch_id.to_bytes(32, "big"))
    gid_preimage.extend(ev_hash)
    assert len(gid_preimage) == 104

    grievance_id = keccak256(gid_preimage)
    return ev_hash, grievance_id, preimage.hex()
```

---

## 4. Go (`golang.org/x/crypto/sha3`, LegacyKeccak256)

```go
package evidence

import (
	"math/big"

	"golang.org/x/crypto/sha3"
)

func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

func EvidenceHashAndID(
	epochID *big.Int,
	accuser, accusedMaker [20]byte,
	hopIndex uint8,
	peeledCommitment [32]byte,
	forwardCiphertext []byte,
) (evHash, grievanceID [32]byte) {
	if epochID.Sign() < 0 || epochID.BitLen() > 256 {
		panic("epochID must fit uint256")
	}

	epochBytes := make([]byte, 32)
	epochID.FillBytes(epochBytes)

	forwardCTHash := keccak256(forwardCiphertext)

	preimage := append([]byte{}, epochBytes...)
	preimage = append(preimage, accuser[:]...)
	preimage = append(preimage, accusedMaker[:]...)
	preimage = append(preimage, hopIndex)
	preimage = append(preimage, peeledCommitment[:]...)
	preimage = append(preimage, forwardCTHash...)
	if len(preimage) != 137 {
		panic("evidence preimage must be 137 bytes")
	}

	copy(evHash[:], keccak256(preimage))

	gidPre := append([]byte{}, accuser[:]...)
	gidPre = append(gidPre, accusedMaker[:]...)
	gidPre = append(gidPre, epochBytes...)
	gidPre = append(gidPre, evHash[:]...)
	if len(gidPre) != 104 {
		panic("grievance preimage must be 104 bytes")
	}

	copy(grievanceID[:], keccak256(gidPre))
	return evHash, grievanceID
}
```

---

## 5. Tests, golden vectors, and regeneration

**Unit + integration:** **`contracts/test/EvidenceHash.t.sol`** (manual `encodePacked` checks, `openGrievance` storage key, fuzz lives in **`FuzzGrievanceCourt.t.sol`** / related fuzz tests).

**Locked golden outputs** (same inputs as the fixed-arity checks in **`EvidenceHash.t.sol`**): **`contracts/test/EvidenceGoldenVectors.t.sol`**. If you change inputs or hashing, update **that test** and the table below.

**Re-emit preimage and hashes** (for docs or client cross-checks): **`contracts/script/PrintEvidenceVectors.s.sol`** — same inputs as the golden test.

```bash
# Local forge
cd contracts && forge script script/PrintEvidenceVectors.s.sol:PrintEvidenceVectors --sig "run()"
```

**Docker / no local `forge`:** from repo root or **`contracts/`**, Docker must be running. Root **`Makefile`** resolves **`contracts/`** from the makefile path (not shell `PWD`), so mounts stay correct.

```bash
make contracts-test-match MATCH=EvidenceGoldenVectorsTest
# or from contracts/
make test-golden
```

See **`contracts/README.md`** for **`make test`** (full suite) and Foundry install notes.

### Fixed inputs (golden row)

- `epochId` = `42`
- `accuser` = `address(uint160(0xBEEF))`
- `accusedMaker` = `address(uint160(0xCAFE))`
- `hopIndex` = `2`
- `peeledCommitment` = `bytes32(uint256(0x1111))`
- `forwardCiphertextHash` = `bytes32(uint256(0x2222))` — the Solidity tests pass this digest directly; for an end-to-end check, choose **`forward_ciphertext`** such that **`keccak256(P)`** equals **`0x…2222`**.

**137-byte evidence preimage (hex, grouped):**

```text
000000000000000000000000000000000000000000000000000000000000002a
000000000000000000000000000000000000beef
000000000000000000000000000000000000cafe
02
0000000000000000000000000000000000000000000000000000000000001111
0000000000000000000000000000000000000000000000000000000000002222
```

**Single line:**

```text
000000000000000000000000000000000000000000000000000000000000002a000000000000000000000000000000000000beef000000000000000000000000000000000000cafe0200000000000000000000000000000000000000000000000000000000000011110000000000000000000000000000000000000000000000000000000000002222
```

**Outputs:**

| Value | Hex |
| ----- | --- |
| `evidenceHash` | `2d4d7ae96f39e2d5037f21782bc831874261ffe22743f74bbf865a39ec4df112` |
| `grievanceId` | `5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3` |

---

## 6. Nostr / coinswapd (pointers)

- **Nostr:** publish the **correct `grievanceId`** (§2) in tags (e.g. grievance-pointer kinds); LitVM stays authoritative.
- **coinswapd:** hash the **exact post-XOR** forward blob (§13.4) and derive **`peeledCommitment`** from **`ltcd`** `mw.Commitment` serialization (§13.3); see the teardown for code paths.

This keeps the happy-path MWEB engine unchanged while making grievance submission consistent across wallets, nodes, and the court.
