# NIP-42 AUTH relay hardening

## Goal

Wire NIP-42 AUTH into the mlnd broadcaster and mln-cli Scout so maker ads (kind 31250) can be published to — and read from — relays that require authentication, closing the cleartext-onion DoS vector at the relay level without new cryptographic ad formats.

## In scope

- Broadcaster AUTH after relay connect (`MLND_NOSTR_AUTH` env var).
- Scout AUTH on subscribe (`MLN_NOSTR_AUTH_NSEC` env var).
- Dashboard self-check AUTH passthrough.
- E2E relay config (`deploy/nostr-rs-relay.toml`) with commented AUTH example.
- Documentation: mlnd README, `research/NOSTR_MLN.md` relay policy, threat model row, operator checklist.
- Unit tests for broadcaster AUTH path.

## Out of scope

- v2 sealed `reachability` (stays draft; future hardening layer).
- `pathfind` changes (still requires non-empty `tor` from verified makers).
- New Nostr key management beyond existing `MLND_NOSTR_NSEC` pattern.
- LitVM contracts or MWEB paths.

## Primary files and canonical docs

| Area | Files |
|------|-------|
| Broadcaster | `mlnd/internal/nostr/broadcaster.go`, `mlnd/internal/nostr/broadcaster_test.go` |
| Scout | `mln-cli/internal/scout/scout.go`, `mln-cli/cmd/mln-cli/main.go` |
| Dashboard | `mlnd/internal/dashboard/nostrself.go`, `mlnd/internal/dashboard/status.go`, `mlnd/cmd/mlnd/main.go` |
| Relay config | `deploy/nostr-rs-relay.toml`, `deploy/docker-compose.e2e.yml` |
| Wire spec | `research/NOSTR_MLN.md` (relay policy section) |
| Threat model | `research/THREAT_MODEL_MLN.md` (cleartext DoS row) |
| Operator docs | `mlnd/README.md`, `deploy/.env.testnet.example`, `research/PHASE_3_OPERATOR_CHECKLIST.md` |
| E2E docs | `PHASE_12_E2E_CRUCIBLE.md` (AUTH testing section) |

Canonical: `AGENTS.md`, `PRODUCT_SPEC.md`, `research/NOSTR_MLN.md`.

## Execution results

### Code changes

- **`mlnd` broadcaster**: Added `AuthEnabled bool` to `BroadcasterConfig`; derived `pubHex` from `secHex` in `NewBroadcaster` via `gnostr.GetPublicKey`; `ensureRelay()` calls `r.Auth(ctx, signFunc)` after `Connect` when enabled, failing fast (backoff) on rejection; `AuthKeys()` accessor for dashboard; `LoadBroadcasterFromEnv` reads `MLND_NOSTR_AUTH` via `strconv.ParseBool`.
- **`mln-cli` Scout**: Added `AuthNsec string` to `scout.Config`; `parseAuthKey` helper (nsec1 + raw hex); `SimplePool` created with `gnostr.WithAuthHandler` when key is set; wired through `runScout` and `loadScoutConfig` (also covers pathfind/route build).
- **Dashboard**: `AuthSigner()` helper in `nostrself.go`; `FetchLatestMakerAdForDTag` accepts variadic `gnostr.PoolOption`; `StatusDeps` carries `NostrAuthSecHex`/`NostrAuthPubHex` populated from `bc.AuthKeys()`.
- **Relay config**: New `deploy/nostr-rs-relay.toml` with commented `[authorization]` section; docker-compose gets commented volume mount.

### Documentation

- `mlnd/README.md`: `MLND_NOSTR_AUTH` env var documented in optional broadcaster table.
- `research/NOSTR_MLN.md`: New "Relay policy (normative recommendations)" section — recommends 1-3 trusted NIP-42 relays with pubkey allowlist; documents what AUTH does and does not do.
- `research/THREAT_MODEL_MLN.md`: Cleartext DoS row updated — controls now include NIP-42 AUTH; residual downgraded to Medium-High with AUTH.
- `research/PHASE_3_OPERATOR_CHECKLIST.md`: Checklist item for `MLND_NOSTR_AUTH`.
- `deploy/.env.testnet.example`: `MLND_NOSTR_AUTH=true` (commented).
- `PHASE_12_E2E_CRUCIBLE.md`: Manual AUTH E2E testing steps.
- `mln-cli` usage text: `MLN_NOSTR_AUTH_NSEC` documented.

### Tests

- 6 new unit tests in `mlnd/internal/nostr/broadcaster_test.go`: pubkey derivation, `AuthKeys` enabled/disabled/nil, env loading with auth=true, env loading with invalid auth.
- All existing tests pass: `go test ./...` green for both `mlnd` and `mln-cli`; `python3 nostr/validate_fixtures.py` + `check_wire_helpers.py` green.

## Verification

```bash
cd mlnd && go build ./... && go test ./...
cd mln-cli && go build ./... && go test ./...
python3 nostr/validate_fixtures.py
python3 nostr/check_wire_helpers.py
```

Manual AUTH E2E: see `PHASE_12_E2E_CRUCIBLE.md` "Optional: NIP-42 AUTH relay testing" section.

## Layer-boundary check

Touches **Nostr** (relay auth, wire spec, Scout, broadcaster) only. LitVM, MWEB, and Tor layers are unchanged. Boundaries respected per `AGENTS.md`.
