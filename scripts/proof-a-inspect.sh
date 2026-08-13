#!/usr/bin/env bash
# Unlock the Mac wallet data dir and print MWEB totals + unspent coins (no secrets).
# Requires WALLET_PASSPHRASE in the environment or in deploy/mixnet.operator.env (chmod 600).
# Quit the Tauri GUI first (exclusive data-dir lock).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${PROOF_A_OPERATOR_ENV:-$ROOT/deploy/mixnet.operator.env}"
DATA_DIR="${WALLET_DATA_DIR:-$HOME/Library/Application Support/com.indigonakamoto.ltc-wallet}"
WALLET_CLI="${WALLET_CLI:-}"
WALLET_MAC="${WALLET_MAC:-$HOME/Dev/ltc-wallet-mac}"

# Parse KEY=value without sourcing (avoids set -u / unquoted-value footguns). Never print values.
load_operator_env() {
  local path="$1"
  [[ -f "$path" ]] || return 0
  eval "$(python3 - "$path" << 'PY'
import os, shlex, sys
from pathlib import Path
path = Path(sys.argv[1])
wanted = (
    "WALLET_PASSPHRASE",
    "MWEB_SCAN_SECRET",
    "MWEB_SPEND_SECRET",
    "E2E_MWEB_DEST",
    "MIX_K0",
    "MIX_K1",
    "MIX_K2",
)
for line in path.read_text().splitlines():
    s = line.strip()
    if not s or s.startswith("#") or "=" not in s:
        continue
    key, _, val = s.partition("=")
    key = key.strip()
    if key not in wanted:
        continue
    val = val.strip()
    if (val.startswith('"') and val.endswith('"')) or (val.startswith("'") and val.endswith("'")):
        val = val[1:-1]
    print(f"export {key}={shlex.quote(val)}")
PY
)"
}

load_operator_env "$ENV_FILE"

if [[ -z "$WALLET_CLI" && -x "$WALLET_MAC/target/debug/wallet-cli" ]]; then
  WALLET_CLI="$WALLET_MAC/target/debug/wallet-cli"
fi

if [[ -z "${WALLET_PASSPHRASE:-}" ]]; then
  echo "error: WALLET_PASSPHRASE is not set. Put it in $ENV_FILE (chmod 600) or export it." >&2
  echo "note: use deploy/mixnet.operator.env (gitignored), not the .example file." >&2
  exit 1
fi

run_wallet_cli() {
  if [[ -n "${WALLET_CLI:-}" && -x "$WALLET_CLI" ]]; then
    "$WALLET_CLI" --data-dir "$DATA_DIR" "$@"
    return
  fi
  if [[ -f "$WALLET_MAC/Cargo.toml" ]]; then
    cargo run --manifest-path "$WALLET_MAC/Cargo.toml" -q -p wallet-cli -- --data-dir "$DATA_DIR" "$@"
    return
  fi
  echo "error: wallet-cli not found (set WALLET_CLI or cargo build -p wallet-cli in ltc-wallet-mac)" >&2
  exit 1
}

echo "=== combined-summary ==="
run_wallet_cli combined-summary
echo "=== mweb-unspent ==="
run_wallet_cli mweb-unspent
echo "Pick one mature unlocked coin with amount_sats > 30000 (3 x feeMinSat)."
echo "Do not spend until that amount is confirmed. Then:"
echo "  cargo run --manifest-path \"$WALLET_MAC/Cargo.toml\" -p wallet-cli -- --data-dir \"$DATA_DIR\" mweb-coinswapd-env --out $ENV_FILE"
