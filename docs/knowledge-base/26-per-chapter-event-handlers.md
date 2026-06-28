# 26 — 逐關戰場事件 handler 細節與腳本化對照(供 remake 去 hardcoding)

> doc 25 證實 FD2 戰場事件是「每章一個編進 EXE 的 C handler 函式」(章節跳表 `0x51b19`)。
> 本篇**逐關挖完 18 個特殊 handler 的「條件→動作」**,並設計成資料驅動腳本,讓 remake 不必照抄硬編碼。
> 方法:`tools/event_handler_dump.py`(遞迴反組譯單一 handler + 標註事件原語);機器可讀結果在 [`docs/data/battle_events.json`](../data/battle_events.json)。
> 標 **[驗]**(disasm 直證)/ **[推]**(語意推定)。

## 1. 事件原語(handler 的「指令集」)

每個 handler 都由這幾個原語組成,正好是 remake 腳本 DSL 的詞彙:

| 原語(EXE) | 語意 | DSL 對應 |
|---|---|---|
| `0x3453e(idx)` | 查單位 #idx 狀態:`[0x53a45 單位陣列][idx+5] & 1` | `unit_state(idx)` [驗];bit0 語意=陣亡/到達 [推] |
| 迴圈 `for idx in a..b: 0x3453e` | 查一段單位群 | `units_in_range(a,b)` [驗] |
| `cmp [0x53bef], N` | 比較回合/進度計數 | `turn >= N` [推] |
| `mov [0x53ecc], 1` | 觸發中途事件(→ 戰役迴圈進世界地圖/中場) | `do: story_event` [驗] |
| `mov [0x53ecc], 2` | 達成(特殊)勝利條件 | `do: victory` [驗] |
| `call 0x15f84(…,資源)` | 繪事件全螢幕畫面 | `do: show_scene(res)` [驗] |
| default 尾段 `0x2067e` | 遍歷單位陣列做標準勝敗(殲滅即勝) | `default_win: annihilate` [驗] |

> 單位索引是**戰場單位陣列的全域 index**(我方 + 敵方 + NPC,每單位 0x50B);小 idx(0–25)多為我方、較大(0x30+)多為敵方/NPC [推]。對應到角色名需配合各章 roster(`extracted/maps/maps_metadata.json`)。

## 2. 全 30 章 handler 對照表 [驗]

`D` = default `0x205b4`(11 章共用,純殲滅即勝)。單位以全域 idx 標(十進位)。

| 章 | handler | 觸發條件 | 結果碼 | 繪圖 | 備註 |
|---|---|---|---|---|---|
| 0,2,3,4,5,6,7,8,10,13,23 | `0x205b4` **D** | 遍歷單位陣列雙方存活 | 標準(2 勝 / 0 續) | — | 一般殲滅戰 |
| 1 | `0x206c5` | 單位群 5–10 狀態 | 1 | — | |
| 9 | `0x20707` | 單位 50、51 | 1 | — | |
| 11 | `0x2073d` | 單位 14 | 1 | — | |
| 12 | `0x20765` | 單位群(<12)+ 單位 48、59 | 1 | ✓ | 多段事件 |
| 14 | `0x20822` | 單位 64 | 1 | — | |
| 15 | `0x2084a` | 單位 65 | 1 | — | |
| 16 | `0x20872` | 單位 52 + 動作 `0x33499` | 1 | ✓ | 含特殊動作 |
| 17 | `0x208cf` | 單位 16、17 → **1**;單位 52 → **2** | 1 / 2 | — | 擊敗 #52 = 勝利 |
| 18 | `0x20926` | 回合 ≥6 + 單位 64 | 1 | — | 回合觸發 |
| 19 | `0x20957` | 單位群(<46)→ 1;單位 52 → **2** | 1 / 2 | ✓ | |
| 20 | `0x20a51` | 單位 16、17 | 1 | — | |
| 21,26,27 | `0x20a87` | 單位(迴圈群) | 1 | — | 三章共用 |
| 22 | `0x20aaf` | 單位 16、17 → 1;單位 18 → **2** | 1 / 2 | — | 擊敗 #18 = 勝利 |
| 24 | `0x20b14` | 單位 16 | 1 | — | |
| 25 | `0x20b3c` | 單位(兩個迴圈群) | 1 | — | |
| 28 | `0x20b72` | 單位 → **2**;單位 40 → 1 | 1 / 2 | ✓ | 結局關 |
| 29 | `0x20bf5` | 單位 20 → **2**;單位 → 1 | 1 / 2 | ✓ | 結局關 |

**讀法**:特殊章的共通結構 =「查特定單位(或單位群)狀態 → 若成立則觸發劇情事件(碼 1,戰役迴圈帶你去世界地圖/中場播劇情)或判定特殊勝利(碼 2)」;未觸發則回落到標準殲滅判定。
有 `碼 2` 的章(17/19/22/28/29)是**有特殊勝利條件的關**(擊敗特定 Boss/目標即勝,不必全殲)。

## 3. 範例:章 17 handler `0x208cf` 反組譯 [驗]

```
0x208db push #(隱含); call 0x3453e   ; 查某單位
0x208e7 push 0x10; call 0x3453e      ; 查單位 16
0x208f5 push 0x11; call 0x3453e      ; 查單位 17
0x20903 mov [0x53ecc],1              ;★ → 觸發中途事件
0x2090d push 0x34; call 0x3453e      ; 查單位 52(Boss)
0x2091b mov [0x53ecc],2              ;★ → 勝利
```
即章 17 規則:**「單位 16/17 相關 → 播事件;擊敗單位 52(Boss)→ 勝利」**。

## 4. 提議的 remake 腳本 schema(取代硬編碼)

把上表表達成 campaign 的每章事件規則(對映 doc 19 腳本系統 + ScenarioRunner):

```jsonc
// campaign.chapters[17].battle_events
{
  "default_win": "annihilate",          // 無事件觸發 → 標準殲滅判定(對應 default handler)
  "events": [
    { "when": { "unit_state": [16, 17] }, "do": "story_event" },   // [0x53ecc]=1
    { "when": { "unit_state": [52] },      "do": "victory",        // [0x53ecc]=2
      "scene": null }
  ]
}
```
- `when.unit_state:[…]` / `units_in_range:[a,b]` / `turn>=N` ← 對應原語
- `do: story_event | victory | show_scene` ← 對應 `[0x53ecc]` 與 `0x15f84`
- 11 個 default 章 → 直接 `{"default_win":"annihilate","events":[]}`,零工作量
- 18 個特殊章 → 用上表填 `events`,**逐關資料化,無一行 hardcode**

機器可讀骨架已生成:[`docs/data/battle_events.json`](../data/battle_events.json)(30 章,各含 handler/trigger_units/result_codes/draw_scene/action_fns),remake 可直接讀進 ScenarioRunner 當初始資料,再補劇情文字(FDTXT,doc 09)與場景資源。

## 5. 對重製流程的銜接

1. `battle_events.json`(本篇)→ 每關「勝利/事件條件」骨架
2. + `maps_metadata.json`(doc 03)→ 單位 idx 對應實際角色/敵人 + 出場位置
3. + 章節文本 FDTXT(doc 09)→ 事件觸發時播的對白
4. + 章節跳表 `0x51d71`/`0x51de9`(doc 23)→ 戰前/戰後劇情
→ 組成完整資料驅動 campaign,ScenarioRunner 解釋執行(doc 19/21),**事件邏輯全在資料,引擎不為任何一關寫死分支**。

## 6. 受阻 / 待驗(誠實標註)

- **[推]** 單位狀態 `byte[+5] bit0` 確切語意(陣亡 / 抵達指定格 / 某 buff)未隔離驗證——影響 `unit_state` 的條件語意。建議重製時以「該單位是否陣亡」為起點,再對齊原版實機行為。
- **[阻]** 迴圈查的單位群(章 1/12/19/21/25)精確 idx 範圍見逐指令 dump(章1=5–10、章12=<12、章19=<46);`battle_events.json` 對迴圈索引只標記 `action_fns` 含 `0x3453e`,未展開範圍(迴圈變數非立即數)。
- **[阻]** 各章特殊動作函式(章16 `0x33499` 等)是增援 / 對話 / 特效未逐一反組譯——逐關重製時按需挖。
- **[阻]** 單位全域 idx → 角色/敵人名 對應,需逐章配 roster 比對。
- **[推]** `[0x53bef]` = 回合 / 進度計數(章12 cmp5、章18 cmp6);與 `[0x53ec8]`(doc 25,clamp 99)的關係待釐清。

> 相關:doc 25(事件系統架構)· doc 24(戰役迴圈 [0x53ecc] 狀態機)· doc 19(腳本系統)· doc 09(劇情)· doc 03(單位結構/roster)。工具:`tools/event_handler_dump.py`;資料:`docs/data/battle_events.json`。
