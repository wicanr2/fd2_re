#!/usr/bin/env python3
"""炎龍騎士團2 — sprite RLE 解碼器(第 3 輪,反組譯還原)。

從 FD2.EXE `0x4EB52`(24×24 sprite 解碼器)反組譯出的 4 模式 RLE 文法。
控制 byte c:高 2 bit = 模式,低 6 bit → count = (c & 0x3F) + 1:

    00xxxxxx  色彩 run     讀 1 像素,重複 count 次
    01xxxxxx  dither/scaled 讀 1 像素,隔位寫,佔 2×count 寬(陰影用)
    10xxxxxx  literal      讀 count 個像素原樣
    11xxxxxx  透明 skip     跳過 count 像素(留底 = 透明)

像素可經 remap 表轉換(原版 [ebp+eax]);本工具預設 identity。

⚠ 狀態:此文法對應 EXE 內 **24×24** 解碼器。FIGANI 戰鬥動畫(任意寬)使用
同家族的**另一參數化變體**(模式/位元配置略不同),套用本文法可精確消耗位元組但
渲染未對齊 → 待反組譯 0x4E000–0x4F800 對應變體後修正。詳見
docs/knowledge-base/06-animation-format.md。

用法:
    python3 decode_sprite.py <frame.bin> <w> <h> <palette.bin> <out.png>
"""
import sys
import struct
from PIL import Image


def decode_rle_sprite(body, width, height, trans=0, mode01="literal"):
    """4 模式 sprite RLE → 像素 bytes(transparent=trans)。"""
    out = bytearray()
    total = width * height
    i = 0
    n = len(body)
    while len(out) < total and i < n:
        c = body[i]; i += 1
        mode = c >> 6
        cnt = (c & 0x3F) + 1
        if mode == 0:                      # 色彩 run
            v = body[i]; i += 1
            out += bytes([v]) * cnt
        elif mode == 3:                    # 透明 skip
            out += bytes([trans]) * cnt
        elif mode == 2:                    # literal
            out += body[i:i + cnt]; i += cnt
        else:                              # mode 01
            if mode01 == "literal":
                out += body[i:i + cnt]; i += cnt
            else:                          # dither
                v = body[i]; i += 1
                out += bytes([trans, v]) * cnt
    return bytes(out[:total]).ljust(total, bytes([trans]))


def load_palette(path):
    raw = open(path, "rb").read()[:768]
    pal = []
    for i in range(256):
        r, g, b = raw[i*3], raw[i*3+1], raw[i*3+2]
        pal += [(r << 2) | (r >> 4), (g << 2) | (g >> 4), (b << 2) | (b >> 4)]
    return pal


def main(argv):
    if len(argv) < 6:
        print(__doc__); return 1
    body = open(argv[1], "rb").read()
    w, h = int(argv[2]), int(argv[3])
    pal = load_palette(argv[4])
    px = decode_rle_sprite(body, w, h)
    im = Image.frombytes("P", (w, h), px)
    im.putpalette(pal)
    im.convert("RGB").save(argv[5])
    print(f"{w}x{h} -> {argv[5]}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
