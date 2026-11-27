#!/usr/bin/env python3
"""列出 Rust fd2-game 產品碼裡還留著的 todo!() 樁（R8 切換前的缺口清單）。

只掃非測試檔；每處印出檔案、行號與 todo! 訊息的第一行，最後依標記（R6／R8／其他）計數。
訊息裡的 R6／r6 表示 HD 主題的缺口，R8／r8 表示原版流程尚未補移植。

用法：tools/rust_todo_inventory.py [--summary]
"""

import re
import sys
from collections import Counter
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1] / "remake-rs" / "crates" / "fd2-game" / "src"
MESSAGE = re.compile(r'todo!\(\s*"([^"\\]*)')


def inventory():
    rows = []
    for path in sorted(ROOT.rglob("*.rs")):
        if path.name.endswith("_test.rs") or "parity_replay" in path.parts:
            continue
        lines = path.read_text(encoding="utf-8").splitlines()
        for number, line in enumerate(lines, 1):
            code = line.split("//", 1)[0]
            if "todo!(" not in code:
                continue
            window = "\n".join(lines[number - 1:number + 3])
            match = MESSAGE.search(window)
            message = match.group(1) if match else ""
            tag = "R6" if re.match(r"(?i)r6", message) else "R8" if re.match(r"(?i)r8", message) else "other"
            rows.append((path.relative_to(ROOT).as_posix(), number, tag, message))
    return rows


def main():
    rows = inventory()
    if "--summary" not in sys.argv:
        for path, number, tag, message in rows:
            print(f"{tag}\t{path}:{number}\t{message}")
    counts = Counter(tag for _, _, tag, _ in rows)
    print(f"共 {len(rows)} 處：R8 {counts['R8']}、R6 {counts['R6']}、其他 {counts['other']}")


if __name__ == "__main__":
    main()
