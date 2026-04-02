#!/usr/bin/env python3
from nostr.filter import Filter


def main() -> int:
    # Minimal listener stub (expand with real relay client wiring as needed).
    grievance_filter = Filter(kinds=[31001])
    _ = grievance_filter
    print("Listening for kind 31001 grievances...")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
