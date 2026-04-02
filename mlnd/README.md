# mlnd — MLN operator daemon (LitVM + Nostr)

`mlnd` watches `GrievanceOpened` logs for your operator address and (optionally) republishes **kind 31250** maker ads to Nostr relays.

## LitVM watcher (required)

| Env | Meaning |
|-----|---------|
| `MLND_WS_URL` | WebSocket JSON-RPC URL (default `ws://127.0.0.1:8545`) |
| `MLND_COURT_ADDR` | `GrievanceCourt` contract (hex) |
| `MLND_OPERATOR_ADDR` | Your maker / accused address (hex) |
| `MLND_DB_PATH` | SQLite path for evidence receipts (default `mlnd.db`) |

## Nostr broadcaster (optional)

Set **`MLND_NOSTR_RELAYS`** (comma-separated `wss://…` URLs) to enable. Also required:

| Env | Meaning |
|-----|---------|
| `MLND_NOSTR_NSEC` | Nostr secret: **nsec1…** bech32 or **64-char** hex (no `0x`) |
| `MLND_LITVM_CHAIN_ID` | Decimal chain id string (e.g. `31337`) |
| `MLND_REGISTRY_ADDR` | `MwixnetRegistry` (hex) |
| `MLND_COURT_ADDR` | Same as watcher |
| `MLND_OPERATOR_ADDR` | Same as watcher; used in NIP-33 `d` tag and must match on-chain maker registration |

Optional:

| Env | Meaning |
|-----|---------|
| `MLND_NOSTR_INTERVAL` | Republish interval (default `30m`; `time.ParseDuration` syntax) |
| `MLND_TOR_ONION` | Tor mix API URL for `content.tor` |
| `MLND_FEE_MIN_SAT` / `MLND_FEE_MAX_SAT` | If both set, adds `fees` object (`sat_per_hop`) |

Wire format: [`research/NOSTR_MLN.md`](../research/NOSTR_MLN.md). Relay smoke flow: [`research/E2E_NOSTR_DEMO.md`](../research/E2E_NOSTR_DEMO.md).

**Dependency note:** imports use module path `github.com/nbd-wtf/go-nostr` with a `replace` to **`github.com/fiatjaf/go-nostr`** (maintained fork). Version is pinned to **v0.35.0** for Go **1.22** CI compatibility.

## Build / test

```bash
cd mlnd
go test ./... -count=1
```

On-chain `defendGrievance` submission is not implemented yet (see `internal/litvm/watcher.go` TODO).
