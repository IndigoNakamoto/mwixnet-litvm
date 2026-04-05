#!/usr/bin/env bash
# Tor SOCKS preflight for Phase 3 live .onion labs (see research/PHASE_3_TOR_OPERATOR_LAB.md).
# Verifies Tor is listening and (optionally) routes traffic via the SOCKS port.
#
# Usage:
#   ./scripts/tor-preflight.sh
#   TOR_SOCKS_HOST=127.0.0.1 TOR_SOCKS_PORT=9150 ./scripts/tor-preflight.sh   # Tor Browser
#   TOR_PREFLIGHT_SKIP_CURL=1 ./scripts/tor-preflight.sh                      # TCP only
#
# Exit: 0 = checks passed, 1 = failure

set -euo pipefail

TOR_SOCKS_HOST="${TOR_SOCKS_HOST:-127.0.0.1}"
TOR_SOCKS_PORT="${TOR_SOCKS_PORT:-9050}"
SKIP_CURL="${TOR_PREFLIGHT_SKIP_CURL:-0}"

usage() {
	sed -n '2,12p' "$0" | tr -d '#'
	exit 0
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
fi

echo "=== MLN Tor preflight (SOCKS ${TOR_SOCKS_HOST}:${TOR_SOCKS_PORT}) ==="

# 1) TCP: Tor SOCKS port accepting connections
if command -v nc >/dev/null 2>&1; then
	if ! nc -z -w 3 "${TOR_SOCKS_HOST}" "${TOR_SOCKS_PORT}" 2>/dev/null; then
		echo "error: no service accepting connections on ${TOR_SOCKS_HOST}:${TOR_SOCKS_PORT}" >&2
		echo "  Start Tor (system tor → 9050, Tor Browser → 9150) or set TOR_SOCKS_HOST / TOR_SOCKS_PORT." >&2
		exit 1
	fi
	echo "ok: TCP connect to SOCKS port succeeded (nc)."
elif exec 3<>"/dev/tcp/${TOR_SOCKS_HOST}/${TOR_SOCKS_PORT}" 2>/dev/null; then
	exec 3<&- 3>&-
	echo "ok: TCP connect to SOCKS port succeeded (/dev/tcp)."
else
	echo "error: cannot reach ${TOR_SOCKS_HOST}:${TOR_SOCKS_PORT} (install netcat or use bash with /dev/tcp)." >&2
	exit 1
fi

# 2) Optional: prove Tor is routing (needs outbound network from the Tor exit)
if [[ "${SKIP_CURL}" == "1" ]]; then
	echo "skip: TOR_PREFLIGHT_SKIP_CURL=1 — not verifying exit routing (curl)."
	echo "=== Tor preflight passed (TCP only). ==="
	exit 0
fi

if ! command -v curl >/dev/null 2>&1; then
	echo "warn: curl not found; skipping exit routing check. Install curl or set TOR_PREFLIGHT_SKIP_CURL=1." >&2
	echo "=== Tor preflight passed (TCP only). ==="
	exit 0
fi

SOCKS_URL="socks5h://${TOR_SOCKS_HOST}:${TOR_SOCKS_PORT}"
# socks5h: hostname resolved at the proxy (same idea as Go socks5h — .onion must not hit local DNS)
if ! out="$(curl --proto '=https' --socks5-hostname "${TOR_SOCKS_HOST}:${TOR_SOCKS_PORT}" -fsS -m 45 https://check.torproject.org/api/ip 2>&1)"; then
	echo "error: curl via Tor SOCKS failed — Tor may not be routing, or network blocked." >&2
	echo "  ${out}" >&2
	echo "  For TCP-only check: TOR_PREFLIGHT_SKIP_CURL=1 $0" >&2
	exit 1
fi

echo "ok: HTTPS via Tor succeeded (check.torproject.org)."
echo "${out}" | head -c 200
echo
echo "=== Tor preflight passed. ==="
echo "Reminder: go-ethereum JSON-RPC over http:// uses Go net/http — set HTTP_PROXY=${SOCKS_URL} (and NO_PROXY as needed) on coinswapd, not only ALL_PROXY."
