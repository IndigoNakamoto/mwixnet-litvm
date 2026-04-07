#!/usr/bin/env bash
# Optional E2E: after openGrievance (see scripts/test-grievance-local.sh), time-warp Anvil and
# permissionless resolveGrievance while phase is Open — verifies accuser-side timeout finality.
#
# Usage: ANVIL_RPC_URL=http://127.0.0.1:8545 ./scripts/grievance-e2e-anvil.sh
# Prerequisites: Anvil running, contracts deployed (scripts/deploy-local-anvil.sh), golden open from test-grievance-local.sh
#
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

readonly EXPECT_GRIEVANCE_ID="${EXPECT_GRIEVANCE_ID:-0x5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3}"
LATEST="$ROOT/contracts/broadcast/Deploy.s.sol/$CHAIN_ID/run-latest.json"
if [[ ! -f "$LATEST" ]]; then
  echo "Error: deploy first: $LATEST missing"
  exit 1
fi

COURT="$(python3 -c "
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
for t in d.get('transactions', []):
    if t.get('transactionType') == 'CREATE' and t.get('contractName') == 'GrievanceCourt':
        print(t['contractAddress'])
        sys.exit(0)
sys.exit(1)
" "$LATEST")"

CW_HEX="$(cast_run call --rpc-url "$RPC_CAST" "$COURT" "challengeWindow()(uint256)" 2>/dev/null | head -1)"
CW="${CW_HEX:-0x15180}"
# strip 0x and convert to decimal (bash arithmetic needs decimal)
CW_DEC=$((CW))

echo "GrievanceCourt=$COURT challengeWindow=$CW_DEC seconds"
echo "== increase time past deadline (Anvil) =="
cast_run rpc --rpc-url "$RPC_CAST" evm_increaseTime "$((CW_DEC + 2))" >/dev/null
cast_run rpc --rpc-url "$RPC_CAST" evm_mine >/dev/null || true

echo "== resolveGrievance(permissionless) =="
# Default Anvil account 0 pays gas (any account may call).
cast_run send --rpc-url "$RPC_CAST" "$COURT" \
  "resolveGrievance(bytes32)" "$EXPECT_GRIEVANCE_ID" \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

PHASE_HEX="$(cast_run call --rpc-url "$RPC_CAST" "$COURT" "grievances(bytes32)(address,address,uint256,bytes32,uint256,uint256,uint8,uint256)" "$EXPECT_GRIEVANCE_ID" 2>/dev/null | tail -1 || true)"
echo "Post-resolve grievances() tail (expect ResolvedSlash phase 3): $PHASE_HEX"
echo "OK — resolve path completed (run test-grievance-local.sh first if phase was not Open)."
