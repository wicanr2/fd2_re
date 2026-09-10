#!/usr/bin/env python3
"""在原版索引畫面上用分離素材的 LMI1 字模做模板匹配。

用途：回答「原版畫面上這塊字樣是不是就是我們手上那格素材，畫在哪裡」。字模的
非透明像素（索引 0 是透明）必須逐一吻合，所以命中通常是唯一的——這比目視對位
可靠，也不需要調色盤正確。

只比對落在 `sub_11EB0` 搬運窗格 (4,4)-(315,195) 內的像素。原版把 UI 畫進離屏
緩衝區再把 312×192 搬到 VGA 的 (4,4)，窗格外那四欄／四列留著先前的內容；不排除
它們的話，任何跨出邊界的字樣都會有幾十個像素「不吻合」，看起來像素材抓錯。

用法：
  python3 tools/match_lmi1_glyph.py <原版幀.png> [更多幀…]

收據：docs/data/ui-traces/fd2-phase-banner-glyph-parity-20260910.json
"""
import struct, sys, zlib
from pathlib import Path

def readpng(p):
    d = Path(p).read_bytes(); i = 8; idat = b""; w = h = ct = None
    while i < len(d):
        ln = struct.unpack(">I", d[i:i+4])[0]; typ = d[i+4:i+8]
        data = d[i+8:i+8+ln]; i += 12+ln
        if typ == b"IHDR": w, h, _, ct = struct.unpack(">IIBB", data[:10])
        elif typ == b"IDAT": idat += data
    assert ct == 3, f"{p} 不是索引色 PNG (colortype={ct})"
    raw = zlib.decompress(idat); rows = []; prev = bytearray(w); pos = 0
    for _ in range(h):
        ft = raw[pos]; pos += 1; line = bytearray(raw[pos:pos+w]); pos += w
        for x in range(w):
            a = line[x-1] if x >= 1 else 0; b = prev[x]; c = prev[x-1] if x >= 1 else 0
            if ft == 1: line[x] = (line[x]+a) & 255
            elif ft == 2: line[x] = (line[x]+b) & 255
            elif ft == 3: line[x] = (line[x]+(a+b)//2) & 255
            elif ft == 4:
                pa, pb, pc = abs(b-c), abs(a-c), abs(a+b-2*c)
                pr = a if pa <= pb and pa <= pc else (b if pb <= pc else c)
                line[x] = (line[x]+pr) & 255
        rows.append(bytes(line)); prev = line
    return w, h, rows

# sub_11EB0 搬到 VGA 的矩形（含端點）。
VIEWPORT = (4, 4, 315, 195)


def match(frame, glyph, gw, gh):
    """回傳字模所有非零像素完全吻合的落點。"""
    fw, fh = len(frame[0]), len(frame)
    x0, y0, x1, y1 = VIEWPORT
    solid = [(x, y, glyph[y][x]) for y in range(gh) for x in range(gw) if glyph[y][x]]
    hits = []
    for oy in range(-gh+1, fh):
        for ox in range(-gw+1, fw):
            inside = 0
            ok = True
            for gx, gy, v in solid:
                px, py = ox+gx, oy+gy
                if not (x0 <= px <= x1 and y0 <= py <= y1):
                    continue           # 窗格外那幾欄不由這次搬運更新
                inside += 1
                if frame[py][px] != v:
                    ok = False; break
            # 幾乎整塊都在窗格外時「全部吻合」是空話，要求過半像素真的比對過。
            if ok and inside * 2 >= len(solid):
                hits.append((ox, oy))
    return hits

base = Path("remake/generated-assets/fd2-original-b97caf22/ui/fdother_005_lmi1_opaque")
glyphs = {}
for name, entry in (("ENEMY", "entry_082"), ("PHASE", "entry_081"), ("PLAYER", "entry_080")):
    gw, gh, grows = readpng(base/entry/"frame.png")
    glyphs[name] = (grows, gw, gh)

for path in sys.argv[1:]:
    fw, fh, frame = readpng(path)
    print(f"== {Path(path).name}")
    for name in ("ENEMY", "PHASE"):
        grows, gw, gh = glyphs[name]
        hits = match(frame, grows, gw, gh)
        print(f"   {name:6s} {gw}x{gh} 落點 {hits if len(hits) <= 4 else str(len(hits))+' 個'}")
