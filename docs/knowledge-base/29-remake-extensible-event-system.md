# 29 — remake 可擴展事件系統規劃(從封閉 handler 到開放 DSL + 事件控制碼)

> **狀態：歷史設計草案，不是原版事件 ABI 或目前 runtime 完成度規格。**
> 本文保留 DSL 方向供擴充設計參考；原版忠實模式必須以
> [`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)、
> [`56-fd2-remake-sdd.md`](56-fd2-remake-sdd.md) 與
> [`91-worklist.md`](91-worklist.md) 檔首的有效佇列為準；`42` 只保留歷史落差快照。
> 早期「handler 只管勝負、動作全在 FDFIELD」已被逐章 post/pre handler 的
> SPAWN/JOIN/PAN/ACT/dialogue/LOADCH call trace 推翻；不得再拿本草案替代
> hard-coded handler 的逐章轉錄。
>
> 已證實的資料邊界只有：FDFIELD 控制段保存 bounded turn-event records，
> roster/group 與出場位置；event id、camp byte及其 caller-specific 行為仍須
> 配合 EXE handler 與 runtime context 解讀，不能由欄位外觀全域命名。
> remake 目標:把整套機制**資料化 + 開放**,新增事件 = 寫資料(JSON + 文本內嵌**事件控制碼**),引擎不為任何一關寫死分支。
> 承接:doc 19(腳本系統)· doc 25/26(原版事件)· doc 28(關卡目標)· doc 14(原版文本控制碼)。資料:`tools/parse_field.py` 解 turn_events/group。

## 1. 設計原則

1. **原版可表達（目標，尚未證明）**:逐章 handler 轉錄完成後，原版 30 關應能由資料化 DSL 描述；目前不能以攻略目標表或本文件的 schema 宣稱表達力已完整驗證。
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
| `unit_alive(id)` / `unit_dead(id)` | remake 高階條件；原版 record `+5` bit 的意義依 caller、camp 與 lifecycle 而異，不能全域等同存活/死亡 | **remake projection；native 待逐 caller 驗證** |
| `roster_has(id)` | 我方隊伍有某角色(原版 `0x33499`) | **原版** doc 26 |
| `turn >= N` / `turn == N` | 回合數(原版 `[0x53bef]`) | **原版** doc 26 |
| `units_in_range(a,b, camp)` | 某陣營某群單位狀態 | **原版** |
| `unit_at(id, x,y)` / `unit_in_region(id, region)` | 單位在某格/某區 | **擴充**(護送到達) |
| `hp_below(id, pct)` | HP 低於% | **擴充**(瀕死觸發) |
| `flag(name)` / `counter(name) >= N` | 自訂旗標/計數器 | **擴充**(跨關劇情) |
| `item_owned(id)` / `chest_opened(id)` | 持有道具/開過寶箱 | **擴充** |
| `and[…] / or[…] / not(…)` | 邏輯組合 | **擴充**(條件樹) |

> 本表是目標schema，不是native enum。原版有哪些等價條件必須逐caller證明；
> `ConditionRegistry`目前也尚未存在於production Go code，不得把「註冊一個
> func」寫成現行擴充接口。

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
- Registry是提案中的擴充層；目前production使用typed `battle.Event/Action`
  與campaign beats，新增條件／動作仍可能需要schema、validation、runtime與
  tests，不是「Go一行即可」。
- `flags`/`counters` 存進存檔(自有格式,doc 27)→ 跨關劇情狀態。
- **目標**是核心迴圈不硬編具體關卡；目前仍有chapter-specific adapters、
  screenshot hooks與未閉合handler，不能宣稱所有差異都已移到資料。

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

→ 目標是讓原版 30 關在逐章 evidence closure 後由本 DSL 重現，同一引擎也能跑
**完全自創的戰役**。目前只有部分可重播 campaign/event/beat primitives，
不能把這個設計目標寫成已完成的忠實模式。

## 11. 第1章「初試身手」早期 DSL 草案（非原版完整轉錄）

以下 JSONC 是 schema 示意，不是目前 `ch01.json`，也沒有覆蓋原版 ch00
pre/post handler、acting、camera、dialogue 與 persistence 全序列。
原版 map0 的 `turn_events`(T3友/T4敵/T5敵/T6友)可作為候選輸入之一；
**主角隊應為索爾／亞雷斯／悠妮／蓋亞（不是妮雅）**。其進場 owner 與
presentation 必須依 handler call trace，不得只因不在 FDFIELD roster 就推定為
`on_battle_start/spawn_march`。
哈諾/哈瓦特雖在 roster(group 3/7)但 T3 才登場;敵援軍/海盜頭目/警備隊按回合各從角落出。

```jsonc
// remake/assets/scenarios/ch01.json
{
  "chapter": 1, "name": "初試身手", "map": 0,
  "win":  { "all_enemies_dead": true },
  "lose": { "unit_dead": "sol" },
  "deploy": { "cells": [[7,20],[10,21],[8,22],[11,23]] },   // 原版 positions 肖像0 的4格
  "events": [
    { "id":"opening", "trigger":"on_battle_start", "once":true, "do":[
        // 主角隊從戰場下緣行軍到部署格(進場演出)→ 對話
        {"spawn_march": {"units":["sol","ares","yuni","gaia"], "from":"edge_bottom", "to":"deploy"}},
        {"dialogue":"ch1_opening"}
    ]},
    { "id":"hano_join", "trigger":"on_turn_end", "when":{"turn==":3}, "once":true, "do":[
        {"spawn_wave":{"units":["hawat","hano"], "camp":"ally", "at":[[11,11],[11,11]]}}, // 從中央房子
        {"dialogue":"ch1_hano_join"}, {"recruit":"hano"}
    ]},
    { "id":"enemy_reinf", "trigger":"on_turn_end", "when":{"turn==":4}, "once":true, "do":[
        {"spawn_wave":{"group":4, "camp":"enemy", "at":"corner_br"}}          // 敵援軍 右下
    ]},
    { "id":"pirate_boss", "trigger":"on_turn_end", "when":{"turn==":5}, "once":true, "do":[
        {"spawn_wave":{"group":5, "camp":"enemy", "at":"corner_bl"}},         // 海盜頭目+屬下 左下
        {"dialogue":"ch1_pirate"}
    ]},
    { "id":"guard_reinf", "trigger":"on_turn_end", "when":{"turn==":6}, "once":true, "do":[
        {"spawn_wave":{"group":6, "camp":"ally", "at":"corner_tr", "act_immediately":true}} // 警備隊 右上,立即行動
    ]},
    { "id":"hawat_berserk", "trigger":"on_unit_death", "when":{"unit_dead":"hano"}, "do":[
        {"set_ai":{"unit":"hawat","mode":"berserk"}}, {"dialogue":"ch1_hawat_berserk"}
    ]}
  ]
}
```

本實例**提議三個可擴充動作**；production目前沒有ActionRegistry，且這些
名稱沒有逐項native provenance：
`spawn_march`(從邊緣行軍進場演出)· `act_immediately`(增援當回合可動,對應青衫「立即行動」)· `set_ai:berserk`(哈諾死→哈瓦特暴走)。
→ 它們只能作未來設計候選；原版進場／暴走與remake現行ch01仍須各自以
handler、FDFIELD、runtime及DOSBox證據驗收。

其中目前 `event.go` 的 `set_ai` 只留下可編輯事件標記，沒有任何人工智慧
規劃器讀取；`berserk` 也不是已證實的原版 `record+0x34` 低四位名稱。
因此現行 ch01 可以觸發對白，卻不能宣稱已改變哈瓦特的行動決策。

> 相關:doc 19(腳本系統設計)· doc 25/26(原版事件)· doc 28(原版關卡目標)· doc 14(原版文本控制碼)· doc 21(Go/Ebiten 架構)。
