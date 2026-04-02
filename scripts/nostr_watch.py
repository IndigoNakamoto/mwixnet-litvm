#!/usr/bin/env python3
"""
Watch MLN Nostr events (kinds 31250 / 31251) on a relay, or publish a signed event JSON.

Uses a direct WebSocket REQ so subscriptions work regardless of Relay._on_open behavior.
See research/E2E_NOSTR_DEMO.md and research/NOSTR_MLN.md.
"""

from __future__ import annotations

import argparse
import json
import socket
import sys
import time
from pathlib import Path

_SCRIPTS = Path(__file__).resolve().parent
if str(_SCRIPTS) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS))

from mln_nostr_wire import (  # noqa: E402
    KIND_GRIEVANCE_POINTER,
    KIND_MAKER_AD,
    TAG_T_GRIEVANCE,
    TAG_T_MAKER_AD,
)
from nostr.event import Event  # noqa: E402
from nostr.filter import Filter  # noqa: E402
from nostr.message_type import ClientMessageType, RelayMessageType  # noqa: E402

try:
    from websocket import WebSocketTimeoutException, create_connection
except ImportError:
    print("Install nostr (includes websocket-client): pip install -r scripts/requirements.txt", file=sys.stderr)
    raise SystemExit(1)


def _mln_filter_dicts() -> list[dict]:
    f_ad = Filter(kinds=[KIND_MAKER_AD])
    f_ad.add_arbitrary_tag("t", [TAG_T_MAKER_AD])
    f_g = Filter(kinds=[KIND_GRIEVANCE_POINTER])
    f_g.add_arbitrary_tag("t", [TAG_T_GRIEVANCE])
    return [f_ad.to_json_object(), f_g.to_json_object()]


def cmd_watch(relay: str, duration: int, sub_id: str) -> int:
    filters = _mln_filter_dicts()
    req = [ClientMessageType.REQUEST, sub_id, *filters]
    ws = create_connection(relay)
    try:
        ws.send(json.dumps(req))
        print(
            f"Subscribed on {relay} for kinds {KIND_MAKER_AD}/{KIND_GRIEVANCE_POINTER} "
            f"({sub_id}), {duration}s…",
            flush=True,
        )
        deadline = time.time() + duration
        while time.time() < deadline:
            remaining = deadline - time.time()
            if remaining <= 0:
                break
            ws.settimeout(min(2.0, remaining))
            try:
                raw = ws.recv()
            except (socket.timeout, WebSocketTimeoutException, OSError):
                continue
            if not raw:
                continue
            try:
                msg = json.loads(raw)
            except json.JSONDecodeError:
                continue
            if not isinstance(msg, list) or len(msg) < 2:
                continue
            if msg[0] == RelayMessageType.EVENT and len(msg) >= 3:
                ev = msg[2]
                if isinstance(ev, dict):
                    kind = ev.get("kind")
                    eid = ev.get("id", "")[:16]
                    content = ev.get("content", "")
                    snippet = (content[:120] + "…") if len(content) > 120 else content
                    print(f"[EVENT] kind={kind} id={eid}… content_snippet={snippet!r}", flush=True)
            elif msg[0] == RelayMessageType.NOTICE:
                print(f"[NOTICE] {msg[1]}", flush=True)
    finally:
        try:
            ws.send(json.dumps([ClientMessageType.CLOSE, sub_id]))
        except Exception:
            pass
        ws.close()
    return 0


def cmd_publish(relay: str, fp) -> int:
    data = json.load(fp)
    required = ("pubkey", "content", "created_at", "kind", "tags", "id", "sig")
    for k in required:
        if k not in data:
            print(f"Missing JSON field: {k}", file=sys.stderr)
            return 1
    ev = Event(
        data["pubkey"],
        data["content"],
        data["created_at"],
        data["kind"],
        data["tags"],
        data["id"],
        data["sig"],
    )
    if not ev.verify():
        print("Event signature verification failed.", file=sys.stderr)
        return 1
    ws = create_connection(relay)
    try:
        ws.send(ev.to_message())
        print(f"Published event id={ev.id[:16]}… to {relay}", flush=True)
        time.sleep(0.5)
    finally:
        ws.close()
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description="Watch or publish MLN Nostr events (31250/31251).")
    parser.add_argument("--relay", default="wss://relay.damus.io", help="WebSocket relay URL")
    parser.add_argument("--duration", type=int, default=30, help="Seconds to watch (watch mode)")
    parser.add_argument("--sub-id", default="mln-watch", help="Subscription id for REQ")
    parser.add_argument(
        "--publish-json",
        metavar="PATH",
        default="",
        help="Publish signed Nostr event JSON from this file, or '-' for stdin",
    )
    args = parser.parse_args()

    if args.publish_json:
        if args.publish_json.strip() == "-":
            return cmd_publish(args.relay, sys.stdin)
        path = Path(args.publish_json)
        with path.open() as f:
            return cmd_publish(args.relay, f)

    return cmd_watch(args.relay, args.duration, args.sub_id)


if __name__ == "__main__":
    raise SystemExit(main())
