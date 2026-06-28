#!/usr/bin/env python3
"""炎龍騎士團2 — DATO.DAT 人物頭像解碼器(第 6 輪)。

DATO.DAT(LLLLLL 容器,137 資源)= 對話頭像。每資源結構:
    +0  uint32[4]  4 個子圖 offset(= 講話嘴型 4 幀)
    各子圖:  uint16 W, uint16 H(多為 80×80), 然後 RLE 像素

RLE codec(反組譯自 0x4F716,與 sprite/背景不同的簡式):
    讀 byte b:
      b <= 0xC0 : 字面像素(值 = b)
      b >  0xC0 : run,重複 (b - 0xC0) 次下一個 byte
無透明(頭像為矩形實心)。調色盤:FDOTHER 資源 #0。
頭像在對話框由 0xFFEF 控制碼依說話者肖像 ID 載入(見 14-text-control-codes)。

用法:
    python3 decode_dato.py frames <DATO_NNN.bin> <palette.bin> <out目錄>
    python3 decode_dato.py --batch <raw/DATO目錄> <palette.bin> <out目錄>
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
        r, g, b = raw[i*3], raw[i*3+1], raw[i*3+2]
        pal += [(r << 2) | (r >> 4), (g << 2) | (g >> 4), (b << 2) | (b >> 4)]
    return pal


def rle(body, total):
    out = bytearray()
    i = 0
    n = len(body)
    while len(out) < total and i < n:
        b = body[i]; i += 1
        if b <= 0xC0:
            out.append(b)
        else:
            v = body[i]; i += 1
            out += bytes([v]) * (b - 0xC0)
    return bytes(out[:total]).ljust(total, b"\0")


def frames(path):
    """回傳 [(w,h,pixels), ...](通常 4 幀)。"""
    d = open(path, "rb").read()
    if len(d) < 16:
        return []
    offs = [struct.unpack_from("<I", d, 4 * i)[0] for i in range(4)]
    if offs[0] != 0x10:
        return []
    out = []
    for k in range(4):
        o = offs[k]
        end = offs[k + 1] if k + 1 < 4 else len(d)
        if o + 4 > len(d):
            continue
        w, h = struct.unpack_from("<HH", d, o)
        if not (0 < w <= 256 and 0 < h <= 256):
            continue
        out.append((w, h, rle(d[o + 4:end], w * h)))
    return out


def save(path, palp, outdir):
    pal = load_palette(palp)
    os.makedirs(outdir, exist_ok=True)
    base = os.path.splitext(os.path.basename(path))[0]
    for k, (w, h, px) in enumerate(frames(path)):
        im = Image.frombytes("P", (w, h), px)
        im.putpalette(pal)
        im.convert("RGB").save(os.path.join(outdir, f"{base}_m{k}.png"))


def main(argv):
    if len(argv) < 4:
        print(__doc__); return 1
    if argv[1] == "--batch":
        src, palp, out = argv[2], argv[3], argv[4]
        n = 0
        for f in sorted(glob.glob(os.path.join(src, "*.bin"))):
            if frames(f):
                save(f, palp, out); n += 1
        print(f"{n} 個頭像 -> {out}")
        return 0
    save(argv[2], argv[3], argv[4])
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
