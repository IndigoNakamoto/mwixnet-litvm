"""Shared helpers for MLN Nostr wire profile (research/NOSTR_MLN.md)."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from typing import Any

KIND_MAKER_AD = 31250
KIND_GRIEVANCE_POINTER = 31251

TAG_T_MAKER_AD = "mln-maker-ad"
TAG_T_GRIEVANCE = "mln-grievance"

RE_ADDR = re.compile(r"^0x[a-f0-9]{40}$")
RE_B32 = re.compile(r"^0x[a-f0-9]{64}$")


def normalize_addr(addr: str) -> str:
    a = addr.strip().lower()
    if not a.startswith("0x"):
        a = "0x" + a
    if not RE_ADDR.match(a):
        raise ValueError(f"invalid 20-byte hex address: {addr!r}")
    return a


def normalize_bytes32(value: str) -> str:
    a = value.strip().lower()
    if not a.startswith("0x"):
        a = "0x" + a
    if not RE_B32.match(a):
        raise ValueError(f"invalid bytes32 hex: {value!r}")
    return a


def d_tag_maker_ad(chain_id: str, maker_evm: str) -> str:
    cid = str(chain_id).strip()
    if not cid.isdigit():
        raise ValueError("chain_id must be a decimal string")
    return f"mln:v1:{cid}:{normalize_addr(maker_evm)}"


def litvm_block(chain_id: str, registry: str, grievance_court: str) -> dict[str, str]:
    return {
        "chainId": str(chain_id).strip(),
        "registry": normalize_addr(registry),
        "grievanceCourt": normalize_addr(grievance_court),
    }


def maker_ad_content_json(
    chain_id: str,
    registry: str,
    grievance_court: str,
    *,
    fees: dict[str, Any] | None = None,
    tor: str | None = None,
    capabilities: list[str] | None = None,
) -> str:
    body: dict[str, Any] = {
        "v": 1,
        "litvm": litvm_block(chain_id, registry, grievance_court),
    }
    if fees is not None:
        body["fees"] = fees
    if tor is not None:
        body["tor"] = tor
    if capabilities is not None:
        body["capabilities"] = capabilities
    return json.dumps(body, separators=(",", ":"))


def grievance_pointer_content_json(
    chain_id: str,
    registry: str,
    grievance_court: str,
    grievance_id: str,
    *,
    epoch_id: str | None = None,
    accused: str | None = None,
    phase_hint: str | None = None,
) -> str:
    body: dict[str, Any] = {
        "v": 1,
        "litvm": litvm_block(chain_id, registry, grievance_court),
        "grievanceId": normalize_bytes32(grievance_id),
    }
    if epoch_id is not None:
        body["epochId"] = str(epoch_id)
    if accused:
        body["accused"] = normalize_addr(accused)
    if phase_hint is not None:
        body["phase_hint"] = phase_hint
    return json.dumps(body, separators=(",", ":"))


def load_registry_court_from_broadcast(path: str | Path) -> tuple[str, str]:
    """Parse Foundry broadcast JSON for MwixnetRegistry + GrievanceCourt CREATE addresses."""
    p = Path(path)
    with p.open() as f:
        d = json.load(f)
    reg, court = None, None
    for t in d.get("transactions", []):
        if t.get("transactionType") != "CREATE":
            continue
        cn = t.get("contractName")
        if cn == "MwixnetRegistry":
            reg = t["contractAddress"]
        elif cn == "GrievanceCourt":
            court = t["contractAddress"]
    if not reg or not court:
        print("missing MwixnetRegistry / GrievanceCourt CREATE in broadcast", file=sys.stderr)
        raise SystemExit(1)
    return reg, court
