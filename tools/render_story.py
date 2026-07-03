#!/usr/bin/env python3
"""炎龍騎士團2 — 劇情 / 對話渲染器(把 FDTXT 章節畫成可讀 PNG)。

文本結構(第 3 輪解出):每個 FDTXT 資源 = 一章。資源內每個字串可含多段對白,
段落格式:[對話控制碼 0xFFxx][說話者肖像 ID][『][對白 glyph...][』]。

- 說話者肖像 ID → 角色名(memory.md 肖像表;>0x41 為 NPC/敵,顯示原 glyph)。
- 控制碼 0xFFxx:視為換行(段落分隔)。
- glyph 索引 < 0x720 → 用自製字型(FDOTHER 資源 #4)渲染。

輸出 = 分頁 PNG(本機對照用;劇本屬遊戲著作權,不入版控)。

用法:
    python3 render_story.py <FDTXT_NNN.bin> <FDOTHER_004.bin> <out前綴>
    python3 render_story.py --all <raw/FDTXT目錄> <FDOTHER_004.bin> <out目錄>
"""
import sys
import os
import glob
import struct
from PIL import Image

sys.path.insert(0, os.path.dirname(__file__))
from decode_text import parse_strings, load_font, render_glyph

# 肖像 ID → 角色名(memory.md)
PORTRAIT = {
    0x00: "索爾", 0x01: "哈諾", 0x02: "鐵諾", 0x03: "哈瓦特", 0x04: "亞雷斯",
    0x05: "洛娜", 0x06: "萊汀", 0x07: "蘭斯洛特", 0x08: "希莉亞", 0x09: "悠妮",
    0x0A: "瑪琳", 0x0B: "索菲亞", 0x0C: "凱麗", 0x0D: "貝克威", 0x0E: "珊",
    0x0F: "賽可邦勒", 0x10: "凱拉斯", 0x11: "米亞斯多德", 0x12: "蜜蒂", 0x13: "羅德曼",
    0x14: "莎拉", 0x15: "約拿", 0x16: "卡里斯", 0x17: "羅蘭", 0x18: "希爾法",
    0x19: "謝多", 0x1A: "聖寇拉斯", 0x1B: "巴拿羅西亞", 0x1C: "達克賽", 0x1D: "亞奇梅吉",
    0x1E: "蓋亞", 0x1F: "渥德",
}

CELL = 18
COLS = 28
LINES_PER_PAGE = 42
TR = 0  # 背景


def lay_out(strings):
    """把整章拆成 render rows。每 row = list of ('g',code) glyph;'sep' 為段間距。"""
    rows = []
    for s in strings:
        line = []
        n = len(s)
        i = 0
        while i < n:
            c = s[i]
            if c >= 0xFF00:        # 對話控制碼 → 換行
                if line:
                    rows.append(line); line = []
                # -17..-20(0xFFEC-0xFFEF)後接一個 operand word=說話者 id/idx。
                # 原版渲染器 0x15F84 把它當二進位參數消耗、從不畫出(見 doc40)。
                # 早期漏跳→id 洩漏成字模,被誤認成「說話者字母代號」,務必跳過。
                if 0xFFEC <= c <= 0xFFEF and i + 1 < n:
                    i += 1
            else:
                line.append(c)
                if len(line) >= COLS:
                    rows.append(line); line = []
            i += 1
        if line:
            rows.append(line)
        rows.append("sep")
    return rows


def render_chapter(txt_path, font, out_prefix):
    strings = parse_strings(txt_path)
    rows = lay_out(strings)
    # 分頁
    pages = []
    buf = []
    cnt = 0
    for r in rows:
        buf.append(r)
        if r != "sep":
            cnt += 1
        if cnt >= LINES_PER_PAGE and r == "sep":
            pages.append(buf); buf = []; cnt = 0
    if buf:
        pages.append(buf)
    for pi, pg in enumerate(pages):
        h = sum(CELL if r != "sep" else 6 for r in pg) + 8
        im = Image.new("L", (COLS * CELL + 12, h), 18)
        y = 4
        for r in pg:
            if r == "sep":
                y += 6; continue
            x = 4
            for c in r:
                im.paste(render_glyph(font, c), (x, y)); x += CELL
            y += CELL
        im.save(f"{out_prefix}_p{pi}.png")
    return len(pages), len(strings)


def main(argv):
    if len(argv) < 4:
        print(__doc__); return 1
    if argv[1] == "--all":
        src, fontp, outdir = argv[2], argv[3], argv[4]
        font, _ = load_font(fontp)
        os.makedirs(outdir, exist_ok=True)
        for f in sorted(glob.glob(os.path.join(src, "*.bin"))):
            base = os.path.splitext(os.path.basename(f))[0]
            np_, ns = render_chapter(f, font, os.path.join(outdir, base))
            print(f"{base}: {ns} 字串 -> {np_} 頁")
        return 0
    font, _ = load_font(argv[2])
    np_, ns = render_chapter(argv[1], font, argv[3])
    print(f"{ns} 字串 -> {np_} 頁")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
