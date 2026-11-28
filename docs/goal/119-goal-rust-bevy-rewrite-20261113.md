# 以 Rust／Bevy 改寫重製引擎

## 定案（使用者 2026-11-13）

| 項目 | 決定 |
|---|---|
| 推進方式 | 凍結 Go 版，直接改寫 |
| 繪圖與平台層 | Bevy |
| 動機 | 與 dosgolem-rust 統一工具鏈；長期維護與擴充引擎；處理 Windows 閃爍、效能與當機；網頁與手機版 |
| 本輪產出 | 計畫書，加上第一個可驗證切片 |

## Go 版凍結的意思

- 凍結點是 `v.1.1.11-20261113`，來源提交 `5186e022`；之後 `remake/` 不再新增遊戲功能，也不再發新版。
- Go 原始碼與測試完整保留在 `remake/`，切換後也不刪除；Rust 程式另放 `remake-rs/`，兩者並存。
- 凍結的 Go 版改當「可執行規格」。它的行為已累積大量原版對拍證據，Rust 版逐模組和它比對輸出。
- 允許的例外只有一種：在 `remake/cmd/` 下新增只輸出黃金資料的工具，不改任何遊戲行為。
- 玩家回報的 Go 版缺陷（#303～#305）不在 Go 修，改列為 Rust 版的驗收項目。真的需要緊急修 Go，另外徵得使用者同意。
- 2026-11-14 使用者同意的例外：玩家回報的指令經驗（提交 `1892614b`）與第一次 F2／F11 卡死（`ea7ef639`）在 Go 修正，發 `v.1.1.12-20261114` 修正版。兩項修正同時列為 Rust 版的驗收項目；後續相關缺口見 #315～#318。
- 2026-11-15 使用者同意的例外：F2／F11 切到 HD 與 HD 設定啟動太慢（#317、#318）在 Go 修正（戰鬥 HD 畫格延後解碼、分組並行載入、啟動改背景載入，提交 `a399da4a`），發 `v.1.1.13-20261115`。R6 的 HD 主題載入照這一版的 Go 行為移植。
- 2026-11-15 使用者再同意一次例外：HD 美術下每句對白期間整張地圖與地圖人物變回原版（玩家回報，v.1.1.11 起就有）。改為 HD 美術時只疊對白實際寫入的像素（提交 `ce192d22`），發 `v.1.1.14-20261115`。R6 的 HD 對白照這一版的 Go 行為移植。
- 2026-11-16 使用者同意第四次例外：遊戲內滑鼠（對應成按鍵：指向移游標與選取列、左鍵 Enter、右鍵 Esc、滾輪上下）、手把（十字鍵／左搖桿方向、A Enter、B Esc、X Space、Start 開設定）與 F12「手把與自訂按鍵」設定面板（介面參考 mm3，設定存 `controls-settings.json`；除錯 HUD 改 Ctrl+F12），提交 `7c710b34`，發 `v.1.1.15-20261116`。Rust 版照這一版的 Go 行為移植（`fd2-app` 的輸入對應層已預留）。
- 原版 `FD2.EXE` 仍是最終判準，dosgolem 對拍、E0／E1／E2 證據分級與「RE → 規格 → 實作」閘門全部照舊。Go 版只是較便宜的中間對照，不能取代原版證據。

## 規模

Go 版正式程式約 13.7 萬行，測試約 15 萬行（2026-11-13 以 `wc -l` 量測）。

| 層 | Go 套件 | 正式程式行數 | 是否依賴 Ebiten |
|---|---|---|---|
| 原版格式 | `fdicon` `fdsave` `afm` `fdtxt` `fdother` `dato` `figani` | 約 1.3 萬 | 否 |
| 規則核心 | `battle` | 20931 | 否 |
| 戰役與內容 | `campaign` `indexedmap` `battlepresent` `ending` `editorcanonical` `localization` `musiccatalog` | 約 2.4 萬 | 否 |
| 遊戲程式 | `cmd/fd2`（255 檔：`native_*` 105、`modern_*` 107、`player_*` 12 等） | 77697 | 是 |
| 工具 | `cmd/fd2-asset-import` 與各 oracle 工具 | 約 0.5 萬 | 部分 |

`internal/` 的相依方向是單向的：格式層 → `battle` → 戰役與內容層，沒有任何套件依賴 Ebiten。這是改寫能逐層推進的前提。

## 架構

新程式放在 `remake-rs/`，是一個 Cargo workspace，和 `remake/` 並存。`remake/assets/` 的劇本、語系、HD 清冊與存檔格式沿用不改，Rust 版直接讀同一份資料與玩家既有存檔。

| crate | 職責 | 對應 Go | 依賴 Bevy |
|---|---|---|---|
| `fd2-formats` | 原版檔案格式：存檔 ABI、字表、圖格、動畫、頭像 | 格式層七個套件 | 否 |
| `fd2-rules` | 戰鬥、AI、事件、經驗、商店等決定性規則 | `battle` | 否 |
| `fd2-campaign` | 劇本節點圖、戰役流程、結局、編輯器資料、語系、音樂清冊 | 戰役與內容層 | 否 |
| `fd2-game` | 遊戲狀態機：標題、劇情、戰場操作、城鎮、整備、存讀檔 | `cmd/fd2` 的流程部分 | 否 |
| `fd2-app` | Bevy 外殼：視窗、繪圖、輸入、音訊、HD 主題、工具列 | `cmd/fd2` 的畫面與平台部分 | 是 |
| `fd2-tools` | 素材匯入、黃金資料比對、oracle 橋接 | `cmd/fd2-*` 工具 | 否 |

設計原則：

- **遊戲邏輯不放進 ECS。** `fd2-game` 以下都是一般 Rust 型別與純函式，給定狀態、輸入與亂數種子就有固定結果。Bevy 只負責把狀態畫出來、收輸入、放聲音。這樣規則可以在沒有 GPU 的容器裡測試，也能讓 dosgolem-rust 直接引用。
- **固定邏輯幀。** 原版行為以 tick 計時，`fd2-game` 用固定步長推進；畫面幀率與邏輯幀分開。Go 版「Draw 在沒有 Update 時也會被呼叫」的時序問題在設計上就不存在。
- **介面窄、內部深。** 每個 crate 只公開少數型別與函式；原始位址、偏移與證據等級寫在程式註解與 `docs/data/`，不靠命名承載。
- **失敗即關閉照舊。** 未知行為回傳錯誤或拒絕操作，不猜測。

## 驗證策略

每個移植單位都要同時通過三層：

| 層 | 內容 | 能證明什麼 |
|---|---|---|
| 1. 移植單元測試 | Go 的 `_test.go` 逐條改寫成 Rust 測試，期望值不改 | Rust 版重現 Go 版已知的邊界與拒收行為 |
| 2. Go 黃金差分 | 凍結的 Go 版輸出黃金 JSON（固定輸入、隨機輸入、真實原版檔），Rust 版逐欄比對 | 兩版在大量輸入下行為相同 |
| 3. 原版 oracle | 既有 dosgolem 收據與章節對拍照舊執行 | 和原版一致；只有這一層能升格 E2 |

黃金差分的隨機輸入以固定種子產生，種子與輸入雜湊寫進收據。第 1、2 層只證明 Rust 版和 Go 版一致，不能單獨宣稱忠於原版。

## 工具鏈

| 項目 | 選擇 |
|---|---|
| Rust | 1.99.1、edition 2024，和 dosgolem-rust 相同 |
| 映像 | `tools/docker/fd2-rust.Dockerfile`；所有建置與測試在 Docker 內執行，`CARGO_HOME` 與 `target/` 放在 `work/rust-build/` |
| Bevy | 版本由 `Cargo.lock` 鎖定；升級另開工單，不隨手更新 |
| Windows | `x86_64-pc-windows-gnu`（mingw-w64），沿用本機 Wine＋Xvfb 自檢 |
| macOS | osxcross（SDK 15.5），沿用 `remake/packaging/Dockerfile.osxcross` 的工具鏈，只做靜態檢查 |
| Linux | AppImage，沿用既有打包流程 |
| 網頁、手機 | wasm32 與 Android／iOS 在 R7 才處理，R0 只確認 Bevy 能建 wasm |

## 階段

| 階段 | 內容 | 完成條件 |
|---|---|---|
| R0 基礎 | workspace、Docker 映像、黃金差分框架、Bevy 最小視窗 | 映像可重建；`cargo test` 通過；Bevy 視窗在 Xvfb 下產出截圖；確認 CJK 字型（含 TTC）與 OGG 播放可行 |
| R1 格式層 | `fd2-formats` 移植七個格式套件 | 三層驗證通過；對 `org_game` 原版檔的解碼結果與 Go 黃金資料逐位元相同 |
| R2 規則核心 | `fd2-rules` 移植 `battle` | 三層驗證通過；Go 的戰鬥決定性測試以相同種子產生相同結果 |
| R3 戰役與內容 | `fd2-campaign` | 讀入全部 `remake/assets` 劇本與語系，與 Go 黃金資料相同 |
| R4 遊戲狀態機 | `fd2-game`：無畫面跑完標題、劇情、第一章戰鬥、城鎮、存讀檔 | 第一章受控流程的狀態軌跡與 Go 黃金軌跡相同；能讀 Go 版的存檔 |
| R5 Bevy 外殼（原版美術） | `fd2-app`：原版像素繪圖、對話框、選單、音訊、輸入 | 開場與第一章畫面和 Go 版同狀態截圖逐像素一致 |
| R6 HD1 與工具列 | HD 主題、工具列設定、畫質與速度 | HD1 清冊全部素材可載入；切換與設定保存相容 |
| R7 平台 | Windows、macOS、Linux、網頁、手機打包 | 三平台完整包可組裝；Windows 實機驗收 #303～#305 |
| R8 切換 | 每章先依該章重播的覆蓋率補移植第一章以外的流程程式，再跑 30 章受控流程對拍、正式發行 | 依 `111` 的章收據規則重跑；通過後 Rust 版成為正式主線，Go 版封存 |

各階段工單：R0 [#306](https://github.com/wicanr2/fd2_re/issues/306)、R1 [#307](https://github.com/wicanr2/fd2_re/issues/307)、R2 [#308](https://github.com/wicanr2/fd2_re/issues/308)、R3 [#309](https://github.com/wicanr2/fd2_re/issues/309)、R4 [#310](https://github.com/wicanr2/fd2_re/issues/310)、R5 [#311](https://github.com/wicanr2/fd2_re/issues/311)、R6 [#312](https://github.com/wicanr2/fd2_re/issues/312)、R7 [#313](https://github.com/wicanr2/fd2_re/issues/313)、R8 [#314](https://github.com/wicanr2/fd2_re/issues/314)。R0～R4 各階段收尾時，在下方「進度」加一個標題寫明該階段已完成的小節（格式見 R0），工單的機器檢查以這個標題判定；其他段落不要寫出那個字串，以免誤判。

R4 的流程與畫面拆分計畫見 [120](120-goal-rust-r4-game-state-split-20261114.md)。R5 的原版美術外殼計畫（CPU 合成、之後加 GPU 加速模式）見 [121](121-goal-rust-r5-original-art-shell-20261115.md)。

R1 到 R3 彼此有相依但可以按 crate 平行；R4 之後必須依序。每個階段完成時更新 `56` 與 `58`，切換前 Go 版的發行包維持可用。

## 風險

| 風險 | 處理 |
|---|---|
| Bevy 版本更新頻繁、API 變動大 | 鎖定版本；Bevy 只在 `fd2-app`，換版時只動這一層 |
| Bevy 編譯時間長、執行檔大 | 開發用動態連結與增量編譯；發行用 release profile 並量測大小 |
| 中文字型：Bevy 是否能直接讀 TTC | R0 驗證；不行就在匯入時把 TTC 拆成 TTF |
| 逐像素一致：GPU 取樣與 Ebiten 不同 | 原版模式以整數倍放大、最近鄰取樣；R5 用同狀態截圖逐像素驗收 |
| Windows 閃爍的根因仍未知（#305） | 換引擎不保證消失；R7 以實機錄影驗收，不以換 Bevy 當作修好 |
| 改寫期間沒有新版 | Go 版 v.1.1.11 維持可玩；改寫進度只以通過驗收的階段計算 |

## 本輪工作（R0 起步與 R1 第一個切片）

1. 建 `remake-rs/` workspace 與 `fd2-rust` 映像。
2. `fd2-formats` 先移植 `fdsave`（存檔 ABI，813 行，無其他相依），三層驗證中的前兩層：
   - `save_test.go` 與 `keyboard_buffer_test.go` 逐條移植。
   - 新增 Go 黃金匯出工具，對固定與隨機輸入輸出解碼結果，Rust 測試逐欄比對。
3. Bevy 最小視窗與 CJK 字型、OGG 的可行性驗證留到下一輪，結果寫回本文件。

## 進度

### R8 後備介面與小型流程缺口（RUNTIME-E1，2026-11-28）

這一輪補的是缺原版素材時才會走的 TTF 後備介面，以及幾個只有晚期章節走得到的小流程。

流程缺口：
- 指令環在巢狀系統選單開著時，改用 `nativeNestedSystemOverlayState`；圖格或紀錄缺件時四向都不可用。
- 近似戰後節點按確認後前進（`continueApproximatePostbattle`）。
- 第 28 章戰後對話在戰場還沒有原生視圖時，以圖塊對齊的戰鬥鏡頭與游標物化視圖（`materializeInheritedNativeDialogueView`），不對齊或為負值就回錯。
- 戰鬥事件出錯時，原生章節結果進行中就以 `failNativeChapterResult` 擋住啟動。

後備介面：
- 戰場物品：原版面板畫不出來時改用八格 TTF 清單，Enter 只顯示訊息、不使用物品（0x20c6f 的效果／目標表未閉合）。
- 原生指令格：依 0x1CEED 的四列配置畫官方指令名稱，名稱以 1.0 倍起每次縮 0.02 放進 138 像素寬，縮到 0.6 倍以下仍放不進就回錯。
- 戰役介面：流程結束的 GAME OVER、choice 與城鎮的後備選單、ending 結語，以及商店、酒店、整備的後備版面。
- TTF 後備商店的輸入：Rust 原本在原生商店沒有啟動時只接受 Esc 離店，Go `main.go:5242` 的買賣流程沒有移植也沒有標記。現在補上買（選收件人、裝備類先詢問是否裝備，Esc 不裝備也成交、最後扣錢）、Tab 切到賣（選人、選道具，得 75% 價格）與逐層退回。Go 的 `shopReceiverIDs`／`shopSellIDs` 補名冊其餘成員時走 map，順序不固定；Rust 的名冊是 `BTreeMap`，依編號遞增。

驗證：
- 移植 Go 的 `TestApproximatePostbattlePreservesAuthoredIntermissionBoundary` 與 `TestOfficialCommandNamesFitNativeGridCell`（四語系各 35 個指令名放得進格子），並在 `TestNestedSystemOverlayUsesRuntimeSaveAndFD2SAVGates` 補上指令環狀態的分派。另補重製端測試：章節結果失敗、繼承視圖的對齊檢查、後備物品清單、後備商店買賣、六種戰役後備版面以真字型畫出。
- fd2-game 561 項、Clippy 通過；`todo!` 從 120 降到 107。
- 在提交 `e9e30299`（流程缺口）與 `de96dfce`（含其後三個後備介面提交）的獨立 worktree 各重跑第 1～30 章，全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-e9e30299/summary.tsv`、`work/rust-parity-all-de96dfce/summary.tsv`）。

### R8 玩家指令 9 演出（RUNTIME-E1，2026-11-28）

玩家指令 9 的地圖演出照 Go `startNativeCommand9PlayerPresentation` 移植（主證據 `docs/data/ida/fd2_command9_player_ai_presentation_ida.txt`）。開演出前先合成全部畫格，計畫、聲音或素材任一不合就不開：

1. 指令調色盤閃爍：依 `native_command_palette_flash.json` 的指令 9 相位改 DAC 第 0 色，底圖是目前地圖畫面。
2. 效果：目標身上疊 FDOTHER#6 的 27 格。效果段結束時才扣 MP、寫入傷害並推進亂數。
3. 數字：FDOTHER#5 的 22 格傷害數字；沒命中且目標在鏡頭內時寫 MISS 四個字。
4. 停 500 ms，交回基準畫面，續行一次。

派送分支接上 id 9。Go 的指令選單（`nativeCommandTargetSupported`）刻意不放行指令 9：目前只有敵方來源，沒有玩家取得途徑的證據（`docs/data/fd2-hd1-feedback-20261104.json`），Rust 照同樣做法，這條演出只由測試直接呼叫。HD 地圖凍結與 HD 幀屬 R6。

驗證：
- 移植 Go `TestModernHDPlayer9NativeSourcesMatchFullVGA` 的原版索引色部分：命中與不中各一次，27 格效果的圖層來源都是 FDOTHER#6 第 87 格起、位置 (24,−6)；MISS 依序寫 116、117、118、118 四個字形；每一格都與不經合成器的逐位元組重組相同。另補一項端到端測試，以正式呈現與 stepper 走完四段，確認 MP 與 HP 只在效果段結束時寫入，續行一次並交回基準畫面。
- fd2-game 553 項、Clippy 通過；`todo!` 移除 2 處、新增 HD 2 處。
- 在提交 `a302e345`（含天空之鑰 `9d67beec` 與玩家指令 9）的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-a302e345/summary.tsv`）。

### R8 第二十章天空之鑰演出（RUNTIME-E1，2026-11-28）

第 21 戰（map20）勝利後，隊伍湊齊六件材料（道具 0xD1～0xD6）時進入天空之鑰的成功臂，handler 在 0x242C9 呼叫 0x24336。這段演出的參數是寫死的原版契約（`IsRecoveredContract`），被改過就在動任何狀態前拒收。流程如下：

1. 預檢：FDOTHER#34 的 101 格逐格試解、ANI #0 的 96 格畫面與調色盤快照都要完整，並先算出捲動終點驗一次。
2. 捲動：鏡頭與絕對游標每步同移一格，先走 X 再走 Y，直到鏡頭在 (14,8)；每步重繪地圖。
3. FDOTHER#34：以第 0 格疊在地圖上當底，播第 1～68 格，每格等 3 個 BIOS tick，並推進一次 0x4DFCC 的調色盤相位。
4. ANI #0：96 格各帶自己的調色盤，每格 15 ms。
5. 白閃與還原：以地圖調色盤為基準整盤加 63 停 100 ms，再還原停 500 ms。
6. 尾段：換回地圖調色盤播第 69～100 格，最後一格的畫面與調色盤交回地圖，接著續行後面的 beat。

每一格都要畫過才推進。任何一步失敗都還原鏡頭、游標、地圖時鐘、畫面、調色盤與調色盤相位。本輪移植 `native_ch20_sky_key.go` 的 15 個函式，接上 beat、`Update`、`Draw` 與呈現回報。Go 的還原是閉包；快照含不能跨執行緒的 `NativeFrameInput`，Rust 改存快照結構再還原。HD 的天空之鑰世界凍結、預檢與 HD 幀屬 R6。

驗證：
- 移植 Go 的 `native_ch20_sky_key_test.go` 六項：分離素材符合契約、ANI 不完整時拒收、改過的 payload 在動狀態前拒收、演出失敗時完整還原地圖交易，以及第 21 戰兩臂的端到端測試。成功臂從勝利經過全部六個階段、相鄰兩段 ACTING、26 段原生對話、JOIN 24／23 到 `town_ch22`，天空之鑰（0x64）進入持續隊伍，存讀檔後隊伍不變；FirstFrames 的調色盤相位剛好推進 68 次。材料不足臂只播 14 段對話、不開演出也不給天空之鑰，同樣 JOIN 24／23 並存讀檔。這兩項以 `ch21.json` 的出戰順序建持續隊伍投影、直接進第 21 戰勝利邊界，屬 RUNTIME-E1，不代表一般玩家路徑。
- 測試 helper `tick` 補上 `stepNativeCh20SkyKey`，與 Go 相同。
- fd2-game 551 項、Clippy 通過；`todo!` 淨減 2 處（移除 4 處，新增 HD 2 處）。演出畫不出來時 Go 另印一行除錯字，Rust 照其他指令演出的做法由 `load_err` 顯示同一失敗。
- 在提交 `a302e345`（含天空之鑰 `9d67beec` 與玩家指令 9）的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-a302e345/summary.tsv`）。

### R8 地圖事件 61（RUNTIME-E1，2026-11-28）

第 26 章的地圖事件 61 是把控制中樞（道具 0xD0）放進機兵頭部、讓渥德加入。單位行動結束時停在 selector1 為 61 的格子上，`planNativeFieldEvent61` 先驗原版事件狀態與觸發者道具；驗不過就停止寫入並回錯。驗過之後：

1. 觸發者沒有控制中樞：只播 `ch26.json` 第 0 幕第 10 句（FDTXT2），不寫任何狀態。
2. 有控制中樞：先播第 11 句（FDTXT3），收框後以中性戰術重繪為底，把 FDOTHER#45 的 59 格逐格疊上去。每一格要畫過、且 BIOS tick 前進到計畫的延遲才換下一格。
3. 最後一格之後才提交：消耗控制中樞、事件狀態 12 設為 1、渥德的記錄寫進持續隊伍（JOIN 31），接著播第 1 幕的十句（FDTXT4），播完才結束行動。

全軍移動走到 selector1 事件格時暫停，交給事件 61 或 75 的擁有者，事件結束才續走；全軍移動的預檢也補上事件 61 的對白與圖格表。

本輪移植 `event61DialogueActions`、`beginNativeFieldEvent61` 後半、`beginNativeFieldEvent61Presentation`、`stepNativeFieldEvent61Tick`、`persistNativeFieldEvent61Join`、`drawNativeFieldEvent61`、`beginNativeSystemGroupMarchFieldEvent`，以及 `preflightNativeSystemGroupMarchEvents` 的事件 61 分支。Go 的走行續行以指標比對 `g.walk == walk`；Rust 的走行沒有身分，改以「同一個單位、仍停在全軍移動事件」判斷。HD 的 `modernHDFieldEvent61` 凍結、HD 世界與 HD HUD 屬 R6。

驗證：
- 移植 Go 的 `native_field_event61_test.go` 六項，從 `loadGame` 進第 26 章戰場：缺控制中樞只播 FDTXT2、攻擊後等全螢幕演出結束才觸發、使用道具後觸發、59 格演出後提交並寫入渥德、全軍移動暫停與續走、缺圖格表時兩個擁有者都拒收且不寫狀態。Go 在載入後才把 `FD2_ASSET_PACK` 換成空目錄；Rust 不能在行程中改環境變數，改由父行程組一份少了事件 61 圖格表的素材包給子行程。
- fd2-game 545 項、Clippy 通過；`todo!` 淨減 1 處（移除 5 處，新增 HD 3 處與演出畫不出來時的除錯字 1 處）。
- 在提交 `27974569` 的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-27974569/summary.tsv`）。

### R8 死亡獎勵滿欄替換（RUNTIME-E1，2026-11-28）

敵人死亡掉落的物品要給擊殺者，擊殺者八格道具欄已滿時，行動收尾開原版 0x1AA56 的丟棄提示。流程如下：

1. 以目前地圖畫面為底，對話框展開並問要丟哪一件（FDTXT#0x1B1）。
2. 選「否」或按 Esc：問句下方寫放棄訊息（FDTXT#0x1B2），停留後收框，道具欄不變。
3. 選「是」：對話框收起，開擊殺者的八格物品面板。方向鍵走原版選擇器，Enter 先在複本上試換，成功才記下這一格並收面板；面板收完才真正丟棄舊物品、把獎勵接在尾端。
4. 在選擇器按 Esc：面板收完後另開一個對話框顯示放棄訊息（0x1AB1F），道具欄不變。
5. 同一次行動有多筆獎勵時依序接手，全部結束才完成擊殺者的行動。

本輪移植 `runPendingNativeDeathRewards` 在物品面板之後的部分、`handleNativeDeathRewardItemInput`、`finishNativeDeathRewardItemClose`、`beginNativeDeathRewardSelectorCancel`、`finishNativeDeathRewardUI`，以及 `finishNativeSystemEndTurnChoice` 的死亡獎勵分支。HD 的 `modernHDFullReward` 凍結與綁定屬 R6，忠實模式下 `hd` 是 `None`。

驗證：
- 移植 Go 的 `native_death_reward_ui_test.go` 四項：選「否」與 Esc 不動道具欄、丟棄只在面板收完後提交、選擇器取消與兩筆獎勵循序接手、缺來源時失敗即關閉。前三項從 `loadGame` 進第一章戰場，以正式 `draw` 與 `update` 推進，環境變數在子行程設。
- fd2-game 539 項、Clippy 通過；`todo!` 淨減 3 處（移除 4 處，新增 HD 主題底圖 1 處）。
- 在提交 `89566a73` 的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-89566a73/summary.tsv`）。

### R8 戰場玩家狀態面板（RUNTIME-E1，2026-11-28）

在地圖上對單位按確認時，`inspectNativePlayerUnit` 讀該單位的原始記錄：+7 為 0x79 或 +0x1F 為 10 的特殊記錄直接吃掉確認鍵；非強制檢視且 +6 為 2、+5 bit7 為 0、+0x26 為 0 時照常選取；其餘情況開原版 0x17AED 的狀態面板。

- 開啟：以目前地圖畫面為底，面板 12 幀由小到大展開；停在能力值頁。
- 確認鍵：單位有指令列時，下半部先收 7 步再展開 7 步，換成指令頁；再按一次（或沒有指令列時）面板 12 幀收起，回到地圖。
- 每一幀都等畫過才推進（`drawn`），與 Go 相同。

本輪移植 `prepareNativeUnitStatusData`、`nativeUnitStatusData.compose`、0x17AED 面板幀合成（`native_church_status_ui.go` 的幀與區域函式）、`beginNativePlayerStatus` 與 `advanceNativePlayerStatus`。素材全部讀分離素材包，缺件時回錯誤、不開面板也不選取單位。HD 面板預檢（`preflightModernHDItemPanel`）屬 R6，`snapshotNativeStatusTransition` 只留忠實模式用到的轉場種類與頁別，`source`／`data` 複本隨 HD 欄位補。教會的狀態檢視在 Rust 走 `native_church_panels.rs` 的既有實作，Go 的 `beginNativeChurchStatus` 等教會入口仍是 `r8`。

驗證：
- 移植 Go 的 `TestNativePlayerStatusTwoPagesWaitForDrawAndRestore`、`TestNativeChurchPanelLifecycleMatches17AEDSchedules`，以及 `TestNativePlayerStatusGatesAndFailedResourcesDoNotSelect` 先前未移植的第三段（空素材包要失敗即關閉，在子行程設 `FD2_ASSET_PACK`）。另補一項用分離素材包真實資源開啟、翻頁到關閉的重製端測試。
- fd2-game 535 項、Clippy 通過；`todo!` 淨減 1 處（移除 2 處，新增 HD 預檢 1 處）。
- 在提交 `39a4de7c` 的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-39a4de7c/summary.tsv`）。

### R8 酒店讀檔（RUNTIME-E1，2026-11-28）

酒店服務 2 的四槽讀檔接上 `confirmNativeHotelLoadSlot`（Go `native_hotel_ui.go:259`）。有 `FD2_NATIVE_SAVE` 時走原版章節槽復原：該章在戰間 gate 表標成要整備（第 22～24、27～29 章）就切到 `load_blocked` 顯示 0x1DF，不寫任何隊伍狀態；其他錯誤只顯示訊息。沒有原版存檔時走 JSON 槽，成功後留在讀出的那間酒店並保持服務 2，Esc 再沿已編寫的邊回城鎮。復原與 JSON 讀檔本身早已移植，本輪只接上分派。

驗證：移植 Go 的 `TestNativeHotelLoadPublishesAtHotelThenReturnsThroughAuthoredTown`、`TestNativeHotelLoadRejectsEveryPreparationGateBeforeRosterPublication`、`TestNativeHotelLoadRejectsTamperAndMissingAuthoredReturn`、`TestNativeHotelLoadUsesJSONBoundaryReader` 與 `writeNativeRestoreFixture` 測具，Go 以 `t.Setenv` 設的環境變數改在子行程設。在提交 `9e624301` 的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-9e624301/summary.tsv`）。

### R8 可編輯法術後備：玩家施法確認與 AI 法術（RUNTIME-E1，2026-11-28）

這條路徑是重製端的可編輯法術後備，不是原版指令流程。玩家側在單位沒有原生指令列、或缺官方實體名稱與介面調色盤時，才從指令環開 `spells` 清單（Go `main.go:6071`）；AI 側由正規化規劃器給出 `spell_id`。兩邊都走 `SpellBook`／`CastArea` 交易，不含原版 0x1598A 的命令評分與索引色法術演出。

本輪移植的範圍：
- `confirm` 的施法分支（Go `main.go:7484`..`7543`）：施法距離與單體法術的陣營閘門、`CastArea` 結算、死亡獎勵、結果訊息，攻擊法術以首個命中目標接全螢幕攻擊演出。
- `executeAISpell` 與 `actorDisplayName`（`ai_spell_execute.go`）：目標、距離、法術列、MP 任一不合就在扣 MP 前回錯；`CastArea` 沒有結果時退回 MP。`ai_step_act` 失敗時寫 `load_err` 並停住 AI。
- `localizedSpellResult`（`locale.go:213`）。

Go 的 `g.rng` 為 nil 時，玩家施法會在 `CastArea` 擲骰處 panic。Rust 改在扣 MP 前寫 `load_err` 並保留選目標狀態，屬失敗即關閉的差異。

驗證：
- 移植 Go 的 `ai_spell_consumer_test.go` 三項與 `TestOfficialLocalesCoverSystemAndSaveMessages`，另補玩家施法確認兩項重製端測試（射程外、陣營不符、封咒、缺亂數都不動 MP 與行動旗標）。
- fd2-game 528 項、Clippy 通過；`todo!` 再減 2 處。
- 在提交 `86fec1b3` 的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-86fec1b3/summary.tsv`）。

### R8 玩家指令施放：指令 23 與傳送物品（RUNTIME-E1，2026-11-28）

傳送術（指令 23）和傳送物品（type 23）共用同一段目的地流程。選定目標後換成模式 6 的目的地選擇器，游標移到目標身上，Esc 退回指令格或物品面板。確認目的地時依序做：

1. 0x115b6 的落點判定：目的地不能有其他在場記錄，目標的移動成本列在該地形不是 20。
2. 在記錄複本上跑 0x2218A 交易，扣 MP、算經驗累加值。
3. 預算 0x22253 的消失段與出現段兩份演出。

三步都成功才開 0x1D6C8 調色盤前奏。前奏八個階段畫完才接消失段，出現段結束時才把記錄寫回，指令 23 再發經驗。任何一步失敗都不動 MP、行動旗標與目標座標。本輪移植 Go 的 `beginNativeCommand23Destination`、`nativeRelocationDestinationAllowed`、`applyNativeRelocationDestination`，以及 `applyNativeTargetItem` 的傳送分支。HD 的 `unitPresent`／`hdNext` 預檢串接屬 R6。

指令 0～35 中只剩 9 沒有玩家入口，和 Go 一樣不在選目標白名單裡；繪製段的 `todo!` 也只剩它的演出工作。

驗證：
- 移植 Go 的 `TestNativeCommand23FormalConfirmCancelAndFullPresentation`、`TestNativeCommand23InsufficientMPOrMissingArtIsAtomic`、`TestNativeRelocationUsesSeparateDestinationCursorAndMode6Legality`，以及三者共用的 `completeNativeUnitPresentGame` 合成測具。正式確認走完 96 個演出階段，中途目標確實離開地圖。
- fd2-game 522 項、Clippy 通過；`todo!` 剩 126 處。
- 在提交 `9274f12f` 的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-9274f12f/summary.tsv`）。

### R8 玩家指令施放：指令 10～12（RUNTIME-E1，2026-11-28）

指令 10～12 在地圖畫面上演出：三種取樣面依原版順序輪播 60 格，再播傷害數字，扣血在結果段才寫入。指令 11、12 先跑 0x2189A 的十次半徑前奏（第 22 章戰後也用同一段），前奏收尾才進主段並播主音效。本輪從 Go 移植建立函式與原版索引色繪製（`native_command10_12_presentation.go`）。

`start_native_2189a` 先前在 `native_cmd1012` 存在時直接拒收。Go 在這個情況只多凍結 HD 前奏用的地圖輸入、徑向 LUT 與穩定場景，錯誤也只記在工作上不擲出，所以原版路徑拿掉這道拒收；HD 欄位屬 R6。

繪製段剩下的 `todo!` 只剩玩家指令 9 的演出工作。指令 9 和 Go 一樣不在選目標白名單裡。指令 23 接在下一輪，見「R8 玩家指令施放：指令 23 與傳送物品」。

驗證：移植 Go 的 `TestNativeCommand1012FormalPlayerConfirmAndReturn`（10、11、12 各走一次正式確認，11、12 經 0x2189A 前奏，結束時 MP 扣掉法術表的消耗、選取與範圍模式都還原）。步進、回滾與前奏消失的測試先前已移植。fd2-game 519 項通過；`todo!` 剩 128 處。

在提交 `b512d2af` 的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-b512d2af/summary.tsv`）；第 22 章戰後的 0x2189A 也走這次放寬的入口。

### R8 玩家指令施放：指令 32～35（RUNTIME-E1，2026-11-28）

這四個指令走 0x27FC9 複合演出：九格前奏、施放者特效、FDOTHER 共用段（#65～#68）、地圖調色盤漸變，最後回到地圖畫面播尾段。Rust 先前只有步進與取消函式，本輪從 Go 移植建立函式與原版索引色繪製：

| 指令 | 共用段 | 地圖尾段 | 寫入時機 |
|---|---|---|---|
| 32 | #65，含共用尾段 | FDOTHER #6 受擊、閃爍、傷害數字 | 數字段第一格 |
| 33 | #66 | 回復效果、遮罩閃爍、回復數字 | 數字段第一格 |
| 34 | #67 | 三段效果、遮罩、數字（0x22721／0x22866／0x22997） | 各段數字的第一格 |
| 35 | #68，含共用尾段 | 三段效果、遮罩、數字（三次 0x22D1B） | 各段數字的第一格 |

交易後的地圖畫面在建立時就用戰場複本預先合成，演出到寫入那一格才換上；取消或失敗時亂數狀態與地圖畫面都還原。指令家族的發經驗函式補上 32～35。HD 複合畫格的綁定與繪製屬 R6。

指令 10～12 接在下一輪，見「R8 玩家指令施放：指令 10～12」。

驗證：
- 移植 Go 的 `native_command32_presentation_test.go` 4 項；33～35 的步進、回滾與取消測試先前已移植，本輪補上三者的原版素材端到端測試（`TestStartNativeCommand33／34／35PresentationUsesOriginalAssetsEndToEnd`）。
- fd2-game 518 項通過；`todo!` 剩 128 處。
- 在提交 `ec208d28` 的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-ec208d28/summary.tsv`）。

### R8 玩家指令施放：指令 24、28～30（RUNTIME-E1，2026-11-28）

這四個指令都走 0x276EC 衍生打擊：九格淡入前奏、施放者特效、BG.DAT 二十格背景滑移、目標受擊。Rust 先前只有步進函式，本輪從 Go 移植建立函式與原版索引色繪製：

| 指令 | 施放者 selector | 目標 | Go 來源 |
|---|---|---|---|
| 24 | 32 | 單體 | `native_command24_presentation.go` |
| 28 | 19 | 確認單位起算的效果範圍，每個目標八段受擊 | `native_command28_presentation.go`，畫格整批預先合成，掛在指令 29 的工作上 |
| 29 | 34 | 以施放者為中心的效果範圍，逐一滑移到每個目標 | `native_command29_presentation.go` |
| 30 | 18 | 選取前游標到確認游標的直線 | `native_command30_presentation.go`，共用指令 29 的建立函式 |

共用部分一併移植：戰鬥底圖組裝（`localizedNativeCommand24CapturedBase`）、指令音效庫檢查（`requireSeparatedCommandSounds`、`separatedCommandSound`），以及消耗亂數前的靜態來源預檢（`preflightNativeDerivedStrikeSources`）。HD 背景來源（`modernSplitStrikeScenery`）在原版路徑也照 Go 先跑，背景或平台幾何不合時在消耗亂數前拒收。HD 分割打擊的繪製與 HD 庫 stage 屬 R6。

完成回呼與 Go `main.go:7843`、`8123` 相同：傷害大於 0 才發死亡獎勵、面向游標、發衍生打擊經驗；指令 30 收尾另清掉選取前游標。

指令 32～35 接在下一輪，見「R8 玩家指令施放：指令 32～35」。

驗證：
- 移植 Go 的 `native_command24_presentation_test.go` 8 項、`native_command28_presentation_test.go` 3 項、`native_command29_presentation_test.go` 5 項、`native_command30_presentation_test.go` 4 項，另寫指令 24 走玩家入口的測試。原版標記的幀位、MP 與 HP 發布次數、晚期失敗整筆回滾、預檢失敗不動亂數都與 Go 的斷言相同；指令 30 的演出結果與純狀態的 `ExecuteNativeCommand30` 逐欄相同。
- fd2-game 511 項通過；`todo!` 剩 128 處。
- 在提交 `d172ff5f` 的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `0abad3d5` 相同（`work/rust-parity-all-d172ff5f/summary.tsv`）。

### R8 玩家指令施放：指令 1～8、13～22、25～27、31（RUNTIME-E1，2026-11-27）

Go 的 AI 與玩家共用同一個指令建立函式，差在計畫來源與完成回呼。Rust 先前只移植了敵方分支，指令 3、6、7 甚至明確拒收我方施放者。本輪照 Go 的結構把指令 2、3、5（含 1、4）、6、7、8、回復 13～16、尾段 20～22／25～27 拆成 AI 入口與共用建立函式：

| 項目 | 敵方 | 玩家 |
|---|---|---|
| 計畫來源 | 0x15311 選定的目標格（`nativeAIActionOrigin`） | 確認的目標；指令 6 以確認時的游標格為中心 |
| 死亡獎勵 | 每個結果都發 | 只發命中的目標 |
| 經驗 | 不發 | `native_command_experience` 依家族發 |
| 收尾 | 交給 AI 的後續 | 清目標標記、範圍模式回 1、結束選取 |

玩家的尾段 20～22、25～27 先播 0x1D6C8 調色盤前奏的八個階段；指令 17～19 用同一個前奏（`startNativeCommandPalettePresentation`，本輪移植原版索引色分支與繪製），八個階段都被畫面承認之後才跑 0x22721／0x22866／0x22997 交易。指令 31 沒有演出，確認即結算。

指令 24、28～30 接在下一輪，見「R8 玩家指令施放：指令 24、28～30」。

驗證：
- 移植 Go 測試 `TestNativePlayerFormalCommandEntryAndPresentation`（Go 測 3、4、5、7；Rust 另加 1、2、8，各兩種 +6 值）、`TestPlayerNativeCommand6CursorCentersThroughConfirm`（MP 1071、HP 4、亂數 33552 與 Go 相同）、`…Command13…`（回復 60）、`…Command17…`（八階段前不改 MP／AP，結算後 MP 6、AP 116）、`…Command20…`、`…Command25…`。指令 31 另寫 Rust 測試。
- AI 路徑重構在提交 `0abad3d5` 的獨立 worktree 重跑第 1～30 章全數通過，每章筆數與存檔 SHA-256 和 `ec0d4a50` 相同。fd2-game 490 項通過；`todo!` 剩 129 處。

### R8 30 章總表與玩家指令格（RUNTIME-E1，2026-11-27）

`tools/rust_parity_all.sh` 依 [`rust-parity-chapters.json`](../data/rust-parity-chapters.json) 逐章重跑。在提交 `ec0d4a50` 的獨立 worktree 上跑，素材指主工作區的 `remake/assets` 與原版分離包，第 1～30 章全部通過（輸出 `work/rust-parity-all-ec0d4a50/summary.tsv`）。第 22 章存檔 `3c7298cd` 與原版逐位元組相同；第 29 章產出的 `33c3d74d` 正是第 30 章的輸入槽。這是受控樣本的回歸證據，不代表樣本外的玩家路徑已移植。

玩家指令格（選「法術」後的原生指令格）先前整段是 `todo!`，打開就會中止。本輪補上指令格輸入、白名單、選目標候選與目標格、進入／取消選目標，以及確認時的共用游標閘門；指令家族的發經驗函式（`native_command_experience.go`）整檔移植。各指令的玩家演出入口逐一補：指令 0 的建立函式改成由呼叫端傳入完成回呼，AI 與玩家共用，玩家收尾另發經驗。其餘 ID 確認時留在選目標並顯示「玩家演出尚未移植」，Esc 回指令格。`todo!` 剩 131 處。

驗證：移植 `TestNativeCommandTargetFieldMaterializeAndReset`、白名單測試與 `TestPlayerNativeCommand0RunsIndexedPresentationThroughCursorConfirm`（缺抗性表、缺 FIGANI 出處都在交易前拒收；MP 發布 1 次、HP 7 次，與 Go 相同）。fd2-game 483 項通過。

### R8 一般玩家路徑補移植：系統選單、戰場快照與快捷鍵（RUNTIME-E1，2026-11-27）

30 章受控重播都通過，只代表樣本走到的函式已移植。產品碼仍留著 `todo!()` 樁，一般玩家走到就會中止；剩下多少一律以 [`tools/rust_todo_inventory.py`](../../tools/rust_todo_inventory.py) 現場產生，不在文件寫死。本輪從 151 處降到 134 處（R8 84→68、R6 65→64），照 Go 補上：

- 地圖操作：取消移動、選攻擊目標時 Esc 退回指令環、空游標 Esc／Z／Numpad5 輪流選我方單位（`cycleNativePlayerUnit`）。
- 系統游標選單（0x16F55）與巢狀選單（0x19DF7）：結束回合確認、系統選項切換（含音樂與 HUD 閘門 A）、離開遊戲確認、全軍移動、戰場資訊畫面。確認框共用一段建立程序，問句、兩組回覆與 10 格開場全部建好才收起命令框。
- 戰場快照（`battle_snapshot.go`，56「#303 戰場JSON快照」）：戰鬥中 F5 與巢狀選單存檔附快照，F9／LOAD 先正常進場再套用。Rust 的 `Game` 不能複製，能預先驗的都在複本上做；寫回後的視角或可操作檢查失敗時重新進入同一個戰鬥節點，結果與 Go 保留剛進場的狀態相同。
- 快捷鍵：F5、F9、F7 字級、F8 速度；工具列的字級與速度改呼叫同一個方法。

仍失敗即關閉、顯示 Go 的同一句訊息，不中止的有：

- 巢狀選單讀回目前戰況（`prepareNativeContinueFromCurrentSnapshot` 未移植）。
- 有原版目前戰況基底時的原版存檔（`buildNativeCurrentSaveStored` 未移植）。
- 含事件 61 的全軍移動（`event61DialogueActions` 未移植）。

HD 介面下的系統確認框與戰場資訊先畫原版索引畫格（HD 版屬 R6）；非繁中語系的戰場資訊目前仍是原版中文。

驗證：照 Go 移植 `TestNativePlayerCycleRawOrderSkipsEachGateAndWraps`、`TestSystemOptionsOverlayUsesExactCellsAndReturnsAfterFourClosePresents`、`TestPrepareNativeSystemInfoUIRunsTwelveOpenWaitTwelveClose`、`TestPrepareNativeSystemInfoUIFailsBeforePublicationWithoutRawCount`、`TestNestedSystemOverlayUsesRuntimeSaveAndFD2SAVGates`、`TestNativeConfirmationMouthMissingFrameFailsBeforeENDMutation`；新增 END 確認取消不結束回合、快照來回與阻擋、版本／節點／地圖不符時不改狀態。fd2-game 480 項與 Clippy 通過。Go 的第一關快照整合測試要先補 `fastForwardShotCampaign`，尚未移植。

### R6 HD 載入、記憶體與繪製尖峰（RUNTIME-E1，2026-11-27）

使用者要求 Rust 版優化圖片讀取與記憶體，載入或切換原版／HD 不要久等。先量再改，畫面結果不變：

- **HD 地圖世界圖重用**（`modern_hd_map_gpu.rs`）：原本每次 draw 都在 CPU 從零合成 1920×1200 世界圖，再包成新的 `Arc`，畫面靜止時外殼也每幀重傳 9 MB 貼圖。現在全部輸入（畫面計畫、人物圖與位置、橫幅）逐像素相同時沿用同一張；背景加地形層另外快取，只重疊人物、前景與介面。比對靠 `Picture` 的內容相等，不靠位址。
- **開場捲動與淡化**（`modern_opening.rs`）：`opening_viewport` 原本對 920 萬個位元組逐一做 f32 內插，改成以同一算式先算 256×256（淡化 3×256）對照表再查表，結果逐位元組相同。
- **開場預解碼**：解完一格就在背景執行緒解下一格，只多留一格。
- **開場不常駐壓縮檔**：載入時照 61 核對每檔 SHA-256 與 PNG 標頭，之後只留路徑；用到時重讀並再驗 SHA-256，檔案不見或被改仍是硬性錯誤。

量測工具：[`tools/rust_hd_load_measure.sh`](../../tools/rust_hd_load_measure.sh) 量載入器（release、每項獨立程序）；視窗流程用 `FD2_FRAME_TIMING=1 tools/rust_r6_theme_window.sh`，外殼每 60 幀輸出 update／draw／放置與 RSS，超過 50 ms 的幀另記一行；主題載入與切換時間在 `FD2_CUTSCENE_LOG` 的 `[theme]` 行。環境為 Docker `--cpus 4`、Xvfb 與 llvmpipe，主機另有負載，秒數只用來比較前後。

| 項目 | 改前 | 改後 |
|---|---|---|
| 載入器（release）秒數／記憶體高峰 | 2.1 秒／320 MiB | 1.4 秒／176 MiB |
| 第一次 F2 到套用（視窗） | 1.4～3.2 秒 | 1.2～2.1 秒 |
| 已載入後原版／HD 來回切換 | 0～1 ms | 0～2 ms |
| 超過 50 ms 的幀：首次 F2／冷啟動／序章來回切換 | 9／5／87 | 1／1／4 |
| 整個程式 HD 下 RSS 高峰（原版 650 MiB） | 1030～1072 MiB | 860～924 MiB |

剩下的尖峰都是一次性：HD 剛套用時開場第一格同步解碼（130～180 ms）、對白中切到 HD 的第一幀重建背景地形層（約 48 ms）、地圖第一次使用時整張地形解碼（START 進序章約 120 ms）。HD 地圖人物 125 組在載入時全部解碼，約 117 MiB，是載入器剩下最大的一塊；改成按需要先把載入時的幾何檢查延後，另案處理。

驗證：新增世界圖重用與全新合成逐位元組比對（含橫幅、範圍層、調色加亮與失敗後不沿用）、開場查表對照逐位元組算式、預解碼與跳格、載入後改檔報錯；`tools/rust_r6_opening.sh` 的 Go 差分 10 項（含開場呈現 173 格）通過；fd2-game 470 項、fd2-app 51 項與 Clippy 通過；視窗驗證 f2、cold、missing、toggle、interface-cold、no-font 全部 PASS。收據在 `work/rust-hd-measure-r1`、`r2` 與 `work/rust-hd-window-timing-r1`～`r4`。

### R6 工具列設定入口

使用者確認將美術、介面、音訊、語言、字級、速度、海報、全螢幕及按鍵設定移到工具列。保留 F5／F9 快速存讀檔與遊戲操作。

滑鼠移到遊戲畫面上緣即可顯示工具列。觸控同樣可從上緣開啟。工具列提供美術、標題介面、音源、語言、音樂開關、字級、速度、全螢幕、海報及按鍵面板。

Linux Bevy 發行組態建置及格式檢查通過。`work/r6-toolbar-20261122-r2` 用滑鼠驗證四種美術與介面組合，並比對按下 F2／F11 前後的標題畫面。`work/r6-toolbar-settings-20261122` 驗證偏好冷啟動、按鍵面板開關，以及音源、字級與速度保存。重跑入口為 [`tools/rust_r6_theme_window.sh`](../../tools/rust_r6_theme_window.sh)。

### R6 工具列語言與原生劇情對白

工具列依繁中、簡中、日文、英文循環。官方介面、對白及實體名稱清冊與法術顯示名稱先完成檢查，才套用並保存 `locale_id`。對白進行中會保留該句已編排的語系。非繁中對白依既有正式譯文、TTF 字寬及原生框位斷行與分頁，沿固定閉嘴頭像的節拍繪製。原生講者、控制碼、來源識別值及劇情動作仍取原始資料。

Linux Bevy（RUNTIME-E1）以 `tools/rust_r6_theme_window.sh` 的 `locales` 模式操作標題工具列。`work/r6-locale-window-20261122` 通過繁中至簡中、日文、英文切換，F4 保持英文設定，英文皇城首句可翻頁並續行。`locale-samples` 模式使用該輪保存的英文設定，驗證首個完成文字區冷啟動逐像素一致。日文及簡中從標題正常進入同一句並核對官方譯文。樣本用軟體 GPU、1280×800、原版美術、`FD2_SEED=4`。後三個重驗樣本使用遊戲四倍速設定，收據在 `work/r6-locale-samples-20261122-r2`。格式、Clippy、發行組態建置及 449 項 Rust 測試通過。

### R6 HD1 海報素材（DATA-READY）

沿 [61 的雙側海報規格](../knowledge-base/61-modern-theme-runtime-spec.md#hd1雙側海報正式接線conformed-scopedruntime-e1285)移植兩組圖稿的清冊、SHA-256、尺寸與不透明度檢查。整組驗過後才交付四張側欄，每張 296×900。內緣遮罩沿凍結 Go 版的分段漸層及預乘色彩規則。

`tools/rust_game_test.sh --release --quiet` 通過格式、Clippy 與 448 項回歸。唯讀掛載私人 HD 包後，在 Docker 以 `cargo test --offline --locked -p fd2-game --release --lib approved_posters_match_independent_crop_and_mask -- --ignored` 重驗正式素材。`FD2_REMAKE_ASSETS` 設為掛載的 `remake/assets`，`FD2_MODERN_THEME_PACK` 設為掛載的私人圖稿目錄。四張裁切圖及其 48 像素預設遮罩的 RGBA 雜湊均與獨立 Pillow 計算一致。

### R6 HD1 雙側海報

從上緣工具列的「海報」按鈕開啟選單。可關閉、切換兩組圖稿、調整內緣寬度及淡入時間，或重播淡入。偏好沿既有 `poster` 物件保存。Bevy 以兩張獨立精靈疊在乾淨地形或深藍底上，每側佔 132 邏輯像素。側欄點擊被排除，輸入以中央 640×400 為原點。HD 只從地形種子、當次色盤及地形圖層組合底圖，原版穩定地圖沿排程的地形階段取樣。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_theme_window.sh work/<新收據目錄> '' posters` 重跑 Linux Bevy 視窗。`work/r6-poster-window-20261122-r2` 檢查兩組切換、側欄點擊、漸層及淡入偏好保存、冷啟動、關閉與缺件回退。1808×800 視窗下的中央標題背景為零像素差異。格式、Clippy、Linux 發行組態建置及 448 項 Rust 回歸通過。

`work/r6-poster-scenery-20261122` 用 `poster-scenery` 測具模式，從標題 START 走到皇城首句對白。以側欄內緣取樣確認標題沿深藍底，皇城在原版與 HD 美術下都改用乾淨地形，對白和人物仍在中央。此為 Linux Bevy 的正常玩家路徑抽樣（RUNTIME-E1）。其他平台與戰役場景未納入本次抽樣。

### R6 原生劇情古金對白框

沿 [61 的古金色對白框規格](../knowledge-base/61-modern-theme-runtime-spec.md#hd1古金色對話框conformed-scopedruntime-e1normal-start)共用工具列的介面偏好。選擇 HD 前先驗證既有 FDOTHER 格網、色盤及 48×48 材質。開框、穩定頁、收框各按原生格網界線定位，游標尾段不畫框。Bevy 以獨立邊帶疊在對白底圖上、HD 頭像下，沿既有畫面淡出與遮擋條件。地圖與頭像繼續由美術設定控制。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_theme_window.sh work/<新收據目錄> '' gold-dialogue` 在 Linux Bevy 正常 START 到皇城。`work/r6-gold-dialogue-20261123` 驗證同一句的四種美術與介面組合。切換介面時文字區為零像素差異，兩種美術及英文翻頁後的對白上框像素相同。樣本為 1280×800、`FD2_SEED=4`、遊戲四倍速。獨立 1860×516 參考圖逐像素核對通過，450 項 Rust 回歸通過。本輪視窗驗證限皇城穩定頁及翻頁（RUNTIME-E1）。

### R6 標題讀檔槽與 HD 細框

標題 LOAD 接回四槽選取、確認和取消。原生中繼資料通過檢查時沿用原生索引畫格，章號 0 的 JSON 存檔走原有後備框。兩種畫面各用實際列位置接收滑鼠，鍵盤上下鍵在首尾槽停止。工具列 HD 介面以各自的尺寸及來源色盤疊上細框。JSON 保留半透明底，原生槽仍沿原字模。確認沿既有存檔驗證與還原交易，遇到尚未支援的戰場中途快照會留在選槽畫面。

從首章章界存檔還原時，沿用凍結 Go 版的劇本起始對白、三列分頁與頭像。這條路徑的對白也使用同一古金框材質，並跟隨說話者切換時的收展動畫。

使用 `FD2_RUST_TARGET=target-r5i bash tools/rust_r6_theme_window.sh work/<新收據目錄> <章界存檔目錄> loadslots`，存檔目錄以唯讀方式掛載。樣本為首章 `battle_ch01` JSON 章界存檔，SHA-256 為 `089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2`。測試以 1280×800、`FD2_SEED=4` 和遊戲四倍速執行。這次的原生版面樣本使用空槽。

`work/r6-loadslots-20261123-r9/json` 核對細框、空槽拒收、取消與首章讀檔。從工具列切換首章對白框時，框線像素改變，框內頭像與文字的比較區為零像素差異。視窗確認可以繼續到戰場，讀檔前後的存檔複本雜湊一致。此驗證屬 RUNTIME-E1。

原版 `FD2.SAV` 另用[第 2 章來源清冊](../data/parity-slots/ch02-manifest.json)中的酒店實存檔。標題 LOAD 拒收第二槽，取消後重開再選第一槽，成功還原羅德鎮。用同一份存檔的 dosgolem `town_loaded` 畫格比對，右側營帳區逐像素相同。工具列下方其餘畫面有 5,760 個差異像素，集中在索爾圖格。存檔複本的 SHA-256 維持 `540c3bc156ba94b2f0dd06ca83f6a0d8c6c6dd48c49621c73add3f839bfc118b`。這份 Rust 收據只涵蓋載入城鎮。

```bash
FD2_RUST_TARGET=target-r5i bash tools/rust_r6_theme_window.sh \
  'work/<新收據目錄>' work/ch02-flow-20261017/input native-save \
  work/ch02-flow-20261017/original-startup-r1/checkpoint-0018.png
```

### R6 F11 標題介面（RUNTIME-E1，2026-11-22）

沿凍結 Go `modern_hd_title_interface.go` 的 FDOTHER #7 六個原生選單格、原生色盤與字型配置。預先驗證來源與字型，保留兩側選取標記、定位和閃爍，於選單中間以三倍畫布排繪 START、LOAD、CONTINUE。初版用 F11 保存 `interface_mode`，F2 保有獨立美術偏好。現行入口為工具列。讀檔槽見上一節，戰場介面另行接線。用正常 Bevy 視窗核對四種組合、缺字型、設定保存與冷啟動。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_theme_window.sh work/r6-f11-title-20261122 work/r6-theme-toggle-20261122-r3` 通過 Linux Bevy 視窗測試。四種美術與介面組合通過，F11 切換前後的上方背景逐像素相同，兩種偏好冷啟動重現相同畫面。缺字型時原版標題像素相同且未寫設定。普通 START 與序章首句的 F2 切換仍通過。此輪使用 Noto Sans CJK 基底字型，區域字型家族仍待移植。

### R6 HD1 美術啟用與背景載入（RUNTIME-E1，2026-11-22）

依 Go `697c0913` 的 `player_theme_loader.go`、`player_shortcuts.go` 與 61 的 HD 主題包契約。第一個 F2 和 HD 美術設定啟動共用背景載入。繪製及輸入照常，成功後由遊戲更新套用，失敗留在原版並回報原因。切回原版時保留已驗證圖庫，重切不讀檔。F2 只寫美術設定，保留既有 F11 介面偏好。

此切片啟用已通過差分的開場、故事頭像、劇情地圖與戰場圖層。沿用各圖庫的清冊身份、雜湊、尺寸與按需解碼。在對白進行中只替換畫面引用與遮罩，保留頁次與節拍。物理攻防沿原生畫面。完整 HD 清冊、F11 與工具列依 #312 追蹤。

`FD2_RUST_TARGET=target-r5i bash tools/rust_game_test.sh --quiet` 通過格式、Clippy 與 446 項 Rust 回歸。`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-theme-golden-20261122` 通過十項 Go 差分，包含開場、頭像、故事對白、計時對白及地圖圖層。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_theme_window.sh work/r6-theme-window-20261122-r6` 通過 Linux Bevy 視窗驗證。首次 F2 後開場持續更新，美術偏好落盤並保留原介面偏好。冷啟動以保存的設定進入相同 HD 標題畫面，兩張 X11 擷取逐像素相同，PNG 的 SHA-256 為 `99edd3b6e1f2b359fe3a8cb305baf1a2816f95f3c71a5b039ecf109600396d57`。缺包時拒收並抵達原版標題，設定檔沒有寫入。視窗測試使用正常 Escape 略過開場。標題上的再次 F2 曾等候逾時，普通 START 後的對白切換仍待視窗驗證。

標題重驗在 Bevy 視窗等到選單就緒。切換 START／LOAD 時，選單上方 1280×656 像素完全相同。F2 切回原版及再切回 HD，各自逐像素重現原版及 HD 標題基準。此前逾時保留在舊收據。Bevy 只在 HD 分層模式及介面像素連續相同時，沿用上一張介面貼圖。進出 HD 都會重新上傳介面，套用對應的透明度。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_theme_window.sh work/r6-theme-toggle-20261122-r3 work/r6-theme-toggle-20261122` 通過標題及序章重驗。普通 START 進入皇城首句對白，F2 來回切換地圖、人物及頭像，原版對白框與同一句台詞持續顯示。首次 F2、冷啟動及缺包驗證在同一執行檔的 `work/r6-theme-toggle-20261122` 通過。F11 與完整 HD 清冊仍依 #312 追蹤。

### R6 HD1 劇情走行與鏡頭滑動（RUNTIME-E1，2026-11-22）

沿 Go `697c0913` 的 `modern_hd_story_motion.go` 及規格 61。使用現行劇情控制器的座標、偏移、面向和節拍。HD 鏡頭投影到私有地格視圖，保留小數像素位移及原生角色落腳點。地形補足右側一欄與下側一列，再裁切到原生視窗。視差採原生每格水平 3 像素與垂直 2 像素的比例。地形與前景的原生索引色映射各照來源遮罩執行。

走行人物由原生 selector 確認圖源，四方向各用三拍。遮擋使用私有視覺座標。取樣只改候選幀，錯誤格網、錯誤鏡頭及缺圖在發布前拒收。輸入、完成回呼、原生索引位置及保存沿既有控制器。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-story-motion-20261122` 通過。皇城 map32 的四向走行、兩次轉向、雙軸移鏡及跟隨捲動合計八張 1920×1200 畫布，皆經 Rust `Game::draw` 與 Go 世界合成比對，單一色彩通道最大差值為 1。底圖像素雜湊、圖格及矩形完全相同。鏡頭兩組取樣使用原版 FDOTHER #16／#17 視差資料。圖層計畫和畫布比較紀錄的 SHA-256 分別為 `a4ed743d0d5753c0a007ef80165cc89bd5ccc005123b6bf17c133e18565662a8` 與 `f56276f3aee134ed945fa5276b6f57ef06091774d84d8a76ae4ee01f8477f067`。

測試確認繪製保留原生視圖與時鐘，走行完成回呼只執行一次，無效鏡頭或缺件保留上一份合成計畫。格式、Clippy、446 項 Rust 回歸與十項 Go 差分通過。此切片使用受控位置與原版介面。完整主題啟用、F2／F11 與一般遊玩仍由 #312 追蹤。[61 的走行和移鏡契約](../knowledge-base/61-modern-theme-runtime-spec.md#故事走位與插值移鏡hdconformed-scopedruntime-e1)

### R6 HD1 固定鏡頭的劇情地圖（RUNTIME-E1，2026-11-22）

依 Go `697c0913` 的 `hasModernHDStoryMapSource`、`prepareModernHDStoryMapFrame` 及規格 61 的故事載體契約。限定具有原生 `LOADCH` 載體、24×24 地格、原生視圖且鏡頭對齊的劇情場景。畫面使用同一場景地形、色盤、人物及前景，關閉戰鬥 HUD 與範圍。原生對白仍由獨立介面繪製。

候選載體讀目前已發布的劇情角色，沿用既有原生前景近似與時鐘規則。預備只操作複本，成功才交付 HD 繪圖計畫。角色名冊、索引工作區、VGA、DAC、相機與時鐘皆保留原控制器的發布邊界。缺來源或圖格時不發布畫面。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-story-map-20261122-r2` 通過。皇城 map32 採正式地形、人物、清冊及場景資料，在兩個固定鏡頭走 `Game::draw`。1920×1200 世界與 Go 的獨立 CPU 合成相比，單一通道最大差值為 1。兩側介面皆透明。重複繪製保留載體、名冊及時鐘，拿掉一張必要人物圖格時保留上一份合成結果。比較紀錄 SHA-256 為 `061f2c377f804421fca6574b4035bbfca8dddec7f49282236ae51f6c4a1081ee`。格式、Clippy、446 項 Rust 回歸與十項差分通過。

`FD2_RUST_TARGET=target-r5i FD2_GOCACHE="$PWD/work/gocache-r5i" FD2_R5P_MODES=opening bash tools/rust_r5p_layers.sh work/r6-story-map-original-20261122` 通過。原版美術從開場、START、序章到第一關的 10,372 個圖層擷取點與 Go 相同，含 8,408 張世界圖層及 72 張淡幕。序列 SHA-256 為 `40f0309d295e993f9c9a8c9fbafe4c95220c6883538e0f0ba486fa5f7a155ef0`。

完整 HD 主題載入與一般啟動仍由 #312 追蹤。[61 的劇情載體契約](../knowledge-base/61-modern-theme-runtime-spec.md#hd1首關出場名冊同步修正conformed-scopedruntime-e1)

### R6 HD1 開場視窗合成（RUNTIME-E1，2026-11-22）

沿 Go `697c0913` 的 `modernOpeningSource`、`drawModernHDOpening`、`drawTitle` 及 61 的開場與按需解碼契約。由原有排程決定 FDOTHER／ANI 資源、畫格、淡色、長卷裁切與調色閃動。ANI #0 沿用共用圖庫，紅閃仍由邏輯畫布持有。開始選單以 FDOTHER #7 的 HD 畫布為底，原介面選單項獨立呈現。[61 的開場契約](../knowledge-base/61-modern-theme-runtime-spec.md#hd開場素材與呈現契約) [61 的按需解碼契約](../knowledge-base/61-modern-theme-runtime-spec.md#hd主題包按需解碼conformed-scopedgo-版2026-11-21)

目前畫格以完整 1920×1200 交給 Bevy 線性取樣，只有長卷或色彩需要處理時才建立新畫布。相同畫格可重用貼圖。畫格若因延後檢查而拒收，下一畫格回到同幕原版圖。說明、啟動錯誤及非開場狀態依現有繪製分流。維持原有輸入、音效、開場時鐘與選單轉移。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-opening-view-20261122` 通過。以受控 HD 狀態走 Rust `Game::draw`，比較 Go 的 12 個 1920×1200 最終畫格。八格完全相同，四格的單一色彩通道最大差為 1。開始選單逐像素相同。比較紀錄 `rust-opening-view.txt` 的 SHA-256 為 `683f9c0068362ac93135973cd8e795800dbb944b7c49cb5679f59a32348c7e3c`。格式、Clippy、446 項 Rust 回歸與十項差分通過。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r5_app_frames.sh work/r6-opening-view-app-20261122 tools/rust_r5_app_scripts/letterbox-1500x850.json` 通過。Bevy 發行版原版美術回歸的 433 格與無畫面端相同，五張視窗截圖逐像素相同。完整主題載入、F2／F11 及正常 HD 啟動仍由 #312 追蹤。

### R6 HD1 戰場計時對白（RUNTIME-E1，2026-11-22）

沿 [61 的計時戰場對話規格](../knowledge-base/61-modern-theme-runtime-spec.md#2026-10-19-計時戰場對話透明圖層conformed-scoped限runtime-e1及明示槽普通輸入)及其 `timed_dialogue_followup_20261019` 收據。移植基準為 Go `697c0913` 的 `freezeModernHDTimedDialogue` 與 `drawModernHDMapFrame`。FDTXT_000 的經驗、升級及中毒訊息已由原生計時對白處理器供應。

戰場只有在唯一一則原生 `FFEC` 下框、已知零位移、五格收框且沒有走行或回合橫幅時才凍結 HD 背景。首次合成從當幀取世界與介面繪圖計畫，由對白處理器保有，接續畫格仍提交當次的原生戰場時鐘與工作區。五張開框、一張穩定格網及五張收框全部通過原生格網覆蓋預檢才發布。對白只疊覆蓋區內的索引像素。缺錯格網時回報錯誤，處理器離場時釋放快照。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-timed-dialogue-20261122` 通過。以正式 FDTXT_000 及 FDOTHER #5 比對五張開框、三頁升級訊息與五張收框。13 張 RGBA 畫格與 Go 逐位元組相同，序列 SHA-256 為 `89e33c5ff45025dce851f1c79aa4d64b931250f3f180356e6e6090a53af73b0f`。Rust 戰場繪製入口證實原生時鐘及候選背景更新時仍沿用首次 HD 畫面，收框後恢復即時戰場。錯誤格網不發布快照。格式、Clippy、445 項 Rust 回歸與九項差分通過。

本輪驗證使用原版介面與受控的升級狀態。完整 HD 主題、F2／F11、普通輸入與原版 DOS 對拍依各自閘門。

### R6 HD1 對白遮罩與頭像視窗呈現（RUNTIME-E1，2026-11-22）

依 Go `697c0913` 的 `freezeModernHDStoryDialogueMasks` 與 `DrawFinalScreen`。原生對白合成器在索引 0 及 255 的底圖各重跑一次。同座標輸出相同即保留該處覆寫，可辨認索引 0 的真實寫入。開框與收框逐格保存遮罩。穩定頁包含全部逐字畫格的寫入聯集。遮罩依階段、頁碼及畫格選用。分頁捲動或缺少遮罩時沿用完整原生幀。

Bevy 將已驗證的 480×480 頭像以線性取樣疊在邏輯介面之上，沿原生 160×160 顯示位置及視窗縮放。全黑／全白、說明或模態狀態依 Go 抑制頭像。畫面淡出以相同的亮度縮放頭像。驗收比較 Go 的遮罩像素與最終座標，再核對 Rust 的對白畫布與視窗回歸。本切片的對白遮罩驗收範圍為一般繁中原生對白。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-dialogue-masks-20261122-r2` 通過。以正式對白綁定的四種控制碼及兩個收框位移（0、37），核對 138 張開框、穩定頁與收框的 RGBA 遮罩。Go 與 Rust 逐張相同，序列 SHA-256 為 `faeffaa87a7bc375195894532290d6d6e6fe9e5173b9f10f820266d29b088922`。兩側使用 HD 美術與原版介面。Rust 另核對索引 0 的不透明寫入、背景保留及分頁捲動的完整原生畫格。格式、Clippy、444 項 Rust 回歸與八項差分通過。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r5_app_frames.sh work/r6-dialogue-app-20261122 tools/rust_r5_app_scripts/letterbox-1500x850.json` 通過。原版美術的 433 個畫格與無畫面端相同，五張 1500×850 視窗截圖逐像素相同。480×480 圖源、原生頭像矩形、淡出及遮蔽另由執行期測試核對。HD 主題啟用後的視窗截圖、其他語系與 F2／F11 仍待驗收。

### R6 HD1 故事頭像清冊與對白接線（RUNTIME-E1，2026-11-22）

依 Go `697c0913` 的 `loadModernStoryPortraitSetWithProgress`、`drawNativeStoryDialogue` 和 `presentNativeStoryDialogueFrame`。候選主題核對 80×80 與 480×480 頭像的講者 ID、靜態閉嘴畫格、清冊契約、PNG 幾何、SHA-256 與完全不透明。480×480 選配集合若存在，講者須與 80×80 集合一一配對，整批驗過才交付。

穩定對白依原生 control 取得頭像矩形。在原生索引背景上疊 80×80 頭像，並交付同位置的 480×480 最終呈現描述。開框、收框與對話進度沿既有原生排程。驗收逐格比對 Go 的正式圖包，並抽樣上下框的疊圖與透明度拒收。HD 世界遮罩、480×480 最終呈現與主題啟用依各自 R6 接線。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-story-portrait-20261122-r2` 通過。正式圖包 44 對頭像在兩種解析度的 RGBA 雜湊逐格相同，序列 SHA-256 為 `c9001c9ade4ebbe1e28a0b3fa9de71399670879e297611a5cdf9771b330fda49`。Go 測具同時讀取旅館與教會場景，保留正式場景與頭像的配對驗證。Rust 以受控對白驗上框、下框、開框與收框的像素，並驗清冊缺件、重複、雜湊、尺寸及透明度拒收。格式、Clippy、442 項 Rust 回歸及七項差分通過。

### R6 HD1 戰場圖源接線（RUNTIME-E1，2026-11-22）

依凍結 Go `697c0913`、[61 的 FDICON 契約](../knowledge-base/61-modern-theme-runtime-spec.md#fdicon-地圖人物-12-格契約)與既有地形按需解碼。正式人物清冊須為每個有原版來源覆寫的圖組提供 24×24 與 144×144 圖格。核對角色清冊的大小、透明度、SHA-256、三拍／十二格規則、重複關係、置中與落腳點。115% 的角色沿原落腳點放大，平手取偶數。使用原生 FDICON 選格的空白畫格規則。

候選主題只能整批收入已核對的人物及地形。在戰場繪製交易中，依地圖索引與當前 FDSHAP 圖庫取地形。首次取用解碼並核對原版遮罩，沿用最近兩張地圖快取。解碼或透明度不合時該地圖回退原生底圖，檔案被更換、缺失或原版遮罩不符則拒收。驗收以正式素材比對 Go 取圖與 115% 矩形，再由 Rust 戰場入口抽測兩圖的地形及人物。整套主題啟用與切換沿既有 R6 閘門。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-map-art-20261122-r2` 通過。正式圖包共 125 組、1,455 對人物圖格，24×24 與 144×144 的像素及四組 115% 的落腳矩形與 Go 逐筆相同。序列 SHA-256 為 `550337549ab5b36b83b5f663f55e59af76f36f492bf56f5738cb286946ffd3b9`。原始地圖索引 0 與 9 的受控 `draw_field` 各取得 104 筆 HD 地形指令和一筆 HD 人物指令，索引 9 使用原版 FDOTHER #15 輔助底圖。格式、Clippy、440 項 Rust 回歸與六項差分通過。

### R6 HD1 戰場圖層合成（RUNTIME-E1，2026-11-22）

依凍結 Go `697c0913` 的 `prepareModernHDMapFrameStageLayout` 和 `drawPreparedModernHDMapFrame`。戰場使用同一批候選時鐘、色盤、地格及單位。底圖讀獨立的地形工作區，原生整幀和 HD 圖層一同通過後發布。普通步進、北移和靜止沿各自的原生排程。範圍與介面使用兩種底色辨認覆寫，保留零索引與同色寫入，拒收目的地相依的寫入。

完整解析度畫布依序畫底圖、HD 地形、範圍、人物、前景與場景覆蓋。普通介面使用獨立的 640×400 畫布。指定在介面上方的覆蓋則在世界畫布合成 HUD 後再畫。所有物件依照原視窗裁切。驗收以 Go 正式繪圖讀回 RGBA 比對兩種覆蓋順序，並確認正常戰場合成的原生狀態與失敗交易。主題載入、畫質衍生圖與各劇情演出依各自的 R6 入口驗收。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-map-layers-20261122` 通過。以固定圖層、非整格裁切、半透明前景及 HUD 圖片跑 Go 正式繪製。兩種覆蓋順序的 1920×1200 世界及 640×400 介面全畫布 RGBA 逐位元組相同，雜湊序列 SHA-256 為 `bc09a12b106d95872dca48787509c2b57540a627553869cd0188a05d12ba654e`。以固定戰場素材走 Rust `draw_field`，核對 DAC、時鐘、循環、工作區與 VGA，並驗缺件時保留前一畫格及原生狀態。格式、Clippy、438 項 Rust 回歸與五項差分通過。

Bevy 以線性取樣顯示世界，再疊 640×400 介面。目前 HD 地圖合成採完整解析度 CPU 路徑，固定尺寸的原版 GPU 合成照原設定運作。此收據涵蓋受控畫格及戰場入口。主題載入、F2／F11、兩種衍生畫質、獨立 HD HUD 圖源與劇情演出另依 R6 驗收。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r5_app_frames.sh work/r6-map-layers-app-20261122 tools/rust_r5_app_scripts/letterbox-1500x850.json` 通過發行建置以及原版美術視窗回歸。無畫面驅動端與 Bevy 的 433 幀雜湊相同，五張視窗截圖逐像素相同。

### R6 HD1 地圖人物與前景繪圖計畫（Go 對照通過，2026-11-22）

依 Go `697c0913` 的 `buildModernHDMapUnitRegionPlan`、`buildModernHDMapForegroundRegionPlan` 與 `prepareModernHDMapFrameStageLayout`。人物沿用 fdicon 的槽位、動作畫格、位移與原生寫入遮罩，空畫格不產生命令。前景沿用原生重繪資格、相鄰格、原生選格與透明遮罩。遮罩中的同色寫入仍遮擋下層，目的地重映射區維持透明。原生前景與 HD 前景共用工作位址、六倍矩形與色盤路徑。

地圖繪圖計畫依繪製階段分別讀取範圍模式改寫前的地形和改寫後的前景地格。前景只取與用途相符的 HD 畫格，缺件沿用已解碼的 24×24 原生前景。人物 HD 畫格缺件或尺寸錯誤、前景來源不完整、色盤或位址異常均拒收整份計畫。驗收對照 Go 的人物與前景計畫，另與 Rust 原生寫入逐像素比對。正式視窗仍須接上主題交易與圖層合成。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-map-geometry-20261122` 通過。固定輸入涵蓋 1,568 組人物與 96 組前景，包括動作、鏡頭位移、色帶、前景資格、查色表和工作區邊界。Go 與 Rust 的指令序列與前景像素雜湊逐筆相同，序列 SHA-256 為 `1b96cbc30c193340809a2a36fbbd2f6b428465f8c6c9addec1c470f7900e00fb`。Rust 原生工作區的寫入對照、改寫前後地格及 HD 用途後備通過。格式、Clippy、436 項 Rust 回歸及前批正式素材差分也通過。

### R6 HD1 地形調色與座標投影（Go 對照通過，2026-11-22）

依凍結 Go 版 `697c0913` 的 `terrainColourContext`、`modernTerrainHDAsset.shade`、`buildModernHDTerrainRegionPlan` 與 `prepareModernHDMapFrameStageLayout`，沿用已核對的 FDSHAP 畫格、遮罩、索引色、地形選格器及原生工作位址。色盤與查色表須各有 256 項。`BlitMode == 0xff` 使用目前色盤，其餘經查色表取色。以四個最近原版色的加權 RGB 差值調整 HD 像素，保留透明像素、同色免重算及全黑／全白收斂。RGB 插值屬現代美術規則。每格保留三種調色結果，原始色盤變更時重建權重。

繪圖計畫依原生座標輸出可見的地形畫格、原工作位址、HD 矩形與像素；缺少替換畫格時留給原生底圖。畫格選取沿用原生翻格與循環參數。視窗使用原生範圍與六倍縮放；向北移動沿用 `native_up_step_layout`，裁切只剔除完全離開視窗的畫格。無效區域、缺少控制記錄及寫入位址越界回報錯誤。驗收以正式 HD1 圖包對照 Go，抽測靜止與北移六拍、原始／重映射色盤及裁切。完整地圖合成與主題發布由後續執行期接線處理。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-terrain-plan-20261122` 通過。三張地圖使用正式 HD1 畫格及固定測試色盤與地格排列。靜止、北移六拍及窄視窗合計 2,385 筆繪圖指令，Go／Rust 的畫格、工作位址、矩形及 RGBA 像素雜湊逐筆相同，序列 SHA-256 為 `2eb94c4c3b4d9889177aa01c91141291fc5bd1dfbd0b2b75f15df5d02dc582d7`。33 張地圖共 3,983 格的取圖差分、開場與共用圖庫複驗通過。格式、Clippy 與 433 項 Rust 回歸通過。另驗色盤長度、透明度、全黑／全白、快取淘汰、地形選格及工作區越界。

### R6 HD1 地形清冊與按需解碼（DATA-READY，2026-11-22）

依 [61 的按需解碼契約](../knowledge-base/61-modern-theme-runtime-spec.md) 與 Go `697c0913` 的 `modern_hd_terrain.go`。清冊 SHA-256 為 `ea41e4b9a9d656f707a4f05c7eb64ddf36f0bbbeaee55ea1ff085b2f067988f1`，其中 492 筆地形登錄涵蓋 33 張地圖、3,983 個畫格。每格保留地圖 ID、原生畫格索引及地形或前景用途。

載入清冊時核對身分、索引、重複登錄、逐檔 SHA-256 和 144×144 PNG 標頭。首次取用以最多三條工作流解碼，同 SHA-256 共用像素，驗過原版 FDSHAP 遮罩與索引色後才交付整張地圖。每個來源路徑在首次使用及快取淘汰後都重驗。解碼或透明度異常讓該地圖退回原版，檔案變動和原版遮罩不符回報硬性錯誤。最近兩張地圖保留解碼結果。驗收比對全部 Go 畫格與原版遮罩，另驗地圖整批拒收、共用與淘汰。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-terrain-20261122` 通過。33 張地圖的 3,983 格畫面、用途與原版索引色集合逐格相同，序列 SHA-256 為 `fbbffed2e8cafd31ca8c9ff65f3a38e285e308a134dc91acd50534c393a8e974`。原版資料由 `load_separated_fdshap_bank` 核對 `FDSHAP.DAT` 識別與來源雜湊，再驗各格透明像素及分離遮罩。開場 173 格與天空之鑰 196 格重跑通過。格式、Clippy 和 431 項 Rust 回歸通過。

同內容但不同檔名在首次解碼及重載時各自驗 SHA-256。Go 的冷載入只重驗各 SHA 群組的代表檔，兩版正式素材輸出相同。快取淘汰會釋出地形圖庫持有的像素，原版圖庫仍為同一份時重用已驗索引色集合。

### R6 天空之鑰共用圖庫（DATA-READY，2026-11-21）

依 [61 的按需解碼契約](../knowledge-base/61-modern-theme-runtime-spec.md)及 Go 提交 `697c0913` 的 `modern_hd_sky_key_assets.go`。同一份正式清冊登錄 ANI #0 的 96 格和 FDOTHER #34 的 100 格。分離素材的原始中繼資料 SHA-256 分別是 `7822c022…6313`、`49b732cb…e25c`。FDOTHER 原版第 0 格只核對來源，HD 從第 1 格開始。

入場驗證來源、製圖計畫、幾何和每格 PNG 雜湊。取圖時再讀同一路徑並核對雜湊，每庫保留最近使用的三張，以內容雜湊共用已解碼的唯讀畫格。ANI #0 的全不透明與黑幕條件、FDOTHER #34 的透明與不透明像素覆蓋均逐格驗證。像素檢查不符時退回該庫的原版畫格，檔案缺失或改動屬硬性錯誤。原版驗證使用分離素材的索引畫格、遮罩與 VGA 色盤。完成條件是兩庫逐格比對 Go 取圖、核對原版來源與測通錯誤邊界。這次接到開場資料入口，畫面與 F2 依完整主題交易接線。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-sky-key-20261121` 重跑開場 173 格與共用圖庫 196 格。兩份序列分別與 Go 一致，共用圖庫序列 SHA-256 為 `983e940824caf698cd8ef0fc9389982a80cac298651fb6c6f7157c29cbceb729`。原版核對涵蓋 FDOTHER #34 的 101 格和 ANI #0 的 96 份索引畫格及色盤。負向測試覆蓋快取命中時缺檔、超出索引、快取淘汰、同一 PNG 對應不同黑幕條件與原生資料改動。

開場取圖直接使用天空之鑰的 ANI #0 圖庫。Rust 以共享的唯讀 `Arc<Picture>` 保護快取像素，持有者修改時會複製。重新解碼時仍核對先前的像素雜湊。第二十章繪製和 HD 世界預檢待接入播放器。

### R6 開場 HD 清冊與逐格解碼（DATA-READY，2026-11-21）

依 [61 的按需解碼契約](../knowledge-base/61-modern-theme-runtime-spec.md) 和凍結 Go 提交 `697c0913` 的 `modern_hd_opening.go`。清冊 SHA-256 為 `ea41e4b9a9d656f707a4f05c7eb64ddf36f0bbbeaee55ea1ff085b2f067988f1`。本切片涵蓋 `opening_presentation_hd_set` 的 12 組、173 格。共用的 ANI #0 另有圖庫與黑幕契約，整套主題要等該圖庫及其餘角色驗證後才可發布。

載入端核對清冊身分、圖組來源、形狀、檔名、來源雜湊、製圖計畫與 PNG 標頭。每條登錄路徑都讀檔並驗 SHA-256。組內相同 SHA 共用壓縮位元組。取圖時才解碼，檢查尺寸和全不透明，保留目前一格。延後檢查失敗時整組退回原版並通知一次。驗收以 Go 正式取圖所解出的像素比對整套清冊，另驗缺件、錯誤雜湊、路徑、重複圖組、截斷圖及非全不透明的拒收邊界。此載入層不改存檔或 DOS 原版行為。

`FD2_RUST_TARGET=target-r5i bash tools/rust_r6_opening.sh work/r6-opening-20261121` 通過。Go 與 Rust 的 173 格像素雜湊逐格相同，序列 SHA-256 為 `02baa600964a9fcf88db55b6d5c169593979d094a8d539a0fdce7a9e224c1f2a`。Rust 的三項邊界測試與 Go 的八項清冊反例通過。16 位元 PNG 以完整 alpha 判斷不透明。`tools/rust_game_test.sh` 的格式、Clippy 與 427 項測試通過。收據只保存雜湊及測試紀錄，私人素材以唯讀掛載。

Rust 開場載入與取圖已可由測具驗證。共用 ANI #0 已接到取圖端。完整主題交易及開場繪製接線後，再驗證 F2 和視窗畫面。

### 2026-11-13 R0 起步、R1 第一個切片：`fdsave`

| 項目 | 結果 |
|---|---|
| 映像 | `fd2-rust:1.99.1-r1`，由 [fd2-rust.Dockerfile](../../tools/docker/fd2-rust.Dockerfile) 建置，rustc 1.99.1（58cf8f9c3 2026-10-13） |
| workspace | [`remake-rs/`](../../remake-rs/Cargo.toml)，edition 2024，`unsafe_code = "forbid"`，rustfmt 120 欄 |
| 移植 | [`fd2-formats::save`](../../remake-rs/crates/fd2-formats/src/save.rs) 與 [`keyboard`](../../remake-rs/crates/fd2-formats/src/keyboard.rs)，錯誤訊息與 Go 版逐字相同 |
| 第 1 層 | Go 的 19 項測試全部移植，另加驗證橋竄改與錯誤文字 2 項，共 21 項通過；Go 版 nil 指標拒寫一項由 Rust 型別排除 |
| 第 2 層 | [Go 黃金匯出工具](../../remake/cmd/fd2-golden-fdsave/main.go)以 PCG 種子 20261113 產生 192 個隨機與結構化案例，另加 12 份真實 FD2.SAV，共 204 案例逐欄相符 |
| 黃金檔 | `work/rust-build/golden/fdsave-v1.json`，SHA-256 `c12a72e270685de3bd579b1ff4ca288722ad58eed9623f3e0a1829f1c7436070`，同輸入重跑兩次相同；真實存檔清單在同目錄 `fdsave-saves.txt` |
| 反向檢查 | 把一個案例的 selector slot 改掉，差分測試精確報出 `case 7 result.continue.selector[0][1]` 並失敗 |
| 覆蓋 | 161 案例解碼成功、43 案例校驗失敗、6 案例大小錯誤；28 案例 CONTINUE 通過且驗證橋成立，其餘涵蓋各類拒收訊息。HUD 錨點 `0xf2` 分支沒有隨機案例命中，由第 1 層測試覆蓋 |
| 第 3 層 | 本切片沒有新增原版對拍；fdsave 的原版證據沿用 Go 版既有收據 |

重跑：`tools/rust_golden_fdsave.sh $(cat work/rust-build/golden/fdsave-saves.txt)`。clippy 以 `-D warnings` 無警告。

下一步：R0 剩下的 Bevy 最小視窗、Xvfb 截圖、CJK TTC 字型與 OGG 播放可行性，以及 R1 其餘六個格式套件。

### 2026-11-13 R0 完成：Bevy 可行性

收據：[rust-r0-probe-20261113.json](../data/rust-r0-probe-20261113.json)，探針 [r0_probe.rs](../../remake-rs/crates/fd2-app/src/bin/r0_probe.rs)，重跑 `tools/rust_r0_probe.sh <輸出目錄> <TTC> <OGG>...`。

| 項目 | 結果 |
|---|---|
| 版本 | Bevy 0.20.1、wgpu 30.0.2；映像 `fd2-rust:1.99.1-r3`（加入 wasm 目標、ALSA null 裝置、`libxkbcommon-x11-0`） |
| 視窗與截圖 | Xvfb＋Mesa llvmpipe（Vulkan）開啟 1280×800 視窗並截圖；同輸入兩次截圖 SHA 相同 |
| 整數倍最近鄰放大 | 中文字框外 934999 個像素全部等於 640×400 測試圖 2 倍放大的預期值 |
| 中文 TTC | Bevy 0.20 改用 parley／fontique，TTC 字型集合可直接載入；以家族名稱與直接載入兩種方式都排出正確字形，不需拆成 TTF |
| OGG | FM 音樂與原版音效完整解碼，聲道與取樣率與 ffprobe 相符；兩個播放器都建立了 AudioSink（容器內是 null 裝置，沒有人耳聽辨） |
| wasm | `wasm32-unknown-unknown` 可編譯（120 MB，未處理） |

待查：OGG 解碼長度與 ffprobe 有小差異（音樂少 0.088 秒、音效多 0.003 秒），R5 要和 Go 版解碼器逐樣本比對；wasm 的 bindgen、瘦身與瀏覽器執行留到 R7；軟體繪圖只證明可行，不代表 Windows 實機幀率或閃爍行為。

### 2026-11-14 R1 進行中：格式模組移植

R1 改由子代理平行移植，每個子代理在獨立 git worktree 工作；主代理審查差異、在主 checkout 重跑黃金差分確認 SHA 相同後，以 squash 合併。

| 模組 | Go 測試移植 | 黃金差分 | 黃金檔 SHA-256 |
|---|---|---|---|
| `archive`、`native_rng`（自 `fdother` 拆出的共用層） | 5／5 | 以原版 FDOTHER.DAT 實跑 | — |
| `dato` | 13／13 | 1285 案例＋DATO.DAT 全部 137 筆＋544 張分離頭像 | `cd0c5d11…` |
| `afm` | 6／6 | 1158 案例＋ANI.DAT 289 幀＋587 個分離動畫檔 | `346f368c…` |
| `fdicon` | 31／31 | 3336 案例＋FDICON.B24＋5041 個單位圖＋24801 個 FDSHAP 檔 | `72003b8f…` |
| `fdtxt` | 13／13 | FDTXT.DAT 全部 35 筆、FDOTHER #4 全部 1824 字形＋分離字表與字型 | `46b35584…` |

分離素材的 JSON、PNG 與路徑一律用已鎖定的 serde、serde_json、png 與 `std::path`，不逐行移植 Go 標準函式庫。原版資料、真實分離素材與格式正確的生成輸入逐欄比對；損壞的 JSON／PNG 只比接受或拒收，差異依種類列在各黃金測試的 `KNOWN_DIFFERENCES`，標為刻意不照做。png crate 不驗 zlib 的 Adler-32，壓縮資料損壞時靠素材包雜湊把關。

未完成：`fdother`（分三份：核心與分離載入、原版流程邏輯、依賴核心的其餘檔案）與 `figani`。讀原版檔的環境變數尚未統一（`FD2_FLAME2_DIR`、`FD2_FDICON_REAL_FLAME`、`FD2_FLAME` 並存），R1 收尾時統一。

### 2026-11-14 R1 完成：原版格式層

`fd2-formats` 已涵蓋 Go 格式層的七個套件（`fdsave`、`fdicon`、`afm`、`fdtxt`、`fdother`、`dato`、`figani`），另從 `fdother` 拆出共用的 `archive` 與 `native_rng`。正式程式約 1.6 萬行、單元測試約 8200 行、黃金差分測試約 7000 行。

| 模組 | Go 測試移植 | 黃金案例 | 黃金檔 SHA-256 |
|---|---|---|---|
| `fdsave` | 21 | 204 | `c12a72e2…` |
| `dato` | 13 | 1285＋DATO.DAT 137 筆＋544 張頭像 | `cd0c5d11…` |
| `afm` | 6 | 1158＋ANI.DAT 289 幀 | `346f368c…` |
| `fdicon` | 31 | 3336 | `72003b8f…` |
| `fdtxt` | 13 | FDTXT.DAT 全部＋1824 字形 | `46b35584…` |
| `figani` | 86 | 5061 | `143e1de1…` |
| `fdother` 核心與分離載入 | 103 | 22200 | `e7bac982…` |
| `fdother` 原版流程邏輯 | 66 | 21362 | `e3aa5a4e…` |
| `fdother` 其餘流程 | 71 | 15249 | `edb5b35a…` |

- 每個黃金檔都在子代理 worktree 與主 checkout 各產一次，SHA 相同；每份都做過竄改反向檢查。
- `tools/rust_formats_test.sh` 在 `FD2_REQUIRE_ORIGINAL=1` 下跑完 421 項單元測試，沒有任何略過。讀原版檔的環境變數統一為 `FD2_FLAME2_DIR`、`FD2_ASSET_PACK`、`FD2_EXTRACTED_RAW_DIR`、`FD2_REMAKE_ASSETS`（`src/test_support.rs`）。
- 刻意不照做：分離素材的損壞 JSON／PNG 只比接受或拒收，差異依種類列在各黃金測試的 `KNOWN_DIFFERENCES`；png crate 不驗 zlib Adler-32；`archive` 以無號整數表示資源索引，Go 的負索引在型別上無法傳入；`figani` 在負容量輸入時不 panic。
- 只證明 Rust 與凍結 Go 版一致，沒有新增原版對拍。

### 2026-11-14 R2 完成：戰鬥規則核心

`fd2-rules` 已涵蓋 Go `internal/battle` 的 129 個檔、633 個函式與方法，PORTING.json 全部標 `ported`。做法分兩段：先建立含全部型別、常數與函式簽名的骨架，定下單位所有權設計（單位本體只在 `State::units`／`roster`，其他地方以索引參照）；再分六組、三個波次平行補上本體。分組、決定與每組紀錄在 `remake-rs/crates/fd2-rules/PORTING.md`。

| 組 | 內容 | Go 測試移植 | 黃金案例 | 黃金檔 SHA-256 |
|---|---|---|---|---|
| — | Go `math/rand` 序列 | 2 | 146432 個值 | `5fbfe200…` |
| A | 單位、記錄、物品欄 | 144 | 6797 | `40a7130d…` |
| B | 指令、目標、死亡程式 | 136 | 8541 | `c677d580…` |
| C | 物品效果、面板繪製 | 97 | 2481 | `e5e6610b…` |
| D | 敵我 AI | 108 | 5570 | `0bd44fb5…` |
| E | 戰鬥、成長、法術 | 83 | 5188 | `30b4c98a…` |
| F | 劇本事件、回合、快照 | 77 | 9385 步 | `4101bc16…` |
| 跨組 | AI 戰鬥逐步重播與跨組路徑 | — | 99 場、36552 步 | `32d0fd19…` |

- 完成條件「Go 的戰鬥決定性測試以相同種子產生相同結果」：Go 測試的固定種子期望值不改即全部通過；跨組黃金在 33 張地圖 × 3 個種子上跑敵我 AI 戰鬥（上限 24 回合），每一步的計畫、結果、單位變化與下一個亂數都和 Go 相同，結局 `StateSnapshot` 相同。
- `tools/rust_rules_test.sh` 在 `FD2_REQUIRE_ORIGINAL=1` 下 647 項單元測試通過，沒有 ignore 或略過；clippy `-D warnings` 乾淨。每份黃金都在子代理 worktree 與整合後的主 checkout 各產一次，SHA 相同，並做過竄改反向檢查。
- 快照與 Go 雙向相容：Rust 讀 Go 寫的快照再寫出與 Go 相同，Rust 寫出的快照 Go 能讀回。
- Go `math/rand` 的 `rngSource` 與 `Intn` 取樣在 `battle/rng.rs` 重現，是「不移植 Go 標準函式庫」的具名例外之一；2026-11-14 使用者另同意登記 `fd2-formats` 的 `go_math`（`math.Sin`／`Cos`）與 `fd2-campaign` 的 `go_filepath`（`filepath.Clean`／`Join`），2026-11-15 再同意登記 `fd2-game` 的 `go_unicode`（斷行用的 `unicode.Han` 等 Script 表與 `unicode.Is`／`In`）與 `go_json_float`（`encoding/json` 寫 `float64` 的格式），以及 `go_x_image_font`（R5 TTF 用的 `golang.org/x/image` 字型度量慣例，第六項），清單見 `docs/data/fd2-lessons.json` 的 `rust-port-no-go-stdlib`。
- `serde_json` 開啟 `float_roundtrip`，讀小數與 Go 一樣正確捨入。
- 跨組黃金抓到一處：Go `native_ai_action_runtime.go` 531 行以 `:=` 重新賦值了 `found`，診斷欄位 `NativeModeBlockedFound` 實際寫入「找到最近對立單位」。Rust 照 Go 的實際值寫入並在註解寫明。
- 刻意不照做：Go 解 JSON 時欄位名不分大小寫，Rust 只認標籤原文；損壞輸入只比接受或拒收；Go map 迭代順序造成的平手與錯誤訊息選擇（`combat.go` reach 平手在重播中出現 0 次）；nil 接收者與不在 `State.Units` 裡的單位指標。各組清單在各黃金測試的 `KNOWN_DIFFERENCES`。
- 跨組重播的我方每回合直接結束回合，並省略死亡程式、格子事件、帶演出的出場與部分原生指令效果，這些路徑由各組單元測試與黃金涵蓋；完整流程的狀態軌跡留給 R4。
- 只證明 Rust 與凍結 Go 版一致，沒有新增原版對拍；`58` 的證據分層沒有變動。

### 2026-11-14 R3 完成：戰役與內容層

`fd2-campaign` 已涵蓋 Go 戰役與內容層的七個套件（`campaign`、`ending`、`editorcanonical`、`localization`、`musiccatalog`、`indexedmap`、`battlepresent`），每個套件一個同名模組。`campaign`／`ending`／`editorcanonical` 比照 R2 先建骨架再分三組平行補本體；其餘四個套件由一組直接移植。分組、決定與各組紀錄在 `remake-rs/crates/fd2-campaign/PORTING.md`。

| 組 | 內容 | Go 測試移植 | 黃金案例 | 黃金檔 SHA-256 |
|---|---|---|---|---|
| present | 語系與音樂清冊 | 20 | 2744 | `e446bb6b…` |
| present | indexedmap 地圖繪製 | 55 | 1196 | `a9aa1288…` |
| present | battlepresent 戰鬥演出 | 70 | 174 | `b5bdb374…` |
| A | 劇本資料與處理器編譯 | 177 | 492 案例、2905 項比對 | `5958f83f…` |
| B | 設施畫面與城鎮／商店／教會規則 | 94 | 2573 | `b4fc7f00…` |
| C | 存檔、CONTINUE、JOIN 與結局 | 112 | 607 | `9004ce71…` |

- 完成條件「讀入全部 `remake/assets` 劇本與語系，結果與 Go 黃金資料相同」：campaign-a 讀入全部劇本資料（scenarios 32、handlers 64、bindings 368、acting 12、dialogue-index 1 與 editor-canonical，共 545 個輸入檔），`CompileCampaignJSON` 輸出與 Go `json.Marshal` 逐位元組相同；present-loc 讀入四個語系的 pack、content、entities 與音樂清冊。
- `tools/rust_campaign_test.sh` 在 `FD2_REQUIRE_ORIGINAL=1` 下 528 項單元測試通過，與 Go 七個套件的測試函式數相同，沒有 ignore 或略過；clippy `-D warnings` 乾淨。每份黃金都在子代理 worktree 與整合後的主 checkout 各產一次，SHA 相同，並做過竄改反向檢查。
- 依賴：`fd2-campaign` 使用工作區已鎖定的 `png 0.18.1`（分離素材的 indexed 與灰階 PNG）與 `serde_json` 的 `raw_value`；`fd2-rules` 的 `go_quote` 等五個 Go 格式化輔助改為公開共用。`Cargo.lock` 沒有新套件或新版本。
- 對齊 Go 實際行為的地方：`PhaseBannerStepMillis` 直接寫 Go 無型別常數在編譯期捨入後的值，Rust 以 f64 執行期除法會差 1 ulp；`localization` 的 `Format` 依 Go 1.22 實測對齊官方語系用到的動詞。
- 刻意不照做：Go 解 JSON 時欄位名不分大小寫與重複鍵；Go map 迭代順序決定的「多項不合法時先報哪一項」與 `ClassChangeCandidates` 不在 order 裡的 id 的附加順序（黃金改比集合）；`%#v`、`%v` 印指標等診斷文字。各組清單在各黃金測試的 `KNOWN_DIFFERENCES`。
- 只證明 Rust 與凍結 Go 版一致，沒有新增原版對拍；`58` 的證據分層沒有變動。

### 2026-11-15 R4 完成：遊戲狀態機

`fd2-game` 已涵蓋第一章受控流程（標題、開場劇情、第一章戰鬥、城鎮與存讀檔）執行到的 `cmd/fd2` 函式。做法依 [120](120-goal-rust-r4-game-state-split-20261114.md)：階段 0 以第一章重播的覆蓋率圈出 565 個函式；階段 1 由產生器建立 186 個模組的骨架；階段 2 分六組、三個波次平行移植；階段 3 定義呈現回報介面，並把 Go 的章重播測試移植成無畫面驅動端。分組與各組紀錄在 `remake-rs/crates/fd2-game/PORTING.md`。

| 項目 | 結果 |
|---|---|
| 移植 | 骨架範圍 565 個函式移植 553 個，剩下 2 個純繪圖（R5）與 10 個 HD 本體（R6）；另移植 78 個原標 R8 的小函式 |
| 第一章檢查點 | Rust 重播 40 個檢查點與 Go 黃金逐位元組相同，SHA-256 `1c846f57…`；子代理 worktree 三次、整合後主 checkout 一次，結果相同 |
| 存檔 | 流程結束時的 `FD2.SAV` 逐位元組相同（`f3942ff2…`）；`hotel-save.json` 在 #319 修正後也逐位元組相同（`190b9f8f…`） |
| 讀 Go 版存檔 | 存讀檔黃金 349 個案例（`dab44975…`），含 7 份 Go 版實際寫出的存檔與 2 份設定檔；191 份接受的存檔由 Rust 讀回再寫出，全部與 Go 逐位元組相同（#319） |
| 原版收據層 | `verify_chapter_parity.py` 以收據同一套政策判 Rust 與 Go 輸出，兩份都是 behavior、nodes、transaction 通過，保存為 `accepted_disclosed_difference`，結論相同 |

- 完成條件「第一章受控流程的狀態軌跡與 Go 黃金軌跡相同；能讀 Go 版的存檔」：依上表成立。比對入口 `tools/rust_parity_ch01.sh`，竄改檢查點的單位 HP 或亂數字組時比對失敗。
- 時間與輸入：Go 讀牆鐘的地方改讀虛擬時鐘，每個邏輯幀前進固定步長；輸入只來自 `TickInput`。
- 呈現回報：`Game::acknowledge_present` 做 Go `Draw` 裡改流程狀態的部分，不畫像素；外殼每畫完一幀呼叫一次，無畫面驅動端在 Go 呼叫 `Draw` 的地方呼叫。第一章用到的分支已移植，其餘分支停在 R5／R6／R8 的 `todo!`。
- `tools/rust_game_test.sh` 在 `FD2_REQUIRE_ORIGINAL=1` 下 310 項通過，4 項 ignore：存讀檔黃金 2 項與章重播 1 項由各自的腳本執行，1 項停在第一章不會走到的 R8 分支。clippy `-D warnings` 乾淨，`Cargo.lock` 沒有新套件。
- 斷行用的 Unicode Script 表照 Go 1.22.12 放在 `go_unicode`，是第四項具名例外。
- 存檔 JSON 的位元組（#319）：`fd2-rules` 的 `NilableVec`／`NilableMap` 記住 Go 切片與 map 的 nil 狀態，`fd2-game` 的 `go_json_float` 照 Go `encoding/json` 寫浮點數（第五項具名例外）；存讀檔黃金與章重播的比對都拿掉了 battle 子樹的正規化。
- 刻意不照做與尚待處理：crate 層的未使用警告豁免保留到 R5 接上正式入口（拿掉後 882 則，多數是 R8 範圍的函式）；第一章沒有執行到的流程留給 R8。
- 只證明 Rust 與凍結 Go 版一致；原版收據層沿用 2026-10-17 的原版輸出，沒有新增原版對拍，`58` 的證據分層沒有變動。
