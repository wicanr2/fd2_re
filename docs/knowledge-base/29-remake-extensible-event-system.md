# 29 — remake 可擴展事件系統規劃(從封閉 handler 到開放 DSL + 事件控制碼)

> 原版 FD2 的事件是**每章一個編進 EXE 的 C handler**(doc 25/26):條件(`unit_state`/`roster_has`/回合)→ 設碼。優點是省事,**缺點是封閉**——改一個事件就要改程式、重編譯,玩家無從擴充。
> remake 的目標:把同一套機制**資料化 + 開放**,讓「新增事件」變成寫資料(JSON/腳本 + 文本內嵌**事件控制碼**),引擎不為任何一關寫死分支。本篇規劃這套系統。
> 承接:doc 19(腳本系統)· doc 26(原版事件原語)· doc 28(關卡目標)· doc 14(原版文本控制碼)。

## 1. 設計原則

1. **原版可表達**:原版 30 關(doc 28)能用這套 DSL 完整描述 → 證明表達力足夠。
2. **開放擴充**:新增條件/動作/事件,只加資料或註冊一個 handler,**不改既有關卡、不改核心迴圈**。
3. **三層解耦**:`觸發時機(trigger) → 條件(when) → 動作(do)`,各自可獨立擴充。
4. **資料優先,腳本兜底**:90% 事件用宣告式 JSON;少數複雜邏輯掛腳本(Go 註冊函式 / 嵌入式 script)。
5. **文本與事件交織**:對話流程中可內嵌事件(**事件控制碼**),不必把劇情切成碎片。

## 2. 事件模型:trigger → when → do

```
            ┌─────────────────────────────────────────────┐
   遊戲迴圈  │  每個 tick / 關鍵時點 觸發對應 trigger        │
            └───────────────┬─────────────────────────────┘
                            ▼
   EventSystem: 對該 trigger 下所有 event 規則,逐條評估
      ┌──────────────┐   滿足   ┌──────────────┐
      │ when(條件樹)  │ ───────▶ │ do(動作序列)  │
      │ AND/OR/NOT    │          │ 依序執行       │
      └──────────────┘          └──────────────┘
                            │
                            ▼  (once? / cooldown? / 旗標標記已觸發)
```

**trigger(觸發時機)**——可擴充列舉:
`on_battle_start` · `on_turn_start` · `on_turn_end`(= 原版 `[0x53bef]` inc 點:我方全動+敵方AI全動)· `on_unit_death` · `on_unit_move` · `on_tile_enter` · `on_dialogue_line` · `on_item_used` · `on_shop_enter` · `on_chapter_clear`。

## 3. 條件原語 `when`(原版已有 + remake 擴充)

| 條件 | 語意 | 來源 |
|---|---|---|
| `unit_alive(id)` / `unit_dead(id)` | 單位存活/陣亡(原版 `unit_state` bit0) | **原版** doc 26 |
| `roster_has(id)` | 我方隊伍有某角色(原版 `0x33499`) | **原版** doc 26 |
| `turn >= N` / `turn == N` | 回合數(原版 `[0x53bef]`) | **原版** doc 26 |
| `units_in_range(a,b, camp)` | 某陣營某群單位狀態 | **原版** |
| `unit_at(id, x,y)` / `unit_in_region(id, region)` | 單位在某格/某區 | **擴充**(護送到達) |
| `hp_below(id, pct)` | HP 低於% | **擴充**(瀕死觸發) |
| `flag(name)` / `counter(name) >= N` | 自訂旗標/計數器 | **擴充**(跨關劇情) |
| `item_owned(id)` / `chest_opened(id)` | 持有道具/開過寶箱 | **擴充** |
| `and[…] / or[…] / not(…)` | 邏輯組合 | **擴充**(條件樹) |

> 原版只有前四種(且寫死在 C);remake 把它們資料化,再往上疊組合與新條件。新增一個條件 = 在 `ConditionRegistry` 註冊一個 `func(ctx) bool`。

## 4. 動作原語 `do`(原版隱含 + remake 擴充)

| 動作 | 語意 | 來源 |
|---|---|---|
| `story_event` / `dialogue(script_id)` | 播劇情/對話 | **原版** 碼1→劇情 |
| `show_scene(res)` | 全螢幕過場圖(原版 `0x15f84`) | **原版** doc 26 |
| `spawn(unit, x,y)` / `spawn_wave(group)` | 增援登場 | **原版** 連鎖事件 |
| `win` / `lose` / `next_chapter` | 結束流程(原版碼2/戰後跳表) | **原版** |
| `set_flag(name)` / `inc_counter(name)` | 設旗標/計數 | **擴充** |
| `give_item(id)` / `recruit(id)` | 給道具/入隊 | **擴充**(原版加入邏輯資料化) |
| `move_unit(id, path)` / `transform(id, into)` | 強制移動/變身 | **擴充**(劇情演出) |
| `play_music(track)` / `play_sfx` / `camera(x,y)` / `weather(type)` | 演出控制 | **擴充** |
| `branch(choice → [events])` | 玩家選擇分支 | **擴充**(擺脫固定路線) |
| `call(event_id)` | 呼叫共用事件(子程序) | **擴充**(複用) |

## 5. 事件控制碼(回應「增加事件控制碼」)

原版文本控制碼(doc 14)只管**渲染**(開框 `0xFFEF`、換行 `0xFFFE`、翻頁 `0xFFFD`、頭像)。remake 在 UTF-8 文本裡擴充一套 **`{{…}}` 事件控制碼**,讓劇情流程中直接觸發遊戲事件——這是「對話即腳本」的關鍵:

```
索爾:這座城就交給你們了……{{flag:set:city_handed}}
{{portrait:hanno:angry}}哈諾:你說什麼!?{{sfx:shock}}
{{wait}}……{{music:tension}}
{{branch:
   "追上去"  -> [ spawn:hanno@10,4, dialogue:chase ]
   "留下防守" -> [ set_flag:stay, spawn_wave:defenders ]
}}
{{call:boss_intro}}
```

**控制碼分兩層(向下相容原版)**:
| 層 | 標記 | 範例 | 對應 |
|---|---|---|---|
| 渲染(繼承原版) | 內建 token | 換行/翻頁/頭像/`{{color:red}}`/`{{speed:slow}}`/`{{shake}}` | doc 14 + 擴充 |
| 事件(新增) | `{{verb:args}}` | `{{flag:set:x}}` `{{spawn:goblin@12,5}}` `{{give:item:42}}` `{{wait}}` `{{branch:…}}` `{{call:id}}` `{{win}}` | §4 動作 |

解析:文本層遇 `{{…}}` → 暫停渲染 → 丟給 EventSystem 執行該動作 → 回來續播。等於把 §4 的 `do` 動作**內嵌進對話時間軸**。
編碼空間:原版用 `0xFFxx` 保留碼(有限);remake 走 UTF-8 純文字標記(`{{}}`),**無數量上限、人類可讀、可版控 diff**。

## 6. 完整 event schema(campaign 節點內)

```jsonc
// campaign.chapters[N].events[]
{
  "id": "ch30_water_god",
  "trigger": "on_turn_end",
  "when": { "and": [ {"turn>=": 4}, {"unit_dead": "water_god"} ] },
  "do": [
    { "spawn_wave": "wind_god_escort" },
    { "dialogue": "空魔神:愚蠢的人類……" },
    { "play_music": "boss2" }
  ],
  "once": true                       // 只觸發一次(用旗標記)
}
```
- 原版第 30 章魔神連鎖(doc 28)= 4 條這種規則(地→水→風→火),純資料。
- 自訂新事件 = 在 `events[]` 加一條,或在對話裡寫 `{{…}}`,**零引擎改動**。

## 7. 擴展範例:做一個原版沒有的事件

**「敗中求生」**——某關我方主將瀕死時,觸發隱藏援軍 + 分支:
```jsonc
{ "trigger":"on_turn_start",
  "when": {"and":[ {"hp_below":["sol",25]}, {"not":{"flag":"rescue_used"}} ]},
  "do": [
    {"set_flag":"rescue_used"},
    {"dialogue":"神秘騎士:撐住!{{spawn:mystery_knight@5,5}}{{music:hope}}"},
    {"branch": { "接受援助":[{"recruit":"mystery_knight"}],
                 "婉拒":   [{"give_item":"elixir"}] }}
  ]}
```
原版做不到(要改 EXE);remake 只是多一條 JSON + 一段帶控制碼的對話。

## 8. 引擎架構(Go/Ebiten,接 doc 21)

```
ScenarioRunner (campaign 流程,doc 19)
   └─ BattleScene
        ├─ EventSystem
        │    ├─ ConditionRegistry  (name → func(ctx) bool)   ← 擴充點
        │    ├─ ActionRegistry     (name → func(ctx) error)  ← 擴充點
        │    └─ rules: []Event (從 campaign.json 載入)
        ├─ DialoguePlayer (解析 {{}} 事件控制碼 → 丟 EventSystem)
        └─ BattleState (units/turn/flags/counters)
```
- 新增條件/動作 = 往 Registry 註冊一個函式(Go 一行);資料端立即可用。
- `flags`/`counters` 存進存檔(自有格式,doc 27)→ 跨關劇情狀態。
- **核心迴圈完全不認得任何具體關卡**——所有關卡差異都在資料。

## 9. 工具鏈(讓非程式者也能做關卡)

- `tools/`:原版 `battle_events.json`(doc 26)+ 關卡目標(doc 28)→ 自動生成原版 30 關 campaign(驗證 DSL 表達力)。
- 未來:視覺化 campaign 編輯器(擺單位、畫觸發區、連事件節點)→ 匯出 campaign.json。
- 文本工具:`encode_text.py`(doc 08)可回寫;事件控制碼 `{{}}` 是純文字,任何編輯器可改。

## 10. 與原版的關係

| | 原版 FD2 | remake |
|---|---|---|
| 事件載體 | 編進 EXE 的 C handler(0x51b19) | campaign.json + 文本 `{{}}` 控制碼 |
| 條件 | 寫死 unit_state/roster/turn | Registry,可組合可擴充 |
| 新增事件 | 改程式 + 重編 | 加資料,零引擎改動 |
| 分支/多結局 | 幾乎沒有(33 固定路線) | branch / flag 任意分支 |
| 編輯門檻 | 工程師 | 資料/腳本,甚至玩家 |

→ 原版 30 關用本 DSL 重現(忠實模式),同一引擎也能跑**完全自創的戰役**——這就是 remake 相對原版的核心增值,呼應 worklist「擺脫原版固定 33 路線」。

> 相關:doc 19(腳本系統設計)· doc 26(原版事件原語=DSL 詞彙來源)· doc 28(原版關卡目標)· doc 14(原版文本控制碼)· doc 21(Go/Ebiten 架構)。
