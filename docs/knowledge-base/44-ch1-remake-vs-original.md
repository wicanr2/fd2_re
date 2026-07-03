# 44 — 第一章「初試身手」：原版正確序列 × remake 錯誤清單

> 目的：回應使用者實玩 remake 回報的 5 項疑點，逐項對照**原版 dosbox 實機**（沿用
> `extracted/orig_play_notes.md` 與 `docs/knowledge-base/25-battle-event-system.md §7.5.1` 同一輪
> 今日已完成的連拍，未重複開新 dosbox session）與**青衫攻略**（`references/text/fd2.md`），
> 核對 remake 程式碼/資料（`remake/assets/scenarios/ch01.json`、`campaign_full.json`、
> `remake/assets/maps/map0/map0_units.json`）。唯讀稽核，未修改任何程式碼、未 commit。

## 0. 證據來源（避免重工，見規則 63/64）

- **dosbox 實機**：`extracted/remake_shots/orig_0*.png`（標題→對話→戰場→攻擊演出全流程）
  + `extracted/story/staging_dosbox/{proof_01,proof_02,proof_03}.png` + `seq/`（220+ 張連拍，序章開場）。
  兩批連拍皆為**今日**（2026-07-03）稍早已完成的實機截圖，本篇直接引用、疊加分析，不重跑 dosbox。
- **EXE 靜態反組譯**：`docs/knowledge-base/25-battle-event-system.md`（§6 turn_events 全 30 章反組譯
  + §7.5.1 章節0 handler `0x3231b` 逐指令複驗）、`docs/data/turn_events.json`（EXE dump，非猜測）。
- **青衫攻略**：`references/text/fd2.md` 第1章。
- **原始劇本文字**：`extracted/story/full_story_auto.md`（FDTXT 全 34 個資源自動解碼，1450+ 句）。
- **remake 程式碼/資料**：`remake/assets/scenarios/{campaign_full.json,ch01.json}`、
  `remake/assets/maps/map0/map0_units.json`、`remake/internal/battle/event.go`。

---

## 1. 原版第一章完整正確序列

### 1.1 開場劇情（新遊戲 → 進戰場，共 3 段文本資源）

原版章節0 handler（`0x3231b`）**不是**單純載入「資源=章節+1」，而是先後暫借章節值切到兩個
獨立資源（doc23 §4、doc25 §7.5.1 已用靜態反組譯 + 今日 dosbox 複驗釘死）：

| 順序 | 資源 | 內容 | dosbox 證據 | 劇本文字證據 |
|---|---|---|---|---|
| ① | `FDTXT_033`（chapter=0x20 暫借值+1） | **王城·父子送別**：父王（羅特帝亞國王）召見索爾，欲傳位給他；索爾自承是義子、無血緣，力辭；父王堅持「屬於強者的時代」，命三日後回覆；索爾告退，之後被亞雷斯撞見獨自苦惱 | `orig_02_dialog_01_sol.png`「兒臣索爾，晉見父王陛下。」、`orig_02_dialog_02_king.png`（國王頭像右上，3行台詞） | `full_story_auto.md:4918-5034`（原文一字不差對上截圖字幕） |
| ② | `FDTXT_032`（chapter=0x1f 暫借值+1） | **草地小憩 → 比劍邀約 → 悠妮/蓋亞加入**：索爾找亞雷斯再比劍被婉拒 → 兩人發現昏倒的悠妮，蓋亞（機兵，「主人」+「慣性導航系統」台詞確認是機器人守衛非人類）在旁戒備 → 悠妮失憶、記得「從很遠的地方（馬拉大陸）來」、噩夢閃回 → 索爾堅持送她回馬拉大陸，亞雷斯一度反對後同意同行 → 決定啟程 | `proof_01_field_rest.png`（草地上兩人對話「怎麼啦？你」）、`orig_02_dialog_03_ares.png`「拜託，今天休息吧！」、`orig_02_dialog_05_yuni.png`「我們..是從哪裡來的?」 | `full_story_auto.md:4828-4915` |
| ③ | `FDTXT_001` | **抵達馬拉大陸外海小島 → 遭遇海盜 → 哈瓦特/哈諾出手相助**（已有精校轉錄，見 `extracted/story/序章_transcript.md`） | `proof_02_pirate_prebattle.png`（索爾/悠妮/蓋亞 + 3 海盜 + 1 機械兵已在最終戰鬥位置對峙）、`orig_02_dialog_04_pirate.png` | `full_story_auto.md:637-823` |

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

### 2.4 主角隊位置/朝向：位置正確，朝向未實作（低，非 bug）

- **位置**：`ch01.json` 的 `deploy_cells`（`[[7,20],[10,21],[8,22],[11,23]]`）與
  `map0_units.json` 的 `own_deploy` 完全一致，後者是直接從原版 FDFIELD「出場位置」資源匯出
  （doc23 §4：FDFIELD 每張地圖含構成/控制+寶箱/出場位置 3 個資源），**位置本身對齊原版，無需修正**。
- **朝向**：`remake/internal/battle/event.go`、`model.go` 全域搜尋 `Facing`/`Direction` **零命中**——
  remake 的 Unit 資料結構完全沒有「朝向」欄位，主角隊固定用同一張預設朝向 sprite。原版是否有
  進場朝向規則，本輪未見對應的 EXE 反組譯記錄，不確定原版本身是否有此機制（§1.2 已確認原版
  「直接定位、無移動」，若原版單位貼圖本身也不分朝向，則remake 現狀就已一致）。**列為低優先、
  待後續 RE 是否有朝向機制後再決定要不要補**，不在本篇判為明確 bug。

---

## 3. 修復優先序

| 優先度 | 項目 | 工作量估計 | 依據 |
|---|---|---|---|
| 1 | **`ch01.json` `initial_groups` 移除 `10, 11`**（§2.1） | 極小：改 1 行 JSON | 三方交叉證據（turn_events.json 全域搜尋、doc25 §7.5.1 反組譯、青衫攻略）一致排除 |
| 2 | **補開場三幕劇情**（§2.2）| 中：3 個 story 節點 + 對白文字（②③已有現成文字，①需校正 speaker） | dosbox 截圖 + FDTXT 原文逐句對應，內容已齊全，缺的是接進 remake |
| 3 | （隨 1 一併解決）敵方組成回歸純海盜（§2.3）| 0（附帶效果） | 同上 |
| 4 | 朝向機制（§2.4）| 待評估：先確認原版是否真有此機制 | 目前無反組譯依據，暫不判定為 bug |

## 4. 未盡事項

- FDTXT_033/032 的 speaker 標籤誤判需要人工重新指認（不影響劇情內容本身的可信度，只影響
  「這句是誰說的」標註），建議下一輪與「speaker→頭像機制 RE」的既有已知風險一併處理。
- group 10/11 在原版究竟是否有其他觸發路徑（例如某個尚未反組譯的分支/世界地圖 handler），
  本輪只確認了「turn_events 與章節0 cutscene 這兩條已知路徑都不會觸發它們」，未做窮舉排除；
  如果之後想讓這兩組「聖騎士/戰士」在原版真的會用到的地方登場，需要另開反組譯任務。
- 朝向（facing）機制原版是否存在、如何運作，本輪未反組譯，留待後續。

> 相關：`docs/knowledge-base/25-battle-event-system.md`（§7.5.1 staging 反組譯+dosbox 複驗）、
> `docs/knowledge-base/28-chapter-objectives-and-recruits.md`（青衫30關目標表）、
> `docs/knowledge-base/42-re-vs-remake-gap-audit.md`（機制落差稽核，本篇補的是「內容/資料層」落差，
> 非機制層）、`extracted/orig_play_notes.md`（今日 dosbox 實機筆記）。
