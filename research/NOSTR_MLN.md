# MLN Nostr wire profile (Phase 2)

**Status:** draft wire spec for Phase 2 — discovery and gossip only. Stake and grievance truth live on **LitVM** ([`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) §5–6). This document is the canonical place for **event kinds**, **`content` JSON**, and **identity binding** to [`MwixnetRegistry`](../contracts/src/MwixnetRegistry.sol).

## Principles

1. **Nostr is transport-only.** Relays see what you publish; design for that ([`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) §6.4, threat model: discovery surveillance).
2. **LitVM is authoritative for stake** — `stake`, `makerNostrKeyHash`, freeze state, and `GrievanceCourt` storage. Wallets **must** confirm balances and cases via RPC, not Nostr.
3. **Payloads are Litecoin/MWEB/LitVM–oriented** where relevant; Nostr carries **pointers and ads**, not MWEB transaction bodies.
4. **Sensitive paths** may use encrypted DMs (NIP-04 / NIP-44) or out-of-band channels; **this profile does not require** encryption for public maker ads. Optional future: sealed fields in `content` (see §Future).
5. **Replaceable events (NIP-33)** for maker ads so each maker has **one logical current ad** per deployment (`d` tag), reducing relay spam.

## Kind allocation (MLN discovery layer)

Reserve **31250–31259** for **MLN discovery / gossip** (this repo). Parsers **filter by `kind`**.

| Kind   | Name                    | Purpose |
| ------ | ----------------------- | ------- |
| 31250  | `mln_maker_ad`          | Replaceable maker advertisement: LitVM pointers, fees, Tor, capabilities. |
| 31251  | `mln_grievance_pointer` | Signed pointer to an on-chain grievance (`grievanceId` + deployment refs). Gossip only — **verify on chain**. |

**Coexistence with other protocols:** Implementations may also use **31240–31246** (or similar) for **MWEB CoinSwap round** coordination in another stack. Those events are **not** part of this profile; relays may carry both. MLN clients distinguish by `kind` and `t` tags.

## `nostrKeyHash` binding (normative)

[`MwixnetRegistry.registerMaker(bytes32 nostrKeyHash)`](../contracts/src/MwixnetRegistry.sol) stores an opaque `bytes32`. For **interoperability**, v1 uses:

Let `P` be the maker’s Nostr **x-only secp256k1** public key (**32 bytes**, big-endian), as in NIP-01 / BIP340.

**`nostrKeyHash` = `keccak256(P)`** — i.e. the EVM/Solidity hash of the 32-byte pubkey **with no prefix** (same as `keccak256(abi.encodePacked(P))` when `P` is a single `bytes32` argument in Solidity).

- Wallets **derive** `P` from the maker’s Nostr secret per standard Nostr crypto.
- Verifiers **compare** `nostrKeyHash` to `registry.makerNostrKeyHash(makerAddress)` after resolving `makerAddress` from LitVM.
- **Note:** The Foundry test suite may use placeholder values (e.g. `keccak256("nostr")`); production clients **must** use the formula above.

If this binding is ever tightened on-chain, bump wire `v` and document a migration.

## Event: `mln_maker_ad` (kind 31250)

### NIP-33 parameters

- **`kind`:** `31250`
- **`tags`:** MUST include:
  - `["d", "<d_tag>"]` — replaceable identifier (see below).
  - `["t", "mln-maker-ad"]` — filter tag for relays/clients.
- **Optional:** `["client", "<name>", "<version>"]`, `["mln", "<semver>"]` for client identification.

**`d` tag (v1):**

`mln:v1:<litvm_chain_id>:<maker_evm_address>`

- `litvm_chain_id`: decimal CAIP-2 style chain id as string (e.g. `"31337"` for local Anvil, or LitVM testnet id when published).
- `maker_evm_address`: lowercase hex with `0x` prefix, 42 characters (20-byte LitVM address).

Same maker on **another chain** = different `d` (separate replaceable stream).

### `content` JSON (string)

UTF-8 JSON object, wire version **`"v": 1`**.

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `v` | number | yes | Wire version; must be `1` for this schema. |
| `litvm` | object | yes | Deployment pointers (see below). |
| `fees` | object | no | Optional fee hints (e.g. `{"unit":"sat_per_hop","min":1,"max":10}`) — **non-binding**; actual fees are off-chain/MWEB per spec. |
| `tor` | string | no | Tor v3 onion for **mix API** (if published; prefer Tor over cleartext where applicable). |
| `swapX25519PubHex` | string | no | **Coinswap onion layer:** 64 **lowercase** hex digits encoding a **32-byte Curve25519** public key for ECDH with takers building `onion.Onion` payloads. Optional for ads that only signal discovery; **required for real MWEB handoff** when takers use `mweb_submitRoute` (see [`COINSWAPD_MLN_FORK_SPEC.md`](COINSWAPD_MLN_FORK_SPEC.md)). |
| `capabilities` | array of string | no | e.g. `["mweb-coinswap-v0"]` — free-form, wallet-defined. |

**`litvm` object (required):**

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `chainId` | string | yes | Same as in `d` tag: decimal chain id. |
| `registry` | string | yes | `MwixnetRegistry` address, `0x` + 40 hex lowercase. |
| `grievanceCourt` | string | yes | `GrievanceCourt` address, same format. |

**`created_at`:** Use normal Nostr event time; replaceable events overwrite by `created_at` per NIP-33.

**Signature:** Standard NIP-01; pubkey must match `P` such that `keccak256(P)` equals on-chain `makerNostrKeyHash` for the advertised maker address (wallets cross-check via registry).

## Event: `mln_grievance_pointer` (kind 31251)

Announces or updates interest in a grievance case. **Does not prove** phase or outcome — clients **read `GrievanceCourt.grievances(grievanceId)`** on LitVM.

### Tags

- MUST: `["t", "mln-grievance"]`
- Optional: `["e", "<related_event_id>"]` if threading to a public accusation note.

### `content` JSON (string)

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `v` | number | yes | Wire version; `1`. |
| `litvm` | object | yes | Same shape as maker ad (`chainId`, `registry`, `grievanceCourt`). |
| `grievanceId` | string | yes | `0x` + 64 hex lowercase — `bytes32` from [`EvidenceLib.grievanceId`](../contracts/src/EvidenceLib.sol) / `GrievanceCourt`. |
| `epochId` | string | no | Decimal string for human display; must match on-chain if present. |
| `accused` | string | no | `0x` + 40 hex — LitVM address. |
| `phase_hint` | string | no | e.g. `"Open"` — **informational only**; authoritative phase is on-chain. |

**Evidence preimages** MUST NOT appear on Nostr for this profile; only **hashes** and pointers ([`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) §6.4).

## Verification recipe (maker ad)

1. Subscribe or query for `kind` **31250**, `t` **mln-maker-ad**, for your target relays.
2. Parse `content` JSON; verify `v == 1` and `litvm.chainId` / addresses match your expected deployment.
3. Verify Schnorr signature per NIP-01; obtain pubkey `P` from the event.
4. Compute `nostrKeyHash = keccak256(P)` (EVM-compatible keccak-256 over 32 bytes).
5. Read `maker` from `d` tag (last segment) and call `makerNostrKeyHash(maker)` on `MwixnetRegistry`; require **equality** with computed hash.
6. Optionally verify `stake(maker) >= minStake` and not frozen.

## Relationship to MWEB round events (31240–31246)

Some deployments use **31240–31246** for **round request → coordinator bid → pledges → transcript** style flows. That is **round execution**, not registry discovery. MLN Phase 2 **this repo** standardizes **31250–31251** for **maker discovery and grievance pointers**. End-to-end Phase 3 may compose **both** profiles; keep kind namespaces distinct.

## Future (non-normative)

- **Sealed `content`:** Optional NIP-44 envelope for fee or Tor fields if publication policy changes.
- **Heartbeats:** Could be new kind in **31252+** or fields inside `mln_maker_ad` — avoid duplicating replaceable events too frequently (prefer updating the ad with a new `created_at` when needed).

## Examples (illustrative)

**Maker ad `content` (pretty-printed; on wire as one JSON string):**

```json
{
  "v": 1,
  "litvm": {
    "chainId": "31337",
    "registry": "0x5fbdb2315678afecb367f032d93f642f64180aa3",
    "grievanceCourt": "0xe7f1725e7734ce288f8367e1bb143e90bb3f0512"
  },
  "fees": { "unit": "sat_per_hop", "min": 1, "max": 99 },
  "tor": "http://abcdefghijklmnop1234567890abcdef1234567890abcdef.onion",
  "capabilities": ["mweb-coinswap-v0"]
}
```

**`d` tag:** `mln:v1:31337:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266`

**Grievance pointer `content`:**

```json
{
  "v": 1,
  "litvm": {
    "chainId": "31337",
    "registry": "0x5fbdb2315678afecb367f032d93f642f64180aa3",
    "grievanceCourt": "0xe7f1725e7734ce288f8367e1bb143e90bb3f0512"
  },
  "grievanceId": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "epochId": "42",
  "accused": "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
  "phase_hint": "Open"
}
```

## See also

- [`USER_STORIES_MLN.md`](USER_STORIES_MLN.md) — User stories, coordination model, epoch semantics, wallet auto-route policy (PoC).
