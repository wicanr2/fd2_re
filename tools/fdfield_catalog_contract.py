#!/usr/bin/env python3
"""驗證完整 FDFIELD runtime 資源 catalog；不讀取原始 FDFIELD.DAT。"""

from __future__ import annotations

import hashlib
import json
import re
import struct
from pathlib import Path, PurePosixPath


CATALOG_PATH = "maps/fdfield_catalog.json"
MAP_ID = re.compile(r"^map([0-9]+)$")
HEX64 = re.compile(r"^[0-9a-f]{64}$")


def digest(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def safe_relative(value: object) -> bool:
    if not isinstance(value, str) or not value or "\\" in value:
        return False
    path = PurePosixPath(value)
    return not path.is_absolute() and ".." not in path.parts


def reconstruct_map(document: object) -> bytes:
    if not isinstance(document, dict):
        raise ValueError("map JSON 必須是物件")
    width, height = document.get("w"), document.get("h")
    if not isinstance(width, int) or not isinstance(height, int) or not (0 < width <= 0xffff) or not (0 < height <= 0xffff):
        raise ValueError("map JSON 寬高無效")
    cells = width * height
    tiles = document.get("tiles")
    events = document.get("native_composition_event_bytes")
    modes = document.get("native_tile_blit_modes")
    if not all(isinstance(values, list) and len(values) == cells for values in (tiles, events, modes)):
        raise ValueError("map JSON 組合格陣列長度不符")
    raw = bytearray(4 + 4 * cells)
    struct.pack_into("<HH", raw, 0, width, height)
    for index, (tile, event, mode) in enumerate(zip(tiles, events, modes)):
        if not isinstance(tile, int) or not 0 <= tile <= 0xffff:
            raise ValueError(f"map JSON tile[{index}] 超出 uint16")
        if not isinstance(event, int) or not 0 <= event <= 0xff:
            raise ValueError(f"map JSON event[{index}] 超出 byte")
        if not isinstance(mode, int) or not 0 <= mode <= 0xff:
            raise ValueError(f"map JSON mode[{index}] 超出 byte")
        struct.pack_into("<HBB", raw, 4 + 4 * index, tile, event, mode)
    return bytes(raw)


def validate_fdfield_assets(
    runtime_root: Path,
    expected_source: dict | None,
    expected_catalog_sha256: str | None = None,
) -> tuple[dict | None, dict[int, dict], list[str]]:
    errors: list[str] = []
    catalog_path = runtime_root / CATALOG_PATH
    try:
        raw_catalog = catalog_path.read_bytes()
        catalog = json.loads(raw_catalog.decode("utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
        return None, {}, [f"無法讀取 FDFIELD runtime catalog：{exc}"]
    catalog_sha256 = hashlib.sha256(raw_catalog).hexdigest()
    if expected_catalog_sha256 is not None and catalog_sha256 != expected_catalog_sha256:
        errors.append("FDFIELD runtime catalog SHA-256 不符")
    if not isinstance(expected_source, dict) or catalog.get("source") != expected_source:
        errors.append("FDFIELD runtime catalog 來源版本不符")
    if catalog.get("schema_version") != 1 or catalog.get("kind") != "fd2_fdfield_runtime_catalog":
        errors.append("FDFIELD runtime catalog identity 不符")
    resources = catalog.get("resources")
    if not isinstance(resources, list) or not resources:
        errors.append("FDFIELD runtime catalog resources 必須是非空陣列")
        resources = []
    result: dict[int, dict] = {}
    required = {
        "resource_index", "map_id", "path", "file_bytes", "file_sha256",
        "raw_bytes", "raw_sha256", "evidence_level", "evidence",
    }
    for index, entry in enumerate(resources):
        prefix = f"FDFIELD runtime catalog resources[{index}]"
        if not isinstance(entry, dict) or set(entry) != required:
            errors.append(f"{prefix} 欄位集合不符")
            continue
        resource = entry.get("resource_index")
        match = MAP_ID.fullmatch(entry.get("map_id", "")) if isinstance(entry.get("map_id"), str) else None
        relative = entry.get("path")
        if not isinstance(resource, int) or resource < 0 or resource in result:
            errors.append(f"{prefix} resource index 無效或重複")
            continue
        if match is None or not safe_relative(relative) or relative != f"maps/{entry['map_id']}/map.json":
            errors.append(f"{prefix} map identity／path 不符")
            continue
        if entry.get("evidence_level") != "confirmed" or not isinstance(entry.get("evidence"), str) or not entry["evidence"].startswith("docs/data/ida/"):
            errors.append(f"{prefix} 證據欄位不符")
        output = runtime_root / relative
        if not output.is_file():
            errors.append(f"{prefix} 缺少 runtime map JSON：{relative}")
            continue
        if not isinstance(entry.get("file_bytes"), int) or output.stat().st_size != entry["file_bytes"]:
            errors.append(f"{prefix} map JSON 大小不符")
        if not isinstance(entry.get("file_sha256"), str) or not HEX64.fullmatch(entry["file_sha256"]) or digest(output) != entry["file_sha256"]:
            errors.append(f"{prefix} map JSON SHA-256 不符")
        try:
            document = json.loads(output.read_text(encoding="utf-8"))
            reconstructed = reconstruct_map(document)
        except (OSError, UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
            errors.append(f"{prefix} 無法重建：{exc}")
            continue
        # 既有標準 map JSON 以受 catalog 約束的穩定路徑識別；舊資料沒有
        # 重複保存 map/map_id。若新文件選擇保存 map，則仍必須一致。
        if "map" in document and document.get("map") != int(match.group(1)):
            errors.append(f"{prefix} map JSON map id 不符")
        raw_sha256 = hashlib.sha256(reconstructed).hexdigest()
        if not isinstance(entry.get("raw_bytes"), int) or len(reconstructed) != entry["raw_bytes"]:
            errors.append(f"{prefix} 重建 raw 大小不符")
        if not isinstance(entry.get("raw_sha256"), str) or not HEX64.fullmatch(entry["raw_sha256"]) or raw_sha256 != entry["raw_sha256"]:
            errors.append(f"{prefix} 重建 raw SHA-256 不符")
        result[resource] = entry
    bridge = {
        "kind": "fd2_fdfield_runtime_catalog",
        "asset_root": "runtime_assets",
        "catalog_path": CATALOG_PATH,
        "catalog_bytes": len(raw_catalog),
        "catalog_sha256": catalog_sha256,
        "schema_version": 1,
        "source_file": "FDFIELD.DAT",
        "resources": len(result),
    }
    return bridge, result, errors
