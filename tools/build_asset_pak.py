#!/usr/bin/env python3
"""把網頁版要用的資產打成一個檔。

為什麼不直接讓瀏覽器逐檔取用：Go 的檔案系統介面是同步回呼，在 js/wasm 底下只能
用同步 XMLHttpRequest 去補，而 Chrome 對主執行緒的同步請求節流得很兇——實測每秒
只過得了兩個檔案，而這個遊戲光是啟動就要讀六千多個。打成一個檔之後只剩一次往返，
之後全部在記憶體裡取用。

格式（刻意簡單，JS 端十行就能解）：

    magic  8 bytes  "FD2PAK01"
    u32    索引長度（小端）
    索引    JSON：{"files": {"相對路徑": [offset, length]}}，offset 從資料區起算
    資料    各檔案內容依索引順序連續排列

用法：tools/build_asset_pak.py <輸出.pak> <前綴>=<目錄> [<前綴>=<目錄>…]

前綴會接在每個相對路徑前面，對應網頁版的 assets/ 與 pack/ 兩個命名空間。
"""
from __future__ import annotations

import json
import struct
import sys
from pathlib import Path

MAGIC = b"FD2PAK01"
# 音樂單檔就有幾 MB，而且要播到才需要；留給逐檔取用，別讓包大到載不動。
SKIP_DIRS = {"music", "music_fm", "music_mt32"}


def collect(prefix: str, root: Path) -> list[tuple[str, Path]]:
    if not root.is_dir():
        raise SystemExit(f"不是目錄：{root}")
    out: list[tuple[str, Path]] = []
    for path in sorted(root.rglob("*")):
        if not path.is_file():
            continue
        relative = path.relative_to(root)
        if relative.parts and relative.parts[0] in SKIP_DIRS:
            continue
        out.append((f"{prefix}/{relative.as_posix()}", path))
    return out


def main() -> None:
    if len(sys.argv) < 3:
        raise SystemExit(__doc__)
    out = Path(sys.argv[1])
    entries: list[tuple[str, Path]] = []
    for spec in sys.argv[2:]:
        prefix, _, directory = spec.partition("=")
        if not prefix or not directory:
            raise SystemExit(f"參數要寫成 <前綴>=<目錄>：{spec}")
        entries.extend(collect(prefix.strip("/"), Path(directory)))
    if not entries:
        raise SystemExit("沒有收到任何檔案，不寫出去")

    index: dict[str, list[int]] = {}
    offset = 0
    for name, path in entries:
        size = path.stat().st_size
        index[name] = [offset, size]
        offset += size
    payload = json.dumps({"files": index}, ensure_ascii=False,
                         separators=(",", ":")).encode("utf-8")
    out.parent.mkdir(parents=True, exist_ok=True)
    with out.open("wb") as handle:
        handle.write(MAGIC)
        handle.write(struct.pack("<I", len(payload)))
        handle.write(payload)
        for _, path in entries:
            handle.write(path.read_bytes())
    total = out.stat().st_size
    print(f"已寫出 {out}：{len(entries)} 個檔案，索引 {len(payload)/1024:.0f} KB，"
          f"合計 {total/1024/1024:.1f} MB")


if __name__ == "__main__":
    main()
