# coinswapd Evidence Integration (Phase 2)

This note explains how to apply and validate the external `coinswapd` evidence patch from this repo.

Canonical patch file:

- `research/coinswapd-evidence.patch`

Related references:

- `research/EVIDENCE_GENERATOR.md` (preimage and hash rules)
- `research/COINSWAPD_TEARDOWN.md` (entry points and code map)
- `research/NOSTR_EVENTS.md` (grievance pointer event profile)

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
2. Use emitted/derived grievance values with `scripts/publish_grievance.py`.
3. Publish `kind=31001` pointers per `research/NOSTR_EVENTS.md`.

This keeps the happy-path MWEB flow unchanged while enabling deterministic failure evidence for LitVM grievance handling.
