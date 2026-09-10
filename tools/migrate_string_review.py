#!/usr/bin/env python3
"""在程式行號漂移後，以受控來源身分遷移字串人工審查。"""
from __future__ import annotations

import argparse
import hashlib
import json
from collections import defaultdict
from pathlib import Path


def load(path: Path):
    return json.loads(path.read_text(encoding="utf-8"))


def signature(entry: dict) -> tuple:
    source = entry["source"]
    return entry["role"], entry["text"], source.get("file"), source.get("function")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--old-inventory", type=Path, required=True)
    parser.add_argument("--new-inventory", type=Path, required=True)
    parser.add_argument("--review", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--drop-id", action="append", default=[])
    args = parser.parse_args()
    old = load(args.old_inventory)
    new = load(args.new_inventory)
    review = load(args.review)
    old_by_id = {entry["string_id"]: entry for entry in old["entries"]}
    new_by_id = {entry["string_id"]: entry for entry in new["entries"]}
    new_by_signature: dict[tuple, list[dict]] = defaultdict(list)
    for entry in new["entries"]:
        new_by_signature[signature(entry)].append(entry)
    # 同一支函式裡出現兩次同樣的字時，簽名不足以分辨。盤點是照原始碼順序產生的，
    # 而插入或刪除幾行不會改變它們彼此的先後，所以用「群組內的第幾個」配對。
    old_by_signature: dict[tuple, list[dict]] = defaultdict(list)
    for entry in old["entries"]:
        old_by_signature[signature(entry)].append(entry)
    old_rank = {
        entry["string_id"]: rank
        for group in old_by_signature.values()
        for rank, entry in enumerate(group)
    }
    dropped = set(args.drop_id)
    for group in review["dispositions"].values():
        migrated = []
        for old_id in group["string_ids"]:
            if old_id in dropped:
                continue
            old_entry = old_by_id[old_id]
            # 先看原地：同一個 id 在新盤點裡還在，而且簽名一模一樣，那它就是
            # 同一筆，不必再問簽名唯不唯一。行號漂移多半只影響改動點之後的
            # 條目，前面那一大半原地不動；而同一支函式裡出現兩次同樣的字
            # （例如 Update 裡的兩個「亞雷斯」）簽名天生就不唯一，逼它唯一
            # 只會讓整批遷移卡在一個其實沒有動過的條目上。
            same = new_by_id.get(old_id)
            if same is not None and signature(same) == signature(old_entry):
                migrated.append(old_id)
                continue
            sig = signature(old_entry)
            matches = new_by_signature[sig]
            if len(matches) == 1:
                migrated.append(matches[0]["string_id"])
                continue
            if not matches:
                raise SystemExit(f"{old_id}: 新盤點裡找不到同簽名的條目")
            if len(matches) != len(old_by_signature[sig]):
                raise SystemExit(
                    f"{old_id}: 同簽名條目由 {len(old_by_signature[sig])} 個變成 "
                    f"{len(matches)} 個，順序配對不安全")
            migrated.append(matches[old_rank[old_id]]["string_id"])
        group["string_ids"] = sorted(migrated)
    raw = args.new_inventory.read_bytes()
    review["inventory_sha256"] = hashlib.sha256(raw).hexdigest()
    args.output.write_text(json.dumps(review, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
