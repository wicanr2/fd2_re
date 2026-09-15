#!/usr/bin/env python3
"""Verify the machine-readable early/mid/late town E2 receipt contract."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
from pathlib import Path


HEX64 = re.compile(r"^[0-9a-f]{64}$")
EXPECTED = {
    "early": ("town_ch06", 0, 37),
    "mid": ("town_ch13", 3, 37),
    "late": ("town_ch27", 1, 1),
}


def fail(message: str) -> None:
    raise ValueError(message)


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify(receipt: dict) -> None:
    if receipt.get("schema_version") != 1:
        fail("schema_version must be 1")
    if receipt.get("kind") != "fd2_town_shop_early_mid_late_e2_receipt":
        fail("unexpected receipt kind")
    if receipt.get("status") not in {"candidate", "passed"}:
        fail("status must be candidate or passed")

    original = receipt.get("original") or {}
    if not HEX64.fullmatch(original.get("original_fd2_exe_sha256", "")):
        fail("original FD2.EXE hash is missing")
    if original.get("lock_ally_hp") or original.get("force_enemy_clear_declared"):
        fail("modified oracle controls are enabled")
    if original.get("state_injections_enabled") != []:
        fail("state injections must be empty")
    if original.get("successful_save_write_bytes") != [22528, 459]:
        fail("DOS save write boundary changed")
    if not re.fullmatch(r"[0-9a-f]{40}", original.get("dosgolem_commit", "")):
        fail("dosgolem commit is not a full hash")

    fixtures = receipt.get("fixtures")
    if not isinstance(fixtures, list) or {f.get("name") for f in fixtures} != set(EXPECTED):
        fail("receipt must contain exactly early/mid/late fixtures")
    for fixture in fixtures:
        name = fixture["name"]
        node, slot, delta = EXPECTED[name]
        if fixture.get("town_node") != node or fixture.get("slot") != slot:
            fail(f"{name}: node or slot mismatch")
        source = fixture.get("source") or {}
        for field in ("archive_sha256", "save_sha256"):
            if not HEX64.fullmatch(source.get(field, "")):
                fail(f"{name}: missing source {field}")
        if not source.get("provenance"):
            fail(f"{name}: source provenance is empty")
        transaction = fixture.get("transaction") or {}
        if transaction.get("gold_delta") != delta:
            fail(f"{name}: unexpected gold delta")
        if transaction.get("inventory_count_delta") != -1:
            fail(f"{name}: inventory was not reduced")
        save = fixture.get("save_boundary") or {}
        if save.get("changed") is not True or not HEX64.fullmatch(save.get("output_sha256", "")):
            fail(f"{name}: save boundary is incomplete")
        facilities = set(fixture.get("facilities", []))
        required = {
            f"preparation_{name if name != 'early' else 'ch06'}",
            f"shop_{name if name != 'early' else 'ch06'}_item",
            f"church_{name if name != 'early' else 'ch06'}",
            "hotel_save",
        }
        # The receipt stores chapter-specific node names; derive the suffix from town_node.
        suffix = node.removeprefix("town_")
        required = {f"preparation_{suffix}", f"shop_{suffix}_item", f"church_{suffix}", "hotel_save", f"shop_{suffix}_secret"}
        if not required <= facilities:
            fail(f"{name}: facility coverage is incomplete")
        images = fixture.get("original_images") or {}
        for label in ("town", "secret"):
            image = images.get(label) or {}
            if not image.get("checkpoint"):
                fail(f"{name}: {label} checkpoint is missing")
            for field in ("png_sha256", "indexed_sha256"):
                if not HEX64.fullmatch(image.get(field, "")):
                    fail(f"{name}: {label} {field} is missing")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("receipt", type=Path)
    args = parser.parse_args()
    verify(json.loads(args.receipt.read_text(encoding="utf-8")))
    print(f"verified {args.receipt}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
