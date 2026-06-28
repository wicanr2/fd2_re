#!/usr/bin/env python3
"""炎龍騎士團2 — 批次抽取全部戰場地圖成 PNG。

FDFIELD.DAT(解包後)每 3 個資源 = 一張地圖(構成 / 控制 / 出場);共 33 張。
構成資源:u16 W, u16 H, 然後每格 (u16 地形索引, u16 事件)。
配對:地圖 N 用 FDSHAP 第 N 個「大資源(tileset)」(實測 = FDSHAP 解包序 2N)。
tileset 格式見 render_map / doc 01 §8。

用法:
    python3 extract_maps.py <raw解包根> <palette.bin> <out目錄>
       raw 解包根需含 raw/FDFIELD/ 與 raw/FDSHAP/(unpack_dat 產出)
"""
import sys
import os
import glob
import struct

sys.path.insert(0, os.path.dirname(__file__))
from render_map import decode_tileset, load_palette, _bgrle  # noqa
from PIL import Image


def render_map(field_path, tiles, tw, th, pal, out):
    d = open(field_path, "rb").read()
    w, h = struct.unpack_from("<HH", d, 0)
    if not (0 < w < 200 and 0 < h < 200):
        return None
    img = Image.new("P", (w * tw, h * th), 0)
    img.putpalette(pal)
    for cy in range(h):
        for cx in range(w):
            ti = struct.unpack_from("<H", d, 4 + (cy * w + cx) * 4)[0]
            if ti < len(tiles):
                img.paste(Image.frombytes("P", (tw, th), tiles[ti]), (cx * tw, cy * th))
    img.convert("RGB").save(out)
    return (w, h)


def main(argv):
    if len(argv) < 4:
        print(__doc__); return 1
    raw, palp, out = argv[1], argv[2], argv[3]
    os.makedirs(out, exist_ok=True)
    pal = load_palette(palp)
    fld = sorted(glob.glob(os.path.join(raw, "FDFIELD", "*.bin")))
    shp = sorted(glob.glob(os.path.join(raw, "FDSHAP", "*.bin")))
    big = [f for f in shp if os.path.getsize(f) > 2000]   # tileset 大資源
    n = len(fld) // 3
    done = 0
    for m in range(n):
        if m >= len(big):
            break
        try:
            tw, th, tiles = decode_tileset(big[m])
            r = render_map(fld[m * 3], tiles, tw, th, pal,
                           os.path.join(out, f"map{m:02d}.png"))
            if r:
                print(f"map{m:02d}: {r[0]}x{r[1]} ({r[0]*tw}x{r[1]*th}px, {len(tiles)} tiles)")
                done += 1
        except Exception as e:
            print(f"map{m:02d}: 失敗 {e}")
    print(f"\n完成 {done}/{n} 張地圖 -> {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
