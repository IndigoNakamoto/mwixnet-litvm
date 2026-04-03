#!/usr/bin/env bash
# Phase 3a: mln-sidecar -mode=rpc → mweb_* on host (mw-rpc-stub or coinswapd), without official LitVM testnet.
# Prereqs: Docker, make, curl; Go toolchain for stub/mln-cli builds.
#
# Usage:
#   ./scripts/e2e-mweb-handoff-stub.sh
#     → stub on :8546, Anvil + relay + rpc sidecar; curl /v1/balance and /v1/swap.
#   E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh
#     → also bootstrap, start makers, mln-cli pathfind + forger (builds mln-cli if needed).
#
# See PHASE_3_MWEB_HANDOFF_SLICE.md

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_BASE="$ROOT/deploy/docker-compose.e2e.yml"
COMPOSE_RPC="$ROOT/deploy/docker-compose.e2e.sidecar-rpc.yml"
BIN_STUB="$ROOT/bin/mw-rpc-stub"
STUB_ADDR="${STUB_ADDR:-:8546}"

cd "$ROOT"

make build-mw-rpc-stub

STUB_PID=""
cleanup() {
	if [[ -n "${STUB_PID}" ]] && kill -0 "${STUB_PID}" 2>/dev/null; then
		kill "${STUB_PID}" 2>/dev/null || true
		wait "${STUB_PID}" 2>/dev/null || true
	fi
}
trap cleanup EXIT

echo "=== Starting mw-rpc-stub on ${STUB_ADDR} ==="
"${BIN_STUB}" -addr "${STUB_ADDR}" &
STUB_PID=$!
sleep 0.3

echo "=== Docker compose (e2e + sidecar rpc override) ==="
docker compose -f "${COMPOSE_BASE}" -f "${COMPOSE_RPC}" up -d

echo "=== Waiting for sidecar /v1/balance ==="
ok=0
for _ in $(seq 1 45); do
	if curl -sf "http://127.0.0.1:8080/v1/balance" >/dev/null 2>&1; then
		ok=1
		break
	fi
	sleep 1
done
if [[ "${ok}" -ne 1 ]]; then
	echo "sidecar did not become ready on http://127.0.0.1:8080" >&2
	exit 1
fi

bal_json=$(curl -sf "http://127.0.0.1:8080/v1/balance")
echo "${bal_json}" | grep -q '"ok":true' || {
	echo "unexpected balance response: ${bal_json}" >&2
	exit 1
}
echo "${bal_json}" | grep -q 'availableSat' || {
	echo "missing availableSat: ${bal_json}" >&2
	exit 1
}
echo "GET /v1/balance OK"

swap_json=$(curl -sf -X POST "http://127.0.0.1:8080/v1/swap" \
	-H "Content-Type: application/json" \
	-d '{"route":[{"tor":"http://n1","feeMinSat":1},{"tor":"http://n2","feeMinSat":2},{"tor":"http://n3","feeMinSat":3}],"destination":"mweb1x","amount":1000000}')
echo "${swap_json}" | grep -q '"ok":true' || {
	echo "unexpected swap response: ${swap_json}" >&2
	exit 1
}
echo "POST /v1/swap OK"

if [[ "${E2E_MWEB_FULL:-}" == "1" ]]; then
	echo "=== E2E_MWEB_FULL: bootstrap + makers + mln-cli pathfind/forger ==="
	./scripts/e2e-bootstrap.sh
	docker compose -f "${COMPOSE_BASE}" -f "${COMPOSE_RPC}" --profile makers up -d --build

	# Allow maker ads to propagate
	sleep 8

	# shellcheck source=/dev/null
	source "${ROOT}/deploy/e2e.generated.env"
	export MLN_NOSTR_RELAYS="${E2E_NOSTR_RELAY_WS}"
	export MLN_LITVM_CHAIN_ID="${E2E_CHAIN_ID}"
	export MLN_LITVM_HTTP_URL="${E2E_ANVIL_HTTP}"
	export MLN_REGISTRY_ADDR="${E2E_MWIXNET_REGISTRY}"
	export MLN_GRIEVANCE_COURT_ADDR="${E2E_GRIEVANCE_COURT}"
	export MLN_SCOUT_TIMEOUT="${MLN_SCOUT_TIMEOUT:-45s}"

	make build-mln-cli
	ROUTE_JSON="${ROOT}/deploy/e2e.mweb-handoff.route.json"
	"${ROOT}/bin/mln-cli" pathfind -json >"${ROUTE_JSON}"
	"${ROOT}/bin/mln-cli" forger -route-json "${ROUTE_JSON}" -dry-run=false \
		-dest "mweb1x" \
		-amount 1000000 \
		-coinswapd-url "http://127.0.0.1:8080/v1/swap"
	echo "mln-cli forger (rpc sidecar) OK"
fi

echo
echo "Phase 3a stub handoff checks passed."
echo "Stub process exits with this script; stack still running — tear down with:"
echo "  docker compose -f ${COMPOSE_BASE} -f ${COMPOSE_RPC} down"
