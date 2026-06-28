#!/usr/bin/env python3
"""炎龍騎士團2 — 調色 remap LUT 解析與套用(陣營 / 狀態著色)。

`FDOTHER.DAT` 資源 #3 = `"LMI1"` 容器,內含 23 張 256-byte remap LUT。
場景單位(24×24)繪製時(EXE `0x4EB52`)以 `LUT[原始像素索引]` 重新著色,
做出「已行動變灰、敵我染色、夜戰色調」等效果(見 10-sprite-rendering-camp-and-state)。

LMI1 格式: magic "LMI1"(4) + uint16 count + uint32[count] offset(相對檔頭) + 各 256-byte LUT。

用法:
    python3 dump_remap.py list  <FDOTHER_003.bin>
    python3 dump_remap.py apply <FDOTHER_003.bin> <palette.bin> <sprite像素.raw> <W> <H> <out目錄>
"""
import sys
import os
import struct
from PIL import Image


def parse_lmi(path):
    d = open(path, "rb").read()
    if d[:4] != b"LMI1":
        raise ValueError("非 LMI1 容器")
    n = struct.unpack_from("<H", d, 4)[0]
    offs = [struct.unpack_from("<I", d, 6 + 4 * i)[0] for i in range(n)]
    luts = []
    for i in range(n):
        s = offs[i]
        e = offs[i + 1] if i + 1 < n else len(d)
        luts.append(d[s:e][:256])
    return luts


def load_palette(path):
    raw = open(path, "rb").read()[:768]
    pal = []
    for i in range(256):
        r, g, b = raw[i*3], raw[i*3+1], raw[i*3+2]
        pal += [(r << 2) | (r >> 4), (g << 2) | (g >> 4), (b << 2) | (b >> 4)]
    return pal


def main(argv):
    if len(argv) < 3:
        print(__doc__); return 1
    if argv[1] == "list":
        luts = parse_lmi(argv[2])
        print(f"{len(luts)} 張 LUT(256-byte each)")
        for i, l in enumerate(luts):
            diff = sum(1 for k in range(min(256, len(l))) if l[k] != k)
            print(f"  LUT{i:2}: 非 identity 項={diff}/256  樣本[64:72]={list(l[64:72])}")
        return 0
    if argv[1] == "apply":
        lmi, palp, rawp, w, h, out = argv[2], argv[3], argv[4], int(argv[5]), int(argv[6]), argv[7]
        luts = parse_lmi(lmi); pal = load_palette(palp)
        base = open(rawp, "rb").read()[:w*h]
        os.makedirs(out, exist_ok=True)
        for i, lut in enumerate(luts):
            if len(lut) < 256:
                continue
            px = bytes(lut[p] for p in base)
            im = Image.frombytes("P", (w, h), px); im.putpalette(pal)
            im.convert("RGB").save(os.path.join(out, f"lut{i:02d}.png"))
        print(f"{len(luts)} 張著色結果 -> {out}")
        return 0
    print(__doc__); return 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
