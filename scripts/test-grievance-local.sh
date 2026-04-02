#!/usr/bin/env bash
# Smoke test: deploy to local Anvil, open a grievance using golden appendix-13 vectors
# (see research/EVIDENCE_GENERATOR.md, contracts/test/EvidenceGoldenVectors.t.sol).
#
# Prerequisites: Anvil on ANVIL_RPC_URL (default http://127.0.0.1:8545), Docker for deploy + optional cast.
# Start Anvil (must bind 0.0.0.0 or host port publish will not work):
#   docker run --rm -p 8545:8545 --entrypoint anvil ghcr.io/foundry-rs/foundry:latest --host 0.0.0.0

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${FOUNDRY_IMAGE:-ghcr.io/foundry-rs/foundry:latest}"
RPC="${ANVIL_RPC_URL:-http://127.0.0.1:8545}"
RPC_DOCKER="$RPC"
case "$RPC" in
  http://127.0.0.1:*)
    RPC_DOCKER="${RPC/http:\/\/127.0.0.1/http://host.docker.internal}"
    ;;
  http://localhost:*)
    RPC_DOCKER="${RPC/http:\/\/localhost/http://host.docker.internal}"
    ;;
esac
CHAIN_ID="${ANVIL_CHAIN_ID:-31337}"

# Prefer host `cast` (sees localhost); otherwise run cast inside Docker (needs host.docker.internal).
if command -v cast >/dev/null 2>&1; then
  cast_run() { cast "$@"; }
  RPC_CAST="$RPC"
else
  cast_run() {
    docker run --rm --add-host=host.docker.internal:host-gateway \
      -v "$ROOT/contracts:/work" -w /work --entrypoint cast "$IMAGE" "$@"
  }
  RPC_CAST="$RPC_DOCKER"
fi

# Golden vector inputs (must match EvidenceGoldenVectors.t.sol)
readonly ACCUSER="0x000000000000000000000000000000000000bEef"
readonly ACCUSED="0x000000000000000000000000000000000000CAfE"
readonly EPOCH_ID=42
readonly EVIDENCE_HASH="0x2d4d7ae96f39e2d5037f21782bc831874261ffe22743f74bbf865a39ec4df112"
readonly EXPECT_GRIEVANCE_ID="0x5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3"

if ! cast_run block-number --rpc-url "$RPC_CAST" &>/dev/null; then
  echo "Error: nothing listening at $RPC"
  echo "Start Anvil, e.g.: docker run --rm -p 8545:8545 --entrypoint anvil $IMAGE --host 0.0.0.0"
  exit 1
fi

echo "== Deploy (see scripts/deploy-local-anvil.sh) =="
"$ROOT/scripts/deploy-local-anvil.sh"

LATEST="$ROOT/contracts/broadcast/Deploy.s.sol/$CHAIN_ID/run-latest.json"
if [[ ! -f "$LATEST" ]]; then
  echo "Error: expected broadcast artifact at $LATEST (chain id $CHAIN_ID)."
  echo "If your Anvil uses another chain id, set ANVIL_CHAIN_ID."
  exit 1
fi

PAIR="$(python3 -c "
import json, sys
path = sys.argv[1]
with open(path) as f:
    d = json.load(f)
reg, court = None, None
for t in d.get('transactions', []):
    if t.get('transactionType') != 'CREATE':
        continue
    cn = t.get('contractName')
    if cn == 'MwixnetRegistry':
        reg = t['contractAddress']
    elif cn == 'GrievanceCourt':
        court = t['contractAddress']
if not reg or not court:
    print('missing CREATE addresses in broadcast', file=sys.stderr)
    sys.exit(1)
print(reg)
print(court)
" "$LATEST")"
REGISTRY=$(printf '%s\n' "$PAIR" | sed -n '1p')
COURT=$(printf '%s\n' "$PAIR" | sed -n '2p')

echo "MwixnetRegistry: $REGISTRY"
echo "GrievanceCourt: $COURT"

echo "== Fund + impersonate golden accuser (Anvil only) =="
cast_run rpc --rpc-url "$RPC_CAST" anvil_setBalance "$ACCUSER" 0x21e19e0c9bab2400000 &>/dev/null
cast_run rpc --rpc-url "$RPC_CAST" anvil_impersonateAccount "$ACCUSER" &>/dev/null

echo "== openGrievance (bond 0.1 ETH, min default 0.01) =="
cast_run send --rpc-url "$RPC_CAST" "$COURT" \
  "openGrievance(address,uint256,bytes32)" "$ACCUSED" "$EPOCH_ID" "$EVIDENCE_HASH" \
  --value 0.1ether --from "$ACCUSER" --unlocked

echo "== Verify grievance + frozen stake =="
CAL=$(cast_run calldata "grievances(bytes32)" "$EXPECT_GRIEVANCE_ID" 2>/dev/null | tail -n 1)
RAW_RET=$(cast_run rpc eth_call "{\"to\":\"$COURT\",\"data\":\"$CAL\"}" latest --rpc-url "$RPC_CAST" 2>/dev/null | tr -d '\n"')

python3 -c "
import sys
accuser, accused, epoch, ev, raw_ret = sys.argv[1:6]
h = raw_ret.strip().removeprefix('0x')
if len(h) != 8 * 64:
    print('unexpected eth_call return length', len(h), file=sys.stderr)
    sys.exit(1)
words = [h[i : i + 64] for i in range(0, len(h), 64)]
got_accuser = '0x' + words[0][-40:]
got_accused = '0x' + words[1][-40:]
got_epoch = int(words[2], 16)
got_ev = '0x' + words[3]
phase = int(words[6], 16)
if got_accuser.lower() != accuser.lower():
    print('accuser mismatch', got_accuser, accuser, file=sys.stderr)
    sys.exit(1)
if got_accused.lower() != accused.lower():
    print('accused mismatch', got_accused, accused, file=sys.stderr)
    sys.exit(1)
if got_epoch != int(epoch):
    print('epoch mismatch', got_epoch, epoch, file=sys.stderr)
    sys.exit(1)
if got_ev.lower() != ev.lower():
    print('evidenceHash mismatch', got_ev, ev, file=sys.stderr)
    sys.exit(1)
if phase != 1:
    print('expected phase Open (1), got', phase, file=sys.stderr)
    sys.exit(1)
" "$ACCUSER" "$ACCUSED" "$EPOCH_ID" "$EVIDENCE_HASH" "$RAW_RET"

frozen=$(cast_run call --rpc-url "$RPC_CAST" "$REGISTRY" "stakeFrozen(address)(bool)" "$ACCUSED" 2>/dev/null)
if [[ "$(echo "$frozen" | tr '[:upper:]' '[:lower:]')" != *"true"* ]]; then
  echo "Error: expected stakeFrozen($ACCUSED) == true, got: $frozen"
  exit 1
fi

echo "stakeFrozen(accused) = true"
echo "grievance phase = 1 (Open)"
echo "OK — golden grievanceId $EXPECT_GRIEVANCE_ID matches on-chain record."
