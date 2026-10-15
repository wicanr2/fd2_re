#!/usr/bin/env python3
"""盤點已宣告的現代素材需求。靜態缺件不等於正常玩家路徑已重現阻擋。

Docker內：python3 tools/audit_modern_theme_coverage.py --verify-private --output <report.json>
故事與scenario只輸出來源JSON pointer，不複製台詞，也不以圖像索引推定角色身分。
"""
import argparse
import collections
import hashlib
import json
from pathlib import Path

from validate_modern_theme_catalog import validate

ROOT = Path(__file__).resolve().parent.parent


def walk(value, pointer=""):
    if isinstance(value, dict):
        yield pointer, value
        for key, item in value.items():
            escaped = key.replace("~", "~0").replace("/", "~1")
            yield from walk(item, pointer + "/" + escaped)
    elif isinstance(value, list):
        for index, item in enumerate(value):
            yield from walk(item, pointer + "/" + str(index))


def audit(verify_private=False):
    catalog = validate(verify_private)
    portraits = {a["speaker_id"] for a in catalog["assets"] if a["role"] == "story_portrait_frame"}
    sprites = {a["source_group"] for a in catalog["assets"] if a["role"] == "map_sprite_set"}
    maps = {a["map_id"] for a in catalog["assets"] if a["role"] == "map_tileset_set"}
    declarations = collections.defaultdict(list)
    sprite_declarations = collections.defaultdict(list)
    map_declarations = collections.defaultdict(list)
    sources = {}
    for directory in ("remake/assets/story", "remake/assets/scenarios"):
        for source in sorted((ROOT / directory).glob("*.json")):
            name = str(source.relative_to(ROOT))
            raw = source.read_bytes()
            sources[name] = hashlib.sha256(raw).hexdigest()
            for pointer, item in walk(json.loads(raw)):
                location = {"source": name, "pointer": pointer}
                if type(item.get("speaker")) is int and isinstance(item.get("text"), str):
                    declarations[item["speaker"]].append(location)
                if type(item.get("fig")) is int:
                    sprite_declarations[item["fig"]].append(location)
                if type(item.get("map")) is int:
                    map_declarations[item["map"]].append(location)
    def missing(declared, present, exceptions=()):
        return [{"id": key, "declaration_count": len(declared[key]),
                 "examples": declared[key][:4]} for key in sorted(set(declared) - present - set(exceptions))]
    consumers = ["remake/cmd/fd2/packaged_startup.go", "remake/cmd/fd2/native_story_dialogue.go",
                 "remake/cmd/fd2/modern_story_portrait.go"]
    return {
        "schema_version": 1,
        "kind": "fd2_modern_theme_static_declaration_inventory",
        "scope": "故事及scenario宣告的來源候選；包含未證明可達或可能已被canonical取代的資料，不作完整玩家需求或可通關判準。",
        "runtime_reachability_verified": False,
        "theme_id": catalog["theme_id"],
        "catalog_sha256": hashlib.sha256((ROOT / "remake/assets/themes/modern/catalog.json").read_bytes()).hexdigest(),
        "private_assets_verified": verify_private,
        "sources": sources,
        "consumer_source_sha256": {p: hashlib.sha256((ROOT / p).read_bytes()).hexdigest() for p in consumers},
        "registered": {"portraits": sorted(portraits), "sprites": sorted(sprites), "maps": sorted(maps)},
        "declared_speakers": sorted(declarations),
        "missing_portrait_declarations": missing({k: v for k, v in declarations.items() if k >= 0}, portraits),
        "negative_speaker_declarations": [{"id": key, "declaration_count": len(declarations[key]),
                                           "examples": declarations[key][:4]}
                                          for key in sorted(declarations) if key < 0],
        "missing_sprite_declarations": missing(sprite_declarations, sprites, (39, 57)),
        "missing_map_declarations": missing(map_declarations, maps),
        "intentional_destination_remap_selectors": [39, 57],
        "unassessed": ["canonical圖的實際可達性與選定scene／line範圍", "動態handler與商店／整備的說話者", "正常GPU圖層返回", "高解析呈現"],
        "limits": "不得把missing直接當產品缺陷，或把零missing當全玩家路徑覆蓋。先以正式consumer或正常輸入確認需要，再依61核准範圍補素材。",
    }


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--verify-private", action="store_true")
    p.add_argument("--output", type=Path, required=True)
    a = p.parse_args()
    if not Path("/.dockerenv").is_file():
        raise SystemExit("本工具只在Docker內執行")
    report = audit(a.verify_private)
    a.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"registered": {k: len(v) for k, v in report["registered"].items()},
                      "declared_speakers": len(report["declared_speakers"]),
                      "missing_portrait_ids": [v["id"] for v in report["missing_portrait_declarations"]],
                      "missing_sprite_ids": [v["id"] for v in report["missing_sprite_declarations"]],
                      "missing_map_ids": [v["id"] for v in report["missing_map_declarations"]],
                      "runtime_reachability_verified": False}, ensure_ascii=False))


if __name__ == "__main__":
    main()
