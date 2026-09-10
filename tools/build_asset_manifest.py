#!/usr/bin/env python3
"""產生網頁版的資產清單。

為什麼需要它：`assetGlob` 在桌面上走 `filepath.Glob` 掃目錄，但 HTTP 沒有「列
目錄」這回事，js/wasm 底下也沒有檔案系統可掃。清單把那三個萬用字元查找變成一次
索引比對。

只收 `assetGlob` 實際用到的目錄——多收沒有壞處，但清單是要載進瀏覽器的，列 2976
個檔案裡用不到的那 900 個只是讓它變大。新增 glob 樣式時要回來加目錄，否則那個
樣式在網頁版會回空（而不是報錯）。

用法：tools/build_asset_manifest.py [輸出檔]
"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
ASSETS = ROOT / "remake" / "assets"
# 與 remake/cmd/fd2 裡的 assetGlob 呼叫一一對應。
GLOB_DIRS = ("figani", "portraits", "sprites")


def globbed_patterns() -> set[str]:
    """從產品碼抓出實際用到的 glob 樣式，用來檢查清單有沒有漏收目錄。"""
    found: set[str] = set()
    for path in (ROOT / "remake" / "cmd" / "fd2").glob("*.go"):
        if path.name.endswith("_test.go"):
            continue
        found.update(re.findall(r'assetGlob\("([^"]+)"', path.read_text(encoding="utf-8")))
    return found


def main() -> None:
    out = Path(sys.argv[1]) if len(sys.argv) > 1 else ROOT / "remake" / "web" / "dist" / "assets-manifest.json"
    patterns = globbed_patterns()
    missing = sorted(
        p for p in patterns
        if not any(p.startswith(f"assets/{d}/") for d in GLOB_DIRS)
    )
    if missing:
        raise SystemExit(
            "這些 assetGlob 樣式不在清單收錄的目錄裡，網頁版會拿到空結果："
            + "、".join(missing)
        )
    files: list[str] = []
    for name in GLOB_DIRS:
        directory = ASSETS / name
        if not directory.is_dir():
            raise SystemExit(f"找不到資產目錄：{directory}")
        for path in sorted(directory.rglob("*")):
            if path.is_file():
                files.append(str(path.relative_to(ROOT / "remake")).replace("\\", "/"))
    if not files:
        raise SystemExit("清單是空的，不寫出去")
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(
        json.dumps({"schema_version": 1, "kind": "fd2_web_asset_manifest",
                    "globbed_dirs": list(GLOB_DIRS), "files": files},
                   ensure_ascii=False, indent=1) + "\n",
        encoding="utf-8")
    print(f"已寫出 {out}（{len(files)} 個檔案，涵蓋 {len(patterns)} 個 glob 樣式）")


if __name__ == "__main__":
    main()
