# Phase 8: LitVM testnet readiness + production packaging

## Context from the repo

- `mlnd/README.md` already has a LitVM testnet subsection with placeholders and links to `research/LITVM.md` and official docs. Phase 8 should add `mlnd/.env.example` (currently missing) and tighten root + mlnd docs rather than duplicating large env tables.
- `Makefile` has contracts + `test-operator-smoke` but no `build` / `docker-build` / `testnet-smoke`.
- CI for Go is minimal: `.github/workflows/mlnd.yml` (`go test` only on `mlnd/**` changes).
- **CGO:** `mlnd/internal/store/db.go` imports `github.com/mattn/go-sqlite3`, so `CGO_ENABLED=1` and a C toolchain are required for real binaries. Static cross-compiles from one runner are painful; prefer native builds per arch (see release workflow below).

## 1) Add PHASE_8_TESTNET_RELEASE.md

Commit the milestone doc at repo root (content as written, minus any YAML).

## 2) Dockerfile for mlnd (multi-stage, Go 1.22, minimal runtime)

- Placement: `mlnd/Dockerfile`
- Builder: `golang:1.22-bookworm`
- Build: `CGO_ENABLED=1 go build -o /out/mlnd ./cmd/mlnd`
- Runtime: `debian:bookworm-slim` with `ca-certificates`
- Docs: sample `docker run` in `mlnd/README.md`

## 3) Makefile targets

Add `build`, `docker-build`, `testnet-smoke` (plus `.PHONY`).

## 4) `make testnet-smoke`

New script `scripts/mlnd-testnet-smoke.sh` that does cheap `cast` checks against user-provided testnet env vars (placeholders only).

## 5) `mlnd/.env.example` + doc touch-ups

Add `.env.example` with commented placeholders. Update READMEs with testnet run + release process sections.

## 6) GitHub Actions release workflow

New `.github/workflows/mlnd-release.yml` triggered on `v*` tags, matrix of native amd64/arm64 builds, GitHub Release assets.

## 7) Mark Phase 8 complete

Update checklist + root roadmap when all deliverables merge.

## Suggested PR order

1. `PHASE_8_TESTNET_RELEASE.md` + `.gitignore` for `bin/` if needed.
2. `mlnd/Dockerfile` + Makefile build targets.
3. `.env.example` + README updates.
4. `testnet-smoke` script + make target.
5. Release workflow.

## Completion (shipped in repo)

- [x] Milestone doc (this file), `/bin/` gitignore
- [x] `mlnd/Dockerfile`, `make build`, `make docker-build`
- [x] `mlnd/.env.example`, README “run on testnet” + release notes
- [x] `scripts/mlnd-testnet-smoke.sh`, `make testnet-smoke`
- [x] `.github/workflows/mlnd-release.yml` (tag `v*`)
