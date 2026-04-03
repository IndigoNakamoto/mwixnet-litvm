#!/usr/bin/env python3
"""
Compute nostrKeyHash = keccak256(x-only secp256k1 pubkey) for MwixnetRegistry.registerMaker.
See research/NOSTR_MLN.md. Requires: pip install -r scripts/requirements.txt (nostr + pycryptodome).
Optional fallback: `cast keccak` on PATH or Docker Foundry image.

Example:
  python3 scripts/e2e_nostr_key_hash.py --secret-hex 1111...1111
"""

from __future__ import annotations

import argparse
import shutil
import subprocess
import sys


def keccak256_bytes(b: bytes) -> bytes:
    try:
        from Crypto.Hash import keccak as _keccak

        h = _keccak.new(digest_bits=256)
        h.update(b)
        return h.digest()
    except ImportError:
        pass
    cast = shutil.which("cast")
    if cast:
        out = subprocess.check_output([cast, "keccak", "0x" + b.hex()], text=True)
        for line in out.strip().splitlines():
            t = line.strip()
            if t.startswith("0x"):
                return bytes.fromhex(t[2:])
        raise RuntimeError("cast keccak: no 0x hash in output")
    image = "ghcr.io/foundry-rs/foundry:latest"
    out = subprocess.check_output(
        [
            "docker",
            "run",
            "--rm",
            "--entrypoint",
            "cast",
            image,
            "keccak",
            "0x" + b.hex(),
        ],
        text=True,
    )
    for line in out.strip().splitlines():
        t = line.strip()
        if t.startswith("0x"):
            return bytes.fromhex(t[2:])
    raise RuntimeError("docker cast keccak: no 0x hash in output")


def main() -> int:
    parser = argparse.ArgumentParser(description="Print x-only pubkey hex and nostrKeyHash for E2E makers.")
    parser.add_argument(
        "--secret-hex",
        required=True,
        help="32-byte Nostr secret as 64 hex chars (no 0x), same as MLND_NOSTR_NSEC hex form.",
    )
    args = parser.parse_args()

    try:
        from nostr.key import PrivateKey
    except ImportError:
        print("Error: install nostr (see scripts/requirements.txt).", file=sys.stderr)
        return 1

    h = args.secret_hex.strip().lower()
    if h.startswith("0x"):
        h = h[2:]
    if len(h) != 64:
        print("Error: --secret-hex must be 64 hex characters.", file=sys.stderr)
        return 1
    pk = PrivateKey(bytes.fromhex(h))
    xonly = pk.public_key.hex()
    if len(xonly) != 64:
        print("Error: unexpected pubkey length.", file=sys.stderr)
        return 1

    try:
        kh_bin = keccak256_bytes(bytes.fromhex(xonly))
        kh = "0x" + kh_bin.hex()
    except (subprocess.CalledProcessError, RuntimeError) as e:
        print("Error: keccak256 failed (%s). Install pycryptodome or use cast/Docker." % e, file=sys.stderr)
        return 1

    print("xonly_pubkey_0x", "0x" + xonly)
    print("nostrKeyHash   ", kh)
    print("nsec           ", pk.bech32())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
