# 44 — 第一章「初試身手」：原版正確序列 × remake 錯誤清單

> 目的：回應使用者實玩 remake 回報的疑點（開場劇情缺口、增援時序、職業名稱錯位、站位/朝向），
> 逐項對照**使用者提供的原版遊玩錄影**（`video/fd2-ch1.mp4`）+ **原版 dosbox 實機**（沿用
> `extracted/orig_play_notes.md` 與 `docs/knowledge-base/25-battle-event-system.md §7.5.1` 同一輪
> 今日已完成的連拍，未重複開新 dosbox session）+ **青衫攻略**（`references/text/fd2.md`，三方 ground
> truth 互相印證），核對 remake 程式碼/資料（`remake/assets/scenarios/ch01.json`、`campaign_full.json`、
> `remake/assets/maps/map0/map0_units.json`、`tools/export_units.py`）。唯讀稽核，未修改任何程式碼、未 commit。

## 0. 證據來源（避免重工，見規則 63/64；影片與青衫攻略並列 ground truth）

- **使用者提供的原版遊玩錄影（最高優先 ground truth）**：`video/fd2-ch1.mp4`（71 分）。已抽每 15 秒一幀
  存 `extracted/story/ref_video/f_001.png`–`f_040.png`（前10分）+ contact sheet `_contact.png`；
  本輪針對第一戰（7:50–9:20）另用 ffmpeg 抽每 2 秒細幀存 `extracted/story/ref_video/fine/t001.png`–`t045.png`
  + `_fine_contact.png`，用來讀取戰鬥狀態欄的**單位名稱文字**（見 §2.5）。
- **dosbox 實機**：`extracted/remake_shots/orig_0*.png`（標題→對話→戰場→攻擊演出全流程）
  + `extracted/story/staging_dosbox/{proof_01,proof_02,proof_03}.png` + `seq/`（220+ 張連拍，序章開場）。
  兩批連拍皆為**今日**（2026-07-03）稍早已完成的實機截圖，本篇直接引用、疊加分析，不重跑 dosbox。
- **EXE 靜態反組譯**：`docs/knowledge-base/25-battle-event-system.md`（§6 turn_events 全 30 章反組譯
  + §7.5.1 章節0 handler `0x3231b` 逐指令複驗）、`docs/data/turn_events.json`（EXE dump，非猜測）、
  `docs/data/exe_tables/unit.json`（(race,cls)→基礎數值/職業名表）、`tools/dump_exe_tables.py`（該表產生器）、
  `tools/parse_field.py`（FDFIELD roster 逐位元組解析）。
- **青衫攻略（與影片並列 ground truth，數值/時序/職業名/敵方組成權威）**：`references/text/fd2.md` 第1章、
  `docs/knowledge-base/28-chapter-objectives-and-recruits.md`。
- **原始劇本文字**：`extracted/story/full_story_auto.md`（FDTXT 全 34 個資源自動解碼，1450+ 句）。
- **remake 程式碼/資料**：`remake/assets/scenarios/{campaign_full.json,ch01.json}`、
  `remake/assets/maps/map0/map0_units.json`、`remake/internal/battle/event.go`、`tools/export_units.py`。

> 分工採用使用者指定的方式：**影片 = 視覺/序列/站位/朝向權威(逐幀看)；青衫攻略 = 數值/時序/職業名/
> 敵方組成/增援回合權威(文字)**。以下每項落差盡量同時附兩邊證據。

---

## 1. 原版第一章完整正確序列

### 1.1 開場劇情（新遊戲 → 進戰場，共 3 段文本資源）

原版章節0 handler（`0x3231b`）**不是**單純載入「資源=章節+1」，而是先後暫借章節值切到兩個
獨立資源（doc23 §4、doc25 §7.5.1 已用靜態反組譯 + 今日 dosbox 複驗釘死）：

| 順序 | 資源 | 內容 | dosbox 證據 | 影片證據 | 劇本文字證據 |
|---|---|---|---|---|---|
| ① | `FDTXT_033`（chapter=0x20 暫借值+1） | **王城·父子送別**：紅毯王座廳，父王（羅特帝亞國王，藍髮八字鬍）與王后並坐雙王座召見索爾，欲傳位給他；索爾自承是義子、無血緣，力辭；父王堅持「屬於強者的時代」，命三日後回覆；索爾告退，之後被亞雷斯撞見獨自苦惱 | `orig_02_dialog_01_sol.png`「兒臣索爾，晉見父王陛下。」、`orig_02_dialog_02_king.png`（國王頭像右上，3行台詞） | `ref_video/f_004.png`（父王特寫「你。這一陣子國務繁忙，我們父子兩個也很久沒聚聚了」）、`ref_video/f_006.png`（雙王座紅毯全景，索爾跪拜，字幕「兒臣以為應該由皇弟迪恩來繼承王位....」——與 FDTXT_033 原文逐字對上） | `full_story_auto.md:4918-5034`（原文一字不差對上截圖字幕） |
| ② | `FDTXT_032`（chapter=0x1f 暫借值+1） | **草地小憩 → 比劍邀約 → 悠妮/蓋亞加入**：索爾找亞雷斯再比劍被婉拒 → 兩人發現昏倒的悠妮，蓋亞（機兵，「主人」+「慣性導航系統」台詞確認是機器人守衛非人類）在旁戒備 → 悠妮失憶、記得「從很遠的地方（馬拉大陸）來」、噩夢閃回 → 索爾堅持送她回馬拉大陸，亞雷斯一度反對後同意同行 → 決定啟程 | `proof_01_field_rest.png`（草地上兩人對話「怎麼啦？你」）、`orig_02_dialog_03_ares.png`「拜託，今天休息吧！」、`orig_02_dialog_05_yuni.png`「我們..是從哪裡來的?」 | contact sheet 第2、3列（草地+對話框，索爾/亞雷斯/紅髮悠妮） | `full_story_auto.md:4828-4915` |
| ③ | `FDTXT_001` | **抵達馬拉大陸外海小島 → 遭遇海盜 → 哈瓦特/哈諾出手相助**（已有精校轉錄，見 `extracted/story/序章_transcript.md`） | `proof_02_pirate_prebattle.png`（索爾/悠妮/蓋亞 + 3 海盜 + 1 機械兵已在最終戰鬥位置對峙）、`orig_02_dialog_04_pirate.png` | `ref_video/fine/t030.png`（哈瓦特特寫「『真是的，吵的要命」，與 FDTXT_001 逐字對上） | `full_story_auto.md:637-823` |

> 兩段獨立捕捉（今日 dosbox 連拍 + 使用者提供的原版錄影）在①③兩段的字幕文字**逐字吻合**，
> 互為交叉驗證，非單一來源的巧合。

> **speaker 標註提醒**：①②兩段資源的自動解碼**說話者標籤有已知誤判**（如②把悠妮的台詞標成
> 「亞雷斯」、把蓋亞的台詞標成「哈瓦特」——內容本身「主人／慣性導航系統」與「悠妮失憶」清楚指向
> 正確角色，僅 portrait ID → 名字的查表在這兩份特殊資源上失準，見 worklist「speaker→頭像機制 RE」
> 已知風險）。**劇情內容本身（誰做了什麼、說了什麼）已用 dosbox 截圖字幕逐句核對過，可信；只有
> 「說話者姓名標籤」在①②兩段需要人工重新指認**，③（FDTXT_001）已有精校版不受影響。

### 1.2 進戰場：直接定位，無行軍動畫

`docs/knowledge-base/25-battle-event-system.md §7.5.1`（今日已反組譯 `0x3231b` 本體 + 220+ 張連拍複驗）
已裁決：原版**沒有**任何單位逐幀移動/行軍動畫，索爾一行人與海盜/蓋亞都是**直接定位**在最終戰鬥座標，
「移動」的視覺印象來自**攝影機平移**（`0x13185`/`0x32999`），不是單位位移。**remake 現行
`focusOnParty`（鏡頭對準）+ `spawn_party`（直接定位）已忠實**，此點**非 bug**，收錄於此表僅為完整記錄
使用者原始疑點 #4（位置/朝向），不重複列入下方錯誤清單。

### 1.3 戰場：回合制增援（EXE `turn_events` 逐回合，青衫攻略逐句對應）

| 回合 | 陣營 | 內容 | turn_events.json（EXE 反組譯，`docs/data/turn_events.json` chapter=1） | 青衫攻略原文 |
|---|---|---|---|---|
| 開戰（T0） | 敵 | **海盜**（portrait 96）分兩批共 8 人（remake group 1/2），對應原版 `0x3231b` 內僅有的兩次 `spawn_group_with_intro`（doc25 §7.5.1：「序章呼叫2次，group 1/2」） | — | 「敵方：LV2盜賊x7」（首行，數量近似） |
| T3 己方結束 | 友 | 哈瓦特+哈諾從房子出來幫忙（group 3/7） | `{"turn":3,"camp":"ally","groups":[3,7]}` | 「第三回合己方結束時，哈瓦特和他的兒子哈諾從房子出來幫忙」 |
| T4 己方結束 | 敵 | 4 名敵方援軍，右下角（group 4） | `{"turn":4,"camp":"enemy","groups":[4]}` | 「第四回合己方結束時，四名敵方援軍出現在右下角」 |
| T5 己方結束 | 敵 | 海盜頭目+4屬下，左下角（group 5） | `{"turn":5,"camp":"enemy","groups":[5]}` | 「第五回合己方結束時，海盜頭目帶著四名屬下出現在左下角」 |
| T6 己方結束 | 友 | 4 名海防隊員（**portrait 68＝一般士兵**，見 doc31），右上角，**立即行動**（group 6） | `{"turn":6,"camp":"ally","groups":[6]}` | 「第六回合己方結束時，四名海防隊員從右上角出來幫忙並立即行動」 |
| 哈諾陣亡時 | — | 哈瓦特暴走（AI 轉狂暴） | — | 「如果哈諾在戰鬥中不幸陣亡，原本可以控制的哈瓦特會暴走」 |

> **「友方 LV2士兵x4」= T6 海防隊員**（portrait 68，doc31 明確標「一般士兵」），**不是**戰場開場就有的
> 常駐部隊——攻略把它跟敵方陣容並列，只是欄位格式（先列雙方戰力表，事件另起一段），不代表開場即在場。

---

## 2. remake 錯誤清單

### 2.1【高】增援時序 bug：開場多冒出 7 名「戰士＋聖騎士」——**根因已定位**

**現象**：remake `ch01.json` 的 `initial_groups: [1, 2, 10, 11]`（`remake/assets/scenarios/ch01.json:6-10`）。
group 1/2 是正確的開場海盜（對應 §1.3 表格「開戰 T0」），但 **group 10（4×「戰士」，portrait 76，
HP18/AP5/DP2 極弱）與 group 11（3×「聖騎士」，portrait 103，HP44/AP26）也被塞進 `initial_groups`**，
於是這 7 個單位在**開戰第 0 回合**就直接出現在戰場（`remake/internal/battle/event.go:87-93`
`Setup()`：非 `initial_groups` 的單位才設 `OnField=false` 待命，group 10/11 在名單內故立即在場）。

**證據鏈（三個獨立來源交叉確認 group 10/11 不該在開場）**：

1. **`docs/data/turn_events.json`（EXE 反組譯 dump，全 30 章）**：chapter 1 只有 4 筆 turn_events，
   引用 group `[3,7]`/`[4]`/`[5]`/`[6]`——**全域搜尋 30 章，group 10、11 從未出現在任何一筆
   turn_events**，即整個遊戲的「回合觸發增援」機制裡，這兩組從未被任何 handler 呼叫過。
2. **doc25 §7.5.1（今日 `0x3231b` 逐指令反組譯）**：章節0 handler 本體內，唯一的「群組登場」呼叫
   （`0x32999 spawn_group_with_intro`）**只出現 2 次，引數 group=1、group=2**，沒有 group 10/11。
3. **青衫攻略**：第一章敵方/友方/事件三段完整列舉（見 §1.3），全程沒有「戰士」「聖騎士」字樣；
   `references/text/fd2.md` 全篇搜尋「聖騎士」的第一次出現在遠後段章節，非第一章。

**結論**：group 10/11（4戰士+3聖騎士）是 map0 FDFIELD roster 裡確實存在、但**原版遊戲從未實際
啟用過**的「死資料」（可能是保留/備用 slot，或需要另一個尚未反組譯的觸發路徑——但至少不是
`turn_events` 也不是章節0 cutscene，兩條已知的第一章觸發機制都排除了）。remake 的 `initial_groups`
把這批死資料當成「開局即在場」納入，是資料層面的誤用，不是刻意設計。

**修復建議**：`ch01.json` 的 `initial_groups` 由 `[1, 2, 10, 11]` 改為 `[1, 2]`。
若之後想保留 group 10/11 供未來反組譯出真正觸發條件時使用，可移出 `initial_groups` 陣列，
留在 `map0_units.json` 原始資料中待命即可（`OnField=false`），不影響現行行為。

**視覺對照**（今日已有的 dosbox 截圖，未含 group10/11 對應單位）：
`extracted/remake_shots/orig_03_battle_start.png` 開戰畫面僅見索爾/亞雷斯/悠妮/蓋亞 + 3 名紅頭巾海盜，
`extracted/story/staging_dosbox/proof_02_pirate_prebattle.png` 對峙畫面同樣只有 3 海盜 + 1 機械兵，
兩張截圖都**沒有「戰士」類弱雜兵或「聖騎士」類重甲單位**，與 group 10/11 的存在矛盾。

### 2.2【高】完全缺少開場劇情三幕（王城父子送別 / 草地小憩比劍 / 悠妮蓋亞加入）

**現象**：`remake/assets/scenarios/campaign_full.json` 的 `story_ch01` 節點只有兩行佔位文字
（`"第1章:初試身手"` / `"目標:敵全滅。"`），直接銜接 `battle_ch01`——**完全沒有** §1.1 表格
①②③三段開場劇情（父王傳位/索爾力辭、草地小憩比劍邀約、發現悠妮+蓋亞失憶加入、渡海遇海盜）。

**對照使用者原始回報 #1、#2**：均證實成立，且比使用者記憶更完整——不只「皇宮父子對話」缺失，
連「草地小憩比劍邀約」「渡海抵達小島」兩段過場也完全沒有，remake 目前是「選新遊戲 → 兩行文字 →
直接開戰」，跳過了原版三段、合計約 200 句對白的完整開場鋪陳。

**修復建議**：`story_ch01` 拆成 3 個 story 節點（對應 FDTXT_033/032/001 三段），文字來源可用
`extracted/story/full_story_auto.md:4828-5034`（②③已有精校版可直接用 `序章_transcript.md`）；
①段（FDTXT_033）目前只有自動解碼，說話者標籤需人工校正後才適合直接入库（見 §1.1 附註）。
著作權處理比照既有做法（`8215ecf` commit 已示範：對白改寫為原創同義句，原文留本機不入版控）。

### 2.3【中】敵方組成因 2.1 而失真

**現象**：因 group 10/11 誤入場，remake 開場敵方變成「8海盜 + 4戰士 + 3聖騎士」共 15 人，
而非原版「8海盜」（T0）。除了憑空多出戰力／畫面雜訊，也讓「聖騎士」這種原版遠後期才出現的
高階敵種在第一章開場現身，破壞難度曲線與角色識別（青衫攻略/原版截圖均確認第一章敵方視覺
統一是紅頭巾海盜）。此項與 2.1 同一根因，**修 2.1 即一併修正**，不需獨立改動。

### 2.4【高，根因已定位】職業名稱錯位：remake 顯示「劍士/聖騎士」，原版顯示「盜賊/士兵」

**現象（使用者實測回報）**：remake 戰鬥中單位職業名顯示錯誤——原版**海盜**類單位，remake 顯示
**劍士**；原版**士兵**類單位，remake 顯示**聖騎士**。

**影片直接證據（決定性）**：`ref_video/fine/t013.png`/`t014.png`/`t022.png`（見 §0，7:50-9:20 精抽幀）
清楚拍到原版戰鬥全螢幕演出的狀態欄文字——蓋亞（我方）攻擊海盜單位時，對方狀態欄**明字顯示
「盜賊　LV·02」**（非「劍士」）。三幀（t013/t014/t022）分別是蓋亞×2 次攻擊 + 索爾×1 次攻擊同一敵人，
「盜賊」字樣**逐幀重複出現，非單幀誤讀**。青衫攻略同步佐證：第一章敵方欄位全程只用「盜賊/海盜頭目」
稱呼，全篇「聖騎士」字樣第一次出現在遠後段章節（`references/text/fd2.md:999`「聖騎士蘭斯洛特」，
第18章加入角色，非第一章敵人）。

**remake 現況**：`map0_units.json`（`export_units.py` 匯出）group1/2（portrait 96，即上述被攻擊的海盜）
`cls_name` 欄位是 `"劍士"`；group11（portrait 103）是 `"聖騎士"`。

**根因（已用原始位元組資料鎖定）**：

1. `tools/parse_field.py` 從 FDFIELD 控制段直接讀出每個單位的 `race`/`cls` 兩個原始欄位（26 位元組
   roster 記錄的 byte2/byte3，非衍生值）。實測 map0 全部單位：海盜（portrait96/97）、哈瓦特（portrait3）、
   哈諾（portrait1）、T6 海防隊員（portrait68）**全部是 `(race=1, cls=1)`**——四種完全不同、視覺/敘事
   毫無關聯的角色，共用同一組 `(race,cls)`。這證明 `(race,cls)` 是**戰鬥數值的機械式範本索引**
   （HP/AP/DP/MV 從哪張表抄),不是單位的顯示名稱來源。
2. `tools/export_units.py` 的 `cls_name` 卻直接拿 `(race,cls)` 去查 `docs/data/exe_tables/unit.json`
   的 `cls_name` 欄位——而該表的 `cls_name` 是 `tools/dump_exe_tables.py` 用**玩家轉職職業表**
   `CLASS_NAMES = ["龍","劍士","戰士","騎士",...,"聖騎士",...]`（25 個玩家職業 + 1 個「龍」，逐位元組
   對上 FDTXT_000 該段名單，經 dump 值與 doc02 §7.2 人物成長表交叉驗證吻合，這張表本身沒有錯）填出來的。
   `cls=1` 對到 `CLASS_NAMES[1]="劍士"`,`cls=11` 對到 `CLASS_NAMES[11]="聖騎士"`——**但這張表的用途
   是幫玩家角色（索爾/亞雷斯/…）算轉職後的屬性成長，不是給敵方/NPC 單位取顯示名字**。`doc02` 全篇
   「聖騎士」的每一筆出現（亞雷斯/洛娜/萊汀/蘭斯洛特轉職）都在玩家轉職語境，從未出現在敵方/怪物語境
   （已逐筆核對 `docs/knowledge-base/02-game-data-reference.md`）。
3. 換句話說：**`export_units.py` 把「玩家轉職職業表」誤當「敵方/NPC 顯示名稱表」在用**——這張表對
   「玩家角色算數值」是對的（remake 的 `growth.go` 用它算轉職成長已交叉驗證過），但拿來當**敵方單位
   UI 顯示名稱**是誤用。真正的顯示名稱（盜賊/士兵/…）另有來源，目前最可能的線索是 `portrait` 欄位——
   `docs/knowledge-base/31-map-unit-sprites-fdicon.md` 已證實「地圖 sprite 組 = portrait 恆等」且已有
   兩筆獨立驗證的 portrait→名稱對應（`68→一般士兵`、`96–107 組→綠衣盜賊`，與本輪影片實測的
   「portrait96＝盜賊」完全吻合），加上 FDTXT_000 另一段「士兵/精英戰士/…/盜賊/盜賊頭目/…」的
   NPC/怪物名單（54 筆，`extracted/story/full_story_auto.md:41-99`），兩者對得上就是正確的顯示名稱表。

**修復建議**：`export_units.py` 的 `cls_name` 欄位改成**用 `portrait` 查一張新的「portrait→顯示名稱」
表**（延伸 doc31 已有的兩筆對應，逐一比對 FDTXT_000 的 54 筆 NPC/怪物名單，需要再一輪針對性 RE 補齊
portrait 96/97/103/76/68 等第一章用到的全部 portrait 的正確名稱)，`(race,cls)` 繼續專職算數值
（HP/AP/DP/MV/暴擊，這條路徑本身正確不要動）,兩者分離。**不建議**直接改 `dump_exe_tables.py` 的
`CLASS_NAMES` 表——那張表對玩家轉職是正確的，錯的是「拿它給敵方單位命名」這個用法本身。

**未完全坐實的部分（誠實標記）**：「士兵→聖騎士」這組使用者回報，本輪**找到了完全相同性質的根因
機制**（`cls=11` 查到玩家表的「聖騎士」)，但**未在影片中直接拍到「士兵」字樣的敵方狀態欄畫面**
（t013/t014/t022 三幀都是同一隻「盜賊」被攻擊；含疑似 portrait76/103 單位的畫面——`ref_video/f_035.png`、
`fine/t028.png`、`t030.png`——目前只看到兩個灰色/藍白裝甲人形站在哈瓦特小屋旁的石徑兩側、連續多幀
不動，較像**裝飾性守衛雕像**而非可戰鬥單位，本輪未能確認它們是否會參戰、參戰時狀態欄文字為何）。
「聖騎士」機制性錯用本身已用 doc02 交叉驗證坐實，「士兵」這個具體對照詞待下一輪影片/dosbox 補一張
明確的敵方狀態欄截圖。

### 2.5 主角隊位置/朝向：區域正確，逐人格位分配**已修正**（高，2026-07-04 二輪核對後定案）；朝向欄位存在但預設值未驗證（中）

> 修正前版本的誤判：先前版本聲稱「remake Unit 結構完全沒有朝向欄位」——**這是錯的**，
> 已修正如下（搜尋關鍵字漏抓 `Dir`，只搜了 `Facing`/`Direction`）。

- **位置（區域級別已驗證，逐人精確分配已於本輪核對修正）**：`ch01.json` 的 `deploy_cells`
  與 `map0_units.json` 的 `own_deploy` 完全一致，後者直接來自原版 FDFIELD「出場位置」資源第3段
  （doc23 §4）。本輪把 `map0.json` 地形+全部 group 座標渲染成標註圖
  （`extracted/story/ref_video/map0_annotated.png`,本機,未入庫）核對：`own_deploy` 4 格落在地圖南側沙洲旁的泥地，
  緊鄰 group2 海盜（(0-4,21-23)，同樣濱海）與哈瓦特小屋（(11,11)，隔一段距離）——**區域位置與影片
  （`ref_video/f_029.png`）、dosbox（`orig_03_battle_start.png`）呈現的「主角隊在草地/泥地、海盜貼著水岸」
  吻合**，方向正確。
  **逐人分配（本輪修正）**：原始 FDFIELD 的 4 個 `own_deploy` 格是**無名格**（只標記「這裡可站己方」，
  不綁定「哪一格給哪個角色」），remake `event.go` 用**陣列順序**指定（`party[i]→deploy_cells[i]`）。
  用 `fig` sprite 外觀逐一核對 `orig_03_battle_start.png`/`f_029.png` 畫面中 4 個單位的身分
  （`fig4`=藍色頭盔騎士=亞雷斯、`fig9`=紅髮=悠妮、`fig30`=機甲=蓋亞，對照
  `remake/assets/sprites/fig_004/009/030_f00.png`），發現影片實際佈局是
  **「索爾+亞雷斯緊鄰聚在一起、悠妮在稍右側、蓋亞最右」**，但 `ch01.json` 原本的 `deploy_cells`
  陣列順序（`[[7,20],[10,21],[8,22],[11,23]]`）配上 `party` 順序（索爾/亞雷斯/悠妮/蓋亞）會把
  **悠妮**分到跟索爾緊鄰的格 (8,22)、**亞雷斯**分到較遠的格 (10,21)——跟影片相反。
  **修正**：交換 `deploy_cells[1]`/`[2]`（改成 `[[7,20],[8,22],[10,21],[11,23]]`），讓亞雷斯拿到
  緊鄰索爾的格、悠妮拿到較遠格，重新截圖（隔離 Xvfb + xdotool 送 Enter 清對白後拍）跟
  `orig_03_battle_start.png` 構圖一致。**信心等級誠實標註**：格子本身（4 個座標）= FDFIELD 直讀，
  高信心；「哪個角色對哪個座標」= 目視影片單位外觀聚落關係反推，非 FDFIELD 直讀（原始資料本身不記
  角色身分，見上），中高信心但非位元組級鐵證——若之後想要更硬的證據，仍可反組譯章節0 handler 裡
  「主角隊 4 次 `0x10b4e` 呼叫」各自的座標引數（doc25 §7.5.1 只確認了呼叫存在與「直接定位」，
  未逐一展開四個呼叫各自的座標值)。
- **朝向**：`remake/internal/battle/model.go:53` 有 `Dir int // 朝向:0下 1左 2上 3右`欄位，
  `event.go:162` 主角隊進場時 `Dir: 0`（下／面向鏡頭）寫死。影片 `f_029.png`/`fine/t013.png` 等幀顯示
  的角色 sprite 確實接近「正面朝下」姿態，與 `Dir:0` 不衝突,但本輪未逐一比對 4 名主角個別朝向
  （可能各自不同，如 87 度側身)，**維持中優先、待後續逐人比對**，不算已證實的 bug，也不算已證實正確。

---

## 3. 修復優先序

| 優先度 | 項目 | 工作量估計 | 依據 |
|---|---|---|---|
| 1 | **`ch01.json` `initial_groups` 移除 `10, 11`**（§2.1） | 極小：改 1 行 JSON | 三方交叉證據（turn_events.json 全域搜尋、doc25 §7.5.1 反組譯、青衫攻略）一致排除 |
| 2 | **職業名稱來源改用 portrait 查表，不用 (race,cls)**（§2.4） | 中：新建 portrait→名稱表（先補第一章用到的 96/97/103/76/68 等）+ 改 `export_units.py` 一處查表邏輯 | 影片狀態欄「盜賊」文字直接證據 + doc02 全篇交叉驗證「聖騎士」只用於玩家轉職語境,根因機制已鎖定 |
| 3 | **補開場三幕劇情**（§2.2）| 中：3 個 story 節點 + 對白文字（②③已有現成文字，①需校正 speaker） | dosbox 截圖 + 影片 + FDTXT 原文三方逐句對應，內容已齊全，缺的是接進 remake |
| 4 | （隨 1 一併解決）敵方組成回歸純海盜（§2.3）| 0（附帶效果） | 同上 |
| 5 | ~~主角隊逐人格位分配精確驗證~~ **已修正**（§2.5 位置部分,2026-07-04）| 已完成:交換 `ch01.json` `deploy_cells[1]`/`[2]` | fig sprite 外觀比對 `orig_03_battle_start.png`/`f_029.png` 逐一核對亞雷斯/悠妮身分,發現原順序把兩人配對顛倒 |
| 6 | 朝向（Dir）預設值逐人驗證（§2.5 朝向部分）| 中：需影片逐一比對 4 名主角個別朝向 | Dir 欄位存在且有預設值(0),未逐人比對正確性 |

## 4. 未盡事項

- FDTXT_033/032 的 speaker 標籤誤判需要人工重新指認（不影響劇情內容本身的可信度，只影響
  「這句是誰說的」標註），建議下一輪與「speaker→頭像機制 RE」的既有已知風險一併處理。
- group 10/11 在原版究竟是否有其他觸發路徑（例如某個尚未反組譯的分支/世界地圖 handler），
  本輪只確認了「turn_events 與章節0 cutscene 這兩條已知路徑都不會觸發它們」，未做窮舉排除；
  影片 `f_035.png`/`fine/t028.png`/`t030.png` 拍到兩個灰色/藍白裝甲人形站在哈瓦特小屋旁石徑兩側，
  連續多幀不動，較像**裝飾性守衛雕像**，但未能確認是否真的是 group10/11 對應的視覺、是否會參戰。
  如果之後想讓這兩組在原版真的會用到的地方登場（或確認它們就是純裝飾物、remake 完全不用管），
  需要另開反組譯/影片追蹤任務。
- 「士兵」這個具體職業名詞待下一輪影片/dosbox 補一張明確的敵方狀態欄截圖（本輪只坐實「聖騎士」
  誤用機制，「盜賊」已有逐幀影片文字證據，「士兵」目前只有 doc31 的既有記錄佐證，未在本輪影片重新
  拍到)。
- 朝向（Dir）欄位存在（`model.go:53`），預設值 `0` 在集中比對下與影片大致不衝突，但未逐人像素級驗證。
- ~~主角隊逐人（索爾/亞雷斯/悠妮/蓋亞）→ `deploy_cells` 逐格精確對應關係~~ **已於 2026-07-04 修正**
  （見 §2.5、優先序表 5）：fig sprite 外觀比對影片/dosbox 截圖確認亞雷斯(fig4)/悠妮(fig9)兩人的
  `deploy_cells` 索引原本顛倒，已交換修正並截圖驗證。

## 5. 開場過場 remake ↔ 文件 溯源與落差(2026-07-05 稽核:王座 + 亞雷斯兩幕)

> 起因:使用者問「已 remake 實作完成的成果跟 markdown 文件之間有什麼關聯」。逐值稽核兩幕。

### 5.1 兩種關聯模式(雙向)
- **模式 A(王座):remake + 使用者截圖 → 回填文件**。工作值 = 靜態 RE(doc47 §11:0x13185 step +
  call 序列 → 對話切分 line0/line1-18)+ dosbox(Y=21)+ 影片(doc55:8,8);**但最終定案值來自使用者
  實玩對原版截圖**:27→**21**、14→**8**、守衛 dir→**0**(非 RE 出來的)。本 session 已回填 doc50 §1.1 / doc47 §11。
- **模式 B(亞雷斯草地):文件(doc55 影片量測)→ 已回填實作**。doc55 逐幀量出「亞雷斯 2 段走近 + 索爾
  對話後單獨走」,remake 已照改(campaign.go:118 註解已更正、palace_path 節點對齊 doc55 量測值)。

### 5.2 逐值溯源表
| remake 值 | 文件出處 | 來源性質 |
|---|---|---|
| (8,21) 第一次停 | doc48/50/55/47 | dosbox Y=21 + 使用者截圖 + FDFIELD 守衛地標 ✓ |
| (8,8) 最終停 | doc50/47/55/48 | 影片 + 使用者「最跟前」✓ |
| 對話 line0 / line1-18 切分 | doc50/47/09 | RE call 序列(STEP×15→對話→STEP×13→對話)✓ |
| 守衛位置 / dir=0 | doc47/50 | FDFIELD map32 roster + 使用者截圖(面向玩家)✓ |
| Sol(4,46) / Ares(13,47→7,46) | doc55 | 影片逐幀量測(「幾乎完全吻合」)✓ |
| cam_max_y 808 | doc47 | ✓ |

### 5.3 落差(要注意)
- ✅ **`pan (72,816,60)`(王座第一拍)= 開場定場鏡頭,有 RE 出處**(先前誤報無出處已更正):
  (72,816)px = (col3,row34)格 = doc47 §3 Part1 beat2 `0x135dd(3,34)`(72=3×24、816=34×24)——
  框住紅毯底端(row42=索爾生成處),在索爾往上走之前。grep 找像素沒對上是因文件記格值。
- frames 210/130/120/85 = 視覺調參(doc50 §2 已註明可調),可接受。
- 「帶他離開」lead/follow:doc55 §4 原判「無畫面」已由上而下更正 = 對應 handler beat11 演出 0x69 +
  使用者截圖 18-08-17;remake 待補淡出前短 lead/follow walk。

### 5.4 方法論(使用者 2026-07-05)
**由上(證據)而下 + 由下(RE)而上 都通往同一事實**。機制已知(doc50:pose/step家族/0x13488/acting)後,
有原版截圖/影片證據時可「由上而下」回去對應 handler beat 還原,不必 RE 到底(如「帶他離開」→beat11 演出0x69)。
記憶見 [[fd2-goal-and-no-speculation-rule]] 第 6 條。

> 相關：`docs/knowledge-base/25-battle-event-system.md`（§7.5.1 staging 反組譯+dosbox 複驗）、
> `docs/knowledge-base/28-chapter-objectives-and-recruits.md`（青衫30關目標表）、
> `docs/knowledge-base/31-map-unit-sprites-fdicon.md`（portrait→sprite組對應，職業名根因引用）、
> `docs/knowledge-base/42-re-vs-remake-gap-audit.md`（機制落差稽核，本篇補的是「內容/資料層」落差，
> 非機制層）、`extracted/orig_play_notes.md`（今日 dosbox 實機筆記）、
> `video/fd2-ch1.mp4` + `extracted/story/ref_video/`（使用者提供原版錄影 + 抽幀，本機不入庫）。
