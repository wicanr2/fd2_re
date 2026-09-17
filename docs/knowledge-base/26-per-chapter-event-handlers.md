# 26 — 逐關戰場事件 handler 細節與腳本化對照(供 remake 去 hardcoding)

> doc 25 證實 FD2 戰場事件是「每章一個編進 EXE 的 C handler 函式」(章節跳表 `0x51b19`)。
> 本篇匯出 30 章 battle-end predicate/result metadata；其中18個非default
> handler有條件骨架。這**不是完整動作、turn-event、postbattle或campaign
> route dump**，不得直接當成忠實runtime腳本。
> 方法:`tools/event_handler_dump.py`(遞迴反組譯單一 handler + 標註事件原語);機器可讀結果在 [`docs/data/battle_events.json`](../data/battle_events.json)。
> 標 **[驗]**(disasm 直證)/ **[推]**(語意推定)。

> ⚠ **澄清(2026-07,gen_campaign v3 誤用後補記,防後人再誤用)**:`battle_events.json` 是**本篇 handler 的 raw result metadata**(is_default/result_codes/trigger_units_flag/extra_conditions/action_fns),**不是「回合增援事件 dump」**——§1 已明講「handler 不含任何動作函式」,少數章只有 `roster_has`/`raw_record_byte5_bit0` 條件字串,完全沒有 turn/group 欄位。**真正的回合制增援資料**(第幾回合、敵/友/特殊、全域 event_id)在 **FDFIELD.DAT 控制段**(`tools/parse_field.py` 的 `turn_events`,doc 29 §11 記載),已 dump 成 [`docs/data/turn_events.json`](../data/turn_events.json)。event_id→group 的消費鏈已由 `25` §6.1 閉合：`0x1a813` 依 turn/camp 篩選，呼叫 `0x51b91` 全域 90-entry 表中的 FDFIELD 子集合 0..57，再由 handler 呼叫 spawn 原語；58..89 另有玩家操作、格子互動與單位行動等通用 consumers，不能再稱整張表只有 58 項。`0x22e5c` 是固定資源 #79 呈現路徑，與 turn_events 無關；「章1專屬中場」舊名稱因缺少章節讀取及原版執行期證據而撤回。`tools/gen_campaign.py` 舊「RE 撞牆」文字僅保留歷史 provenance，不得當成現況。

## 1. 事件原語(handler 的「指令集」)

每個 handler 都由這幾個原語組成,正好是 remake 腳本 DSL 的詞彙:

| 原語(EXE) | 語意 | DSL 對應 |
|---|---|---|
| `0x3453e(idx)` | **NativeRecordByte5Bit0(idx)**：取第 idx 單位(`[0x53a45] + idx*0x50 + 5`) 的 `byte[+5] & 1` | raw predicate [驗]；高階語意由各 caller branch 另證 |
| 迴圈 `for idx in a..b: 0x3453e` | 查一段單位群的狀態旗標 | `units_in_range(a,b)` [驗] |
| `0x33499(id)` | 線性搜尋我方名冊 `[0x53bf7]`(32 槽×0x50B,計數 `[0x53bfb]`)，比對 `byte[+8]` | `roster_has(raw_identity_key)` [驗]；高階角色名依caller |
| `cmp [0x53bef], N` | 比較raw phase/turn counter | `counter >= N` [驗]；玩家層回合語意依caller |
| `mov [0x53ecc], 1` | 寫 battle-result code 1 | `result_code: 1` [驗]；中途／勝敗語意依外層caller |
| `mov [0x53ecc], 2` | 寫 battle-result code 2 | `result_code: 2` [驗]；中途／勝敗語意依外層caller |
| `call 0x15f84(…,資源)` | 繪事件全螢幕畫面 | `do: show_scene(res)` [驗] |
| default 尾段 `0x2067e` | 共用 battle-end predicate／result path | raw default callee [驗]；「殲滅即勝」需另有caller/gameplay證據 |

**`0x3453e` 全貌(已驗證)**:`idx*4 + idx = idx*5`,再 `<<4` = `idx*0x50`(單位結構大小)+ 基底 + 5 → 取 bit0。
**raw byte 說明(2026-07-27)**：constructor `0x10eed` 寫 `byte[+5]=0`，HP 路徑 `0x1dc61/0x1dd4c` 與 `0x32975` 會寫入 `1`；這些是已觀察的 writer，不能單獨把 bit0 提升成所有 caller 共用的死亡／存活欄位。`0x3453e` 只回傳 `byte[+5] & 1`，bit7 writer 亦另行追蹤。每個 handler 的 `test/jcc` 必須按該 caller 的 raw predicate 解讀。

這個定案亦直接解開兩個劇情/回合 handler：

- **ch01 post `0x22f42`** 掃 slots 5..10：任一 slot 的 raw bit0 非零走對白 #7、無獎勵；全部為零才走對白 #6 並給 item 198。這是該 caller 的 raw branch，不泛化成全域生命欄位。
- **ch03 turn 3 `0x344c2`** 查 slot6：raw bit0 非零直接跳過；為零才生成 group2、移鏡並播對白 #4。對白旁證不能取代 raw branch evidence。

> **範圍更正:**`battle_events.json` 這份匯出目前只記錄到的戰場事件 caller，其 `action_fns`
> 為空；在這個資料集範圍內可說它們只保存「**條件查詢→設碼/可選繪圖**」。這不能外推到
> postbattle/cutscene handlers：ch14/ch15 post 已直接證實含 layout、dialog、acting、sync、JOIN
> 與 set-chapter 原語，這些動作仍須由 editable handler script 保存。增援、對話是否在 handler
> 內執行，必須按每個 caller 的 CFG 判定，不能一概移到世界地圖流程。

> 單位索引是**戰場單位陣列 `[0x53a45]` 的全域 index**(我方 + 敵方 + NPC,每單位 0x50B);對應到角色名需配合各章 roster(`extracted/maps/maps_metadata.json` 的 `units`,含 camp/portrait/race/cls/lv)。一般 normalized scenario 可用具名單位/陣營投影，但凡 handler 直接呼叫 `0x3453e`、`0x233c6` 或以 raw slot 讀寫 record，重製必須保留原始 slot order/數量與 provenance；不能再宣稱所有 trigger 都可脫離原版 idx。

**兩個單位陣列(別混淆)** [驗]:
- **`[0x53a45]` = 戰場全單位陣列**(malloc `0x1e00` = **96 槽** × 0x50B,我方上場+敵方+NPC)→ `NativeRecordByte5Bit0(idx)`(0x3453e)查 raw bit。
- **`[0x53bf7]` = 我方隊伍/角色名冊**(場景載入時 memcpy `0xa00` = **32 槽** × 0x50B,計數 `[0x53bfb]`)→ `roster_has(id)`(0x33499)查它,找 `byte[+8]==id`。
- 即：remake 的 `unit_inactive` alias 只包裝「**場上**某 caller 的 raw bit0 predicate」；`roster_has` 問「**我方隊伍**是否擁有某角色」(招募/角色分支條件)。章 16 即用 `roster_has(0x12=角色18)`。

### 回合計數:`[0x53bef]` 是回合數(非 `[0x53ec8]`)[驗]

修正先前把 `[0x53ec8]` 當回合的說法。逐讀寫點反組譯:

| 變數 | 性質 | 證據 |
|---|---|---|
| **`[0x53bef]` = 回合數** | 戰場開始 `mov 1`(0x2066e,handler prologue 區)→ `inc`(0x1a5b9,回合切換處)→ handler `cmp N` | 章12 `cmp 5`、章18 `cmp 6` = 第 N 回合觸發事件 [驗] |
| `[0x53ec8]` = 累積計數 | `add [v],reg`(加變量,非 +1)+ 戰場每 tick `clamp` 99 + 多處 `reset 0`;跨戰鬥/世界地圖 | 非回合;確切語意(累積量)待定 [推] |

→ DSL 的 `turn>=N` 對應 `[0x53bef]`。

### 單位 0x50B 結構（來源分流勘誤）

舊表把 persistent party constructor 與 FDFIELD group constructor 拼成
單一路徑，並把若干來源 byte 直接當成 runtime offset；該模型已失效。
目前直接指令固定兩條不同來源：

- `0x1088D` 的 party loop 先複製完整 persistent `0x50` record，再由
  position resource 覆寫座標及部分 transient；
- `0x10B4E→0x10C50` 才由 FDFIELD 26-byte row、position resource 與
  EXE constructor tables 建立新的 runtime record。

目前可安全共用的 runtime 邊界如下：

| 偏移 | 內容 | 狀態 |
|---|---|---|
| +0/+1 | live map X/Y；constructor 來源是 position resource，不是 control row `b0/b2` | [驗] |
| +2 | `0x11019` 依 raw key 建立的 process-global FDICON cache slot；CONTINUE 會依 saved runtime order 與 `+7` 重算 | [驗] |
| +3/+4 | live pose/motion；新建 record 初值 0，之後由移動與演出 writer 改寫 | [驗] |
| +5 | **raw mask byte**：已驗證 caller 讀取 `bit0`／`bit7(0x80)`；不得直接命名成死亡、存活或已行動 | [驗] |
| +6 | party constructor 寫 literal 2；FDFIELD constructor 直接複製 row `b0`。只保留 raw selector，不把兩條來源概括成面向 | [驗] |
| +7 | party 路徑保留 persistent raw key；FDFIELD 路徑複製 row `b1`。CONTINUE 以此重建 `+2`，不可稱為 `+2` 的直接複製 | [驗] |
| +8 | FDFIELD constructor 同樣複製 row `b1`；persistent `roster_has` 另以此欄比對角色。跨兩種來源的統一高階語意尚未閉合 | [驗欄位／未知統一語意] |
| +0x1F/+0x20 | raw race/class；由 constructor table 或 persistent record 提供 | [驗] |
| +0x22..+0x27 | 六個獨立 live transient bytes；不可降成單一 buff 狀態 | [驗] |
| +0x31..+0x33 | FDFIELD row `b22..b24` 直拷；party constructor 另將 `+0x31` 設為 `0xFF` | [驗] |
| +0x34..+0x36 | FDFIELD row `b17..b19` 直拷；只有 `+0x34` low nibble 的 dispatch 已閉合 | [驗] |
| +0x3D | FDFIELD row `b2` 由 `0x10FC8` 直拷；它不是 constructor table 寫入 `+0x1F` 的 race | [驗] |
| +0x37..+0x4E | 基礎／有效能力、HP/MP 等 word 與 byte 欄位；目前 typed view 保留已證實 offsets | [部分驗] |

重製可以使用具型別 `Unit`，但 CONTINUE、存檔相容與原生 renderer 需要的
raw 欄位仍必須逐一保存來源與 consumer；不能只對齊一份高階屬性清單就宣稱
等價。現況權威見 SDD 的 current-runtime snapshot／CONTINUE 段落及
[`fd2_current_field_control_ida.txt`](../data/fd2_current_field_control_ida.txt)。

2026-07-30 的完整 `0x10B4E..0x11018` Capstone 重核再修正 source row：
`b2→runtime +0x3D`；`b3`、`b20`、`b25` 在這個 constructor 內沒有 reader。
因此 parser 的歷史 `race=b2`／`cls=b3` 只能保留為 normalized exporter
相容標籤，不是 runtime `+0x1F/+0x20` 的 ABI 證據。後兩欄仍由 EXE
constructor tables 寫入。

position resource 的 row stride 是6；`0x10C85..0x10C9F` 以 unit row index
取 `base+2+index*6`，只讀 X/Y word 的 low byte。`[0x53AFA]==0` 時，
`0x10C50` 先依序呼叫 `0x145CD(0)` 與 `(1)`：兩次合起來把所有
`unit+5 bit0==0` 的現行 runtime 單位格標成 `0x40`，四鄰標成
`0x80`。後續只排除 `0x40`，再以 row-major 掃全圖，選 Manhattan
距離最小格；`jg` 使同距離由後掃到的格覆蓋。這不是地形可通行判斷。
`[0x53AFA]!=0` 才直接採原座標。因此 future group adapter 不能只拿
目前 JSON `x/y` 當作無條件結果。33份 map assets 現另外保存 exact
`native_position_record{x_word,y_word,raw_key}`、raw `b2`、三-byte death
triple、未讀的 `b3/b20/b25`，以及 b1-selected constructor record。
`NativeFutureGroupPlacement` 已轉寫 placement prefix；
`DecodeNativeFutureConstructorBase` 已轉寫兩條 table 分支並以33圖
1,885筆 unit record 核對 race/class/HP/MP。合法 IDA 與 Capstone 進一步閉合
`0x1B750` 的八格裝備、`+0x22/+0x23/+0x24` modifier、binary64 1.15 與 x87
朝零捨入；`MaterializeNativeFutureConstructor` 現已把 table、inventory 與
effective-stat 重算原子接進 future-group append。handler path 另以
`raw_placement_gate` 保存逐 call-site byte，並用
`AppendGroupWithNativePlacement` 接上 position／occupancy／row-order append
前綴。global turn-event 的45筆可解析 schedule 也已降階為46個 editable
actions，保存 `native_event_id` 與逐 call source/via/gate；只有具完整 runtime
roster 的 `runtime_append_groups` 情境走相同精確配置，缺資料即失敗即關閉。
ch01 的 event1/2 現由 UI battle-event runner 承接 `spawn_group_with_intro`：
分別保存 group4／5、12次 FDOTHER #9 呈現與 caller ACTING(3／4)，並以 scenario
欄位指向可編輯 acting resource set。未遷移情境的正規化分支、其他未知 intro
caller，以及尚無同狀態 DOSBox 比較的畫面仍不得提升為原版等價。直接指令見
[`fd2_future_group_constructor_capstone.txt`](../data/fd2_future_group_constructor_capstone.txt)
與 [`fd2_future_group_raw_gate_ida.txt`](../data/fd2_future_group_raw_gate_ida.txt)。

## 2. 全 30 章原版 handler 對照表（raw 對映已驗；不等於重製流程完成）

`D` = default `0x205b4`（11 章共用，落入 `0x205be` 的 raw 三值規則）。
這個位元組規則常與殲滅玩法一致，但不可只靠函式名稱直接提升為「純殲滅」。
單位以全域 idx 標（十進位）。

| 章 | handler | 觸發條件 | 結果碼 | 繪圖 | 備註 |
|---|---|---|---|---|---|
| 0,2,3,4,5,6,7,8,10,13,23 | `0x205b4` **D** | 掃 `+6==0 && (+5&1)==0`；另查 record0 bit0 | raw code 0/1/2 | — | 玩家可見殲滅語意需逐關／外層驗證 |
| 1 | `0x206c5` | 單位群 5–10 狀態 | 1 | — | |
| 9 | `0x20707` | 先 `0x20711 call 0x205BE`；`0x3453E(0x32)` 或 `0x3453E(0x33)` 非 0 → `0x20732 [0x53ECC]=1` | 1 | — | Capstone 已驗（2026-09-17）；重製端 `native_result_code1_records [50,51]`（ch10.json） |
| 11 | `0x2073d` | 單位 14 | 1 | — | |
| 12 | `0x20765` | 單位群(<12)+ 單位 48、59 | 1 | ✓ | 多段事件 |
| 14 | `0x20822` | 單位 64 | 1 | — | |
| 15 | `0x2084a` | 單位 65 | 1 | — | |
| 16 | `0x20872` | `roster_has(18)` + 單位 52 | 1 | ✓ | 我方有角色18 + #52 相關 |
| 17 | `0x208cf` | 單位 0/16/17 任一 inactive→**1**;單位 52 inactive→**2** | 1 / 2 | — | 條件=指定單位 inactive |
| 18 | `0x20926` | 回合 ≥6 + 單位 64 | 1 | — | 回合觸發 |
| 19 | `0x20957` | 單位群(<46)→ 1;單位 52 → **2** | 1 / 2 | ✓ | |
| 20 | `0x20a51` | 單位 16、17 | 1 | — | |
| 21,26,27 | `0x20a87` | 單位(迴圈群) | 1 | — | 三章共用 |
| 22 | `0x20aaf` | 單位 16/17 inactive→1;單位 18 inactive→**2** | 1 / 2 | — | 條件=指定單位 inactive |
| 24 | `0x20b14` | 單位 16 | 1 | — | |
| 25 | `0x20b3c` | 單位(兩個迴圈群) | 1 | — | |
| 28 | `0x20b72` | 單位 → **2**;單位 40 → 1 | 1 / 2 | ✓ | 結局關 |
| 29 | `0x20bf5` | 單位 20 → **2**;單位 → 1 | 1 / 2 | ✓ | 結局關 |

**讀法**：特殊章的共同結構只證實「查特定單位或單位群的 raw 狀態，
命中時寫 pending 碼 1／2；未命中時可回落到共用三值規則」。碼 1 的外層
固定走資源 #79 呈現；碼 2 走章節索引戰後表與 `0x2CAD7` gate。不能只靠
數值把各 branch 命名成劇情事件、特殊勝利或標準殲滅。確切玩法語意
（護衛目標／主將／其他章節條件）仍須逐章配合原版一般玩家路徑驗證。

## 3. 範例:章 17 handler `0x208cf` 反組譯 [驗]

```
0x208d9 push 0;    call 0x3453e; test; jne 0x20903   ; 單位0 inactive→設碼1
0x208e7 push 0x10; call 0x3453e; test; jne 0x20903   ; 單位16 inactive→設碼1
0x208f5 push 0x11; call 0x3453e; test; je  0x2090d   ; 單位17 active→跳過
0x20903 mov [0x53ecc],1                              ;★ 0/16/17 任一 inactive → 碼1(中途事件)
0x2090d push 0x34; call 0x3453e; test; je 0x20925    ; 單位52 active→跳過
0x2091b mov [0x53ecc],2                              ;★ 單位52 inactive → 碼2
```
即章 17 caller 規則:**「單位 0/16/17 任一 raw bit0 非零→播中途事件；單位 52 raw bit0 非零→設碼 2」**。這是該 handler 的 branch，不能泛化成全域生命欄位。#52 確切角色/陣營未深究(重製不需,見 doc 27 #9-10)。

## 4. 提議的 remake 腳本 schema(取代硬編碼)

下例只示範可編輯的 **candidate DSL projection**；raw predicate/result code
尚未逐caller閉合前，不得由 ScenarioRunner 自動視為原版語意：

```jsonc
// campaign.chapters[17].battle_events
{
  "default_win": "annihilate",          // 無事件觸發 → 標準殲滅判定(對應 default handler)
  "events": [
    { "when": { "unit_inactive": [16, 17] }, "do": "story_event" }, // [0x53ecc]=1
    { "when": { "unit_inactive": [52] },      "do": "victory" }      // [0x53ecc]=2
  ]
}
```
- `when.unit_inactive:[…]` / `units_in_range:[a,b]` / `turn>=N` ← 對應原語
- `do: story_event | victory | show_scene` ← 對應 `[0x53ecc]` 與 `0x15f84`
- 11 個 default 章 → 直接 `{"default_win":"annihilate","events":[]}`,零工作量
- 18 個非default章 → 可先填candidate `events`，但仍須逐caller/章驗證

機器可讀骨架已生成:[`docs/data/battle_events.json`](../data/battle_events.json)
(30章，各含handler/trigger_units/result_codes/draw_scene/action_fns)。它可供
editor/audit作初始資料，但必須經typed adapter與逐章evidence gate，不能直接
餵入production ScenarioRunner當原版真值。

## 5. 對重製流程的銜接

1. `battle_events.json`(本篇)→ 每關「勝利/事件條件」骨架
2. + `maps_metadata.json`(doc 03)→ 單位 idx 對應實際角色/敵人 + 出場位置
3. + 章節文本 FDTXT(doc 09)→ 事件觸發時播的對白
4. + 章節跳表 `0x51d71`/`0x51de9`(doc 23)→ 戰前/戰後劇情
→ 這些來源可組成editable campaign scaffold；完整原版流程仍須補
handler CFG、postbattle side effects、town/shop/preparation route與save/reload
regression。資料驅動是目標架構，不是目前已證實所有事件語意都在資料內。

## 6. 受阻 / 待驗(誠實標註)

- **[修正]** byte[+5] 的 bit0 reader／writer 已分開閉合：`0x3453e` 是 raw `&1` predicate，`0x32975` 是整 byte overwrite，constructor/death writers 是其他 caller。舊說「使用者確認 bit0=存活」已撤回；各 handler 仍需保留自己的 branch evidence。
- **[已驗，範圍限定]** 章16 `0x33499` 不是動作,是條件查詢(roster_has)；`battle_events.json` 的 battle-event skeleton 匯出 `action_fns` 皆空，僅代表該匯出層未記錄動作，不能推廣為所有 postbattle/cutscene handler 無動作。後者須按實際 CFG 與 editable script 逐一驗證。
- **[已驗]** 回合數 = **`[0x53bef]`**(戰場開始 `mov 1`、`inc`、handler `cmp N`),**非 `[0x53ec8]`**(後者是 `add reg` 累積計數+clamp 99,語意待定)。詳見 §7。
- **[阻]** 迴圈查的單位群(章 1/12/19/21/25)精確 idx 範圍見逐指令 dump(章1=5–10、章12=<12、章19=<46);`battle_events.json` 的 `trigger_units_flag` 只收立即數 push,迴圈索引另記於 `extra_conditions`。
- **[阻]** 單位全域 idx → 角色/敵人名 對應 + 我方槽數 M；normalized remake 可使用自有具名單位投影，但 raw handler path 仍需保留原始 96-slot order/provenance，不能一概略過。
- **[已驗證 raw]** byte[+5] bit0／bit7 的個別 mask 使用；回合 `[0x53bef]` increment 與 team-completion 語意仍需 state-machine caller evidence，不在本表宣稱。
- **[低優先]** `[0x53ec8]` 累積計數(靜態:累加單位 +0x21,clamp99)非重製核心,可選。

> 相關:doc 25(事件系統架構)· doc 24(戰役迴圈 [0x53ecc] 狀態機)· doc 19(腳本系統)· doc 09(劇情)· doc 03(單位結構/roster)。工具:`tools/event_handler_dump.py`;資料:`docs/data/battle_events.json`。
