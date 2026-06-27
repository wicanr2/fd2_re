#!/usr/bin/env python3
"""炎龍騎士團2 — 一鍵全素材抽取(把原版資產解成現代格式,分類整齊輸出)。

把原版(玩家自備)的 FLAME2/ 解成:
    extracted/
      raw/<CONTAINER>/        .DAT 容器解包的原始 sub-resource
      images/                 標題 / 戰鬥背景等全幅圖 (PNG)
      animations/<RES>/*.png  FIGANI 戰鬥動畫逐幀 (PNG)
      animations/<RES>.gif    每個動畫的 GIF
      music/*.mid             XMIDI → 標準 MIDI
      fonts/atlas.png         自製 16×16 中文字型字模表
      exe_tables/*.json       FD2.EXE 內數值表
      INDEX.md                本次抽取清單

⚠ 輸出全為遊戲著作權內容(漢堂國際),僅供本機研究 / 重製,不散布、不入版控。

用法:
    python3 tools/extract_all.py <FLAME2目錄> <輸出目錄> [--anim-limit N]
"""
import sys
import os
import glob
import struct

sys.path.insert(0, os.path.dirname(__file__))
import unpack_dat
import decode_image
import decode_figani
import decode_text
import xmi2mid


def main(argv):
    if len(argv) < 3:
        print(__doc__); return 1
    G = argv[1]
    OUT = argv[2]
    anim_limit = None
    if "--anim-limit" in argv:
        anim_limit = int(argv[argv.index("--anim-limit") + 1])

    os.makedirs(OUT, exist_ok=True)
    raw = os.path.join(OUT, "raw")
    log = []

    # 1. 解包所有 .DAT 容器
    os.makedirs(raw, exist_ok=True)
    nres = 0
    for f in sorted(glob.glob(os.path.join(G, "*.DAT"))):
        try:
            nres += unpack_dat.unpack(f, raw)
        except unpack_dat.NotAContainer:
            pass
    log.append(f"raw/ : 解包 .DAT 容器,共 {nres} 個 sub-resource")

    palp = os.path.join(raw, "FDOTHER", "FDOTHER_000.bin")
    if not os.path.exists(palp):
        print("找不到調色盤 FDOTHER_000,中止"); return 2
    pal = decode_image.load_palette(palp)

    # 2. 圖像(全幅:title/bg/fdother 未壓縮與 RLE)
    imgdir = os.path.join(OUT, "images")
    os.makedirs(imgdir, exist_ok=True)
    nimg = 0
    for f in sorted(glob.glob(os.path.join(raw, "*", "*.bin"))):
        r = decode_image.decode_image(f)
        if r and r[2] is not None:
            w, h, idx, mode = r
            name = os.path.splitext(os.path.relpath(f, raw).replace(os.sep, "_"))[0]
            decode_image.save_png(w, h, idx, pal, os.path.join(imgdir, f"{name}.png"))
            nimg += 1
    log.append(f"images/ : {nimg} 張全幅圖(未壓縮 + RLE 背景/標題)")

    # 3. FIGANI 動畫逐幀 + GIF
    animdir = os.path.join(OUT, "animations")
    os.makedirs(animdir, exist_ok=True)
    nanim = nframe = 0
    figani = sorted(glob.glob(os.path.join(raw, "FIGANI", "*.bin")))
    if anim_limit:
        figani = figani[:anim_limit]
    for f in figani:
        d = open(f, "rb").read()
        if len(d) < 12:
            continue
        try:
            frames = decode_figani.parse_anim(d)
        except Exception:
            continue
        if not frames:
            continue
        base = os.path.splitext(os.path.basename(f))[0]
        decode_figani.cmd_frames(f, palp, os.path.join(animdir, base))
        decode_figani.cmd_gif(f, palp, os.path.join(animdir, base + ".gif"))
        nanim += 1
        nframe += len(frames)
    log.append(f"animations/ : {nanim} 個動畫,共 {nframe} 幀(PNG 序列 + GIF)")

    # 4. 音樂 XMIDI → MIDI
    musdir = os.path.join(OUT, "music")
    os.makedirs(musdir, exist_ok=True)
    nmid = 0
    for f in sorted(glob.glob(os.path.join(raw, "FDMUS", "*.bin"))):
        d = open(f, "rb").read()
        if d[:4] != b"FORM":
            continue
        base = os.path.splitext(os.path.basename(f))[0]
        r = xmi2mid.convert(d, os.path.join(musdir, base + ".mid"), verbose=False)
        nmid += len(r)
    log.append(f"music/ : {nmid} 首 MIDI(XMIDI 轉出)")

    # 5. 字型字模表
    fontdir = os.path.join(OUT, "fonts")
    os.makedirs(fontdir, exist_ok=True)
    fontres = os.path.join(raw, "FDOTHER", "FDOTHER_004.bin")
    if os.path.exists(fontres):
        decode_text.font_atlas(fontres, os.path.join(fontdir, "atlas.png"))
        log.append("fonts/ : atlas.png(1824 字模 16×16 自製字型)")

    # 6. INDEX
    with open(os.path.join(OUT, "INDEX.md"), "w", encoding="utf-8") as fp:
        fp.write("# 炎龍騎士團2 — 抽取素材清單\n\n")
        fp.write("> 全為漢堂國際遊戲著作權內容,僅本機研究/重製用,不散布。\n\n")
        for line in log:
            fp.write(f"- {line}\n")
    print("\n".join(log))
    print(f"\n完成 → {OUT}/INDEX.md")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
