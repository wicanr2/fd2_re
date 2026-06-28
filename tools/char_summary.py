#!/usr/bin/env python3
"""炎龍騎士團2 — 角色總覽 summary:每組 FDICON 地圖 sprite + DATO face 頭像 + 組號。

驗證並一覽「統一角色編號」(doc 31):角色 id N → FDICON 組 N(地圖 sprite,icon_N*12)
+ DATO_N(對話頭像,4 嘴型,取 m0)。140 組排成一張,方便挑角色 / 對 face / 加新人時看缺號。

前置:`extracted/fdicon/`(decode_fdicon.py)、`extracted/portraits/`(decode_dato.py)。
輸出本機 PNG(著作權,不入庫)。

用法:
  python3 char_summary.py <out.png> [groups=140] [cols=10]
"""
import sys, os
from PIL import Image, ImageDraw

# 註:NAMES 標在「組號」格。恆等角色 組=portrait;龍人/轉職系跳號(凱拉斯 face=DATO67,sprite=組49)。
NAMES = {0: "索爾", 1: "哈諾", 2: "鐵諾", 3: "哈瓦特", 4: "亞雷斯", 5: "洛娜", 6: "萊汀",
         7: "蘭斯洛特", 8: "莎拉", 9: "悠妮", 49: "凱拉斯sprite", 68: "士兵", 76: "士兵(援軍)",
         96: "盜賊", 97: "盜賊頭目", 103: "獸人"}


def main(argv):
    out = argv[1] if len(argv) > 1 else "extracted/remake_shots/character_summary.png"
    ngrp = int(argv[2]) if len(argv) > 2 else 140
    cols = int(argv[3]) if len(argv) > 3 else 10
    rows = (ngrp + cols - 1) // cols
    sp, fc = 44, 40
    cw, ch = sp + fc + 6, sp + 14
    sheet = Image.new("RGB", (cols * cw, rows * ch), (25, 25, 32))
    dr = ImageDraw.Draw(sheet)
    for g in range(ngrp):
        cx, cy = (g % cols) * cw, (g // cols) * ch
        spp = f"extracted/fdicon/icon_{g*12:04d}.png"   # 組正面站幀 = 組×12
        if os.path.exists(spp):
            im = Image.open(spp).convert("RGBA").resize((sp, sp), Image.NEAREST)
            sheet.paste(im, (cx, cy + 12), im)
        fp = f"extracted/portraits/DATO_{g:03d}_m0.png"
        if os.path.exists(fp):
            fm = Image.open(fp).convert("RGBA").resize((fc, fc), Image.NEAREST)
            sheet.paste(fm, (cx + sp + 4, cy + 12), fm)
        lbl = str(g) + (" " + NAMES[g] if g in NAMES else "")
        dr.text((cx + 1, cy + 1), lbl, fill=(255, 235, 120))
    os.makedirs(os.path.dirname(out), exist_ok=True)
    sheet.save(out)
    print(f"角色總覽 {ngrp} 組(sprite+face)-> {out} {sheet.size}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
