#!/usr/bin/env python3
"""炎龍騎士團2 — .DAT 圖像解碼器(第 2 輪)。

圖像格式(已驗證):
    +0  uint16 LE  width
    +2  uint16 LE  height
    +4  pixel data,8-bit VGA 調色盤索引(mode 13h)

像素資料有兩種:
  (a) 未壓縮:body == W*H(如 FDOTHER_015 / FDOTHER_055)。
  (b) 壓縮:body < W*H(RLE,演算法見 decode_rle;第 2 輪持續驗證中)。

調色盤:預設取 FDOTHER 容器資源 #0(768B = 256×RGB,6-bit 0..63 → ×4 轉 8-bit)。
不同畫面可能切換調色盤,後輪再對應。

用法:
    python3 decode_image.py <resource.bin> <palette.bin> <out.png>
    python3 decode_image.py --batch <extracted根> <palette.bin> <out目錄>
"""
import sys
import os
import struct
import glob
from PIL import Image


def load_palette(path):
    raw = open(path, "rb").read()[:768]
    pal = []
    for i in range(256):
        r, g, b = raw[i * 3], raw[i * 3 + 1], raw[i * 3 + 2]
        # 6-bit VGA → 8-bit
        pal.extend([(r << 2) | (r >> 4), (g << 2) | (g >> 4), (b << 2) | (b >> 4)])
    return pal


def decode_rle(body, target):
    """RLE 解壓(第 2 輪破解):
        讀 byte c
          c >= 0x80 : 文字串(literal),接下來 (c&0x7f)+1 個 byte 原樣輸出
          c <  0x80 : 連續執行(run),下一個 byte 重複 (c+1) 次
    成功(輸出剛好 target)回傳 bytes,否則 None。"""
    out = bytearray()
    i = 0
    n = len(body)
    while i < n and len(out) < target:
        c = body[i]
        i += 1
        if c >= 0x80:
            cnt = (c & 0x7f) + 1
            out += body[i:i + cnt]
            i += cnt
        else:
            if i >= n:
                break
            out += bytes([body[i]]) * (c + 1)
            i += 1
    return bytes(out) if len(out) == target else None


def decode_image(path):
    """回傳 (w, h, indices_bytes_or_None, mode)。"""
    d = open(path, "rb").read()
    if len(d) < 6:
        return None
    w, h = struct.unpack_from("<HH", d, 0)
    body = d[4:]
    if not (0 < w <= 1024 and 0 < h <= 1024):
        return None
    if len(body) == w * h:
        return (w, h, body, "raw")
    px = decode_rle(body, w * h)
    if px is not None:
        return (w, h, px, "rle")
    return (w, h, None, "compressed")


def save_png(w, h, idx, palette, out):
    im = Image.frombytes("P", (w, h), bytes(idx))
    im.putpalette(palette)
    im.convert("RGB").save(out)


def main(argv):
    if len(argv) < 2:
        print(__doc__); return 1
    if argv[1] == "--batch":
        root, palp, outdir = argv[2], argv[3], argv[4]
        pal = load_palette(palp)
        os.makedirs(outdir, exist_ok=True)
        n_raw = n_skip = 0
        for f in sorted(glob.glob(os.path.join(root, "*", "*.bin"))):
            r = decode_image(f)
            if r and r[2] is not None:
                w, h, idx, mode = r
                name = os.path.splitext(os.path.relpath(f, root).replace(os.sep, "_"))[0]
                save_png(w, h, idx, pal, os.path.join(outdir, f"{name}.png"))
                n_raw += 1
            else:
                n_skip += 1
        print(f"輸出 {n_raw} 張(未壓縮/可解),略過 {n_skip}(壓縮待解或非圖)")
        return 0
    src, palp, out = argv[1], argv[2], argv[3]
    pal = load_palette(palp)
    r = decode_image(src)
    if not r:
        print("非圖像格式"); return 1
    w, h, idx, mode = r
    if idx is None:
        print(f"{w}x{h} 壓縮({mode}),解壓未實作"); return 2
    save_png(w, h, idx, pal, out)
    print(f"{w}x{h} {mode} -> {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
