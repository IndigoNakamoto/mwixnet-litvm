#!/usr/bin/env python3
"""Build a signed kind-31251 mln_grievance_pointer event (research/NOSTR_MLN.md).

Evidence preimages and hashes MUST NOT be published on Nostr; verify grievances on LitVM.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

_SCRIPTS = Path(__file__).resolve().parent
if str(_SCRIPTS) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS))

from nostr.event import Event
from nostr.key import PrivateKey

from mln_nostr_wire import (
    KIND_GRIEVANCE_POINTER,
    TAG_T_GRIEVANCE,
    grievance_pointer_content_json,
    load_registry_court_from_broadcast,
)


def _strip_0x(value: str) -> str:
    return value[2:] if value.startswith("0x") else value


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Emit JSON for a signed MLN grievance pointer (kind 31251)."
    )
    parser.add_argument("grievance_id", help="bytes32 grievance id, with or without 0x")
    parser.add_argument("epoch_id", help="decimal epoch id string for content.epochId")
    parser.add_argument("nostr_privkey_hex", help="Nostr secp256k1 private key hex")
    parser.add_argument(
        "--chain-id",
        default="31337",
        help="LitVM chain id (decimal string, must match d-tag / deployment)",
    )
    parser.add_argument(
        "--registry",
        default="",
        help="MwixnetRegistry address (or set MLN_LITVM_REGISTRY)",
    )
    parser.add_argument(
        "--grievance-court",
        default="",
        help="GrievanceCourt address (or set MLN_LITVM_GRIEVANCE_COURT)",
    )
    parser.add_argument(
        "--broadcast-json",
        default="",
        help="Foundry run-latest.json path (reads registry + court from CREATE txs)",
    )
    parser.add_argument(
        "--accused",
        default="",
        help="Optional accused LitVM address (0x + 40 hex)",
    )
    parser.add_argument(
        "--phase-hint",
        default="Open",
        help='Informational phase label (default "Open")',
    )
    args = parser.parse_args()

    import os

    registry = args.registry.strip() or os.environ.get("MLN_LITVM_REGISTRY", "")
    court = args.grievance_court.strip() or os.environ.get("MLN_LITVM_GRIEVANCE_COURT", "")

    if args.broadcast_json:
        reg2, court2 = load_registry_court_from_broadcast(Path(args.broadcast_json))
        registry = registry or reg2
        court = court or court2

    if not registry or not court:
        print(
            "Provide --registry and --grievance-court, or --broadcast-json, "
            "or MLN_LITVM_REGISTRY / MLN_LITVM_GRIEVANCE_COURT.",
            file=sys.stderr,
        )
        return 1

    privkey = PrivateKey(bytes.fromhex(_strip_0x(args.nostr_privkey_hex)))

    content = grievance_pointer_content_json(
        args.chain_id,
        registry,
        court,
        args.grievance_id,
        epoch_id=args.epoch_id,
        accused=args.accused or None,
        phase_hint=args.phase_hint,
    )

    event = Event(
        public_key=privkey.public_key.hex(),
        content=content,
        kind=KIND_GRIEVANCE_POINTER,
        tags=[["t", TAG_T_GRIEVANCE]],
    )
    privkey.sign_event(event)
    out = {
        "id": event.id,
        "pubkey": event.public_key,
        "created_at": event.created_at,
        "kind": event.kind,
        "tags": event.tags,
        "content": event.content,
        "sig": event.signature,
    }
    print(json.dumps(out, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
