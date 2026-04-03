# MLN Wallet (Wails desktop)

Phase 11 taker GUI: same stack as `mln-cli` (scout → pathfind → forger) with settings persisted under the OS user config directory (`mln-wallet/settings.json`). The balance panel calls **`GET /v1/balance`** on your MLN sidecar (see [`PHASE_10_TAKER_CLI.md`](../../PHASE_10_TAKER_CLI.md)); implement it on your `coinswapd` fork.

## Prerequisites

- **Go 1.22+** with **CGO enabled** (Wails uses a native webview).
- **Node 18+** and npm (for the React/Vite frontend).
- OS dev libraries for Wails v2 (see [Wails installation](https://wails.io/docs/gettingstarted/installation)).

## Build (repo root)

```bash
make build-mln-wallet
```

Produces `bin/mln-wallet`. This runs `npm ci` + `npm run build` in `frontend/`, then `go build -tags=wails ./desktop/`.

## Develop

1. Build the SPA once (or after UI edits): `cd frontend && npm install && npm run build`
2. From `mln-cli`: `CGO_ENABLED=1 go run -tags=wails ./desktop/`

For Vite HMR, run `npm run dev` in `frontend/` in one terminal and use the Wails CLI from the same module if you install it (optional); the hand-maintained JS bridge is `frontend/src/wailsjs/go/main/App.js`.

## Implementation notes

- Desktop sources are behind the **`wails` build tag** so `go test ./...` in CI does not link the GUI (no GTK/WebKit on the default mln-cli job).
- Go bindings expect `window.go.main.App` (Wails v2).
