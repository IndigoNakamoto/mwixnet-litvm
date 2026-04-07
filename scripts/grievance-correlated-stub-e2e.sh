#!/usr/bin/env bash
# Helper: Phase 3a RPC stub path with correlator-aligned golden receipts (accused = hop[0].operator).
# See PHASE_3_MWEB_HANDOFF_SLICE.md — "Correlator-aligned receipts and maker auto-defend".
#
# Prerequisites:
#   - Anvil + nostr + mln-sidecar (-mode=rpc) + makers + mw-rpc-stub on :8546 (e.g. same compose as e2e-mweb-handoff-stub.sh).
#   - ./scripts/e2e-bootstrap.sh → deploy/e2e.generated.env
#   - jq; make + Go for mln-cli when using --run-forger
#
# Usage:
#   ./scripts/grievance-correlated-stub-e2e.sh              # map env, route build, print mlnd + grievance hints
#   CORRELATED_RUN_FORGER=1 ./scripts/grievance-correlated-stub-e2e.sh   # also run forger -vault -trigger-batch -wait-batch
#
# No secret material is written by this script; export keys only in your shell (see e2e-bootstrap well-known Anvil keys).

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GEN_ENV="${CORRELATED_GEN_ENV:-$ROOT/deploy/e2e.generated.env}"
ROUTE_JSON="${CORRELATED_ROUTE_JSON:-/tmp/grievance-correlated-route.json}"
VAULT_DB="${CORRELATED_VAULT:-/tmp/grievance-correlated-vault.db}"
SIDECAR_URL="${CORRELATED_SIDECAR_URL:-http://127.0.0.1:8080/v1/swap}"
DEST="${CORRELATED_DEST:-ltcmweb1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq0}"
AMOUNT="${CORRELATED_AMOUNT:-1000000}"
# Default accuser: Anvil account #0 (same deployer default as e2e-bootstrap PRIVATE_KEY); override MLN_ACCUSER_ETH_KEY in environment.
ACCUSER_KEY="${MLN_ACCUSER_ETH_KEY:-0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80}"
EPOCH_ID="${MLN_RECEIPT_EPOCH_ID:-1}"

usage() {
	sed -n '2,/^set -euo pipefail$/p' "$0" | tail -n +2
}

case "${1:-}" in
-h | --help)
	usage
	exit 0
	;;
esac

if ! command -v jq &>/dev/null; then
	echo "error: jq is required" >&2
	exit 1
fi

if [[ ! -f "$GEN_ENV" ]]; then
	echo "error: missing $GEN_ENV (run ./scripts/e2e-bootstrap.sh with Anvil up)" >&2
	exit 1
fi

# shellcheck source=/dev/null
set -a
source "$GEN_ENV"
set +a

export MLN_NOSTR_RELAYS="${E2E_NOSTR_RELAY_WS}"
export MLN_LITVM_CHAIN_ID="${E2E_CHAIN_ID}"
export MLN_LITVM_HTTP_URL="${E2E_ANVIL_HTTP}"
export MLN_REGISTRY_ADDR="${E2E_MWIXNET_REGISTRY}"
export MLN_GRIEVANCE_COURT_ADDR="${E2E_GRIEVANCE_COURT}"
export MLN_SCOUT_TIMEOUT="${MLN_SCOUT_TIMEOUT:-45s}"
export MLN_ACCUSER_ETH_KEY="$ACCUSER_KEY"
export MLN_RECEIPT_EPOCH_ID="$EPOCH_ID"

cd "$ROOT"
if [[ ! -x "$ROOT/bin/mln-cli" ]]; then
	make build-mln-cli
fi

echo "=== mln-cli route build → $ROUTE_JSON ==="
"$ROOT/bin/mln-cli" route build -out "$ROUTE_JSON"

N1="$(jq -r '.hops[0].operator // empty' "$ROUTE_JSON")"
if [[ -z "$N1" || "$N1" == "null" ]]; then
	echo "error: route has no .hops[0].operator (need live makers + scout)" >&2
	exit 1
fi

echo
echo "First-hop operator (N1, must match receipt accusedMaker / defend): $N1"
echo "Shared vault path (forger -vault and mlnd MLND_DB_PATH): $VAULT_DB"
echo
echo "Start mlnd for N1 *before* openGrievance, e.g.:"
echo "  export MLND_WS_URL=${E2E_ANVIL_WS}"
echo "  export MLND_COURT_ADDR=${E2E_GRIEVANCE_COURT}"
echo "  export MLND_REGISTRY_ADDR=${E2E_MWIXNET_REGISTRY}"
echo "  export MLND_LITVM_CHAIN_ID=${E2E_CHAIN_ID}"
echo "  export MLND_OPERATOR_ADDR=$N1"
echo "  export MLND_DB_PATH=$VAULT_DB"
echo "  export MLND_DEFEND_AUTO=1"
echo "  # export MLND_DEFEND_DRY_RUN=1   # optional: log defense only"
echo "  export MLND_OPERATOR_PRIVATE_KEY='<64-hex from deploy/e2e.maker?.env for this operator>'"
echo "  # bin/mlnd   (from repo root after make build-mlnd)"
echo
echo "Then file grievance (flags before swap_id):"
echo "  # export same MLN_* as for route build + MLN_PRIVATE_KEY or MLN_ACCUSER_ETH_KEY for tx"
echo "  # bin/mln-cli grievance file -vault $VAULT_DB '<swap_id_from_forger>'"
echo

if [[ "${CORRELATED_RUN_FORGER:-}" == "1" ]]; then
	echo "=== CORRELATED_RUN_FORGER: forger with vault (stub golden receipt) ==="
	rm -f "$VAULT_DB"
	FORGER_OUT="$(
		"$ROOT/bin/mln-cli" forger \
			-route-json "$ROUTE_JSON" \
			-dry-run=false \
			-dest "$DEST" \
			-amount "$AMOUNT" \
			-coinswapd-url "$SIDECAR_URL" \
			-trigger-batch \
			-wait-batch \
			-vault "$VAULT_DB" 2>&1
	)" || {
		echo "$FORGER_OUT"
		exit 1
	}
	echo "$FORGER_OUT"
	SWAP_ID="$(echo "$FORGER_OUT" | sed -n 's/.*swap_id=\([^ ]*\) .*/\1/p' | head -1)"
	if [[ -z "$SWAP_ID" ]]; then
		echo "warning: could not parse swap_id from forger output; check Receipt vault line manually" >&2
	else
		echo
		echo "Next (after mlnd is up): bin/mln-cli grievance file -vault $VAULT_DB $SWAP_ID"
	fi
	echo
	echo "mln-judge: with a real defendGrievance (BuildDefenseData), verify decode via JUDGE_DRY_RUN=1 per mln-judge/README.md"
fi
