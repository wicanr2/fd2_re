#!/usr/bin/env python3
"""由固定 FDTXT_000 metadata 匯出可顯示物品名稱與 confirmed-empty ID。"""

from __future__ import annotations

import argparse
import json
from pathlib import Path


EXPECTED_SOURCE = {
    "file": "FDTXT.DAT",
    "resource": 0,
    "size": 120502,
    "md5": "fe5c487ce4313485f1da9d48d35b05f9",
    "sha256": "a4555f8a0e61e884b4f504d56a8bdde11672583bbbbc6506281ae10dcdfb1f69",
}
FIRST_STRING = 181
ITEM_COUNT = 215


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--resource", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    resource = json.loads(args.resource.read_text(encoding="utf-8"))
    if resource.get("kind") != "fdtxt_word_table" or resource.get("evidence") != "confirmed":
        raise ValueError("FDTXT_000 metadata 不是已證實 word table")
    source = resource.get("source", {})
    for key, expected in EXPECTED_SOURCE.items():
        if source.get(key) != expected:
            raise ValueError(f"FDTXT_000 source {key} 不符：{source.get(key)!r}")
    strings = resource.get("strings", [])
    if len(strings) < FIRST_STRING + ITEM_COUNT:
        raise ValueError("FDTXT_000 字串數不足")

    items: dict[str, dict[str, object]] = {}
    empty_ids: list[int] = []
    for item_id in range(ITEM_COUNT):
        source_index = FIRST_STRING + item_id
        row = strings[source_index]
        expected_id = f"FDTXT_000/string_{source_index:04d}"
        if row.get("source_index") != source_index or row.get("string_id") != expected_id:
            raise ValueError(f"FDTXT item {item_id} 的來源索引不連續")
        name = row.get("text")
        if not isinstance(name, str):
            raise ValueError(f"FDTXT item {item_id} 名稱不是字串")
        if not name:
            empty_ids.append(item_id)
            continue
        items[str(item_id)] = {"name": name, "source_string_id": expected_id}

    if empty_ids != list(range(108, 123)) or len(items) != 200:
        raise ValueError(f"FDTXT item 空洞或有效數量改變：{empty_ids}, {len(items)}")
    result = {
        "schema_version": 1,
        "kind": "fd2_original_item_names",
        "source": EXPECTED_SOURCE,
        "first_string_index": FIRST_STRING,
        "raw_item_count": ITEM_COUNT,
        "displayable_item_count": len(items),
        "confirmed_empty_item_ids": empty_ids,
        "items": items,
    }
    args.output.write_text(
        json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )
    print(f"wrote {len(items)} names; confirmed empty: {empty_ids}")


if __name__ == "__main__":
    main()
