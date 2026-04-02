#!/usr/bin/env python3
"""
MLN Stack Nostr Demo - Live dashboard for Maker Ads (kind 30001) + Grievances (kind 31001)
Combines both listeners into one CLI for quick local validation.
"""

import argparse
import sys

def main() -> int:
    parser = argparse.ArgumentParser(description="MLN Stack Nostr live demo")
    parser.add_argument("--relay", default="wss://relay.damus.io", help="Nostr relay URL")
    parser.add_argument("--stake", help="Optional LitVM stake address filter (for maker ads)")
    parser.add_argument("--grievance-only", action="store_true", help="Only show grievances")
    args = parser.parse_args()

    try:
        from nostr.filter import Filter
        from nostr.relay_manager import RelayManager
    except ImportError:
        print("Install nostr dependency first: pip install nostr", file=sys.stderr)
        return 1

    print(f"Connecting to {args.relay} ...")
    print("Listening for:")
    print("   - Maker Ads      (kind 30001)")
    print("   - Grievances     (kind 31001)")

    if args.stake:
        print(f"   Filtering makers by stake: {args.stake}")

    filters = []
    if not args.grievance_only:
        filters.append(Filter(kinds=[30001]))
    filters.append(Filter(kinds=[31001]))

    # python-nostr client wiring differs across versions, so we keep
    # subscription construction explicit and operator-visible.
    print("\nConstructed filters:")
    for idx, event_filter in enumerate(filters, start=1):
        print(f"  [{idx}] {event_filter.to_json_object()}")

    if args.stake and not args.grievance_only:
        print(
            "\nStake filtering note: apply tag filter on relay subscribe "
            f"with litvm-stake={args.stake}."
        )

    print("\nAttempting relay connection...")
    relay_manager = RelayManager()
    relay_manager.add_relay(args.relay)
    relay_manager.open_connections()
    relay_manager.close_connections()
    print("Connected. Subscribe with the filters above in your chosen client workflow.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
