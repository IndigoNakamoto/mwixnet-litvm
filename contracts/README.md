# MLN Solidity (LitVM)

Contracts implementing a **maker registry** (`MwixnetRegistry`) and **grievance / judicial skeleton** (`GrievanceCourt`) for the MLN stack. See [`../PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) and [`../research/LITVM.md`](../research/LITVM.md).

## Quick start

```bash
forge build
forge test -vv
```

Deploy script: `script/Deploy.s.sol`. Environment: copy `.env.example` to `.env`.

## Layout

| Path | Description |
|------|-------------|
| `src/MwixnetRegistry.sol` | Stake (native), `registerMaker`, freeze/unfreeze for judicial contract |
| `src/GrievanceCourt.sol` | `openGrievance`, `defendGrievance`, `resolveGrievance` — not audited |
| `script/Deploy.s.sol` | Deploy registry → court → `setGrievanceCourt` |
| `test/` | Anvil unit tests |

`lib/forge-std` is vendored for reproducible builds; you can replace with `forge install foundry-rs/forge-std` if you prefer.
