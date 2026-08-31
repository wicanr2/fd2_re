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

KNOWN_ZH_HANT_FDTXT_CORRECTIONS = {
    93: ("白金勳章", "白金徽章"),
    94: ("生命之實", "生命之貫"),
    139: ("神秘裝", "神祕裝"),
}


def read_json(path: Path):
    return json.loads(path.read_text(encoding="utf-8"))


def write_json(path: Path, value) -> None:
    path.write_text(
        json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )


def build(campaign_path: Path, content_path: Path, overrides_path: Path, characters_path: Path,
          item_source_path: Path, item_supplements_path: Path, battle_source_path: Path,
          battle_supplements_path: Path, command_source_path: Path,
          command_supplements_path: Path) -> dict:
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
    item_source = read_json(item_source_path)
    if item_source["kind"] != "fd2_original_item_names" or item_source["displayable_item_count"] != 200:
        raise ValueError("完整原版物品名稱清冊格式錯誤")
    source_items = item_source["items"]
    if set(map(str, names)) - set(source_items):
        raise ValueError("商店商品 ID 不在完整原版物品清冊")
    if content["locale"] == "zh-Hant":
        for item_id, values in names.items():
            actual = next(iter(values))
            source_name = source_items[str(item_id)]["name"]
            if actual != source_name and KNOWN_ZH_HANT_FDTXT_CORRECTIONS.get(item_id) != (actual, source_name):
                raise ValueError(f"商品 {item_id} 名稱與 FDTXT 原文不符")
    supplements = read_json(item_supplements_path)
    if supplements["kind"] != "fd2_item_name_supplements":
        raise ValueError("物品翻譯補充表格式錯誤")
    missing = set(source_items) - {str(item_id) for item_id in names}
    if set(supplements["items"]) != missing:
        raise ValueError("物品翻譯補充表必須恰好涵蓋商店目錄缺項")
    locale = content["locale"]
    default_status = {
        "zh-Hant": "original_confirmed",
        "zh-Hans": "deterministic_script_conversion",
        "ja": "machine_draft",
        "en": "machine_draft",
    }[locale]
    items = {
        str(item_id): {
            "name": locale_overrides.get(str(item_id), next(iter(names[item_id]))),
            "source_string_ids": [source_items[str(item_id)]["source_string_id"], *sorted(sources[item_id])],
            "status": default_status,
        }
        for item_id in sorted(names)
    }
    for raw_id in sorted(missing, key=int):
        if locale == "zh-Hant":
            name = source_items[raw_id]["name"]
            status = "original_confirmed"
        else:
            name = supplements["items"][raw_id][locale]
            status = supplements["status"][locale]
        if not name:
            raise ValueError(f"物品 {raw_id} 的 {locale} 名稱為空")
        items[raw_id] = {
            "name": name,
            "source_string_ids": [source_items[raw_id]["source_string_id"]],
            "status": status,
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
    battle_source = read_json(battle_source_path)
    if battle_source["kind"] != "fd2_original_battle_names" or battle_source["displayable_name_count"] != 94:
        raise ValueError("完整原版戰鬥姓名清冊格式錯誤")
    battle_source_names = battle_source["names"]
    supplements = read_json(battle_supplements_path)
    if supplements["kind"] != "fd2_battle_name_supplements":
        raise ValueError("戰鬥姓名翻譯補充表格式錯誤")
    party_ids = {str(row["native_identity"]) for row in character_rows}
    if set(supplements["names"]) != set(battle_source_names) - party_ids:
        raise ValueError("戰鬥姓名翻譯補充表必須恰好涵蓋非隊伍名稱")
    battle_names = {}
    for raw_id, source_entry in battle_source_names.items():
        if raw_id in party_ids:
            name = characters[raw_id]["name"]
            status = characters[raw_id]["status"]
            if locale == "zh-Hant" and name != source_entry["name"]:
                raise ValueError(f"戰鬥姓名 {raw_id} 與 FDTXT 原文不符")
        elif locale == "zh-Hant":
            name = source_entry["name"]
            status = "original_confirmed"
        else:
            name = supplements["names"][raw_id][locale]
            status = supplements["status"][locale]
        if not name:
            raise ValueError(f"戰鬥姓名 {raw_id} 的 {locale} 名稱為空")
        battle_names[raw_id] = {
            "name": name,
            "source_string_ids": [source_entry["source_string_id"]],
            "status": status,
        }
    command_source = read_json(command_source_path)
    if command_source["schema"] != "fd2.native_command_labels.v1":
        raise ValueError("原版指令名稱清冊格式錯誤")
    source_commands = {
        str(entry["command_id"]): entry
        for entry in command_source["entries"]
        if 0 <= entry["command_id"] <= 35 and entry["label"]
    }
    command_supplements = read_json(command_supplements_path)
    if command_supplements["kind"] != "fd2_command_name_supplements" or \
            set(command_supplements["names"]) != set(source_commands) or len(source_commands) != 35:
        raise ValueError("指令名稱補充表必須恰好涵蓋35個原版非空名稱")
    command_names = {}
    corrections = command_supplements["source_glyph_corrections"]
    for raw_id, source_entry in source_commands.items():
        name = command_supplements["names"][raw_id][locale]
        if not name:
            raise ValueError(f"指令名稱 {raw_id} 的 {locale} 名稱為空")
        if locale == "zh-Hant" and name != source_entry["label"]:
            correction = corrections.get(raw_id)
            if correction is None or correction["decoded_source"] != source_entry["label"] or \
                    correction["display"] != name:
                raise ValueError(f"指令名稱 {raw_id} 與來源不符且無受控勘誤")
        status = "original_confirmed"
        if locale != "zh-Hant" or raw_id in corrections:
            status = command_supplements["status"][locale]
        command_names[raw_id] = {
            "name": name,
            "source_string_ids": [f"FDTXT_000/string_{source_entry['string_index']:04d}"],
            "status": status,
        }
    return {
        "schema_version": 1,
        "kind": "fd2_locale_entities",
        "locale": content["locale"],
        "source_locale": "zh-Hant",
        "item_count": len(items),
        "items": items,
        "character_count": len(characters),
        "characters": characters,
        "battle_name_count": len(battle_names),
        "battle_names": battle_names,
        "command_name_count": len(command_names),
        "command_names": command_names,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--campaign", type=Path, required=True)
    parser.add_argument("--content", type=Path, required=True)
    parser.add_argument("--overrides", type=Path, required=True)
    parser.add_argument("--characters", type=Path, required=True)
    parser.add_argument("--item-source", type=Path, required=True)
    parser.add_argument("--item-supplements", type=Path, required=True)
    parser.add_argument("--battle-source", type=Path, required=True)
    parser.add_argument("--battle-supplements", type=Path, required=True)
    parser.add_argument("--command-source", type=Path, required=True)
    parser.add_argument("--command-supplements", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    result = build(args.campaign, args.content, args.overrides, args.characters,
                   args.item_source, args.item_supplements, args.battle_source,
                   args.battle_supplements, args.command_source, args.command_supplements)
    write_json(args.output, result)
    print(f"{result['locale']}: wrote {result['item_count']} item names to {args.output}")


if __name__ == "__main__":
    main()
