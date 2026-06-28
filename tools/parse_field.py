#!/usr/bin/env python3
"""炎龍騎士團2 — FDFIELD 戰場定義解析(構成 / 控制 / 出場 三段)。

FDFIELD.DAT 每 3 資源 = 一張戰場:
  資源 3N+0 構成: u16 W, u16 H, 每格 (u16 地形索引, u16 事件/寶箱)
  資源 3N+1 控制: u8 地圖編號, u8 己方可出場數, u8 敵友出場總數,
                  回合事件[16]×3B(回合,事件u16), 保留[16]×2B,
                  寶箱[16]×3B(型態u8:0物品/1金錢, 內容u16),
                  出場人物[敵友總數]×26B(陣營,肖像,種族,職業,等級, 物品×8, 法術×8, 出場回合, 掉落×4)
  資源 3N+2 出場位置: u16 人數, 每組 (u16 X, u16 Y, u16 肖像;0=己方)

用法:
    python3 parse_field.py <raw解包根> <map編號>        # 印該圖定義(JSON 摘要)
    python3 parse_field.py --all <raw解包根> <out.json>  # 全 33 圖 metadata → JSON
"""
import sys
import os
import json
import struct
import glob


def parse_map(raw, m):
    fld = sorted(glob.glob(os.path.join(raw, "FDFIELD", "*.bin")))
    comp = open(fld[m * 3], "rb").read()
    ctl = open(fld[m * 3 + 1], "rb").read()
    spw = open(fld[m * 3 + 2], "rb").read()
    w, h = struct.unpack_from("<HH", comp, 0)
    info = {"map": m, "w": w, "h": h,
            "own_deploy": ctl[1], "enemy_ally_total": ctl[2]}
    # 回合事件
    o = 3
    info["turn_events"] = [{"turn": ctl[o+i*3], "event": struct.unpack_from("<H", ctl, o+i*3+1)[0]}
                           for i in range(16) if ctl[o+i*3] != 0xFF]
    o += 16*3 + 16*2
    info["chests"] = []
    for i in range(16):
        t = ctl[o+i*3]; v = struct.unpack_from("<H", ctl, o+i*3+1)[0]
        if t != 0xFF and v != 0:
            info["chests"].append({"type": "gold" if t == 1 else "item", "value": v})
    o += 16*3
    units = []
    for k in range(ctl[2]):
        b = ctl[o+k*26:o+(k+1)*26]
        if len(b) < 26:
            break
        units.append({"camp": ["enemy", "ally", "own"][b[0]] if b[0] < 3 else b[0],
                      "portrait": b[1], "race": b[2], "cls": b[3], "lv": b[4],
                      "spawn_turn": b[20]})
    info["units"] = units
    n = struct.unpack_from("<H", spw, 0)[0]
    info["positions"] = [list(struct.unpack_from("<HHH", spw, 2+k*6)) for k in range(n)]
    return info


def main(argv):
    if len(argv) < 3:
        print(__doc__); return 1
    if argv[1] == "--all":
        raw, out = argv[2], argv[3]
        fld = sorted(glob.glob(os.path.join(raw, "FDFIELD", "*.bin")))
        maps = []
        for m in range(len(fld)//3):
            try:
                maps.append(parse_map(raw, m))
            except Exception as e:
                maps.append({"map": m, "error": str(e)})
        json.dump(maps, open(out, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
        print(f"{len(maps)} 張戰場 metadata -> {out}")
        return 0
    info = parse_map(argv[1] if False else argv[1], int(argv[2]))
    print(json.dumps(info, ensure_ascii=False, indent=1))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
