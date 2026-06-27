#!/usr/bin/env python3
"""炎龍騎士團2 — 文本與自製字型解碼(第 3 輪)。

兩項已破解的格式:

1. **文本(FDTXT.DAT)**:每個資源是一組字串。
   - 開頭 uint16 LE 次目錄(字串數 = offsets[0] / 2),指向各字串。
   - 每字串 = 一串 **uint16 LE**:
       0x0000..0x071F  字型字模索引(glyph id;0-9=數字,10+=英數,高位=漢字)
       0xFF00..0xFFFE  控制碼(格式/換行/名稱插入等)
       0xFFFF          字串結束
   - 不是 Big5;漢堂用自製字型 + 內部索引(1990 年代台灣遊戲常見做法)。

2. **字型(FDOTHER.DAT 資源 #4)**:1824 個字模,每字模 **16×16 1bpp**(32 byte:16 列 ×2 byte,MSB 在左)。
   58368 = 1824 × 32。索引即文本中的 glyph id。

用法:
    python3 decode_text.py font   <FDOTHER_004.bin> <out_atlas.png>
    python3 decode_text.py render <FDOTHER_004.bin> <FDTXT_NNN.bin> <out.png>
    python3 decode_text.py dump   <FDTXT_NNN.bin>            # 印 glyph-id 序列
"""
import sys
import struct
from PIL import Image

GLYPH_W = GLYPH_H = 16
GLYPH_BYTES = 32
CTRL_MIN = 0xFF00
STR_END = 0xFFFF


def load_font(path):
    raw = open(path, "rb").read()
    return raw, len(raw) // GLYPH_BYTES


def render_glyph(font, idx):
    g = font[idx * GLYPH_BYTES: idx * GLYPH_BYTES + GLYPH_BYTES]
    im = Image.new("L", (GLYPH_W, GLYPH_H), 0)
    if len(g) < GLYPH_BYTES:
        return im
    px = im.load()
    for r in range(GLYPH_H):
        bits = (g[r * 2] << 8) | g[r * 2 + 1]
        for c in range(GLYPH_W):
            if bits & (0x8000 >> c):
                px[c, r] = 255
    return im


def parse_strings(path):
    d = open(path, "rb").read()
    if len(d) < 2:
        return []
    first = struct.unpack_from("<H", d, 0)[0]
    if first == 0 or first % 2:
        return []
    n = first // 2
    offs = [struct.unpack_from("<H", d, 2 * i)[0] for i in range(n)]
    strings = []
    for i in range(n):
        s = offs[i]
        e = offs[i + 1] if i + 1 < n else len(d)
        seq = []
        j = s
        while j + 1 < e:
            ch = struct.unpack_from("<H", d, j)[0]
            j += 2
            if ch == STR_END:
                break
            seq.append(ch)
        strings.append(seq)
    return strings


def font_atlas(font_path, out, cols=32):
    font, n = load_font(font_path)
    rows = (n + cols - 1) // cols
    cell = GLYPH_W + 1
    atlas = Image.new("L", (cols * cell, rows * cell), 40)
    for idx in range(n):
        atlas.paste(render_glyph(font, idx), ((idx % cols) * cell, (idx // cols) * cell))
    atlas.save(out)
    print(f"字型 atlas:{n} 個字模 -> {out}")


def render_text(font_path, txt_path, out, max_rows=40, cell=17):
    font, _ = load_font(font_path)
    strings = parse_strings(txt_path)
    rows = [s for s in strings if any(c < 0x720 for c in s)][:max_rows]
    maxw = max((sum(1 for c in s if c < 0x720) for s in rows), default=1)
    canvas = Image.new("L", (maxw * cell + 4, len(rows) * cell + 4), 30)
    for r, seq in enumerate(rows):
        x = 0
        for ch in seq:
            if ch < 0x720:
                canvas.paste(render_glyph(font, ch), (2 + x * cell, 2 + r * cell))
                x += 1
    canvas.save(out)
    print(f"渲染 {len(rows)} 條字串 -> {out}")


def dump(txt_path):
    strings = parse_strings(txt_path)
    print(f"{txt_path}: {len(strings)} 條字串")
    for i, s in enumerate(strings[:30]):
        disp = " ".join(f"{c:04x}" for c in s)
        print(f"  [{i}] {disp}")


def main(argv):
    if len(argv) < 2:
        print(__doc__); return 1
    cmd = argv[1]
    if cmd == "font":
        font_atlas(argv[2], argv[3])
    elif cmd == "render":
        render_text(argv[2], argv[3], argv[4])
    elif cmd == "dump":
        dump(argv[2])
    else:
        print(__doc__); return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
