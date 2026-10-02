#!/usr/bin/env python3
"""依固定 FD2.EXE 匯出原生成長的 256 個 byte selector 槽。

0x4E4D1 沒有 68 列界限；68 列以後是相鄰資料，不是作者設計的角色成長表。
原始範圍不排序，負跨距的消費契約見 fd2_ch15_ai_growth_20261002.json。
本工具只能在 Docker 中使用；--check 核對既有輸出，不修改原版。
"""
import argparse
import hashlib
import json
from pathlib import Path

SHA256 = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
FILE_BASE = 0x55EA1
LINEAR_BASE = 0x620A1


def extract(raw):
    if len(raw) != 357074 or hashlib.sha256(raw).hexdigest() != SHA256:
        raise ValueError("FD2.EXE 大小或 SHA-256 不符固定原版")
    rows = []
    for idx in range(256):
        off = FILE_BASE + idx * 11
        row = raw[off:off + 11]
        if len(row) != 11:
            raise ValueError("原生成長槽超出輸入")
        rows.append({"idx": idx, "off": hex(off), "linear": hex(LINEAR_BASE + idx * 11),
                     **{key: list(row[i:i + 2]) for key, i in
                        (("ap", 0), ("dp", 2), ("dx", 4), ("hp", 6), ("mp", 8))},
                     "learn_idx": row[10], "raw": row.hex()})
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--exe", type=Path, required=True)
    parser.add_argument("--output", type=Path, action="append", required=True)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    rows = extract(args.exe.read_bytes())
    for output in args.output:
        if args.check:
            if json.loads(output.read_text()) != rows:
                raise SystemExit(f"成長槽與固定原版不符：{output}")
        else:
            output.write_text(json.dumps(rows, ensure_ascii=False, indent=1) + "\n")
    print(f"原生成長槽 256 列{'核對' if args.check else '匯出'}通過；前68列為作者表，其後保留原版相鄰讀取")


if __name__ == "__main__":
    main()
