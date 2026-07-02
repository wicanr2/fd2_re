#!/usr/bin/env python3
"""tools/gen_campaign.py — 原版 33 關線性 campaign 自動生成器(M4)。

把已 RE 的資料串成一條完整可跑的節點圖,產出 remake/assets/scenarios/campaign_full.json
(schema 見 remake/internal/campaign/campaign.go 的 Node 型別):

  doc28(30 章勝利/護衛/加入條件,青衫攻略 ground truth)
  + docs/data/shops.json(69 家商店,含 23 家祕密商店)
  + remake/assets/maps/mapN/(map.json + tileset.png + mapN_units.json,全 33 圖已匯出)
  → campaign_full.json:每章 story → battle → (choice→祕密商店線索 →) shop* → 下一章;
    battle 敗北走 retreat 節點重打本章(不是 game over)。

章→map 對應(重要,別誤用別的公式):
  唯一有逐關特徵核對過的資料點是「攻略第1章(bev ch0)= map0」(doc25 §「1章=map0」)。
  doc25 同段**明確推翻**「map = 章節×3+2」公式(套用後與 map2 實際敵方組成對不上)。
  在沒有逐章重新核對 33 張圖敵方特徵之前,本工具採用**順序對應**:
      攻略章 C(1..30) → bev ch (C-1) → map (C-1)
  這與唯一已驗證的資料點一致,是「查不到公式時的合理近似」,非新驗證結果。
  map30/31/32 目前沒有對應的攻略章(30 章戰鬥 vs 33 張圖,差 3 張非戰鬥/分支圖),
  本工具不生成節點去用它們——見檔尾 warm hand-off 註記。

本輪範圍(v1/v2 刻意不做的部分,別誤以為漏做):
  - 不寫逐章劇情全文,story 節點只放「章節標題 + 目標句」(1-2 句,doc28 摘要)。

v3 新增(本輪):
  - **招募**:依 doc28 第5欄「加入角色」名單,party 逐章累積(第 N 章 party = 基礎4人
    + 前面各章已加入角色;當章新加入者不算進當章 party,對映 ch01 哈諾/哈瓦特 T3 才
    登場的既有模式)。數值查 docs/data/exe_tables/characters.json(真實 RE 資料);
    查無角色(doc28 名稱與 characters.json 對不上者,已知僅「達可塞」一例,見
    NAME_ALIASES)或重複加入(如 doc28 萊汀在 ch9/ch10 各出現一次)皆跳過並列印列冊。
  - **成長(粗略,非精確)**:等級 `lv = max(角色初始lv, 章數//2)`(近似節奏,非原版精確
    經驗值曲線)。HP 用 characters.json base_hp(真實資料)+ growth.json 逐級平均增量
    (真實資料,idx 與 characters.json index 對齊,已核對);AP/DP/MV/MP **無逐角色
    base table 可查**(unit.json 68 筆是另一張表,idx 對不上 characters.json 的 32
    名角色,已實測排除),對基礎4人(索爾/亞雷斯/悠妮/蓋亞)用 ch01.json 已知 lv1
    真實數值 + growth.json 逐級增量;對其餘角色用「HP 比例粗算」(比例學習自基礎4人
    ap/hp≈0.48、dp/hp≈0.286、法系 mp/hp≈0.55,見 build_recruit_stats())——**這段是
    近似值,不是反組譯結果**,精確 base table 待未來補 RE。
  - **回合增援:本輪不做,非漏做,是撞牆後的判斷**——見下方「RE 撞牆記錄」。

v4 新增(本輪,接續 v3 撞牆點):
  - **回合增援真資料疊入**:v3 撞牆點(event_id → group 清單)已在後續反組譯輪解開
    (doc25 §6.1:消費點 `0x1a813` + 全域 event_id 跳表 `0x51b91` + spawn 原語 `0x10b4e`),
    `docs/data/turn_events.json` 已補上全 30 章的 `groups`/`handler` 真實資料,
    map0 ground truth 4/4 驗證通過。本輪把這批真資料疊進 ch02-30 的 scenario:
    對每筆有具體 group 的記錄,加 `on_turn_end` + `spawn_group` 事件(寫法比照 ch01.json
    既有事件),並把該 group 從 `initial_groups` 移除(改為回合增援登場,不再全部開局在場)。
    見 `apply_reinforcements()`。
  - **三種 groups 值**:整數清單(直接可用)/ `$turn_counter[0x53bef]`(動態=觸發當下回合數,
    因每筆記錄本身即為單回合觸發實例,直接拿該筆 `turn` 當 group 即可——用 map7/23/25 的
    真實 group 集合核對過,turn 值與圖上 group 編號完全對得上,見 doc25 §6.1)/
    其他動態值(如 `$reg_or_mem(eax)`,event_id 47/49,暫存器值無法靜態解析)。
  - **安全網(寧可全開也不要單位消失)**:單筆記錄任一 group 不存在於該章 units.json →
    整筆退回列冊,不動 initial_groups;全部有效記錄套用後 initial_groups 會清空 →
    整章退回列冊,維持 v2 全開安全預設。無 spawn handler 的記錄(對話/AI類,groups 為空)
    直接列冊跳過。

## RE 撞牆記錄:為何 v3 不做「回合增援」spawn_group(2026-07 gen_campaign v3)

派工時以為 `docs/data/battle_events.json` 是「30 章回合事件 dump(條件→動作)」,可以
直接轉成 chNN.json 的 `spawn_group` 事件。**核實後這個假設是錯的**:battle_events.json
其實是 doc26 的 handler「勝負判定」metadata(is_default/result_codes/extra_conditions/
trigger_units_flag),30 章裡只有 7 章有 `roster_has`/`unit_flag` 這種條件字串,**完全沒有
turn/group 欄位**——它跟「第幾回合誰增援」無關。

真正的回合增援資料在 **FDFIELD.DAT 控制段**(`tools/parse_field.py` 的 `turn_events`:
每條 `turn u8 + 全域event_id u8 + camp u8`),ch01.json 現有的 T3/T4/T5/T6 事件就是從這裡
配合青衫攻略人工核對出來的(doc29 §11 記載「當實作藍本」,唯一一份有 ground truth 核對過
的關卡)。**已把全 30 章 turn_events 真實資料(2026-07 用 parse_field.py 重新 dump)存進
`docs/data/turn_events.json`**(每筆含 map/chapter + turn_events[]:turn/event_id/camp),
供下一輪專門的 `0x22e5c`(event_id→實際 group 清單)反組譯任務使用。

卡點:turn_events 只給「第幾回合 + 敵/友/特殊」三個欄位,**不給「具體哪個 unit group」**。
用 map0(=ch1,唯一有人工核對過的正解)反向驗證了兩種機械式分配法都失敗:
  - 「群組座標聚類」:group3/7 兩個 1 人小隊確實共用座標 (11,11)(印證聚類本身有效),
    但「哪一波該留在場上 vs 哪一波該延後登場」這件事,座標和 group 編號都測不出規律——
    真正的增援是 group4/5(數字比較大),但初始群卻包含 group1/2/10/11 這種數字更小或
    更大都有的組合,沒有單調規則。
  - 「group 編號升冪對應 turn 升冪」:對 map0 給出 group1→T4、group2→T5,但正解其實是
    group4→T4、group5→T5,group1/2 應留在初始——直接測試失敗,不是理論推測。
  → 這條資訊(event_id → 實際 group 清單)大概率藏在 doc25 §6 提到但尚未反組譯的
  world-map/中場 handler(`0x22e5c`,標記 `[阻]`),不在 FDFIELD 資料裡,純資料驅動猜不出來。
  硬套會產出「看起來資料驅動、實際是編出來的」假 RE 事實,違反本專案「保全歷史」的紅線
  (rulebook 83),所以這輪**不做**,新生成的 ch02-30 initial_groups 維持 v2「全開」安全預設
  (原地圖真實分組全部開局在場,不精確還原原版分波節奏,但不會播錯資料)。

scenario stub(chNN.json,ch2-30 本輪新生成,見 build_scenario_stub()):
  - party:沿用 ch01.json 的 4 人數值/spells(index 對齊,不做逐章成長)。
  - deploy_cells:優先取 mapN_units.json 的 own_deploy(= FDFIELD 出場位置資源
    portrait==0 格,tools/export_units.py 萃取,map0 版本已與人工核對過的 ch01.json
    完全吻合——ground truth)。own_deploy 部分地圖有重複座標/與真實單位重疊
    (資料本身如此,非解析 bug),先去重+濾重疊,不足 4 格再用 spiral 保底搜尋
    (見 pick_deploy_cells());三層資料來源標記 metadata / metadata+fallback_spiral。
  - initial_groups:units.json 全部真實 group(**排除 group==255**——實測每張圖的
    255 群體固定堆疊在同一座標點,是資料裡的未用槽位/padding,非真實出場波次,
    見 tools/parse_field.py b21=group 註解 + 逐圖驗證)。
  - ch01 沿用既有人工版 assets/scenarios/ch01.json,不覆寫。

用法:
    python3 tools/gen_campaign.py
輸出:
    remake/assets/scenarios/campaign_full.json(battle 節點含 scenario 欄位)
    remake/assets/scenarios/ch02.json ~ ch30.json(新生成)
    + 終端列印驗證結果(節點數/轉場完整性/資產檔案存在性/scenario 資料來源統計)
"""
from __future__ import annotations

import json
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
REMAKE = os.path.join(ROOT, "remake")
DOC28 = os.path.join(ROOT, "docs", "knowledge-base", "28-chapter-objectives-and-recruits.md")
SHOPS_JSON = os.path.join(ROOT, "docs", "data", "shops.json")
CHARACTERS_JSON = os.path.join(ROOT, "docs", "data", "exe_tables", "characters.json")
GROWTH_JSON = os.path.join(ROOT, "docs", "data", "exe_tables", "growth.json")
TURN_EVENTS_JSON = os.path.join(ROOT, "docs", "data", "turn_events.json")
OUT_PATH = os.path.join(REMAKE, "assets", "scenarios", "campaign_full.json")

BGM_BATTLE = "FDMUS_018"  # 戰鬥(doc12:track 18,實測 32 處 play_bgm 呼叫)
BGM_STORY = "FDMUS_010"   # 城鎮/劇情(doc12:track 10)

TOTAL_CHAPTERS = 30  # battle_events.json 的 chapter 0..29(= 攻略 30 關)

# doc28 加入角色名單的用字與 characters.json 對不上的少數例外(核對後只有這一筆)。
NAME_ALIASES = {"達可塞": "達克賽"}

# 無逐角色 AP/DP/MV/MP base table 可查(unit.json 68 筆是另一張表,idx 對不上
# characters.json 的 32 名角色,已實測排除)。用基礎4人(索爾/亞雷斯/悠妮/蓋亞)的
# ch01.json 已知真實數值反推 HP 比例,近似套用到其餘角色 —— 這段是近似值不是 RE 結果。
CASTER_CLASSES = {"法師", "大法師", "僧侶", "祭師", "聖者"}
MV_BY_CLASS = {"騎士": 7, "聖騎士": 7, "龍騎士": 7, "神射手": 6,
               "法師": 5, "大法師": 5, "僧侶": 5, "祭師": 5, "聖者": 5}
DEFAULT_MV = 6
AP_HP_RATIO_CASTER = 0.36
AP_HP_RATIO_NORMAL = 0.48
DP_HP_RATIO = 0.286
MP_HP_RATIO_CASTER = 0.55


def parse_doc28(path: str) -> list[dict]:
    """解析 doc28 的 30 關目標表(攻略章|bev ch|標題|勝利條件|額外護衛|加入角色)。"""
    rows = []
    with open(path, encoding="utf-8") as f:
        lines = f.readlines()
    in_table = False
    for line in lines:
        line = line.rstrip("\n")
        if line.startswith("| 攻略章"):
            in_table = True
            continue
        if not in_table:
            continue
        if line.startswith("|---"):
            continue
        if not line.startswith("|"):
            break
        cols = [c.strip() for c in line.strip().strip("|").split("|")]
        if len(cols) < 6:
            continue
        try:
            c = int(cols[0])
        except ValueError:
            continue
        strip_md = lambda s: re.sub(r"\*\*", "", s).strip()
        rows.append(
            dict(
                c=c,
                bev=int(cols[1]),
                title=strip_md(cols[2]),
                win=strip_md(cols[3]),
                guard=strip_md(cols[4]),
                recruit=strip_md(cols[5]),
            )
        )
    if len(rows) != TOTAL_CHAPTERS:
        raise SystemExit(f"doc28 解析出 {len(rows)} 列,預期 {TOTAL_CHAPTERS} 列(表格格式可能變了)")
    return rows


def load_shops_by_chapter(path: str) -> dict[int, list[dict]]:
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
    by_ch: dict[int, list[dict]] = {}
    for s in data["shops"]:
        by_ch.setdefault(s["chapter"], []).append(s)
    # 固定順序:weapon → item → 其他 → secret 最後(secret 掛在最後一個一般商店節點上)
    order = {"weapon": 0, "item": 1, "secret": 99}
    for ch, rows in by_ch.items():
        rows.sort(key=lambda s: order.get(s["kind"], 50))
    return by_ch


def load_ch01_party() -> list[dict]:
    """讀既有人工版 ch01.json 的 party 區塊(基礎4人,含 ch01 已知真實 lv1 數值)。"""
    ch01_path = os.path.join(REMAKE, "assets", "scenarios", "ch01.json")
    with open(ch01_path, encoding="utf-8") as f:
        return json.load(f)["party"]


def load_characters(path: str) -> dict[str, dict]:
    """讀 characters.json,回傳 name -> record(含 index,供對齊 growth.json idx)。"""
    with open(path, encoding="utf-8") as f:
        rows = json.load(f)
    return {r["name"]: r for r in rows}


def load_growth(path: str) -> dict[int, dict]:
    """讀 growth.json,回傳 idx -> 逐級增量 record(ap/dp/dx/hp/mp 皆 [min,max])。"""
    with open(path, encoding="utf-8") as f:
        rows = json.load(f)
    return {r["idx"]: r for r in rows}


def _avg(rng: list[int]) -> float:
    return (rng[0] + rng[1]) / 2


def parse_recruit_names(raw: str) -> list[str]:
    """doc28 第5欄「加入角色」→ 乾淨角色名清單(去條件括號、拆多人)。"""
    if not raw or raw.strip() in ("—", "-", ""):
        return []
    text = re.sub(r"[\(（][^\)）]*[\)）]", "", raw)
    parts = re.split(r"[、;,]", text)
    return [NAME_ALIASES.get(p.strip(), p.strip()) for p in parts if p.strip() and p.strip() != "—"]


def build_base4_stats(
    base4_by_name: dict[str, dict],
    characters_by_name: dict[str, dict],
    growth_by_idx: dict[int, dict],
    name: str,
    chapter: int,
) -> dict:
    """基礎4人(索爾/亞雷斯/悠妮/蓋亞):ch01.json 已知 lv1 真實數值 + growth.json 逐級真實增量。"""
    base = base4_by_name[name]
    char = characters_by_name.get(name)
    lv = max(1, chapter // 2)
    levels = max(0, lv - 1)
    growth = growth_by_idx.get(char["index"]) if char else None
    hp, ap, dp, mp = base["hp"], base["ap"], base["dp"], base["mp"]
    if growth and levels:
        hp += round(levels * _avg(growth["hp"]))
        ap += round(levels * _avg(growth["ap"]))
        dp += round(levels * _avg(growth["dp"]))
        mp += round(levels * _avg(growth["mp"]))
    return {
        "name": name, "cls": base["cls"], "fig": base["fig"], "portrait": base["portrait"],
        "lv": lv, "hp": hp, "mp": mp, "ap": ap, "dp": dp, "mv": base["mv"],
        **({"spells": base["spells"]} if "spells" in base else {}),
    }


def build_recruit_stats(
    char: dict, growth_by_idx: dict[int, dict], chapter: int,
) -> dict:
    """招募角色(非基礎4人):characters.json 真實 base_hp + growth.json 真實逐級 HP 增量;
    AP/DP/MV/MP 無逐角色 base table,用「HP 比例粗算」近似(見檔頭 RE 撞牆記錄同段落說明)。
    """
    initial_lv, initial_hp = char["lv"], char["base_hp"]
    lv = max(initial_lv, chapter // 2)
    levels = max(0, lv - initial_lv)
    growth = growth_by_idx.get(char["index"])
    hp = initial_hp
    if growth and levels:
        hp += round(levels * _avg(growth["hp"]))
    is_caster = char["cls_name"] in CASTER_CLASSES
    ap_ratio = AP_HP_RATIO_CASTER if is_caster else AP_HP_RATIO_NORMAL
    return {
        "name": char["name"], "cls": char["cls_name"],
        "fig": char["sprite_group"], "portrait": char["face_portrait"],
        "lv": lv, "hp": hp,
        "mp": round(hp * MP_HP_RATIO_CASTER) if is_caster else 0,
        "ap": round(hp * ap_ratio), "dp": round(hp * DP_HP_RATIO),
        "mv": MV_BY_CLASS.get(char["cls_name"], DEFAULT_MV),
    }


def build_recruit_plan(
    rows: list[dict], characters_by_name: dict[str, dict], base4_names: set[str]
) -> tuple[dict[int, list[str]], list[str]]:
    """doc28 第5欄逐章解析加入名單,回傳 (chapter -> 新加入角色名清單, 跳過列冊)。

    跳過情形:doc28 名稱在 characters.json 查無(alias 表已處理已知例外)、
    或角色已在先前章節加入過(如 doc28 萊汀在 ch9/ch10 各出現一次,以先出現者為準)。
    """
    plan: dict[int, list[str]] = {}
    skipped: list[str] = []
    already = set(base4_names)
    for row in rows:
        names = parse_recruit_names(row["recruit"])
        joined_this_chapter = []
        for name in names:
            if name in already:
                skipped.append(f"ch{row['c']:02d}:{name}(重複加入,已在先前章節列入 party,跳過)")
                continue
            if name not in characters_by_name:
                skipped.append(f"ch{row['c']:02d}:{name}(characters.json 查無此名,跳過)")
                continue
            joined_this_chapter.append(name)
            already.add(name)
        plan[row["c"]] = joined_this_chapter
    return plan, skipped


def spiral_offsets(max_r: int):
    """以 (0,0) 為中心,由近到遠的方格螺旋座標偏移量(含自身)。"""
    yield (0, 0)
    for r in range(1, max_r + 1):
        for dx in range(-r, r + 1):
            for dy in range(-r, r + 1):
                if max(abs(dx), abs(dy)) == r:
                    yield (dx, dy)


def pick_deploy_cells(
    own_deploy: list[dict], occupied: set[tuple[int, int]], w: int, h: int, n: int
) -> tuple[list[tuple[int, int]], str]:
    """從 mapN_units.json 的 own_deploy(FDFIELD 出場位置,portrait==0 格)挑 n 個部署格。

    own_deploy 部分地圖本身就有重複座標,或座標與「全開」後的真實單位重疊
    (見檔頭說明,不是解析錯誤)。優先去重+濾重疊+界內取滿 n 個(source='metadata');
    不足時以第一個 own_deploy 格為錨點做螺旋搜尋補足空格(source='metadata+fallback_spiral',
    worklist 要求的保底邏輯——目前 30 章實測皆有 own_deploy 可用,此路徑只在座標退化
    〔如整批重複同一點〕時觸發)。
    """
    seen: set[tuple[int, int]] = set()
    clean: list[tuple[int, int]] = []
    for c in own_deploy:
        pt = (c["x"], c["y"])
        if pt in seen or pt in occupied or not (0 <= pt[0] < w and 0 <= pt[1] < h):
            continue
        seen.add(pt)
        clean.append(pt)
        if len(clean) >= n:
            return clean, "metadata"

    anchor = (own_deploy[0]["x"], own_deploy[0]["y"]) if own_deploy else (w // 2, h - 1)
    for dx, dy in spiral_offsets(max(w, h)):
        pt = (anchor[0] + dx, anchor[1] + dy)
        if pt in seen or pt in occupied or not (0 <= pt[0] < w and 0 <= pt[1] < h):
            continue
        seen.add(pt)
        clean.append(pt)
        if len(clean) >= n:
            break
    return clean[:n], "metadata+fallback_spiral"


def load_turn_events(path: str) -> dict[int, list[dict]]:
    """讀 turn_events.json,回傳 chapter -> turn_events[] (見 doc25 §6.1)。"""
    with open(path, encoding="utf-8") as f:
        rows = json.load(f)
    return {row["chapter"]: row["turn_events"] for row in rows}


def apply_reinforcements(
    scenario: dict, records: list[dict], map_groups: set[int], cid: str
) -> tuple[int, list[str]]:
    """把 turn_events.json 該章記錄疊進 scenario:initial_groups 移除對應 group +
    events 加 on_turn_end/spawn_group(機制見 doc25 §6.1、消費點 0x1a813,ground truth
    見 map0 4/4 驗證;寫法比照 ch01.json 既有事件——`when.turn` 是精確比對,非 turn_gte,
    對映反組譯 0x01a844 `cmp edx,[0x53bef]` 精確比對、event.go `When.match()` 同語意)。

    三種記錄類型:
      - groups 為整數清單 → 直接可用。
      - groups==["$turn_counter[0x53bef]"](event_id 27/54/57)→ 動態值 = 觸發當下回合數,
        因每筆記錄本身已是單回合觸發實例,直接以該筆 turn 當 group(已用 map7/23/25 的
        真實 group 集合核對過,turn 值與圖上 group 編號完全對得上)。
      - 其他動態值(如 $reg_or_mem(eax),event_id 47/49)→ 無法靜態解析實際 group,跳過列冊。
      - groups==[] → 無 spawn handler(對話/AI類),跳過列冊。

    安全網(寧可全開也不要單位消失):
      - 單筆記錄內任一 group 不存在於該章 units.json → 整筆退回列冊,不動 initial_groups。
      - 全部有效記錄套用後 initial_groups 會變空 → 整章退回列冊(維持 v2 全開安全預設)。
    """
    skipped: list[str] = []
    candidates: list[tuple[int, int, str, list[int]]] = []  # turn, event_id, camp, groups

    for rec in records:
        groups_raw = rec["groups"]
        turn, event_id, camp = rec["turn"], rec["event_id"], rec["camp"]
        if not groups_raw:
            skipped.append(f"ch{cid}:event{event_id}@T{turn}(無 spawn handler/對話類,跳過)")
            continue
        if groups_raw == ["$turn_counter[0x53bef]"]:
            resolved = [turn]
        elif all(isinstance(g, int) for g in groups_raw):
            resolved = list(groups_raw)
        else:
            skipped.append(f"ch{cid}:event{event_id}@T{turn}(動態值{groups_raw!r}無法靜態解析,跳過)")
            continue
        missing = [g for g in resolved if g not in map_groups]
        if missing:
            skipped.append(
                f"ch{cid}:event{event_id}@T{turn} groups={resolved}(含 map 不存在的 group{missing},整筆退回列冊)"
            )
            continue
        candidates.append((turn, event_id, camp, resolved))

    if not candidates:
        return 0, skipped

    reinforce_groups = {g for _, _, _, gs in candidates for g in gs}
    remaining_initial = [g for g in scenario["initial_groups"] if g not in reinforce_groups]
    if scenario["initial_groups"] and not remaining_initial:
        skipped.append(
            f"ch{cid}: 全部 {len(candidates)} 筆增援套用後 initial_groups 會清空,"
            "整章退回列冊(維持全開安全預設)"
        )
        return 0, skipped

    scenario["initial_groups"] = remaining_initial
    for turn, event_id, camp, gs in candidates:
        scenario["events"].append(
            {
                "id": f"reinforce_ch{cid}_e{event_id}_t{turn}",
                "trigger": "on_turn_end",
                "when": {"turn": turn},
                "once": True,
                "do": [{"type": "spawn_group", "groups": sorted(gs), "camp": camp}],
            }
        )
    return len(candidates), skipped


def build_scenario_stub(
    c: int, map_idx: int, title: str, party: list[dict], turn_records: list[dict]
) -> tuple[dict, str, int, list[str]]:
    """為攻略章 c(對映 map_idx)生成 scenario stub(party 進場+部署格+回合增援疊入)。"""
    units_path = os.path.join(REMAKE, "assets", "maps", f"map{map_idx}", f"map{map_idx}_units.json")
    with open(units_path, encoding="utf-8") as f:
        md = json.load(f)

    real_units = [u for u in md["units"] if u.get("group", 0) != 255]  # 排除 255 padding 槽位
    occupied = {(u["x"], u["y"]) for u in real_units}
    cells, source = pick_deploy_cells(md.get("own_deploy", []), occupied, md["w"], md["h"], len(party))
    groups = sorted({u["group"] for u in real_units})

    scenario = {
        "chapter": c,
        "name": title,
        "map": map_idx,
        "initial_groups": groups,
        "party": party,
        "deploy_cells": [[x, y] for x, y in cells],
        "events": [
            {
                "id": "opening",
                "trigger": "on_battle_start",
                "once": True,
                "do": [{"type": "spawn_party"}],
            }
        ],
    }
    n_reinforced, skipped = apply_reinforcements(scenario, turn_records, set(groups), f"{c:02d}")
    return scenario, source, n_reinforced, skipped


def goal_sentence(row: dict) -> str:
    win = row["win"] if row["win"] and row["win"] != "—" else "敵全滅"
    text = f"目標:{win}。"
    if row["guard"] and row["guard"] != "—":
        text += f"務必保護「{row['guard']}」存活。"
    return text


def build_campaign(
    rows: list[dict], shops_by_ch: dict[int, list[dict]],
    characters_by_name: dict[str, dict], growth_by_idx: dict[int, dict],
    turn_events_by_ch: dict[int, list[dict]],
) -> tuple[dict, dict[str, str], dict[int, list[str]], list[str], dict[str, int], list[str]]:
    nodes: dict[str, dict] = {}
    flags: dict[str, bool] = {}
    base4 = load_ch01_party()
    base4_by_name = {m["name"]: m for m in base4}
    base4_order = [m["name"] for m in base4]
    scenario_sources: dict[str, str] = {}  # chNN -> 'ch01(既有)' / 'metadata' / 'metadata+fallback_spiral'
    reinforce_counts: dict[str, int] = {}  # chNN -> 疊入的增援事件筆數
    reinforce_skipped: list[str] = []  # 全 30 章列冊跳過統計

    recruit_plan, skipped_recruits = build_recruit_plan(rows, characters_by_name, set(base4_order))
    joined_so_far: list[str] = []  # 前面各章已加入角色(不含當章新加入者,對映 party 累積規格)

    for row in rows:
        c = row["c"]
        cid = f"{c:02d}"
        map_idx = c - 1  # 順序對應(見檔頭說明);ch1(c=1)→map0,與唯一已驗證資料點一致
        map_dir = f"assets/maps/map{map_idx}"
        units_path = f"assets/maps/map{map_idx}/map{map_idx}_units.json"

        # --- 本章 party:基礎4人(逐章成長)+ 前面各章已加入角色(逐章成長,近似值見檔頭) ---
        party = (
            [build_base4_stats(base4_by_name, characters_by_name, growth_by_idx, name, c) for name in base4_order]
            + [build_recruit_stats(characters_by_name[name], growth_by_idx, c) for name in joined_so_far]
        )

        # --- scenario stub(主角隊進場+部署格+分組全開;見 build_scenario_stub()) ---
        scenario_path = f"assets/scenarios/ch{cid}.json"
        if c == 1:
            scenario_sources[cid] = "ch01(既有人工版,未覆寫)"
        else:
            scenario, source, n_reinforced, skipped = build_scenario_stub(
                c, map_idx, row["title"], party, turn_events_by_ch.get(c, [])
            )
            scenario_sources[cid] = source
            reinforce_counts[cid] = n_reinforced
            reinforce_skipped.extend(skipped)
            out_scn = os.path.join(REMAKE, "assets", "scenarios", f"ch{cid}.json")
            with open(out_scn, "w", encoding="utf-8") as f:
                json.dump(scenario, f, ensure_ascii=False, indent=2)
                f.write("\n")

        joined_so_far.extend(recruit_plan.get(c, []))  # 當章新加入者從下一章開始才算進 party

        story_id = f"story_ch{cid}"
        battle_id = f"battle_ch{cid}"
        retreat_id = f"retreat_ch{cid}"

        is_last = c == TOTAL_CHAPTERS
        next_story_id = None if is_last else f"story_ch{c + 1:02d}"
        ending_id = "ending" if is_last else None

        chapter_shops = shops_by_ch.get(c, [])
        secret_row = next((s for s in chapter_shops if s["kind"] == "secret"), None)
        normal_rows = [s for s in chapter_shops if s["kind"] != "secret"]

        # --- story 節點(章節標題 + doc28 目標句) ---
        nodes[story_id] = {
            "type": "story",
            "bgm": BGM_STORY,
            "lines": [
                {"speaker": -1, "text": f"第{c}章:{row['title']}"},
                {"speaker": -1, "text": goal_sentence(row)},
            ],
            "next": battle_id,
        }

        # --- battle 節點 ---
        after_battle_id = None  # on_win 目標,稍後依有無商店決定

        # --- shop 鏈(有商店才生成) ---
        shop_node_ids: list[str] = []
        if normal_rows:
            for i, s in enumerate(normal_rows):
                sid = f"shop_ch{cid}_{s['kind']}"
                shop_node_ids.append(sid)
                nodes[sid] = {
                    "type": "shop",
                    "bgm": BGM_STORY,
                    "goods": [{"name": g["name"], "price": g["price"]} for g in s["goods"]],
                    "next": None,  # 稍後串接
                }
            # secret 掛在最後一個一般商店節點
            if secret_row:
                last_sid = shop_node_ids[-1]
                flag_name = f"found_secret_ch{cid}"
                flags[flag_name] = False
                nodes[last_sid]["secret_if"] = flag_name
                nodes[last_sid]["secret"] = [
                    {"name": g["name"], "price": g["price"]} for g in secret_row["goods"]
                ]
            # 串接商店鏈
            for i in range(len(shop_node_ids) - 1):
                nodes[shop_node_ids[i]]["next"] = shop_node_ids[i + 1]
            tail_target = ending_id if is_last else next_story_id
            nodes[shop_node_ids[-1]]["next"] = tail_target

            if secret_row:
                choice_id = f"choice_ch{cid}"
                rumor_id = f"rumor_ch{cid}"
                flag_name = f"found_secret_ch{cid}"
                nodes[choice_id] = {
                    "type": "choice",
                    "prompt": "進城前要不要先打聽消息?",
                    "options": [
                        {"label": "打聽消息", "to": rumor_id},
                        {"label": "直接進城採買", "to": shop_node_ids[0]},
                    ],
                }
                nodes[rumor_id] = {
                    "type": "story",
                    "bgm": BGM_STORY,
                    "lines": [
                        {
                            "speaker": -1,
                            "text": f"聽說這裡有不擺在檯面上的好東西……({secret_row.get('how_to_enter', '')})",
                        }
                    ],
                    "set_flags": {flag_name: True},
                    "next": shop_node_ids[0],
                }
                after_battle_id = choice_id
            else:
                after_battle_id = shop_node_ids[0]
        else:
            after_battle_id = ending_id if is_last else next_story_id

        nodes[battle_id] = {
            "type": "battle",
            "bgm": BGM_BATTLE,
            "map": map_dir,
            "units": units_path,
            "scenario": scenario_path,
            "on_win": after_battle_id,
            "on_lose": retreat_id,
        }

        retry_flag = f"retried_ch{cid}"
        flags[retry_flag] = False
        nodes[retreat_id] = {
            "type": "story",
            "bgm": BGM_STORY,
            "lines": [
                {"speaker": -1, "text": "撤退!先回頭整頓,再找機會反攻……"},
            ],
            "set_flags": {retry_flag: True},
            "next": battle_id,
        }

    nodes["ending"] = {
        "type": "ending",
        "text": (
            "傳說的終章——空魔神殞落,炎龍騎士團的旅程至此告一段落。"
            "(campaign_full.json 自動生成 v1:節點骨架完整,逐章劇情全文/回合事件/"
            "主角隊招募待下一輪補完)"
        ),
    }

    # 清掉 shop 節點暫時的 None next(理論上都已在上面填滿,防呆用)
    for n in nodes.values():
        if n.get("type") == "shop" and n.get("next") is None:
            raise SystemExit(f"內部錯誤:shop 節點 next 未串接 {n}")

    campaign = {
        "title": "炎龍騎士團2 — 全 30 章線性 campaign(自動生成)",
        "start": "story_ch01",
        "flags": flags,
        "nodes": nodes,
    }
    return campaign, scenario_sources, recruit_plan, skipped_recruits, reinforce_counts, reinforce_skipped


def validate(campaign: dict) -> list[str]:
    """比照 campaign.go Load() 的驗證邏輯(轉場目標存在)+ 額外檔案存在性檢查。"""
    problems: list[str] = []
    nodes = campaign["nodes"]

    if campaign["start"] not in nodes:
        problems.append(f"start 節點 {campaign['start']!r} 不存在")

    def check_target(from_id: str, to: str | None):
        if not to:
            return
        if to not in nodes:
            problems.append(f"節點 {from_id!r} 的轉場目標 {to!r} 不存在")

    for nid, n in nodes.items():
        check_target(nid, n.get("next"))
        check_target(nid, n.get("on_win"))
        check_target(nid, n.get("on_lose"))
        for opt in n.get("options", []):
            check_target(nid, opt.get("to"))
        if n.get("type") == "battle":
            map_dir = n.get("map")
            units_path = n.get("units")
            if map_dir:
                abs_dir = os.path.join(REMAKE, map_dir)
                if not os.path.isdir(abs_dir):
                    problems.append(f"battle 節點 {nid!r} 的 map 目錄不存在:{map_dir}")
                else:
                    for fn in ("map.json", "tileset.png"):
                        if not os.path.isfile(os.path.join(abs_dir, fn)):
                            problems.append(f"battle 節點 {nid!r} 缺 {fn}:{map_dir}")
            if units_path:
                abs_units = os.path.join(REMAKE, units_path)
                if not os.path.isfile(abs_units):
                    problems.append(f"battle 節點 {nid!r} 的 units 檔不存在:{units_path}")
            scn_path = n.get("scenario")
            if scn_path:
                abs_scn = os.path.join(REMAKE, scn_path)
                if not os.path.isfile(abs_scn):
                    problems.append(f"battle 節點 {nid!r} 的 scenario 檔不存在:{scn_path}")
                elif units_path and os.path.isfile(os.path.join(REMAKE, units_path)):
                    with open(abs_scn, encoding="utf-8") as f:
                        sc = json.load(f)
                    with open(os.path.join(REMAKE, units_path), encoding="utf-8") as f:
                        um = json.load(f)
                    w, h = um["w"], um["h"]
                    occupied = {(u["x"], u["y"]) for u in um["units"] if u.get("group", 0) != 255}
                    seen_cells: set[tuple[int, int]] = set()
                    for x, y in sc.get("deploy_cells", []):
                        if not (0 <= x < w and 0 <= y < h):
                            problems.append(f"scenario {scn_path!r} 部署格越界:({x},{y}) vs {w}x{h}")
                        if (x, y) in occupied:
                            problems.append(f"scenario {scn_path!r} 部署格 ({x},{y}) 與真實單位重疊")
                        if (x, y) in seen_cells:
                            problems.append(f"scenario {scn_path!r} 部署格重複:({x},{y})")
                        seen_cells.add((x, y))

    # 可達性檢查(從 start 走一遍,確保沒有孤兒/斷鏈節點——不算 hard error,只警示)
    seen = set()
    stack = [campaign["start"]]
    while stack:
        cur = stack.pop()
        if cur in seen or cur not in nodes:
            continue
        seen.add(cur)
        n = nodes[cur]
        for to in (n.get("next"), n.get("on_win"), n.get("on_lose")):
            if to:
                stack.append(to)
        for opt in n.get("options", []):
            if opt.get("to"):
                stack.append(opt["to"])
    unreachable = set(nodes) - seen
    if unreachable:
        problems.append(f"從 start 走不到的節點({len(unreachable)} 個):{sorted(unreachable)}")

    return problems


def main():
    rows = parse_doc28(DOC28)
    shops_by_ch = load_shops_by_chapter(SHOPS_JSON)
    characters_by_name = load_characters(CHARACTERS_JSON)
    growth_by_idx = load_growth(GROWTH_JSON)
    turn_events_by_ch = load_turn_events(TURN_EVENTS_JSON)
    campaign, scenario_sources, recruit_plan, skipped_recruits, reinforce_counts, reinforce_skipped = build_campaign(
        rows, shops_by_ch, characters_by_name, growth_by_idx, turn_events_by_ch
    )

    os.makedirs(os.path.dirname(OUT_PATH), exist_ok=True)
    with open(OUT_PATH, "w", encoding="utf-8") as f:
        json.dump(campaign, f, ensure_ascii=False, indent=2)
        f.write("\n")

    n_nodes = len(campaign["nodes"])
    n_battle = sum(1 for n in campaign["nodes"].values() if n["type"] == "battle")
    n_shop = sum(1 for n in campaign["nodes"].values() if n["type"] == "shop")
    n_choice = sum(1 for n in campaign["nodes"].values() if n["type"] == "choice")
    n_story = sum(1 for n in campaign["nodes"].values() if n["type"] == "story")

    print(f"寫出 {OUT_PATH}")
    print(f"章數:{TOTAL_CHAPTERS}  節點總數:{n_nodes}")
    print(f"  battle={n_battle} shop={n_shop} choice={n_choice} story={n_story} ending=1")

    n_new_scn = sum(1 for cid, src in scenario_sources.items() if cid != "01")
    n_meta = sum(1 for cid, src in scenario_sources.items() if src == "metadata")
    n_fb = sum(1 for cid, src in scenario_sources.items() if src == "metadata+fallback_spiral")
    print(f"scenario stub:新生成 {n_new_scn} 章(ch02-ch{TOTAL_CHAPTERS:02d})"
          f"  來源=metadata:{n_meta}  metadata+fallback_spiral:{n_fb}")
    if n_fb:
        fb_chs = [cid for cid, src in scenario_sources.items() if src == "metadata+fallback_spiral"]
        print(f"  fallback 章節:{sorted(fb_chs)}")

    n_recruited = sum(len(v) for v in recruit_plan.values())
    print(f"\n招募:doc28 名單解析出 {n_recruited} 名新加入角色(不含基礎4人)")
    for c in sorted(recruit_plan):
        if recruit_plan[c]:
            print(f"  ch{c:02d}: {'、'.join(recruit_plan[c])}")
    if skipped_recruits:
        print(f"  列冊跳過 {len(skipped_recruits)} 筆:")
        for s in skipped_recruits:
            print(f"    - {s}")

    n_reinforce_events = sum(reinforce_counts.values())
    n_reinforce_chapters = sum(1 for n in reinforce_counts.values() if n > 0)
    print(f"\n回合增援:{n_reinforce_chapters} 章疊入共 {n_reinforce_events} 筆 spawn_group 事件")
    for cid in sorted(reinforce_counts):
        if reinforce_counts[cid] > 0:
            print(f"  ch{cid}: {reinforce_counts[cid]} 筆")
    if reinforce_skipped:
        print(f"  列冊跳過 {len(reinforce_skipped)} 筆:")
        for s in reinforce_skipped:
            print(f"    - {s}")

    # JSON 合法性(已用 json.dump 產生,這裡重讀一次確保可被 json.load 解析)
    with open(OUT_PATH, encoding="utf-8") as f:
        json.load(f)
    print("JSON 合法性:OK")

    problems = validate(campaign)
    if problems:
        print(f"\n驗證發現 {len(problems)} 個問題:")
        for p in problems:
            print(f"  - {p}")
        sys.exit(1)
    else:
        print("驗證:轉場目標完整、map/units 檔案皆存在、無孤兒節點。")


if __name__ == "__main__":
    main()
