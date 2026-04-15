# MLN Nostr wire profile (Phase 2)

**Status:** **normative v1** for kinds **31250–31251** — discovery and gossip only. Stake and grievance truth live on **LitVM** ([`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) §5–6). This document is the canonical place for **event kinds**, **`content` JSON**, and **identity binding** to [`MwixnetRegistry`](../contracts/src/MwixnetRegistry.sol).

**v2 (draft, pre-normative):** [§ Maker ad wire v2 (draft)](#maker-ad-wire-v2-draft) describes **sealed reachability** and relay-policy mitigations for **cleartext `.onion` / mix keys on public relays** (availability / targeted DoS). Until that section is promoted to normative text, implementations **SHOULD** keep shipping **v1** for interoperability; clients that only implement v1 will **ignore or reject** v2 payloads per their policy.

**In-repo closure:** Golden fixtures and CI validation live under [`nostr/`](../nostr/); playbook and operator notes: [`PHASE_2_NOSTR.md`](../PHASE_2_NOSTR.md).

## Principles

1. **Nostr is transport-only.** Relays see what you publish; design for that ([`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) §6.4, threat model: discovery surveillance).
2. **LitVM is authoritative for stake** — `stake`, `makerNostrKeyHash`, freeze state, and `GrievanceCourt` storage. Wallets **must** confirm balances and cases via RPC, not Nostr.
3. **Payloads are Litecoin/MWEB/LitVM–oriented** where relevant; Nostr carries **pointers and ads**, not MWEB transaction bodies.
4. **Sensitive paths** may use encrypted DMs (NIP-04 / NIP-44) or out-of-band channels; **v1 does not require** encryption for public maker ads. **Draft v2** ([§ Maker ad wire v2 (draft)](#maker-ad-wire-v2-draft)) standardizes an optional **NIP-44-shaped** sealed block for reachability; see also [`research/THREAT_MODEL_MLN.md`](THREAT_MODEL_MLN.md) (discovery / DoS row).
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

## Maker ad wire v2 (draft)

**Goal:** Keep **LitVM pointers** (`litvm`) visible so **Scout + `makerNostrKeyHash` binding** stay unchanged, while **not** publishing cleartext **Tor mix API URLs** or **`swapX25519PubHex`** to every relay when operators choose hardened discovery.

**Threat addressed:** Anyone with relay read access learns **live `.onion` + port** (and optional X25519 material) and can **target makers with connection exhaustion or RPC spam**; small maker sets and epoch-shaped traffic make timing cheap ([`THREAT_MODEL_MLN.md`](THREAT_MODEL_MLN.md)).

**Normative status:** **Draft only** — field names and required combinations may change before lock-in. **v1** remains the interoperability baseline.

### v2 `content` object (kind 31250 unchanged)

- **`v`:** MUST be **`2`**.
- **`litvm`:** Same object and rules as v1 (required).
- **`fees`**, **`capabilities`:** Same optional semantics as v1.
- **Reachability (exactly one style per event):**
  1. **Sealed (preferred for public relays):** `reachability` object (below). In this mode the top-level **`tor`** and **`swapX25519PubHex`** fields MUST be **absent** or **empty strings** (parsers SHOULD treat whitespace-only as empty).
  2. **Cleartext (legacy within v2):** Omit `reachability` and populate **`tor`** and/or **`swapX25519PubHex`** as in v1. Intended for **closed or operator-controlled relays**, not as a substitute for sealing on **public** relays.

### `reachability` object (sealed style)

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `scheme` | string | yes | **`nip44-v2`** today — NIP-44 version 2 payload string ([NIP-44](https://github.com/nostr-protocol/nips/blob/master/44.md)). |
| `ciphertext` | string | yes | Opaque **NIP-44 v2** ciphertext (ASCII string as produced by a NIP-44 implementation). **Not** a second JSON layer on the wire. |

**Plaintext inside the ciphertext (informative, for implementers):** After the recipient decrypts with an agreed keying path (see below), JSON UTF-8 with the **same optional keys as v1**:

```json
{ "tor": "http://....onion:port", "swapX25519PubHex": "...." }
```

Both keys remain optional inside the plaintext, but **production** mix ads SHOULD include whatever v1 would have published for dialing and onion handoff.

**Keying (out of band for this wire doc):** NIP-44 requires a shared secret between encryptor and decryptor. Real deployments MUST document one or more of:

- **Invite-only / allowlisted relays** plus **pairwise** NIP-44 (coordinator or taker ephemeral key distributed through another channel).
- **Group / epoch keys** distributed after takers commit (still does not help fully open “browse all makers” without *some* gating step).
- **NIP-59 Gift Wrap** as an **outer** wrapper (kind **1059**) around a **rumor** that still carries kind **31250**: relays see ciphertext and **recipient hints**, not cleartext onions — suitable when the **reader set is bounded**; it is **not** a magic “public broadcast to unlimited anonymous readers” primitive by itself.

Clients that cannot decrypt **MUST** skip the maker for route construction (same as missing `tor` today).

### Relay and transport mitigations (non-normative)

- **AUTH / private relays**, **rate limits** on **`mln-sidecar`** and maker JSON-RPC, and **Tor** tuning are **defense in depth**, not replacements for hiding reachability from global readers when that is the deployment goal.

### Verification recipe (extension for v2)

1–3. Same as v1 (filter, parse JSON, verify Schnorr).
4. If `content.v == 2` and `reachability` is present: enforce **no cleartext** `tor` / `swapX25519PubHex`; verify `scheme == "nip44-v2"` and non-empty `ciphertext`; then decrypt out of band per deployment policy and validate inner JSON shape.
5. If `content.v == 2` and `reachability` is absent: fall back to v1-style checks for optional cleartext `tor` / `swapX25519PubHex`.
6. **`nostrKeyHash` / registry** checks unchanged (still use rumor pubkey `P` for kind **31250**).

## Relay policy (normative recommendations)

Run or partner with **1–3 small, trusted NIP-42 relays** (ideally with a **pubkey allowlist** in the relay config). Do **not** rely on random public relays for ads that carry cleartext `tor` or `swapX25519PubHex` fields.

**NIP-42 AUTH** ([NIP-42](https://github.com/nostr-protocol/nips/blob/master/42.md)) requires WebSocket clients to sign a kind-22242 challenge before the relay serves events. When enabled:

- **Makers** (`mlnd`): set `MLND_NOSTR_AUTH=true`. The broadcaster signs the relay's challenge with the same `MLND_NOSTR_NSEC` key and fails fast (backoff + retry) if AUTH is rejected.
- **Takers** (`mln-cli` / Scout): set `MLN_NOSTR_AUTH_NSEC` to an ephemeral Nostr key (or the taker's key). Scout creates a `SimplePool` with `WithAuthHandler` so subscriptions succeed on AUTH-required relays.
- **Dashboard** self-check: when `MLND_NOSTR_AUTH=true` is set, the dashboard pool also authenticates using the broadcaster's key.

**What AUTH does and does not do:**

- AUTH raises the bar from "anyone with a scraper" to "anyone with an allowed key."
- Without a relay-side **pubkey allowlist**, AUTH only blocks anonymous scrapers (anyone can generate a key and authenticate). Combine AUTH with a limited allowlist for meaningful access control.
- AUTH is **not** end-to-end encryption; it is transport-level gating. See **draft v2** (`reachability`) for a future cryptographic layer.

**Relay software:** [nostr-rs-relay](https://github.com/scsibug/nostr-rs-relay) supports NIP-42 via `[authorization]` in its TOML config (see `deploy/nostr-rs-relay.toml` for an example).

## Future (non-normative)

- **Promote v2** to normative after at least one interoperable implementation (publish + Scout + pathfind) and fixture lock-in.
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

**Maker ad v2 `content` (sealed style — draft; `ciphertext` is a placeholder, not a valid NIP-44 string):**

```json
{
  "v": 2,
  "litvm": {
    "chainId": "31337",
    "registry": "0x5fbdb2315678afecb367f032d93f642f64180aa3",
    "grievanceCourt": "0xe7f1725e7734ce288f8367e1bb143e90bb3f0512"
  },
  "fees": { "unit": "sat_per_hop", "min": 1, "max": 99 },
  "capabilities": ["mweb-coinswap-v0"],
  "reachability": {
    "scheme": "nip44-v2",
    "ciphertext": "nip44v2-placeholder-not-valid-ciphertext"
  }
}
```

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

- [`PHASE_2_NOSTR.md`](../PHASE_2_NOSTR.md) — Phase 2 playbook (fixtures, CI, Scout, address rotation).
- [`USER_STORIES_MLN.md`](USER_STORIES_MLN.md) — User stories, coordination model, epoch semantics, wallet auto-route policy (PoC).
