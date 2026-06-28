# 19 — 劇本 / 關卡腳本系統設計

> 設計一套 FD2 的**劇本腳本系統**:把遊戲從原版「固定 33 關線性流程」進化成**可分支的關卡圖**——
> 關卡接關卡、中間進商店、劇情過場、**戰敗走不同路線**、甚至自創新戰場。這是把 `17`(擴充評估)、
> `18`(文字)、地圖/頭像/音樂等已 RE 的零件**串成一個可玩流程**的「黏著層」。本篇是設計規格。

## 核心觀念:把「線性 33 關」變成「節點圖」

原版流程其實是一條鏈:`戰役0 → 劇情 → 商店 → 戰役1 → …`,寫死在程式裡。
重製改成 **有向圖(graph)**:**節點(node)** = 一個遊戲段落,**轉場(transition)** = 依結果/條件決定下一個節點。

```
        ┌─勝─▶ 戰役2 ─▶ 商店B ─▶ …
序章戰役 ┤
        └─敗─▶ 撤退戰(敗北路線) ─▶ 劇情(被俘?) ─▶ 戰役2'
```

- **原版模式**:節點圖就是一條直線(忠實還原 33 關),由原始資料自動生成。
- **擴充模式**:作者在圖上加分支、敗北路線、岔路選擇、新戰場——不用改引擎,只改腳本資料。

## 節點型別(node types)

| 型別 | 做什麼 | 結果(outcome)→ 轉場 |
|---|---|---|
| `battle` | 一場戰棋(地圖+出場+勝敗條件) | `win` / `lose`(各接不同節點) |
| `story` | 對話/過場(UTF-8 腳本+頭像+音樂) | `next` |
| `shop` | 武器/道具/神秘商店 | `next`(逛完離開) |
| `choice` | 玩家選擇 | `opt0` / `opt1` / …(分支) |
| `event` | 設旗標/給道具/收夥伴/轉職點 | `next` |
| `ending` | 結局(可多結局) | — |

## 戰役節點(battle)定義

引用已 RE 的資料(各見章節),不重造:

```json
{
  "id": "stage_prologue",
  "type": "battle",
  "map": { "field": "FDFIELD_000", "tileset": "FDSHAP_000" },   // 01§8 / parse_field
  "deploy_max": 4,                                              // FDFIELD 控制
  "units": "<from parse_field maps_metadata[0]>",               // 敵我 roster + 座標
  "victory": { "type": "defeat_all" },                          // 見下「條件」
  "defeat":  { "type": "all_own_dead" },
  "bgm": "FDMUS_001",                                           // 12 / OGG
  "pre":  "story_prologue_intro",     // 戰前劇情節點 id(FDTXT 章節)
  "on_win":  "shop_rodo",            // 勝 → 下一節點
  "on_lose": "gameover"              // 敗 → 預設 game over;可改成敗北路線節點
}
```

**勝利 / 失敗條件**(可組合,擴充原版只有「殲滅」的單一條件):
- 勝:`defeat_all` / `defeat_boss(unitId)` / `survive_turns(N)` / `reach(x,y)` / `protect(unitId, turns)` / `escape`。
- 敗:`all_own_dead` / `lose_unit(unitId)` / `turn_limit(N)` / `protected_died`。
→ 多樣條件本身就讓「同一張地圖玩出不同戰法」,是擴充的第一層。

## 對話 / 商店 / 事件節點

- `story`:引用 `18` 規劃的 **UTF-8 script.json**(開框/說話者/頭像/翻頁)。
- `shop`:引用 EXE 商店表(`03`)或自訂品項清單(武器/道具/神秘商店 + 價格)。
- `event`:`set_flag` / `give_item` / `recruit(portraitId)` / `unlock_class` / `gold(±N)`。

## 分支與旗標(branching)

**旗標 / 變數**是分支的關鍵——原版幾乎沒有,重製可加:

```json
"flags": { "saved_teno": false, "route": "sea" }
```
- 轉場可帶**條件**:`{ "to": "stage_secret", "if": "flags.saved_teno == true" }`。
- **敗北路線**(使用者想要的):`battle.on_lose` 不一定是 game over,可指向「撤退戰 / 被俘劇情 / 替代戰役」,
  之後再 `merge` 回主線或走出獨立支線。
- **岔路選擇**:`choice` 節點(「走山路 / 走海路」)→ 不同戰場序列。
- **多結局**:依累積旗標走不同 `ending`。

## 原版 → 腳本系統(自動生成 + 可編輯)

1. **抽資料**(已具備工具):
   - 戰役:`parse_field.py` → 33 圖 metadata(出場/條件/寶箱/roster)。
   - 劇情:`decode_story_text.py --script-json`(規劃中,`18`)→ 35 章 UTF-8 script。
   - 商店:EXE 商店表(`03`)→ 各章品項。
   - 音樂:scene→track(`12`)+ OGG(`16`)。
2. **生成預設 campaign.json**:把上面串成一條線性圖(= 忠實原版 33 關),即「原版模式」。
3. **作者編輯**:在 campaign.json 上加分支 / 敗北路線 / 新戰場 / 新商店 / 旗標——擴充模式。

## 引擎執行(ScenarioRunner)

一個狀態機驅動整個流程:

```
current = campaign.start
loop:
  node = campaign[current]
  outcome = run(node)            # battle→打一場;story→播對白;shop→開店;choice→等選擇
  current = resolve_transition(node, outcome, flags)   # 依結果+旗標決定下一個
  if node.type == "ending": break
存檔 = { current 節點 id, flags, 隊伍狀態, 物品 }       # 取代 FD2.SAV 的自有存檔
```

- 戰鬥子系統:用已 RE 的單位結構(`03`)、AI(`11`)、選單(`13`)、地圖(`01`§8)。
- 文字子系統:`18` 的 TTF + script.json。
- 音訊:`16` 的 OGG(MT-32)/ SoundFont。

## 為什麼這樣設計

- **資料驅動**:加關卡 / 改路線 = 編 JSON,不動引擎(呼應 `17`)。
- **圖 > 線性**:天然支援敗北路線、岔路、多結局——原版做不到的「不一樣」。
- **忠實 + 擴充並存**:預設圖 = 原版 33 關;擴充圖 = 自由創作。兩者同引擎、可切換(對應音源/字型的雙模式哲學)。
- **零件已就位**:地圖、敵我、頭像、對話、商店、音樂都已 RE 成可讀資料,腳本系統只是把它們排成流程。

## 實作階段(建議)
1. 定 `campaign.json` schema(本篇)+ 寫一個範例(序章→商店→第2章 + 一條敗北路線)。見 `docs/data/campaign_sample.json`。
2. `parse_field` / `decode_story_text --script-json` / 商店表 → 自動生成「原版線性 campaign」。
3. 引擎 `ScenarioRunner` 狀態機 + 各節點 runner(先 story/shop,再 battle)。
4. 旗標系統 + 條件轉場 → 開放分支 / 敗北路線。
5. 簡易關卡 / 劇本編輯器(吃同一份 schema),讓加關卡 = 填表。

## 風險 / 注意
- **存檔**:存「節點 id + 旗標 + 隊伍」,別存死進度索引,分支才存得住。
- **平衡**:新戰場 / 敗北路線的敵人強度要配合進度(用 `03` 成長曲線當基準)。
- **保全原味**:預設 campaign 必須 1:1 還原原版流程;擴充走另一份 campaign 檔。
- **資料引用 vs 內含**:campaign 引用原版資源 id(玩家自備原版);新增劇情/戰場是衍生創作。
