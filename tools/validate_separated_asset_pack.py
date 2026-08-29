#!/usr/bin/env python3
"""驗證 FD2 分離素材包清冊、來源版本、輸出 hash 與交叉引用。

本工具只讀分離素材包。原始 `.DAT` 只在指定 ``--original-dir`` 時用來核對
provenance；遊戲 runtime 不應呼叫本工具或取得 archive 路徑。
"""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import sys

from music_catalog_contract import validate_music_assets
from generate_separated_asset_manifest import build_coverage_summary

HEX = re.compile(r"^[0-9a-f]+$")
KINDS = {
    "map", "tileset", "map_sprite", "portrait", "battle_animation",
    "cutscene_animation", "background", "ui", "text", "sfx", "music",
    "font", "metadata",
}
STATUSES = {"exported", "intentionally_raw", "blocked"}
EVIDENCE = {"confirmed", "strong_inference", "hypothesis", "unknown"}
DISPOSITIONS = {"standardized", "confirmed_empty", "blocked", "unknown"}


def digest(path: Path, algorithm: str) -> str:
    h = hashlib.new(algorithm)
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def safe_relative(value: object) -> bool:
    if not isinstance(value, str) or not value or "\\" in value:
        return False
    path = PurePosixPath(value)
    return not path.is_absolute() and ".." not in path.parts


def validate(
    manifest_path: Path,
    original_dir: Path | None = None,
    runtime_assets: Path | None = None,
    coverage_summary: Path | None = None,
) -> list[str]:
    errors: list[str] = []
    try:
        doc = json.loads(manifest_path.read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        return [f"無法讀取 manifest：{exc}"]

    if doc.get("schema_version") != 2:
        errors.append("schema_version 必須為 2")
    if not isinstance(doc.get("pack_id"), str) or not doc["pack_id"]:
        errors.append("pack_id 不可為空")

    sources = doc.get("source_set")
    if not isinstance(sources, list) or not sources:
        errors.append("source_set 必須是非空陣列")
        sources = []
    source_names: set[str] = set()
    for index, source in enumerate(sources):
        prefix = f"source_set[{index}]"
        if not isinstance(source, dict):
            errors.append(f"{prefix} 必須是物件")
            continue
        name = source.get("file")
        if not isinstance(name, str) or not name or "/" in name or "\\" in name:
            errors.append(f"{prefix}.file 必須是純檔名")
            continue
        if name in source_names:
            errors.append(f"重複來源檔：{name}")
        source_names.add(name)
        for key, length in (("md5", 32), ("sha256", 64)):
            value = source.get(key)
            if not isinstance(value, str) or len(value) != length or not HEX.fullmatch(value):
                errors.append(f"{prefix}.{key} 格式錯誤")
        if not isinstance(source.get("size"), int) or source["size"] < 0:
            errors.append(f"{prefix}.size 格式錯誤")
        if original_dir is not None:
            path = original_dir / name
            if not path.is_file():
                errors.append(f"缺少原始來源：{name}")
            else:
                if path.stat().st_size != source.get("size"):
                    errors.append(f"來源大小不符：{name}")
                if digest(path, "md5") != source.get("md5"):
                    errors.append(f"來源 MD5 不符：{name}")
                if digest(path, "sha256") != source.get("sha256"):
                    errors.append(f"來源 SHA-256 不符：{name}")

    assets = doc.get("assets")
    if not isinstance(assets, list):
        errors.append("assets 必須是陣列")
        assets = []
    ids: set[str] = set()
    assets_by_id: dict[str, dict] = {}
    pack_root = manifest_path.parent
    for index, asset in enumerate(assets):
        prefix = f"assets[{index}]"
        if not isinstance(asset, dict):
            errors.append(f"{prefix} 必須是物件")
            continue
        asset_id = asset.get("asset_id")
        if not isinstance(asset_id, str) or not asset_id:
            errors.append(f"{prefix}.asset_id 不可為空")
        elif asset_id in ids:
            errors.append(f"重複 asset_id：{asset_id}")
        else:
            ids.add(asset_id)
            assets_by_id[asset_id] = asset
        if asset.get("kind") not in KINDS:
            errors.append(f"{prefix}.kind 未知：{asset.get('kind')!r}")
        if asset.get("status") not in STATUSES:
            errors.append(f"{prefix}.status 未知：{asset.get('status')!r}")
        if asset.get("evidence") not in EVIDENCE:
            errors.append(f"{prefix}.evidence 未知：{asset.get('evidence')!r}")
        if asset.get("source_file") not in source_names:
            errors.append(f"{prefix}.source_file 未登記：{asset.get('source_file')!r}")
        relative = asset.get("path")
        if not safe_relative(relative):
            errors.append(f"{prefix}.path 不是安全相對路徑")
            continue
        output = pack_root / relative
        if asset.get("status") == "exported":
            if not output.is_file():
                errors.append(f"缺少輸出素材：{relative}")
            else:
                expected = asset.get("sha256")
                if not isinstance(expected, str) or len(expected) != 64 or not HEX.fullmatch(expected):
                    errors.append(f"{prefix}.sha256 格式錯誤")
                elif digest(output, "sha256") != expected:
                    errors.append(f"輸出 SHA-256 不符：{relative}")

    # 複合 metadata 可讓同一標準輸出對應多個 raw resource；這些關聯
    # 必須由文件中明列的 resource 欄位證明，不能只由檔名推測。
    composite: dict[tuple[str, int], set[str]] = {}
    composite_fields = {
        "FDSHAP.DAT": ("image_resource", "control_resource"),
        "FDFIELD.DAT": ("map_resource", "control_resource", "positions_resource"),
    }
    for asset_id, asset in assets_by_id.items():
        source_file = asset.get("source_file")
        if source_file not in composite_fields or asset.get("status") != "exported":
            continue
        relative = asset.get("path")
        if not safe_relative(relative) or not str(relative).endswith(".json"):
            continue
        try:
            metadata = json.loads((pack_root / relative).read_text(encoding="utf-8"))
        except (OSError, UnicodeDecodeError, json.JSONDecodeError):
            continue
        for field in composite_fields[source_file]:
            resource = metadata.get(field)
            if isinstance(resource, int) and resource >= 0:
                composite.setdefault((source_file, resource), set()).add(asset_id)

    relationships = doc.get("relationships")
    if not isinstance(relationships, list):
        errors.append("relationships 必須是陣列")
        relationships = []
    for index, relation in enumerate(relationships):
        if not isinstance(relation, dict):
            errors.append(f"relationships[{index}] 必須是物件")
            continue
        for endpoint in ("from", "to"):
            value = relation.get(endpoint)
            if value not in ids:
                errors.append(f"relationships[{index}].{endpoint} 引用不存在：{value!r}")
    runtime_catalogs = doc.get("runtime_catalogs")
    music_resource_refs: set[str] = set()
    if runtime_catalogs is not None:
        if not isinstance(runtime_catalogs, dict) or set(runtime_catalogs) != {"music"}:
            errors.append("runtime_catalogs 只接受 music")
        else:
            bridge = runtime_catalogs["music"]
            required = {
                "kind", "asset_root", "catalog_path", "catalog_bytes", "catalog_sha256",
                "schema_version", "source_file", "profiles", "tracks", "renders",
            }
            if not isinstance(bridge, dict) or set(bridge) != required:
                errors.append("runtime_catalogs.music bridge 欄位不符")
            elif (
                bridge.get("kind") != "fd2_music_catalog"
                or bridge.get("asset_root") != "runtime_assets"
                or bridge.get("catalog_path") != "music_catalog.json"
                or not safe_relative(bridge.get("catalog_path"))
                or bridge.get("schema_version") != 1
                or bridge.get("source_file") != "FDMUS.DAT"
                or bridge.get("profiles") != 2
                or bridge.get("tracks") != 15
                or bridge.get("renders") != 30
            ):
                errors.append("runtime_catalogs.music 固定契約不符")
            elif runtime_assets is None:
                errors.append("runtime_catalogs.music 需要明確 runtime assets root")
            else:
                source = next((item for item in sources if item.get("file") == "FDMUS.DAT"), None)
                expected_hash = bridge.get("catalog_sha256")
                actual, music_errors = validate_music_assets(runtime_assets, source, expected_hash)
                errors.extend(music_errors)
                if actual is not None and actual != bridge:
                    errors.append("runtime_catalogs.music bridge metadata 不符")
                try:
                    catalog = json.loads((runtime_assets / "music_catalog.json").read_text(encoding="utf-8"))
                except (OSError, UnicodeDecodeError, json.JSONDecodeError):
                    catalog = {}
                for track in catalog.get("tracks", []):
                    if not isinstance(track, dict):
                        continue
                    index = track.get("resource_index")
                    track_id = track.get("track_id")
                    if isinstance(index, int) and isinstance(track_id, str):
                        music_resource_refs.add(f"music/{track_id}")

    source_resources = doc.get("source_resources")
    if not isinstance(source_resources, list):
        errors.append("source_resources 必須是陣列")
        source_resources = []
    raw_assets: dict[tuple[str, int], dict] = {}
    for item in assets:
        if not isinstance(item, dict) or item.get("status") != "intentionally_raw":
            continue
        raw_key = (item.get("source_file"), item.get("source_resource"))
        if raw_key in raw_assets:
            errors.append(f"重複 raw source resource：{raw_key[0]}#{raw_key[1]}")
        raw_assets[raw_key] = item
    seen_resources: set[tuple[str, int]] = set()
    for index, entry in enumerate(source_resources):
        prefix = f"source_resources[{index}]"
        if not isinstance(entry, dict):
            errors.append(f"{prefix} 必須是物件")
            continue
        required = {
            "source_file", "source_resource", "raw_asset_id", "raw_bytes", "raw_sha256",
            "disposition", "output_asset_ids", "runtime_catalog_refs", "reason_code",
        }
        if set(entry) != required:
            errors.append(f"{prefix} 欄位集合不符")
            continue
        key = (entry.get("source_file"), entry.get("source_resource"))
        if not isinstance(key[0], str) or not isinstance(key[1], int) or key[1] < 0:
            errors.append(f"{prefix} 原始定位無效")
            continue
        if key in seen_resources:
            errors.append(f"重複 source resource：{key[0]}#{key[1]}")
        seen_resources.add(key)
        raw = raw_assets.get(key)
        if raw is None:
            errors.append(f"{prefix} 找不到對應 raw asset")
            continue
        raw_path = pack_root / raw["path"]
        if (
            entry.get("raw_asset_id") != raw.get("asset_id")
            or entry.get("raw_sha256") != raw.get("sha256")
            or not isinstance(entry.get("raw_bytes"), int)
            or entry["raw_bytes"] < 0
            or not raw_path.is_file()
            or (raw_path.is_file() and raw_path.stat().st_size != entry["raw_bytes"])
            or (raw_path.is_file() and digest(raw_path, "sha256") != entry.get("raw_sha256"))
        ):
            errors.append(f"{prefix} raw identity／大小／雜湊不符")
        disposition = entry.get("disposition")
        if disposition not in DISPOSITIONS:
            errors.append(f"{prefix}.disposition 未知：{disposition!r}")
            continue
        output_ids = entry.get("output_asset_ids")
        catalog_refs = entry.get("runtime_catalog_refs")
        if not isinstance(output_ids, list) or len(output_ids) != len(set(output_ids)):
            errors.append(f"{prefix}.output_asset_ids 必須是不重複陣列")
            output_ids = []
        if not isinstance(catalog_refs, list) or len(catalog_refs) != len(set(catalog_refs)):
            errors.append(f"{prefix}.runtime_catalog_refs 必須是不重複陣列")
            catalog_refs = []
        direct = {
            asset_id for asset_id, asset in assets_by_id.items()
            if asset.get("source_file") == key[0] and asset.get("source_resource") == key[1]
            and asset.get("status") != "intentionally_raw"
        }
        allowed_outputs = direct | composite.get(key, set())
        for asset_id in output_ids:
            if asset_id not in allowed_outputs:
                errors.append(f"{prefix}.output_asset_ids 無法證明關聯：{asset_id!r}")
        for ref in catalog_refs:
            if key[0] != "FDMUS.DAT" or ref not in music_resource_refs:
                errors.append(f"{prefix}.runtime_catalog_refs 不存在：{ref!r}")
        exported = any(assets_by_id.get(asset_id, {}).get("status") == "exported" for asset_id in output_ids)
        blocked_output = any(assets_by_id.get(asset_id, {}).get("status") == "blocked" for asset_id in output_ids)
        if disposition == "standardized" and not (exported or catalog_refs):
            errors.append(f"{prefix} standardized 缺少正式輸出")
        elif disposition == "confirmed_empty" and (entry.get("raw_bytes") != 0 or output_ids or catalog_refs):
            errors.append(f"{prefix} confirmed_empty 契約不符")
        elif disposition == "blocked" and (not blocked_output or exported or catalog_refs):
            errors.append(f"{prefix} blocked 契約不符")
        elif disposition == "unknown" and (output_ids or catalog_refs or entry.get("raw_bytes") == 0):
            errors.append(f"{prefix} unknown 契約不符")
    missing_ledger = set(raw_assets) - seen_resources
    extra_ledger = seen_resources - set(raw_assets)
    if missing_ledger:
        errors.append(f"source_resources 缺少 {len(missing_ledger)} 筆 raw resource")
    if extra_ledger:
        errors.append(f"source_resources 多出 {len(extra_ledger)} 筆不存在 resource")
    if coverage_summary is not None:
        try:
            summary = json.loads(coverage_summary.read_text(encoding="utf-8"))
        except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
            errors.append(f"無法讀取 source-resource 覆蓋摘要：{exc}")
        else:
            expected_summary = build_coverage_summary(doc, digest(manifest_path, "sha256"))
            if summary != expected_summary:
                errors.append("source-resource 覆蓋摘要與 manifest 不符")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--original-dir", type=Path)
    parser.add_argument("--runtime-assets", type=Path)
    parser.add_argument("--coverage-summary", type=Path)
    args = parser.parse_args()
    errors = validate(
        args.manifest, args.original_dir, args.runtime_assets, args.coverage_summary,
    )
    if errors:
        for error in errors:
            print(f"錯誤：{error}", file=sys.stderr)
        return 1
    print(f"分離素材包驗證通過：{args.manifest}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
