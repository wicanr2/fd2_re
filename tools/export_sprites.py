#!/usr/bin/env python3
"""炎龍騎士團2 — 從 FDICON.B24 導出地圖單位 Q 版 sprite(待機分鏡)給 remake。

FDICON.B24 = 1680 個 24×24 Q 版小人(原版戰場地圖單位,非 FIGANI 戰鬥全身)。
每角色一「組」= 12 sprite:**4 方向 × 3 幀**(站 / 抬左手 / 抬右手 → 待機手擺動感)。
方向順序(實測組 0):0-2 下、3-5 左、6-8 上、9-11 右。

本工具對指定角色組,導出「面向下」的 3 待機幀(透明 PNG),供地圖待機循環。
輸出 <out>/fig_<grp>_f<0..2>.png(24×24 RGBA)。資產屬著作權,只在本機,不入庫。

用法:
  python3 export_sprites.py <out_dir> <grp[,grp...]> [palette.bin]
"""
import sys, os
sys.path.insert(0, os.path.dirname(__file__))
from decode_fdicon import load, tile_img
from decode_image import load_palette

ICON = "org_game/炎龍騎士團/FLAME2/FDICON.B24"
DEFAULT_PAL = "extracted/raw/FDOTHER/FDOTHER_000.bin"
FRAMES = range(12)  # 全 12 幀:4 方向(下/左/上/右)× 3(站/抬左/抬右),供走動分鏡


def main(argv):
    if len(argv) < 3:
        print(__doc__); return 1
    out = argv[1]
    grps = [int(x) for x in argv[2].split(",") if x.strip()]
    palp = argv[3] if len(argv) > 3 else DEFAULT_PAL
    os.makedirs(out, exist_ok=True)
    d, tw, th, cnt, offs = load(ICON)
    pal = load_palette(palp)
    total = 0
    for grp in grps:
        base = grp * 12
        for k in FRAMES:  # 導全 12 幀(fig_<grp>_f00..f11 = dir*3+frame),drawUnitSprite 按 Dir 取
            idx = base + k
            if idx >= cnt:
                continue
            tile_img(d, tw, th, offs, idx, cnt, pal).save(
                os.path.join(out, f"fig_{grp:03d}_f{k:02d}.png"))
            total += 1
    print(f"導出 {total} 幀({len(grps)} 角色組 × 12,FDICON 24×24)-> {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
