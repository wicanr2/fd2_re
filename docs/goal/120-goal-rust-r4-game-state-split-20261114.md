# 120 R4 遊戲狀態機：`cmd/fd2` 的流程與畫面拆分

依 [119](119-goal-rust-bevy-rewrite-20261113.md) 的 R4（工單 [#310](https://github.com/wicanr2/fd2_re/issues/310)）。R4 把凍結 Go 版 `remake/cmd/fd2` 的流程部分移成 `remake-rs/crates/fd2-game`：不依賴 Bevy，給定狀態、輸入與亂數就有固定結果。畫面、音訊與平台部分留給 R5 的 `fd2-app`。

## 現況量測

2026-11-14 在凍結點 `5186e022` 之後的 `remake/cmd/fd2` 以 `wc`／`grep` 量測：

| 項目 | 數量 |
|---|---|
| 正式程式 | 255 檔、77697 行；其中 121 檔 import Ebiten |
| 沒有任何 Ebiten 呼叫的檔 | 131 檔、21979 行 |
| `main.go` | 13464 行，含 `Game` 結構、`Update`、`Draw` |
| `Game` 結構 | 欄位宣告約 560 行；`func (g *Game)` 方法 1158 個 |
| `modern_*` HD 呈現 | `modern_hd*` 105 檔、21277 行 |
| `native_command*` 動畫指令播放 | 22 檔、9801 行 |
| 「畫過才推進」的旗標 | `drawn`／`presented` 類欄位 64 檔、451 處 |
| 測試 | 395 檔、1461 個 `Test` 函式；150 檔建立 `ebiten.Image` 或呼叫 `Draw` |

## 流程和畫面的耦合點

Go 版的 `Game` 是一個單體：場景流程、演出工作、輸入與繪圖資源都掛在同一個結構上。拆分要處理五種耦合。

| 耦合 | Go 版寫法 | `fd2-game` 的做法 |
|---|---|---|
| 輸入 | `Update` 與各 `step*` 函式直接呼叫 `inpututil.IsKeyJustPressed`、`ebiten.IsKeyPressed` | 每個邏輯幀收一份 `TickInput`（本幀按下、放開、按住的邏輯鍵）；實體鍵對應由 `fd2-app` 負責 |
| 時間 | `gameNow()` 取牆鐘 `time.Now()` 乘遊戲速度；`nativeBIOSClock` 由牆鐘換算 BIOS tick；19 個函式收 `now time.Time` | 虛擬時鐘：每個邏輯幀前進固定步長再乘速度，`now` 參數照 Go 保留，來源改成虛擬時鐘 |
| 畫過才推進 | 演出工作在 `Draw` 裡設 `drawn = true`，下一次 `Update` 才前進；離屏測試以 `ackPresents` 補標 | 狀態機發出「呈現請求」，外殼畫完回報「已呈現」；無畫面驅動端直接回報。依賴關係變成明確的介面，不藏在 `Draw` 裡 |
| 狀態裡的繪圖資源 | 演出工作保存 `*ebiten.Image`（例如攻擊場景的 `preludeImages`） | 狀態只保存索引色畫面、調色盤與幀數；轉成貼圖是外殼的事 |
| 延續閉包 | 工作結束時呼叫 `after func()`，閉包捕捉 `g` | `Box<dyn FnOnce(&mut Game)>`，取出後以 `&mut self` 呼叫；能列舉的延續改成 enum |

`Game` 結構在 Rust 保留單一型別，欄位只留流程用的部分；每個 Go 檔對應一個模組，各自寫 `impl Game`。這和 R2、R3「一個 Go 檔一個模組」的做法相同，移植時可以逐函式對照。

## 範圍

R4 的完成條件是第一章受控流程：標題、開場劇情、第一章戰鬥、城鎮與存讀檔。實際要移植哪些函式，以 Go 版跑這段流程時執行到的程式為準，不憑檔名猜。

| 部分 | 處理 |
|---|---|
| 第一章流程執行到的流程函式 | R4 移植 |
| 同一批函式裡的繪圖部分 | 記下拆分點，R5 移植 |
| 存檔、設定讀寫（`save.go`、`settings.go`） | R4 移植，要能讀 Go 版的存檔與設定 |
| `modern_*` HD 呈現、海報、工具列 | R6 |
| 音訊播放 | R5；`fd2-game` 只發出播放請求 |
| 截圖模式、效能追蹤、執行紀錄 | 除錯工具，視需要放 `fd2-tools` |
| 第一章沒有執行到的流程（其他章的處理器、結局等） | R8，見「其他章的流程」 |

R4 只跑忠實模式。Go 版的 F2 只重建圖層、不改戰況，所以 HD 不影響流程軌跡；這點在階段 0 以覆蓋率核對。

## 驗證

沿用 119 的三層驗證。第一章已有原版對拍收據 [`parity-ch01.json`](../data/ui-traces/parity-ch01.json)（2026-10-17，流程驗收，40 個檢查點），R4 直接重用它的控制計畫與原版輸出。

| 層 | 做法 |
|---|---|
| 1. Go 測試移植 | 範圍內函式的 Go 測試逐條移植；斷言畫面的測試列入 R5 清單，不在 R4 略過計數 |
| 2. Go 黃金軌跡 | 凍結 Go 版以 `TestChapterParityReplay` 重播 `ch01-start-flow-r4.jsonl`，對照原版輸出 `work/ch01-flow-20261017/original-full-r4`，關閉畫面擷取，在 `FD2_PARITY_OUT` 寫出 `checkpoints.jsonl`；Rust 重播同一份計畫，產出同格式檢查點後逐欄比對。流程結束時的存檔位元組也比對 |
| 3. 原版 oracle | Rust 的檢查點交給 `tools/verify_chapter_parity.py`，以收據同一套政策（`ch01-start-residual-20261017`）判定行為、節點與交易閘門 |

比對單位是檢查點，不是逐幀。Go 版重播用牆鐘，兩個檢查點之間跑了幾幀會隨主機速度變動；檢查點只出現在玩家決策點與場景轉換，不受這個影響。這和既有收據「同步之間的演出消耗兩側不逐 tick 對齊」是同一個限制，亂數也同樣在決策點依原版紀錄對齊。

Go 的 `cmd/fd2` 是 `package main`，別的程式無法 import，所以 Go 側的黃金輸出只能放在 `cmd/fd2` 的測試檔裡。既有重播測試已能輸出 `checkpoints.jsonl`；需要補欄位時，新增以環境變數啟用、只輸出資料的 `_test.go`，屬於 119「只輸出黃金資料的工具」例外，不改任何遊戲行為。

另外對收 `now` 參數的演出工作做單元層黃金：以固定的虛擬時間序列驅動 Go 與 Rust，逐幀比對狀態。這一層可以逐幀，因為時間由測試給定。

## 階段

| 步驟 | 內容 | 產出 |
|---|---|---|
| 0. 範圍盤點 | 在 Docker 以 `-coverprofile` 跑第一章重播與標題、城鎮、存讀檔相關測試；逐函式分成流程、畫面、混合、平台 | `fd2-game/PORTING.json`（函式清單與分類）、混合函式的拆分點 |
| 1. 骨架 | `Game`（流程欄位）、`TickInput`、虛擬時鐘、呈現請求與回報、每個 Go 檔一個模組；函式本體 `todo!()` | 可編譯的骨架與簽名變更紀錄 |
| 2. 分組移植 | 依階段 0 的相依關係分組，各組在自己的 worktree 平行移植並移植測試 | 各組測試與單元層黃金通過 |
| 3. 整合 | 無畫面重播驅動端；Go 黃金軌跡比對；`verify_chapter_parity.py`；讀 Go 版存檔 | 119「進度」的 R4 收尾小節與收據 |

分組已在階段 1 定案為六組，見「階段 1 結果」。

## 階段 0 結果（2026-11-14）

以 [`tools/rust_r4_ch01_coverage.sh`](../../tools/rust_r4_ch01_coverage.sh) 在 Docker 內重播第一章，結果存於 [`rust-r4-ch01-coverage-20261114.json`](../data/rust-r4-ch01-coverage-20261114.json)。

| 項目 | 結果 |
|---|---|
| Go 黃金軌跡 | 40 個檢查點，SHA-256 `1c846f573434658c…`；兩次執行逐位元組相同，也與原版收據採用的 `remake-full-r10` 相同 |
| 覆蓋率 | `cmd/fd2` 敘述 15.2%；1922 個正式函式中執行到 565 個，分布在 138 個檔 |
| 草稿分類 | 流程 462、混合 21、畫面 15；HD／工具列檔 67（忠實模式下仍會執行的預檢與狀態） |
| 牆鐘影響 | 兩次執行只有嘴型動畫 `stepDialogueMouth` 的覆蓋比例不同，函式集合相同 |

`main.go` 佔 154 個，其次是音訊、標題、出場、指令環、字型、死亡演出與城鎮介面。分類只看名稱、簽名與是否呼叫 Ebiten，骨架階段要逐函式核對，特別是被列為「流程」但經由輔助函式建立圖片的函式。

### 骨架的設計輸入

| 項目 | 量測 | 對骨架的意義 |
|---|---|---|
| `Game` 欄位 | 538 個，第一章執行到的函式用到 516 個 | 重設類函式會碰到幾乎所有欄位，不能靠刪欄位縮小範圍；`Game` 整體移植，只替換帶繪圖資源的欄位 |
| 直接是 Ebiten 型別的 `Game` 欄位 | 18 個（地圖圖塊、頭像、FIGANI、對話框、面板等） | 改成索引色畫面或素材索引，貼圖由 `fd2-app` 建立 |
| 持有 Ebiten 圖片或音訊的結構 | 282 個結構中有 47 個直接持有，19 個在 HD 檔；連同間接持有共 136 個 | 非 HD 的 28 個多半是動畫指令的演出工作，改存索引色幀與幀數 |

盤點方式：解析 `cmd/fd2` 正式程式的結構宣告與第一章執行到的函式本體（2026-11-14）。

## 階段 1 骨架的對應規則（2026-11-14）

型別與所有權沿用 R2、R3（`fd2-rules`、`fd2-campaign` 的 `PORTING.md`）。`cmd/fd2` 特有的部分如下。

| Go | `fd2-game` |
|---|---|
| `package main`（單一套件） | crate `fd2-game` 的模組 `game`；每個 Go 檔一個子模組 `game/<檔名>.rs`，各自寫 `impl Game` |
| 函式範圍 | 只產生[階段 0 清單](../data/rust-r4-ch01-coverage-20261114.json)的 565 個函式；同檔其餘函式在 `PORTING.json` 標 `r8`，不產生 |
| 型別範圍 | `Game` 欄位與上述函式簽名可達的全部型別 |
| `*ebiten.Image` | `Option<Arc<Picture>>`；`Picture` 是 RGBA 像素（寬、高、位元組），索引色畫面用 `IndexedFrame`。GPU 貼圖由 `fd2-app` 建立 |
| `inpututil`、`ebiten.IsKeyPressed` | 每個邏輯幀的 `TickInput`（本幀按下、放開、按住的 `Key` 集合）；`Key` 只列 Go 用到的鍵 |
| `time.Time`、`gameNow()` | `GameInstant`（虛擬時鐘的奈秒）；`time.Duration` 是 `GoDuration`（`i64` 奈秒） |
| `drawn`／`presented` 旗標 | 欄位照留；由無畫面驅動端或 `fd2-app` 回報「已呈現」時設定 |
| `audio.Player` | `AudioVoice`（播放請求與狀態）；實際播放由 `fd2-app` |
| `func()` 欄位 | `Option<Box<dyn FnOnce(&mut Game) + Send>>`；函式參數 `func` 是 `&mut dyn FnMut` |
| `*battle.Unit` 欄位 | `Option<UnitIdx>`，指向 `st.units` |
| HD（`modern_*`、海報、工具列）型別 | 保留型別名；欄位只留忠實模式下第一章會執行到的，其餘 R6 |
| `error` | `GameError`（同 `BattleError` 形狀，`Display` 與 Go 逐字相同） |

骨架由受版控的產生器（`tools/rust_r4_skeleton/`）以 Go `go/types` 分析 `cmd/fd2` 產生，可以重跑；R2、R3 的產生器沒有進版控。

## 階段 1 結果（2026-11-14）

骨架提交 `b8dbcfc3`，由 [`tools/rust_r4_skeleton.sh`](../../tools/rust_r4_skeleton.sh) 產生；在主 checkout 重跑，191 個輸出檔的總雜湊 `c076fdd5…` 相同。逐項落地與偏離見 [`fd2-game/PORTING.md`](../../remake-rs/crates/fd2-game/PORTING.md)。

| 項目 | 結果 |
|---|---|
| 規模 | 186 個 Go 檔各一個子模組；型別 221 個（HD 87 個只留第一章用到的欄位）、常數 143、套件層變數 19；565 個函式有簽名與 `todo!()`，其餘 1357 個在 `PORTING.json` 標 `r8` |
| `Game` | 538 個 Go 欄位全部保留，另加 `tick_input` 與 `virtual_now` |
| 檢查 | `tools/rust_game_test.sh` 的 fmt、clippy `-D warnings` 通過；`fd2-rules` 653 項、`fd2-campaign` 530 項照舊通過；`Cargo.lock` 沒有新外部套件 |

和上面對應規則不同的地方：

- 會重複呼叫的 func 欄位（觀察者、交易、時鐘注入，共 9 個）用 `FnMut`；只呼叫一次的延續照規則用 `FnOnce`。
- 會被存進工作結構的 func 參數（30 個）收 `Option<Box<dyn FnOnce..>>`，借用的閉包存不進 `Game`。
- `*ebiten.Image` 參數依用途分成要畫上去的 `&mut Picture` 與只讀的 `&Arc<Picture>`。
- Go `any` 只用來比對擁有者是否相同，對應成 `Option<OwnerToken>`。
- 常數與變數只產生 565 個函式用到的；`go:embed` 的一個素材初始式暫留 `todo!()`。
- `fd2_campaign::indexedmap::NativeFrameInput` 與 `NativeTransitionFrameInput` 借用呼叫端資源，Go 卻把它們存在 `Game` 與演出工作裡。定案：`fd2-game` 保存資源本身，每幀再組出借用的輸入，`fd2-campaign` 不改；骨架暫寫 `<'static>` 只為了編譯，階段 2 由負責的組改掉。

階段 2 分六組，`main.go` 的函式依呼叫關係分給各組，各組只改自己函式的本體；`Game` 結構與共用型別要調整時交主代理整合。

| 組 | 主題 | 函式數 |
|---|---|---|
| A | 主迴圈、啟動、標題與開場（含 `Update`、`Draw`、`Layout`） | 156 |
| B | 劇情對話與事件演出 | 108 |
| C | 戰場操作、指令環與敵方回合 | 83 |
| D | 攻擊演出與動畫指令播放 | 120 |
| E | 城鎮、商店、教會與整備 | 70 |
| F | 存讀檔與設定 | 28 |

組間呼叫有循環，沒有一組完全獨立；同一波平行移植，跨組的測試照 R3 標 `ignore` 等待。原本預估的「敵方回合」只有 13 個函式，併入 C。

## 其他章的流程（使用者 2026-11-14 定案）

第一章沒有執行到的流程程式（其他章的處理器、結局等）放進 R8。R8 每一章先以該章重播的覆蓋率補移植缺少的函式，再跑該章對拍。R4 的範圍表裡「第一章沒有執行到的流程」一列依此處理。
