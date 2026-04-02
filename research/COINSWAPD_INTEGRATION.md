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
