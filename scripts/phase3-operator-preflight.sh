#!/usr/bin/env bash
# Phase 3 operator preflight: Tor SOCKS + recommended coinswapd proxy exports +
# optional JSON-RPC reachability to one .onion (curl --socks5-hostname).
#
# Usage:
#   ./scripts/phase3-operator-preflight.sh
#   TOR_SOCKS_PORT=9150 ./scripts/phase3-operator-preflight.sh
#
# Optional 1-hop JSON-RPC (must look like http://host:port for curl):
#   PHASE3_ONION_JSONRPC_URL=http://xxxx.onion:8334 ./scripts/phase3-operator-preflight.sh
#
# See: research/PHASE_3_OPERATOR_CHECKLIST.md, research/PHASE_3_TOR_OPERATOR_LAB.md

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

TOR_SOCKS_HOST="${TOR_SOCKS_HOST:-127.0.0.1}"
TOR_SOCKS_PORT="${TOR_SOCKS_PORT:-9050}"
export TOR_SOCKS_HOST TOR_SOCKS_PORT

echo "Running base Tor preflight..."
"${ROOT}/scripts/tor-preflight.sh" "$@"

SOCKS_URL="socks5h://${TOR_SOCKS_HOST}:${TOR_SOCKS_PORT}"

echo ""
echo "=== Recommended exports for coinswapd (inter-hop .onion JSON-RPC) ==="
echo "Copy into the shell or systemd unit that starts coinswapd-research:"
echo ""
echo "  export HTTP_PROXY=\"${SOCKS_URL}\""
echo "  export HTTPS_PROXY=\"${SOCKS_URL}\""
echo "  export NO_PROXY=\"127.0.0.1,localhost\""
echo ""

if [[ -n "${PHASE3_ONION_JSONRPC_URL:-}" ]]; then
	echo "=== Optional: POST JSON-RPC to PHASE3_ONION_JSONRPC_URL via Tor (curl socks5h) ==="
	if ! command -v curl >/dev/null 2>&1; then
		echo "warn: curl not found; skipping PHASE3_ONION_JSONRPC_URL check." >&2
	else
		body='{"jsonrpc":"2.0","method":"mweb_getBalance","params":[],"id":1}'
		if out="$(curl --proto '=http' --socks5-hostname "${TOR_SOCKS_HOST}:${TOR_SOCKS_PORT}" -fsS -m 90 \
			-H 'Content-Type: application/json' -d "${body}" "${PHASE3_ONION_JSONRPC_URL}" 2>&1)"; then
			echo "ok: HTTP response from maker JSON-RPC (truncated):"
			echo "${out}" | head -c 400
			echo
		else
			echo "error: could not complete JSON-RPC POST to ${PHASE3_ONION_JSONRPC_URL}" >&2
			echo "  ${out}" >&2
			echo "  Check HS port, Tor circuit, and URL scheme (http://…)." >&2
			exit 1
		fi
	fi
else
	echo "(Set PHASE3_ONION_JSONRPC_URL=http://yourmaker.onion:PORT for optional 1-hop RPC check.)"
fi

echo "=== phase3-operator-preflight done ==="
