#!/usr/bin/env python3
import argparse
from typing import Any, Iterable


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Listen for kind 30001 maker advertisements (stub filter output)."
    )
    parser.add_argument(
        "--stake",
        default="",
        help="Optional litvm-stake tag value to highlight (0x... address).",
    )
    parser.add_argument(
        "--relay",
        default="",
        help="Optional relay URL for operator notes (network subscribe wiring is external).",
    )
    return parser.parse_args()


def _format_filters(stake_value: str) -> Iterable[Any]:
    from nostr.filter import Filter

    # Base maker-ad filter per research/NOSTR_EVENTS.md (kind 30001).
    yield Filter(kinds=[30001])
    if stake_value:
        yield Filter(kinds=[30001], tags={"litvm-stake": [stake_value]})


def main() -> int:
    args = _parse_args()
    stake_value = args.stake.strip()
    relay = args.relay.strip()
    filters = list(_format_filters(stake_value))

    print("Listening for kind 30001 maker advertisements...")
    if relay:
        print(f"Relay hint: {relay}")
    print("Generated subscription filters:")
    for idx, event_filter in enumerate(filters, start=1):
        print(f"  [{idx}] {event_filter.to_json_object()}")

    if stake_value:
        print(f"\nStake focus: litvm-stake={stake_value}")
    else:
        print("\nStake focus: none (show all maker ads)")

    print(
        "\nNote: this script prints subscription filters and expected fields. "
        "Wire to your relay client and print tags: litvm-stake, fee, tor, epoch."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
