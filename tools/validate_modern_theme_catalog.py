#!/usr/bin/env python3
"""Validate the tracked modern-theme catalog and optional private originals."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import struct
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CATALOG = ROOT / "remake/assets/themes/modern/catalog.json"
ID_RE = re.compile(r"^[a-z0-9]+(?:[._-][a-z0-9]+)+$")
SHA_RE = re.compile(r"^[0-9a-f]{64}$")
SOURCE_ASSET_RE = re.compile(r"^asset:[A-Za-z0-9][A-Za-z0-9._-]+$")
PROTOTYPE_ROLES = {"portrait_concept", "battlefield_concept", "battle_hud_concept"}
ROLES = PROTOTYPE_ROLES | {"story_portrait_frame", "map_sprite_set"}
STATUSES = {"prototype", "concept", "approved", "runtime_candidate", "runtime_ready"}


def png_size(path: Path) -> tuple[int, int]:
    raw = path.read_bytes()[:24]
    if len(raw) != 24 or raw[:8] != b"\x89PNG\r\n\x1a\n" or raw[12:16] != b"IHDR":
        raise ValueError(f"{path}: not a PNG")
    return struct.unpack(">II", raw[16:24])


def validate(verify_private: bool) -> dict:
    data = json.loads(CATALOG.read_text(encoding="utf-8"))
    if set(data) != {"schema_version", "theme_id", "status", "private_root", "assets"}:
        raise ValueError("catalog fields mismatch")
    if data["schema_version"] != 1 or data["status"] != "prototype":
        raise ValueError("unsupported catalog version or status")
    if not ID_RE.fullmatch(data["theme_id"]):
        raise ValueError("invalid theme_id")
    private_root = Path(data["private_root"])
    if private_root.is_absolute() or ".." in private_root.parts:
        raise ValueError("private_root must be a safe repository-relative path")
    assets = data["assets"]
    if not isinstance(assets, list) or not assets:
        raise ValueError("assets must be a non-empty list")
    ids: set[str] = set()
    roles: set[str] = set()
    for entry in assets:
        required = {"asset_id", "role", "status", "file", "width", "height", "sha256", "source_refs"}
        candidate = {"master_file", "consumer_contract", "speaker_id", "frame", "mouth_state"}
        sprite_set = {
            "asset_id", "role", "status", "files", "width", "height", "frame_count",
            "frame_sha256", "source_group", "consumer_contract", "alpha_contract",
            "cycle_policy", "source_refs",
        }
        if not isinstance(entry, dict):
            raise ValueError("asset entry must be an object")
        if entry.get("role") == "story_portrait_frame":
            expected = required | candidate
        elif entry.get("role") == "map_sprite_set":
            expected = sprite_set
        else:
            expected = required
        if set(entry) != expected:
            raise ValueError("asset fields mismatch")
        asset_id = entry["asset_id"]
        if not isinstance(asset_id, str) or not ID_RE.fullmatch(asset_id) or asset_id in ids:
            raise ValueError(f"invalid or duplicate asset_id: {asset_id!r}")
        ids.add(asset_id)
        if entry["role"] not in ROLES or (entry["role"] in PROTOTYPE_ROLES and entry["role"] in roles):
            raise ValueError(f"invalid or duplicate role: {entry['role']!r}")
        roles.add(entry["role"])
        if entry["status"] not in STATUSES:
            raise ValueError(f"invalid status: {entry['status']!r}")
        if entry["status"] == "prototype" and entry["role"] != "map_sprite_set":
            raise ValueError(f"prototype status is restricted to sprite sets: {asset_id}")
        if entry["role"] == "story_portrait_frame":
            master = Path(entry["master_file"])
            if master.is_absolute() or len(master.parts) != 1 or master.suffix.lower() != ".png":
                raise ValueError(f"unsafe private master file: {entry['master_file']!r}")
            if entry["consumer_contract"] != "native_story_dialogue_rgba_overlay_v1":
                raise ValueError(f"invalid consumer contract: {asset_id}")
            if not isinstance(entry["speaker_id"], int) or entry["speaker_id"] < 0:
                raise ValueError(f"invalid speaker identity: {asset_id}")
            if entry["frame"] not in range(4) or entry["mouth_state"] not in {"closed", "open"}:
                raise ValueError(f"invalid portrait frame identity: {asset_id}")
        if entry["role"] == "map_sprite_set":
            if entry["frame_count"] != 12 or entry["source_group"] not in range(96):
                raise ValueError(f"invalid map sprite identity: {asset_id}")
            if entry["consumer_contract"] != "fdicon_map_sprite_12x24_v1":
                raise ValueError(f"invalid map sprite contract: {asset_id}")
            if entry["alpha_contract"] != "binary":
                raise ValueError(f"invalid alpha contract: {asset_id}")
            if entry["cycle_policy"] not in {"three_distinct_cycles", "cycle_2_reuses_cycle_0_prototype"}:
                raise ValueError(f"invalid cycle policy: {asset_id}")
            files = entry["files"]
            hashes = entry["frame_sha256"]
            if not isinstance(files, list) or len(files) != 12 or len(set(files)) != 12:
                raise ValueError(f"invalid map sprite file set: {asset_id}")
            if not isinstance(hashes, list) or len(hashes) != 12 or not all(
                isinstance(value, str) and SHA_RE.fullmatch(value) for value in hashes
            ):
                raise ValueError(f"invalid map sprite hashes: {asset_id}")
            if entry["cycle_policy"] == "three_distinct_cycles" and len(set(hashes)) != 12:
                raise ValueError(f"map sprite cycles are not distinct: {asset_id}")
            file_paths = [Path(value) for value in files]
        else:
            file_paths = [Path(entry["file"])]
        for file_path in file_paths:
            if file_path.is_absolute() or len(file_path.parts) != 1 or file_path.suffix.lower() != ".png":
                raise ValueError(f"unsafe private file: {str(file_path)!r}")
        if not isinstance(entry["width"], int) or entry["width"] <= 0:
            raise ValueError(f"invalid width: {asset_id}")
        if not isinstance(entry["height"], int) or entry["height"] <= 0:
            raise ValueError(f"invalid height: {asset_id}")
        if entry["role"] != "map_sprite_set" and (
            not isinstance(entry["sha256"], str) or not SHA_RE.fullmatch(entry["sha256"])
        ):
            raise ValueError(f"invalid sha256: {asset_id}")
        refs = entry["source_refs"]
        if not isinstance(refs, list) or not refs:
            raise ValueError(f"missing source_refs: {asset_id}")
        for ref in refs:
            if not isinstance(ref, str):
                raise ValueError(f"invalid source_ref for {asset_id}: {ref!r}")
            if SOURCE_ASSET_RE.fullmatch(ref):
                continue
            ref_path = Path(ref)
            if ref_path.is_absolute() or ".." in ref_path.parts or not (ROOT / ref_path).is_file():
                raise ValueError(f"invalid source_ref for {asset_id}: {ref!r}")
        if verify_private:
            expected_hashes = entry["frame_sha256"] if entry["role"] == "map_sprite_set" else [entry["sha256"]]
            for file_path, expected_hash in zip(file_paths, expected_hashes):
                private_path = ROOT / private_root / file_path
                if not private_path.is_file():
                    raise ValueError(f"private asset missing: {private_path}")
                if png_size(private_path) != (entry["width"], entry["height"]):
                    raise ValueError(f"dimension mismatch: {asset_id}/{file_path.name}")
                digest = hashlib.sha256(private_path.read_bytes()).hexdigest()
                if digest != expected_hash:
                    raise ValueError(f"sha256 mismatch: {asset_id}/{file_path.name}")
                if entry["role"] == "map_sprite_set":
                    from PIL import Image
                    alpha = Image.open(private_path).convert("RGBA").getchannel("A")
                    if not set(alpha.getdata()).issubset({0, 255}):
                        raise ValueError(f"non-binary alpha: {asset_id}/{file_path.name}")
    if not PROTOTYPE_ROLES.issubset(roles):
        raise ValueError(f"required prototype roles missing: {sorted(PROTOTYPE_ROLES - roles)}")
    return data


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--verify-private", action="store_true")
    args = parser.parse_args()
    catalog = validate(args.verify_private)
    print(f"validated {len(catalog['assets'])} modern-theme asset(s)")
