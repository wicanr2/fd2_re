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

本輪範圍(刻意不做的部分,別誤以為漏做):
  - 不寫逐章劇情全文,story 節點只放「章節標題 + 目標句」(1-2 句,doc28 摘要)。
  - 不做主角隊招募(莉希亞/鐵諾/瑪琳…doc28 第5欄)狀態機、不做逐章成長曲線
    (party 數值/spells 全章固定沿用 ch01),亦屬下一輪。
  - 不做回合制增援(events 只放 on_battle_start→spawn_party);每章 initial_groups
    全開(units.json 全部真實分組一次到齊),原版 turn_events 的分批增援節奏
    留給下一輪逐章核對 doc28/turn_events 後補。

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
OUT_PATH = os.path.join(REMAKE, "assets", "scenarios", "campaign_full.json")

BGM_BATTLE = "FDMUS_018"  # 戰鬥(doc12:track 18,實測 32 處 play_bgm 呼叫)
BGM_STORY = "FDMUS_010"   # 城鎮/劇情(doc12:track 10)

TOTAL_CHAPTERS = 30  # battle_events.json 的 chapter 0..29(= 攻略 30 關)


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
    """讀既有人工版 ch01.json 的 party 區塊,ch2-30 stub 沿用同一份數值/spells。"""
    ch01_path = os.path.join(REMAKE, "assets", "scenarios", "ch01.json")
    with open(ch01_path, encoding="utf-8") as f:
        return json.load(f)["party"]


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


def build_scenario_stub(c: int, map_idx: int, title: str, party: list[dict]) -> tuple[dict, str]:
    """為攻略章 c(對映 map_idx)生成 scenario stub(party 進場+部署格+分組全開)。"""
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
    return scenario, source


def goal_sentence(row: dict) -> str:
    win = row["win"] if row["win"] and row["win"] != "—" else "敵全滅"
    text = f"目標:{win}。"
    if row["guard"] and row["guard"] != "—":
        text += f"務必保護「{row['guard']}」存活。"
    return text


def build_campaign(
    rows: list[dict], shops_by_ch: dict[int, list[dict]]
) -> tuple[dict, dict[str, str]]:
    nodes: dict[str, dict] = {}
    flags: dict[str, bool] = {}
    party = load_ch01_party()
    scenario_sources: dict[str, str] = {}  # chNN -> 'ch01(既有)' / 'metadata' / 'metadata+fallback_spiral'

    for row in rows:
        c = row["c"]
        cid = f"{c:02d}"
        map_idx = c - 1  # 順序對應(見檔頭說明);ch1(c=1)→map0,與唯一已驗證資料點一致
        map_dir = f"assets/maps/map{map_idx}"
        units_path = f"assets/maps/map{map_idx}/map{map_idx}_units.json"

        # --- scenario stub(主角隊進場+部署格+分組全開;見 build_scenario_stub()) ---
        scenario_path = f"assets/scenarios/ch{cid}.json"
        if c == 1:
            scenario_sources[cid] = "ch01(既有人工版,未覆寫)"
        else:
            scenario, source = build_scenario_stub(c, map_idx, row["title"], party)
            scenario_sources[cid] = source
            out_scn = os.path.join(REMAKE, "assets", "scenarios", f"ch{cid}.json")
            with open(out_scn, "w", encoding="utf-8") as f:
                json.dump(scenario, f, ensure_ascii=False, indent=2)
                f.write("\n")

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
    return campaign, scenario_sources


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
    campaign, scenario_sources = build_campaign(rows, shops_by_ch)

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
