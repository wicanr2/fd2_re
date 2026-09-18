# 117 — 先修寶箱與 HUD（#44／#43），再做第十二章工作單元（北山道，map 11）（2026-09-18）

> 這是 [`111`](111-goal-original-parity-campaign-20260915.md) 的下一個工作單元，接在
> [`116`](116-goal-readme-progress-and-ch11-parity-20260918.md)（#42 README 逐章進度、第十一章已
> passed，提交 `584bceb1`）之後。接手時先讀 `CLAUDE.md`，再讀 `111`，再讀本檔。本檔只定義這一輪的
> 順序、門檻與交付物；證據分級與位址規則仍以 `56`／`57`／`58` 為準，結果寫回那三份、
> `91-worklist-history.md` 與台帳，本檔不承載結果。

## 使用者定案（2026-09-18）

| 項目 | 定案 |
|---|---|
| 這一輪的順序 | **先修寶箱與 HUD 兩條**（[#44](https://github.com/wicanr2/fd2_re/issues/44)、[#43](https://github.com/wicanr2/fd2_re/issues/43)），再做第十二章工作單元 |
| 敵人撿走的寶物要掉出來 | 這是戰役分支的一環：天空之城鑰匙那組道具靠它取得（`0x24B14(0x64)` 的 gate）。第十一章已把 `DeathEffect`／`DeathReward` 接上，之後每一章的收據都要維持這個行為 |
| 建構槽政策 | 沿用 [`114`](114-goal-boosted-slot-and-ch09-parity-20260917.md)：`--levels-per-chapter 6 --seed 4 --event-states 7:17=1 --boost-base-ap 200 --boost-base-dp 0 --boost-base-dx 60`，target 改 11 |
| 回歸範圍 | 已 passed 的章（第四～十一章）不因為新修正而重跑；第十二章的收據就是這一輪的驗收。[#41](https://github.com/wicanr2/fd2_re/issues/41) 也不另外重跑 |
| 對拍工具 | 原版側只用 dosgolem `apps/fd2/cmd/oracle`（`tools/dosgolem_oracle.sh`）；缺能力就補 dosgolem 並提交，不改用 DOSBox 當收據 |

## 0. 先修 #44：玩家開箱沒有走原版的可變地圖緩衝

現況（第十一章之後）：敵方 mode 5 那一條已經接上——`0x12263` 把事件格的圖塊字 +1（關著的箱子換成
打開的）並清掉格子事件 byte，繪圖端讀同一份緩衝（`State.NativeMapDrawTiles`）。
**玩家自己踏上寶箱那一條還是舊的**：`battle.ClaimTreasure` 只寫 remake 自己的
`OpenedTreasure map[int]bool`，沒有寫 `NativeEventState`，也沒有走圖塊字 +1。後果有兩個，
第二個不是外觀問題：

1. 玩家開完箱，那一格仍然畫著關著的箱子。
2. `[0x53AD5+event]` 還是 0，敵方 mode 5 之後仍會把那一格當成沒人拿過的事件格走過去撿。
   原版是同一次寫入同時擋掉這件事；「誰先拿到那一格」和天空之鑰那組道具是同一件事。

做法（順序固定，不要跳過第 1 步直接改執行期）：

1. **RE 先行**：反組譯玩家側開箱那條路徑（`0x190AC` 之後的狀態寫入端），列出原版實際寫了哪些位址——
   `[0x53AD5+event]`、可變地圖緩衝的圖塊字與事件 byte、`+0x31..+0x33`（如果玩家側也寫）。
   先查 `58` 的「不要重做」索引與 `docs/data/ida/`；結論連同 caller／consumer 寫進 `58` 再動執行期。
2. **接執行期**：玩家開箱與敵方 mode 5 共用同一份原版狀態。`OpenedTreasure` 不再是平行來源
   （降成投影或直接移除）；`ClaimTreasure` 的呼叫端（`main.go` 的 `TreasureAt`／
   `beginNativeTreasureItemPrompt`）跟著改。
3. **測試**：至少兩個——「玩家開箱之後那一格的圖塊 +1」與「同一格敵方 mode 5 不再重複觸發」。
4. **驗收**：第十二章收據的抽樣要包含「玩家踏上寶箱之後那一格的畫面」；沒有玩家開箱的機會就在
   第十二章的計畫裡安排一次（這一章只有 1 個寶箱，slot 0 物品 56，位置要先從收據確認）。
   `#44` 的 verify 綁 `model.go` 裡 `OpenedTreasure` 自承 remake-owned 的那段註解。

## 1. 再修 #43：HUD 地形描述子讀哪一份地圖

`nativeMapHUDInput` 取游標格的地形描述子時仍讀 `g.m.Tiles`（載入時的可編輯地圖）。原版讀的若也是
可變緩衝，游標停在已打開的箱子上時 HUD 小窗會跟著換。**不要先照「應該一樣」改**：

1. 反組譯 HUD 小窗那條路徑的地形描述子讀取端，看它從哪個位址取 tile。
2. 或做一次同狀態實驗：把游標停在已被撿走的事件格上，比對原版 HUD 小窗。
   第十二章的原版側跑起來之後順手取一次就夠——#44 修好之後玩家開的那一格也可以用。
3. 讀可變緩衝就把重製端改成同一個來源並補收據；讀初始地圖就在 `56` 記下結論與位址，
   把那句自承改掉。

## 2. 第十二章：這一章有什麼（開工前已知，動手時用收據核）

| 項目 | 內容 | 出處 |
|---|---|---|
| 地圖 | map11，28×50（比第十一章窄、更長），我方部署在南端 `(16,49)` 一帶 | `remake/assets/maps/map11/map11_units.json` |
| 開場編組 | `initial_groups [1]`：敵方 10 筆在**北端** y 0..3，另有 group 1 的友軍 1 筆 | 同上、`remake/assets/scenarios/ch12.json` |
| 增援 | 事件 35（`0x34C76`，spawn 在 `0x34C95`）在**第 1 回合結束**登場 group 2（8 筆，`raw_placement_gate 1`） | `docs/data/event_id_groups.json`、`ch12.json` 的 `reinforce_ch12_e35_t1` |
| 還沒有觸發條件的編組 | group 3（6 筆）、group 4（6 筆，全部擠在 `(1,30)`）在劇本裡沒有登場事件——**動手前先查是哪個事件／回合登場**，不要當成佔位列略過 | `map11_units.json` |
| 佔位列 | 29 筆 `group 255`，與第十一章同一種形狀：不能當初始編組（會讓地圖幀第 0 筆沒有來源） | 116 §第十一章 |
| 寶箱 | 只有 1 個（slot 0，物品 56）——這一章正好適合驗收 #44 | `map11_units.json` |
| 戰前 | `ch11_pre`（`0x333F5`）：LOADCH、pan(4,4)、spawn group 1、ACTING 40、pan(11,40)、ACTING 41、重設姿態、text 0、聚焦 slot 0 | `remake/assets/cutscenes/handlers/ch11_pre.json` |
| 戰後 | `ch11_post`（`0x237D5`）：layout、text 3、ACTING 45、text 4、同步隊伍、**JOIN 17**、set_chapter 12 | `ch11_post.json` |
| 戰後城鎮 | `town_ch13` 哈斯米爾之戰（variant 1），祕密商店 selection 1＋scan 105（`0x69`＝alt-f2） | `campaign_full.json` |
| 缺的視圖 | `battle_ch12` 沒有 `native_map_view`／`native_map_hud_inherited`（第十一章這一步是必要的），要照原版 battle_start 補 | `campaign_full.json` |

## 3. 順序（固定）

1. **建槽**：`tools/fd2_chapter_slot.py build --base work/parity-state/ch02-cleared/FD2.SAV --target 11
   --levels-per-chapter 6 --seed 4 --event-states 7:17=1 --boost-base-ap 200 --boost-base-dp 0
   --boost-base-dx 60 --out-dir work/parity-slot-ch12`，manifest 進
   `docs/data/parity-slots/ch12-manifest.json`；合法性檢查計畫 `docs/data/parity-plans/ch12-slot-load.jsonl`
   （LOAD 進北山道城鎮）。
2. **RE 先行**：group 3／4 的登場事件、事件 35 的完整轉寫（不是只有 spawn）、這一章的勝敗處理器，
   連同 §0 的玩家開箱寫入端，一起先查 `58` 的「不要重做」索引與 `docs/data/ida/`，結論寫進 `58`
   再接執行期。
3. **原版側**：`FD2_ORACLE_STATE=<覆蓋層> FD2_ORACLE_STEPS=<預算> FD2_ORACLE_FORCE_ENEMY_CLEAR=1
   FD2_ORACLE_EIP_TRACE=0x13A9F,0x1E54A,0x4E893,0x34C76,0x12CEA tools/dosgolem_oracle.sh
   work/parity-slot-ch12/sample-rN docs/data/parity-plans/ch12-sample.jsonl`。
   **預算先抓 3e10 並抽樣到第 6 回合左右**：第十章與第十一章各作廢過一輪，都是計畫排太多回合。
   這一章地圖是南北向的長條（28×50），驅動端要往北推進。計畫裡安排一次**玩家走上那個寶箱**
   （#44 的驗收材料），並在開完之後把游標停在該格取一次 HUD（#43 的材料）。
4. **重製側**：`tools/chapter_parity.sh 12 work/parity-slot-ch12 <oracle run> <out>`。
   `ch12.json`／`battle_ch12` 的開場編組與視圖照第十一章的作法核一次；改過劇本要重跑
   `tools/export_editor_canonical.py --output remake/assets/editor-canonical --without-animations`。
5. **抽樣截圖**：`tools/parity_sample_sheet.py`，指定點至少包含戰前兩段 ACTING 之後的第一個比較點、
   事件 35 增援之後、玩家開箱前後那一格、戰後城鎮與祕密商店（alt-f2）。
6. **看差異不要只看有沒有過**：非零差異依連通區域分群（教訓
   `gate-passed-is-not-pixel-identical`），同一塊固定大小的區域反覆出現就是真缺口。
7. **收尾**：任一 gate 失敗就開 缺陷／RE待解 issue 修掉重跑；全過後台帳第十二章 `passed`
   （`slot_policy` 照 114）；`56`／`57`／`58` 各一段、`91-worklist-history.md` 一則，教訓寫成規則；
   `fd2_worklist_issues.py pull` → `fd2_worklist.py verify` → `labels --apply` →
   `tools/rebind_string_review.sh`（**最後一次改 Go 檔之後**）→ 全套 Go 回歸 → `render_parity_progress.py`
   → commit，push 前問一次。#44／#43 修好的那條要用 `fd2_worklist_issues.py close` 關掉並留言寫提交。

## 4. 帶過來的未決項（這一輪要記得，不一定要做）

- **[#41](https://github.com/wicanr2/fd2_re/issues/41)**：攻擊演出收尾、`0x1DEAE`、訊息關框 `0x19742`
  三類重繪還沒經過 HUD anchor 入口。這一章收據若出現對應差異就順手接掉並關 #41，沒出現就維持未接。
- **第四～十一章重跑**：116 改了繪圖端的地圖來源（可變緩衝），這一輪的 #44 會再改一次同一層。
  台帳與收據維持 `passed`，下次重跑那些章時要以新收據確認。
- **玩家法術／物品**仍沒有驅動端指令；要納入抽樣得先補 oracle 的控制動詞。
