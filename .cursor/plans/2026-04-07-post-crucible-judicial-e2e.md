# Post–Phase 12 judicial E2E — completed run

## Goal

After a Phase 12-style grievance in **`Open`** with **`stakeFrozen`**, exercise **Path 1:** Anvil time warp plus **`resolveGrievance`** through **`ResolvedSlash`; then **Path 2:** RPC stub E2E stack, taker handoff + optional grievance flow, and a **`Contested`** transition with **`mln-judge` in dry-run** (v1 decode limits per [`mln-judge/README.md`](../../mln-judge/README.md)). Intent matches the post-crucible playbook (external plan: post-crucible judicial E2E); this file is the in-repo outcome log.

## In scope / out of scope

**In scope:** `cast` / Foundry-in-Docker vs Anvil; permissionless **`resolveGrievance`**; [`scripts/e2e-mweb-handoff-stub.sh`](../../scripts/e2e-mweb-handoff-stub.sh) with **`E2E_MWEB_FULL=1`**; **`mln-cli`** route/forger/vault and **`grievance file`**; **`cast`** **`openGrievance` / `defendGrievance`** for a registered maker; **`mln-judge`** with **`JUDGE_DRY_RUN=1`**.

**Out of scope:** Changing contracts or the attached plan markdown under `~/.cursor/plans/`; persisting real secrets; claiming full §13 signature verification in **`mln-judge`** today.

## Primary files and canonical docs

- [`contracts/src/GrievanceCourt.sol`](../../contracts/src/GrievanceCourt.sol) — phases **`Open` / `Contested` / `ResolvedSlash`**, **`resolveGrievance`**, **`defendGrievance`**, **`adjudicateGrievance`**.
- [`scripts/grievance-e2e-anvil.sh`](../../scripts/grievance-e2e-anvil.sh) — **`evm_increaseTime`**, **`evm_mine`**, **`resolveGrievance`** pattern.
- [`mlnd/internal/litvm/accuser_resolve.go`](../../mlnd/internal/litvm/accuser_resolve.go) + [`accuser_watcher.go`](../../mlnd/internal/litvm/accuser_watcher.go) — accuser auto-resolve (event-order limitation if grievance predates watcher).
- [`scripts/e2e-mweb-handoff-stub.sh`](../../scripts/e2e-mweb-handoff-stub.sh), [`deploy/docker-compose.e2e.sidecar-rpc.yml`](../../deploy/docker-compose.e2e.sidecar-rpc.yml) — RPC sidecar + **`mw-rpc-stub`**.
- [`mln-judge/README.md`](../../mln-judge/README.md) — env vars, dry-run vs auto-adjudicate, stub decode note.
- [`research/THREAT_MODEL_MLN.md`](../../research/THREAT_MODEL_MLN.md) — grievance lifecycle narrative.

## Execution results

### Path 1 — Timeout slash

1. Read **`challengeWindow()`** from **`GrievanceCourt`** on local Anvil (observed **86400** seconds for the deployment used).
2. **`anvil_mine` / `evm_increaseTime`** past **`challengeWindow +` small buffer** (e.g. +2 s style per [`grievance-e2e-anvil.sh`](../../scripts/grievance-e2e-anvil.sh)), then **`evm_mine`**.
3. **`resolveGrievance(bytes32)`** submitted with a funded default Anvil caller (permissionless path documented in script comments).
4. **Observed:** **`grievances(grievanceId)`** returned **phase `3` (`ResolvedSlash`)**; accused **`stakeFrozen`** became **false** and **stake** **0** for that case’s accused (registry slash path ran).

### Path 2 — RPC stack + defense + judge

1. **Tear down** standalone mock E2E profile/volumes as needed so ports and networks are clean.
2. **`E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh`** — started **`mw-rpc-stub`**, brought up **compose base + sidecar-rpc override**, ran **`e2e-bootstrap.sh`**, started **makers**, **`mln-cli` pathfind + forger** with **batch/status** — completed successfully. Script tears down stub on exit; **stub was restarted manually** when continuing local tests against **`:8546`**.
3. **`mln-cli route build`**, **`forger`** with **`-vault`** against **RPC sidecar**: golden receipt path OK; **`grievance file`** produced an on-chain case whose **`accused`** was the **placeholder address** used by the stub receipt (**`0x00…01`**) — **not** a registered maker, so **`defendGrievance`** from an e2e maker key is **not** applicable to that specific filing (matches the plan’s note: correlators must align for maker auto-defend / receipt-in-DB story).
4. **Synthetic `Contested` row:** **`openGrievance`** via **`cast send`** (accuser = deploy default account, accused = **maker2** from bootstrap, test **`evidenceHash`**, bond **0.01 ether**), then **`defendGrievance`** with **maker2’s bootstrap private key** and minimal **`bytes`** calldata.
5. **Observed:** **`grievances`** **phase `2` (`Contested`)**; **`Contested`** event emitted.
6. **`mln-judge`** built from [`mln-judge/`](../../mln-judge/), run with **`JUDGE_DRY_RUN=1`**, **`JUDGE_PRIVATE_KEY`** matching contract **`judge`**, **`JUDGE_LITVM_WS_URL`** to Anvil — subscriber fired but **v1 defense tuple decode failed** on the intentionally short **`defenseData`** (**`length insufficient … require 32`**). This matches README expectations: structured decode is strict; **automated crypto verification is not the v1 story**.

## Verification

- **Slash:** `cast call GrievanceCourt grievances(bytes32)…` — phase **3** after resolve; registry **`stake` / `stakeFrozen`** on accused.
- **RPC e2e:** script prints **GET /v1/balance** and **forger** success; optional **`curl`** / **`mln-cli`** smoke per [`PHASE_3_MWEB_HANDOFF_SLICE.md`](../../PHASE_3_MWEB_HANDOFF_SLICE.md).
- **Contested:** `cast` grievances tuple phase **2** after **`defendGrievance`**.
- **Judge:** **`mln-judge`** logs show **Contested** handling path and decode outcome.

## Layer-boundary check

**LitVM** (Anvil) carried all court/registry state transitions; **MWEB** exercised only via **sidecar → stub RPC** handoff and **mln-cli** POSTs; **Nostr** only as far as the e2e compose makers advertise; **Tor** as localhost hop URLs in matrix — no stake truth on Nostr, no full mix verification on-chain (see [`AGENTS.md`](../../AGENTS.md)).

## Follow-ups

- For **grievance correlators** that match a **real registered maker**, adjust stub/fork receipt **`accusedMaker`** (or filing path) so **`openGrievance`** and receipt preimage align — then **`mlnd`** auto-defend + receipt store integration can be tested without **`cast`** shortcuts.
- For **`mln-judge`** happy-path decode, submit **`defenseData`** encoding the **v1 tuple** expected by [`mlnd/pkg/litvmevidence`](../../mlnd/pkg/litvmevidence) (see judge service and tests).
- **Compose cleanup:** dual-file stack tear-down:  
  `docker compose -f deploy/docker-compose.e2e.yml -f deploy/docker-compose.e2e.sidecar-rpc.yml --profile makers down` (add **`-v`** if you need volumes reset).
