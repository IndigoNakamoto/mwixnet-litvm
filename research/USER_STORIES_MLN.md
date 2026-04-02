# MLN user stories and CoinSwap interactions

**Status:** product / UX narrative — complements `[PRODUCT_SPEC.md](../PRODUCT_SPEC.md)` (architecture, economics, evidence) and `[NOSTR_MLN.md](NOSTR_MLN.md)` (wire format).

## Canonical roles and layer split


| Layer     | Role in CoinSwap                                                                                           |
| --------- | ---------------------------------------------------------------------------------------------------------- |
| **MWEB**  | Privacy engine: onion build/peel, kernels, per-hop fees in the MWEB fee budget (PRODUCT_SPEC section 5.2). |
| **LitVM** | Registry stake, grievance bonds, judicial lifecycle — not happy-path mix verification.                     |
| **Nostr** | Discovery and gossip (maker ads, optional grievance pointers) — not authoritative for stake.               |
| **Tor**   | Transport for mix API; hides client/node IP linkage.                                                       |


## Coordination model

**Intent:** CoinSwaps run on **discrete epochs** (example: **once per day at midnight** — an implementation choice; PRODUCT_SPEC appendix 14 notes “daily local midnight” as an example batch boundary for `coinswapd`-style batching, not a consensus rule). Between epochs, **takers** accumulate a **queue** (intent to join the next batch). **Makers** join and leave unpredictably; the stack must still allow **route construction** for the upcoming round.

### What manages “the list”


| Need                                    | Mechanism                                                          | Why                                                                                                                                       |
| --------------------------------------- | ------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| **Authoritative stake + identity**      | LitVM `MwixnetRegistry` (and RPC)                                  | Takers filter operators by deposit/bond; Nostr cannot be source of truth.                                                                 |
| **Current maker set + Tor + fee hints** | Nostr replaceable `mln_maker_ad` (kind 31250), optional heartbeats | High-churn “who is advertising” without spamming L2.                                                                                      |
| **Actual mix + queue consumption**      | MWEB + node protocol (`coinswapd`-style), Tor                      | Batching closes the round; the **queue** is a **client/node** concept for the next epoch (v1 does not require on-chain per-taker queues). |
| **Epoch identity for disputes**         | LitVM `epochId` in grievances + `evidenceHash`                     | Failed batches bind accusations to the agreed batch id (PRODUCT_SPEC appendix 13).                                                        |


**Flow:** Makers publish or update ads during the window; takers watch relays and verify stake on LitVM, building a **local** eligible set. Takers enqueue swap intents for **epoch E**. At the epoch boundary, nodes run the **batched** MWixnet round; anonymity set scales with participation in **E**. If the round fails, funds stay unspent; grievance logic references **epoch E** on LitVM.

**Open product choices:** Timezone for “midnight,” whether the queue is purely client-side vs partially visible on Nostr, and whether **size-based** cutover (“close when N outputs”) is combined with the clock.

## Maker exit (timelocked unstaking)

Makers cannot **instant-withdraw** registered stake: that would allow a “hit-and-run” (drop a payload in an epoch, exit, and evade slashing before grievances resolve). The registry uses **`requestWithdrawal` → cooldown → `withdrawStake`** (see PRODUCT_SPEC section 5.1.1). **`T_cooldown`** must exceed max epoch length plus **`T_challenge`**. Off-chain, the maker stops publishing **`mln_maker_ad`** when signaling exit. **`GrievanceCourt`** tracks **`openGrievanceCountAgainst`** so exit and final withdrawal are blocked while a case is open; stake **freeze** during an open grievance pauses withdrawal until resolution.

## Epoch semantics (schedule and `epochId`)

Implementations that use a **fixed schedule** (e.g. one batch per calendar day) must define the following so that **wallets**, **nodes**, and **grievance** evidence agree:

1. **Timezone anchor** — Choose one: **UTC midnight**, **local wall-clock midnight** per user (harder to align globally), or a **fixed offset** from UTC. Document the choice in release notes and wallet settings.
2. **Cutover instant** — The **epoch boundary** is the instant when the batch **closes** and mix processing for that round is defined to run (e.g. `00:00:00.000` on the chosen clock). Submissions intended for epoch **E** must be rejected or attributed to **E+1** after that instant per node policy.
3. **`epochId` mapping** — `epochId` (uint256 in contracts and appendix 13 preimage) must use a **single convention** across software, for example:
   - **Ordinal counter** — Deployed or agreed registry/scheduler increments `epochId` on-chain or via a well-known contract; **or**
   - **Time bucket** — `epochId = f(period_start_unix)` for a documented function `f` (e.g. floor division of Unix time by 86400 for daily epochs, with explicit timezone handling in the spec of “day”).
   Grievance **`evidenceHash`** and **`openGrievance(..., epochId, ...)`** must reference the **same** `epochId` the taker and makers used for that onion route. Until a convention is frozen, treat **testnet** and **production** mappings as **deployment-specific** and version the wallet wire format.
4. **LitVM optional role** — PRODUCT_SPEC section 7: LitVM *may* hold round parameters (epoch id, merkle root of commitments). If used, `epochId` on-chain must match off-chain batching.

## Taker goals (UX)

**Primary outcome:** Reduce **input/output linkability** with **as few steps and as little mental load** as possible, and **low total fees** (MWEB routing fee budget dominates in v1; LitVM is not the per-hop fee rail in the hybrid v1 model — PRODUCT_SPEC section 5.2).

**Automatic hop list:** The **wallet (or local daemon)** assembles the **ordered list of makers** — not manual hop-by-hop selection. Inputs: **Nostr** ads (reachability, Tor endpoints, fee hints) + **LitVM** RPC (stake thresholds, `nostrKeyHash` binding). The client applies a **route policy** (below) and builds the onion; the user typically confirms **amount** and a **preset** (“mix next epoch” / fee vs privacy tradeoff), not each hop.

**Tension — speed vs batch privacy:** Faster completion conflicts with **waiting for a large batched epoch**. Mitigations: smaller/faster batches, size-triggered cutover, or **presets** (“privacy max” vs “speed max”) without exposing hop lists in the default UI.

## Wallet PoC: automatic route policy

Normative **protocol** for hop order and cryptography remains the MWixnet / MWEB path (`coinswapd`-style). The **wallet policy** is client-side and should be **documented per implementation**. Recommended dimensions for a PoC:


| Dimension         | Purpose          | Example knobs                                                                                                   |
| ----------------- | ---------------- | --------------------------------------------------------------------------------------------------------------- |
| **Stake floor**   | Sybil resistance | Minimum registry stake per hop; minimum lock duration if exposed on-chain.                                      |
| **Fee objective** | Low fees         | Minimize estimated **total MWEB** routing fee subject to constraints; or cap fee rate per hop from Nostr hints. |
| **Hop count**     | Privacy vs cost  | Minimum and maximum hops in the path; randomness among eligible routes to reduce fingerprinting.                |
| **Diversity**     | Correlation      | Avoid reusing same maker set as last mix; optional geographic / ASN diversity if metadata exists (advanced).    |
| **Liveness**      | Reachability     | Prefer makers with recent replaceable ad or heartbeat; drop stale Tor endpoints.                                |


**Defaults:** Ship a **single conservative default** (e.g. prioritize **minimum total fee** subject to stake floor and min hops) and an **“advanced”** screen for power users (manual route override) if offered.

**Out of scope for v1 LitVM:** Per-hop fee escrow on L2; policy only affects **which makers** and **order** for MWEB-settled fees.

## Who the users are (summary)


| Persona     | Protocol name                               | Primary job                                                                                     |
| ----------- | ------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| End user    | **Taker**                                   | Confirms amount and policy; wallet auto-selects hop list, builds onion, pays MWEB routing fees. |
| Operator    | **Maker** (may include N1 and N2…Nn)        | LitVM registration; Nostr ads; `coinswapd`-style API over Tor; earns MWEB fee share on success. |
| Entry hop   | **Swap server (N1)**                        | Accepts swap API; validates inputs against UTXO set (MWEB rules).                               |
| Middle/last | **Mixers (N2…Nn)**                          | Transform/sort commitments; forward inner onion layers.                                         |
| Dispute     | **Accuser** (often taker or upstream maker) | `openGrievance` with `evidenceHash` + bond after a failed epoch.                                |
| Dispute     | **Accused maker**                           | `defendGrievance` within challenge window; stake may be frozen or slashed.                      |


Nostr relay operators and LitVM RPC providers are infrastructure; they are not CoinSwap protocol roles.

## Example user stories

1. **As a taker,** I want to **break the link between my inputs and outputs** with **one or two confirmations** (amount, optional speed/privacy preset), **low fees**, and **no manual hop picking** — the wallet should **put makers into the hop list automatically** using Nostr + LitVM rules I do not have to think about.
2. **As a taker,** I want to **join the queue** for the next **scheduled epoch**, discover **which makers are in** for that window via **Nostr ads** while **confirming stake on LitVM**, then run the **MWEB** batch when the epoch closes — without a single coordinator owning the list.
3. **As a taker,** I want to find makers with enough locked stake and compatible fees **without trusting a single coordinator**, so I combine **Nostr discovery** with **LitVM stake checks** and connect over **Tor** for the **MWEB CoinSwap** round.
4. **As a maker,** I want **stake and identity binding** on LitVM to be what wallets trust, and Nostr to only **mirror** reachability/fees, so I register on-chain and publish a **replaceable maker ad** — updating it when I **join, leave, or change** Tor/fees before the next epoch.
5. **As a taker (or upstream maker),** if a batch never completes and I have evidence of who failed, I want to **open a grievance** on LitVM with a pre-committed `**evidenceHash`**, lock a bond, and let the accused defend or face slash — without putting full mix transcripts on-chain in the happy path.
6. **As any participant,** I expect failed mixes to be **non-custodial** (UTXOs unspent), with **delay** as the main cost; **slashing/bounties** are the economic backstop, not the default fee rail (PRODUCT_SPEC sections 5.3 and 6).

## Related documents

- `[PRODUCT_SPEC.md](../PRODUCT_SPEC.md)` — Sections 4 (roles), 5–6 (LitVM, grievances), 7 (coordination), appendix 13 (`evidenceHash`), appendix 14 (MWEB batching notes).
- `[NOSTR_MLN.md](NOSTR_MLN.md)` — Event kinds 31250–31251, `nostrKeyHash`, maker ad schema.

