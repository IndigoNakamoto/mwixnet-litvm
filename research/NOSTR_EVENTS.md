# Nostr event profile (Phase 2 draft)

This note defines an initial Nostr profile for MLN discovery and grievance visibility.

- Canonical judicial hash definitions remain in `PRODUCT_SPEC.md` appendix 13 and `contracts/src/EvidenceLib.sol`.
- Off-chain implementer details and golden vectors are in `research/EVIDENCE_GENERATOR.md`.
- The local on-chain smoke flow is `make test-grievance` (repo root), which verifies the same `grievanceId` path end to end.

Nostr here is a discovery/gossip rail. LitVM remains the authority for stake and grievance state.

---

## Event kinds

| Kind | Name | Purpose |
| ---- | ---- | ------- |
| `30001` | Maker Advertisement | Publish maker reachability, stake pointer, and fee metadata |
| `31001` | Grievance Pointer | Publish a signed pointer to a LitVM grievance (`grievanceId`, `evidenceHash`) |
| `31002` | Resolution Echo | Publish outcome pointer after on-chain resolution |

These values are a project-local profile and may evolve into a formal NIP later.

---

## 1) Kind `30001` - Maker Advertisement

Suggested required tags:

- `["litvm-stake","0x<maker-address-or-registry-pointer>"]`
- `["fee","<human-readable fee hint>"]` (for example `"0.25%"` or policy string)
- `["tor","<onion-endpoint-or-contact-hint>"]`

Recommended optional tags:

- `["network","mainnet|testnet|local"]`
- `["version","<maker-software-version>"]`
- `["capability","coinswap-v1"]`

Suggested `content` payload:

```json
{
  "policy": {
    "min_size_ltc": "0.01",
    "max_size_ltc": "10",
    "fee_model": "mweb-hop-fee"
  },
  "notes": "Maker available"
}
```

---

## 2) Kind `31001` - Grievance Pointer

Suggested required tags:

- `["epoch","<epochId>"]`
- `["grievance","0x<grievanceId>"]`
- `["evidenceHash","0x<evidenceHash>"]`

Recommended optional tags:

- `["court","0x<GrievanceCourtAddress>"]`
- `["accused","0x<maker-address>"]`
- `["chain","litvm-testnet|anvil-31337|..."]`

Suggested `content` payload:

```json
{
  "message": "Grievance opened on LitVM",
  "txHash": "0x..."
}
```

Critical rule: compute and publish the same `grievanceId` that LitVM derives from
`keccak256(abi.encodePacked(accuser, accused, epochId, evidenceHash))`.
Use `research/EVIDENCE_GENERATOR.md` as the canonical implementer guide.

---

## 3) Kind `31002` - Resolution Echo

Suggested required tags:

- `["grievance","0x<grievanceId>"]`
- `["status","slashed|exonerated"]`

Recommended optional tags:

- `["court","0x<GrievanceCourtAddress>"]`
- `["tx","0x<resolve-tx-hash>"]`

Suggested `content` payload:

```json
{
  "message": "Resolution observed on LitVM"
}
```

---

## Example publish script (`python`)

The snippet below signs a `kind=31001` event and prints JSON that can be sent through any Nostr relay client.

```python
#!/usr/bin/env python3
import json
import sys
from nostr.event import Event
from nostr.key import PrivateKey

# Usage:
# python3 scripts/publish_grievance.py <grievanceId_hex> <epochId> <evidenceHash_hex> <nostr_privkey_hex>

grievance_id = sys.argv[1].removeprefix("0x")
epoch_id = sys.argv[2]
evidence_hash = sys.argv[3].removeprefix("0x")
privkey_hex = sys.argv[4].removeprefix("0x")

privkey = PrivateKey(bytes.fromhex(privkey_hex))

event = Event(
    kind=31001,
    content=json.dumps({"message": "Grievance opened on LitVM"}),
    tags=[
        ["epoch", str(epoch_id)],
        ["grievance", "0x" + grievance_id],
        ["evidenceHash", "0x" + evidence_hash],
    ],
)

event.sign(privkey)
print(json.dumps(event.to_dict()))
```

---

## Subscribe/filter examples

`REQ` filter for grievance pointers:

```json
["REQ","sub-grievances",{"kinds":[31001],"#grievance":["0x5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3"]}]
```

`REQ` filter for all resolution echoes:

```json
["REQ","sub-resolutions",{"kinds":[31002]}]
```

Wallet UX can treat `kind=31001` as a pending incident and reconcile final state from LitVM (source of truth).

---

## Integration with local grievance test

1. Run `make test-grievance` from repo root.
2. Read the emitted `grievanceId` / `evidenceHash` (golden vector in local flow).
3. Publish a `kind=31001` event with those values.
4. After `resolveGrievance`, publish `kind=31002`.

This gives an immediate bridge from local judicial execution to relay-visible operational telemetry without changing the MWEB happy path.
