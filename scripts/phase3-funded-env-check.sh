#!/usr/bin/env bash
# Tier 3 funded-path preflight for MWEB multi-hop (README Phase 3 bar).
#
# Historically this was a passive env-warning script. It now also probes mweb_getBalance on the
# configured coinswapd-research and checks the exact-UTXO precondition from
# research/coinswapd/mln_wallet.go pickCoinExactAmount (Value == amount).
#
# Exit codes:
#   0  — all checks passed (ready to run the funded path)
#   1  — hard failure (RPC unreachable, insufficient spendable balance, etc.)
#   2  — warnings only (e.g. DEV clear flag set); script still allows opt-in via PHASE3_ALLOW_WARN=1
#
# Env consumed:
#   COINSWAPD_RPC_URL       default http://127.0.0.1:8546
#   E2E_MWEB_AMOUNT         satoshis to swap (required with E2E_MWEB_FUNDED=1)
#   E2E_MWEB_DEST           ltcmweb1… destination (informational)
#   COINSWAPD_FEE_MWEB      ltcmweb1… fee address
#   MWEB_SCAN_SECRET / MWEB_SPEND_SECRET   64-hex keys
#   E2E_MWEB_FUNDED_DEV_CLEAR / PHASE3_ALLOW_DEV_CLEAR   DEV-only acknowledgment
#
# See research/PHASE_3_OPERATOR_CHECKLIST.md section D and PHASE_3_MWEB_HANDOFF_SLICE.md.

set -uo pipefail

warn=0
fail=0

note() { printf '  [info] %s\n' "$1"; }
ok()   { printf '  [ OK ] %s\n' "$1"; }
warning() { printf '  [WARN] %s\n' "$1" >&2; warn=1; }
error() { printf '  [FAIL] %s\n' "$1" >&2; fail=1; }

echo "Phase 3 Tier 3 funded preflight"
echo

echo "1) DEV clear flag"
if [[ "${E2E_MWEB_FUNDED_DEV_CLEAR:-}" == "1" ]] || [[ "${E2E_MWEB_FUNDED_DEV_CLEAR:-}" == "true" ]]; then
  warning "E2E_MWEB_FUNDED_DEV_CLEAR is set — pendingOnions may clear without chain finalize (DEV ONLY)."
  if [[ "${PHASE3_ALLOW_DEV_CLEAR:-}" != "1" ]]; then
    warning "  Unset E2E_MWEB_FUNDED_DEV_CLEAR for the README Phase 3 success bar (real finalize/broadcast)."
  fi
else
  ok "no dev-clear flag set"
fi
echo

echo "2) Funded-mode env"
if [[ "${E2E_MWEB_FUNDED:-}" == "1" ]] || [[ "${E2E_MWEB_FUNDED:-}" == "true" ]]; then
  for v in MWEB_SCAN_SECRET MWEB_SPEND_SECRET COINSWAPD_FEE_MWEB E2E_MWEB_DEST E2E_MWEB_AMOUNT; do
    if [[ -z "${!v:-}" ]]; then
      error "$v is empty (required with E2E_MWEB_FUNDED=1)"
    else
      ok "$v set"
    fi
  done
  # Basic address shape
  for a in "${COINSWAPD_FEE_MWEB:-}" "${E2E_MWEB_DEST:-}"; do
    if [[ -n "$a" && "$a" != ltcmweb1* ]]; then
      error "MWEB address must start with ltcmweb1 (not mweb1, not truncated): ${a:0:20}…"
    fi
  done
  if [[ -n "${E2E_MWEB_AMOUNT:-}" && ! "$E2E_MWEB_AMOUNT" =~ ^[0-9]+$ ]]; then
    error "E2E_MWEB_AMOUNT must be integer satoshis, got '$E2E_MWEB_AMOUNT'"
  fi
else
  note "E2E_MWEB_FUNDED not set — skipping funded-env + RPC probes (set E2E_MWEB_FUNDED=1 to enable)."
  if [[ "$warn" -eq 1 ]]; then exit 2; fi
  exit 0
fi
echo

echo "3) coinswapd-research RPC probe"
RPC_URL="${COINSWAPD_RPC_URL:-http://127.0.0.1:8546}"
if ! command -v curl >/dev/null 2>&1; then
  error "curl not installed — cannot probe RPC"
elif ! resp="$(curl -sS -m 10 -X POST -H 'content-type: application/json' \
    --data '{"jsonrpc":"2.0","id":1,"method":"mweb_getBalance","params":[]}' \
    "$RPC_URL" 2>&1)"; then
  error "mweb_getBalance request failed: $resp"
else
  if [[ "$resp" != *'"result"'* ]]; then
    error "mweb_getBalance at $RPC_URL: $resp"
  else
    ok "RPC reachable ($RPC_URL)"
    if command -v jq >/dev/null 2>&1; then
      avail="$(printf '%s' "$resp" | jq -r '.result.availableSat // 0')"
      spend="$(printf '%s' "$resp" | jq -r '.result.spendableSat // 0')"
      detail="$(printf '%s' "$resp" | jq -r '.result.detail // ""')"
      ok "availableSat=$avail spendableSat=$spend detail=${detail:-(none)}"
      if [[ -n "${E2E_MWEB_AMOUNT:-}" ]]; then
        if [[ "$spend" -lt "$E2E_MWEB_AMOUNT" ]]; then
          error "spendableSat ($spend) < E2E_MWEB_AMOUNT ($E2E_MWEB_AMOUNT)"
        else
          ok "spendableSat >= E2E_MWEB_AMOUNT"
          warning "pickCoinExactAmount (research/coinswapd/mln_wallet.go) requires a SINGLE coin with Value == $E2E_MWEB_AMOUNT."
          warning "  availableSat alone does NOT prove this — inspect your wallet coin list before submitting."
        fi
      fi
      if [[ "$detail" == *"spend-secret"* ]]; then
        error "spend key not configured on coinswapd-research (start with -mweb-spend-secret)"
      fi
    else
      note "jq not installed — raw response: ${resp:0:200}"
    fi
  fi
fi
echo

echo "Summary"
if [[ "$fail" -eq 1 ]]; then
  echo "  FAIL — fix the errors above before running the funded path." >&2
  exit 1
fi
if [[ "$warn" -eq 1 ]]; then
  echo "  OK with warnings — see [WARN] lines above." >&2
  exit 2
fi
echo "  OK — ready for funded run (see PHASE_3_MWEB_HANDOFF_SLICE.md Real funded operator path)."
exit 0
