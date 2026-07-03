#!/usr/bin/env python3
"""炎龍騎士團2 — 把一張戰場的單位(roster + 出場座標 + EXE 數值)輸出成引擎用 units.json。

組合三方資料:
  FDFIELD 控制段(parse_field):每單位 camp/portrait/race/cls/lv/spawn_turn
  FDFIELD 出場段(parse_field positions):(X, Y, 肖像;0=己方部署格)
  EXE 單位表(docs/data/exe_tables/unit.json):各 (race,cls) base hp/mp/ap/dp/dx/mv/ex

輸出 <out>/map<N>_units.json:
  { "map","w","h",
    "own_deploy":[{x,y}...],                       # 己方可部署格(肖像==0)
    "units":[ {camp,cls,cls_name,lv,hp,mp,ap,dp,hit,ev,crit,mv,ex,portrait,x,y} ... ] }

數值:以 (race,cls) 查 EXE base;查不到則用 race 任一 cls 近似,再不到給保底值。
重製 Unit 用此 json;原版資產(著作權)只在本機,不入庫。

hit/ev/crit(worklist 第 8 輪 gap-audit doc42 第 1 項補欄,doc02 §4.1 物理攻擊公式需要):
  - crit:職業暴擊率,查 docs/data/exe_tables/resist_crit.json(EXE 0x5219B 反組譯,已與
    doc02 §7.2 人物成長表交叉驗證吻合,如劍士5%/騎士3%/法師3%),查不到(單位表 cls 超出
    26 種玩家職業,如怪物專屬職業)一律 0(保守預設,非文件明確值)。
  - hit/ev:doc03 明確記載這是「衍生值(由上面計算,直接改無效)」——由 DX + 已裝備武器/
    護甲的 HIT/EV(item.json)在遊戲執行時合成,**不是**「敵/友單位 10B」表的原始欄位
    (該表只有 RA/CL/HP/MP/AP/DP/DX/MV/EX,無 HIT/EV 欄)。remake 尚無裝備系統(doc42 gap
    #12),也沒有敵方武器指定資料可用,故此處用固定近似值(DEFAULT_HIT/DEFAULT_EV,取
    item.json 早期武器/護甲數值的中位數區間),而非反組譯或計算所得——doc42 稱「只是
    匯出腳本未取用」不完全準確,實際是來源表本身缺這兩欄,待裝備系統/HIT-EV 合成公式
    RE 出來後再取代此近似值。

用法:
  python3 export_units.py <extracted/raw> <map_index> <out_dir>
"""
import sys, os, json
sys.path.insert(0, os.path.dirname(__file__))
import parse_field

EXE_UNIT = os.path.join(os.path.dirname(__file__), "..", "docs", "data", "exe_tables", "unit.json")
EXE_RESIST_CRIT = os.path.join(os.path.dirname(__file__), "..", "docs", "data", "exe_tables", "resist_crit.json")

# HIT/EV 固定近似值(見檔頭註解:來源表本身無此欄,待裝備系統/合成公式 RE 出來後取代)。
# 落點取 item.json 早期刀劍/防具 HIT(90~100 區間)、EV(5~10 區間)的中位數。
DEFAULT_HIT = 90
DEFAULT_EV = 5


def crit_by_cls(resist_crit, cls):
    for e in resist_crit:
        if e.get("cls") == cls:
            return e.get("crit_pct", 0)
    return 0  # 查不到(如怪物專屬職業,超出 26 種玩家職業表)保守回 0

# 地圖 sprite 組 = portrait(DATO 角色 id),已反組譯確認(doc 31):
#   繪製碼 0x12831 組=unit[+2](角色 id);0x1291e ×12;0x12926 方向×3 →
#   FDICON sprite index = 組×12 + 方向×3 + 幀。組==portrait(視覺 11/11 + 統一編號)。
#   故 fig = portrait 直接(恆等),不需對應表。

# 職業名(cls_name)顯示錯位 bug 根因(worklist 第 8 輪職業名映射修復):
#   exe unit.json 的 cls_name 是 CLASS_NAMES[cls]——「機械職業(戰鬥動畫/成長曲線用)」,
#   如劍士/戰士/騎士,經確認對照 modify2.md §7 職業魔抗/暴擊表 26 筆全部吻合,這個表本身沒錯。
#   但 modify2.md §8「敵/友單位」表證實同一 (race,cls) 會被多個不同「敘事身分」共用
#   (如 race1,cls2 同時是 士兵/傭兵/精英戰士/鎧甲武士/黑暗戰士/狂戰士 六種不同 NPC),
#   而 map0 實測第一章海盜/士兵的 (race,cls) 甚至是 (1,1)——單位表裡唯一對到的是「黑暗殺手」
#   (idx31,與海盜/士兵毫無關係),base_stats() 取第一筆造成顯示「劍士」這種不相干的機械職業名。
#   ∴ CLASS_NAMES[cls] 只適合當「查不到時的機械職業名 fallback」,不能當敘事身分名。
#   真正的敘事身分綁在 portrait(= DATO 角色 id,doc31 §8「已定論」:士兵68、盜賊96、頭目97),
#   故用 portrait 覆寫表,查得到就蓋掉 cls_name;查不到(其餘怪物尚未逐一 RE portrait 對照,
#   留待後續章節逐步補)才落回 CLASS_NAMES[cls] 舊行為,不影響其他已正常顯示的單位。
PORTRAIT_CLS_NAME = {
    68: "士兵",     # doc31 §8:友軍海防隊員(第一章 T6 增援,portrait 68)
    96: "盜賊",     # doc31 §8:第一章開場敵方(俗稱「海盜」,青衫攻略敵方列表寫「LV2盜賊」)
    97: "盜賊頭目",  # doc31 §8:第一章 T5 增援首領(青衫攻略「LV3海盜頭目」)
}


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
    resist_crit = json.load(open(EXE_RESIST_CRIT, encoding="utf-8"))

    # positions[i] 對應 roster[i](index 對齊;肖像 0=己方部署堆疊,額外的 0 格=可部署格)
    positions = info.get("positions", [])           # [X,Y,肖像]
    own_cells = [{"x": x, "y": y} for (x, y, p) in positions if p == 0]

    units = []
    for i, u in enumerate(info["units"]):
        bs = base_stats(exe, u["race"], u["cls"])
        cls_name = PORTRAIT_CLS_NAME.get(u["portrait"], bs.get("cls_name", ""))
        rec = {
            "camp": u["camp"], "cls": u["cls"], "cls_name": cls_name,
            "lv": u["lv"], "portrait": u["portrait"], "group": u.get("group", 0),
            "hp": bs["hp"], "mp": bs["mp"], "ap": bs["ap"], "dp": bs["dp"], "mv": bs["mv"],
            "hit": DEFAULT_HIT, "ev": DEFAULT_EV, "crit": crit_by_cls(resist_crit, u["cls"]),
            "fig": u["portrait"],  # 反組譯確認:sprite 組 = portrait(角色 id 恆等)
            # ex:每級經驗(doc02 §4.5「守方每級經驗」;worklist 第 9 輪經驗值系統補接線)。
            # 與 hp/mp/ap/dp/mv 用同一顆 base_stats() (race,cls) 查表——同一份已驗證 EXE
            # 資料(docs/data/exe_tables/unit.json),同一種「該表多筆同 (race,cls) 只取第一筆」
            # 近似(既有限制,見 base_stats() docstring),非另外新造的猜測值。
            "ex": bs.get("ex", 0),
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
