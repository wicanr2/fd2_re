#!/usr/bin/env python3
"""把既有 occurrence 翻譯正規化成以遊戲實體 ID 為鍵的語系目錄。"""

from __future__ import annotations

import argparse
import json
import re
from collections import defaultdict
from pathlib import Path


ITEM_ID = re.compile(
    r"^legacy\.json\.remake\.assets\.scenarios\.campaign_full\.nodes\."
    r"([^.]+)\.(goods|secret)\.(\d+)\.name$"
)


def read_json(path: Path):
    return json.loads(path.read_text(encoding="utf-8"))


def write_json(path: Path, value) -> None:
    path.write_text(
        json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )


def build(campaign_path: Path, content_path: Path, overrides_path: Path, characters_path: Path) -> dict:
    campaign = read_json(campaign_path)
    content = read_json(content_path)
    names: dict[int, set[str]] = defaultdict(set)
    sources: dict[int, list[str]] = defaultdict(list)
    for entry in content["entries"]:
        match = ITEM_ID.fullmatch(entry["string_id"])
        if match is None:
            continue
        node_id, group, raw_index = match.groups()
        node = campaign["nodes"].get(node_id)
        if node is None or group not in node:
            raise ValueError(f"找不到商品來源：{entry['string_id']}")
        index = int(raw_index)
        if index < 0 or index >= len(node[group]):
            raise ValueError(f"商品索引越界：{entry['string_id']}")
        if entry["role"] != "entity_name" or entry["status"] == "blocked":
            raise ValueError(f"商品翻譯不可使用：{entry['string_id']}")
        item_id = node[group][index]["id"]
        names[item_id].add(entry["text"])
        sources[item_id].append(entry["string_id"])
    if not names:
        raise ValueError("內容目錄沒有 campaign_full 商品名稱")
    conflicts = {item_id: values for item_id, values in names.items() if len(values) != 1}
    if conflicts:
        raise ValueError(f"同一商品 ID 有多種譯名：{conflicts}")
    overrides = read_json(overrides_path)
    locale_overrides = overrides["locales"].get(content["locale"], {})
    unknown = set(locale_overrides) - {str(item_id) for item_id in names}
    if unknown:
        raise ValueError(f"override 指向不存在的商品 ID：{sorted(unknown)}")
    items = {
        str(item_id): {
            "name": locale_overrides.get(str(item_id), next(iter(names[item_id]))),
            "source_string_ids": sorted(sources[item_id]),
        }
        for item_id in sorted(names)
    }
    character_source = read_json(characters_path)
    if character_source["kind"] != "fd2_party_character_names":
        raise ValueError("角色名稱來源格式錯誤")
    character_rows = character_source["characters"]
    identities = [row["native_identity"] for row in character_rows]
    if identities != list(range(32)):
        raise ValueError("角色 native_identity 必須完整且依序涵蓋 0..31")
    locale = content["locale"]
    status = character_source["name_status"].get(locale)
    if status not in {"original_confirmed", "deterministic_script_conversion", "curated_remake_transliteration"}:
        raise ValueError(f"角色名稱狀態不可用：{locale}")
    characters = {
        str(row["native_identity"]): {
            "name": row[locale],
            "source_string_ids": [
                f"FDTXT_000/string_{row['native_identity'] + 1:04d}",
                f"native_identity/{row['native_identity']}",
            ],
            "status": status,
        }
        for row in character_rows
    }
    if any(not entry["name"] for entry in characters.values()):
        raise ValueError(f"角色名稱不可為空：{locale}")
    return {
        "schema_version": 1,
        "kind": "fd2_locale_entities",
        "locale": content["locale"],
        "source_locale": "zh-Hant",
        "item_count": len(items),
        "items": items,
        "character_count": len(characters),
        "characters": characters,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--campaign", type=Path, required=True)
    parser.add_argument("--content", type=Path, required=True)
    parser.add_argument("--overrides", type=Path, required=True)
    parser.add_argument("--characters", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    result = build(args.campaign, args.content, args.overrides, args.characters)
    write_json(args.output, result)
    print(f"{result['locale']}: wrote {result['item_count']} item names to {args.output}")


if __name__ == "__main__":
    main()
