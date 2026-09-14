#!/usr/bin/env python3
"""把素材清冊的 disposition 統計匯出成受版控的摘要。

`manifest.json` 在 `remake/generated-assets/` 底下、被 `.gitignore` 排除，所以
「還有幾筆沒交叉核對」這個狀態沒有任何受版控的訊號可以綁——工作清單那一條只能標
`manual`，而 `manual` 的意思是「沒人在看」。

這支把統計與那幾筆的定位資訊抽出來存進 `docs/data/`。摘要裡刻意**不含**原始位元組
或可重組的內容，只有來源檔、資源編號、大小與雜湊——那是定位用的，不是素材。

用法：
    python3 tools/summarize_asset_dispositions.py \\
        --manifest remake/generated-assets/fd2-original-b97caf22/manifest.json \\
        --consumer-review docs/data/asset-consumer-review.json \\
        --output docs/data/asset-disposition-summary.json
"""

import argparse
import collections
import json
import pathlib
import hashlib
import sys

# 摘要只保留這幾個欄位。raw_asset_id 是 pack 內的路徑，留著是為了回查得到那一筆；
# output_asset_ids 與 runtime_catalog_refs 對 unknown 來說一定是空的，不重複存。
LOCATOR_FIELDS = ("source_file", "source_resource", "raw_asset_id",
                  "raw_bytes", "raw_sha256", "reason_code")


def canonical_hash(value):
    payload = json.dumps(
        value, ensure_ascii=False, sort_keys=True, separators=(",", ":"),
    ).encode("utf-8")
    return hashlib.sha256(payload).hexdigest()


def summarize(manifest, review=None, review_path=None):
    resources = manifest.get("source_resources") or []
    counts = collections.Counter(entry.get("disposition") for entry in resources)
    unknown = [entry for entry in resources if entry.get("disposition") == "unknown"]
    unknown.sort(key=lambda e: (e.get("source_file") or "", e.get("source_resource") or 0))
    reviewed = []
    if review is not None:
        if review.get("kind") != "fd2_unknown_asset_consumer_review":
            raise ValueError("審查帳本 kind 不符")
        if review.get("pack_id") != manifest.get("pack_id"):
            raise ValueError("審查帳本 pack_id 不符")
        locators = [{key: entry.get(key) for key in LOCATOR_FIELDS} for entry in unknown]
        if review.get("reviewed_locator_sha256") != canonical_hash(locators):
            raise ValueError("審查帳本與目前 manifest unknown 身分不符")
        reviewed = review.get("entries") or []
        reviewed_keys = [(e.get("source_file"), e.get("source_resource")) for e in reviewed]
        unknown_keys = [(e.get("source_file"), e.get("source_resource")) for e in unknown]
        if reviewed_keys != unknown_keys or len(set(reviewed_keys)) != len(reviewed_keys):
            raise ValueError("審查帳本未逐筆且唯一覆蓋 manifest unknown")
        allowed = {
            "player_data_consumer_registered", "confirmed_non_playable_sentinel",
            "no_registered_player_consumer",
        }
        if any(e.get("review_outcome") not in allowed for e in reviewed):
            raise ValueError("審查帳本含未知 outcome")
    reviewed_keys = {(e["source_file"], e["source_resource"]) for e in reviewed}
    remaining = [
        entry for entry in unknown
        if (entry.get("source_file"), entry.get("source_resource")) not in reviewed_keys
    ]
    result = {
        "schema_version": 2,
        "kind": "fd2_asset_disposition_summary",
        "note": ("統計來自 pack 的 manifest.json（不在版控）。這份只有定位資訊，"
                 "不含原始位元組。manifest_unknown_total 保留原 disposition；"
                 "unknown_remaining 只計尚未完成 consumer review 的項目。"),
        "pack_id": manifest.get("pack_id"),
        "source_resource_count": len(resources),
        "disposition_counts": dict(sorted(counts.items())),
        "manifest_unknown_total": len(unknown),
        "reviewed_unknown_total": len(reviewed),
        "unknown_remaining": len(remaining),
        "unknown_by_source_file": dict(sorted(
            collections.Counter(e.get("source_file") for e in remaining).items())),
        "unknown": [{key: entry.get(key) for key in LOCATOR_FIELDS} for entry in remaining],
    }
    if review is not None:
        result["consumer_review"] = {
            "path": str(review_path) if review_path is not None else None,
            "sha256": (
                hashlib.sha256(pathlib.Path(review_path).read_bytes()).hexdigest()
                if review_path is not None else canonical_hash(review)
            ),
            "outcome_counts": review.get("outcome_counts"),
        }
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", type=pathlib.Path, required=True)
    parser.add_argument("--consumer-review", type=pathlib.Path)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    args = parser.parse_args()
    if not args.manifest.is_file():
        print(f"找不到清冊：{args.manifest}", file=sys.stderr)
        return 2
    manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
    review = None
    if args.consumer_review is not None:
        if not args.consumer_review.is_file():
            print(f"找不到審查帳本：{args.consumer_review}", file=sys.stderr)
            return 2
        review = json.loads(args.consumer_review.read_text(encoding="utf-8"))
    try:
        summary = summarize(manifest, review, args.consumer_review)
    except ValueError as exc:
        print(f"審查帳本無效：{exc}", file=sys.stderr)
        return 2
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(
        json.dumps(summary, ensure_ascii=False, indent=2, sort_keys=False) + "\n",
        encoding="utf-8")
    print(f"{args.output}：{summary['source_resource_count']} 筆來源資源，"
          f"unknown {summary['unknown_remaining']} 筆"
          f"（{summary['unknown_by_source_file']}）")
    return 0


if __name__ == "__main__":
    sys.exit(main())
