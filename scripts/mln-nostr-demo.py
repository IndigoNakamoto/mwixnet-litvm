#!/usr/bin/env python3
"""
MLN Stack Nostr demo — subscription filters for maker ads (31250) + grievance pointers (31251).
See research/NOSTR_MLN.md.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

_SCRIPTS = Path(__file__).resolve().parent
if str(_SCRIPTS) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS))

from mln_nostr_wire import (
    KIND_GRIEVANCE_POINTER,
    KIND_MAKER_AD,
    TAG_T_GRIEVANCE,
    TAG_T_MAKER_AD,
    d_tag_maker_ad,
)


def main() -> int:
    parser = argparse.ArgumentParser(description="MLN Stack Nostr live demo (filter builder)")
    parser.add_argument("--relay", default="wss://relay.damus.io", help="Nostr relay URL")
    parser.add_argument(
        "--chain-id",
        default="",
        help='With --maker, build d-tag filter mln:v1:<chain>:<maker> (e.g. "31337").',
    )
    parser.add_argument(
        "--maker",
        default="",
        help="Maker LitVM address for optional d-tag filter (requires --chain-id).",
    )
    parser.add_argument("--grievance-only", action="store_true", help="Only grievance filters")
    args = parser.parse_args()

    try:
        from nostr.filter import Filter
        from nostr.relay_manager import RelayManager
    except ImportError:
        print("Install nostr dependency first: pip install nostr", file=sys.stderr)
        return 1

    print(f"Connecting to {args.relay} ...")
    print("Listening for:")
    print(f"   - Maker Ads      (kind {KIND_MAKER_AD}, tag t={TAG_T_MAKER_AD})")
    print(f"   - Grievances     (kind {KIND_GRIEVANCE_POINTER}, tag t={TAG_T_GRIEVANCE})")

    filters = []
    if not args.grievance_only:
        if args.chain_id and args.maker:
            d_val = d_tag_maker_ad(args.chain_id, args.maker)
            mf = Filter(kinds=[KIND_MAKER_AD])
            mf.add_arbitrary_tag("t", [TAG_T_MAKER_AD])
            mf.add_arbitrary_tag("d", [d_val])
            filters.append(mf)
            print(f"   Narrowing maker ads to d={d_val}")
        elif args.chain_id or args.maker:
            print(
                "Error: use both --chain-id and --maker for a d-tag filter, or neither for all maker ads.",
                file=sys.stderr,
            )
            return 1
        else:
            mf = Filter(kinds=[KIND_MAKER_AD])
            mf.add_arbitrary_tag("t", [TAG_T_MAKER_AD])
            filters.append(mf)

    gf = Filter(kinds=[KIND_GRIEVANCE_POINTER])
    gf.add_arbitrary_tag("t", [TAG_T_GRIEVANCE])
    filters.append(gf)

    print("\nConstructed filters:")
    for idx, event_filter in enumerate(filters, start=1):
        print(f"  [{idx}] {event_filter.to_json_object()}")

    print("\nAttempting relay connection...")
    relay_manager = RelayManager()
    relay_manager.add_relay(args.relay)
    relay_manager.open_connections()
    relay_manager.close_connections()
    print("Connected. Subscribe with the filters above in your chosen client workflow.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
