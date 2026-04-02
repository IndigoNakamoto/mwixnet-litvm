#!/usr/bin/env python3
"""Structural validation for MLN Nostr fixtures (see research/NOSTR_MLN.md).

Does not verify Schnorr signatures or keccak(nostr pubkey); clients use Nostr
libraries + LitVM RPC for that.
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from typing import Any

RE_ADDR = re.compile(r"^0x[a-f0-9]{40}$")
RE_B32 = re.compile(r"^0x[a-f0-9]{64}$")
RE_D = re.compile(r"^mln:v1:\d+:0x[a-f0-9]{40}$")


def _tags_map(tags: list[Any]) -> dict[str, str]:
    out: dict[str, str] = {}
    for t in tags:
        if len(t) >= 2 and isinstance(t[0], str):
            out[t[0]] = t[1]
    return out


def _require_litvm(obj: dict[str, Any]) -> None:
    lit = obj.get("litvm")
    if not isinstance(lit, dict):
        raise ValueError("content.litvm must be an object")
    cid = lit.get("chainId")
    reg = lit.get("registry")
    court = lit.get("grievanceCourt")
    if not isinstance(cid, str) or not cid.isdigit():
        raise ValueError("litvm.chainId must be a decimal string")
    if not isinstance(reg, str) or not RE_ADDR.match(reg):
        raise ValueError("litvm.registry must be 0x + 40 hex lowercase")
    if not isinstance(court, str) or not RE_ADDR.match(court):
        raise ValueError("litvm.grievanceCourt must be 0x + 40 hex lowercase")


def validate_maker_ad(event: dict[str, Any]) -> None:
    if event.get("kind") != 31250:
        raise ValueError("kind must be 31250")
    tags = event.get("tags")
    if not isinstance(tags, list):
        raise ValueError("tags must be a list")
    tm = _tags_map(tags)
    if tm.get("t") != "mln-maker-ad":
        raise ValueError("missing tag [t, mln-maker-ad]")
    d = tm.get("d")
    if not isinstance(d, str) or not RE_D.match(d):
        raise ValueError("missing or invalid d tag (expected mln:v1:<chainId>:0x<addr>)")
    raw = event.get("content")
    if not isinstance(raw, str):
        raise ValueError("content must be a JSON string")
    body = json.loads(raw)
    if body.get("v") != 1:
        raise ValueError("content JSON v must be 1")
    _require_litvm(body)
    parts = d.split(":")
    if len(parts) != 4:
        raise ValueError("d tag malformed")
    _, _, chain_from_d, addr_from_d = parts
    lit = body["litvm"]
    if lit.get("chainId") != chain_from_d:
        raise ValueError("litvm.chainId must match d tag chain segment")
    if lit.get("registry", "").lower() != lit.get("registry"):
        raise ValueError("use lowercase hex in litvm addresses")
    if addr_from_d.lower() != addr_from_d:
        raise ValueError("d tag maker address must be lowercase hex")


def validate_grievance_pointer(event: dict[str, Any]) -> None:
    if event.get("kind") != 31251:
        raise ValueError("kind must be 31251")
    tags = event.get("tags")
    if not isinstance(tags, list):
        raise ValueError("tags must be a list")
    tm = _tags_map(tags)
    if tm.get("t") != "mln-grievance":
        raise ValueError("missing tag [t, mln-grievance]")
    raw = event.get("content")
    if not isinstance(raw, str):
        raise ValueError("content must be a JSON string")
    body = json.loads(raw)
    if body.get("v") != 1:
        raise ValueError("content JSON v must be 1")
    _require_litvm(body)
    gid = body.get("grievanceId")
    if not isinstance(gid, str) or not RE_B32.match(gid):
        raise ValueError("grievanceId must be 0x + 64 hex lowercase")
    acc = body.get("accused")
    if acc is not None and (not isinstance(acc, str) or not RE_ADDR.match(acc)):
        raise ValueError("accused must be a valid 0x address if present")


def validate_event(event: dict[str, Any]) -> None:
    k = event.get("kind")
    if k == 31250:
        validate_maker_ad(event)
    elif k == 31251:
        validate_grievance_pointer(event)
    else:
        raise ValueError(f"unsupported kind {k!r}")


def main() -> int:
    root = Path(__file__).resolve().parent
    valid_dir = root / "fixtures" / "valid"
    if not valid_dir.is_dir():
        print("missing nostr/fixtures/valid/", file=sys.stderr)
        return 2
    paths = sorted(valid_dir.glob("*.json"))
    if not paths:
        print("no fixtures in nostr/fixtures/valid/", file=sys.stderr)
        return 2
    errors: list[str] = []
    for p in paths:
        try:
            data = json.loads(p.read_text(encoding="utf-8"))
            validate_event(data)
            print(f"ok {p.name}")
        except (json.JSONDecodeError, ValueError, KeyError) as e:
            errors.append(f"{p.name}: {e}")
    if errors:
        for line in errors:
            print(line, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
