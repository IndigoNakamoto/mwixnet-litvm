# MLN Solidity (LitVM)

Contracts implementing a **maker registry** (`MwixnetRegistry`), **grievance / judicial skeleton** (`GrievanceCourt`), and **appendix 13 hash helpers** (`EvidenceLib`) for the MLN stack. See [`../PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) and [`../research/LITVM.md`](../research/LITVM.md).

## Quick start

With Foundry installed, from `contracts/`:

```bash
forge build
forge test -vv
```

From repo root using **Docker** (no local `forge`):

```bash
make contracts-test
```

## Phase 1 (local, no LitVM testnet RPC)

You can complete build, test, and **Anvil deploy** without public LitVM parameters:

1. Run tests: `make contracts-test` or Docker `forge test` (see [`../research/LITVM.md`](../research/LITVM.md)).
2. Start **Anvil** in a second terminal, e.g.  
   `docker run --rm -p 8545:8545 ghcr.io/foundry-rs/foundry:latest anvil --host 0.0.0.0`
3. Run [`../scripts/deploy-local-anvil.sh`](../scripts/deploy-local-anvil.sh) or `make deploy-local`. Uses the default first Anvil private key — **local only**.
4. Copy addresses from `broadcast/` into `deployments/anvil-local.json` if you want a stable record (see `deployments/anvil-local.example.json`; `anvil-local.json` is gitignored).

LitVM **testnet** broadcast remains blocked until [official RPC / chain ID](https://docs.litvm.com/get-started-on-testnet/add-to-wallet) are published.

## Layout

| Path | Description |
|------|-------------|
| `src/EvidenceLib.sol` | Pure `evidenceHash` (appendix 13.5) and `grievanceId` (matches `GrievanceCourt`) |
| `src/MwixnetRegistry.sol` | Stake (native), `registerMaker`, freeze/unfreeze for judicial contract |
| `src/GrievanceCourt.sol` | `openGrievance`, `defendGrievance`, `resolveGrievance` — not audited |
| `script/Deploy.s.sol` | Deploy registry → court → `setGrievanceCourt` |
| `test/` | Unit + fuzz tests |
| `deployments/anvil-local.example.json` | Example recorded addresses after local deploy |

Deploy script for **real LitVM**: copy `.env.example` to `.env`, set `PRIVATE_KEY` and `LITVM_RPC_URL`.

`lib/forge-std` is vendored for reproducible builds; you can replace with `forge install foundry-rs/forge-std` if you prefer.
