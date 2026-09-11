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
        --output docs/data/asset-disposition-summary.json
"""

import argparse
import collections
import json
import pathlib
import sys

# 摘要只保留這幾個欄位。raw_asset_id 是 pack 內的路徑，留著是為了回查得到那一筆；
# output_asset_ids 與 runtime_catalog_refs 對 unknown 來說一定是空的，不重複存。
LOCATOR_FIELDS = ("source_file", "source_resource", "raw_asset_id",
                  "raw_bytes", "raw_sha256", "reason_code")


def summarize(manifest):
    resources = manifest.get("source_resources") or []
    counts = collections.Counter(entry.get("disposition") for entry in resources)
    unknown = [entry for entry in resources if entry.get("disposition") == "unknown"]
    unknown.sort(key=lambda e: (e.get("source_file") or "", e.get("source_resource") or 0))
    return {
        "schema_version": 1,
        "kind": "fd2_asset_disposition_summary",
        "note": ("統計來自 pack 的 manifest.json（不在版控）。這份只有定位資訊，"
                 "不含原始位元組。unknown_remaining 歸零代表 93 筆都交叉核對完了。"),
        "pack_id": manifest.get("pack_id"),
        "source_resource_count": len(resources),
        "disposition_counts": dict(sorted(counts.items())),
        "unknown_remaining": len(unknown),
        "unknown_by_source_file": dict(sorted(
            collections.Counter(e.get("source_file") for e in unknown).items())),
        "unknown": [{key: entry.get(key) for key in LOCATOR_FIELDS} for entry in unknown],
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    args = parser.parse_args()
    if not args.manifest.is_file():
        print(f"找不到清冊：{args.manifest}", file=sys.stderr)
        return 2
    manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
    summary = summarize(manifest)
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
