# MLN stack — code review and threat model (accepted snapshot)

**Date:** April 2026  
**Codebase state:** `main` post-Phase 15; Phase 3a + integration slice + **completed swap path** on **`mw-rpc-stub`** (`mweb_runBatch` / `mweb_getRouteStatus`, sidecar **`/v1/route/*`**, E2E **`E2E_MWEB_FULL=1`**) plus **real funded operator runbook** (`E2E_MWEB_FUNDED=1`, **`swapX25519PubHex`** via bootstrap, optional **`-mweb-dev-clear-pending-after-batch`** — DEV ONLY) as of 2026-04  
**Status:** Reviewed and accepted by the team (Harper, Benjamin, Lucas, Indigo). Findings were validated line-by-line against the repo; no material inaccuracies noted. This document preserves the audit and threat model for the repository record.

**Related:** [`README.md`](../README.md) (roadmap and scaffold disclaimers), [`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md), phase docs at repo root, [`AGENTS.md`](../AGENTS.md), adversarial narrative [`RED_TEAM_MLN.md`](RED_TEAM_MLN.md).

---

## Table of contents

1. [Code review / audit report](#1-code-review--audit-report)
2. [Threat model tables](#2-threat-model-tables)

---

## 1. Code review / audit report

**Repository:** `mwixnet-litvm` (MLN: MWEB + LitVM + Nostr + Tor)  
**Review type:** Architecture alignment, security-relevant implementation, CI coverage, operational risks  
**Assumption:** Code is **pre-production** (README and contracts state “not audited”).

### 1.1 Executive summary

The repo delivers a **coherent split** across layers: Solidity for registry and grievance **scaffold**, Go for **maker daemon (`mlnd`)**, **taker CLI (`mln-cli`)**, and **HTTP sidecar (`mln-sidecar`)**, plus scripts/fixtures for Nostr and local Anvil.

**Strengths**

- Clear separation: LitVM does not try to verify MWEB execution on-chain; evidence hashes are defined in `EvidenceLib` and mirrored in Go (`mlnd/internal/litvm/defense.go`).
- **`mlnd` auto-defend** validates receipts against `GrievanceOpened` before building `defenseData` and submitting `defendGrievance`.
- **Desktop wallet** warns when the sidecar URL is not localhost / `.onion` over cleartext HTTP.
- **SQLite** receipt storage uses parameterized queries.
- **CI** runs Foundry tests on contract changes, Go tests on `mlnd` / `mln-cli` / `mln-sidecar` changes, Nostr fixture validation, and a **main-branch** full-stack Anvil + grievance + Nostr script path.

**Highest-impact gaps (must fix before treating as economic security)**

1. **`GrievanceCourt` does not verify `defenseData` contents on-chain** — accused can call `defendGrievance` with arbitrary bytes (phase → **Contested**). **Exoneration** (accuser bond to accused) requires the deployment **`judge`** to call **`adjudicateGrievance(..., true)`**; permissionless **`resolveGrievance`** only **slashes** when the accused stayed **Open** past the deadline — closing the prior **bond-theft** vector from bogus defense + `resolve`. **Integrity of rulings** still depends on **interim judge** behavior (or future on-chain verifier); not audited.
2. **`mln-sidecar` default server binds `0.0.0.0`** with **no authentication** — any process that can reach the port can POST routes, trigger batch, read route status, and read mock balance (local-dev appropriate; dangerous if exposed).
3. **Operator / taker secrets on disk**: wallet stores **`OperatorEthPrivateKeyHex` in plaintext JSON** under the user config dir (`0o600` file, `0o700` dir) — documented in code comments but still a **high-value target** for malware or backups.
4. **Forger → sidecar HTTP** uses default `http.Client` with **no TLS pinning** — acceptable for loopback; risky if extended to remote URLs without HTTPS and strong trust model.

### 1.2 Scope and methodology

**In scope**

- `contracts/src/*.sol` (focused: `MwixnetRegistry`, `GrievanceCourt`, `EvidenceLib`)
- `mlnd/` (watcher, defender, defense validation, store, Nostr broadcaster entrypoints)
- `mln-cli/` (forger client, pathfind, scout, config, Wails `desktop/`)
- `mln-sidecar/`
- `.github/workflows/*.yml`
- Cross-check with `README.md` roadmap claims

**Out of scope / not deeply reviewed**

- Full `PRODUCT_SPEC.md` line-by-line vs every behavior
- Python CLIs beyond CI presence
- Patched `coinswapd` fork behavior (patch file referenced, not audited in this snapshot)
- Cryptographic correctness of MWEB / onion construction (delegated to `coinswapd` / spec)

### 1.3 Architecture and spec alignment

| Layer | Intended role (spec / AGENTS) | Repo behavior |
|--------|-------------------------------|---------------|
| MWEB | Privacy + per-hop fees | Not implemented here; sidecar mocks or forwards |
| LitVM | Registry, stake freeze, grievance lifecycle | Contracts + `mlnd` watcher/defender |
| Nostr | Discovery, signed ads | Kind 31250 broadcaster + scout; fixtures in CI |
| Tor | Transport | Tor URLs in ads; **`mln-cli pathfind`** requires non-empty maker **`tor`**; forger normalizes bare **`*.onion`** to **`http://`** for RPC dial (no live Tor probe in reviewed code; **`coinswapd`** may need operator proxy for `.onion`) |

**Alignment:** Layer boundaries are mostly respected. **Tension to track:** README Phase 3 (“end-to-end integration”) is still open (live `.onion` round + L2 slash path on public LitVM); local E2E defaults to **`-mode=mock`** on **`mln-sidecar`**; **`-mode=rpc`** forwards to in-repo **`research/coinswapd/`** (**`mweb_submitRoute`** / **`mweb_getBalance`**) with stub + host smoke verified — full swap completion and production hardening remain open.

### 1.4 Smart contracts

**`MwixnetRegistry.sol`**

- **Positives:** `setGrievanceCourt` is one-shot; freeze/unfreeze only from court; registered makers use **exit queue** with **open grievance** guard on `requestWithdrawal` / `withdrawStake`.
- **Notes:** Central `owner`; operational dependency on correct `cooldownPeriod` vs epoch/window (commented in contract). Native ETH-style `call` for withdrawals follows **checks-effects-interactions** adequately for the reviewed paths.

**`GrievanceCourt.sol`**

- **Flow:** `defendGrievance` sets **`Contested`** (emits digest of calldata). **`resolveGrievance`** applies **only** to **`Open`** cases after `deadline` (timeout slash). **`adjudicateGrievance`** (**`judge`** only) resolves **`Contested`** → slash (false) or exonerate (true).
- **Finding:** `defenseData` is still **not semantically verified** on-chain; **correctness** of exoneration vs uphold depends on **`judge`** (v1 interim trust) until a **verifier** or stronger oracle exists. **Judge** key compromise or policy failure is an operational risk — document deployment **`INTERIM_JUDGE`** explicitly for testnet/mainnet (see [`contracts/.env.example`](../contracts/.env.example)).

**`EvidenceLib.sol`**

- **Positives:** Small, testable surface; `evidenceHash` and `grievanceId` match the documented packed encoding approach.
- **Note:** Preimage is fixed-width fields; no dynamic-type `abi.encodePacked` ambiguity in the shown layout.

**Contract testing**

- **CI:** `contracts.yml` runs `forge build` and `forge test` in Docker on `contracts/**` changes, and runs **Slither** (Crytic action) on those paths. Foundry **fuzz / invariant** tests also exercise registry economics (e.g. stake invariants). Continue to **triage** new Slither findings like any static signal—not a substitute for audit.

### 1.5 Maker daemon (`mlnd`)

**Evidence and defense pipeline**

- **`ValidateReceiptForGrievance`** checks accuser, epoch, accused, and recomputes `evidenceHash` / `grievanceId` against the log — strong consistency between SQLite row and chain event **before** building defense calldata.
- **`BuildDefenseData`** ABI-encodes a versioned tuple for opaque on-chain submission — good forward-compatibility pattern **if** the contract eventually validates it.

**Keys and transactions**

- **`LoadDefenderFromEnv`:** Requires `MLND_DEFEND_AUTO` and `MLND_OPERATOR_PRIVATE_KEY`; **derived address must match** `MLND_OPERATOR_ADDR` — prevents accidental wrong-key submit.
- **`SubmitDefend`:** Retries on **transport** errors, not on reverts — reasonable.
- **Logging:** DRY-RUN logs **full `defenseData` hex** — operational leak surface (metadata / correlators in logs); not the private key, but sensitive in adversarial log environments.

**Watcher / storage**

- **SQLite:** Parameterized `INSERT` — no obvious SQL injection from NDJSON fields.
- **Bridge:** NDJSON ingestion — trust boundary is **who can write files** in `MLND_BRIDGE_RECEIPTS_DIR`; treat directory permissions as security-critical.

**Nostr broadcaster**

- Loads **`MLND_NOSTR_NSEC`** when relays are configured; same process may hold **LitVM operator key** if auto-defend is on — **single host compromise** exposes both identities.

### 1.6 Taker CLI (`mln-cli`) and desktop wallet

**Scout / registry verification**

- Verifies Schnorr signatures and filters by deployment tags (`chainId`, registry, optional court) — aligns with `research/NOSTR_MLN.md` intent.
- **Relays are trusted for availability and censorship** — expected for Nostr; document for operators.
- **Cleartext Tor / mix keys in v1 kind 31250** are visible to **any relay reader**; see **draft v2** sealed reachability in [`NOSTR_MLN.md`](NOSTR_MLN.md) (“Maker ad wire v2 (draft)”).

**Pathfind**

- **`math/rand`** seeded from time — **fine for tie-breaking**, not for security decisions. If route selection ever implied secrets, switch to `crypto/rand`. (Team confirmed: intentional for PoC policy.)

**Forger / HTTP client**

- POSTs JSON route + destination + amount to configurable URL; **no auth**, **no TLS options** in `NewSidecarClient`.
- **Desktop mitigation:** `warnNonLoopback` blocks non-local cleartext sidecar URLs (allows `.onion`).

**Wallet settings persistence**

- Plaintext secp256k1 key in `settings.json` with `0o600` — **acceptable only with explicit user threat model** (Phase 14 scoped “accept for now, tighten later”). Recommend **OS keychain integration** or **file encryption with user password** before mainnet-style use.

### 1.7 `mln-sidecar`

- **Endpoints:** `GET /v1/balance` (mock constants), `POST /v1/swap` (validates JSON, `DisallowUnknownFields`, mock onion in default path).
- **Binding:** `Addr: ":port"` → **all interfaces**.

**Finding (medium, deployment):** On a shared machine or misconfigured firewall, this is an **open local RPC**. Bind to loopback by default or require explicit host flag for LAN use.

**Finding (low):** Mock success copy may imply injection into `coinswapd` queue — clarify for operators when running mock mode.

### 1.8 CI / regression coverage

| Workflow | Trigger | Coverage |
|----------|---------|----------|
| `contracts.yml` | `contracts/**` | `forge build` / `test` |
| `mlnd.yml` | `mlnd/**`, `mln-cli/**`, `mln-sidecar/**` | `go test ./...` (CGO for `mlnd`) |
| `nostr-fixtures.yml` | `nostr/**` | Python fixture validation |
| `test-full-stack.yml` | **main** PR/push only | Anvil deploy, `make test-grievance`, `make test-full-stack` |

**Gaps**

- **`make test-operator-smoke`** (golden NDJSON → bridge → grievance) does **not** appear in the default CI matrix — risk of regressions unless run manually. **P1:** add to CI.
- **Wails / `wails` build tag** — not evident in `mlnd.yml`; desktop builds may lack CI coverage.
- **Cross-package changes** (e.g. only `scripts/` or `Makefile`) may skip Go/contract jobs until main full-stack runs.

### 1.9 Prioritized recommendations (mapped to Phase 15+)

**P0 — Before any real stake / testnet money**

1. **Redesign `GrievanceCourt`** (or document immutably as non-production): on-chain **verification hook** for `defenseData`; **review** bond/stake edge cases under Phase 15 economics (implemented but not audited). Do not market **full judicial integrity** until `defenseData` is enforced.
2. **Sidecar:** default bind **127.0.0.1**; document firewall; add optional **auth** or Unix socket for local coupling to `coinswapd`.

**P1 — Security hardening**

3. Wallet: **keychain** or encrypted store; minimize logging of defense-related blobs in `mlnd` production configs.
4. Add **operator-smoke** (and optionally **Wails build**) to CI.

**P2 — Quality / clarity**

5. Document **`math/rand`** as tie-break-only (or switch if policy becomes security-sensitive).
6. **Slither** is already enforced in **`.github/workflows/contracts.yml`** on `contracts/**` changes; keep the job green and file issues for any new high-severity findings after triage.

### 1.10 Conclusion

The codebase is a **credible research and integration scaffold**: **`mlnd`**’s receipt validation and defense encoding are thoughtfully aligned with **`EvidenceLib`**, and the **taker path** is honest about delegating MWEB work to **`coinswapd`/sidecar**. The **LitVM judicial layer** runs **Phase 15 economics** (slash and bond transfers on-chain) but **still lacks verification of `defenseData`** — so it is **not yet a sound integrity mechanism** for production disputes; README already flags **not audited**, and the implementation confirms the verifier gap. **Operational security** (sidecar exposure, plaintext wallet keys, shared-process keys) should be treated as **blocking** for production deployment until addressed or explicitly accepted with user-facing disclosure.

---

## 2. Threat model tables

**Legend**

- **Adversary:** who can plausibly mount the attack.
- **Residual risk:** what remains after intended controls (often “accept”, “monitor”, or “fix later”).

### 2.1 Primary threat table

| Asset / surface | Threat | Adversary | Scenario | Impact | Existing controls | Residual risk |
|-----------------|--------|-----------|----------|--------|-------------------|---------------|
| **LitVM stake (`MwixnetRegistry`)** | Tampering / elevation | Malicious or compromised **registry `owner`** | `owner` misconfigures or upgrades policy off-spec (if upgradeability is added later) | Stake lock-in, unfair freeze, griefing | Single deployer `owner`; immutable params in current design | **High** if owner key compromised; **low** for fixed immutable deploy |
| **Stake freeze / unfreeze** | Tampering | **Anyone** calling registry directly | N/A — only `GrievanceCourt` may freeze/unfreeze | Unauthorized freeze | `onlyGrievanceCourt` modifier | **Low** if court address correct |
| **`GrievanceCourt` outcomes** | Tampering / repudiation | **Accused maker** | Calls `defendGrievance` with garbage → **`Contested`**; **no** permissionless path to exoneration | Cannot auto-steal accuser bond via `resolve` alone | **`judge`** must **`adjudicate`** to exonerate | **High** for ruling integrity vs receipts until `defenseData` verified on-chain or judge process is audited |
| **`GrievanceCourt` economics** | Tampering | **Accuser / accused / judge** | Timeout slash vs **`adjudicate`** uphold/exonerate moves stake and bonds | Phases may not match receipt reality if **judge** errs or is compromised | Interim **`judge`** + Phase 15 split; README not audited | **High** for end-to-end **integrity** vs receipts; bond-theft via bogus defend + `resolve` **closed** |
| **False grievance with wrong preimage** | DoS / griefing | **Any funded accuser** | Opens grievances with incorrect `evidenceHash` vs real mix | Maker stake frozen while open; accuser pays bond | `grievanceBondMin`; per-grievance bond | **Medium** — griefing cost vs freeze harm; court logic does not validate preimage on-chain |
| **`mlnd` SQLite vault** | Tampering / info disclosure | **Local user / malware** on host | Read or replace `mlnd.db` | Fake or leaked hop receipts | OS file permissions; DB not encrypted in app | **Medium** on shared hosts; backup leakage |
| **NDJSON bridge directory** | Tampering / elevation | **Any writer** to `MLND_BRIDGE_RECEIPTS_DIR` | Inject line matching a future grievance’s correlators | Bad receipt stored; wrong defense or failed validation | `mlnd` validates receipt vs on-chain event before defend | **Medium** — writer must predict/know correlators; filesystem permissions are the real gate |
| **`mlnd` LitVM operator key** | Info disclosure / elevation | **Malware, insider, backup theft** | Steal `MLND_OPERATOR_PRIVATE_KEY` | `defendGrievance` as maker; drain gas; nonce fights | Key only in env; address match check | **High** — hot key on disk/env is standard ops risk |
| **`mlnd` Nostr key (`MLND_NOSTR_NSEC`)** | Spoofing / repudiation | Same as above | Publish fake maker ads for bound operator identity | Takers scout wrong Tor/fees; reputation harm | Ads still need **registry** `makerNostrKeyHash` match for Scout | **Medium** — cannot steal stake via Nostr alone; hurts discovery |
| **Scout + registry verification** | Spoofing | **Relay + fake events** | Publishes events that fail Schnorr or registry checks | None for “verified” set | Schnorr verify; `eth_call` to registry | **Low** for verified path |
| **Scout + sybil makers** | Tampering | **Many funded makers** | Flood relays with cheap registered makers | Bad routes, censorship of good ads | Min stake on LitVM; fee/stake policy in pathfind | **Medium** — economic not technical |
| **Nostr relays** | DoS / censorship | **Relay operator** | Drop or delay 31250/31251 | Poor discovery; stuck routes | Multi-relay; user-configured | **Medium** — inherent to Nostr |
| **Cleartext `tor` / `swapX25519PubHex` in public kind 31250 (v1)** | DoS / availability | **Anyone** scraping relays | Target advertised `.onion` mix APIs (SYN / slowloris / RPC spam) during epochs | Failed batches; collapsed anonymity set if makers drop | **NIP-42 AUTH** relay hardening (`MLND_NOSTR_AUTH` / `MLN_NOSTR_AUTH_NSEC`); draft v2 sealed reachability; private relays; sidecar rate limits ([`NOSTR_MLN.md`](NOSTR_MLN.md) relay policy + v2 draft) | **Medium–High** — AUTH raises the bar from "anyone with a scraper" to "anyone with an allowed key"; **High** remains on public relays without AUTH or allowlist |
| **`mln-cli` Forger → sidecar** | Tampering / info disclosure | **LAN attacker / malicious sidecar** | MITM cleartext HTTP to non-loopback URL | Fake “ok”; exfil route + dest + amount | Wallet warns on non-localhost HTTP; CLI has no equivalent | **High** if user bypasses warning or uses CLI remotely over HTTP |
| **`mln-sidecar` HTTP** | Tampering / DoS | **Network peer** if port exposed | POST arbitrary swaps; hammer endpoint | Fake success; resource exhaustion | Validation of JSON body; timeouts | **High** if bound `0.0.0.0` without firewall; **Low** on loopback-only |
| **Desktop `settings.json`** | Info disclosure | **Malware, forensics, backup** | Read `OperatorEthPrivateKeyHex` | Impersonate maker on LitVM; self-route abuse | Dir `0o700`, file `0o600`; comment warns user | **High** — plaintext secret on disk |
| **Path selection (`pathfind`)** | Tampering (policy) | **Colluding makers** | Manipulate fee hints / stake visibility | Cheaper surveillance routes or biased hops | Registry stake + frozen flag checked in Scout | **Low–medium** — policy game, not crypto break |
| **`coinswapd` (stock)** | Tampering | **Remote mix API** | Behavior bugs, logging, metadata | Privacy loss, failed mixes | Tor; spec alignment | **High** — out-of-repo; must track fork |
| **Patched `coinswapd` NDJSON** | Tampering | **Compromised fork or flags** | Wrong `epochId` / `accuser` / correlators merged into line | Accuser/receipt mismatch; failed defense or wrong defense | `ValidateReceiptForGrievance` ties line to log | **Medium** — trust in fork + flag discipline |
| **`swap_forward` failure receipts (MLN fork)** | Griefing / policy | **Taker** | `mweb_getLastReceipt` attributes fault to downstream **LitVM `operator`** from the route; correlators bind peel + forward payload | **Dial** failures can look like **application** rejections; `signature` may be testnet sentinel **`unsigned-swap-forward-failure-v1`** (not a maker proof) | Automated adjudicators (**`mln-judge`**) and operators must treat the sentinel as **explicitly unsigned** until PRODUCT_SPEC appendix 13 defense hardening; document fairness limits for dial-vs-RPC classes | **Medium** — cryptographic binding improves evidence vs stubs; policy still governs interpretation |
| **Anvil / local scripts** | Elevation | **Developer** running scripts | Supply-chain in pip/Docker images | Compromised dev machine | Pinned images in docs; CI uses Foundry action | **Low** for end users; **ops** for devs |
| **CI (`test-full-stack`)** | DoS / supply chain | **PR author on `main`** | Broken tests block merges | Delay release | Gated to `main` | **Low** |

### 2.2 Compact adversary-capability view

| Capability | Touches | Typical impact |
|------------|---------|----------------|
| **LitVM RPC observer** | All `eth_call` / txs from `mln-cli` / `mlnd` | Metadata (who queries whom); no key material if HTTPS/WSS used |
| **Nostr relay** | Scout, broadcaster | Censorship, timing; cannot forge verified ads without matching registry binding |
| **Local host compromise** | Keys, DB, bridge dir, sidecar | **Full** taker/maker operational compromise |
| **Smart contract attacker** | On-chain calls | **Griefing** and **bogus defense** within current court rules; **not** full stake drain via reviewed registry paths without further bugs |

### 2.3 How to use this document

- Treat **High** rows on **`GrievanceCourt` / judge** as **integrity** blockers for “trustless court” claims until **`defenseData` verification** or stronger policy exists. Re-read [`GrievanceCourt`](../contracts/src/GrievanceCourt.sol) / [`MwixnetRegistry`](../contracts/src/MwixnetRegistry.sol) when triaging.
- Treat **“High” operational** rows as **deployment checklists**: loopback bind, firewall, key storage, TLS for any non-local sidecar.

---

## Document history

| Date | Note |
|------|------|
| 2026-04 | Initial commit: external audit narrative + threat tables, team acceptance recorded. |
| 2026-04 | Doc sync: Slither + invariant tooling reflected as CI-enforced; codebase state bumped post-Phase 15. |
| 2026-04 | [`RED_TEAM_MLN.md`](RED_TEAM_MLN.md) added; §1.1 / §1.4 / table “GrievanceCourt economics” aligned with Phase 15 slash and exoneration bond (bond no longer refunded to accuser on exonerate). |
| 2026-04 | Doc sync: **`mln-cli maker onboard`** (operator LitVM txs) and **`mlnd` loopback dashboard** (`MLND_DASHBOARD_ADDR`, optional token) noted as new operator surfaces alongside existing hot-key / sidecar rows. |
| 2026-04 | Phase 3 integration slice: **`research/coinswapd`** build target + optional host smoke vs **`mln-sidecar -mode=rpc`**; hop **`tor`** URL normalization and pathfind **Tor-required** filtering — operator proxy (`HTTP_PROXY` / Tor SOCKS) still required for live `.onion` RPC dials inside **coinswapd**. |
| 2026-04 | Doc sync: §1.3 Tor row + §1.3 alignment paragraph vs shipped **`pathfind`** Tor requirement and in-tree **`coinswapd`** fork (vs “fork required” wording). |
| 2026-04 | **`mln-sidecar`** adds **`GET /v1/route/status`** and **`POST /v1/route/batch`** (same trust boundary as **`/v1/swap`**: unauthenticated HTTP; loopback-only in ops guidance). |
| 2026-04 | Doc sync: header **codebase state** + §1.1 sidecar gap text for **completed swap path** (`mweb_runBatch` / status); **`AGENTS` / README / PRODUCT_SPEC §9** parity. |
| 2026-04 | Fork **`research/coinswapd`**: **`-mweb-dev-clear-pending-after-batch`** (DEV ONLY) clears onion DB after **`mweb_runBatch`** without **`SendTransaction`** — operator misconfiguration risk if enabled on production nodes; see [`COINSWAPD_MLN_FORK_SPEC.md`](COINSWAPD_MLN_FORK_SPEC.md) §2.7a. |
| 2026-04 | **Hop-failure receipts:** LitVM metadata routes require per-hop **`operator`**; **`mweb_getLastReceipt`** + **`GET /v1/route/receipt`** poll async **`swap_forward`** failures; testnet receipt **`signature`** may be **`unsigned-swap-forward-failure-v1`** (sentinel, not maker-signed — see new threat row). |
| 2026-04 | Doc sync: threat model **codebase state** line reflects **`feat(mweb): real funded swap path`** (committed operator E2E + runbook; production **`pendingOnions=0`** still implies real **`finalize`** / P2P per Phase 3 doc). |
