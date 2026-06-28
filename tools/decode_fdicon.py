#!/usr/bin/env python3
"""炎龍騎士團2 — 解碼 FDICON.B24 = 地圖單位 Q 版小人 sprite(1680 個 24×24)。

格式(同 FDSHAP tileset,但 tile 用 sprite 4-mode RLE 含透明):
  +0 u16 tileW(24)  +2 u16 tileH(24)  +4 u16 count(1680)
  +6 u32[count] offset 表(相對檔頭)
  各 tile:sprite 4-mode RLE(decode_figani.decode_rle),透明區=index 0

每角色一組連續 sprite(多方向 × 待機/走幀),即原版戰場地圖上的單位(非 FIGANI 戰鬥全身)。
輸出透明 PNG;index 0 設為透明。資產屬遊戲著作權,只在本機,不入庫。

用法:
  python3 decode_fdicon.py <FDICON.B24> <palette.bin> <out_dir> [start] [count]
  python3 decode_fdicon.py <FDICON.B24> <palette.bin> --overview <out.png> [n]
"""
import sys, os, struct
sys.path.insert(0, os.path.dirname(__file__))
from decode_figani import decode_rle
from decode_image import load_palette
from PIL import Image


def load(path):
    d = open(path, "rb").read()
    tw, th, cnt = struct.unpack_from("<HHH", d, 0)
    offs = [struct.unpack_from("<I", d, 6 + i * 4)[0] for i in range(cnt)]
    return d, tw, th, cnt, offs


def tile_img(d, tw, th, offs, i, cnt, pal):
    end = offs[i + 1] if i + 1 < cnt else len(d)
    px = decode_rle(d[offs[i]:end], tw, th, trans=0)
    im = Image.frombytes("P", (tw, th), bytes(px))
    im.putpalette(pal)
    rgba = im.convert("RGBA")
    pix = rgba.load()
    for y in range(th):
        for x in range(tw):
            if px[y * tw + x] == 0:
                pix[x, y] = (0, 0, 0, 0)
    return rgba


def main(argv):
    if len(argv) < 4:
        print(__doc__); return 1
    path, palp = argv[1], argv[2]
    d, tw, th, cnt, offs = load(path)
    pal = load_palette(palp)

    if argv[3] == "--overview":
        out = argv[4]
        n = int(argv[5]) if len(argv) > 5 else 256
        n = min(n, cnt)
        cols = 12
        rows = (n + cols - 1) // cols
        from PIL import ImageDraw
        scale = 2
        sheet = Image.new("RGB", (cols * (tw * scale + 2), rows * (th * scale + 10)), (30, 30, 40))
        dr = ImageDraw.Draw(sheet)
        for i in range(n):
            im = tile_img(d, tw, th, offs, i, cnt, pal).resize((tw * scale, th * scale), Image.NEAREST)
            x = (i % cols) * (tw * scale + 2)
            y = (i // cols) * (th * scale + 10)
            sheet.paste(im, (x, y + 9), im)
            dr.text((x, y), str(i), fill=(255, 255, 0))
        sheet.save(out)
        print(f"overview {n}/{cnt} ({tw}x{th}) -> {out}")
        return 0

    out = argv[3]
    os.makedirs(out, exist_ok=True)
    start = int(argv[4]) if len(argv) > 4 else 0
    num = int(argv[5]) if len(argv) > 5 else cnt
    for i in range(start, min(start + num, cnt)):
        tile_img(d, tw, th, offs, i, cnt, pal).save(os.path.join(out, f"icon_{i:04d}.png"))
    print(f"FDICON: {cnt} sprite {tw}x{th};導出 {min(num, cnt-start)} 個 -> {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
