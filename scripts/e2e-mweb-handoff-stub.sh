#!/usr/bin/env bash
# Phase 3a: mln-sidecar -mode=rpc → mweb_* on host (mw-rpc-stub or research/coinswapd), without official LitVM testnet.
# Prereqs: Docker, make, curl; Go toolchain for stub/mln-cli/coinswapd builds.
#
# Usage:
#   ./scripts/e2e-mweb-handoff-stub.sh
#     → stub on :8546, Anvil + relay + rpc sidecar; curl /v1/balance and /v1/swap.
#   E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh
#     → also bootstrap, start makers, mln-cli pathfind + forger (builds mln-cli if needed).
#
# Optional real fork (Neutrino + keys; swap may fail without funded MWEB UTXO):
#   MWEB_RPC_BACKEND=coinswapd COINSWAPD_FEE_MWEB='ltcmweb1qq<full string from wallet>' ./scripts/e2e-mweb-handoff-stub.sh
#   ltcmweb/ltcd mainnet Bech32 HRP is "ltcmweb" → addresses start with ltcmweb1 (not mweb1). Paste the full string; no … or ...
#   (-a must decode as mainnet MWEB; bad strings make coinswapd exit: "decoded address is of unknown format")
#   E2E_MWEB_FULL=1 is not supported with coinswapd (forger needs wallet state); use stub for full CLI path.
#
# See PHASE_3_MWEB_HANDOFF_SLICE.md

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_BASE="$ROOT/deploy/docker-compose.e2e.yml"
COMPOSE_RPC="$ROOT/deploy/docker-compose.e2e.sidecar-rpc.yml"
BIN_STUB="$ROOT/bin/mw-rpc-stub"
BIN_COIN="$ROOT/bin/coinswapd-research"
STUB_ADDR="${STUB_ADDR:-:8546}"
MWEB_RPC_BACKEND="${MWEB_RPC_BACKEND:-stub}"
LISTEN_PORT="${STUB_ADDR#:}"

cd "$ROOT"

RPC_PID=""
cleanup() {
	if [[ -n "${RPC_PID}" ]] && kill -0 "${RPC_PID}" 2>/dev/null; then
		kill "${RPC_PID}" 2>/dev/null || true
		wait "${RPC_PID}" 2>/dev/null || true
	fi
}
trap cleanup EXIT

case "${MWEB_RPC_BACKEND}" in
stub)
	make build-mw-rpc-stub
	echo "=== Starting mw-rpc-stub on ${STUB_ADDR} ==="
	"${BIN_STUB}" -addr "${STUB_ADDR}" &
	RPC_PID=$!
	;;
coinswapd)
	if [[ "${E2E_MWEB_FULL:-}" == "1" ]]; then
		echo "E2E_MWEB_FULL with MWEB_RPC_BACKEND=coinswapd is unsupported (forger needs funded coinswapd wallet)." >&2
		exit 1
	fi
	: "${COINSWAPD_FEE_MWEB:?set COINSWAPD_FEE_MWEB to a mainnet MWEB stealth address (-a)}"
	case "${COINSWAPD_FEE_MWEB}" in
	*\<*|*\>*|*REPLACE*|*your-mweb*|*YOUR_REAL_*|*example.com*|*changeme*)
		echo "error: COINSWAPD_FEE_MWEB looks like a documentation placeholder, not a real address." >&2
		echo "  Use your wallet's mainnet MWEB receive string (Bech32; ltcmweb/ltcd → prefix ltcmweb1)." >&2
		exit 1
		;;
	# Unicode ellipsis (…) or ASCII ... are doc shorthand, not Bech32
	*…*|*...*)
		echo "error: COINSWAPD_FEE_MWEB contains … or ... — paste the complete address from your wallet, not an abbreviated example." >&2
		exit 1
		;;
	esac
	fee_lc="$(printf '%s' "${COINSWAPD_FEE_MWEB}" | tr '[:upper:]' '[:lower:]')"
	if [[ "${fee_lc}" != ltcmweb1* ]]; then
		echo "error: COINSWAPD_FEE_MWEB must be a mainnet MWEB stealth address (Bech32 prefix ltcmweb1)." >&2
		echo "  github.com/ltcmweb/ltcd chaincfg.MainNetParams uses Bech32HRPMweb=ltcmweb; coinswapd -a decodes with that." >&2
		echo "  Wallets following that fork show addresses starting with ltcmweb1; an mweb1 prefix will not decode here." >&2
		exit 1
	fi
	fee_len="${#COINSWAPD_FEE_MWEB}"
	if [[ "${fee_len}" -lt 42 ]]; then
		echo "error: COINSWAPD_FEE_MWEB is only ${fee_len} characters; a real mainnet MWEB stealth address is usually much longer." >&2
		echo "  Copy the full string from your wallet (no truncation)." >&2
		exit 1
	fi
	make build-research-coinswapd
	RANDK="$(openssl rand -hex 32)"
	MWEB_SCAN="${MWEB_SCAN_SECRET:-$(openssl rand -hex 32)}"
	MWEB_SPEND="${MWEB_SPEND_SECRET:-$(openssl rand -hex 32)}"
	echo "=== Starting coinswapd-research on :${LISTEN_PORT} (Neutrino; HTTP RPC when main returns) ==="
	"${BIN_COIN}" \
		-l "${LISTEN_PORT}" \
		-a "${COINSWAPD_FEE_MWEB}" \
		-k "${RANDK}" \
		-mweb-scan-secret "${MWEB_SCAN}" \
		-mweb-spend-secret "${MWEB_SPEND}" \
		-mln-local-taker \
		${COINSWAPD_EXTRA_FLAGS:-} &
	RPC_PID=$!
	echo "=== Waiting for coinswapd JSON-RPC on ${LISTEN_PORT} ==="
	ok_rpc=0
	for _ in $(seq 1 90); do
		if curl -sf "http://127.0.0.1:${LISTEN_PORT}/" \
			-H 'Content-Type: application/json' \
			-d '{"jsonrpc":"2.0","id":1,"method":"mweb_getBalance","params":[]}' | grep -q 'availableSat'; then
			ok_rpc=1
			break
		fi
		sleep 1
	done
	if [[ "${ok_rpc}" -ne 1 ]]; then
		echo "coinswapd did not answer mweb_getBalance on http://127.0.0.1:${LISTEN_PORT}/" >&2
		echo "hint: if the process exited immediately, check COINSWAPD_FEE_MWEB (full mainnet mweb1 address) and stderr above." >&2
		exit 1
	fi
	;;
*)
	echo "MWEB_RPC_BACKEND must be stub or coinswapd, got ${MWEB_RPC_BACKEND}" >&2
	exit 1
	;;
esac
if [[ "${MWEB_RPC_BACKEND}" == "stub" ]]; then
	sleep 0.3
fi

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

swap_payload='{"route":[{"tor":"http://n1","feeMinSat":1},{"tor":"http://n2","feeMinSat":2},{"tor":"http://n3","feeMinSat":3}],"destination":"mweb1x","amount":1000000}'

if [[ "${MWEB_RPC_BACKEND}" == "stub" ]]; then
	swap_json=$(curl -sf -X POST "http://127.0.0.1:8080/v1/swap" \
		-H "Content-Type: application/json" \
		-d "${swap_payload}")
	echo "${swap_json}" | grep -q '"ok":true' || {
		echo "unexpected swap response: ${swap_json}" >&2
		exit 1
	}
	echo "POST /v1/swap OK"
else
	# coinswapd: expect JSON-RPC / wallet failure (missing keys or UTXO), not transport failure.
	swap_code=$(curl -sS -o /tmp/mln-mweb-swap.out -w '%{http_code}' -X POST "http://127.0.0.1:8080/v1/swap" \
		-H "Content-Type: application/json" \
		-d "${swap_payload}" || true)
	swap_body=$(cat /tmp/mln-mweb-swap.out)
	if [[ "${swap_code}" == "502" ]] && echo "${swap_body}" | grep -q '"ok":false'; then
		echo "POST /v1/swap: sidecar returned 502 (expected without swapX25519PubHex / UTXO on real fork) — RPC path live."
	elif [[ "${swap_code}" == "200" ]] && echo "${swap_body}" | grep -q '"ok":true'; then
		echo "POST /v1/swap OK (unexpected success — wallet hit?)"
	else
		echo "unexpected swap: HTTP ${swap_code} body: ${swap_body}" >&2
		exit 1
	fi
fi

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
		-coinswapd-url "http://127.0.0.1:8080/v1/swap" \
		-trigger-batch -wait-batch \
		-batch-poll 500ms -batch-timeout 30s
	echo "mln-cli forger (rpc sidecar + batch/status) OK"
fi

echo
echo "Phase 3a stub handoff checks passed."
echo "RPC backend: ${MWEB_RPC_BACKEND}. Tear down stack with:"
echo "  docker compose -f ${COMPOSE_BASE} -f ${COMPOSE_RPC} down"
