#!/usr/bin/env bash
# Phase 12 — deploy MwixnetRegistry + GrievanceCourt to local Anvil, fund three makers,
# deposit stake, registerMaker(bytes32 nostrKeyHash) for each, emit deploy/* env + wallet JSON.
#
# Prereq: Anvil reachable at ANVIL_RPC_URL (default http://127.0.0.1:8545), e.g.
#   docker compose -f deploy/docker-compose.e2e.yml up -d
#
# Then start makers (after this script):
#   docker compose -f deploy/docker-compose.e2e.yml --profile makers up -d
#
# Nostr binding: keccak256(x-only pubkey) per research/NOSTR_MLN.md (precomputed for fixed test nsecs).

set -euo pipefail

export FOUNDRY_DISABLE_NIGHTLY_WARNING="${FOUNDRY_DISABLE_NIGHTLY_WARNING:-1}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="$ROOT/deploy"
CONTRACTS="$ROOT/contracts"
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

export PRIVATE_KEY="${ANVIL_PRIVATE_KEY:-0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80}"

# Anvil default accounts 1–3 (EVM operators); pre-funded on a fresh Anvil chain.
MAKER1_PK="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
MAKER2_PK="0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
MAKER3_PK="0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"

# Nostr signing secrets (64 hex, no 0x) — paired with keccak256(x-only pubkey) below.
MAKER1_NSEC_HEX="1111111111111111111111111111111111111111111111111111111111111111"
MAKER2_NSEC_HEX="2222222222222222222222222222222222222222222222222222222222222222"
MAKER3_NSEC_HEX="3333333333333333333333333333333333333333333333333333333333333333"

# cast keccak 0x<xonly-pubkey-hex>
MAKER1_NOSTR_KEY_HASH="0xc3e938ad5923a01650d87289b7aa7613f6aadde783d3396eb396add0a33103cc"
MAKER2_NOSTR_KEY_HASH="0x027d946c45b7d0602884a19faf2f49b69d70c345161f6ca3fda700c1c822264b"
MAKER3_NOSTR_KEY_HASH="0x63f8dfa958ca88a88c62aff94fc63b82773a78539c0f19f5d2caab8a29bc03ee"

DEPOSIT_VALUE="${E2E_STAKE_DEPOSIT:-0.11ether}"

forge_docker() {
  docker run --rm --add-host=host.docker.internal:host-gateway \
    -e FOUNDRY_DISABLE_NIGHTLY_WARNING \
    -e PRIVATE_KEY -e MIN_STAKE -e COOLDOWN_PERIOD -e CHALLENGE_WINDOW -e GRIEVANCE_BOND_MIN \
    -v "$CONTRACTS:/work" -w /work --entrypoint forge "$IMAGE" "$@"
}

cast_docker() {
  docker run --rm --add-host=host.docker.internal:host-gateway \
    -e FOUNDRY_DISABLE_NIGHTLY_WARNING \
    -v "$CONTRACTS:/work" -w /work --entrypoint cast "$IMAGE" "$@"
}

if ! command -v jq &>/dev/null; then
  echo "Error: jq is required (brew install jq / apt install jq)."
  exit 1
fi

echo "Waiting for Anvil at $RPC ..."
for _ in $(seq 1 60); do
  if cast_docker block-number --rpc-url "$RPC_DOCKER" &>/dev/null; then
    break
  fi
  sleep 1
done
if ! cast_docker block-number --rpc-url "$RPC_DOCKER" &>/dev/null; then
  echo "Error: nothing listening at $RPC (from Foundry container: $RPC_DOCKER)"
  echo "Start Anvil, e.g.: docker compose -f deploy/docker-compose.e2e.yml up -d"
  exit 1
fi

CHAIN_ID_DEC="$(cast_docker chain-id --rpc-url "$RPC_DOCKER" | tr -d '[:space:]')"
if [[ -z "$CHAIN_ID_DEC" || ! "$CHAIN_ID_DEC" =~ ^[0-9]+$ ]]; then
  echo "Error: could not read chain id from $RPC"
  exit 1
fi

echo "Deploying contracts (chain id $CHAIN_ID_DEC) ..."
cd "$CONTRACTS"
forge_docker script script/Deploy.s.sol:Deploy --rpc-url "$RPC_DOCKER" --broadcast -vvv

RUN_JSON="$CONTRACTS/broadcast/Deploy.s.sol/$CHAIN_ID_DEC/run-latest.json"
if [[ ! -f "$RUN_JSON" ]]; then
  echo "Error: missing broadcast artifact $RUN_JSON"
  exit 1
fi

REGISTRY="$(jq -r '.transactions[] | select(.transactionType == "CREATE" and .contractName == "MwixnetRegistry") | .contractAddress' "$RUN_JSON" | head -1)"
COURT="$(jq -r '.transactions[] | select(.transactionType == "CREATE" and .contractName == "GrievanceCourt") | .contractAddress' "$RUN_JSON" | head -1)"

if [[ -z "$REGISTRY" || "$REGISTRY" == "null" || -z "$COURT" || "$COURT" == "null" ]]; then
  echo "Error: could not parse MwixnetRegistry / GrievanceCourt from $RUN_JSON"
  exit 1
fi

REGISTRY_LC="$(echo "$REGISTRY" | tr '[:upper:]' '[:lower:]')"
COURT_LC="$(echo "$COURT" | tr '[:upper:]' '[:lower:]')"

maker_addr() {
  cast_docker wallet address --private-key "$1"
}

ADDR1="$(maker_addr "$MAKER1_PK")"
ADDR2="$(maker_addr "$MAKER2_PK")"
ADDR3="$(maker_addr "$MAKER3_PK")"
ADDR1_LC="$(echo "$ADDR1" | tr '[:upper:]' '[:lower:]')"
ADDR2_LC="$(echo "$ADDR2" | tr '[:upper:]' '[:lower:]')"
ADDR3_LC="$(echo "$ADDR3" | tr '[:upper:]' '[:lower:]')"

register_maker() {
  local pk="$1"
  local hash="$2"
  local name="$3"
  echo "  [$name] deposit + registerMaker ..."
  cast_docker send --rpc-url "$RPC_DOCKER" --private-key "$pk" "$REGISTRY_LC" "deposit()" --value "$DEPOSIT_VALUE"
  cast_docker send --rpc-url "$RPC_DOCKER" --private-key "$pk" "$REGISTRY_LC" "registerMaker(bytes32)" "$hash"
}

echo "Registering three makers (stake deposit + registerMaker) ..."
register_maker "$MAKER1_PK" "$MAKER1_NOSTR_KEY_HASH" "maker1"
register_maker "$MAKER2_PK" "$MAKER2_NOSTR_KEY_HASH" "maker2"
register_maker "$MAKER3_PK" "$MAKER3_NOSTR_KEY_HASH" "maker3"

mkdir -p "$DEPLOY_DIR"

GEN_ENV="$DEPLOY_DIR/e2e.generated.env"
cat >"$GEN_ENV" <<EOF
# Generated by scripts/e2e-bootstrap.sh — do not commit (see deploy/.gitignore).
E2E_MWIXNET_REGISTRY=$REGISTRY_LC
E2E_GRIEVANCE_COURT=$COURT_LC
E2E_CHAIN_ID=$CHAIN_ID_DEC
E2E_ANVIL_HTTP=$RPC
E2E_ANVIL_WS=ws://127.0.0.1:8545
E2E_NOSTR_RELAY_WS=ws://127.0.0.1:7080/
EOF

write_maker_env() {
  local path="$1"
  local pk="$2"
  local addr_lc="$3"
  local nsec_hex="$4"
  local tor_url="$5"
  local swap_x25519_pub_hex="$6"

  cat >"$path" <<EOF
MLND_WS_URL=ws://anvil:8545
MLND_COURT_ADDR=$COURT_LC
MLND_OPERATOR_ADDR=$addr_lc
MLND_DB_PATH=/data/mlnd.db
MLND_LITVM_CHAIN_ID=$CHAIN_ID_DEC
MLND_REGISTRY_ADDR=$REGISTRY_LC
MLND_NOSTR_RELAYS=ws://nostr:8080/
MLND_NOSTR_INTERVAL=30s
MLND_NOSTR_NSEC=$nsec_hex
MLND_OPERATOR_PRIVATE_KEY=$pk
MLND_TOR_ONION=$tor_url
MLND_FEE_MIN_SAT=1
MLND_FEE_MAX_SAT=10
MLND_SWAP_X25519_PUB_HEX=$swap_x25519_pub_hex
EOF
}

# Fixed X25519 pub hex values for Phase 3a funded-route smoke (Nostr ads → pathfind → mweb_submitRoute).
# Generated once via X25519 keygen; safe to commit — they are public keys only.
E2E_SWAP_PUB_MAKER1=72a140d084c2a4f86a8a842bd3944810720812f65bff1ddf9ec1396900f71d09
E2E_SWAP_PUB_MAKER2=a8c28e54ec5602e805411b18426767731fd35ec43cbe48eea5e4f177e8777511
E2E_SWAP_PUB_MAKER3=78ea23ce22bc0d099422a5136db73f51c466cb267009f14efe1aa855e25ba73a

write_maker_env "$DEPLOY_DIR/e2e.maker1.env" "$MAKER1_PK" "$ADDR1_LC" "$MAKER1_NSEC_HEX" "http://127.0.0.1:8081" "$E2E_SWAP_PUB_MAKER1"
write_maker_env "$DEPLOY_DIR/e2e.maker2.env" "$MAKER2_PK" "$ADDR2_LC" "$MAKER2_NSEC_HEX" "http://127.0.0.1:8082" "$E2E_SWAP_PUB_MAKER2"
write_maker_env "$DEPLOY_DIR/e2e.maker3.env" "$MAKER3_PK" "$ADDR3_LC" "$MAKER3_NSEC_HEX" "http://127.0.0.1:8083" "$E2E_SWAP_PUB_MAKER3"

WALLET_JSON="$DEPLOY_DIR/e2e.wallet-settings.generated.json"
RELAY_WS="${E2E_NOSTR_HOST_WS:-ws://127.0.0.1:7080/}"
jq -n \
  --arg relay "$RELAY_WS" \
  --arg chain "$CHAIN_ID_DEC" \
  --arg http "$RPC" \
  --arg reg "$REGISTRY_LC" \
  --arg court "$COURT_LC" \
  '{
    nostrRelays: [$relay],
    litvmChainId: $chain,
    litvmHttpUrl: $http,
    registryAddr: $reg,
    grievanceCourtAddr: $court,
    scoutTimeout: "30s",
    defaultSidecarUrl: "http://127.0.0.1:8080/v1/swap",
    forgerHttpTimeout: "10s"
  }' >"$WALLET_JSON"

echo
echo "Wrote:"
echo "  $GEN_ENV"
echo "  $DEPLOY_DIR/e2e.maker1.env"
echo "  $DEPLOY_DIR/e2e.maker2.env"
echo "  $DEPLOY_DIR/e2e.maker3.env"
echo "  $WALLET_JSON"
echo
echo "MwixnetRegistry: $REGISTRY_LC"
echo "GrievanceCourt:  $COURT_LC"
echo "Maker operators: $ADDR1_LC $ADDR2_LC $ADDR3_LC"
echo
echo "Next: docker compose -f deploy/docker-compose.e2e.yml --profile makers up -d --build"
echo "Wallet: merge $WALLET_JSON into MLN Wallet network settings (see PHASE_12_E2E_CRUCIBLE.md)."
