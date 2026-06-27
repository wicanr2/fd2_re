#!/usr/bin/env python3
"""炎龍騎士團2 — 字模網格渲染(供建立 glyph→Unicode 對照表)。

把自製字型(FDOTHER 資源 #4,1824 個 16×16 字模)分批渲染成帶索引標籤的大網格,
方便人工 / 多模態逐格判讀,建立 glyph 索引 → Unicode 對照表。

用法:
    python3 font_grid.py <FDOTHER_004.bin> <start> <count> <out.png> [scale] [cols]
"""
import sys
from PIL import Image, ImageDraw

GW = GH = 16
GB = 32


def render_glyph(font, idx, scale):
    g = font[idx * GB: idx * GB + GB]
    im = Image.new("L", (GW, GH), 0)
    if len(g) >= GB:
        px = im.load()
        for r in range(GH):
            bits = (g[r * 2] << 8) | g[r * 2 + 1]
            for c in range(GW):
                if bits & (0x8000 >> c):
                    px[c, r] = 255
    return im.resize((GW * scale, GH * scale), Image.NEAREST)


def main(argv):
    font = open(argv[1], "rb").read()
    start = int(argv[2]); count = int(argv[3]); out = argv[4]
    scale = int(argv[5]) if len(argv) > 5 else 3
    cols = int(argv[6]) if len(argv) > 6 else 12
    gw = GW * scale
    cellw = gw + 30          # 留索引標籤空間
    cellh = gh = GH * scale + 16
    rows = (count + cols - 1) // cols
    img = Image.new("RGB", (cols * cellw + 4, rows * cellh + 4), (20, 20, 24))
    dr = ImageDraw.Draw(img)
    for k in range(count):
        idx = start + k
        cx = (k % cols) * cellw + 2
        cy = (k // cols) * cellh + 2
        dr.text((cx, cy), f"{idx}", fill=(120, 200, 120))      # 十進位索引
        gly = render_glyph(font, idx, scale).convert("RGB")
        img.paste(gly, (cx, cy + 12))
    img.save(out)
    print(f"glyph {start}..{start+count-1} -> {out}  ({img.size[0]}x{img.size[1]})")


if __name__ == "__main__":
    sys.exit(main(sys.argv))
