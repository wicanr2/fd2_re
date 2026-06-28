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

# portrait → FDICON 角色組(地圖 Q 版小人,每組 12=4方向×3幀);
# 精確 Z1 圖形對應待反組譯,先用原版截圖 oracle 粗對(可校)。組 0=紅帽、1=藍帽、9=紅髮、2=灰甲機器人…
PORTRAIT_TO_GROUP = {3: 0, 1: 1}
DEFAULT_GROUP = {"own": 0, "ally": 9, "enemy": 8}


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
            "fig": PORTRAIT_TO_GROUP.get(u["portrait"], DEFAULT_GROUP.get(u["camp"], 0)),
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
