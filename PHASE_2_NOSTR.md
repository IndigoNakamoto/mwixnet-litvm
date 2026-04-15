# Phase 2: Nostr wire profile (complete in-tree)

**Status:** **complete** for **v1** kinds **31250** (`mln_maker_ad`) and **31251** (`mln_grievance_pointer`) — normative spec, golden fixtures, Go clients, Python CLIs, and CI validation.

**Draft v2 (kind 31250 only):** sealed `reachability` for DoS-hardening is specified and fixture-validated in [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md); **Scout still expects cleartext `tor` for pathfind** until a decrypting client path lands.

**Canonical wire:** [`research/NOSTR_MLN.md`](research/NOSTR_MLN.md) (kinds, NIP-33 `d` tag, `content` JSON, `nostrKeyHash` binding to [`MwixnetRegistry`](contracts/src/MwixnetRegistry.sol)).

## What ships in this repo

| Area | Role |
| ---- | ---- |
| [`nostr/fixtures/valid/`](nostr/fixtures/valid/) | Golden JSON events (maker ad full + minimal + **v2 sealed draft**, grievance pointer). |
| [`nostr/validate_fixtures.py`](nostr/validate_fixtures.py) | Structural validation (tags, `litvm` block, hex shapes). |
| [`nostr/check_wire_helpers.py`](nostr/check_wire_helpers.py) | Asserts [`scripts/mln_nostr_wire.py`](scripts/mln_nostr_wire.py) output passes the same rules. |
| [`.github/workflows/nostr-fixtures.yml`](.github/workflows/nostr-fixtures.yml) | CI on `nostr/**`, `scripts/mln_nostr_wire.py`, and this workflow. |
| [`mln-cli`](mln-cli/) Scout | Subscribes to kind **31250**, parses via [`mlnd/pkg/makerad`](mlnd/pkg/makerad), filters by **`chainId`**, **`registry`**, optional **`grievanceCourt`**, then verifies **`makerNostrKeyHash`** on LitVM. |
| [`mlnd`](mlnd/) | Publishes replaceable maker ads from env (`MLND_REGISTRY_ADDR`, `MLND_COURT_ADDR`, …). |
| [`scripts/`](scripts/) | `mln_nostr_wire.py`, `publish_grievance.py`, demos — same shapes as the spec. |

## When registry or court addresses change

After **redeploy** (Anvil, testnet, or mainnet), **`content.litvm.registry`** and **`content.litvm.grievanceCourt`** in ads and grievance pointers **must** match the live deployment or **Scout** will reject events (`registry address mismatch` / `grievance court mismatch`).

1. **Fixtures (CI):** Update JSON under [`nostr/fixtures/valid/`](nostr/fixtures/valid/) so golden files stay aligned with the default Anvil layout you document, **or** document that fixtures use a fixed example chain and update them whenever the default deploy script changes contract addresses.
2. **Operators:** Set `MLND_REGISTRY_ADDR` / `MLND_COURT_ADDR` (and taker `MLN_*`) to the new addresses before republishing ads.
3. **Scripts:** Use `publish_grievance.py --broadcast-json …/run-latest.json` to pull registry + court from a Foundry broadcast file.

## Local validation

```bash
python3 nostr/validate_fixtures.py
python3 nostr/check_wire_helpers.py
```

Full stack with grievance smoke + Nostr tooling: `make test-full-stack` (see [`PHASE_12_E2E_CRUCIBLE.md`](PHASE_12_E2E_CRUCIBLE.md)). Relay walkthrough: [`research/E2E_NOSTR_DEMO.md`](research/E2E_NOSTR_DEMO.md).

## Related

- [`PHASE_10_TAKER_CLI.md`](PHASE_10_TAKER_CLI.md) — Scout / pathfind / forger.
- [`PHASE_5_NOSTR_TOR_BRIDGE.md`](PHASE_5_NOSTR_TOR_BRIDGE.md) — relay + bridge context.
- [`AGENTS.md`](AGENTS.md) — Nostr is discovery only; LitVM is stake authority.
