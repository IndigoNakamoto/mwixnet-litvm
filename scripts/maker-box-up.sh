#!/usr/bin/env bash
# One-command maker-box bring-up for Tier 2 / Tier 3 on macOS (Apple Silicon or Intel).
#
# Reads deploy/tier2.maker-box.env (copy from deploy/tier2.maker-box.env.example, chmod 600).
# Starts coinswapd-research and mlnd in the foreground or as background processes.
# Assumes `tor` is already running via `brew services start tor` with the torrc fragment
# documented in deploy/MAKER_BOX_SETUP.md.
#
# Usage:
#   ./scripts/maker-box-up.sh              # build if needed, start both daemons in background
#   ./scripts/maker-box-up.sh foreground   # start coinswapd-research in foreground (mlnd in bg)
#   ./scripts/maker-box-up.sh stop         # stop the two daemons started here

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${MLN_MAKER_ENV:-$ROOT/deploy/tier2.maker-box.env}"
PIDS_DIR="${MLN_MAKER_PIDS:-$ROOT/deploy/.maker-box.pids}"
LOGS_DIR="${MLN_MAKER_LOGS:-$ROOT/deploy/.maker-box.logs}"

mkdir -p "$PIDS_DIR" "$LOGS_DIR"

mode="${1:-background}"

kill_if_running() {
  local pidfile="$1"
  if [[ -f "$pidfile" ]]; then
    local pid
    pid="$(cat "$pidfile" 2>/dev/null || true)"
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      echo "stopping pid $pid ($(basename "$pidfile"))"
      kill "$pid" || true
      # Give it a second to drain, then SIGKILL if needed.
      for _ in 1 2 3 4 5; do
        if ! kill -0 "$pid" 2>/dev/null; then break; fi
        sleep 1
      done
      kill -9 "$pid" 2>/dev/null || true
    fi
    rm -f "$pidfile"
  fi
}

if [[ "$mode" == "stop" ]]; then
  kill_if_running "$PIDS_DIR/mlnd.pid"
  kill_if_running "$PIDS_DIR/coinswapd.pid"
  echo "maker-box: stopped."
  exit 0
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "error: $ENV_FILE not found." >&2
  echo "Copy deploy/tier2.maker-box.env.example and fill in the THIS_MAKER values (chmod 600)." >&2
  exit 2
fi

# Safety: refuse to run a world-readable env file with secrets.
perm="$(stat -f '%A' "$ENV_FILE" 2>/dev/null || stat -c '%a' "$ENV_FILE" 2>/dev/null || echo '000')"
if [[ "$perm" != "600" && "$perm" != "400" ]]; then
  echo "warning: $ENV_FILE permissions are $perm; run 'chmod 600 $ENV_FILE' before starting daemons." >&2
fi

# shellcheck disable=SC1090
set -a; . "$ENV_FILE"; set +a

: "${MLND_OPERATOR_ADDR:?missing in $ENV_FILE}"
: "${MLND_OPERATOR_PRIVATE_KEY:?missing in $ENV_FILE}"
: "${MLND_NOSTR_NSEC:?missing in $ENV_FILE}"
: "${MLND_TOR_ONION:?missing in $ENV_FILE}"
: "${MLND_NOSTR_RELAYS:?missing in $ENV_FILE}"
: "${COINSWAPD_FEE_MWEB:?missing in $ENV_FILE}"
: "${COINSWAPD_MESH_K:?missing in $ENV_FILE}"
: "${MWEB_SCAN_SECRET:?missing in $ENV_FILE}"
: "${MWEB_SPEND_SECRET:?missing in $ENV_FILE}"

PORT="${COINSWAPD_LISTEN_PORT:-8334}"

# Cross-platform $HOME expansion (env file may embed $HOME via shell expansion; envsubst handled above by shell).

# Build binaries if missing.
if [[ ! -x "$ROOT/bin/coinswapd-research" ]]; then
  echo "building coinswapd-research..."
  (cd "$ROOT" && make build-research-coinswapd)
fi
if [[ ! -x "$ROOT/bin/mlnd" ]]; then
  echo "building mlnd..."
  (cd "$ROOT" && make build)
fi

# Torrc sanity (non-fatal).
HS_DIR_GUESS="/opt/homebrew/var/lib/tor/mln-maker"
[[ -d "/usr/local/var/lib/tor/mln-maker" ]] && HS_DIR_GUESS="/usr/local/var/lib/tor/mln-maker"
if [[ -d "$HS_DIR_GUESS" && -f "$HS_DIR_GUESS/hostname" ]]; then
  got="$(cat "$HS_DIR_GUESS/hostname" 2>/dev/null | tr -d '[:space:]')"
  if [[ -n "$got" ]]; then
    if [[ "$MLND_TOR_ONION" != *"$got"* ]]; then
      echo "warning: MLND_TOR_ONION ($MLND_TOR_ONION) does not contain Tor hostname ($got)" >&2
      echo "         update $ENV_FILE or your torrc so they match before publishing ads." >&2
    else
      echo "tor hidden service ok: $got"
    fi
  fi
else
  echo "note: could not locate Tor hidden service dir at $HS_DIR_GUESS; see deploy/MAKER_BOX_SETUP.md" >&2
fi

# Stop any prior instances we started.
kill_if_running "$PIDS_DIR/mlnd.pid"
kill_if_running "$PIDS_DIR/coinswapd.pid"

echo "starting coinswapd-research on 127.0.0.1:$PORT (fee $COINSWAPD_FEE_MWEB)"
CSD_LOG="$LOGS_DIR/coinswapd.log"
CSD_CMD=( "$ROOT/bin/coinswapd-research"
  -l "$PORT"
  -a "$COINSWAPD_FEE_MWEB"
  -k "$COINSWAPD_MESH_K"
  -mweb-scan-secret "$MWEB_SCAN_SECRET"
  -mweb-spend-secret "$MWEB_SPEND_SECRET"
)

if [[ "$mode" == "foreground" ]]; then
  echo "(foreground mode: coinswapd-research in this terminal; starting mlnd in background.)"
  # mlnd background first so foreground coinswapd log is unmixed.
  MLND_LOG="$LOGS_DIR/mlnd.log"
  nohup "$ROOT/bin/mlnd" >"$MLND_LOG" 2>&1 &
  echo $! > "$PIDS_DIR/mlnd.pid"
  echo "mlnd pid $(cat "$PIDS_DIR/mlnd.pid") (log: $MLND_LOG)"
  echo "coinswapd-research ->"
  exec "${CSD_CMD[@]}"
fi

nohup "${CSD_CMD[@]}" >"$CSD_LOG" 2>&1 &
echo $! > "$PIDS_DIR/coinswapd.pid"
echo "coinswapd-research pid $(cat "$PIDS_DIR/coinswapd.pid") (log: $CSD_LOG)"

# Wait for RPC to come up before starting mlnd — mlnd does not depend on it directly but
# this gives a clearer failure mode if coinswapd crashes on boot (keys/flags wrong).
for _ in 1 2 3 4 5; do
  if curl -fsS -m 1 -X POST -H 'content-type: application/json' \
       --data '{"jsonrpc":"2.0","id":1,"method":"mweb_getBalance","params":[]}' \
       "http://127.0.0.1:$PORT" >/dev/null 2>&1; then
    echo "coinswapd-research mweb_getBalance OK"
    break
  fi
  sleep 1
done

echo "starting mlnd (publishes kind 31250 ads to $MLND_NOSTR_RELAYS)"
MLND_LOG="$LOGS_DIR/mlnd.log"
nohup "$ROOT/bin/mlnd" >"$MLND_LOG" 2>&1 &
echo $! > "$PIDS_DIR/mlnd.pid"
echo "mlnd pid $(cat "$PIDS_DIR/mlnd.pid") (log: $MLND_LOG)"

cat <<EOF

Both daemons started in background. Logs: $LOGS_DIR
  tail -f $CSD_LOG
  tail -f $MLND_LOG

Stop with: ./scripts/maker-box-up.sh stop
EOF
