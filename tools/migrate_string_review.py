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
    new_by_signature: dict[tuple, list[dict]] = defaultdict(list)
    for entry in new["entries"]:
        new_by_signature[signature(entry)].append(entry)
    dropped = set(args.drop_id)
    for group in review["dispositions"].values():
        migrated = []
        for old_id in group["string_ids"]:
            if old_id in dropped:
                continue
            matches = new_by_signature[signature(old_by_id[old_id])]
            if len(matches) != 1:
                raise SystemExit(f"{old_id}: 遷移候選數 {len(matches)}")
            migrated.append(matches[0]["string_id"])
        group["string_ids"] = sorted(migrated)
    raw = args.new_inventory.read_bytes()
    review["inventory_sha256"] = hashlib.sha256(raw).hexdigest()
    args.output.write_text(json.dumps(review, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
