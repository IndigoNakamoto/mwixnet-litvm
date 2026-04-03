# coinswapd Evidence Integration (Phase 2)

This note explains how to apply and validate the external `coinswapd` evidence patch from this repo.

Canonical patch file:

- `research/coinswapd-evidence.patch`

Related references:

- `research/EVIDENCE_GENERATOR.md` (preimage and hash rules)
- `research/COINSWAPD_TEARDOWN.md` (entry points and code map)
- `research/NOSTR_MLN.md` (normative Nostr kinds **31250–31251** and `content` JSON; see also archived pointer in `research/NOSTR_EVENTS.md`)

## 1) Clone or update the local reference tree

From repo root:

```bash
git clone https://github.com/ltcmweb/coinswapd.git research/coinswapd
```

If you already have a local clone:

```bash
cd research/coinswapd
git fetch origin
git checkout main
git pull --ff-only
```

## 2) Apply the patch

From repo root:

```bash
cd research/coinswapd
git apply ../coinswapd-evidence.patch
```

If your fork has drifted, inspect and apply manually:

```bash
git apply --reject --whitespace=fix ../coinswapd-evidence.patch
```

## 3) Verify patch footprint

```bash
git status --short
git diff -- config/config.go evidence/evidence.go swap/swap.go
```

Expected files touched:

- `config/config.go` (feature toggles and evidence log options)
- `evidence/evidence.go` (canonical `evidenceHash` + `grievanceId` helper)
- `swap/swap.go` (failure-path evidence logging hook)

## 4) Build and sanity check

Run normal build/tests for your `coinswapd` fork:

```bash
go build ./...
go test ./...
```

Then exercise a failure-path scenario with flags enabled:

- `--litvm-enabled`
- `--grievance-court <0x...>` (optional for logging-only flow)
- `--evidence-log-dir <path>` (optional; defaults to `evidence-logs`)

Confirm logs contain:

- `grievanceId`
- `evidenceHash`
- epoch context for the failure

## 5) Bridge back to this repo workflow

1. Run `make test-grievance` (or `make test-full-stack`) in this repo to verify local LitVM grievance flow.
2. Use emitted on-chain `grievanceId` (and deployment addresses) with `scripts/publish_grievance.py` — **do not** put `evidenceHash` or preimages on Nostr; see `research/NOSTR_MLN.md`.
3. Publish **`kind=31251`** `mln_grievance_pointer` events per `research/NOSTR_MLN.md` (tags + `content` schema match [`nostr/fixtures/valid/grievance_pointer.json`](../nostr/fixtures/valid/grievance_pointer.json)).

This keeps the happy-path MWEB flow unchanged while enabling deterministic failure evidence for LitVM grievance handling.

## 6) Fork + validate a live failure flow

Use this when moving from local vectors to a real patched `coinswapd` fork.

### Fork setup

```bash
# In your fork on GitHub first, then wire remotes locally.
cd research/coinswapd
git remote rename origin upstream
git remote add origin git@github.com:<you>/coinswapd.git
git fetch upstream
git checkout -b mln/evidence-hooks upstream/main
```

### Re-apply patch on the fork branch

```bash
git apply ../coinswapd-evidence.patch || git apply --reject --whitespace=fix ../coinswapd-evidence.patch
git status --short
git diff -- config/config.go evidence/evidence.go swap/swap.go
```

### Build/test and run failure simulation

```bash
go build ./...
go test ./...
```

Then run your local failure scenario with evidence logging enabled (for example, misconfigured/withheld broadcast on one maker hop) and capture generated evidence JSON.

### Minimum capture checklist (grievance-ready)

For at least one failed swap, confirm your logs include:

- `grievanceId` (32-byte hex, `0x` prefixed)
- `evidenceHash` (32-byte hex, `0x` prefixed)
- epoch context used to derive `grievanceId` / preimage payload
- accused maker identifier used in your local runbook

Finally, bridge the result back into this repo flow:

1. Open a local grievance via `make test-grievance` or `make test-full-stack`.
2. Publish pointer data with `scripts/publish_grievance.py`.
3. Verify relay visibility with `scripts/listen_grievances.py` and/or `scripts/listen_makers.py`.

## 7) Happy-path hop receipts for `mlnd` (NDJSON)

Operator checklist and smoke target: [`PHASE_7_END_TO_END.md`](../PHASE_7_END_TO_END.md) (`make test-operator-smoke`).

The Phase 2 patch above targets **failure-path** logging. For **automatic SQLite receipts** when a mix completes successfully, the fork (or a sidecar) must emit **LitVM-correlated** hop rows: registry **`epochId`**, **`accuser`** (grievance opener identity), **`accusedMaker`**, peel correlators, and defense strings. Stock `swap_*` RPCs do not expose that bundle alone; see [`PHASE_6_BRIDGE_INTEGRATION.md`](../PHASE_6_BRIDGE_INTEGRATION.md) for the **NDJSON line schema** consumed by `mlnd` (`MLND_BRIDGE_RECEIPTS_DIR`).

### Committable patch (ltcmweb/coinswapd)

Apply from a [`coinswapd`](https://github.com/ltcmweb/coinswapd) clone root (same idea as §2):

```bash
cd research/coinswapd   # your local clone; gitignored in mwixnet-litvm
git apply ../coinswapd-receipt-ndjson.patch
# or: patch -p1 < ../coinswapd-receipt-ndjson.patch
```

Canonical file in this repo: [`coinswapd-receipt-ndjson.patch`](coinswapd-receipt-ndjson.patch). Base revision: **`ltcmweb/coinswapd` `master`** as of the patch author date; if hunks fail after upstream drift, re-apply manually using the hook description below.

### Hook site

- **File:** `swap.go`
- **Function:** `(*swapService) forward`
- **Placement:** Immediately after `cipher.XORKeyStream(data.Bytes(), data.Bytes())` on the forward blob to the next hop (bytes are **post-XOR** `P` for `forwardCiphertextHash` per [`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) appendix 13 / [`EVIDENCE_GENERATOR.md`](EVIDENCE_GENERATOR.md)).
- **Package:** new `mlnreceipt/writer.go` — `peeledCommitment` = `sha256(33-byte mw.Commitment)` (appendix 13.3); `forwardCiphertextHash` = `keccak256(postXOR)`.

### New flags (coinswapd)

| Flag | Role |
|------|------|
| `-mln-receipt-dir` | Directory for append-only `mlnd-receipts.ndjson` (set to the **same path** as mlnd’s `MLND_BRIDGE_RECEIPTS_DIR`). |
| `-mln-receipt-epoch-id` | Decimal epoch string. |
| `-mln-receipt-accuser` | `0x` accuser (taker) address. |
| `-mln-receipt-accused` | `0x` accused maker address. |
| `-mln-receipt-next-hop-pubkey` | Defense v1 UTF-8 string. |
| `-mln-receipt-signature` | Defense v1 UTF-8 string. |

**v1 scope:** a line is written only when `len(commits)==1` after `peelOnions()` (single-commit forward batch). Larger batches are skipped without a line.

### Example line (shape only)

```json
{"epochId":"42","accuser":"0x…","accusedMaker":"0x…","hopIndex":0,"peeledCommitment":"0x…64 hex…","forwardCiphertextHash":"0x…64 hex…","nextHopPubkey":"…","signature":"…"}
```

Golden-vector row matching local LitVM smoke tests: [`scripts/mlnd-bridge-litvm-smoke.sh`](../scripts/mlnd-bridge-litvm-smoke.sh) (`make test-operator-smoke`).

### Toolchain

Upstream `coinswapd` **go.mod** may require **Go 1.23+**; build the fork with a matching toolchain after applying the patch.
