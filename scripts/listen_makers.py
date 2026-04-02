#!/usr/bin/env python3
"""Print subscription filters for MLN maker ads (kind 31250, research/NOSTR_MLN.md)."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

_SCRIPTS = Path(__file__).resolve().parent
if str(_SCRIPTS) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS))

from mln_nostr_wire import KIND_MAKER_AD, TAG_T_MAKER_AD, d_tag_maker_ad


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Build nostr filters for mln_maker_ad (kind 31250)."
    )
    parser.add_argument(
        "--chain-id",
        default="",
        help='LitVM chain id for d-tag filter (decimal string, e.g. "31337").',
    )
    parser.add_argument(
        "--maker",
        default="",
        help="Maker LitVM address (0x + 40 hex); with --chain-id adds exact d-tag filter.",
    )
    parser.add_argument(
        "--relay",
        default="",
        help="Optional relay URL for operator notes (subscription wiring is external).",
    )
    return parser.parse_args()


def main() -> int:
    from nostr.filter import Filter

    args = _parse_args()
    relay = args.relay.strip()

    if args.chain_id or args.maker:
        if not (args.chain_id and args.maker):
            print(
                "Provide both --chain-id and --maker for a d-tag filter, or neither for kind-only.",
                file=sys.stderr,
            )
            return 1

    flt = Filter(kinds=[KIND_MAKER_AD])
    flt.add_arbitrary_tag("t", [TAG_T_MAKER_AD])
    if args.chain_id and args.maker:
        flt.add_arbitrary_tag("d", [d_tag_maker_ad(args.chain_id, args.maker)])

    print(f"Listening for kind {KIND_MAKER_AD} maker advertisements ({TAG_T_MAKER_AD})...")
    if relay:
        print(f"Relay hint: {relay}")
    print("Generated subscription filter:")
    print(f"  {flt.to_json_object()}")
    print(
        "\nNote: content.litvm.registry is not relay-indexed; filter client-side if you scope by deployment."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
