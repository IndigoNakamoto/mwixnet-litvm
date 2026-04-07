# Stub correlator + auto-defend E2E (completed)

## Goal

Align **`mw-rpc-stub`** golden LitVM receipts so **`accusedMaker`** matches **`route[0].operator`** (parity with **`MockBridge`**), document the shared-vault and **start-`mlnd`-before-`grievance`** ordering needed for maker auto-defend, and add an optional host script plus **`mln-judge`** verification notes—so local RPC-path flows can close correlators with registered makers without cast-only workarounds.

## In scope / out of scope

**In scope:** `submitRouteBody` hop **`operator`**, **`goldenReceiptAccusedMaker`**, table tests under **`cmd/mw-rpc-stub`**, **`go test ./...`** in **`mln-sidecar/`**, doc updates in **`PHASE_3_MWEB_HANDOFF_SLICE.md`** and cross-link in **`PHASE_12_E2E_CRUCIBLE.md`**, **`scripts/grievance-correlated-stub-e2e.sh`**, **`mln-judge`** / **`JUDGE_DRY_RUN`** wording for real **`BuildDefenseData`** vs toy calldata.

**Out of scope:** pathfind randomness or forcing N1 in production; **`coinswapd`** real forward-failure correlators (separate backlog).

## Primary files and canonical docs

- `mln-sidecar/cmd/mw-rpc-stub/main.go` — hop **`Operator`** JSON + golden receipt wiring.
- `mln-sidecar/cmd/mw-rpc-stub/receipt_accused.go` — **`goldenReceiptAccusedMaker`** (fallback **`0x000…0001`**).
- `mln-sidecar/cmd/mw-rpc-stub/main_test.go` — JSON table tests.
- `scripts/grievance-correlated-stub-e2e.sh` — **`e2e.generated.env` → MLN_***, **`route build`**, optional **`CORRELATED_RUN_FORGER=1`**, printed **`mlnd`** / **`grievance file`** hints.
- `PHASE_3_MWEB_HANDOFF_SLICE.md` — correlator + auto-defend subsection; **`mln-judge`** note.
- `PHASE_12_E2E_CRUCIBLE.md` — cross-link to Phase 3 + script.
- Correlator rules: `mlnd/pkg/litvmevidence/defense.go` **`ValidateReceiptForGrievance`**; mock parity `mln-sidecar/internal/mweb/bridge.go`.
- Product / layers: `AGENTS.md`, `PRODUCT_SPEC.md` appendix 13 (unchanged economics).

## Execution results

- **Stub:** Golden receipt **`accusedMaker`** is **`strings.ToLower(common.HexToAddress(op).Hex())`** when hop 0 **`operator`** is a valid 20-byte hex address (optional **`0x`**); otherwise legacy **`0x0000000000000000000000000000000000000001`** for minimal RPC payloads without **`operator`**.
- **Docs:** Documented shared SQLite (**`mln-cli forger -vault`** and **`MLND_DB_PATH`**), N1 = **`jq -r '.hops[0].operator'`**, watcher subscription ordering (**`mlnd` before `openGrievance`**), accuser/epoch envs; **`mln-judge`** decodes only real v1 **`BuildDefenseData`** (not stub **`0xdeadbeef`** bytes).
- **Script:** Executable helper maps bootstrap **`E2E_`** vars to **`MLN_*`**, runs **`route build`**, optional forger with vault/batch; does not write private keys (references **`e2e.maker*.env`** / operator key export in comments).

## Verification

- `cd mln-sidecar && go test ./...`
- `make build-mw-rpc-stub` (from repo root)

## Layer-boundary check

**MWEB:** stub/sidecar receipt JSON shape only. **LitVM:** existing correlators and contracts unchanged. **Nostr** / **Tor:** unchanged. No new on-chain economics.

## Follow-ups

- Run full docker stack + **`CORRELATED_RUN_FORGER=1`** + host **`mlnd`** + **`grievance file`** + **`mln-judge`** **`JUDGE_DRY_RUN=1`** as an operator exercise (not required for the code/doc deliverable).
