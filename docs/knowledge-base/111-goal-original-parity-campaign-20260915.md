# 111 — 目標提示詞：全戰役原版體驗一致（2026-09-15 定案）

> 這份提示詞是後續每個工作階段的開工指令。接手時先讀 `CLAUDE.md`，再讀本檔，
> 然後從 [`parity-campaign-progress.json`](../data/parity-campaign-progress.json)
> 挑最小的未完成章節動工。本檔只定義目標、門檻與工作單元；
> 證據分級與位址規則仍以 `56`／`57`／`58` 為準。

## 目標

讓 Go／Ebiten 重製在**原版素材**下，與 DOS 原版《炎龍騎士團 2》在玩家可見
的每一層一致：按鈕與選單順序、對白與過場、戰鬥（指令、移動、攻擊、法術、
物品、敵方回合、事件與增援、死亡與升級）、戰後節點、城鎮／商店／教會／整備
／祕密商店、存檔與讀檔。全部 30 章都要有同狀態的原版收據，不再只靠代表性
抽樣或人工回報。

## 完成定義

當下列條件同時成立，這個目標才算達成：

1. `parity-campaign-progress.json` 裡 30 章每一章的 `status` 都是 `passed`，
   且每一章的收據都能由受版控工具從固定雜湊的 `FD2.EXE`、受版控的控制計畫
   與受版控的建構槽重生。
2. GitHub issue #12（戰鬥交易）、#13（戰間介面）、#14（戰後節點）都已關閉，
   關閉留言各自指向本目標的收據。
3. `58` 覆蓋矩陣中「玩家驗證」欄不再出現「缺完整 E2」；每章都寫成
   `PLAYER-E2` 並附收據連結。
4. `README.md` 與 `REMAKE-STATUS.md` 的進度敘述指向上述收據，不自行保存數字。

## 範圍

**做**：#12、#13、#14 三條 player 分層 issue；工作中發現的重製端缺陷與
原版 RE 待解題目（用 `tools/fd2_worklist_issues.py new` 開成 `缺陷`／`RE待解`）；
dosgolem 為此需要補的能力。

**不做**（使用者 2026-09-15 定案，另開目標或留給人）：
#15 音效人耳、#9 Windows／macOS 實機、#6 身份複核、#7 網頁版、#8 Android、
#4 現代美術主題、#10／#11 編輯器端到端驗收。這些 issue 不關、不改、不順手做。

## 驗收門檻

沿用第 29 戰採用的玩家可見 99% 門檻，拆成四個各自獨立的 gate，
每章收據四個都要過：

| gate | 比較對象 | 判準 |
|---|---|---|
| 行為 | 每個控制序列後的 `ui` 狀態、游標、回合數、我方／敵方存活數、HP／EXP／等級 | 逐序列相等，零容忍 |
| 節點 | 進入的畫面與對白順序（戰前對白、回合事件、戰後對白、城鎮建築、商店子選單） | 序列相等，零容忍 |
| 交易 | 金幣、物品欄、裝備、隊伍順序、酒店寫出的 `FD2.SAV` 槽 bytes | 相等，零容忍；存檔差異要逐 byte 列出並歸因 |
| 畫面 | 每章至少 12 張代表畫面（戰前對白、回合起手、指令環、攻擊演出、敵方回合、戰後對白、城鎮、商店、教會、整備、存檔槽、LOAD） | 320×200 未遮罩，每張差異像素 ≤ 1%（640 像素）；超過就是失敗，未解釋的差異要寫進 `limitations` |

畫面 gate 的 1% 是上限不是目標；已知的字模碰撞、背景殘差要逐張記錄像素數
與位置，累積到同一類差異在三章以上出現時，開一條 `缺陷` issue 修重製端。

## 起點存檔與證據等級

使用者 2026-09-15 定案：**依攻略校準的建構槽跳關 ＋ 章內正常輸入 ＋
`force-enemy-clear` 進戰後節點，這條路徑視同完成該章的 `PLAYER-E2`**。
不另設中間等級。`CLAUDE.md` 與 `56` §2 已同步加上這條例外；比它早的
「修改路徑必須降級」敘述不再適用於本目標的章收據。

規則：

- 建構槽由受版控工具產生：`remake/cmd/fd2-chapter-slot`（Go，在容器內跑）
  從基底存檔套用後續各章戰後 handler 已證實的持續隊伍寫入——`join` 走
  `campaign.MaterializePersistentRecord`（sub_112A5 轉寫）、`grant_item` 走
  0x1c220 語意、`set_chapter` 寫槽 chapter byte；升級走 0x1E292 成長列＋0x1B750
  重算。`tools/fd2_chapter_slot.py build／compare／inspect` 是主機端封裝與
  正對照工具。攻略（`02-game-data-reference.md`，青衫）只當數值旁證；
  沒有證實語意的欄位保持基底存檔的 raw bytes，不猜。
- 每章升幾級、金幣多少不是原版證據，工具預設不升級、不改金幣；要給就用
  `--levels-per-chapter`／`--level-overrides`／`--gold` 明示，manifest 逐項記成
  assumption。正對照（ch01→ch02、ch02→ch03 對真實通關槽）證實：工具寫的
  欄位全部對上，剩餘差異都是遊玩決定的（擊殺經驗與升級、掉落物品、
  戰場座標、`+0x28..+0x36` 殘值）與一個條件式加入。
- 基底一律用 `work/parity-state/ch02-cleared`（最後一份隊伍組成符合一般玩家
  路徑的真實存檔）往前建。`ch03-cleared` 缺鐵諾——它是修改路徑跑出來的，
  鐵諾在第三關倒下，走了 `ch02_post` 的 `any_unit_inactive[6]` 分支；攻略與
  handler 的 else 分支都說一般玩家路徑會讓他入隊。
- `if` 分支用「一般玩家最佳情況」決定：無人陣亡、回合數未超限；能從已建
  隊伍判定的（`roster_has`、物品在不在）照實判定。每個決定寫進 manifest。
- 建構槽必須先通過原版本身的合法性檢查：由 dosgolem 走標題 LOAD 進城鎮或
  戰前對白，原版不拒絕、不出現異常畫面、隊伍名冊與攻略所述相符，才算合法槽。
  原版拒絕就是槽錯，不是原版錯。
- 正對照：先用 ch01–03 已有的真實通關狀態（`work/parity-state/ch0N-cleared`）
  校準工具——用工具建第四章槽，與真實 `ch03-cleared` 的酒店存檔逐 byte 比較，
  差異全部要能歸因到攻略值與實際遊玩值的差（等級、金幣、物品），才可以
  拿工具去建第五章以後的槽。
- 章內一律 BIOS 正常鍵盤輸入，不注入狀態。唯一例外是抽樣完該章的戰鬥節拍後，
  用 `force-enemy-clear` 結束戰鬥以進入戰後節點。
- 收據仍如實記錄槽的來源（工具版本、攻略值、繼承的 raw bytes 雜湊）與
  `state_injections`；這是出處，不是降級。從標題 START 連續通關、正常戰鬥
  勝利條件與跨章持續隊伍漂移不在本目標的驗收範圍，收據的 `limitations`
  提一句即可，不阻擋 `PLAYER-E2`。

## 每一章的工作單元

一章就是一個垂直切片，順序固定，做完再做下一章；不要同時開多章。

1. **建槽**：`tools/fd2_chapter_slot.py build --base work/parity-state/ch02-cleared/FD2.SAV
   --target N-1 --out-dir work/parity-slot-chNN/`，manifest 複製到
   `docs/data/parity-slots/chNN-manifest.json`（版控的是 manifest 與雜湊，不是
   原版 bytes；bytes 由同一命令重生）。跑合法性檢查，收據存 `work/parity-slot-chNN/`。
2. **寫控制計畫** `docs/data/parity-plans/chNN-sample.jsonl`。動手前先 grep
   重製端 `internal/campaign` 的該章資料（handler、事件回合、增援回合、
   死亡事件、對白數），把「這一章有什麼可抽」列進計畫開頭的註解。每章至少涵蓋：
   標題 LOAD → 戰前對白 → 第 1 回合起手 → 指令環 → 一次我方移動＋攻擊 →
   一次敵方回合 → 該章資料裡有的事件／增援回合（推到那一回合為止）→
   若該章有法術或物品可用，各用一次 → `force-enemy-clear` → 戰後對白 →
   城鎮五棟建築各進一次 → 一筆買、一筆賣 → 酒店存檔。
   開跑前報數字：序列數 × 每序列預估秒數 ≈ 總時長，佔幾核。
3. **原版側**：`tools/dosgolem_oracle.sh` 跑計畫，產生 checkpoint 與畫面。
   等待逾時先看 `oracle.log` 尾端的 `error`；是 dosgolem 指令缺口就先補
   dosgolem（用 capstone 掃同族指令一次補齊），再重跑，不繞路。
4. **重製側**：同一個槽、同一份輸入序列，由重製端測試重播（參考
   `town_shop_early_mid_late_e2_test.go` 的做法），輸出同格式的 checkpoint。
5. **比較**：`tools/verify_chapter_parity.py chNN` 產生
   `docs/data/ui-traces/parity-chNN.json`，四個 gate 各自輸出 pass／fail 與差異明細。
   任一 gate 失敗：開 `缺陷` issue（重製端）或 `RE待解` issue（原版語意不明），
   修掉、重跑，直到過為止；修不掉就把該章標 `blocked` 並寫原因，不硬過。
6. **登記**：更新 `parity-campaign-progress.json`（該章 `status`、收據路徑、
   dosgolem commit、槽雜湊、日期、limitations），`fd2_worklist_issues.py pull`
   ＋ `fd2_worklist.py verify`，提交。`57`／`58` 只加一行指向收據。

進度台帳 `docs/data/parity-campaign-progress.json` 是 JSON 不是散文，每章一筆：
`chapter`、`status`（`todo`／`slot-ready`／`oracle-done`／`passed`／`blocked`）、
`receipt`、`slot_sha256`、`dosgolem_commit`、`date`、`limitations[]`。
`todo` 以外的每一筆都要有可重跑的 `verify` 命令。頂層另有
`all_chapters_passed`（布林），只有 30 章都 `passed` 時由工具寫成 `true`；
issue #14 的 `verify` 就盯這個欄位。

## 開工順序

- **第 0 步（已完成，2026-09-15，提交 `682d6359`）**：戰間介面早／中／晚
  三章收據升 `passed`，issue #13 關閉。它的控制計畫
  （`town-shop-ch{06,13,27}-e2.jsonl`）、重製端重播測試
  （`town_shop_early_mid_late_e2_test.go`）與驗證工具
  （`verify_town_shop_early_mid_late_e2.py`）是後面每章工作單元的範本。
- **第 1 步**：寫 `tools/fd2_chapter_slot.py`，用 ch01–03 真實狀態
  （`work/parity-state/ch0N-cleared`）做正對照校準；同時建立
  `docs/data/parity-campaign-progress.json` 台帳（30 筆 `todo`，
  ch01–03 可先用現成通關狀態補收據）。
- **第 2 步**：從第 4 章起逐章推進。每章一個工作階段，做完就提交；
  同一階段不要跨章。
- 每五章回頭看一次累積的畫面差異分類，決定哪些要開 `缺陷` issue。

## 紀律（本目標特別容易踩的）

- 慢迴圈：一章原版側估 15–40 分鐘。開跑前把要回答的問題寫清楚，
  跑完沒產出先拿掉 `| tail` 看完整輸出，不猜。
- 同一章第二次卡在同一處，先停下來看是不是 dosgolem 缺口、槽不合法、
  或計畫按錯順序（畫面上開著吃方向鍵的選單），不要再送一輪。
- oracle 跑的時候不要同時起 Go 全套件回歸。
- `force-enemy-clear` 只在抽樣完該章戰鬥節拍之後用，收據記錄它；
  用在抽樣之前就等於沒抽到戰鬥，那一章重跑。
- 攻略是旁證，欄位語意來自 `58`；攻略與原版執行結果矛盾時以原版為準，
  並把矛盾記進收據。
- 不改 `#15`／`#9`／`#6`／`#7`／`#8`／`#4`／`#10`／`#11`；碰到相關內容只在
  收據 `limitations` 提一句。
- 提交身分 `wicanr2@gmail.com`（repo-local）；commit 不放 session 連結。
- 每批 Docker 工作後 `docker ps -a` 檢查 FD2 容器，禁止任何 prune／rmi。
