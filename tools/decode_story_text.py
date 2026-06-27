#!/usr/bin/env python3
"""用 glyph_map.json 把 FDTXT 章節解成 UTF-8 文字(含說話者)。

對話結構:[控制碼 0xFFxx][說話者肖像ID][『=557][對白…][』=560],0xFFFF 結束。
說話者肖像 ID → 角色名(memory.md);>0x1F 的 NPC/敵以字模顯示。
段內控制碼視為換行(同一說話者續行)。

用法:
    python3 decode_story_text.py <FDTXT_NNN.bin>            # 印單章
    python3 decode_story_text.py --all <FDTXT目錄> <out.md>  # 全章合一檔
"""
import sys
import os
import json
import glob

OPEN, CLOSE, END = 557, 560, 0xFFFF
PORT = {0: "索爾", 1: "哈諾", 2: "鐵諾", 3: "哈瓦特", 4: "亞雷斯", 5: "洛娜",
        6: "萊汀", 7: "蘭斯洛特", 8: "希莉亞", 9: "悠妮", 0xA: "瑪琳", 0xB: "索菲亞",
        0xC: "凱麗", 0xD: "貝克威", 0xE: "珊", 0xF: "賽可邦勒", 0x10: "凱拉斯",
        0x11: "米亞斯多德", 0x12: "蜜蒂", 0x13: "羅德曼", 0x14: "莎拉", 0x15: "約拿",
        0x16: "卡里斯", 0x17: "羅蘭", 0x18: "希爾法", 0x19: "謝多", 0x1A: "聖寇拉斯",
        0x1B: "巴拿羅西亞", 0x1C: "達克賽", 0x1D: "亞奇梅吉", 0x1E: "蓋亞", 0x1F: "渥德"}

sys.path.insert(0, os.path.dirname(__file__))
from decode_text import parse_strings

_GM = None
def gm():
    global _GM
    if _GM is None:
        d = os.path.join(os.path.dirname(__file__), "..", "docs", "data", "glyph_map.json")
        m = json.load(open(d, encoding="utf-8"))
        _GM = {int(k): v for k, v in m.items() if k != "_comment"}
    return _GM


def g2s(codes):
    m = gm()
    return "".join(m.get(c, f"〈{c}〉") for c in codes)


def decode_string(codes):
    """回傳 list of (speaker_or_None, text)。"""
    if END in codes:
        codes = codes[:codes.index(END)]
    segs = []
    cur = []
    for c in codes:
        if 0xFF00 <= c <= 0xFFFE:
            segs.append(cur); cur = []
        else:
            cur.append(c)
    segs.append(cur)
    out = []
    for seg in segs:
        if not seg:
            continue
        if len(seg) >= 2 and seg[1] == OPEN:
            spk = seg[0]
            name = PORT.get(spk, g2s([spk]))
            body = [c for c in seg[2:] if c not in (OPEN, CLOSE)]
            out.append((name, g2s(body)))
        else:
            body = [c for c in seg if c not in (OPEN, CLOSE)]
            out.append((None, g2s(body)))
    return out


def render_chapter(path):
    lines = []
    for codes in parse_strings(path):
        for spk, text in decode_string(codes):
            if not text.strip():
                continue
            if spk:
                lines.append(f"- **{spk}**：{text}")
            else:
                lines.append(f"  {text}")
    return lines


def main(argv):
    if len(argv) < 2:
        print(__doc__); return 1
    if argv[1] == "--all":
        src, out = argv[2], argv[3]
        with open(out, "w", encoding="utf-8") as f:
            f.write("# 炎龍騎士團2 — 全劇情自動解碼\n\n")
            f.write("> 由 FDTXT.DAT + glyph_map.json 自動解碼。遊戲著作權內容,僅本機對照用,不散布。\n\n")
            for p in sorted(glob.glob(os.path.join(src, "*.bin"))):
                base = os.path.splitext(os.path.basename(p))[0]
                ls = render_chapter(p)
                if not ls:
                    continue
                f.write(f"\n## {base}\n\n")
                f.write("\n".join(ls) + "\n")
        print(f"全章 -> {out}")
        return 0
    for ln in render_chapter(argv[1]):
        print(ln)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
