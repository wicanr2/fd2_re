#!/usr/bin/env python3
"""炎龍騎士團2 — 把一張戰場的單位(roster + 出場座標 + EXE 數值)輸出成引擎用 units.json。

組合三方資料:
  FDFIELD 控制段(parse_field):每單位 camp/portrait/race/cls/lv/spawn_turn
  FDFIELD 出場段(parse_field positions):(X, Y, 肖像;0=己方部署格)
  EXE 單位表(docs/data/exe_tables/unit.json):各 (race,cls) base hp/mp/ap/dp/dx/mv/ex

輸出 <out>/map<N>_units.json:
  { "map","w","h",
    "own_deploy":[{x,y}...],                       # 己方可部署格(肖像==0)
    "units":[ {camp,cls,cls_name,lv,hp,mp,ap,dp,mv,portrait,x,y} ... ] }

數值:以 (race,cls) 查 EXE base;查不到則用 race 任一 cls 近似,再不到給保底值。
重製 Unit 用此 json;原版資產(著作權)只在本機,不入庫。

用法:
  python3 export_units.py <extracted/raw> <map_index> <out_dir>
"""
import sys, os, json
sys.path.insert(0, os.path.dirname(__file__))
import parse_field

EXE_UNIT = os.path.join(os.path.dirname(__file__), "..", "docs", "data", "exe_tables", "unit.json")

# 地圖 sprite 組 = portrait(DATO 角色 id),已反組譯確認(doc 31):
#   繪製碼 0x12831 組=unit[+2](角色 id);0x1291e ×12;0x12926 方向×3 →
#   FDICON sprite index = 組×12 + 方向×3 + 幀。組==portrait(視覺 11/11 + 統一編號)。
#   故 fig = portrait 直接(恆等),不需對應表。


def base_stats(exe, race, cls):
    for u in exe:
        if u.get("race") == race and u.get("cls") == cls:
            return u
    for u in exe:                       # 退而求其次:同 race
        if u.get("race") == race:
            return u
    return {"hp": 20, "mp": 0, "ap": 5, "dp": 2, "dx": 1, "mv": 4, "cls_name": f"r{race}c{cls}"}


def main(argv):
    if len(argv) < 4:
        print(__doc__); return 1
    raw, m, out = argv[1], int(argv[2]), argv[3]
    os.makedirs(out, exist_ok=True)
    info = parse_field.parse_map(raw, m)
    exe = json.load(open(EXE_UNIT, encoding="utf-8"))

    # positions[i] 對應 roster[i](index 對齊;肖像 0=己方部署堆疊,額外的 0 格=可部署格)
    positions = info.get("positions", [])           # [X,Y,肖像]
    own_cells = [{"x": x, "y": y} for (x, y, p) in positions if p == 0]

    units = []
    for i, u in enumerate(info["units"]):
        bs = base_stats(exe, u["race"], u["cls"])
        rec = {
            "camp": u["camp"], "cls": u["cls"], "cls_name": bs.get("cls_name", ""),
            "lv": u["lv"], "portrait": u["portrait"], "spawn_turn": u.get("spawn_turn", 0),
            "hp": bs["hp"], "mp": bs["mp"], "ap": bs["ap"], "dp": bs["dp"], "mv": bs["mv"],
            "fig": u["portrait"],  # 反組譯確認:sprite 組 = portrait(角色 id 恆等)
        }
        if i < len(positions):                       # 固定出場座標(我方會被引擎改放部署格)
            rec["x"], rec["y"] = positions[i][0], positions[i][1]
        units.append(rec)

    res = {"map": m, "w": info["w"], "h": info["h"],
           "own_deploy": own_cells, "units": units}
    p = os.path.join(out, f"map{m}_units.json")
    json.dump(res, open(p, "w", encoding="utf-8"), ensure_ascii=False, separators=(",", ":"))
    print(f"map{m}: {len(units)} units ({sum(1 for u in units if u['camp']!='own')} 敵/友 配座標), "
          f"{len(own_cells)} 己方部署格 -> {p}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
