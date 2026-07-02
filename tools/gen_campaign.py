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
  - 不生成 remake/assets/scenarios/chNN.json(逐章 Scenario:party/initial_groups/
    deploy_cells/回合事件)。campaign.go 的 resetBattle() 在 Scenario 路徑空字串時
    會 fallback 到 assets/scenarios/ch01.json——這對 chapter 2-30 是錯的(ch01 的
    initial_groups=[1,2,10,11] 只適用 map0,套到別張圖會把不在該清單的分組錯誤設成
    OnField=false,主角隊 Party/DeployCells 也是 ch01 專屬)。这是刻意保留給下一輪
    的已知缺口,不在本工具動手範圍(worklist 指示只能動 gen_campaign.py 與
    campaign_full.json 兩個檔案,不能新增 29 個 scenario stub)。
  - 不做主角隊招募(莉希亞/鐵諾/瑪琳…doc28 第5欄)狀態機,亦屬下一輪。

用法:
    python3 tools/gen_campaign.py
輸出:
    remake/assets/scenarios/campaign_full.json
    + 終端列印驗證結果(節點數/轉場完整性/資產檔案存在性)
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


def goal_sentence(row: dict) -> str:
    win = row["win"] if row["win"] and row["win"] != "—" else "敵全滅"
    text = f"目標:{win}。"
    if row["guard"] and row["guard"] != "—":
        text += f"務必保護「{row['guard']}」存活。"
    return text


def build_campaign(rows: list[dict], shops_by_ch: dict[int, list[dict]]) -> dict:
    nodes: dict[str, dict] = {}
    flags: dict[str, bool] = {}

    for row in rows:
        c = row["c"]
        cid = f"{c:02d}"
        map_idx = c - 1  # 順序對應(見檔頭說明);ch1(c=1)→map0,與唯一已驗證資料點一致
        map_dir = f"assets/maps/map{map_idx}"
        units_path = f"assets/maps/map{map_idx}/map{map_idx}_units.json"

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

    return {
        "title": "炎龍騎士團2 — 全 30 章線性 campaign(自動生成)",
        "start": "story_ch01",
        "flags": flags,
        "nodes": nodes,
    }


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
    campaign = build_campaign(rows, shops_by_ch)

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
