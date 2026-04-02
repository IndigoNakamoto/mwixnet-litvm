# LitVM — toolchain and testnet notes

This project uses **[Foundry](https://book.getfoundry.sh/)** for Solidity under [`contracts/`](../contracts/). Product intent and judicial design are in [`PRODUCT_SPEC.md`](../PRODUCT_SPEC.md) (sections 5–6, appendix 13).

## Official docs

- [LitVM documentation](https://docs.litvm.com/)
- [Deploy on testnet (Foundry)](https://docs.litvm.com/deploy-on-testnet/foundry)
- [Add LitVM to your wallet / testnet params](https://docs.litvm.com/get-started-on-testnet/add-to-wallet)

**RPC URL, chain ID, and block explorer** for LitVM testnet are published by the LitVM team (historically marked TBA until launch). **Do not hardcode** guessed values: read current values from official docs or community channels (e.g. [Telegram](https://t.me/litecoinvm)) before deploying.

Native gas token on LitVM is **`zkLTC`** per docs. The contracts in this repo use **native `msg.value`** for stake, bonds, and withdrawals—adjust if you later use an ERC-20 stake token.

## Phase 1 without testnet (local)

Public **RPC / chain ID / faucet** may still be [TBA](https://docs.litvm.com/get-started-on-testnet/add-to-wallet). Until then:

- **CI / Docker:** `make contracts-test` or `make contracts-build` from the repo root ([`Makefile`](../Makefile)).
- **Anvil deploy:** start Anvil, then run [`scripts/deploy-local-anvil.sh`](../scripts/deploy-local-anvil.sh) (see [`contracts/README.md`](../contracts/README.md)). Optional env: `COOLDOWN_PERIOD` (seconds) for registry maker exit cooldown; default in `Deploy.s.sol` is 48 hours.
- **Hash alignment:** [`contracts/src/EvidenceLib.sol`](../contracts/src/EvidenceLib.sol) matches `PRODUCT_SPEC.md` appendix 13.5; tests in `contracts/test/EvidenceHash.t.sol`.

## Foundry via Docker (no host install)

Pull the official image and verify `forge` (one line):

```bash
docker pull ghcr.io/foundry-rs/foundry:latest && docker run --rm --entrypoint forge ghcr.io/foundry-rs/foundry:latest --version
```

From **`contracts/`**, use a session alias so `forge` matches a normal install:

```bash
cd contracts
alias forge='docker run --rm -v "$PWD:/work" -w /work --entrypoint forge ghcr.io/foundry-rs/foundry:latest'
forge build && forge test -vv
```

## Local commands

From `contracts/` (with [Foundry](https://book.getfoundry.sh/getting-started/installation) installed):

```bash
forge build
forge test -vv
```

Using Docker (no local Foundry install):

```bash
docker run --rm --entrypoint forge -v "$(pwd)/contracts:/work" -w /work ghcr.io/foundry-rs/foundry:latest build
docker run --rm --entrypoint forge -v "$(pwd)/contracts:/work" -w /work ghcr.io/foundry-rs/foundry:latest test -vv
```

## Environment

Copy [`contracts/.env.example`](../contracts/.env.example) to `contracts/.env` (gitignored). You need at least:

- `PRIVATE_KEY` — deployer (testnet only; never reuse mainnet keys)
- `LITVM_RPC_URL` — from LitVM docs when available

Deploy (after `source .env` or export vars):

```bash
cd contracts
forge script script/Deploy.s.sol:Deploy --rpc-url "$LITVM_RPC_URL" --broadcast
```

Optional env overrides: `MIN_STAKE`, `CHALLENGE_WINDOW`, `GRIEVANCE_BOND_MIN` (see `Deploy.s.sol`).

## Spec alignment

- **`evidenceHash`:** Computed **off-chain** per appendix 13 in `PRODUCT_SPEC.md`; contracts only store `bytes32`.
- **v1:** No LitVM fee escrow for per-hop routing (section 5.2); judicial bonds and stake only.
