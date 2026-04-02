# LitVM Testnet Deployment Guide (Phase 2 -> Public)

This guide takes the local contracts (`MwixnetRegistry` + `GrievanceCourt`) live on LitVM testnet so grievances become publicly verifiable and Nostr pointers point to real on-chain state.

## Prerequisites

- Funded deployer wallet on LitVM testnet (get testnet LTC from faucet).
- `.env` with:

```env
LITVM_TESTNET_RPC=https://testnet.litvm.com/rpc
DEPLOYER_PRIVATE_KEY=0x...
```

- `forge` (already in Docker image)

## 1. Deploy

```bash
cd contracts
cp .env.example .env          # edit with real RPC + key
forge script script/Deploy.s.sol:Deploy \
  --rpc-url $LITVM_TESTNET_RPC \
  --broadcast \
  --verify
```

Broadcast JSON will be in `broadcast/Deploy.s.sol/<chainId>/run-latest.json`.

Contract addresses will be printed - record them:

- `MwixnetRegistry`: `0x...`
- `GrievanceCourt`: `0x...`

## 2. Update Nostr wiring

Edit `research/NOSTR_EVENTS.md` and the scripts:

- Add a top-level note with the real addresses.
- Update `scripts/publish_grievance.py` / `mln-nostr-demo.py` to include `["litvm-registry","0x..."]` tag on maker ads.

## 3. Verify on testnet

```bash
cast call <GrievanceCourt> "getGrievance(bytes32)" <grievanceId> --rpc-url $LITVM_TESTNET_RPC
```

## 4. Next (after deployment)

- Update `NOSTR_EVENTS.md` example tags with real contract addresses.
- Run `make test-full-stack-with-nostr` pointing at the real LitVM RPC.
- Announce the testnet on Nostr with the new maker-ad events.

See `scripts/deploy-local-anvil.sh` for the local pattern this mirrors.
