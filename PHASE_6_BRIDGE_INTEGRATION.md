# Phase 6: Real coinswapd receipt bridge + testnet readiness (mlnd)

name: mlnd Phase 6 – Full bridge + testnet
overview: Replace the no-op bridge stub with real receipt ingestion from coinswapd (JSON-RPC `swap_*` or structured log tail per research/COINSWAPD_TEARDOWN.md). Wire it so completed mixes automatically land in SQLite. Add testnet RPC defaults and a simple end-to-end smoke test. This closes the operator loop end-to-end.

todos:
  - id: bridge-real
    content: Implement Coinswapd bridge (JSON-RPC client or log parser) → store.SaveReceipt. Use optional go.mod replace to research/coinswapd only under a build tag if needed.
    status: pending
  - id: testnet-defaults
    content: Add sensible defaults for LitVM testnet RPC + court address in README and Load*FromEnv helpers.
    status: pending
  - id: e2e-smoke
    content: Extend internal/flow/ with a smoke test that simulates a completed mix → receipt → defend dry-run + Nostr ad.
    status: pending
  - id: docs-update
    content: Update mlnd/README.md with full bridge env vars and "how to run with real coinswapd" section.
    status: pending
isProject: false
