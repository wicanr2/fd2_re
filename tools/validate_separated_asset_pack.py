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

HEX = re.compile(r"^[0-9a-f]+$")
KINDS = {
    "map", "tileset", "map_sprite", "portrait", "battle_animation",
    "cutscene_animation", "background", "ui", "text", "sfx", "music",
    "font", "metadata",
}
STATUSES = {"exported", "intentionally_raw", "blocked"}
EVIDENCE = {"confirmed", "strong_inference", "hypothesis", "unknown"}


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
) -> list[str]:
    errors: list[str] = []
    try:
        doc = json.loads(manifest_path.read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        return [f"無法讀取 manifest：{exc}"]

    if doc.get("schema_version") != 1:
        errors.append("schema_version 必須為 1")
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
    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--original-dir", type=Path)
    parser.add_argument("--runtime-assets", type=Path)
    args = parser.parse_args()
    errors = validate(args.manifest, args.original_dir, args.runtime_assets)
    if errors:
        for error in errors:
            print(f"錯誤：{error}", file=sys.stderr)
        return 1
    print(f"分離素材包驗證通過：{args.manifest}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
