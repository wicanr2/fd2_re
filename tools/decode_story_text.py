#!/usr/bin/env python3
"""用 glyph_map.json 把 FDTXT 章節解成 UTF-8 文字(含場景相依說話者定位)。

對話結構:[控制碼 0xFFxx][說話者肖像ID][『=557][對白…][』=560],0xFFFF 結束。
說話者 operand 不是全域角色 ID。0xFFEF/0xFFEE 後是身分標籤，
0xFFED/0xFFEC 後是目前場景單位 index；靜態工具無場景單位表，不猜人名。
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
SPEAKER_CONTROL = {
    0xFFEF: "身分標籤",
    0xFFEE: "身分標籤",
    0xFFED: "單位索引",
    0xFFEC: "單位索引",
}

sys.path.insert(0, os.path.dirname(__file__))
from decode_text import parse_strings

_GM = None
def gm():
    global _GM
    if _GM is None:
        d = os.path.join(os.path.dirname(__file__), "..", "docs", "data", "glyph_map.json")
        with open(d, encoding="utf-8") as handle:
            m = json.load(handle)
        _GM = {int(k): v for k, v in m.items() if k != "_comment"}
    return _GM


def g2s(codes):
    m = gm()
    return "".join(m.get(c, f"〈{c}〉") for c in codes)


def decode_string(codes):
    """回傳 list of (speaker_or_None, text)。"""
    if END in codes:
        codes = codes[:codes.index(END)]
    out = []
    i = 0
    while i < len(codes):
        control = codes[i] if codes[i] in SPEAKER_CONTROL else None
        if control is not None and i + 2 < len(codes) and codes[i + 2] == OPEN:
            operand = codes[i + 1]
            i += 3
            body = []
            while i < len(codes) and not (0xFF00 <= codes[i] <= 0xFFFE):
                if codes[i] not in (OPEN, CLOSE):
                    body.append(codes[i])
                i += 1
            out.append((f"{SPEAKER_CONTROL[control]} {operand}", g2s(body)))
            continue

        i += 1
        body = []
        while i < len(codes) and not (0xFF00 <= codes[i] <= 0xFFFE):
            if codes[i] not in (OPEN, CLOSE):
                body.append(codes[i])
            i += 1
        if body:
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
