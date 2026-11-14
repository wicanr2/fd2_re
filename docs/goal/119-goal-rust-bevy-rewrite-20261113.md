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
| R6 HD1 與工具列 | HD 主題、F2／F11、畫質、速度設定 | HD1 清冊全部素材可載入；切換與設定保存相容 |
| R7 平台 | Windows、macOS、Linux、網頁、手機打包 | 三平台完整包可組裝；Windows 實機驗收 #303～#305 |
| R8 切換 | 30 章受控流程對拍、正式發行 | 依 `111` 的章收據規則重跑；通過後 Rust 版成為正式主線，Go 版封存 |

各階段工單：R0 [#306](https://github.com/wicanr2/fd2_re/issues/306)、R1 [#307](https://github.com/wicanr2/fd2_re/issues/307)、R2 [#308](https://github.com/wicanr2/fd2_re/issues/308)、R3 [#309](https://github.com/wicanr2/fd2_re/issues/309)、R4 [#310](https://github.com/wicanr2/fd2_re/issues/310)、R5 [#311](https://github.com/wicanr2/fd2_re/issues/311)、R6 [#312](https://github.com/wicanr2/fd2_re/issues/312)、R7 [#313](https://github.com/wicanr2/fd2_re/issues/313)、R8 [#314](https://github.com/wicanr2/fd2_re/issues/314)。R0～R4 各階段收尾時，在下方「進度」加一個標題寫明該階段已完成的小節（格式見 R0），工單的機器檢查以這個標題判定；其他段落不要寫出那個字串，以免誤判。

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
- Go `math/rand` 的 `rngSource` 與 `Intn` 取樣在 `battle/rng.rs` 重現，是「不移植 Go 標準函式庫」的唯一具名例外（見 `docs/data/fd2-lessons.json` 的 `rust-port-no-go-stdlib`）。
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

