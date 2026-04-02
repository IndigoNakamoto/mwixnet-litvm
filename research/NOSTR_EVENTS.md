# Nostr Event Kinds for MLN Stack (Phase 2)

Canonical reference for discovery and grievance pointers in the LitVM + MWEB + Nostr stack.
LitVM remains the single source of truth for all on-chain state (staking, grievances, slashing). Nostr is only a public bulletin board.

All events use NIP-34-style custom kinds in the 30000+ range to avoid collisions with other Nostr clients.

See `PRODUCT_SPEC.md` section 7 (Nostr discovery layer) and `research/EVIDENCE_GENERATOR.md` for how `grievanceId` / `evidenceHash` are derived.

---

## Event Kinds

| kind | purpose | required tags | content (JSON) | example use |
|------|---------|---------------|----------------|-------------|
| **30001** | Maker Advertisement | `["litvm-stake","0x..."]` `["fee","0.25"]` `["tor","onion..."]` `["epoch","<unix>"]` | Fee schedule + contact info | Wallets filter active makers |
| **31001** | Grievance Pointer (public accusation) | `["epoch","<epochId>"]` `["grievance","0x<grievanceId>"]` `["evidenceHash","0x<...>"]` `["accused","<nostr-pubkey>"]` | Optional signed defense notes | Takers broadcast failures |
| **31002** | Grievance Resolution Echo | `["grievance","0x<grievanceId>"]` `["status","slashed\|exonerated"]` | LitVM tx hash (optional) | Wallets show final outcome |
| **30000** | Epoch Announcement (optional) | `["epoch","<epochId>"]` `["start","<unix>"]` | Midnight batch metadata | Coordination only |

---

## Tagging & Privacy Rules

- `grievanceId` must be the exact 32-byte value from `EvidenceLib.grievanceId(...)` (see `EVIDENCE_GENERATOR.md` section 2).
- Never publish raw MWEB data (commitments, ciphertexts, etc.) - only the LitVM hashes.
- Makers sign advertisements with their Nostr private key (linked to on-chain stake via `MwixnetRegistry.registerMaker`).
- Wallets subscribe with filters like `{"kinds":[31001],"#grievance":["0x..."]}` for instant grievance visibility.

---

## Sample Subscription (Python)

```python
# Minimal Nostr listener for grievances
from nostr.filter import Filter
# ...
filter = Filter(kinds=[31001], tags={"grievance": ["0x5020b346..."]})
# relay.subscribe(filter)
```

This spec makes grievances publicly discoverable while keeping all economic enforcement on LitVM.
