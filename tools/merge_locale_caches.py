#!/usr/bin/env python3
"""合併互斥的離線翻譯快取分片，有衝突時失敗。"""
import argparse
import json
from pathlib import Path


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("inputs", nargs="+", type=Path)
    args = parser.parse_args()
    merged = {}
    for path in args.inputs:
        shard = json.loads(path.read_text(encoding="utf-8"))
        for source, text in shard.items():
            if source in merged and merged[source] != text:
                raise SystemExit(f"快取衝突：{source}")
            merged[source] = text
    args.output.write_text(json.dumps(merged, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(f"已合併 {len(merged)} 筆互異譯文")


if __name__ == "__main__":
    main()
