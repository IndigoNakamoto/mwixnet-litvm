#!/usr/bin/env bash
# Tier 1 post-bootstrap dashboard: prints a single-page summary of the local E2E stack.
#
# Checks:
#   1) Anvil / LitVM RPC reachable (eth_chainId)
#   2) Nostr relay reachable (TCP connect)
#   3) mln-sidecar /v1/balance healthy (if running)
#   4) mln-cli scout — verified makers + stake + Tor endpoint
#
# Sources deploy/e2e.generated.env if present; allows override via MLN_* env.
# Non-fatal: exits 0 even if individual checks fail, so it can be folded into bootstrap pipelines.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="$ROOT/deploy"
GEN_ENV="$DEPLOY_DIR/e2e.generated.env"

bold() { printf '\033[1m%s\033[0m\n' "$1"; }
ok()   { printf '  [\033[32m OK \033[0m] %s\n' "$1"; }
bad()  { printf '  [\033[31mFAIL\033[0m] %s\n' "$1"; }
note() { printf '  [info] %s\n' "$1"; }

if [[ -f "$GEN_ENV" ]]; then
  # shellcheck disable=SC1090
  set -a; . "$GEN_ENV"; set +a
fi

LITVM_HTTP="${MLN_LITVM_HTTP_URL:-${E2E_ANVIL_HTTP:-http://127.0.0.1:8545}}"
NOSTR_WS="${MLN_NOSTR_RELAYS:-${E2E_NOSTR_RELAY_WS:-ws://127.0.0.1:7080/}}"
REGISTRY="${MLN_REGISTRY_ADDR:-${E2E_MWIXNET_REGISTRY:-}}"
COURT="${MLN_GRIEVANCE_COURT_ADDR:-${E2E_GRIEVANCE_COURT:-}}"
CHAIN_ID="${MLN_LITVM_CHAIN_ID:-${E2E_CHAIN_ID:-}}"
SIDECAR_URL="${MLN_SIDECAR_URL:-http://127.0.0.1:8080}"

bold "MLN E2E status"
echo "  LitVM HTTP   : $LITVM_HTTP"
echo "  Nostr relay  : $NOSTR_WS"
echo "  Chain ID     : ${CHAIN_ID:-(unset)}"
echo "  Registry     : ${REGISTRY:-(unset)}"
echo "  Court        : ${COURT:-(unset)}"
echo "  Sidecar      : $SIDECAR_URL"
echo

bold "1) LitVM RPC"
if command -v curl >/dev/null 2>&1; then
  resp="$(curl -sS -m 3 -X POST -H 'content-type: application/json' \
    --data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' \
    "$LITVM_HTTP" 2>/dev/null || true)"
  if [[ "$resp" == *"\"result\""* ]]; then
    hex="$(printf '%s' "$resp" | sed -n 's/.*"result":"\(0x[0-9a-fA-F]*\)".*/\1/p')"
    if [[ -n "$hex" ]]; then
      dec="$(printf '%d' "$hex" 2>/dev/null || echo '?')"
      ok "eth_chainId = $hex ($dec)"
      if [[ -n "${CHAIN_ID:-}" && "$dec" != "$CHAIN_ID" ]]; then
        bad "chain id mismatch: RPC reports $dec, env has $CHAIN_ID"
      fi
    else
      bad "could not parse eth_chainId from $resp"
    fi
  else
    bad "no eth_chainId response from $LITVM_HTTP"
  fi
else
  note "curl not installed — skipping"
fi
echo

bold "2) Nostr relay"
# Extract host:port from ws(s)://host:port/...
host_port="$(printf '%s' "$NOSTR_WS" | sed -E 's#^wss?://##; s#/.*##')"
host="${host_port%%:*}"
port="${host_port##*:}"
if [[ "$host" == "$port" ]]; then port="80"; fi
if [[ -z "$host" ]]; then
  bad "unable to parse host from $NOSTR_WS"
elif command -v nc >/dev/null 2>&1; then
  if nc -z -w 2 "$host" "$port" 2>/dev/null; then
    ok "TCP connect $host:$port"
  else
    bad "cannot reach $host:$port"
  fi
else
  note "nc not installed — skipping TCP probe"
fi
echo

bold "3) mln-sidecar /v1/balance"
if command -v curl >/dev/null 2>&1; then
  if resp="$(curl -sS -m 3 "$SIDECAR_URL/v1/balance" 2>/dev/null)"; then
    if [[ "$resp" == *'"ok"'* || "$resp" == *'spendableSat'* ]]; then
      ok "$resp"
    else
      bad "unexpected response: ${resp:0:120}"
    fi
  else
    note "sidecar not running (ok for scout-only sanity)"
  fi
fi
echo

bold "4) mln-cli scout"
CLI="${MLN_CLI_BIN:-$ROOT/bin/mln-cli}"
if [[ ! -x "$CLI" ]]; then
  note "$CLI not built — run 'make build-mln-cli'"
else
  export MLN_NOSTR_RELAYS="${MLN_NOSTR_RELAYS:-$NOSTR_WS}"
  export MLN_LITVM_HTTP_URL="$LITVM_HTTP"
  export MLN_REGISTRY_ADDR="${REGISTRY:-}"
  [[ -n "$CHAIN_ID" ]] && export MLN_LITVM_CHAIN_ID="$CHAIN_ID"
  [[ -n "$COURT" ]] && export MLN_GRIEVANCE_COURT_ADDR="$COURT"
  if [[ -z "$REGISTRY" || -z "$CHAIN_ID" ]]; then
    note "registry/chain id unset — skipping scout"
  else
    if out="$("$CLI" scout -json -quiet 2>/dev/null)"; then
      if command -v jq >/dev/null 2>&1; then
        count="$(printf '%s' "$out" | jq '.verified | length' 2>/dev/null || echo '?')"
        if [[ "$count" =~ ^[0-9]+$ && "$count" -gt 0 ]]; then
          ok "$count verified makers"
          printf '\n  %-42s %-24s %s\n' "OPERATOR" "STAKE" "TOR"
          printf '%s\n' "$out" | jq -r '.verified[] | "  \(.operator)   \(.stake // "?")   \(.tor // "(no tor)")"'
        else
          bad "0 verified makers (see mln-cli scout without -quiet for rejection reasons)"
        fi
      else
        ok "scout returned JSON (install jq for per-maker table)"
      fi
    else
      bad "mln-cli scout failed"
    fi
  fi
fi

echo
bold "Next"
echo "  Tier 1: E2E_MWEB_FULL=1 ./scripts/e2e-mweb-handoff-stub.sh  (or: make e2e-tier1)"
echo "  Tier 2: scripts/phase3-tier2-setup.sh + research/PHASE_3_TIER2_RELAY.md"
echo "  Tier 3: make phase3-funded-preflight  + research/PHASE_3_OPERATOR_CHECKLIST.md D"
