#!/usr/bin/env python3
"""炎龍騎士團2 — 把一張戰場輸出成 Go/Ebiten 引擎用的資產(tileset 網格 PNG + 地圖 JSON)。

輸出:
  <out>/tileset.png  該 tileset 全部 24×24 圖塊排成網格(cols 欄)
  <out>/map.json     {"w","h","tileW","tileH","cols","tiles":[地形索引...]}

引擎(remake/cmd/fd2)讀這兩個檔即可渲染地圖。資產屬遊戲著作權,只在本機,不入庫。

用法:
    python3 export_engine_assets.py <FDFIELD構成.bin> <FDSHAP_tileset.bin> <palette.bin> <out目錄> [cols]
"""
import sys
import os
import json
import struct
from PIL import Image

sys.path.insert(0, os.path.dirname(__file__))
from render_map import decode_tileset, load_palette


def main(argv):
    if len(argv) < 5:
        print(__doc__); return 1
    fieldp, shapp, palp, out = argv[1], argv[2], argv[3], argv[4]
    cols = int(argv[5]) if len(argv) > 5 else 16
    os.makedirs(out, exist_ok=True)
    pal = load_palette(palp)
    tw, th, tiles = decode_tileset(shapp)
    rows = (len(tiles) + cols - 1) // cols
    sheet = Image.new("P", (cols * tw, rows * th), 0)
    sheet.putpalette(pal)
    for i, px in enumerate(tiles):
        sheet.paste(Image.frombytes("P", (tw, th), px), ((i % cols) * tw, (i // cols) * th))
    sheet.convert("RGB").save(os.path.join(out, "tileset.png"))

    d = open(fieldp, "rb").read()
    w, h = struct.unpack_from("<HH", d, 0)
    tilesidx = [struct.unpack_from("<H", d, 4 + i * 4)[0] for i in range(w * h)]
    meta = {"w": w, "h": h, "tileW": tw, "tileH": th, "cols": cols, "tiles": tilesidx}
    json.dump(meta, open(os.path.join(out, "map.json"), "w"), separators=(",", ":"))
    print(f"tileset.png ({cols}×{rows} 圖塊) + map.json ({w}×{h}) -> {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
