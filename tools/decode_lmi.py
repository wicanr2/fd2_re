#!/usr/bin/env python3
"""LMI1 容器(FDOTHER #5 等 UI sprite 集)解碼。

LMI1 結構:
  +0   char[4]  "LMI1"
  +4   uint16   sub-resource 數 N
  +6   uint32[N] 各 sub-resource offset(相對檔頭)
  各 sub-resource: uint16 w, uint16 h, 接 codec 資料(見下)

像素 codec(反組譯 FD2.EXE 0x4e916,逐像素取值):
  讀控制 byte c:
    c <= 0xC0 : c 本身就是一個像素值(literal 單 px)
    c >  0xC0 : run,長度 = c-0xC0,後跟 1 個像素值,重複該長度
  (透明 = palette index 0;run 可跨行,線性解 w*h px)
  → 純 literal 的小圖(如 1xN 血條 cell)剛好等同 raw。

用法:
  python3 decode_lmi.py <FDOTHER_005.bin> <palette.bin> <out目錄> [idx...]
  不給 idx = 列出所有 sub-resource 的 w×h;給 idx = 解出該些為 PNG(index0 透明)。
"""
import struct
import sys
import os
from PIL import Image


def lmi_offsets(d):
    assert d[:4] == b"LMI1", "非 LMI1 容器"
    n = struct.unpack_from("<H", d, 4)[0]
    return [struct.unpack_from("<I", d, 6 + i * 4)[0] for i in range(n)]


def decode_pixels(body, total):
    """0x4e916 codec → index bytes(透明=0)。"""
    out = bytearray()
    i = 0
    while len(out) < total and i < len(body):
        c = body[i]
        i += 1
        if c <= 0xC0:
            out.append(c)
        else:
            cnt = c - 0xC0
            v = body[i]
            i += 1
            out += bytes([v]) * cnt
    return bytes(out[:total]).ljust(total, b"\x00")


def load_palette(path):
    p = open(path, "rb").read()
    return [(p[i * 3], p[i * 3 + 1], p[i * 3 + 2]) for i in range(256)]


def sub_to_png(d, offs, idx, prgb, out_path):
    o = offs[idx]
    w, h = struct.unpack_from("<HH", d, o)
    end = offs[idx + 1] if idx + 1 < len(offs) else len(d)
    px = decode_pixels(d[o + 4:end], w * h)
    img = Image.new("RGBA", (w, h))
    pix = img.load()
    for y in range(h):
        for x in range(w):
            ci = px[y * w + x]
            pix[x, y] = (0, 0, 0, 0) if ci == 0 else (*prgb[ci], 255)
    img.save(out_path)
    return w, h


def main(argv):
    if len(argv) < 4:
        print(__doc__)
        return 1
    d = open(argv[1], "rb").read()
    prgb = load_palette(argv[2])
    outdir = argv[3]
    offs = lmi_offsets(d)
    os.makedirs(outdir, exist_ok=True)
    if len(argv) == 4:
        for i, o in enumerate(offs):
            w, h = struct.unpack_from("<HH", d, o)
            print(f"#{i} @{hex(o)} {w}x{h}")
        return 0
    for a in argv[4:]:
        idx = int(a)
        w, h = sub_to_png(d, offs, idx, prgb, os.path.join(outdir, f"lmi_{idx:03d}.png"))
        print(f"#{idx} -> {w}x{h}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
