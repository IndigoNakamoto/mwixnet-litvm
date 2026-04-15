#!/usr/bin/env python3
"""Ensure scripts/mln_nostr_wire.py output satisfies nostr/validate_fixtures.py rules."""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def _load_validate_fixtures():
    path = ROOT / "nostr" / "validate_fixtures.py"
    spec = importlib.util.spec_from_file_location("mln_validate_fixtures", path)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


def main() -> int:
    sys.path.insert(0, str(ROOT / "scripts"))
    import mln_nostr_wire as w

    vf = _load_validate_fixtures()

    reg = "0x5fbdb2315678afecb367f032d93f642f64180aa3"
    court = "0xe7f1725e7734ce288f8367e1bb143e90bb3f0512"
    maker = "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
    chain = "31337"

    maker_content = w.maker_ad_content_json(
        chain,
        reg,
        court,
        fees={"unit": "sat_per_hop", "min": 1, "max": 99},
        swap_x25519_pub_hex="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        capabilities=["mweb-coinswap-v0"],
    )
    maker_event = {
        "kind": w.KIND_MAKER_AD,
        "tags": [
            ["d", w.d_tag_maker_ad(chain, maker)],
            ["t", w.TAG_T_MAKER_AD],
            ["client", "mln-wire-check", "0"],
        ],
        "content": maker_content,
    }
    vf.validate_event(maker_event)

    v2_content = w.maker_ad_v2_sealed_content_json(
        chain,
        reg,
        court,
        ciphertext="nip44v2-placeholder-not-valid-ciphertext",
        capabilities=["mweb-coinswap-v0"],
    )
    maker2 = "0x70997970c51812dc3a010c7d01b50e0d17dc79c8"
    v2_event = {
        "kind": w.KIND_MAKER_AD,
        "tags": [
            ["d", w.d_tag_maker_ad(chain, maker2)],
            ["t", w.TAG_T_MAKER_AD],
        ],
        "content": v2_content,
    }
    vf.validate_event(v2_event)

    gid = "0x" + "aa" * 32
    g_content = w.grievance_pointer_content_json(
        chain,
        reg,
        court,
        gid,
        epoch_id="42",
        accused=maker,
        phase_hint="Open",
    )
    g_event = {
        "kind": w.KIND_GRIEVANCE_POINTER,
        "tags": [["t", w.TAG_T_GRIEVANCE]],
        "content": g_content,
    }
    vf.validate_event(g_event)

    print("ok mln_nostr_wire matches validate_fixtures schema")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
