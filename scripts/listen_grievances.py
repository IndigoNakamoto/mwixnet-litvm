#!/usr/bin/env python3
"""Print subscription filter for MLN grievance pointers (kind 31251, research/NOSTR_MLN.md)."""

from __future__ import annotations

import sys
from pathlib import Path

_SCRIPTS = Path(__file__).resolve().parent
if str(_SCRIPTS) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS))

from mln_nostr_wire import KIND_GRIEVANCE_POINTER, TAG_T_GRIEVANCE


def main() -> int:
    from nostr.filter import Filter

    grievance_filter = Filter(kinds=[KIND_GRIEVANCE_POINTER])
    grievance_filter.add_arbitrary_tag("t", [TAG_T_GRIEVANCE])
    print(f"Listening for kind {KIND_GRIEVANCE_POINTER} grievance pointers ({TAG_T_GRIEVANCE})...")
    print(f"Filter: {grievance_filter.to_json_object()}")
    print("\nVerify grievanceId and phase on LitVM; Nostr is gossip only.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
