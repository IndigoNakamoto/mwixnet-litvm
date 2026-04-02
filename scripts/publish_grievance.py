#!/usr/bin/env python3
import json
import sys
from nostr.event import Event
from nostr.key import PrivateKey


def _strip_0x(value: str) -> str:
    return value[2:] if value.startswith("0x") else value


def main() -> int:
    if len(sys.argv) != 5:
        print(
            "Usage: ./scripts/publish_grievance.py <grievanceId_hex> <epochId> <evidenceHash_hex> <nostr_privkey_hex>",
            file=sys.stderr,
        )
        return 1

    grievance_id = _strip_0x(sys.argv[1])
    epoch_id = sys.argv[2]
    evidence_hash = _strip_0x(sys.argv[3])
    privkey_hex = _strip_0x(sys.argv[4])

    privkey = PrivateKey(bytes.fromhex(privkey_hex))

    event = Event(
        kind=31001,
        content=json.dumps({"notes": "Grievance opened via LitVM test-grievance"}),
        tags=[
            ["epoch", epoch_id],
            ["grievance", "0x" + grievance_id],
            ["evidenceHash", "0x" + evidence_hash],
        ],
    )

    event.sign(privkey)
    print(json.dumps(event.to_dict(), indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
