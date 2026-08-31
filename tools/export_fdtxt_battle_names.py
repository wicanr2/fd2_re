#!/usr/bin/env python3
"""由固定 FDTXT_000 metadata 匯出 record[8]+1 戰鬥面板名稱。"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from export_fdtxt_item_names import EXPECTED_SOURCE


FIRST_STRING = 1
RAW_NAME_COUNT = 139
EXPECTED_EMPTY = [
    *range(32, 68), 75, *range(128, 133), *range(136, 139),
]


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
    if len(strings) < FIRST_STRING + RAW_NAME_COUNT:
        raise ValueError("FDTXT_000 字串數不足")

    names: dict[str, dict[str, object]] = {}
    empty: list[int] = []
    for raw_name_id in range(RAW_NAME_COUNT):
        source_index = FIRST_STRING + raw_name_id
        row = strings[source_index]
        expected_id = f"FDTXT_000/string_{source_index:04d}"
        if row.get("source_index") != source_index or row.get("string_id") != expected_id:
            raise ValueError(f"battle name {raw_name_id} 的來源索引不連續")
        text = row.get("text")
        if not isinstance(text, str):
            raise ValueError(f"battle name {raw_name_id} 不是字串")
        if not text:
            empty.append(raw_name_id)
            continue
        names[str(raw_name_id)] = {"name": text, "source_string_id": expected_id}
    if empty != EXPECTED_EMPTY or len(names) != 94:
        raise ValueError(f"battle name 空洞或有效數量改變：{empty}, {len(names)}")

    result = {
        "schema_version": 1,
        "kind": "fd2_original_battle_names",
        "selector": "native_record_byte_8",
        "source_string_formula": "1 + native_record_byte_8",
        "source": EXPECTED_SOURCE,
        "raw_name_count": RAW_NAME_COUNT,
        "displayable_name_count": len(names),
        "confirmed_empty_name_ids": empty,
        "names": names,
    }
    args.output.write_text(
        json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )
    print(f"wrote {len(names)} battle names; confirmed empty: {len(empty)}")


if __name__ == "__main__":
    main()
