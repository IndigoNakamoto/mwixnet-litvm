#!/usr/bin/env bash
# Start one permissioned mixnode (no taker scan/spend). Does not print secrets.
# Usage: scripts/mixnode-start.sh
# Env: PROOF_B_HOP_ENV (default deploy/mixnet.hop.env)
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${PROOF_B_HOP_ENV:-$ROOT/deploy/mixnet.hop.env}"
BIN="$ROOT/bin/coinswapd-research"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing $ENV_FILE (copy deploy/mixnet.hop.env.example)" >&2
  exit 1
fi

eval "$(python3 - "$ENV_FILE" << 'PY'
import shlex, sys
from pathlib import Path
wanted = ("MIX_K", "E2E_MWEB_DEST", "MLN_HOP_INDEX", "LISTEN_PORT", "MIXNET_DIRECTORY", "MIXNODE_DATADIR")
for line in Path(sys.argv[1]).read_text().splitlines():
    s = line.strip()
    if not s or s.startswith("#") or "=" not in s:
        continue
    k, _, v = s.partition("=")
    k, v = k.strip(), v.strip()
    if k not in wanted:
        continue
    if (v.startswith('"') and v.endswith('"')) or (v.startswith("'") and v.endswith("'")):
        v = v[1:-1]
    print(f"export {k}={shlex.quote(v)}")
PY
)"

IDX="${MLN_HOP_INDEX:?MLN_HOP_INDEX required (0-2)}"
PORT="${LISTEN_PORT:?LISTEN_PORT required}"
DIR_JSON="${MIXNET_DIRECTORY:-$ROOT/deploy/mixnet.directory.json}"
WORKDIR="${MIXNODE_DATADIR:-$ROOT/data/mixnode-hop$IDX}"

if [[ ! -x "$BIN" ]]; then
  echo "missing $BIN (make build-research-coinswapd)" >&2
  exit 1
fi
if [[ -z "${E2E_MWEB_DEST:-}" || -z "${MIX_K:-}" ]]; then
  echo "missing E2E_MWEB_DEST or MIX_K in $ENV_FILE" >&2
  exit 1
fi
if [[ ! -f "$DIR_JSON" ]]; then
  echo "missing directory $DIR_JSON" >&2
  exit 1
fi
case "$IDX" in
  0|1|2) ;;
  *) echo "MLN_HOP_INDEX must be 0, 1, or 2" >&2; exit 2 ;;
esac

mkdir -p "$WORKDIR"
cd "$WORKDIR"
export HTTP_PROXY="${HTTP_PROXY:-socks5h://127.0.0.1:9050}"
export HTTPS_PROXY="${HTTPS_PROXY:-$HTTP_PROXY}"
export NO_PROXY="${NO_PROXY:-127.0.0.1,localhost}"

echo "mixnode-start: hop-index=$IDX listen=$PORT directory=$DIR_JSON workdir=$WORKDIR" >&2
exec "$BIN" \
  -k "$MIX_K" \
  -l "$PORT" \
  -a "$E2E_MWEB_DEST" \
  -mln-directory "$DIR_JSON" \
  -mln-hop-index "$IDX"
