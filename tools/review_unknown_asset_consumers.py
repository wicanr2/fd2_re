#!/usr/bin/env python3
"""逐筆重生素材清冊 unknown 的玩家 consumer 審查帳本。

這支工具不改寫 manifest 的 disposition，也不把局部投影冒充標準化素材。它只把
目前 93 筆 ``no_standard_output`` 與可回查的正式資料鏈交叉核對：FDFIELD 保留其
既有玩家資料 consumer；FDMUS 的三位元組項確認為非播放哨兵；FDOTHER 則記錄目前
正式玩家程式沒有登記 consumer。任何集合、raw 身分或必要入口漂移都失敗即關閉。
"""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
import struct
from collections import Counter
from pathlib import Path

import parse_field


LOCATOR_FIELDS = (
    "source_file", "source_resource", "raw_asset_id", "raw_bytes",
    "raw_sha256", "reason_code",
)
EXPECTED_COUNTS = {"FDFIELD.DAT": 79, "FDMUS.DAT": 5, "FDOTHER.DAT": 9}
EXPECTED_FDMUS = {0, 2, 5, 7, 9}
EXPECTED_FDOTHER = {47, 48, 49, 51, 52, 64, 81, 89, 96}
FDOTHER_PCM_CANDIDATES = {48, 49, 51, 52, 64}
SENTINEL = b" \r\n"


def canonical_hash(value: object) -> str:
    payload = json.dumps(
        value, ensure_ascii=False, sort_keys=True, separators=(",", ":"),
    ).encode("utf-8")
    return hashlib.sha256(payload).hexdigest()


def require_file(repo: Path, relative: str) -> None:
    if not (repo / relative).is_file():
        raise ValueError(f"缺少審查入口：{relative}")


def raw_path(pack: Path, entry: dict) -> Path:
    asset_id = entry.get("raw_asset_id", "")
    prefix = "metadata/"
    if not isinstance(asset_id, str) or not asset_id.startswith(prefix):
        raise ValueError(f"無效 raw_asset_id：{asset_id!r}")
    container = entry.get("source_file", "").split(".", 1)[0]
    resource = entry.get("source_resource")
    path = pack / "raw" / container / f"{container}_{resource:03d}.bin"
    if not path.is_file():
        raise ValueError(f"找不到 raw resource：{path}")
    data = path.read_bytes()
    if len(data) != entry.get("raw_bytes"):
        raise ValueError(f"raw 大小漂移：{entry['source_file']}#{entry['source_resource']}")
    if hashlib.sha256(data).hexdigest() != entry.get("raw_sha256"):
        raise ValueError(f"raw SHA-256 漂移：{entry['source_file']}#{entry['source_resource']}")
    return path


def verify_fdfield_projection(pack: Path, repo: Path, map_index: int, component: str) -> list[str]:
    """確認 raw resource 的玩家可用投影確實存在，不只檢查同名檔案。"""
    map_relative = f"remake/assets/maps/map{map_index}/map.json"
    units_relative = f"remake/assets/maps/map{map_index}/map{map_index}_units.json"
    map_doc = json.loads((repo / map_relative).read_text(encoding="utf-8"))
    units_doc = json.loads((repo / units_relative).read_text(encoding="utf-8"))
    info = parse_field.parse_map(str(pack / "raw"), map_index)
    if component == "composition":
        cells = info["w"] * info["h"]
        rebuilt = bytearray(struct.pack("<HH", map_doc["w"], map_doc["h"]))
        for index in range(cells):
            event_word = (
                map_doc["native_composition_event_bytes"][index]
                | (map_doc["native_tile_blit_modes"][index] << 8)
            )
            rebuilt.extend(struct.pack("<HH", map_doc["tiles"][index], event_word))
        raw = (pack / "raw" / "FDFIELD" / f"FDFIELD_{map_index * 3:03d}.bin").read_bytes()
        if bytes(rebuilt) != raw:
            raise ValueError(f"map{map_index} composition 投影不能重建 raw resource")
        return [map_relative]
    if component == "control":
        turn_relative = "remake/assets/maps/native_turn_event_controls.json"
        turns = json.loads((repo / turn_relative).read_text(encoding="utf-8"))
        turn_row = next((row for row in turns["maps"] if row["map"] == map_index), None)
        if turn_row is None or turn_row.get("controls") != info["turn_event_controls"]:
            raise ValueError(f"map{map_index} turn-event control 投影漂移")
        projected_field_events = [
            {"event_id": row["event_id"], "selector": row["selector"]}
            for row in info["field_events"]
        ]
        if map_doc.get("native_field_events") != projected_field_events:
            raise ValueError(f"map{map_index} field-event control 投影漂移")
        if units_doc.get("chests") != info["chests"]:
            raise ValueError(f"map{map_index} chest control 投影漂移")
        source_units = info["units"]
        output_units = units_doc.get("units", [])
        if len(source_units) != len(output_units):
            raise ValueError(f"map{map_index} roster control 筆數漂移")
        fields = (
            ("camp", "camp"), ("lv", "lv"), ("group", "group"),
            ("raw_unit_key", "native_record_byte8"),
            ("native_record_byte6", "native_record_byte6"),
            ("native_record_byte3d", "native_record_byte3d"),
            ("native_source_byte3", "native_source_byte3"),
            ("native_source_byte20", "native_source_byte20"),
            ("native_source_byte25", "native_source_byte25"),
            ("inventory_slots", "inventory_slots"),
            ("initial_command_mask", "initial_command_mask"),
            ("native_record_byte34", "native_record_byte34"),
            ("native_record_byte35", "native_record_byte35"),
            ("native_record_byte36", "native_record_byte36"),
            ("native_record_death_effect", "native_record_death_effect"),
        )
        for index, (source_unit, output_unit) in enumerate(zip(source_units, output_units)):
            for source_key, output_key in fields:
                if source_unit.get(source_key) != output_unit.get(output_key):
                    raise ValueError(
                        f"map{map_index} roster[{index}] {source_key} 投影漂移"
                    )
        return [map_relative, units_relative, turn_relative]

    source_positions = info["positions"]
    output_units = units_doc.get("units", [])
    for index, unit in enumerate(output_units):
        row = unit.get("native_position_record")
        want = source_positions[index]
        if row != {"x_word": want[0], "y_word": want[1], "raw_key": want[2]}:
            raise ValueError(f"map{map_index} unit position[{index}] 投影漂移")
    deploy = source_positions[len(output_units):]
    if units_doc.get("own_deploy") != [{"x": row[0], "y": row[1]} for row in deploy]:
        raise ValueError(f"map{map_index} deployment position 投影漂移")
    return [units_relative]


def review(manifest: dict, pack: Path, repo: Path) -> dict:
    unknown = [
        {key: item.get(key) for key in LOCATOR_FIELDS}
        for item in manifest.get("source_resources", [])
        if item.get("disposition") == "unknown"
    ]
    unknown.sort(key=lambda item: (item["source_file"], item["source_resource"]))
    counts = Counter(item["source_file"] for item in unknown)
    if dict(counts) != EXPECTED_COUNTS:
        raise ValueError(f"unknown 集合筆數漂移：{dict(counts)}，預期 {EXPECTED_COUNTS}")
    by_file = {
        name: {item["source_resource"] for item in unknown if item["source_file"] == name}
        for name in counts
    }
    if by_file["FDMUS.DAT"] != EXPECTED_FDMUS:
        raise ValueError("FDMUS unknown 集合漂移")
    if by_file["FDOTHER.DAT"] != EXPECTED_FDOTHER:
        raise ValueError("FDOTHER unknown 集合漂移")

    shared_paths = [
        "tools/parse_field.py",
        "tools/export_engine_assets.py",
        "tools/export_units.py",
        "remake/cmd/fd2/native_map_assets.go",
        "remake/cmd/fd2/main.go",
        "docs/knowledge-base/36-sfx-audio-data.md",
        "docs/knowledge-base/58-fd2-exe-re-coverage.md",
    ]
    for path in shared_paths:
        require_file(repo, path)

    entries = []
    for item in unknown:
        path = raw_path(pack, item)
        source = item["source_file"]
        resource = item["source_resource"]
        reviewed = dict(item)
        if source == "FDFIELD.DAT":
            map_index, component_index = divmod(resource, 3)
            if map_index > 32:
                raise ValueError(f"FDFIELD#{resource} 無對應的 0..32 地圖")
            component = ("composition", "control", "positions")[component_index]
            map_path = f"remake/assets/maps/map{map_index}/map.json"
            units_path = f"remake/assets/maps/map{map_index}/map{map_index}_units.json"
            require_file(repo, map_path)
            require_file(repo, units_path)
            data_paths = verify_fdfield_projection(pack, repo, map_index, component)
            reviewed.update({
                "review_outcome": "player_data_consumer_registered",
                "map_id": f"map{map_index}",
                "component": component,
                "producer_paths": [
                    "tools/export_engine_assets.py" if component == "composition" else "tools/export_units.py",
                    "tools/parse_field.py",
                ],
                "data_paths": data_paths,
                "consumer_paths": [
                    "remake/cmd/fd2/native_map_assets.go",
                    "remake/cmd/fd2/main.go",
                ],
                "scope_note": (
                    "確認目前可編輯地圖／隊伍資料投影與正式玩家載入端；composition 可"
                    "逐位元重建，control／positions 則核對所有現行玩家 consumer 欄位。"
                    "這不自動改變 manifest disposition。"
                ),
            })
        elif source == "FDMUS.DAT":
            if path.read_bytes() != SENTINEL:
                raise ValueError(f"FDMUS#{resource} 不是預期的 20 0d 0a 哨兵")
            reviewed.update({
                "review_outcome": "confirmed_non_playable_sentinel",
                "raw_hex": SENTINEL.hex(" "),
                "consumer_paths": [],
                "evidence_paths": ["tools/review_unknown_asset_consumers.py"],
                "scope_note": "固定三位元組 20 0d 0a，不是 MIDI／XMI／OGG 播放 payload。",
            })
        else:
            candidate = resource in FDOTHER_PCM_CANDIDATES
            reviewed.update({
                "review_outcome": "no_registered_player_consumer",
                "consumer_paths": [],
                "evidence_paths": [
                    "docs/data/asset-export-audit-20260828.json",
                    "docs/knowledge-base/36-sfx-audio-data.md",
                    "docs/knowledge-base/58-fd2-exe-re-coverage.md",
                ],
                "candidate_classification": (
                    "pcm_bank_shape_strong_inference" if candidate else "unknown"
                ),
                "scope_note": (
                    "目前正式 Game caller／分離素材清冊沒有登記玩家 consumer；"
                    "不等於證明原版資料永不可達。發現 caller 時必須重開獨立切片。"
                ),
            })
        entries.append(reviewed)

    outcomes = Counter(item["review_outcome"] for item in entries)
    return {
        "schema_version": 1,
        "kind": "fd2_unknown_asset_consumer_review",
        "pack_id": manifest.get("pack_id"),
        "reviewed_locator_sha256": canonical_hash(unknown),
        "reviewed_total": len(entries),
        "outcome_counts": dict(sorted(outcomes.items())),
        "review_scope": {
            "production_player_data": "remake/assets + remake/cmd/fd2",
            "source_oracles_excluded": True,
            "manifest_dispositions_changed": False,
        },
        "entries": entries,
        "generated_by": {
            "tool": "tools/review_unknown_asset_consumers.py",
            "version": "1",
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", type=Path, required=True)
    parser.add_argument("--repo", type=Path, default=Path("."))
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    try:
        manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
        document = review(manifest, args.manifest.parent, args.repo)
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(
            json.dumps(document, ensure_ascii=False, indent=2) + "\n", encoding="utf-8",
        )
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"錯誤：無法建立 unknown consumer 審查帳本：{exc}", file=sys.stderr)
        return 1
    print(f"{args.output}：已逐筆審查 {document['reviewed_total']} 筆")
    return 0


if __name__ == "__main__":
    sys.exit(main())
