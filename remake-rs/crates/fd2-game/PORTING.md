# fd2-game 移植說明

## R4 階段 1 骨架（2026-11-14）

`fd2_game::game` 對應凍結 Go 版 `remake/cmd/fd2`（`package main`）的流程部分，依
[120](../../../docs/goal/120-goal-rust-r4-game-state-split-20261114.md)「階段 1 骨架的對應規則」。階段 1 把
`Game` 與可達的型別、函式本體用到的常數與表格，以及階段 0 清單的 565 個函式簽名搬過來；函式本體是
`todo!("<file>.go:<FuncName>")`。型別與所有權的決定寫在 [`src/game.rs`](src/game.rs) 的模組文件，支援型別在
[`src/game/support.rs`](src/game/support.rs)。

逐檔、逐函式的機器可讀清單在 [`PORTING.json`](PORTING.json)：每個 Go 檔的 Rust 路徑與行數、型別／常數／變數對應、
每個函式（含未產生的）的 Rust 名稱、狀態（`todo` 是骨架已有簽名，`r8` 是第一章沒有執行到、這次不產生）、階段 0
的分類與覆蓋率、階段 2 組別、檔內呼叫到的其他選定函式，以及型別對應表、callback 欄位種類與分組統計。

### 骨架現況

以凍結 Go 版 `remake/cmd/fd2`（階段 0 量測時的 `5747771c`，`cmd/fd2` 之後沒有改動）產生。

| 項目 | 數量 |
|---|---|
| Go 正式檔（linux build，不含 `_test.go`）／行數 | 254／77822 |
| 被 build tag 排除的 Go 檔 | 3（`assets_glob_js.go`、`run_log_windows.go`、`separated_asset_path_js.go`） |
| 有 Rust 子模組的 Go 檔 | 186（其餘 68 檔沒有選定函式、也沒有可達的型別或用到的常數） |
| 型別 | 221（套件全體 311）：結構 197、具名整數 newtype 23、別名 1（`sfxVoice` → `AudioVoice`）；其中 HD 型別 87 |
| 常數 | 143（選定函式本體用到的；套件全體 230） |
| 套件層變數 | 19（套件全體 62）：唯讀表格 11、執行期會改寫的 6（`RwLock`）、`errors.New` 哨兵 2 |
| 函式 | 1922 個中產生 565 個（`todo`），其餘 1357 個標 `r8` |
| `Game` 欄位 | Go 538 個全部保留，另加 Rust 的 `tick_input`、`virtual_now` |
| Rust 行數 | 約 12100（含 `support.rs`） |

- 每個 Go 檔 `x.go` 一個子模組 `src/game/x.rs`，各自寫 `impl Game`（或其他接收者型別）。手寫、沒有 Go 對應的檔只有
  `support.rs`。子模組以 `use super::*` 互相看見，和 Go 的單一套件命名空間相同。
- 每個函式的 doc 保留 Go 註解，接著一行 `Go：<檔>:<行>` 與原始 Go 簽名，再接簽名改寫的說明行（接收者 nil 檢查、
  延續參數、借用回傳等）。Go 的 JSON 標籤保留在欄位 doc 的「Go 標籤：」。
- `cargo check`、`cargo clippy --offline --locked -p fd2-game --all-targets -- -D warnings`、`cargo fmt --check` 通過
  （`tools/rust_game_test.sh`）。`lib.rs` 的 `allow(unused, dead_code, unreachable_code, clippy::todo)` 是骨架專用，
  階段 2 全部移植後移除；`allow(clippy::too_many_arguments, clippy::type_complexity)` 與兩個 doc lint 照
  `fd2-rules` 長期保留。
- `Cargo.toml` 依賴 `fd2-campaign`、`fd2-formats`、`fd2-rules` 與工作區已鎖定的 `base64`、`serde`、`serde_json`、`png`、
  `sha2`（版本照 `fd2-campaign`）。`Cargo.lock` 只多了 `fd2-game` 這個工作區成員與它的依賴邊，沒有新外部套件或版本。

### 產生器

骨架由受版控的產生器產生，可以重跑：

```
tools/rust_r4_skeleton.sh          # 在工作樹副本中執行時加 FD2_GOCACHE=<主工作區>/work/gocache
```

- 第一個容器（`fd2-go-test-local`，`--network none`）跑 `go list -export -deps -json ./cmd/fd2` 取得相依套件的匯出資料，
  建置 [`tools/rust_r4_skeleton/`](../../../tools/rust_r4_skeleton) 並以 `go/parser`＋`go/types` 分析 `cmd/fd2` 正式程式。
  `remake/` 唯讀掛載，產生器不在 `remake/` 底下。
- 第二個容器（`fd2-rust`）以 `rustfmt` 排版；最後取代本 crate 的 `Cargo.toml`、`PORTING.json` 與 `src/`。**本檔
  （`PORTING.md`）是人工維護的，產生器不會動它。**
- 輸入：`remake/cmd/fd2`、[`docs/data/rust-r4-ch01-coverage-20261114.json`](../../../docs/data/rust-r4-ch01-coverage-20261114.json)
  （SHA-256 寫在 `PORTING.json` 的 `coverage_input`），以及 `fd2-rules`、`fd2-formats`、`fd2-campaign` 的 Rust 原始碼
  （外部型別的 Rust 名稱、derive、方法的接收者是 `&self` 還是 `&mut self`）。
- 腳本最後印出輸出檔數與總雜湊（每個輸出檔 SHA-256 清單的 SHA-256）。2026-11-14 連跑兩次都是 191 個檔、
  `c076fdd5f995ac230cf36166b26e8ac9b038bdf7a603e1666adef9f89a66ced2`。
- 階段 2 開始移植後，crate 內的檔案改為人工維護；再跑產生器會蓋掉移植成果，只用來重現骨架提交或比對簽名。

產生器的檔案：`load.go`（`go list` 匯出資料與型別檢查）、`scope.go`（函式選取、型別閉包、HD 欄位裁剪）、
`analysis.go`（參數改寫、nil 檢查、延續參數、借用回傳）、`rtype.go`（型別對應與 trait）、`emit.go`（Rust 輸出）、
`groups.go`（分組）、`generate.go`（命名與 `PORTING.json`）、`rustscan.go`（掃描既有 crate，改寫自 R3 產生器）、
`templates/`（`support.rs`、`game.rs` 檔頭、`lib.rs`、`Cargo.toml`）。

### 對應規則的實際落地

| Go | Rust | 範圍 |
|---|---|---|
| `*ebiten.Image`（欄位、容器元素、回傳值） | `Option<Arc<Picture>>` | 120 的規則，容器元素也保留 `Option`（Go 的畫格快取常留 nil） |
| `*ebiten.Image` 參數 | 會畫上去的 `&mut Picture`；只讀的 `&Arc<Picture>`（可直接存進狀態）；Go 檢查 nil 時包 `Option` | 11 個參數是 `&mut Picture` |
| `*image.NRGBA`、`image.Image` | `Option<Arc<Picture>>` | |
| `*image.Paletted` | `Option<Arc<IndexedFrame>>` | |
| `*image.RGBA`、`image.Point`、`image.Rectangle` | `fd2_campaign::campaign` 的 `Rgba`、`Point`、`Rectangle` | 與 R3 共用同一份 |
| `color.RGBA`、`color.NRGBA`、`color.Palette` | `ColorRgba`、`ColorNrgba`、`ColorPalette`（`Vec<ColorRgba>`） | |
| `inpututil`、`ebiten.IsKeyPressed` | `Game::tick_input: TickInput` | `Key` 列出 Go 正式程式用到的 33 個鍵 |
| `time.Time`、`gameNow()`、`time.Duration` | `GameInstant`、`Game::virtual_now`、`GoDuration` | `now` 參數照 Go 保留 |
| `drawn`／`presented` 旗標 | 欄位照留 | 呈現請求與回報的介面在階段 3 的驅動端定義 |
| `*audio.Player`、`sfxVoice`、`*audio.Context` | `Option<AudioVoice>`、`AudioVoice` 別名、`Option<AudioContext>` | |
| func 欄位 | `Option<Box<dyn FnOnce/FnMut(&mut Game, ..) -> R + Send>>` | 見下方偏離 1 |
| func 參數 | `&mut dyn FnMut(..)`；被存起來的是擁有的 `Box` | 見下方偏離 2 |
| `*battle.Unit` | 欄位與容器 `UnitIdx`（欄位 `Option<UnitIdx>`）；`Game` 方法與帶 `*battle.State` 的函式的參數 `UnitIdx`；其他函式 `&battle::Unit`／`&mut battle::Unit` | 同 R2、R3 |
| `error` | 欄位 `Option<GameError>`，回傳 `Result<T>`，參數 `Option<&GameError>` | |
| HD 型別 | 保留型別名；欄位只留選定函式本體（欄位選取或複合字面值）用到的 | 87 個 HD 型別：31 個部分保留、37 個 0 欄位，其餘 19 個欄位全留或不是結構 |
| 套件層變數 | 唯讀 `LazyLock<T>`、執行期會改寫的 `LazyLock<RwLock<T>>`、`errors.New` 哨兵 `static GameError` | 初始式的字面值與複合字面值都翻好了（包括 `cutScript`、`officialLocaleContract`） |

接收者與參數的可變性由 `go/types` 分析：寫入欄位、索引、取址、遞增、`append`／`copy`／`delete`、呼叫會改寫接收者
的方法（套件內以不動點分析；外部 crate 以 Rust 方法收 `&self` 或 `&mut self` 判定）、把參考型別交給會改寫該參數的
函式，以及經區域變數別名的寫入，都算改寫；拿不準的（例如交給未知的外部函式、存進複合字面值）算改寫。
結果：方法 263 個 `&mut self`、66 個 `&self`。`&st.Units[i]` 在 Rust 是 `UnitIdx`，不算借用。

指標回傳值：Go 回傳 `&g.x`、接收者欄位或指向它們的區域變數時回傳借用（`prepare_modern_hd_facility_dialogue`
回傳 `Option<&mut ModernHDFacilityDialogue>`）；其他新建的回傳擁有的值。3 個函式回傳指向參數的指標，生命週期無法
省略，骨架先回傳擁有的值並在 doc 註明。

### 偏離 120 對應規則的地方

1. **func 欄位不一律是 `FnOnce(&mut Game)`。** 120 寫 `Option<Box<dyn FnOnce(&mut Game) + Send>>`。實際上 66 個 func 欄位
   裡，名稱是 `then`／`after`／`rollback`（工作結束時只呼叫一次的延續）的 57 個是 `FnOnce`；另外 9 個會被重複呼叫，
   改成 `FnMut`：`Game` 的 `aiPlanObserver`、`growthRollObserver`、`modernThemeLoader`、`nativeEndingNow`、
   `nativeMapFrozenNow`、`nativeTurnPairRNGInput`，以及 `transaction`（兩個）、`mutate`。Go 的 func 有參數或回傳值時，
   Rust 在 `&mut Game` 之後照列（例如 `then func([]battle.NativeCommandDamageResult)` →
   `FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>)`），`*battle.Unit` 參數是 `UnitIdx`。種類清單在
   `PORTING.json` 的 `callback_fields`。
2. **存起來的 func 參數收擁有的 `Box`。** 120 寫函式參數的 func 是 `&mut dyn FnMut`。Go 把 30 個 func 參數（`then` 19、
   `after` 8、`compose` 2、`action` 1）指定給工作結構的欄位、放進閉包或交給會存起來的參數；借用的閉包不能存進
   `Game`，所以這些參數與欄位同型：`Option<Box<dyn FnOnce(&mut Game, ..) + Send>>`。其餘照 120。
3. **`*ebiten.Image` 參數。** 120 只定了欄位。參數依用途分成 `&mut Picture`（Go 對它 `DrawImage`、`Fill` 等）與
   `&Arc<Picture>`（只讀，可以直接存進狀態）。
4. **`Game` 多兩個 Rust 欄位。** 120 階段 1 要求 `TickInput` 與虛擬時鐘；骨架把它們放成 `Game::tick_input` 與
   `Game::virtual_now`，Go 散在數百個函式的 `inpututil` 查詢與 `gameNow()` 都改讀這兩個欄位，不必改動 Go 對應的簽名。
   驅動端在每次 `update` 前設定。
5. **支援型別比任務列的多。** `Picture`、`IndexedFrame`、`TickInput`／`Key`、`GameInstant`／`GoDuration`、`AudioVoice`、
   `GameError`／`game_err!` 之外，型別閉包還需要 `ColorRgba`、`ColorNrgba`、`ColorPalette`、`AudioContext`
   （`audioCtx`）、`FontFace`／`SfntFont`（TTF 繪字的 `font.Face`、`*sfnt.Font`）、`OwnerToken`；`*os.File` 直接用
   `std::fs::File`，`chan struct{}`（HD 主題背景載入的完成通知）用 `std::sync::mpsc::Receiver<()>`。
6. **`any` → `Option<OwnerToken>`。** `cmd/fd2` 的 `any` 欄位（`nativeConfirmationMouthOwner`、HD 戰鬥的 `owner`／
   `cameraOwner`）只用來比對「是不是同一個擁有者」，不是 JSON 值，所以不沿用 R3 的 `serde_json::Value`。
7. **`GameError` 多記哨兵來源。** 形狀同 `BattleError`，但 `cmd/fd2` 有兩處 `errors.Is`
   （`errAttackPresentationYield`、`errNativeHotelPreparationSlot`），所以 `GameError::sentinel` 建立的錯誤經 `wrap`
   仍保留來源，`GameError::is` 對應 `errors.Is`。
8. **借用資源的外部型別暫用 `'static`。** `indexedmap::NativeFrameInput<'a>`、`NativeTransitionFrameInput<'a>` 借用呼叫端
   的資源，Go 卻把 `*indexedmap.NativeFrameInput` 存在 `Game.nativeMapFrameSource` 與演出工作的 `hdInput`／`hdBaseline`。
   骨架在欄位與回傳值寫 `<'static>`、參數寫 `<'_>`，只為了能編譯；見下方缺口。
9. **常數與變數的範圍。** 120 只定了型別範圍。骨架產生選定函式本體用到的常數與變數（含變數初始式遞移用到的），
   其餘留給用到它們的 `r8` 函式。型別依 120 是 `Game` 與選定函式簽名可達的全部型別。
10. **`go:embed`。** `modernHDGoldDialogueTile` 是 `//go:embed modern_hd_gold_dialogue.png`；素材不在 `remake-rs` 底下
    （Rust 容器只掛 `remake-rs`），骨架的初始式是 `todo!()`，由擁有該檔的組決定 `include_bytes!` 的素材位置或改成執行期讀檔。
11. **同名方法以行號指定。** 階段 0 清單只記函式名稱；`modern_hd_compound_map_source.go` 的 `digest`、`seal`、`validate`
    與 `title_menu_state.go` 的 `Step` 在同檔有兩個接收者不同的同名方法。產生器以階段 0 `go tool cover -func` 的行號
    （`work/r4-stage0-20261114/cover-func.txt`：執行到的覆蓋率非 0，另一個是 0.0%）指定，寫在 `scope.go` 的 `coverageLines`。

### 階段 2 分組

產生器先依檔名把專用檔歸組（`groups.go` 的前綴規則），`main.go`（154 個選定函式）、HD 檔與沒對上的檔再逐函式歸到呼叫邊
（選定函式之間，雙向計數）最多的組；Ebiten 主迴圈入口 `Update`、`Draw`、`Layout` 固定歸 A（否則會被拉到呼叫最多的
演出組）。120 預估的「敵方回合銜接」只有 13 個函式（`native_ai_*`、`native_end_turn_recovery`），併入 C，共六組。

| 組 | 主題 | 函式 | 其中 `main.go` | 其中 HD | 檔數 |
|---|---|---|---|---|---|
| A | 主迴圈、啟動、標題與開場 | 156 | 69 | 13 | 27 |
| B | 劇情對話與事件演出 | 108 | 43 | 2 | 22 |
| C | 戰場操作、指令環與敵方回合 | 83 | 22 | 7 | 23 |
| D | 攻擊演出與動畫指令播放 | 120 | 16 | 25 | 44 |
| E | 城鎮、商店、教會與整備 | 70 | 3 | 20 | 24 |
| F | 存讀檔與設定 | 28 | 1 | 0 | 8 |

各組的專用檔（`main.go` 與跨組的 HD 檔另依函式分給多組，逐函式的組別在 `PORTING.json` 的 `group`）：

- **A**：`title`、`title_menu_state`、`packaged_startup`、`assets`、`assets_glob_native`、`font`、`locale`、`audio`、`run_log`、
  `input_audit_trace`、`separated_asset_path_native`、`separated_ui_assets`、`native_bios_clock`、
  `player_{balance,battle_speed,game_clock,hd_quality,interface,theme_loader}`；HD：`modern_hd_title_interface`、
  `modern_hd_confirmation`、`player_poster_controls`，以及 `modern_hd_text`、`modern_hd_battle_{panel,runtime,surface}` 的一部分
- **B**：`native_story_{dialogue,dialogue_locale,input,map}`、`native_event_dialogue`、`native_confirmation_mouth`、`native_spawn_intro`、
  `native_staging_present`、`native_battle_staging_present`、`native_field_event{61,75,_pending}`、`native_turn_event76`、
  `native_ch{22,23}_post`、`native_ch28_post_present`、`native_palette_{pulse,ramp}`、`native_indexed_transition`；
  HD：`modern_hd_death_stage`、`modern_hd_map_overlay_assets`
- **C**：`action_overlay_runtime`、`native_player_{input,movement}`、`native_map_{assets,frame_input,unit_window}`、`native_battle_name`、
  `native_battle_panel_locale`、`native_command_locale`、`native_item_panel_ui`、`native_turn_staging`、`native_system_info_{locale,ui}`、
  `battle_protection`、`native_ai_{idle_recovery,item_presentation,modifier_presentation}`、`native_end_turn_recovery`；
  HD：`modern_hd_battle_gpu`、`modern_hd_battle_panel_chrome`、`modern_hd_movement_panel`，以及 `modern_hd_battle_panel` 的一部分
- **D**：`native_command{0..9,10_12,24,29,32..35}_*presentation`、`native_command_{heal,modifier}_presentation`、
  `native_physical_{body,scene}`、`native_attack_assets`、`native_action_death_present`、`native_death_{program_runtime,reward_message,reward_ui}`、
  `native_transient_{phase,presentation}`、`native_defeat`、`native_level_up_dialogue`；
  HD：`modern_hd_commands10_12_map`、`modern_hd_compound_map{,_source}`、`modern_hd_player9_map`、`modern_hd_indexed_transition`、
  `modern_hd_physical_sfx`、`modern_hd_transient_expiry`、`modern_hd_dialogue_chrome`、`modern_hd_battle_{camera,scroll}`，以及
  `modern_hd_battle_{panel,runtime,surface}` 的一部分
- **E**：`native_town_{ui,departure,secret_input}`、`native_shop_{ui,locale,transaction_ui}`、`native_church_ui`、`native_class_ui{,_lifecycle}`、
  `native_hotel_ui`、`native_preparation_{ui,input}`；HD：`modern_hd_facility_*`、`modern_hd_preparation_*`、`modern_hd_shop_digits`、
  `modern_hd_service_menu_transition`、`modern_hd_church_confirmation_transition`，以及 `modern_hd_text` 的一部分
- **F**：`save`、`settings`、`balance_save`、`native_current_save`、`native_chapter_slot_save`、`native_continue_handoff`、`native_load_slots_ui`

組間相依（選定函式之間的呼叫邊數，列＝呼叫端所在組，行＝被呼叫端所在組）：

| 呼叫端＼被呼叫端 | A | B | C | D | E | F |
|---|---|---|---|---|---|---|
| A | | 29 | 35 | 33 | 23 | 4 |
| B | 26 | | 16 | 9 | 4 | |
| C | 26 | 16 | | 15 | | 1 |
| D | 35 | 32 | 13 | | 2 | |
| E | 25 | 9 | 2 | 1 | | 3 |
| F | 14 | | 3 | 3 | | |

- 組間呼叫有循環（A 的 `Update` 呼叫各組的 `step*`，各組又呼叫 A 的共用工具），沒有一組完全獨立，F 依賴最少。和 R3
  一樣六組可以同一波次平行移植；呼叫到別組未完成的函式就停在對方的 `todo!()`，對應的測試標
  `#[ignore = "等 <組> 合併"]`，整合時拿掉。
- `main.go` 被六組共用（154 個函式分到 A 69、B 43、C 22、D 16、E 3、F 1）。各組只改自己函式的本體，git 依 hunk 合併；
  `Game` 結構與其他型別定義不歸任何一組改，需要調整時回報主代理。
- A 最大（156 個函式），其中 69 個是 `main.go` 裡只由主迴圈呼叫的流程函式；D 次之（120 個、44 檔），但
  `native_command*_presentation` 二十個檔的結構相同（演出工作＋`step` 函式），可以照同一個範本移植。

### fd2-rules／fd2-campaign 缺口

沒有改那兩個 crate。Go 用到的外部型別都有公開的 Rust 對應（`PORTING.json` 的 `missing_external_types` 是空的）；下列是
形狀不足：

| 需要 | 現況 | 暫時做法 |
|---|---|---|
| 擁有資源、可以存進 `Game` 的 `indexedmap::NativeFrameInput` 與 `NativeTransitionFrameInput`（Go 存在 `Game.nativeMapFrameSource` 與演出工作的 `hdInput`／`hdBaseline`） | 兩個型別帶生命週期 `'a`，以 `Option<&'a T>`／`&'a [u8]` 借用呼叫端的資源（R3 present 組的決定） | 骨架欄位與回傳值寫 `<'static>`；建議 `fd2-campaign` 補擁有資源的版本（或由 `fd2-game` 存資源本身、每幀重組借用的輸入），由主代理決定 |
| `battle::GoMathRand` 的 `Default`、`PartialEq` | 只有 `Clone`、`Debug` | 含它的結構只 derive 做得到的 trait（`Game` 本身因 callback 欄位只有 `Default`） |
| `fdtxt::Font` 等沒有 `Default`／`PartialEq` 的格式型別 | 同 R3 | 產生器依掃描結果自動決定 derive |

### 移植時要注意

- **畫格的型別**：120 要求演出工作只保存索引色畫面與調色盤。骨架照 120 的型別表把 `*ebiten.Image` 欄位寫成
  `Option<Arc<Picture>>`；擁有演出工作的組若確認 Go 的 `*ebiten.Image` 是由索引色畫格轉出（例如攻擊場景的
  `preludeImages`），把欄位改成 `IndexedFrame` 或 `Vec<u8>` 並記在本檔的簽名變更紀錄。
- **接收者 nil 檢查**：104 個方法的 Go 本體檢查 `g == nil`（或其他接收者）；Rust 的 `self` 不會是 null，那些分支不移植，
  對它們傳 nil 的 Go 測試分支也不移植（同 R2）。
- **值接收者**：3 個方法是 Go 值接收者，本體對接收者的改寫只影響副本；Rust 收 `&self`，需要改寫時在本體 `clone()`。
- **callback 借用**：呼叫存在欄位裡的延續前先 `take()`，再以 `&mut self` 呼叫；`FnMut` 的觀察者呼叫完放回。
- **map 迭代順序**：Go `range` map 的順序隨機、Rust `BTreeMap` 有序；結果受順序影響的地方由各組在移植時列出。

### 簽名變更紀錄

階段 2 改動骨架簽名時，在這裡逐條記錄（函式、原簽名、新簽名、理由）。

| 組 | 函式 | 原簽名 | 新簽名 | 理由 |
|---|---|---|---|---|
| A | `Game::start_fade_transition` | `action: Option<Box<dyn FnMut(&mut Game) + Send>>` | `Option<Box<dyn FnOnce(&mut Game) + Send>>` | Go 的 `action` 只呼叫一次 |
| A | `Game::native_restored_party_units_for_load_ch` | `-> Result<Vec<UnitIdx>>` | `-> Result<Option<Vec<battle::Unit>>>` | 回傳的是 LOADCH 新建、不在 `st` 裡的單位；`None` 對應 Go 的 nil 結果 |
| A | `Game::apply_load_ch` | `state: Option<&mut campaign::LoadCHState>` | `Option<&campaign::LoadCHState>` | Go 不改寫 `state` |
| A | `Game::seed_persistent_party_from_load_ch` | `units: &[UnitIdx]` | `units: &[battle::Unit]` | 同上，單位不在 `st` 裡 |
| A | `Game::interface_theme_for_art` | `-> Option<ModernStoryPortraitSet>` | `<'a>(&'a self, Option<&'a ModernStoryPortraitSet>) -> Option<&'a ModernStoryPortraitSet>` | Go 回傳指標；不複製整份主題 |
| A | `Game::interface_asset_theme` | `(&mut self) -> Option<ModernStoryPortraitSet>` | `(&self) -> Option<&ModernStoryPortraitSet>` | 同上 |
| A | `Game::locale_message` | `args: &[Option<OwnerToken>]` | `args: &[localization::FormatArg]` | Go 的 `...any` 是格式參數，不是擁有者比對 |
| A | `Game::play_raw`、`Game::play_native_physical_sound` | `b: &mut [u8]` | `b: &[u8]` | 不改寫樣本；呼叫端傳 `&mut Vec<u8>` 仍可編譯 |
| A | `key_repeat` | `fn key_repeat(k: Key) -> bool` | `fn key_repeat(input: &TickInput, k: Key) -> bool` | Go 讀全域 `inpututil.KeyPressDuration`；改讀本幀輸入 |
| A | `title_menu_hd_seal` | `h: Option<&mut ModernHDTitleMenuItem>` | `h: Option<&ModernHDTitleMenuItem>` | 只讀 |
| A | `freeze_modern_hd_title_menu_item` | `original: Option<&mut Picture>` | `original: Option<&Arc<Picture>>` | Go 以指標身分比對，存同一個 `Arc` |
| A | `title_load_slots_hd_seal`、`freeze_modern_hd_title_load_slots` | `Option<&mut NativeLoadSlotsUIAssets>`（及 `Option<&mut ModernHDTitleLoadSlots>`） | `Option<&NativeLoadSlotsUIAssets>`（及 `Option<&ModernHDTitleLoadSlots>`） | 只讀；F 組的呼叫端改用 `.as_ref()` |
| A（B 的函式，跨組介面） | `Game::step_battle_walk_segment` | `(&mut self, w: &mut WalkAnim, finish: &mut dyn FnMut(i64)) -> bool` | `(&mut self, finish: &mut dyn FnMut(&mut Game, i64)) -> bool` | 見「A 組：跨組介面」；本體仍是 B 的 `todo!()`，由主代理與 B 組定案 |
| A | `Font`（型別，A 組專用檔） | — | 加欄位 `measure: TextMeasurer` | 量寬由外殼注入（見「A 組：字型」） |
| F | `PlayerPosterSettings`（A 組 `player_poster_assets.rs`） | 0 欄位、`Copy` | `set`、`edge`、`fade_ms` 三欄與 serde，去掉 `Copy` | `fd2_settings.json` 的 `poster` 要能讀回並照原樣寫出；另補 `default_player_posters`、`valid_poster_mode`、`normalize_player_posters`（原標 `r8`，`loadSettings` 呼叫） |
| F | `BalanceSavedState`、`BalanceSidecar`、`Settings` | 沒有 serde | serde，欄位名與 `omitempty` 照 Go 標籤 | 附檔與設定檔的讀寫；`BalanceSidecar` 改手寫 `Default`（`version: 1`，Go 解碼前的起點） |
| C | `build_native_map_frame_input` | `(assets: &mut NativeMapAssets, field: Option<&MapData>, state: Option<&mut battle::State>, ..) -> Result<NativeFrameInput<'static>>` | `<'a>(assets: Option<&'a NativeMapAssets>, field: Option<&MapData>, state: Option<&'a battle::State>, ..) -> Result<NativeFrameInput<'a>>` | 120 定案：`fd2-game` 保存資源本身，每次呼叫重組借用的輸入；Go 三個指標都檢查 nil、都不改寫。D 組十幾個呼叫端照這個簽名傳 `self.native_map_assets.as_ref()`／`self.st.as_ref()` |
| C | `Game::native_map_selection_overlay` | `(state: Option<&mut battle::State>) -> Result<Option<Box<dyn FnMut(&mut Game, Vec<u8>, i64, i64) -> Result<()>>>>` | `(state: Option<&battle::State>) -> Result<Option<NativeMapSelectionOverlay>>` | `indexedmap::SelectionOverlay` 是借用的 `&dyn Fn(&mut [u8], i64, i64)`；Rust 回傳記下 Go 閉包捕捉值的 enum，合成時以 `render(&Game, &State, ..)` 組成借用的閉包 |
| C | `Game::native_map_action_overlay` | `(state: &mut battle::State) -> Option<Box<dyn FnMut(..)>>` | `(state: &battle::State) -> NativeMapSelectionOverlay` | 同上；Go 在建立時就失敗的閉包是 `NativeMapSelectionOverlay::Failed` |
| C | `Game::render_battle_panel_text` | `record: &mut [u8]` | `record: &[u8]` | 只讀；`&mut Vec<u8>` 的呼叫端照樣可編譯 |
| C | `Game::freeze_modern_hd_movement_panel` | `plan: Option<&mut NativePlayerMovement>, source: Option<&mut Picture>, panels: &mut [ModernBattlePanelText]` | `plan: Option<&NativePlayerMovement>, source: Option<&Arc<Picture>>, panels: &[ModernBattlePanelText]` | 只讀；Go 以指標存進 HD 狀態，Rust 存同一個 `Arc` 與複本 |
| C | `movement_panel_source_seal` | `h: Option<&mut ModernHDMovementPanel>` | `h: Option<&ModernHDMovementPanel>` | 只讀 |
| C | `clone_native_turn_staging_state` | `source: Option<&mut battle::State>` | `source: Option<&battle::State>` | 只讀；`State::clone` 已是深複製 |
| B | `Game::acting_actor` | `(&self, target: campaign::ActingUnit) -> Option<UnitIdx>` | `(&self, target: &campaign::ActingUnit) -> Option<HandlerUnitRef>` | Go 的 `*battle.Unit` 可能指向 `g.storyActors`，`UnitIdx` 只能指 `st.units`；新列舉 `HandlerUnitRef` 見「B 組：所有權」 |
| B | `Game::handler_unit_at` | `-> Option<UnitIdx>` | `-> Option<HandlerUnitRef>` | 同上 |
| B | `Game::begin_acting_frame` | `(&self, j: &mut ActPoseJob)` | `(&mut self, j: &mut ActPoseJob)` | Go 經單位指標改寫姿態 |
| B | `Game::finish_act_job` | `j: &ActPoseJob` | `j: &mut ActPoseJob` | 延續要 `take()` 出來 |
| B | `Game::resolve_campaign_dialog_line` | `upper_override: Option<&mut bool>, native_override: &mut campaign::NativeDialogueLayout` | `upper_override: Option<bool>, native_override: Option<&campaign::NativeDialogueLayout>` | Go 只讀兩者，`nativeOverride` 可為 nil |
| B | `Game::prepare_native_story_map_spawn_state` | `actors: &[battle::Unit]` | `actors: &mut [battle::Unit]` | Go 的 `view.storyActors = actors` 共用底層陣列，selector slot 寫回呼叫端 |
| B | `clone_story_unit_pointers`、`clone_battle_unit_pointers` | `(&[battle::Unit]) -> Vec<UnitIdx>`、`(&[UnitIdx]) -> Vec<UnitIdx>` | 都是 `(&[battle::Unit]) -> Vec<battle::Unit>` | Go 複製單位值；`UnitIdx` 指不到新陣列。C 組 `native_turn_staging`、D 組 `native_unit_present` 的呼叫端照新簽名 |
| B | `Game::build_native_indexed_transition_input_for_actors`、`…_for_state` | `(&mut self, …) -> Result<indexedmap::NativeTransitionFrameInput<'static>>`，`state: Option<&mut battle::State>` | `(&self, …) -> Result<NativeTransitionFrameSource>`，`state: Option<&battle::State>` | 借用的輸入不能回傳（`for_actors` 的載體是區域變數）；呼叫端以 `source.input(assets)` 組出借用輸入。D 組（補血、單位登場）照用 |
| B | `NativeIndexedTransitionJob::input`（B 組型別） | `indexedmap::NativeTransitionFrameInput<'static>` | `NativeTransitionFrameSource` | 同上 |
| B | `native_transition_input` | `-> NativeTransitionFrameInput<'static>` | `<'a>(&NativeFrameInput<'a>) -> NativeTransitionFrameInput<'a>` | 只是轉換欄位，借用照傳 |
| B | `Game::start_native_unit_death_present_mode` | `prelude: &mut dyn FnMut(&battle::State) -> Result<()>` | `prelude: &mut dyn FnMut(&mut battle::State) -> Result<()>` | Go 的 prelude 改寫候選狀態（清 +0x40） |
| B | `NativeCh28PostPresentJob::retained_state`（B 組型別） | `Option<battle::State>` | `bool` | Go 存 `g.st` 指標本身；Rust 保留模式就是不替換 `g.st` |
| B（D 的函式，跨組介面） | `Game::award_death_reward` | `killer: UnitIdx` | `killer: Option<UnitIdx>` | B 的 `completeTurn` 傳 nil killer（`main.go:13084`）；本體仍是 D 的 `todo!()` |
| D | `Game::begin_native_action_death_presentation` | `after: Option<Box<dyn FnOnce(&mut Game) + Send>>` | `after: &SharedAfter` | Go 的 `after` 同時被 `finishSuccessfulUnitAction` 的五個續行閉包捕捉；`FnOnce` 只能交出一次，改放共用格子（見「D 組：所有權」） |
| D | `capture_native_action_death_flags` | `(units: &[UnitIdx])` | `(st: &battle::State, units: &[Option<UnitIdx>])` | 要讀 HP 與 +5；Go 可傳 nil 單位 |
| D | `native_death_flag_prelude` | `-> Option<Box<dyn FnMut(&mut Game, &battle::State) -> Result<()> + Send>>` | `-> impl FnMut(&mut battle::State) -> Result<()>` | 直接交給 B 組 `start_native_unit_death_present_mode` 的 `prelude` |
| D | `Game::start_native_map_death_visual` | `flags: &mut BTreeMap<i64, u8>` | `flags: &BTreeMap<i64, u8>` | 只讀 |
| D | `Game::preflight_native_transient_death`、`native_transient_death_preview`、`native_transient_focus_preview` | `(&self, candidate: &mut battle::State, …) -> Result<Option<Game>>` | `(&mut self, candidate: &battle::State, …) -> Result<NativeTransientPreviewSaved>`（preflight 回 `Result<()>`） | Go 淺複製整個 `Game`；Rust 就地預覽再還原（見「D 組：預覽」） |
| D | `Game::grant_native_death_reward` | `killer: UnitIdx` | `killer: Option<UnitIdx>` | Go 可傳 nil（狀態致死、`deathProgramKiller` 為 nil） |
| D | `Game::queue_native_level_up_dialogue` | `ups: &mut [battle::LevelUpEvent]` | `ups: &[battle::LevelUpEvent]` | 只讀 |
| D | `expand_native_message_words`、`native_level_up_dialogue_lines`（另補 `native_message_pages`） | `strings: &fdtxt::Strings` | `strings: &dyn NativeMessageStrings` | Go 參數是匿名介面，測試傳假字串表 |
| D | `Game::render_physical_battle_panel`、`native_physical_base` | `record: &mut [u8]`（及 `actor_record`、`target_record`） | `&[u8]` | 只讀 |
| D | `bind_modern_physical_panel_text`、`bind_modern_battle_panel_text` | `(scene: Option<&mut NativePhysicalScene>, sources: &mut …)`、`panels: &mut [..]` | `(scene: Option<&NativePhysicalScene>, sources: Option<&mut ModernHDBattleSources>)`、`panels: &[..]` | 面板只讀；Go 的 `sources` 可為 nil |
| D | `modern_hd_battle_body_sources` | `(base: &mut [u8], …, present: NativePhysicalBodyPresent, …) -> Option<ModernHDBattleSources>` | `(base: &[u8], …, present: &NativePhysicalBodyPresent, …) -> ModernHDBattleSources` | 只讀；Go 一定回傳非 nil |
| D | `Game::prepare_native_physical_body_resources`、`prepare_native_physical_prelude` | `attack: &mut figani::Animation` | `attack: &figani::Animation` | 只讀 |
| D | `Game::preflight_native_physical_map_return` | `(&self)` | `(&mut self)` | Go `probe := *g` 試合成；Rust 就地合成再還原 |
| D | `NativePhysicalBodyResources::{main_sounds, counter_sounds}`、`NativePhysicalBodyJob::sounds`（D 組型別） | `BTreeMap<i64, Vec<u8>>` | `Arc<BTreeMap<i64, Vec<u8>>>` | Go 所有工作共用同一份 map |
| D | `NativePhysicalScene::prelude_images`（D 組型別）、`native_command24_prelude_images` | `Vec<Option<Arc<Picture>>>`、`-> Result<Vec<Option<Arc<Picture>>>>` | `Vec<IndexedFrame>`、`-> Result<Vec<IndexedFrame>>` | 120：演出工作保存索引色畫格，GPU 貼圖在 R5 |
| D | `NativeCommand29PresentationJob::actor_cues`（D 組型別） | `BTreeMap<i64, Vec<u8>>` | `Option<BTreeMap<i64, Vec<u8>>>` | Go 以 `actorCues == nil` 決定播 `sfxActor`（`native_command29_presentation.go:436`），空 map 分不出 nil |
| D | `Game::build_native_transient_phases`、`build_native_transient_phases_typed` | `-> Result<(Option<battle::State>, …)>` | `-> Result<(battle::State, …)>` | 成功時候選一定存在 |
| D | `Game::adopt_native_state_candidate` | `candidate: Option<&mut battle::State>` | `candidate: Option<battle::State>` | 換上候選狀態；對照表見「D 組：所有權」 |
| D | `Game::preflight_modern_hd_transient_expiry` | `candidate: &mut battle::State, …, damage: &mut [..]` | `&battle::State`、`&[..]` | 只讀 |
| E | `native_modifier_held`、`pressed_native_town_secret_scan` | `()` | `(input: &TickInput)` | Go 讀全域鍵盤；改讀本幀輸入。A 組呼叫端（`update_frame` 四處傳 `&input`、`player_poster_controls_allowed` 傳 `&self.tick_input`）只改呼叫那一行 |
| E | `current_native_hotel_input`（`r8`，隨 `campInput` 補移植） | `(enter: bool)` | `(input: &TickInput, enter: bool)` | 同上 |
| E | `Game::snapshot_native_town_source` | `(&mut self)` | `(&self)` | Go 不改寫 `g` |
| E | `NativeTownFrameSource::compose` | `(&mut self, fill: Option<&u8>, cursor, font: &mut fdtxt::Font)` | `(&self, fill: Option<u8>, cursor, font: Option<&fdtxt::Font>)` | Go 先複製 `scene` 再改背景；`*byte` 只讀；`font` 可為 nil |
| E | `Game::compose_native_town_frame`、`…_with_background`、`…_layers` | `(&mut self)`，`fill: &mut u8` | `(&self)`，`fill: Option<u8>` | 同上 |
| E | `Game::native_shop_state` | `-> (Option<campaign::NativeShopAssets>, …)` | `-> (Option<&campaign::NativeShopAssets>, …)` | Go 回傳指向 `nativeShopUI` 的指標；每幀合成不複製整份商店素材 |
| E | `Game::snapshot_native_preparation_transition` | `source: &mut [u8]` | `source: &[u8]` | 只讀、複製 |
| E | `Game::bind_modern_facility_service_menu` | `Option<&mut NativeClassUIJob>`、`Option<&mut NativeChurchUIJob>` | `Option<&NativeClassUIJob>`、`Option<&NativeChurchUIJob>` | 只讀 `frames`／`restore` |
| E | `NativeServiceMenuPresentation::compose` | `(&mut self)` | `(&self)` | 只讀接收者 |
| E | `Game::capture_modern_facility_confirmation_source`、`Game::freeze_modern_facility_list_writer` | `compose: Option<Box<dyn FnMut(&mut Game) -> (Vec<u8>, bool) + Send>>` | `compose: &mut dyn FnMut(&mut Game) -> (Vec<u8>, bool)` | Go 的 `compose` 是讀 `g` 的方法值，只在本次呼叫內重複呼叫、不存起來 |
| E | `capture_modern_facility_list_text`、`Game::compose_modern_shop_list_marker`、`Game::compose_modern_facility_marker` | `compose: &mut dyn FnMut() -> (Vec<u8>, bool)` | `compose: &mut dyn FnMut(&mut Game) -> (Vec<u8>, bool)` | `compose` 要讀 `Game`，而這些函式在呼叫期間換掉 `Game` 的素材欄位 |
| 階段 3 | `Game::growth_roll_observer`（欄位） | `Option<Box<dyn FnMut(&mut Game, UnitIdx, u16) -> u16 + Send>>` | `Option<battle::Callback<dyn Fn(&battle::Unit, u16) -> u16 + Send + Sync>>` | Go 把同一個 func 交給 `st.NativeGrowthRollObserver`（`main.go:4088`）；見「R4 階段 3 整合紀錄」 |

## R4 階段 2：A 組（主迴圈、啟動、標題與開場）

2026-11-14，代號 `r4a`，合併基準 `451620de`。範圍是 `PORTING.json` 裡 `group` 為 `A` 的 156 個函式（含 `main.go` 的 69 個），
另加 Go `Layout`（`r8`，只回傳 640×400，照移）。

### 移植結果

| 項目 | 數量 |
|---|---|
| A 組函式 | 156 個全部有本體；另隨呼叫端補移植 7 個 `r8` 小函式（下表） |
| 整個本體是 `todo!()` 的 | 3 個：`draw_battle_scene`（R5）、`load_hd_regional_fonts`（R6）、`draw_modern_battle_panel_ink`（R6） |
| 部分是 `todo!()` 的 | `draw`（重設本幀旗標之後是 R5）、`record_input_audit_state`（環境變數閘門之後是 fd2-tools）、`update`（截圖模式設定是 fd2-tools；第一章沒有的分支是 R8）、`load_game`（HD 主題 R6；截圖與結局預覽的環境變數分支 R8）、`enter_node`（物品閘門／合成／原生結局節點 R8）、`title_update`／`apply_title_menu_event`（LOAD 讀檔槽、CONTINUE、戰敗後重新開始 R8）、`modern_hd_*` 與 `poll_modern_theme_load`（HD 開啟後的分支 R6）、`face_for`／`width`（沒有注入量測器時 R5）、`bind_command_learn`（設了測試觀察者時，共用變更） |
| `todo!` 分類 | R5 5、R6 14、R8 35、fd2-tools 2、共用變更 1 |

R8 的 `todo!` 都在第一章重播沒有執行到的分支上（呼叫的函式在 `PORTING.json` 標 `r8`）：F2～F12 的玩家設定鍵、F5／F9 快速存讀、
敗北、結局、第二十章以後的章節演出、物品閘門與合成節點、ESC 輪選單位等。

隨呼叫端補移植的 `r8` 函式（A 組函式需要它們，本體照 Go）：

| Go | Rust | 呼叫端 |
|---|---|---|
| `audio.go:musicRenderPath` | `Game::music_render_path`（私有） | `play_bgm_count` |
| `audio.go:playPCMVoice` | `Game::play_pcm_voice`（私有） | `play_raw`、`play_sfx` |
| `title.go:titleLogoTransitionIndex` | `title_logo_transition_index` | `try_skip_title_cut_step` |
| `run_log.go:runLogNode` | `Game::run_log_node`（私有） | `trace_run_log` |
| `main.go:defaultChapterStoryScript` | `enter_node` 內的區域函式 | `enter_node` |
| `main.go:(*atkAnim).beginCounterStage` | `AtkAnim::begin_counter_stage`（私有） | `step_attack_presentation_tick` |
| `main.go:Layout` | `Game::layout` | Ebiten 主迴圈（外殼） |

Rust 新增、沒有 Go 對應的工具（都在 A 組的檔，其他組可用）：`assets.rs` 的 `join_path`、`go_int_slice`、`color_palette_from_nrgba`、
`nrgba_palette_from_color`、`picture_from_indexed`、`picture_from_paletted`、`indexed_frame`、`crop_picture`、`decode_png_picture`；
`modern_hd_title_interface.rs` 的 `sha256_hex`；`font.rs` 的 `rect_dx`／`rect_dy`／`rect_empty`；`main.rs` 的 `ERR_TERMINATION`、
`Game::update_frame`、`center_camera_on_cursor`、`shot_termination`、`tile_aligned_story_map`、`materialize_story_native_map_view`、
`current_node_type_is`、`load_numbered_frames`、`scan_numbered_frame`。`main.rs` 新增的私有函式都緊鄰 A 組的函式，不動其他組的位置。

### 虛擬時鐘

Go 主迴圈的時間有兩層：Ebiten 以 `SetTPS(60 × speed)` 決定每秒呼叫幾次 `Update`（`main.go:13475`、`modern_hd_presentation.go:83`），
`gameNow()` 再把牆鐘差乘 `speed`（`player_game_clock.go:17`）。穩態下每次 `Update` 的遊戲時間恰好前進 1/60 秒，速度只改變每秒跑幾次
`Update`。Rust 照這個關係，牆鐘換成 `Game::virtual_now`：

| 項目 | 規則 |
|---|---|
| 誰推進 | 驅動端持有 `VirtualFrameClock`（`player_game_clock.rs`），每個邏輯幀（每次 `update` 之前）呼叫一次 `Game::advance_virtual_frame` |
| 每幀前進多少 | 虛擬牆鐘前進 `1e9 / (60 × speed)` 奈秒；除不盡的部分以 `1/(60 × speed)` 奈秒為單位累積，速度固定時每 `60 × speed` 幀牆鐘恰為 1 秒 |
| 速度從哪裡來 | `normalize_player_speed(player_clock.speed)`，即 Go `syncPlayerClockSpeedAt` 寫入、`SetTPS` 讀的同一個值；越界照 Go 視為 1 |
| 速度倍率怎麼套 | 照 Go 由 `PlayerGameClock::sample` 把牆鐘差乘 speed，所以每幀遊戲時間是 1/60 秒（逐幀誤差小於 speed 奈秒） |
| 起點 | `VIRTUAL_CLOCK_ORIGIN` = 1 秒。Go 的 `time.Now()` 不會是零值，而 `playerGameClock.sample`、各演出工作的 `started.IsZero()` 都把零值當「尚未開始」 |
| 讀取 | `game_now()` 讀 `virtual_now`；BIOS 時鐘（`NativeBIOSClock`）與所有收 `now` 參數的函式都從 `game_now()` 取值 |
| 截圖決定性時鐘 | Go `FD2_SHOT_DETERMINISTIC=1` 的 `time.Unix(0, frame × 16666666)` 改成 `VIRTUAL_CLOCK_ORIGIN + frame × 16666666`，相鄰幀的差相同 |

速度改變時，未滿 1 奈秒的累積餘數沿用新分母，誤差小於 1 奈秒。單元測試在 `player_game_clock/player_game_clock_test.rs`：起點非零、
速度 1 時 60 幀恰為 1 秒、速度 1～4 每幀遊戲時間 1/60 秒且 `60 × speed` 幀恰為 `speed` 秒、越界速度、60 幀驅動 BIOS 時鐘跨 18 tick。

### 輸入

- `update`、`title_update` 等只讀 `self.tick_input`。`update` 開頭複製一份本幀輸入，整個函式用同一份（Go 每次查 `inpututil` 也是同一幀的值）。
- `inpututil.AppendJustPressedKeys(nil)` 非空 → `TickInput::any_just_pressed`；`inpututil.KeyPressDuration` → `TickInput::key_press_duration`
  （共用變更：`TickInput::press_duration`）。`key_repeat` 改收 `&TickInput`。
- Go `inpututil.AppendJustPressedTouchIDs`（點格移動游標，`main.go:9087`）沒有移植：`TickInput` 沒有觸控，座標換算 `playerTouchCoordinates`
  是 `r8`。列為共用變更需求，R5 接上。
- Alt+Enter 的 `ebiten.SetFullscreen` 改設 `Game::fullscreen_toggle_requested`（共用變更），外殼讀取後清除。
- `ebiten.Termination` 對應 `ERR_TERMINATION`；`update` 回傳 `Err(e)` 且 `e.is(&ERR_TERMINATION)` 表示正常結束。

### `Update` 的 `defer`

Go `Update` 開頭排了四個 `defer`：`syncBalanceState`、清除玩家面板請求旗標、`recordInputAuditState`、`traceRunLog`，在每條 return 路徑上依
後進先出執行。Rust 把本體放在私有的 `update_frame`，`update` 在它回來後依 `traceRunLog` → `recordInputAuditState` → 清旗標 →
`syncBalanceState` 的順序呼叫（Go 的 panic 路徑不移植）。

### Draw 中改狀態的位置

階段 3 要定義「呈現請求與回報」介面。下列是 Go `Draw`（`main.go:9355`..`10188`）與 A 組繪圖函式裡改寫流程狀態的地方。Rust 的 `draw` 只做
第 1、3 項，其餘停在 `todo!("R5: …")`；表中屬於別組的旗標由該組的 `draw*` 函式設定。R5 階段 1 起 `draw` 是完整的分派骨架，
見「R5 階段 1」的「`Game::draw` 分派骨架」。

| # | Go 位置 | 改寫 | 階段 3 的對應 |
|---|---|---|---|
| 1 | `main.go:9356`..`9368` | 每幀開頭清 `nativeMapPresented`、`movementPanelDraw`、`nativeInterfaceHUDVisible`、`modernHDOverlay`、`modernHDScene`、三個 HD 名冊、`modernHDPanelPortraits`、`modernHDBattleVisible`、`modernHDWorldVisible`、`modernHDTimedDialogueVisible`；呼叫 `syncModernHDBattleOwner` | Rust `draw` 已移植；無畫面驅動端每幀也要做（或改成呈現回報的開頭） |
| 2 | `main.go:9369`..`9371` | HD：`modernHDMapGPU.BannerName = ""` | R6 |
| 3 | `main.go:9372` | `beginModernHDText`：清 `modernHDText`，HD 介面時把字型的 `hd` 指向本幀文字框 | 忠實模式只清欄位（已移植） |
| 4 | `main.go:9386`..`9389`（defer） | `nativeTurnStaging.drawn = true`（非 indexed 的 flash 階段） | `stepNativeTurnStaging` 等這個旗標才前進，呈現回報要設 |
| 5 | `main.go:9391`（defer） | 說明畫面時 `clearModernHDText` | HD 文字層，R6 |
| 6 | `main.go:9409` 等約 30 處 | `captureShot`：截圖模式存 PNG 並設 `shotTaken`，下一次 `Update` 回傳 `Termination` | 截圖模式（fd2-tools） |
| 7 | `main.go:9420`..`9706` | 各原生演出的 `draw*` 回傳 false 時呼叫 `failNative…Presentation`（ch20 天空之鑰、ch23、2189A、單位登場、ch28、24B4D、AI 指令修飾、指令 0～35、AI 物品） | 呈現失敗要回報給流程，否則演出不收尾 |
| 8 | 各 `drawNative…Presentation`（B／C／D 組） | 演出工作的 `drawn`／`presented` 旗標（指令環、指令 0～35、補血、修飾、物理攻擊前奏 `preludeDrawn`、尾段 `leadDrawn`、24B4D 的 `transitionReveal.drawn`、spawn intro、索引轉場、調色盤漸變等） | 對應的 `step*` 等旗標才前進；這是 120「畫過才推進」的主體 |
| 9 | `main.go:9768` | END 回收畫不出來時 `cancelNativeEndTurnRecovery` | 同 7 |
| 10 | `main.go:9780`..`9783` | AI 閒置回收畫不出來時 `restoreNativeAIIdleRecoveryRange`、清 `nativeAIIdleRecovery`、寫 `loadErr`、`aiBusy = false` | 同 7 |
| 11 | `main.go:9817` | 建立 `storyView` 離屏畫布（快取） | 畫面資源，R5 |
| 12 | `main.go:9958` → `drawNativeMapFrame`（`main.go:12451`，C 組） | `composeNativeMapFrame`：取 `gameNow`、推進原生地圖 BIOS 時鐘與地形／單位相位、合成整幀並發布 `nativeMapVGA`；設 `nativeInterfaceHUDVisible`、`nativeMapPresented = true`；首次顯示回合橫幅時複製 `bannerFrame` | **影響流程計時**：原生地圖時鐘只在合成時推進（`advance_native_map_clock` 只從合成交易呼叫）。無畫面驅動端每幀要照 Go `Draw` 的條件執行合成 |
| 13 | `main.go:10026`..`10027` | HD 對話框外框準備失敗寫 `loadErr` | R6 |
| 14 | `main.go:10046` | 對話框漸層 `dlgGrad` 快取 | 畫面資源，R5 |
| 15 | `main.go:10089` | HD：`modernHDOverlay` | R6 |
| 16 | `main.go:10159` | 指令環已在原生整幀中時 `markActionOverlayDrawn`（否則在 `drawRing` 內設） | 指令環動畫等這個旗標前進 |
| 17 | `main.go:11280`（`drawBattleScene`） | 物理攻擊面板畫不出來時寫 `loadErr` | 同 7 |

### 音訊

- `AudioVoice`（共用變更：加 `encoded`、`looping`）記錄播放請求：`source`（BGM 是檔案路徑，音效是 `"pcm"`）、`encoded`（編碼後位元組）、
  `looping`（BGM loop count 0）、`playing`（建立時為 true，外殼播完設 false）、`closed`。
- 音效庫與 `sfx*` 欄位保存**編碼後**的 OGG／WAV 位元組；Go 在載入時解碼成 PCM，Rust 的解碼在 R5（`fd2-app`）。
- `title_ani1_voice`、`native_physical_voice` 改成 `Option<SfxVoice>`（共用變更，Go 介面值可為 nil）。
- `step_sfx_voices` 丟棄 `!is_playing()` 的 voice；無畫面驅動端要回報播完，否則佇列會一直累積。

### 字型

`fd2-game` 沒有 TTF 解析器（不加依賴）。`Font` 新增 `measure: TextMeasurer`，外殼注入 `TextMeasure`（`ascent`、`bound_string`、`has_glyph`，
對應 Go 的 opentype face、`text.BoundString`、`sfnt.GlyphIndex`）。沒有字型檔時 `width` 照 Go 回 0；有字型卻沒注入量測器時 `face_for`／`width`
停在 `todo!("R5: …")`，不以猜測的寬度斷行。`load_font_sized` 只檢查 TTC／sfnt 檔頭；依家族名稱選區域字型（`hdFonts`）只給 HD 用，在 R6。
`rune_breakable` 的 Han／Hiragana／Katakana／Hangul 判斷放在 `src/go_unicode.rs`，是教訓 `rust-port-no-go-stdlib` 的第四項具名例外
（2026-11-15 使用者同意）：表格照 Go 1.22.12 `unicode.Scripts`（Unicode 15.0.0），`is`／`is_in` 對應 `unicode.Is`／`In`。2026-11-15 以
`fd2-go-test-local` 容器列印 Go 的四張表，與 Rust 表逐列比對 54 列相同。

### A 組：跨組介面

`stepBattleWalk`（A）與 `stepBattleWalkSegment`（B）：Go 的 `w` 與 `g.walk` 是同一個指標，segment 內 `nativeCh12UpWalkFrame` 讀 `g.walk`、
`beginNativeSystemGroupMarchFieldEvent` 比對 `g.walk != walk`、錯誤路徑寫 `g.walk = nil`，`finish` 閉包要 `&mut Game`。骨架的
`(w: &mut WalkAnim, finish: &mut dyn FnMut(i64))` 無法同時滿足。A 組在本工作樹把 B 的骨架簽名改成
`(&mut self, finish: &mut dyn FnMut(&mut Game, i64)) -> bool`：walk 留在 `self.walk`，segment 以 `self.walk.as_mut()` 存取，在 Go 呼叫
`finish(pose)` 的地方呼叫 `finish(self, pose)`；`finish` 開頭把 walk 取出（呼叫當下 `g.walk == w`）。整合時與 B 組的實作對齊。

`OFFICIAL_LOCALE_CONTRACT`（A）：Go `native_system_info_locale.go:14` 的 `init()` 在套件初始化時加入 68 個 `battle.info.*` 鍵，骨架沒有產生
`init`。A 組在這個變數的初始式呼叫 `system_info_locale_contract()`（C 組，`todo!()`），C 組合併前載入官方語系會停在那裡。

### 共用變更需求

本工作樹已以最小改動先做（主代理整合時決定採用）：

| 項目 | 原因 | 寫法 |
|---|---|---|
| `TickInput::press_duration` 與 `key_press_duration`、`any_just_pressed` | `keyRepeat` 讀 `KeyPressDuration`；多處「任意鍵」 | `support.rs`，`BTreeMap<Key, i64>` |
| `AudioVoice::encoded`、`AudioVoice::looping` | 外殼要知道播什麼、BGM 是否循環 | `support.rs` |
| `Game::title_ani1_voice`、`Game::native_physical_voice` 改 `Option<SfxVoice>` | Go 介面值可為 nil | `main.rs` 欄位型別 |
| `Game::fullscreen_toggle_requested`（新欄位） | `SetFullscreen` 是外殼的事 | `main.rs` |
| `MapData` derive `Deserialize` | `loadMap` 讀 `map.json` | `main.rs`，`[]byte` 欄位用 `battle::go_bytes` |
| `step_battle_walk_segment` 簽名（B 的函式） | 見上節 | `main.rs` |

未做、需要主代理決定：

| 項目 | 原因 | 建議 |
|---|---|---|
| `TickInput` 加觸控（本幀點擊的邏輯座標） | Go 點格移動游標 | `touches: Vec<(f64, f64)>`，外殼先做 `playerTouchCoordinates` 的換算 |
| `Game::growth_roll_observer` 改成 `State::native_growth_roll_observer` 的型別 | Go 把同一個 func 交給 `battle.State`；Rust 兩邊型別不同，`bind_command_learn` 有設定時停在 `todo!` | `Option<battle::Callback<dyn Fn(&battle::Unit, u16) -> u16 + Send + Sync>>` |
| TTF 量測 | 斷行與名稱寬度影響對話分頁 | 由 `fd2-app` 注入 `TextMeasure`；無畫面重播若要與 Go 同寬，需要與 `golang.org/x/image/font/opentype` 同結果的量測 |
| `fd2_campaign::go_filepath` 公開 | 路徑 Join 的詞法 Clean | 目前 `pub(crate)`；A 組改用 `std::path`（只略過空段），呼叫端只傳固定相對路徑 |

### KNOWN_DIFFERENCES

| 名稱 | 差異 | 影響 |
|---|---|---|
| `load-game-seed-no-wall-clock` | 沒有 `FD2_SEED` 時 Go 用 `time.Now().UnixNano()`，Rust 用 `VIRTUAL_CLOCK_ORIGIN` 的奈秒值 | 不讀主機時間；要不同種子的外殼設 `FD2_SEED` 或替換 `rng` |
| `audio-decode-deferred` | Go 在載入時把 OGG／WAV 解成 PCM，解不開就拒收；Rust 只檢查編碼位元組非空 | 損壞音檔改在 R5 播放時才發現 |
| `run-log-no-wall-timestamp` | Go `log.Printf` 帶主機時間前綴，Rust 不寫 | 只影響 run log 文字 |
| `hd-title-seal-bytes` | HD 標題與讀檔槽的 `seal` 是 Rust 自訂位元組序列的 SHA-256，不是 Go `json.Marshal` 的 | `seal` 只和自己比對 |
| `locale-contract-map-order`、`figani-delays-map-order`、`party-roster-map-order` | Go `range` map 隨機，Rust 依鍵排序 | 只在同時有多個錯誤或名冊 identity 重複時，回報的項目不同 |
| `figani-meta-pair-length` | Go `[2]int` 接受長度不是 2 的 JSON 陣列，Rust 整份視為解碼失敗 | 正式匯出檔都是長度 2 |
| `go-stdlib-error-text` | 讀檔、JSON、PNG 的錯誤文字是 Rust 的（教訓 `rust-port-no-go-stdlib`） | 接受與拒收相同 |
| `story-walk-callback-view` | `stepStoryWalks` 迴圈中延續看到的 `story_walks` 是空的（Go 看到原長度）；延續推入的工作照 Go 在迴圈結尾被丟掉（`main.go:947`） | 目前沒有延續讀 `storyWalks` |
| `asset-glob-subset` | `filepath.Glob` 只實作「最後一段含 `*`」 | 三個呼叫端的樣式都是這種 |
| `map-json-key-case` | Go `json.Unmarshal` 的鍵比對不分大小寫，Rust 只認標籤上的寫法 | 正式匯出檔與標籤相同 |

`NORMALIZATIONS`：無（A 組沒有黃金差分）。

### A 組測試

`tools/rust_game_test.sh`（`FD2_REQUIRE_ORIGINAL=1`，輸出沒有「略過」）：72 個測試，63 通過、9 個 `ignore`。

| ignore | 等 | 原因 |
|---|---|---|
| `official_locales_satisfy_contract`、`locale_message_reports_missing_catalog_and_bad_keys` | C | `OFFICIAL_LOCALE_CONTRACT` 初始式呼叫 `system_info_locale_contract` |
| `test_play_raw_retains_real_audio_player`、`test_fm_and_mt32_tracks_decode_and_switch_at_runtime`、`test_unknown_catalog_track_does_not_mutate_current_bgm` | C | `current_native_system_options` |
| `test_separated_title_fdother_matches_fixed_archive` | E | HD 選單凍結呼叫 `modern_hd_native_text_color` |
| `test_game_materializes_editable_native_map_runtime` | B | `sync_native_map_view` |
| `test_native_nested_system_exit_publishes_ebiten_termination`、`update_runs_deferred_cleanup_on_early_return` | E | `update` 開頭的 `native_modifier_held` |

移植範圍：測試檔名對應 A 組 Go 檔的測試（`player_game_clock`、`native_bios_clock`、`player_battle_speed`、`player_hd_quality`、`title_menu_state`、
`audio_voice`、`separated_command_sound`、`separated_ui_assets`、`text_wrap_segments`、`font`、`font_scale`、`modern_hd_title_interface`），以及
以 A 組 `main.go` 函式為主題的測試（`atk_animation_test.go`、`beatrunner_test.go` 的隊伍同步／排序／篩選、`persistent_native_command_test.go`、
`native_map_presentation_test.go`、`native_continue_handoff_test.go` 的 Termination）。`Update`、`loadGame`、`enterNode`、`resetBattle` 被其他主題的
Go 測試當驅動用（以函式名掃描，共 460 個 Go 測試直接引用 A 組函式），那些測試的主題屬於別組，由主題組或階段 3 移植。

未移植的 A 組檔案測試：

| 類別 | 測試 | 原因 |
|---|---|---|
| R5（畫面或真字型量寬） | `TestFontUserScaleChangesMeasuredWidth`（`font_scale_test.go:99`）、`TestLocalizedDialogueWrapKeepsEnglishWordsWhole`（`text_wrap_segments_test.go:71`）、`TestLocaleDialogueCorpusNeverSplitsWords`（`text_wrap_locale_corpus_test.go:19`）、`TestModernHDTitleInterfaceRuntimeGPU`（`modern_hd_title_interface_gpu_test.go:450`）、`TestModernHDTitleInterfacePrototype`（`modern_hd_title_interface_prototype_test.go:88`）、`TestHDTextureQualityBattleGeometryGPU`（`player_hd_quality_gpu_test.go:108`）、`TestHDTextureQualityKeepsSourceAndAlpha`（`player_hd_quality_test.go:12`）、`TestTitleScrollPalettePulseMatchesNativeRowWindows`、`TestTitlePublisherScheduleAndBrightness`（`title_menu_state_test.go:110`、`129`，繪圖用的 `r8` 函式）、`TestOriginalInterfaceKeepsHDPanelPortrait`、`TestInterfaceBattlePanelSelectionKeepsNativePixels`、`TestInterfaceSystemInfoWithOriginalArtwork`、`TestInterfaceStatusWithOriginalArtwork`（`player_interface_test.go:49`、`89`、`128`、`151`） | 量寬由外殼注入；或斷言畫面像素 |
| R6（HD 切換） | `TestIndependentInterfaceFourModesKeepArtworkAndBattle`、`TestInterfaceRejectsBeforeChangingModeOrArtwork`、`TestInterfaceSettingsSurviveLocaleAndAudio`（`player_interface_test.go:12`、`36`、`62`）、`TestFirstThemeSwitchLoadsInBackgroundAndAppliesOnce`、`TestBackgroundThemeLoadFailureLeavesFaithfulMode`、`TestFirstInterfaceSwitchDefersWhileThemeLoads`（`player_theme_loader_test.go:38`、`74`、`89`）、`TestModernHDTitleInterfaceSources`、`TestModernHDTitleLoadSlotsChromeSource` 與 `TestModernHDTitleInterfaceRejectsSource` 的前半（`modern_hd_title_interface_test.go:25`、`173`、`98`）、`TestHDQualityRasterCoordinatesAndSceneryCache`、`TestHDTextureQualitySettingsAndToolbar`（`player_hd_quality_test.go:45`、`80`） | 經 `setInterfaceMode`／`cycleTheme`／HD 繪製 |
| 改寫環境變數 | `separated_ui_assets_test.go` 的 3 個（`:133`、`:184`、`:193`）、`separated_command_sound_test.go:51`、`:79`、`assets_test.go` 的 3 個（`:39`、`:52`、`:71`）、`packaged_startup_test.go` 的 3 個（`:9`、`:38`、`:83`）、`title_menu_state_test.go:339`、`:387`、`:398`、`player_game_clock_test.go` `TestPlayerSpeedIsSeparateFromCampaign`、`player_battle_speed_test.go:32`、`input_audit_trace_test.go:12` | `t.Setenv`：Rust 2024 的 `std::env::set_var` 是 `unsafe`，工作區 `forbid(unsafe_code)`。對真實素材包的對應測試已另寫 |
| `r8` 函式 | `player_battle_speed_test.go` 的 `:9`、`:57`、`:94`、`:101`、`font_scale_test.go:10`、`:64`、`title_menu_state_test.go:165`、`:194`（`decode…`；雜湊已併入 `test_separated_title_publisher_matches_fixed_archive`）、`:349`、`:453`、`audio_voice_test.go:59` | 受測函式第一章沒有執行到，骨架沒有產生 |
| 別組主題 | `audio_voice_test.go` 的物理演出 7 個（`:129`..`:283`，D 組）、`TestPlayTitleStartThroughChapterOneToTown`（`native_title_to_town_test.go:689`，整合） | 主題屬於別組或階段 3 的重播 |
| 逐句譯文核對 | `story_locale_test.go` 13 個（`TestReviewed…`、`TestAllRuntimeStoriesMatchCanonicalLineIdentities` 等） | 測的是受版控的譯文資料；`load_story_script_with_identity_at` 與 `localized_story_text` 的行為已有對應測試。逐句清單可交由較低成本的子代理照抄 |

黃金差分：A 組沒有加。`title_update` 的開場排程、`step_attack_presentation_tick` 等收 `now` 的工作要等 B～E 合併後才跑得完整，適合在
階段 3 以重播檢查點比對。

### fd2-rules／fd2-campaign 缺口

沒有改那兩個 crate。`battle::Unit`／`battle::State` 有私有欄位，結構字面值不能用 `..Default::default()`，改成先 `default()` 再逐欄寫入。
`fd2_campaign::go_filepath` 是 `pub(crate)`，見共用變更需求。

## R4 階段 2 F 組：存讀檔與設定（2026-11-14）

- `PORTING.json` 的 F 組 28 個函式全部移植，沒有剩下 F 組的 `todo!()`。同檔另外移植原標 `r8` 的
  `savePath`、`validateSavePartyTopology`、`saveGame`、`loadGame`、`loadGameFromSlot`、`loadJSONGameFromSlot`、
  `saveSettings`、`nativeHotelNodeForTown`、`restoreBalanceSavedState`：120 的 R4 完成條件要能讀 Go 版存檔，
  `confirmTitleLoadSlot` 的 JSON 分支與 Go 測試都經過它們。`PORTING.json` 的狀態沒有改（不在 F 組可改的檔），
  由主代理整合時更新。
- 新型別：`SaveData`、`BattleSnapshot`（Go `battle_snapshot.go:20`；該檔沒有子模組，型別先放 `save.rs`）。
  JSON 輔助：`go_json_marshal_indent`（Go `json.MarshalIndent(v, "", " ")`，serde_json `PrettyFormatter` 縮排一格）、
  `go_json_unmarshal_zero`（零值起點的 `json.Unmarshal`：頂層 `null` 得零值、頂層陣列拒收）、
  `go_json_int_key_map`（整數鍵 map 依十進位字串排序寫出）、`go_json_balance_options`（`BalanceOptions` 以 Go 的鍵順序寫出）、
  `settings.rs` 的 `go_json_float64`（整數值的 `float64` 寫成 `1` 而不是 `1.0`）與 `go_json_unmarshal_settings`
  （`loadSettings` 起點上的解碼：沒寫或寫 `null` 的欄位保留起點，重複鍵取最後一個）。都是 serde 的欄位層寫法，
  沒有移植 `encoding/json`。
- 停在 `todo!()` 的地方：
  - 別組：A（`user_data_path`、`asset_path`、`separated_asset_path`、`locale_message`、`enter_node`、
    `freeze_modern_hd_title_load_slots`、`load_native_battle_palette`、`normalize_player_speed` 等正規化、
    `Font::set_user_scale`）、C（`current_native_system_options`、`clear_native_player_movement`、
    `clear_native_item_panel`）、E（`reset_native_shop_ui_pulse`、`reset_native_church_ui_pulse`、
    `reset_native_class_ui_pulse`）。
  - R8：`saveGameToSlotFrom` 的戰鬥中分支（`captureBattleSnapshot`）、`loadJSONGameFromSlot` 讀到
    `battle_snapshot` 時的 `validateBattleSnapshotNode`／`applyBattleSnapshot`。三個函式在 `battle_snapshot.go`，
    第一章受控流程沒有執行到，骨架沒有子模組。存檔的 `battle_snapshot` 已能讀回、照原樣寫出（黃金涵蓋）。
- Go 的 nil 與空：`saveData.items`、`flags` 沒有 `omitempty`，`SaveData` 用 `Option` 區分。從 `Game` 存檔時
  `items` 空的寫 `null`（Go 版 `items` 只由讀檔設定，平常是 nil），`flags` 寫 `{}`（Go `NewRunner` 建立非 nil map）。
- map 迭代順序：`validateSavePartyTopology`、`readBalanceSidecar` 與 `BalanceOptions.Validate` 在多筆違規時 Go 報哪一筆
  隨 map 順序而定，Rust 依鍵升冪。黃金在 Go 端重跑 300 次收集所有可能的錯誤，Rust 的結果必須是其中之一。
- `restoreNativeChapterSlot`：Go 讓 `g.partyRoster` 等欄位與 `nativeChapterRestore` 共用同一份 map；
  `nativeChapterRestore` 之後只當旗標讀（`main.go:3044`、`3188`、`3930`），Rust 存複本。

### 測試

- 移植 Go 測試 21 個：`save_test.go` 12、`settings_test.go` 2、`font_scale_test.go` 1、`balance_save_test.go` 2、
  `native_load_slots_ui_test.go` 4；另有 F 組補充測試 5 個（標「F 組補充」）與黃金差分 2 個，共 28 個。
  `tools/rust_game_test.sh` 在 `FD2_REQUIRE_ORIGINAL=1` 下 11 通過、17 ignore、沒有略過。ignore：等 A 10 個、
  等 A、C 4 個、等 A～E 1 個（`campaign_save_load_restores_town_boundary_and_party`）、黃金差分 2 個（由
  `tools/rust_golden_save.sh` 以 `--ignored` 執行，其中 `golden_save_settings_loaded_matches_go` 另等 A）。
- 改環境變數的 Go 測試（`t.Setenv("XDG_DATA_HOME"|"FD2_NATIVE_SAVE")`）：工作區禁止 `unsafe`，`set_var` 不能用，
  改由 `save/f_test_env.rs` 的 `f_child_env` 在子行程重跑該測試並傳入環境變數。
- 不在這裡移植的 Go 測試（都不需要畫面，所以不列 R5）：
  - `native_current_save_test.go` 的 3 個測試先呼叫 `buildNativeCurrentSaveStored`（`r8`），R8 移植。
  - `settings_test.go:11` 測 A 組語系目錄；`balance_save_test.go` 前兩個測 A 組 `player_balance.go`；
    `font_scale_test.go` 其餘 3 個測 A 組 `Font`、R8 的 `cycleFontScale` 與真字型量寬。
  - `native_load_slots_ui_test.go:116` 測截圖模式的 `parseNativeLoadSlotShotState`（`r8`）。
  - `native_chapter_slot_restore_test.go`、`native_continue_handoff_test.go` 與各章流程測試也經過 F 組函式，
    但測的是整段讀檔流程（`enterNode`、CONTINUE、整備），屬於呼叫端各組與階段 3 的整合。
- R5 測試清單：F 組沒有需要畫面的測試（`drawNativeLoadSlots` 是 `r8`，不在骨架）。

### 黃金差分

`tools/rust_golden_save.sh`：Go 端 `remake/cmd/fd2/rust_golden_save_test.go`（`TestRustGoldenSave`，只在
`FD2_RUST_GOLDEN_SAVE_OUT` 設定時執行、只輸出資料）連跑兩次、SHA-256 必須相同；Rust 端
`src/game/save/golden_save_test.rs`。`fd2-game` 的函式與型別是 `pub(crate)`，`tests/` 的整合測試看不到，所以
黃金比對放在 crate 內的單元測試，以 `#[ignore]` 擋在一般 `cargo test --lib` 之外。

| 項目 | 內容 |
|---|---|
| 黃金檔 | `work/rust-build/golden/fd2-game-save-r4f-v1.json`，349 個案例，SHA-256 `dab44975b1397534f88679fee5fdc490f1558f9725d9d7ee9138028ac1169faa`（go1.22.12，Go 端連跑兩次相同） |
| 存檔 196 | Go 寫出的固定與固定種子隨機 `saveData` 164 份（含 JOIN 建構器建立的真實持續隊伍單位）、手寫讀取端輸入 23 份、第一章戰鬥中 F5 存檔 2 份（附戰場快照）、真實 Go 版存檔 7 份 |
| 設定 80 | Go 寫出的 64 份、手寫 14 份、真實設定檔 2 份；比零值起點與 `loadSettings` 起點的解碼及重新寫出的位元組 |
| 附檔 45、附檔鍵 28 | `readBalanceSidecar` 的接受／拒收與錯誤文字、讀回與重新寫出；`balanceSaveKey` 的鍵與錯誤 |
| 存檔位元組 | 191 份接受的存檔由 Rust 重新寫出全部與 Go 逐位元組相同（#319 修正後；之前 127 份只在 battle 套件子樹不同，見下方「fd2-rules 缺口」）。設定與附檔逐位元組相同 |
| 竄改反向檢查 | 測試內對九類比對各改一個欄位（讀回金幣、名冊 HP、外層位元組、名冊單位的浮點寫法 `0`→`0.0`、nil 切片 `null`→`[]`、設定位元組、設定讀回、附檔錯誤文字、附檔鍵），全部被抓到 |

真實輸入（主工作區 `work/`，唯讀；大小與 SHA-256 寫進黃金檔）：

| 檔 | 大小 | SHA-256 |
|---|---|---|
| `ch01-flow-20261017/remake-full-r10/hotel-save.json` | 14825 | `190b9f8f614c69a4…` |
| `ch02-flow-20261017/remake-full-r4/hotel-save.json` | 17755 | `69d122a76091cb21…` |
| `ch03-flow-20261017/remake-full-r2/hotel-save.json` | 20475 | `5e4388d33faf7912…` |
| `hd1-feedback-20261104-r83/special-sources-r4/command30/fd2_save.json` | 88129 | `1363b356a9c411e5…` |
| `hd1-performance-60fps-20261105-r89/candidate-gui/user-data/fd2_re/fd2_save.json` | 12982 | `089b077b8d334bab…` |
| `hd1-performance-60fps-20261105-r89/candidate-gui/user-data/fd2_re/fd2_settings.json` | 146 | `8670d2eae96de202…` |
| `hd1-performance-60fps-20261105-r89/quality-normal-gui/user-data/fd2_re/fd2_settings.json` | 181 | `a171535e1465913f…` |
| `promo-v.1.0.0-20261017/r2/user-data/fd2_re/fd2_save.json` | 14864 | `e144fa57c6e4127f…` |
| `r4-stage0-20261114/parity-out/hotel-save.json` | 14825 | `190b9f8f614c69a4…`（與 ch01 r10 相同） |

`KNOWN_DIFFERENCES`（`save.rs` 的 `SAVE_JSON_KNOWN_DIFFERENCES`）：JSON 欄位名大小寫；battle 套件型別的純量 null；
型別錯誤時的部分填入（設定檔）；字串的 HTML 與控制字元跳脫；整數鍵的非標準寫法；作業系統錯誤文字；非有限小數；
巢狀結構欄位的陣列寫法；存檔與附檔 JSON 的重複鍵。`NORMALIZATIONS`：只剩讀回結果的 `party_join_order` 的 null 與 `[]`
（`omitempty`，位元組相同）；battle 子樹的正規化在 #319 拿掉。

### fd2-rules 缺口（F 組發現，2026-11-15 以 #319 修正）

Go 寫出的存檔由 Rust 讀回再寫出時，位元組原本只在 fd2-rules 型別的 serde 上不同（黃金統計，127 份存檔）：

| 型別 | 欄位 | Go | 修正前的 fd2-rules | 修正 |
|---|---|---|---|---|
| `Unit` | `OffX`、`OffY`、`Exp`（`float64`） | 整數值寫 `0` | `0.0` | `go_json_marshal_indent` 改用 `src/go_json_float.rs` 的 `GoJsonFormatter`（第五項具名例外） |
| `Unit` | `Spells`、`Inventory`、`Equipped`、`InventorySlots`、`NativeInventoryFlags` | nil 寫 `null`、空的寫 `[]` | `Vec` 不分 nil，寫 `[]` | 欄位改為 fd2-rules 的 `NilableVec` |
| `State`／`StateSnapshot` | `flags` | nil 寫 `null`、空的寫 `{}` | 空的寫 `null` | 欄位改為 `NilableMap`，快照照 Go `cloneMap` 保留 nil |
| `BalanceOptions` | `characters`、`skill_hit`（整數鍵） | 依鍵的字串排序 | 依整數排序 | F 組寫出時已改走 `go_json_balance_options` |

`NilableVec`／`NilableMap` 以 `Deref` 當成 `Vec`／`BTreeMap` 使用；重新指定整個欄位的地方照 Go 原檔選 nil 或非 nil：
`append([]T(nil), src...)` 用 `NilableVec::appended`（沒有元素時是 nil），`make` 與字面值用 `from_vec`，`= nil` 用 `nil()`。
修正後 191 份存檔與第一章章重播的 `hotel-save.json`（`190b9f8f…`）都與 Go 逐位元組相同。

### 整合紀錄（A＋F，2026-11-14）

- A（`2a714a36`）與 F（`e217df8b`）合併。兩組都把 `Game::locale_message` 的參數改成 `&[localization::FormatArg]`，保留 A 的本體；F 的簽名變更列併入上方表格。F 的呼叫端配合 A 的只讀簽名改成 `freeze_modern_hd_title_load_slots(Some(&assets), …)`。
- 合併後 8 個原本「等 A 合併」的測試通過，拿掉 ignore；其餘依實際停點改標等 B／C／E。`tools/rust_game_test.sh`：82 通過、18 ignore。
- 存檔黃金在主 checkout 重跑時，Go 端兩次輸出不同：第一章戰中 F5 存檔的快照含 BIOS 地圖時鐘、地形翻轉、單位位移、地圖循環與地形相位，隨牆鐘前進。黃金工具在兩次存檔前以 `rustGoldenPinSnapshotTiming` 把這些欄位固定成同一組值（只影響測試輸入）。修正後 Go 端連續四次輸出相同，SHA-256 `dab44975…`。

## R4 階段 2 C 組：戰場操作、指令環與敵方回合（2026-11-14）

代號 `r4c`，合併基準 `1310436e`。範圍是 `PORTING.json` 裡 `group` 為 `C` 的 83 個函式（含 `main.go` 的 22 個）。

### 移植結果

| 項目 | 數量 |
|---|---|
| C 組函式 | 83 個全部有本體 |
| 整個本體是 `todo!()` 的 | 1 個：`draw_native_end_turn_recovery`（R5，純繪圖；`job.drawn` 由呈現回報設） |
| 分支停在 `todo!()` 的 | R8 21 處、R6 4 處（見下） |
| 隨呼叫端補移植的 `r8` 函式 | 12 個（下表） |

R8 的 `todo!` 都在第一章重播沒有執行到的分支（呼叫的函式在 `PORTING.json` 標 `r8`，或整段是 `r8` 的演出）：`ring_input` 的 END 確認、系統設定、巢狀系統選單、系統游標選單、物品面板、原始指令格；`confirm` 的施法、物品目標、物品／傳送術目的地與原始指令 0..35 目標確認（含 2026-11-14 修正的 `awardNative*Experience` 經驗發放，R8 移植這些分支時照 v.1.1.12 的 Go 現行版本）；`finish_selected_wait` 的原版取寶提示；`inspect_native_player_unit` 的狀態頁；`native_action_overlay_state` 的巢狀選單；`step_native_system_end_turn` 收合延續的取寶、死亡獎勵與全軍移動；`start_native_raw_camp0_turn_events` 的 `beginNativeStagingJob` 與 `step_native_turn_staging` 的第二次呼叫。R6 是 HD：`compose_native_map_frame_buffers` 的 HD 地圖合成與 HD 介面 HUD、END 收合的 `bindModernHDFullReward`／`bindModernHDSystemConfirmation`（忠實模式下 `hd` 是 nil，不會進入）。

隨呼叫端補移植的 `r8` 函式（本體照 Go，放在各自的 C 組檔）：

| Go | Rust | 呼叫端 |
|---|---|---|
| `action_overlay_runtime.go:nativeSystemOverlayReady` | `Game::native_system_overlay_ready` | `confirm` |
| `modern_hd_full_reward.go:bindModernHDFullReward`、`modern_hd_system_confirmation.go:bindModernHDSystemConfirmation` | `Game::bind_native_system_end_turn_hd`（私有，忠實模式無動作，HD 分支 R6） | `step_native_system_end_turn` |
| `modern_hd_battle_panel_chrome.go:preserveNativePanelMarker` | `ModernBattlePanelText::preserve_native_panel_marker`（私有） | `raster` |
| `native_player_movement.go:restorePlayerMovementCursor` | `Game::restore_player_movement_cursor` | `ring_input` 的 ESC |
| `native_turn_staging.go:failNativeTurnStaging` | `Game::fail_native_turn_staging`（私有） | `step_native_turn_staging` |
| `native_indexed_transition.go:nativeDelayTicks`（B 組檔） | `native_delay_ticks_ms`（`native_turn_staging.rs`） | 回合演出、AI 物品／修飾演出的 500 ms 停留；另取名字避免與 B 組補移植的同名函式衝突，整合時可合併 |
| `native_end_turn_recovery.go:cancelNativeEndTurnRecovery` | `Game::cancel_native_end_turn_recovery` | `step_native_end_turn_recovery`；Go `Draw` 在 END 回收畫不出來時也呼叫 |
| `native_ai_idle_recovery.go:restoreNativeAIIdleRecoveryRange` | 併入 `Game::fail_native_ai_idle_recovery_commit`（私有） | `step_native_ai_idle_recovery` |
| `native_ai_item_presentation.go:failNativeAIItemPresentation` | `Game::fail_native_ai_item_presentation` | `step_native_ai_item_presentation`（Go `Draw` 也呼叫） |
| `native_ai_modifier_presentation.go:failNativeAICommandModifierPresentation` | `Game::fail_native_ai_command_modifier_presentation` | `step_native_ai_command_modifier_presentation` |
| `native_item_panel_ui.go:nativeItemRuntimeRecords`、`syncNativeItemRuntimeRecord`、`commitNativeAIItemTransaction` | 同名 | AI 物品演出的發布與取消 |

Rust 新增、沒有 Go 對應的工具：`native_player_movement.rs` 的 `go_copy`（Go 內建 `copy`）；`modern_hd_battle_panel_chrome.rs` 的 `rgba_rect`、`new_rgba_canvas`；`modern_hd_battle_panel.rs` 的 `same_panel_font`、`picture_from_rgba`；`native_map_unit_window.rs` 的 `NativeMapSelectionOverlay`、`NativeMapActionOverlay`；`main.rs` 的私有 `compose_native_map_frame_buffers`（緊鄰 `compose_native_map_frame_at_with_palette_cycle`）。

### C 組：呈現回報

Go 由 `Draw` 設下列旗標，對應的 `step*` 等旗標才前進；Rust 的流程端都不設，階段 3 的呈現回報介面要涵蓋（無畫面驅動端直接回報）：

| 旗標 | Go 寫入端 | 等它的流程 | 備註 |
|---|---|---|---|
| `action_overlay_drawn`（經 `mark_action_overlay_drawn`） | `main.go:10159`（指令環已在原生整幀中）、`drawRing`（`main.go:10240`、`10302`）、HD 的 `modern_hd_action_icon.go:74`、`modern_hd_map_gpu.go:581`、`588` | `step_action_overlay_lifecycle`：開合各四幀，每幀要一次回報 | `mark_action_overlay_drawn` 已移植，只在開合中（`action_overlay_blocks_input`）生效；回報就是呼叫它 |
| `NativeTurnStagingJob::drawn` | `drawNativeTurnStaging`（`native_turn_staging.go:503`）與 `Draw` 的 defer（`main.go:9386`） | `step_native_turn_staging` 的白閃 | `drawNativeTurnStaging` 在 PAN 階段**先合成整幀**再複製到 `job.vga`（`native_turn_staging.go:491`），影響流程，無畫面驅動端也要做 |
| `NativePlayerStatusState::drawn` | `drawNativePlayerStatus`（`native_player_input.go:220`..`236`） | `step_native_player_status` | |
| `NativeSystemInfoUIState::drawn` | `drawNativeSystemInfoUI`（`native_system_info_ui.go:231`） | `step_native_system_info_ui`（開合十二幀、穩態調色盤循環） | |
| `NativeAIIdleRecoveryJob::drawn` | `drawNativeAIIdleRecovery`（`native_ai_idle_recovery.go:348`） | `step_native_ai_idle_recovery`：三張畫面後才提交 HP | 畫不出來時 `Draw` 復原 range、清 job、寫 `loadErr`、`aiBusy = false`（A 組表第 10 項） |
| `NativeEndTurnRecoveryJob::drawn` | `drawNativeEndTurnRecovery`（`native_end_turn_recovery.go:177`） | `step_native_end_turn_recovery` | 畫不出來時 `Draw` 呼叫 `cancel_native_end_turn_recovery`（A 組表第 9 項） |
| `NativeAIItemPresentationJob::drawn` | `drawNativeAIItemPresentation`（`native_ai_item_presentation.go:634`、`644`） | `step_native_ai_item_presentation` | HD 圖源缺件時 `Draw` 呼叫 `fail_native_ai_item_presentation` |
| `NativeAICommandModifierPresentationJob::drawn` | `drawNativeAICommandModifierPresentation`（`native_ai_modifier_presentation.go:596`、`602`） | `step_native_ai_command_modifier_presentation` | |

C 組的測試照 Go 測試直接呼叫 `mark_action_overlay_drawn` 或寫 `drawn = true`，等同回報。

### C 組：原生地圖整幀

- `compose_native_map_frame_at_with_palette_cycle` 照 Go 的交易：候選狀態與候選時鐘推進、DAC 循環、組輸入、合成，全部成功才發布。Go 另建 `candidateGame := Game{st, nativeMapClock}` 推進時鐘；`advance_native_map_clock` 只讀寫這兩個欄位，Rust 暫時把候選值換進 `self` 再換回。
- Go 的 `work`／`vga` 與 `g.nativeMapWork`／`g.nativeMapVGA` 共用記憶體（長度正確、沒有 ch23 暫存也沒有 HD 合成時），合成到一半失敗時已寫入的像素留在 `g` 上。Rust 照樣：就地寫入的緩衝失敗時也放回。
- 選取覆蓋層：Go 在輸入建好之後才呼叫 `nativeMapSelectionOverlay`（它延遲載入單位視窗素材）。Rust 的覆蓋層要 `&mut self`、輸入借用 `self`，所以先建覆蓋層、把它的錯誤留到同一個位置才回報；先建只多做一次素材快取，輸入建不成時 Go 不會載入這份素材。
- `nativeMapFrameSource`：Go 在不走行、沒有回合橫幅時把這一幀的輸入發布到 `g.nativeMapFrameSource`。它借用素材與戰場狀態，Rust 不保存，`Game::native_map_frame_source` 一律是 `None`；讀它的都是 HD（`modern_hd_ch23`、`modern_hd_sky_key`、`modern_hd_player9_map`、`modern_hd_story_motion`、`player_interface` 的 F11 切換、`player_poster`），R6 由那些呼叫端以同一組資源重組。見共用變更需求。

### C 組：共用變更需求

本工作樹沒有改 `Game`、`support.rs` 或別組的型別。下列需要主代理決定：

| 項目 | 原因 | 建議 |
|---|---|---|
| `draw_modern_battle_panel_ink`（A 組，`todo!("R6")`）在忠實模式也會被呼叫 | Go `prepareNativePlayerMovement`（與 D 組的物理演出）在字型有 TTF、`localeEntities` 存在時就收集 HD 面板文字，`renderBattlePanelText` 以 `raster` 預檢，`raster` 呼叫 `drawModernBattlePanelInk`；第一章重播的選取單位就會走到。只要字型載入了 TTF，`prepare_native_player_movement` 會停在這個 `todo!` | 把 `drawModernBattlePanelInk` 的「找得到不超框的字級」檢查（`faceFor` 與 `font.BoundString`，經 `Font::measure`）在 R4 移植，像素繪製留 R5；或在 R5 前讓無畫面驅動端不載入 TTF（`font.sf` 為 `None` 時不收集，與 Go 相同） |
| `Game::native_map_frame_source` 的型別 | `NativeFrameInput<'static>` 存不進借用的輸入（見上） | 改存重組所需的小型描述（`NativeMapFrameRuntime`、`range_mode`、覆蓋層描述與「已發布」旗標），或維持 `None` 到 R6 |
| `ModernBattlePanelText::font` 的比較 | Go 以 `previous.font == p.font` 比 `*Font` 指標；Rust 存複本。現以 `same_panel_font`（同一份 `sf` 的 `Arc`、基準尺寸與字級倍率相同）比較 | 若要與 Go 完全相同，`font` 改成共用的 `Arc<Font>` 再以 `Arc::ptr_eq` 比較 |
| `native_delay_ticks_ms` 與 B 組的 `native_delay_ticks` | 同一個 `nativeDelayTicks`，B 組的檔 | 整合時留一個 |

### C 組：KNOWN_DIFFERENCES

| 名稱 | 差異 | 影響 |
|---|---|---|
| `reach-nil-vs-empty` | Go `g.reach == nil` 與空 map 不同；Rust 的 `reach` 是 `BTreeMap`，nil 與空都是空，`native_map_selection_overlay` 以 `is_empty()` 代替 `== nil` | Go 只有在可達集合非 nil 卻是空的時候才不同；`PlanNativePlayerMovement` 的可達集合至少含原地 |
| `panel-font-identity` | 見上表 `same_panel_font` | 同一次面板收集裡 Go 的指標都是 `g.font` 或同一個區域字型，兩種比法結果相同 |
| `selection-overlay-asset-cache-order` | 輸入建不成時 Rust 已載入單位視窗素材（Go 不載入） | 只是快取，下一幀照樣使用 |
| `movement-seal-bytes` | `movementPanelSourceSeal` 以 serde_json 寫出與 Go `json.Marshal` 同形狀的陣列再取 SHA-256；位元組不保證與 Go 相同 | `seal` 只和自己比對 |
| `end-recovery-log-text` | Go `FD2_SHOT_AI` 時以 `log.Printf` 記 END 回收；Rust 以 `eprintln!`，沒有時間前綴 | 只影響診斷輸出 |
| `native-system-end-turn-state-take` | Go 的收合延續捕捉 `state` 指標，Rust 在延續執行時才從 `Game::native_system_end_turn_ui` 取出 | 收合期間沒有別的寫入端替換這個欄位 |

`NORMALIZATIONS`：無（C 組沒有黃金差分）。

### C 組：測試

`tools/rust_game_test.sh`（`FD2_REQUIRE_ORIGINAL=1`，輸出沒有「略過」）整個 crate：135 通過、22 ignore。C 組新增 57 個測試（Go 移植 45、Rust 新增 12），43 通過、14 ignore：

| ignore | 等 | 停點 |
|---|---|---|
| END 回收 4 個、AI 閒置回收 2 個、`test_compose_native_map_hud_uses_advanced_cycle` | B | 夾具或受測函式合成整幀，停在 `native_map_hud_input` |
| `test_native_player_focus_uses_raw_battle_view_and_x_first_safe_band`、`test_native_player_movement_cancel_restores_raw_cursor_before_sync` | B | `redraw_native_map_hud`、`sync_native_map_view` |
| `treasure_test.go` 4 個（`finishSelectedWait`） | D | `finish_successful_unit_action` |
| `reset_action_overlay_lifecycle_clears_end_turn_owner_and_ring`（Rust 新增） | D | `cancel_native_command32_presentation` |

Go 以 `t.Setenv("FD2_MUTE", "1")` 的 END／AI 回收夾具，在子行程設這個變數（F 組的 `f_child_env`）。

原本等 C 的 A、F 組測試：10 個通過、拿掉 ignore（`audio_test.rs` 3、`locale_test.rs` 2、`save_test.rs` 5）；`save_test.rs` 的 `title_json_load_accepts_saved_zero_chapter_and_keeps_bytes` 與 `campaign_save_load_restores_town_boundary_and_party` 改標實際停點「等 E」（`reset_native_class_ui_pulse`）。只改了屬性行。

R5 測試清單（需要 `Draw`、GPU 圖片或真 TTF 繪字，不在本階段移植）：

| 測試 | Go 檔:行 | 原因 |
|---|---|---|
| `TestNativeEndTurnRecoveryDrawCommitAndCallbackOrder` | `native_end_turn_recovery_test.go:87` | 以 `g.Draw` 與 `g.Update` 推進 |
| `TestNativeAIIdleRecoveryThreeDrawsBeforeHPCommit` | `native_ai_idle_recovery_test.go:74` | 同上 |
| `TestModernBattlePanelTextFrozenSources`、`TestModernBattlePanelChromeNativeSpans`、`TestModernBattlePanelChromeFrozenPaletteAndReject` | `modern_hd_battle_panel_test.go:19`、`124`、`192` | `raster` 經 `drawModernBattlePanelInk` 以真 TTF 繪字（另經 D 組 `renderPhysicalBattlePanel`）；外框與調色盤部分改由 Rust 新增測試驗 |
| `TestModernBattlePanelTextGPU`、`TestModernBattlePanelTextFourLocalePreview` | `modern_hd_battle_panel_test.go:238`、`353` | GPU 圖片 |
| `TestExecuteNativeAIItemRestoreUsesOriginalIndexedTail`、`…Damage…1CD17Tail`、`…Type21…1CAC7Tail` | `native_ai_item_presentation_test.go:14`、`108`、`347` | 以 `Draw` 推進演出 |
| `TestExecuteNativeAICommand17UsesOriginalIndexedTail` | `native_ai_modifier_presentation_test.go:13` | 同上 |
| `TestEvent63RunsBeforeEnemyAIWithTwoAtomicStagingCalls` | `native_turn_staging_test.go:210` | 以 `Draw` 推進白閃（另需 R8 的 `beginNativeStagingJob`） |
| `TestNativeAIIdleRecoverySameSourceProbe`、`TestNativeEndTurnRecoverySameSourceProbe` | `native_ai_idle_recovery_probe_test.go:25`、`native_end_turn_recovery_probe_test.go:24` | 以環境變數啟用的同源探針，輸出畫面（fd2-tools） |
| `TestModernHDMovementPanelLegacyNativeAndClear`、`TestModernHDMovementPanelSources`、`TestModernHDMovementPanelRejectsChangedSource` | `modern_hd_movement_panel_test.go:201`、`79`、`134` | 需要 `FD2_HD_MOVEMENT_PANEL_SAVE` 外部存檔與 `drawNativeMovementPanel` |

其他沒有移植的 C 組主題 Go 測試（不需要畫面，所以不列 R5）：

- 受測函式是 `r8`：`TestSystemOptionsOverlayUsesExactCellsAndReturnsAfterFourClosePresents`（`nativeSystemOptionsReady`、`toggleNativeSystemOption`）、`TestNativeCommandTargetFieldMaterializeAndReset`（`materializeNativeCommandTargetField`）、`TestNativePlayerStatusTwoPagesWaitForDrawAndRestore`（`advanceNativePlayerStatus`）、`TestNativeAIItemPresentationLateCancelRollsBack`（`planNativeAITargetItem`）、`TestPrepareNativeSystemInfoUIRunsTwelveOpenWaitTwelveClose`（`prepareNativeSystemInfoUI`；改以 Rust 新增測試直接建立狀態）、`native_system_info_locale_test.go` 3 個（`localizeSystemInfo`）、`native_battle_name_test.go` 其餘 3 個（`renderNativeBattleNameIndexed` 與兩個原點常數，都是 `r8`）、`TestNativeEndTurnRecoveryCommittedFieldsRoundTripCurrentSave`（`buildNativeCurrentSaveStored`）。
- 走到本組 R8 分支：`native_action_overlay_test.go` 的原始指令 0／13／17／25..27／32..35 確認 8 個（`confirm` 的原始指令目標確認）、`native_turn_staging_test.go` 的 event52／63／72／74／76／79 9 個（`beginNativeStagingJob`，第二十三章以後）。
- 部分移植：`TestNativePlayerStatusGatesAndFailedResourcesDoNotSelect` 只移前兩段；第三段要 `t.Setenv("FD2_ASSET_PACK", 空目錄)` 並走 `r8` 的 `beginNativePlayerStatus`。`TestBundledNativeBattleNameAssetsCoverImpactFixture` 只移載入部分；後半的 `nativeBattleNameImage` 是 `r8` 的畫面轉換。
- 別組主題、以 C 組函式當驅動：`native_continue_handoff_test.go`、`battleevent_test.go`、`native_physical_scene_test.go`、`native_command*_presentation_test.go`、各章流程測試等，由主題組或階段 3 移植。

黃金差分：C 組沒有加。指令環生命週期、availability 與 END 回收是決定性的狀態機，已以 Go 測試的期望值覆蓋；合成整幀要等 B 合併才跑得完整，適合在階段 3 以重播檢查點比對。

### C 組：fd2-rules／fd2-campaign 缺口

沒有改那兩個 crate。測試發現：`fd2_formats::fdother::FdotherError` 沒有 `PartialEq`（測試改比 `ok()`）；`battle::Unit` 有私有欄位，不能用結構更新語法建立。

### C 組：敵方回合與亂數

`native_ai_*`、`native_end_turn_recovery` 只呼叫 `fd2_rules` 的計畫、驗證、提交與取消函式，亂數狀態沿用 `Game::native_rng_state`（Go `nativeRNGState`）與各計畫自帶的 `rng_state`，沒有另造亂數來源。單位一律以 `UnitIdx` 指向 `st.units`；Go 以指標相等找單位的 `unitIndex` 改成檢查索引範圍。

## R4 階段 2：B 組（劇情對話與事件演出）

2026-11-14，代號 `r4b`，合併基準 `1310436e`。範圍是 `PORTING.json` 裡 `group` 為 `B` 的 108 個函式（含 `main.go` 的 43 個）。

### 移植結果

| 項目 | 數量 |
|---|---|
| B 組函式 | 108 個全部有本體；另隨呼叫端補移植 6 個 `r8` 小函式（下表） |
| 整個本體是 `todo!()` 的 | 2 個，都是 R5：`draw_native_palette_ramp`（繪圖；`drawn` 由呈現回報設定）、`draw_indexed_localized_text`（非繁中語系的 TTF 點陣化） |
| 部分是 `todo!()` 的 | `beat_start`（第一章沒有的原語）、`resolve_campaign_dialog_line`／`prepare_native_dialogue_frames`（非繁中語系、HD 介面、第 28 章視圖繼承）、`dlg_page_count`（非原生對白）、`step_battle_walk_segment`（全軍移動的格子事件）、`advance_battle_event`／`finish_battle_event_with_error`（ch20 系 staging、章節結果失敗）、`handle_native_story_input`（近似戰後）、`begin_native_field_event61`／`75`、`start_native_raw_camp2_turn_events`（命中之後的對白與演出）、`start_native_unit_death_present_mode`（map 23 舞台、HD 死亡素材） |
| `todo!` 分類 | R5 2、R6 4、R8 30 |

R8 的 `todo!` 都在第一章重播沒有執行到的分支上：分支呼叫的 Go 函式在 `PORTING.json` 標 `r8`（例如 `startNativeCh20SkyKeySequence`、`startNativeCh23Loop`、`startNative2189A`、`startNativeUnitPresent`、`evalBeatCondition`、`grantItemToParty`、`event61DialogueActions`、`preflightNativeEvent76Final`）。進 `todo!` 之前的 payload 檢查與 `loadErr` 照 Go 移植，失敗即關閉的訊息與 Go 相同。

隨呼叫端補移植的 `r8` 函式：

| Go | Rust | 呼叫端 |
|---|---|---|
| `main.go:finishBattleEventWithError` | `Game::finish_battle_event_with_error` | `advance_battle_event`、`handle_battle_event_dialogue_input` |
| `native_indexed_transition.go:nativeDelayTicks` | `native_delay_ticks` | `step_native_indexed_transition` |
| `native_indexed_transition.go:applyNativeIndexedTransitionPalette` | `Game::apply_native_indexed_transition_palette` | 同上 |
| `native_indexed_transition.go:buildNativeIndexedTransitionInput` | `Game::build_native_indexed_transition_input` | `ensure_native_palette_frame` |
| `native_palette_pulse.go:(*nativePalettePulseJob).applyCurrent` | `NativePalettePulseJob::apply_current` | `step_native_palette_pulse` |
| `native_ch23_post.go:(*nativeCh23AdapterState).rotateRows` | `NativeCh23AdapterState::rotate_rows` | `prepare_native_ch23_steady_stage` |

`cloneNativeCh23AdapterState`（`r8`）就是 `Clone`，`itoa`（`r8`）對整數與 `to_string()` 相同，兩者直接寫在呼叫處。

Rust 新增、沒有 Go 對應的項目：`main.rs` 的 `HandlerUnitRef`、`Game::handler_unit`／`handler_unit_mut`、`Game::focus_native_map_cursor_steps_with_hud`、`redraw_native_map_hud_state`、`Game::walk_unit`／`walk_unit_mut`、`beat_advance_then`、`go_int_ptr`；`native_story_map.rs` 的 `Game::sync_story_native_map_units`、`append_native_story_map_actors_into`、`go_native_map_view_plus_v`；`native_indexed_transition.rs` 的 `NativeTransitionFrameSource`；`native_story_dialogue.rs` 的 `STORY_MOUTH_RNG_SEED`、`Game::story_mouth_intn30`；`native_confirmation_mouth.rs` 的 `CONFIRMATION_MOUTH_RNG_SEED`、`NATIVE_CONFIRMATION_OWNER_END_TURN`、`confirmation_owner_token`。`main.rs` 新增的項目都緊鄰 B 組的函式。

### B 組：所有權

- **處理器單位參照**：Go 的 `actingActor`、`handlerUnitAt` 回傳 `*battle.Unit`，戰場上指 `g.st.Units[slot]`，劇情場景指 `&g.storyActors[slot]`。Rust 回傳 `HandlerUnitRef::{Battle(UnitIdx), Story(usize)}`，再以 `handler_unit(_mut)` 取用。
- **演出工作**：`step_act_job` 先把 job 從 `self.act_job` 取出再推進，沒有收尾才放回；收尾時延續可能裝上新 job（`beat_advance` → 下一個 `act` 拍），所以不放回舊 job。`step_focus_unit` 反過來把 job 留在 `self.focus_job` 原地改寫，因為 A 組的 `sync_story_native_map_focus_view` 在步進途中讀 `focus_job.native_view`（Go `main.go:1150`）。其他演出工作（索引轉場、調色盤漸變／脈衝、出場、0x1DB65）也原地推進，收尾時 `take()` 出來再呼叫延續。
- **走行**：照 A 組定的跨組介面，`step_battle_walk_segment` 一律讀寫 `self.walk`，走行單位是 `st.units` 的索引（Go 的 `w.u` 為 nil 時解參考 panic，Rust 以 `expect` 對應）。
- **游標聚焦**：Go 把會改寫 `Game` 的 `nativeCursorStepHUD` 當 callback 交給 `battle.State.FocusNativeMapCursorSteps`；借用 `st` 時不能再借用 `Game`，所以 `focus_native_map_cursor_steps_with_hud` 在 `Game` 這層照 fd2-rules 的迴圈重寫（含追蹤 hook 呼叫），三個呼叫端（`focus_native_map_cursor_on_unit`、`ai_focus_cursor`、`position_screenshot_cursor`）共用。
- **劇情地圖載體**：Go 的 `storyNativeMapState.Units[i]` 是 `&g.storyActors[i]`。Rust 的載體擁有單位複本，建立與 SPAWN 時與 `story_actors` 等長，`compose_native_story_map_frame` 合成前以 `sync_story_native_map_units` 重新對齊。**別組讀 `story_native_map_state.units` 之前要先呼叫它**（Go 讀這份指標的還有 `native_24b4d.go`、`modern_hd_story_map.go`、`evalBeatCondition`，都是 R6／R8）。
- **0x24618 輸入**：`indexedmap::NativeTransitionFrameInput<'a>` 借用素材庫與選擇器快取。`build_native_indexed_transition_input_for_*` 回傳擁有資源的 `NativeTransitionFrameSource`（選擇器快取、格子、單位層等複本與 LUT 索引），合成前以 `source.input(assets)` 組出借用輸入（120「fd2-game 保存資源本身，每幀組出借用的輸入」）。`fd2-campaign` 沒有改。
- **0x1DB65 的狀態快照**：非保留模式時 Go 讓 `g.st` 指向 job 的逐幀快照，Rust 每次發布複製快照；保留模式不換 `g.st`，只在最後一幀同步 HUD 狀態，rollback 同理。`hd_inputs` 只給 HD（map 23 舞台、HD 死亡素材），忠實模式不填。

### 虛擬時鐘與輸入

- 對白的逐字（`step_native_story_dialogue_progress`）、開框／收框（`native_dialogue_closing_t`）、翻頁捲動（`dlg_scroll_t`）、嘴型（`step_dialogue_mouth` 看 `self.frame` 的奇偶）與等鍵箭頭（`native_dialogue_arrow_ticks`）都按邏輯幀推進，一幀一步，不讀牆鐘也不讀 `virtual_now`。驅動端照 A 組「虛擬時鐘」每幀呼叫一次 `update` 即可。
- B 組沒有直接讀鍵盤：推進鍵由呼叫端組成 `NativeStoryInput`（Enter／ESC）或 `native_battle_dialogue_advance_input` 的參數，都來自 `tick_input`（A 組 `update`）。
- 演出工作的 `drawn` 旗標（索引轉場、調色盤漸變／脈衝、出場、0x1DB65）在 Go 由 `draw*` 設定；Rust 的流程端不設，等階段 3 的呈現回報。沒有回報時這些工作停在第一幀不前進（與 Go 不呼叫 `Draw` 時相同）。

### 亂數

Go `stepDialogueMouth` 呼叫套件層 `rand.Intn(30)`（全域來源，每次啟動不同種子），`sampleNativeConfirmationMouthRandom` 以 `rand.Int63()` 當私有來源的種子。兩者都只影響嘴型，不碰 `g.rng` 與存檔。Rust 不用全域狀態：故事對白嘴型新增 `Game::story_mouth_rng`（共用變更），確認嘴型沿用 `native_confirmation_mouth_rng`，都以固定種子惰性建立（`VIRTUAL_CLOCK_ORIGIN` 的奈秒值加 1、加 2，與遊戲亂數的預設種子錯開）。同一輸入序列的嘴型因此可以重現；Go 本來就不可重現，所以這不改變任何可比對的輸出。

### TTF 量測的影響

對白的量寬與斷行只在兩種情況用到 TTF（A 組由外殼注入的 `TextMeasurer`）：

| 函式 | 用到 TTF 的分支 | 無畫面時目前的行為 |
|---|---|---|
| `dlg_page_count` | 非原生對白（沒有 `native_dialogue`）以 `dlgWrap` 依字寬斷行算頁數 | 停在 `todo!("R8: main.go:dlgWrap")` |
| `resolve_campaign_dialog_line` | 非繁中語系以 `localizedNativeDialogueLayout` 依字寬重排原生頁 | 停在 `todo!("R8: …")` |
| `prepare_native_dialogue_frames` | 非繁中語系以 `composeLocalizedNativeDialogueProgressiveFrames` 逐字合成 | 停在 `todo!("R8: …")` |
| `draw_indexed_localized_text` | TTF 點陣化進索引色畫格 | `todo!("R5: …")` |

忠實模式繁中的原生對白用 FDTXT 點陣字與編譯好的頁面（`NativeDialogueLayout.pages`／`glyph_ids`），完全不經 TTF，結果決定性。第一章重播的對白都是原生繁中（`dlgWrap` 在階段 0 覆蓋率是 0），所以階段 3 的檢查點重播不受 TTF 量測影響；改跑其他語系或含一般對白的章節時，`dlgWrap` 與在地化重排要先有與 Go 相同的量寬。

### 共用變更需求

本工作樹已以最小改動先做（主代理整合時決定採用）：

| 項目 | 原因 | 寫法 |
|---|---|---|
| `Game::story_mouth_rng: Option<battle::GoMathRand>`（新欄位） | 故事對白嘴型的亂數不能用全域狀態，也不能借用遊戲亂數 `rng` | `main.rs` 的 `Game`，緊鄰 `native_confirmation_mouth_rng` |
| `Game::award_death_reward` 的 `killer` 改 `Option<UnitIdx>`（D 組函式的簽名） | `completeTurn` 傳 nil killer | `main.rs`，本體仍是 D 的 `todo!()`；D 組呼叫端傳 `Some(..)` |
| `HandlerUnitRef` 與 `Game::handler_unit(_mut)`（新型別，放在共用的 `main.rs`） | 處理器單位可能在 `st.units` 或 `story_actors` | 緊鄰 `acting_actor` |
| `main.rs` 尾端 `mod main_b_test;` 一行 | B 組 `main.go` 函式的測試 | 緊鄰 A 組的 `mod main_a_test;` |
| A 組測試 `test_game_materializes_editable_native_map_runtime` 拿掉 ignore | `sync_native_map_view` 已移植，`--ignored` 跑過通過 | 只刪那一行屬性 |

需要主代理或別組決定：

| 項目 | 原因 | 建議 |
|---|---|---|
| C 組 `build_native_map_frame_input` 的回傳生命週期 | 骨架是 `(&mut NativeMapAssets, …) -> NativeFrameInput<'static>`；B 組 `native_ch28_post_frame_input` 照這個簽名轉回。C 組改成借用素材庫的 `NativeFrameInput<'a>` 時，`native_ch28_post_frame_input` 要跟著加生命週期，`start_native_unit_death_present_mode` 逐幀用完即丟，不受影響 | 整合時一起改 |
| D 組讀 `NativeTransitionFrameSource` | `build_native_indexed_transition_input_for_state` 的 D 組呼叫端（補血演出、單位登場）照新回傳型別 | 呼叫端改用 `source.input(assets)` |
| 別組讀 `story_native_map_state.units` | Rust 存複本，見「B 組：所有權」 | 讀之前呼叫 `sync_story_native_map_units` |

### KNOWN_DIFFERENCES

| 名稱 | 差異 | 影響 |
|---|---|---|
| `cosmetic-mouth-rng-fixed-seed` | 嘴型亂數用固定種子（見「亂數」） | Go 本來就不可重現；不影響戰況與存檔 |
| `confirmation-mouth-owner-token` | Go 的擁有者是 `any`（結束回合介面指標或字串），Rust 對應成 `OwnerToken`：結束回合介面是固定標記，字串是 FNV-1a 雜湊。前後兩個結束回合介面在 Go 是不同擁有者 | 兩個介面之間一定經過沒有擁有者的幀，嘴型照樣重來 |
| `go-int-pointer-format` | Go 錯誤訊息以 `%v` 印 `*int`，非 nil 時是記憶體位址；Rust 印值（nil 都是 `<nil>`） | 只影響 `loadErr` 文字 |
| `story-roster-nil-vs-empty` | Go 以 `g.storyRoster != nil`、`g.storySpawned == nil`、`g.nativeBattleGlyphs == nil` 區分 nil 與空，Rust 的容器不分 | 只在 LOADCH 給出空的非 nil 名冊時走不同分支；正式資料沒有 |
| `handler-story-path-no-clean` | Go `filepath.Join` 會詞法整理路徑，Rust 用 `join_path` | StoryIndexMap 的 script 都是單純相對檔名 |
| `story-native-map-units-copy` | Go 載體的單位是指向 `storyActors` 的指標，Rust 存複本、讀前同步 | B 組讀取處都先同步；別組見共用變更需求 |
| `battle-event-run-pointer` | Go `advanceBattleEvent` 持有 `run` 指標逐項推進；若某個動作中途換掉 `g.battleEvent`，Go 繼續舊的，Rust 繼續 `self.battle_event` | 目前沒有動作在迴圈內換掉 `battle_event` |
| `native-dialogue-bounds-error-order` | Go 以 map 字面值（順序隨機）逐 phase 算對白框範圍，Rust 依 opening、steady、closing 的順序 | 只在多個 phase 同時算不出範圍時，回報的錯誤不同 |
| `cutscene-log-no-timestamp` | Go `log.Printf` 帶主機時間前綴，Rust `eprintln!` 不帶 | 只影響 `FD2_CUTSCENE_LOG` 的除錯輸出 |
| `ch28-post-hd-inputs-unrecorded` | 0x1DB65 的 `hd_inputs` 不填 | 只給 HD 與 map 23（R6／R8） |

`NORMALIZATIONS`：無（B 組沒有黃金差分）。

### B 組測試

`tools/rust_game_test.sh`（`FD2_REQUIRE_ORIGINAL=1`，輸出沒有「略過」）：全 crate 133 通過、24 ignore。B 組 57 個測試，50 通過、7 個 `ignore`；其中 Go 測試 51 個、B 組補充 6 個。

| 檔 | Go 來源 | 測試 |
|---|---|---|
| `main/main_b_test.rs` | `beatrunner_test.go` 30 個、`native_map_presentation_test.go` 6 個、`native_hud_redraw_test.go` 4 個、`battleevent_test.go` 1 個 | 41 個 Go 測試、4 個補充 |
| `native_story_dialogue/native_story_dialogue_test.rs` | `native_story_dialogue_progress_test.go` 7 個 | 7 個 Go 測試、1 個補充（嘴型亂數可重現且不碰 `rng`） |
| `native_palette_ramp/native_palette_ramp_test.rs` | `native_palette_ramp_test.go:200` | 1 個 Go 測試、1 個補充 |
| `native_story_input/native_story_input_test.rs` | `native_story_input_test.go` 1 個 | 1 |
| `native_event_dialogue/native_event_dialogue_test.rs` | `native_event_dialogue_input_test.go` 1 個 | 1 |

| ignore | 等 | 原因 |
|---|---|---|
| `test_beat_palette_update_presents_recovered_full_dac_white_flash`、`test_beat_native_palette_fade_out_fails_closed_without_indexed_input` | C | `native_map_assets_available` |
| `test_beat_sequence_end_triggers_node_transition` | C | `bgm` 拍經 `play_bgm` 呼叫 `current_native_system_options` |
| `test_native_story_advance_accepts_escape_like_enter` | C | `enter_node` 呼叫 `reset_action_overlay_lifecycle` |
| `test_battle_walk_preserves_native_seven_tick_record_lifecycle`、`test_battle_walk_notes_selector0_during_walk_and_dispatches_at_action_end`、`test_battle_walk_right_step_also_notes_selector0` | C | 走行每格呼叫 `native_ch12_up_walk_frame`，走完呼叫 `begin_battle_action_overlay` |

測試寫法的調整：Go 的 `newBeatTestGame` 呼叫 `attachOfficialLocale`，這些測試的 `campLines` 沒有 `line_id`，`localizedStoryText` 在 `zh-Hant` 下直接回傳原文，Rust 的 helper 只設 `locale_id`（官方語系目錄要等 C 組的語系契約）。Go 以 `assetPath("assets/…")` 讀受版控素材，Rust 改用 `test_support::remake_asset`（同一個檔）。Go 走行測試的單位不在 `g.st` 裡，Rust 的走行單位是 `st.units` 的索引，所以放進一個沒有視圖、沒有格子事件的 `State`。

R5 測試清單（需要畫面或真字型量寬，不在本階段移植）：

| 測試 | Go 檔:行 | 原因 |
|---|---|---|
| `TestDlgPaginationPreservesText`、`TestDlgShortLineSinglePage`、`TestDlgPaginationStartsSmoothScrollAndBlocksSkip` | `dlg_test.go:12`、`:40`、`:47` | 非原生對白的頁數經 `dlgWrap`（TTF 量寬） |
| `TestNativeStoryOpeningFrameIndexPreservesFiveStageOrder` | `native_story_dialogue_progress_test.go:176` | 受測的 `nativeStoryOpeningFrameIndex` 只給 `drawNativeStoryDialogue` 用（`r8`） |
| `TestNativePaletteRampDrawAcknowledgesIndexedFrame` | `native_palette_ramp_test.go:53` | 斷言 `drawNativePaletteRamp` 的畫面 |
| `TestDrawIndexedLocalizedLineUsesNativePaletteAndRejectsMissingGlyph`、`TestAllHandlerNativeDialoguesCompileForThreeTranslatedLocales`（另 `:14`、`:53` 測 `r8` 的在地化重排） | `native_story_dialogue_locale_test.go:97`、`:123` | 非繁中語系以 TTF 點陣化、重排與逐字合成 |
| `TestNativeDeathStageUsesSnapshotSeed` | `modern_hd_death_stage_test.go:165` | HD 死亡舞台的畫面（R6） |

未移植的其他 Go 測試（不需要畫面，但不屬於這一階段能單獨跑的範圍）：

| 類別 | 測試 | 原因 |
|---|---|---|
| 依賴 C 組測試 fixture | `native_palette_ramp_test.go:14`、`native_indexed_transition_test.go:150`、`native_story_map_spawn_test.go:23`、`:59`、`native_ch28_post_present_test.go:49`、`:89` | 都先以 `completeNativeMapFrameFixture`（`native_map_frame_input_test.go`）建素材庫，並經 `native_map_assets_available`；C 組合併後連同 fixture 移植 |
| `r8` 函式或其他章 | `beatrunner_test.go` 的條件分支（`TestBeatAnyUnitInactive…` 3 個、`TestBeatRosterHas…`、`TestBeatNativeEventState…`、`TestBeatNativeInactiveCount…`、`TestBeatNativeCh15Raw…`、`TestBeatChapter22Raw…`、`TestBeatNativeAnyOf…`，共 9 個）、`TestBeatGrantItem…`、`TestBeatNativePalettePulseFailsClosed…`、`TestBeatNativeStagingPresentFailsClosed…`、`TestBeatNativeCh23LoopFailsClosed…`；`native_palette_pulse_test.go` 的 `:18`、`:58`；`native_field_event61_test.go` 5 個、`native_field_event75_test.go` 2 個、`native_turn_event76_test.go` 1 個 | 受測分支呼叫 `r8` 函式（`evalBeatCondition`、`grantItemToParty`、`startNativePalettePulse`、`startNativePaletteUpdate`、`event61DialogueActions` 等），在 R8 對應章移植 |
| 整段流程（階段 3） | `beatrunner_test.go` 其餘（`TestChapter1PostRuntimeContext…`、`TestCh00…`、`TestChapter1PreLoadCH…`、`TestPostSpawnJoinTail…`、`TestBeatSyncParty…`、`TestCampaignPersistence…`、`TestMap0Acting…`、`TestBeatSpawnCarriesRawPlacementGate…`、`TestBeatSpawnIntroWithoutNativeAssets…`、`TestFastForwardShotCampaign…`、`TestChapter2StoryPreserves…`）、`battleevent_test.go` 其餘 19 個、`ch*_flow_test.go`、`native_dialogue_parity_test.go`、`storybg_dialogue_parity_test.go` 等 | 以真實章節腳本、地圖與戰役圖跑整段（經 `enter_node`、`load_game`、`assetPath` 解析），屬於別組函式或階段 3 的重播；`fastForwardShotCampaign` 是 `r8` |
| 別組主題 | `native_confirmation_mouth_test.go` 3 個（整備確認與結束回合介面，E／C）、`native_action_death_present_test.go`、`native_unit_present_test.go`、`native_turn_staging_test.go` 等呼叫 B 組函式的演出測試 | 主題屬於 C／D／E，由主題組移植 |

黃金差分：B 組沒有加。可以逐幀比對的是對白逐字與 0x1DB65 等演出的 `step*`，但它們需要 C 組的素材庫（`native_map_assets_available`）與呈現回報；適合在階段 3 以重播檢查點或收 `drawn` 回報的單元黃金比對。

### fd2-rules／fd2-campaign 缺口

沒有改那兩個 crate。`battle::State::focus_native_map_cursor_steps` 的 callback 只收兩個視圖值，不能改寫呼叫端的 `Game`，B 組在 `Game` 這層重寫同一個迴圈（見「B 組：所有權」）；`indexedmap::NativeTransitionFrameInput` 借用資源，B 組以 `NativeTransitionFrameSource` 擁有資源。兩者都不需要改上游。

### 整合紀錄（B＋C，2026-11-14）

- C（`fb4b47dc`）與 B（`43176839`）合併；`main.rs` 自動合併，`PORTING.md` 以 C 版為底接上 B 段落，B 的 13 列簽名變更併入表格。
- 跨組簽名對齊兩處：C 的 `award_death_reward(Some(tgt), sel)` 配合 B 改成 `Option<UnitIdx>` 的擊殺者；B 的 `native_ch28_post_frame_input` 改成借用 `self` 與候選狀態的 `<'a>` 簽名，配合 C 的 `build_native_map_frame_input`。
- C 的 `native_delay_ticks_ms` 與 B 補移植的 `native_delay_ticks` 是同一個 Go `nativeDelayTicks`（`native_indexed_transition.go:55`），留 B 的一份，C 的三個呼叫端改用它。
- 合併後 14 個互等的測試通過，拿掉 ignore；其餘 12 個依實際停點改標等 D（`finish_successful_unit_action`、`cancel_native_command32_presentation`）或等 E。`tools/rust_game_test.sh`：200 通過、14 ignore（含 2 個由黃金腳本執行）。
- C 組提出、主代理的決定：
  - A 組 `draw_modern_battle_panel_ink` 在忠實模式也會被呼叫（有 TTF 與 `locale_entities` 時預檢 HD 面板文字），第一章選取單位就會走到。R4 先移植「找得到不超框字級」的量測部分，像素繪製留 R5；交給第三波 E 組（`modern_hd_text`）。
  - `Game::native_map_frame_source` 只有 HD 程式讀，維持 `None` 到 R6。
  - 面板字型以 `same_panel_font` 比較，列為已知差異；要完全一致時再把 `font` 改成 `Arc<Font>`。

## R4 階段 2：D 組（攻擊演出與動畫指令播放）

2026-11-15，代號 `r4d`，合併基準 `57e34259`。範圍是 `PORTING.json` 裡 `group` 為 `D` 的 120 個函式（含 `main.go` 的 16 個）。
`native_command*_presentation` 十七個檔的 step／cancel 由同組分身在獨立工作樹移植（`6f81b961`），再快轉併入。

### 移植結果

| 項目 | 數量 |
|---|---|
| D 組函式 | 120 個全部有本體；另隨呼叫端補移植 `r8` 函式（下表） |
| 整個本體是 `todo!()` 的 | R5 2 個：`draw_native_physical_body`、`draw_native_physical_prelude`（純繪圖；旗標由呈現回報設）。R6 9 個：HD 凍結來源 `freeze_native_command9_map_input`、`new_modern_hd_compound_map_scene`（兩個同名，一個自由函式、一個 `Game` 方法）、`freeze_compound_map_input`、`ModernHDCompoundMapScene::{digest, seal, validate}`、`freeze_modern_hd_command_geometry`、`indexed_hd_input_identity`（見「D 組：HD 預凍結延後」） |
| 分支停在 `todo!()` 的 | R8 15 處、R6 3 處（見下） |

R8 的 `todo!` 都在第一章重播沒有執行到的分支（呼叫的函式在 `PORTING.json` 標 `r8`）：`check_result` 的原生章節結果（`beginNativeChapterResult`）、`ai_step` 的 mode 11、敵方法術與敵方指令／物品（`startNativeAIMode11`、`executeAISpell`、`executeNativeAIAction`）、物理攻擊 FIGANI header1 非零的出發與捲動（`prepareNativePhysicalDeparture`、`nativePhysicalScroll`、兩擊之間的回捲）、死亡程式的 `staging` 與 `clear_hp_from`、滿欄掉落的丟棄提示（`prepareNativeItemPanel` 之後）、敗北返回標題（`startNativeDefeat`）、暫時狀態到期提示（`composeNativeTransientExpiryFrame`）、command29 的 command28 分支、補血演出的 `validateModernHDHealSource` 與 `buildNativeCommandHealDigitFrames`。R6 是 HD 開啟時才走的分支：HD 素材預檢（`prepareBattleHDResources`）、HD 到期框預檢、HD 主題的獎勵訊息底圖（`prepareModernHDRewardWorld`）。

隨呼叫端補移植的 `r8` 函式（本體照 Go，放在各自的 D 組檔）：

| Go | Rust | 呼叫端 |
|---|---|---|
| `native_command{0,1,2,3,5,6,7,8,9_ai,9_player,10_12,24,29,32,33,34,35}_presentation.go` 的 `failNative…Presentation` | 同名 `fail_native_…_presentation` | 各自的 step；Go `Draw` 畫不出來時也呼叫 |
| `modern_hd_heal_source.go:76` 的 rollback | `native_heal_rollback` | `step_native_command_heal_presentation` |
| `native_command24_presentation.go` 的 `nativeCommand24RawFramesDelay` 與延遲表 | `native_command24_raw_frames_delay`、`NATIVE_COMMAND24_RAW_DELAYS` | `step_native_command24_presentation` |
| `native_command24_presentation.go:109` `nativeFIGANIImages` | `native_figani_images`（`native_attack_assets.rs`） | `ensure_native_attack_presentation` |
| `native_level_up_dialogue.go:104` `nativeMessagePages` | `native_message_pages` | `begin_native_transient_phases` |
| `native_transient_presentation.go:187` `focusNativeTransientRecord`、`:205` `nativeTransientPresentationEvent` | 同名 | `begin_native_transient_phases` |
| `modern_hd_battle_surface.go:192` `modernPhysicalTargetBGIndex` | `modern_physical_target_bg_index` | 物理攻擊 HD 來源 |
| `player_interface.go:32` `hdPresentationEnabled` | 寫在 `preflight_modern_hd_transient_expiry` 內 | 同左 |

Rust 新增、沒有 Go 對應的項目：`native_command0_presentation.rs` 的 `presentation_units`（演出工作與計畫的單位在 `g.st.Units`，`fd2_rules` 的發布函式另收單位切片）與各指令檔的 `publish_native_commandN_mp`；`main.rs` 的 `SharedAfter`、`take_shared_after`、`finish_successful_unit_action_then`、`Game::ai_step_act`、`ai_step_finish`、`ai_step_write_byte5`（`aiStep` 的三個閉包）；`native_action_death_present.rs` 的 `NativeTransientPreviewSaved`、`save_native_game_preview`、`restore_native_transient_preview`、`native_transient_focus_preview_slots`；`native_physical_body.rs` 的 `NativePhysicalBodyAssembly`、`append_native_physical_body_stage`；`native_death_program_runtime.rs` 的 `run_pending_death_program_step`；`native_transient_presentation.rs` 的 `NativeTransientRun`、`run_native_transient_damage`、`run_native_transient_expiry`；`native_transient_phase.rs` 的 `native_record_i16`；`native_level_up_dialogue.rs` 的 `NativeMessageStrings`；`modern_hd_battle_panel.rs` 的 `same_panel_text`；`modern_hd_dialogue_chrome.rs` 的 `MODERN_HD_GOLD_DIALOGUE_TILE_BYTES`。`main.rs` 新增的項目都緊鄰 D 組的函式。

### D 組：所有權

- **行動收尾的 `after`**：Go `finishSuccessfulUnitAction(actor, after)` 把同一個 `after` 交給五個續行閉包（死亡演出、掉落訊息、滿欄丟棄、死亡程式、升級對話），每個閉包都以同一個 `after` 重入自己；沒有被存起來的閉包就丟掉。`FnOnce` 只能交出一次，Rust 放進 `SharedAfter`（`Arc<Mutex<Option<Box<dyn FnOnce>>>>`），真正執行的那條路才取出。沒有開始的工作把閉包丟掉，`after` 仍留在格子裡，與 Go 相同。
- **演出工作的單位**：計畫與工作的 Go `*battle.Unit` 都是 `st.units` 的 `UnitIdx`；`fd2_rules` 的發布、完成與中止函式收 `presentation_units(&mut self.st)`（`st` 為 `None` 時以 `expect` 對應 Go 解參考 nil）。
- **AI 計畫**：Go 的 `act`、`finish` 閉包捕捉 `*battle.AIPlan`；Rust 以 `Arc<AIPlan>` 交給走行延續與演出收尾。
- **暫時狀態的候選**：Go `adoptNativeStateCandidate` 把候選的單位內容寫回 `g.st` 原本的 `*Unit`，回傳「候選指標 → 原指標」讓握著舊指標的地方不失效；Rust 的單位一律是索引，候選與現況索引相同，直接換上候選狀態，對照永遠是空的（`resolve` 是恆等）。
- **掉落訊息的 `message`**：Go 的開框、關框閉包捕捉 `message` 指標並把 `message.opening`／`closing` 指向 `g.nativeClassUIJob`。Rust 的閉包在執行時從 `native_system_end_turn_ui.reward_message` 取同一則訊息；`opening`／`closing` 只給 HD 繪製比對，Rust 不另存複本（R6）。
- **物理攻擊的 HD 來源**：Go 的脈衝工作是前一個工作的值複製，與它共用同一個 `*modernHDBattleSources`，最後一輪以指標去重綁定一次；Rust 每個工作各有一份、逐份綁定。綁定對同一份來源是冪等的（清台座底層、補面板文字），結果相同。

### D 組：預覽（Go `preview := *g`）

`nativeTransientFocusPreview` 與 `preflightNativePhysicalMapReturn` 淺複製整個 `Game`，在副本上聚焦、合成或試建 0x1DB65 演出再丟掉。Rust 的 `Game` 有不能複製的延續與工作欄位，改在 `self` 上就地執行：先以 `save_native_game_preview` 保存預覽路徑實際寫入的欄位（`st`、`native_map_work`、`native_map_vga`、`native_ch28_post_present`、`load_err`、`cur_x/y`、`cam_x/y`、`native_ch23_state`、`native_interface_hud`、`native_map_clock`、`native_map_dac`、`native_fdother_palette_phase/tick`），結束時 `restore_native_transient_preview` 放回。欄位清單來自 `redraw_native_map_hud`、`native_cursor_step_hud`、`sync_native_map_view`、`compose_native_map_frame_*`、`start_native_unit_death_present_mode` 與它發布的第一幀在 Rust 的寫入端；單位視窗素材的延遲載入快取不還原（只是快取）。別組改動這些函式、多寫了欄位時，要同步這份清單。

### D 組：HD 預凍結延後

Go 在忠實模式也會凍結 HD 地圖場景（`freezeNativeCommand9MapInput`、`newModernHDCompoundMapScene` 等），結果只存進 `hdScene`／`hdInput` 一類欄位，只有 HD 繪製讀。這些函式需要擁有資源的 `indexedmap::NativeFrameInput`（Go 存指標，`fd2-campaign` 的型別借用資源）與 `json.Marshal` 摘要，Rust 留到 R6，本體是 `todo!("R6: …")`；D 組在忠實模式的呼叫端（`begin_native_death_reward_message`）不呼叫，`hd_scene` 是 `None`。`freeze_modern_hd_dialogue_chrome` 另外移植了：A 組 HD 標題選單以 `?` 呼叫它，錯誤會影響流程。它要的金色素材 `modern_hd_gold_dialogue.png`（310 位元組、由 SVG 產生，不是原版美術）寫成位元組常數，單元測試核對 SHA-256。

物理攻擊的 HD 來源（`modern_hd_battle_{surface,scroll,runtime,panel}` 的 D 組函式）照 Go 移植：它們在忠實模式也跑，台座位置檢查等會讓整個演出失敗即關閉。

### D 組：呈現回報

Go 由 `Draw` 設下列旗標，對應的 `step*` 等旗標才前進；Rust 的流程端都不設，階段 3 的呈現回報介面要涵蓋（無畫面驅動端直接回報）。各 `draw*` 回傳 false 時 Go `Draw` 呼叫對應的 `fail*`／`cancel*`（`main.go:9502`..`9698`，A 組表第 7 項）。

| 旗標 | Go 寫入端（檔:行） | 等它的流程 |
|---|---|---|
| `NativeCommand0PresentationJob::drawn` | `native_command0_presentation.go:537`、`544` | `step_native_command0_presentation` |
| `NativeCommand1…8` 的 `drawn` | command1 `518`、`525`；command2 `580`、`587`；command3 `499`、`506`；command5 `568`、`575`；command6 `551`、`558`；command7 `502`、`509`；command8 `441`、`448` | 各自的 step |
| `NativeCommand9AIPresentationJob::drawn`、`NativeCommand9PlayerJob::drawn` | `native_command9_ai_presentation.go:447`、`454`；`native_command9_player_presentation.go:262`、`298` | 各自的 step（player 的 hold 階段不等旗標） |
| `NativeCommand1012Job::drawn` | `native_command10_12_presentation.go:321`、`344` | main 與 result 階段；prelude 階段等 `native2189_a` 結束 |
| `NativeCommand24／29PresentationJob::drawn` | command24 `475`、`486`、`496`、`535`；command29 `519`、`530`、`541`、`583` | 各自的 step |
| `NativeCommand32…35PresentationJob::drawn` | command32 `514`、`520`；command33 `479`、`485`；command34 `528`、`534`；command35 `545`、`551` | 各自的 step |
| `NativeCommandHealPresentationJob::drawn`、`NativeCommandModifierPresentationJob::drawn` | `native_command_heal_presentation.go:474`、`493`；`native_command_modifier_presentation.go:127`、`136` | 各自的 step（補血只在畫格階段） |
| `NativePhysicalScene::prelude_drawn` | `native_physical_scene.go:383`、`387` | A 組 `step_attack_presentation_tick` 的前奏 9 幀 |
| `NativePhysicalScene::lead_drawn` | `native_physical_scene.go:404`、`408` | 同上，出發與捲動（第一章沒有） |
| `NativePhysicalBodyPlayback::drawn` | `native_physical_body.go:430`、`452` | `step_native_physical_body_millis`：每個有畫面的工作要一次回報才累積等待時間 |
| `drawNativePhysicalPrelude` 寫的 `loadErr` | `native_physical_scene.go:380`、`391`、`397`、`401`、`412` | HD 前奏／本體畫不出來；流程端要收到失敗才不卡住 |

### D 組：虛擬時鐘與輸入

- D 組沒有讀牆鐘也沒有讀鍵盤：物理演出以 A 組 `step_attack_presentation_tick` 傳入的顯示 tick 毫秒推進，指令演出依呈現回報逐格推進，`native_delay_ticks(500)` 的停留也是幀數。`battle_fpt` 讀環境變數 `FD2_BATTLE_FPT`（同 Go）。
- 升級對話的推進鍵（`step_native_level_up_dialogue(advance)`）由呼叫端從 `tick_input` 組出。

### D 組：共用變更需求

本工作樹已以最小改動先做（主代理整合時決定採用）：

| 項目 | 原因 | 寫法 |
|---|---|---|
| `main.rs` 尾端 `mod main_d_test;` 三行 | D 組 `main.go` 函式的測試 | 緊鄰 B 組的 `mod main_b_test;` |
| 6 個原本「等 D 合併」的測試拿掉 `ignore`；`test_native_story_advance_accepts_escape_like_enter` 改標實際停點（B 組 R8 的 `dlgWrap`） | `--ignored` 跑過 | 只改屬性行 |

需要主代理或別組決定：

| 項目 | 原因 | 建議 |
|---|---|---|
| A 組 `step_attack_presentation_tick` 呼叫 `step_native_physical_body_millis` 前把 `native_scene` 取出 | 沒有原生地圖素材時本體直接 `finish_attack_presentation`；Go 那時看得到 scene（清 VGA、停物理音效），Rust 看不到。第一章有原生地圖素材，走 DAC 漸變路徑不受影響 | 把 step 改成讀寫 `self.atk.native_scene`（不收 `scene` 參數），或在 A 的呼叫端呼叫完再判斷收尾 |
| `begin_native_transient_phases` 的 `selectors` 維持骨架的 `&mut [u8]` | 只讀，但 C 組兩個呼叫端傳 `&mut [..]`，改成 `&` 會讓 clippy 抓 `unnecessary_mut_passed` | 整合時一起改成 `&[u8]` |
| `NativeTransientPreviewSaved` 的欄位清單 | 見「D 組：預覽」 | 別組改到預覽路徑的寫入端時同步 |
| HD 凍結來源需要擁有資源的 `NativeFrameInput` | 見「D 組：HD 預凍結延後」 | R6 由 `fd2-game` 存資源本身、凍結時記下重組所需的小型描述（同 C 組 `native_map_frame_source` 的建議） |

### D 組：KNOWN_DIFFERENCES

| 名稱 | 差異 | 影響 |
|---|---|---|
| `death-flag-map-order` | Go `range` map 隨機，Rust 依鍵升冪（`nativeDeathFlagPrelude`、`freezeModernHDDialogueChrome` 的格子檢查） | 只在多筆同時不合法時回報的項目不同 |
| `transient-preview-in-place` | 預覽在 `self` 上就地執行再還原，不複製 `Game` | 只要保存清單完整，結果相同；單位視窗素材快取會留下 |
| `hd-preflight-scene-deferred` | Go 在忠實模式也凍結 HD 地圖場景，Rust 不凍結（R6） | 結果只給 HD 繪製讀；忠實模式流程不受影響 |
| `panel-text-identity` | Go 以 `*modernBattlePanelText` 指標相等辨認同一塊面板，Rust 比對原點、名稱、數值、外框、字型與原生像素 | 同一場景內兩者等價 |
| `mode5-audio-after-apply` | Go 在 `ApplyNativeAIMode5EventWithAudioCue` 的回呼裡直接 `playSFX`，Rust 套用完再依序播放 | 只在套用失敗時 Go 已先播了音效 |
| `reward-message-owner-take` | 掉落訊息的關框延續在執行時才從 `nativeSystemEndTurnUI` 取出訊息（Go 捕捉指標） | 關框期間沒有別的寫入端替換這個欄位 |
| `transient-candidate-deep-clone` | Go 的候選是淺複製 `State` 再逐筆複製單位，Rust 深複製整個 `State` | 只在 Go 經共用切片在失敗路徑改到 `g.st` 時不同；目前的寫入端都寫候選自己的單位與記錄 |
| `shot-ai-log-text` | `FD2_SHOT_AI` 的診斷以 `eprintln!` 輸出，沒有時間前綴；`%v`／`%+v` 改成 Rust 的 `{:?}` | 只影響診斷輸出 |
| `gold-tile-embedded-bytes` | Go `//go:embed` 的 PNG 在 Rust 是位元組常數 | SHA-256 由單元測試核對 |
| 分身組的 `presentation-units-from-st`、`heal-rollback-by-index` | 見上；command0～9_ai 的 `fail*` 遇到 `None` 單位時略過（Go 解參考 nil 會 panic），command10_12、24、29、9_player 以 `expect` 對應 | 正式路徑的工作只在戰場上建立 |

`NORMALIZATIONS`：無（D 組沒有黃金差分）。

### D 組：測試

`tools/rust_game_test.sh` 同樣的檢查（`FD2_REQUIRE_ORIGINAL=1`，輸出沒有「略過」）：整個 crate 266 通過、8 個 `ignore`（等 E 5 個、黃金腳本 2 個、B 組 R8 1 個）。D 組新增 60 個測試，全部通過：

| 檔 | 測試 |
|---|---|
| `native_command{1,2,3,5,6,7,8,9_ai,9_player,10_12,33,34,35}_presentation/*_test.rs`（分身） | 31 個 Go 測試；共用夾具在 `native_command1_presentation_test.rs` |
| `main/main_d_test.rs` | `atk_animation_test.go` 4 個（`:12`、`:38`、`:199`、`:257`）、`phase_banner_test.go:13` 的長度部分、`native_death_program_test.go:189`；補充 4 個（橫幅幀數與截圖模式、`awardDeathReward` 一次性與死亡程式登記、`battleFPT`／`figaniIndex`、`resetAllNativeMapPoses`） |
| `native_level_up_dialogue/native_level_up_dialogue_test.rs` | `native_level_up_dialogue_test.go` 3 個、`native_transient_phase_test.go:311`；補充 2 個 |
| `native_physical_scene/native_physical_scene_test.rs` | `native_physical_scene_test.go` 的 `:61`（真實分離素材包建第八章 BG55／TAI55 場景）、`:79`、`:100`、`:156` 與 `:183` 的 player 分支 |
| `native_physical_body/native_physical_body_test.rs` | `native_physical_body_test.go:30`（多擊、反擊、狀態與暴擊脈衝、最後還原畫格）；補充 1 個（`step_native_physical_body_millis` 等回報與餘數） |
| `native_attack_assets/native_attack_assets_test.rs` | `native_attack_assets_test.go:13`、`:101`（子行程換空素材包）；補充 1 個 |
| `modern_hd_dialogue_chrome/modern_hd_dialogue_chrome_test.rs` | 嵌入素材 SHA-256、取色規則與拒收（`modern_hd_dialogue_chrome_test.go:174` 的來源拒收段） |

改環境變數的 Go 測試（`FD2_BATTLE_FPT`、`FD2_ASSET_PACK`）以 F 組的 `f_child_env` 在子行程執行。

原本等 D 的測試：`finishSelectedWait` 4 個、`reset_action_overlay_lifecycle_clears_end_turn_owner_and_ring`、`test_beat_sequence_end_triggers_node_transition` 通過，拿掉 ignore；`test_native_story_advance_accepts_escape_like_enter` 現在停在 B 組 `dlg_page_count` 的 R8 `dlgWrap`，改標那個停點。

R5 測試清單（需要 `Draw`、GPU 圖片或畫面用的 `r8` 函式，不在本階段移植）：

| 測試 | Go 檔:行 | 原因 |
|---|---|---|
| `TestPhaseBannerScheduleFollowsTheBiosTickContract` 的逐幀部分、`TestPhaseBannerHoldKeepsTheLastEnterStep` | `phase_banner_test.go:13`、`:86` | `phaseBannerStep`／`phaseBannerSlideOffset` 只給繪圖用（`r8`） |
| `phase_banner_glyph_test.go` 5 個 | `:14`、`:93`、`:126`、`:161`、`:186` | `blitPhaseBannerGlyphs` 是繪圖（`r8`） |
| `TestFIGANIFrameAtDisplayTickUsesNativeDelays`、`TestNativeImpactDACRequiresRawFrameAndDamageProvenance`、`TestBattleImpactHPCommitsPostHitValueAtImpactBoundary`、`TestNativeImpactDisplacementPreserves2939DPhaseAndDirection`、`TestAttackPresentationTickFollowsTheBiosTick`、`TestAttackPresentationTicksKeepsTheRemainder`、`TestAttackPresentationRunsBothStages` | `atk_animation_test.go:55`、`:69`、`:91`、`:104`、`:127`、`:149`、`:287` | 受測的是 A 組 `main.go` 的演出推進與繪圖輔助，由 A 組或階段 3 移植 |
| `TestNativePhysicalBodyGPU`、`TestNativePhysicalReturnClearsVGA`、`TestNativePhysicalReturnWaitSixBIOSTicks` | `native_physical_body_test.go:349`、`:769`、`:818` | 以 `Draw` 推進 |
| `TestNativePhysicalDepartureGPU`、`TestNativePhysicalPreludeGPU`、`TestNativePhysicalOwnerMapHandoffFromOracle` | `native_physical_scene_test.go:586`、`:723`、`native_physical_owner_test.go:27` | GPU 圖片或 `Draw` |
| `TestNativeDeathRewardDiscardCommitsOnlyAfterItemPanelClose`、`TestNativeDefeatRequiresDrawAndFixedHolds`、`TestNativeDefeatChapter12ReturnsTitleAndStartResetsCampaign` | `native_death_reward_ui_test.go:155`、`native_defeat_test.go:53`、`:126` | 以 `Draw` 推進（另需 R8 的滿欄提示與敗北演出） |
| HD 的 GPU 測試 | `modern_hd_battle_surface_test.go:142`、`:568`、`modern_hd_battle_camera_gpu_test.go:89`、`modern_hd_battle_scroll_test.go:350`、`:444`、`modern_hd_battle_scroll_lifecycle_test.go:20`、`modern_hd_battle_panel_test.go:238`、`:353`、`modern_hd_battle_panel_style_prototype_test.go:76`、`modern_hd_player9_map_gpu_test.go:287`、`:350`、`modern_hd_player9_stage_gpu_test.go:151`、`modern_hd_transient_expiry_test.go:117`、`:457`、`modern_hd_indexed_transition_gpu_test.go:277`、`modern_hd_commands10_12_map_gpu_test.go:208`、`modern_hd_dialogue_chrome_layer_test.go:152`、`modern_hd_gold_dialogue_test.go:287` | GPU 圖片（同時是 R6 的 HD） |
| 分身組列的指令演出測試 | `native_command3_presentation_test.go:12`、`native_command6_presentation_test.go:163`、`:955`、`native_command29_presentation_test.go:13`、`native_command10_12_presentation_test.go:17`、`:69` | 以 `Draw` 推進（另需 `r8` 的 confirm 分支或 AI 執行路徑） |

其他沒有移植的 D 組主題 Go 測試（不需要畫面，所以不列 R5）：

- 受測或夾具是 `r8` 函式：`native_transient_phase_test.go` 的 `:14`..`:138`、`:151`、`:272`、`:290`、`:358`（夾具 `nativeCurrentSaveTestGame` 經 `buildNativeCurrentSaveStored`，受測的 `applyNativeTransientPhase(s)` 是 `r8`）；`native_command24_presentation_test.go` 8 個、`native_command29_presentation_test.go` 4 個、`TestStartNativeCommand33／34／35…`（start 函式是 `r8`）；`native_ai_physical_atomic_test.go` 2 個（mode 11、AI 指令 0）；`TestNativePhysicalSceneMissingBGStopsPlayerAndMode11BeforeSettlement`、`:156`／`:183` 的 mode11 分支、`TestNativePhysicalDepartureMissingLayerStopsBeforeSettlement`（`executeNativeAIMode11Physical`、出發段是 `r8`）；`native_death_reward_ui_test.go` 其餘 3 個（滿欄丟棄提示）；`native_defeat_test.go:18`、`:113`、`native_defeat_parity_test.go:62`（敗北演出）；`native_death_program_test.go:129`、`:165`（死亡程式的 `staging`）。
- 原版 oracle 或整段流程（階段 3）：`native_physical_body_test.go` 的 `NormalOracle`／`MapReturnCompleteFrameFromOracle` 7 個與 `:107`、`native_physical_scene_test.go:356`、`:425`、`native_action_death_present_test.go` 7 個、`native_death_program_test.go:83`、`:228`、`TestPlayerPhysicalAttackPreflightsSeparatedPairBeforeMutation`、`…LoadsDynamicSeparatedPairWithoutArchive`（`native_attack_assets_test.go:36`、`:68`，經 `confirm` 走整段玩家攻擊並比對 Go 亂數序列）、`TestCompatiblePhysicalReturnKeepsOwnVGA`（`native_physical_body_test.go:806`，受測的 `finishAttackPresentation` 是 A 組）、`native_transient_phase_test.go:319`。
- HD（R6）：`modern_hd_*_test.go` 其餘非 GPU 的測試（battle surface 5、physical sfx 4、battle camera 5、battle scroll 4、battle panel 4、compound map 2、player9 map 4、player9 stage 3、transient expiry 2、indexed transition 7、commands10_12 map 2、dialogue chrome 1、gold dialogue 2），都經 HD 主題、HD 凍結來源或 HD 繪製。

黃金差分：D 組沒有加。可以逐幀比對的是指令演出與物理演出的 step，但它們要等呈現回報；適合在階段 3 以收 `drawn` 回報的驅動端或重播檢查點比對。

### D 組：fd2-rules／fd2-campaign 缺口

沒有改那兩個 crate，也沒有發現缺少的函式。`battle::Unit`／`State`、`fdother::Frame` 有私有欄位，測試以 `build!` 巨集先 `default()` 再寫入。`Cargo.lock` 沒有變動。

## R4 階段 2：E 組（城鎮、商店、教會與整備）

2026-11-15，代號 `r4e`，合併基準 `57e34259`。範圍是 `PORTING.json` 裡 `group` 為 `E` 的 70 個函式（含 `main.go` 的
`setupPreparation`、`campInput`、`leaveHotel`），另加主代理指派的 A 組 `draw_modern_battle_panel_ink`。

### E 組：移植結果

| 項目 | 數量 |
|---|---|
| E 組函式 | 70 個全部有本體；另隨呼叫端補移植 9 個 `r8` 小函式（下表） |
| 整個本體是 `todo!()` 的 | 0 個 |
| 分支停在 `todo!()` 的 | R8 12 處、R6 5 處（見下） |

R8 的 `todo!` 都在第一章重播沒有執行到的分支（覆蓋率取自 `work/r4-stage0-20261114/cover.out` 的區塊計數），呼叫的 Go 函式在
`PORTING.json` 標 `r8`：

- `camp_input`：教會選單以下的全部輸入（`handleNativeChurch*Input`、`returnToNativeChurchMenu`，`main.go:5121`）與商店輸入
  （`handleNativeShopInput` 與相容購買／販賣路徑，`main.go:5251`）。城鎮、劇情、選項、整備、酒店、結局與戰鬥分支照 Go 移植。
- `handle_native_preparation_input`：獨立整備的記錄槽、必要角色提示、出擊名單確認、出發（`acceptTownDeparturePrompt` →
  `finishNativePreparationDeparture`）、選人的 Enter 與獨立整備的 ESC，共 8 處。第一章執行到的城鎮出口提示（左右換選項、
  ESC 或「否」收框回城鎮）與選人的方向鍵已移植。
- `handle_native_hotel_input`：讀檔槽的 Enter（`confirmNativeHotelLoadSlot`）。
- `step_native_shop_ui_lifecycle`：購買後的裝備面板（`stepNativeShopEquipPanelLifecycle`）。

R6 的 `todo!` 都是 HD 文字擷取：`modern_hd_facility_font`（以 `WithNativePresenter` 換上記錄 HD 文字的 presenter）、
`begin_modern_hd_facility_text`、`capture_modern_facility_list_text`、`begin_modern_hd_shop_digits`、`modern_hd_decimal_sprite`
（`TextPlaceholder` 的繪字回呼）。Go 的 presenter 與佔位回呼在繪字時寫回 `g.font.indexedHD`，Rust 的 `fdtxt::NativePresenter`
是 `Arc<dyn Fn>`、佔位回呼要 `'static`，要寫回 `Game` 須改成共享的擷取狀態，屬於 R6 的 HD 文字層設計。各函式在 `todo!` 之前照 Go
移植退出條件（`modernHDText`、`font`、`nativeClassUI` 為 nil 等）。

隨呼叫端補移植的 `r8` 函式（本體照 Go）：

| Go | Rust | 呼叫端 |
|---|---|---|
| `native_town_secret_input.go:nativeBIOSFunctionScan` 與 `nativeFunction*` 常數 | `native_bios_function_scan`、`NATIVE_FUNCTION_*` | `pressed_native_town_secret_scan` |
| `native_town_ui.go:captureNativeTownCaller`、`revealNativeTownSecret` | `Game::capture_native_town_caller`、`Game::reveal_native_town_secret` | `camp_input` 的城鎮分支 |
| `native_shop_ui.go:stepNativeShopUIPulseTick` | `Game::step_native_shop_ui_pulse_tick` | `step_native_shop_ui_lifecycle` |
| `native_church_ui.go:stepNativeChurchUIPulseTick` | `Game::step_native_church_ui_pulse_tick` | `step_native_church_ui_lifecycle` |
| `native_preparation_ui.go:stepNativePreparationCycleTick` | `Game::step_native_preparation_cycle_tick` | `step_native_preparation_ui_lifecycle` |
| `native_hotel_ui.go:currentNativeHotelInput` | `current_native_hotel_input`（改收 `&TickInput`） | `camp_input` 的酒店分支 |
| `main.go:applyHotelServiceSelection`、`stepCampaignMenu` | `Game::apply_hotel_service_selection`、`Game::step_campaign_menu`（`main.rs`，緊鄰 `leave_hotel`／`camp_input`） | `camp_input`、`handle_native_hotel_input` |

`native_church_ui.go:nativeChurchUIBlocksInput`（`r8`）只是 `g.nativeChurchUIJob != nil`，直接寫在 `camp_input`。Rust 新增、沒有 Go
對應的項目：`native_preparation_input.rs` 的 `leave_preparation`（Go 的區域閉包）與 `Game::close_preparation_prompt_then`
（Go 的區域閉包 `closeThen`）、`modern_hd_facility_dialogue.rs` 的 `facility_dialogue_key`。

### E 組：虛擬時鐘、輸入與呈現回報

- 城鎮、商店、教會、轉職／確認框與整備選人的脈衝都由 `NativeBIOSClock::sample(now)` 換算 BIOS tick，`now` 來自 A 組的
  `game_now()`（虛擬時鐘）；城鎮的 tick 由 A 組 `update` 傳入 `step_native_town_ui_lifecycle`。E 組沒有讀牆鐘。
- 輸入一律讀 `tick_input`：`camp_input` 開頭複製一份本幀輸入；`native_modifier_held`、`pressed_native_town_secret_scan`、
  `current_native_hotel_input` 改收 `&TickInput`（見簽名變更紀錄）。
- 下列旗標在 Go 由 `Draw` 設定，對應的 `step*` 等旗標才前進；Rust 的流程端不設，階段 3 的呈現回報要涵蓋：

| 旗標 | Go 寫入端 | 等它的流程 |
|---|---|---|
| `NativeClassUIJob::drawn`（整備提示、轉職／復活確認、商店確認等共用工作） | `drawNativeClassUIJob`（`native_class_ui_lifecycle.go:179`） | `step_native_class_ui_lifecycle`：開框、收框每格一次回報，收框另要還原畫面一次回報才呼叫延續 |
| `Game::native_shop_ui_job` 的 `drawn` | `drawNativeShopUIJob`（`native_shop_ui.go:1495`） | `step_native_shop_ui_lifecycle` |
| `NativeChurchUIJob::drawn` | `drawNativeChurchUIJob`（`native_church_ui.go:214`） | `step_native_church_ui_lifecycle` |
| `NativeTownDepartureJob::drawn` | `drawNativeTownDeparture`（`native_town_departure.go:86`） | `step_native_town_departure` |

E 組的測試照 Go 測試直接寫 `drawn = true`，等同回報。

### E 組：忠實模式下的 HD 預檢

Go 的 `bindModernFacilityServiceMenu`（商店、教會服務選單的開框與收框）只看 `g.font.sf != nil`，與 HD 介面開關無關：載入 TTF 後，
忠實模式也會預先建立 HD 描述，過程以 HD 文字擷取重組畫面（上面的 R6 函式）。這份描述只有 HD 繪製
（`drawModernHDFacilityServiceMenu`，R6）與輸入稽核的 `lastDraw`（只由 HD 繪製寫入）讀取，不影響節點、交易或延續。Rust 照 Go 移植
全部前置檢查，在 HD 介面關閉時於前置檢查之後回 `None`（KNOWN_DIFFERENCES `facility-hd-precompute-faithful-skip`），不進入要 TTF
擷取的部分；HD 介面開啟時照 Go 往下做，停在 R6 的 `todo!`。要主代理確認這個作法；若要逐位元組保留 Go 的預檢，需要 R6 的擷取狀態。

`snapshot_native_preparation_transition`（整備提示的 HD 配方）不需要 TTF，忠實模式照 Go 建立。

### E 組：HD 面板文字的字級量測（主代理指派）

C 組回報：`prepareNativePlayerMovement` 與物理演出在字型有 TTF、`localeEntities` 存在時收集 HD 面板文字，`renderBattlePanelText` 以
`raster` 預檢（`modern_hd_battle_panel.go:114`），預檢失敗會讓流程回錯誤；`raster` 呼叫 `drawModernBattlePanelInk`。E 組把
`draw_modern_battle_panel_ink`（A 組檔，本體原是 `todo!("R6")`）改成：

- 照 Go `modern_hd_battle_panel.go:224` 逐行移植字級搜尋：`int(f.effectiveScale(maximum))` 向零截斷，從大到小試到 24，
  `faceFor` 取不到 face 回 `modern HD panel: font unavailable`，找不到不超框（`Dx+2`、`Dy+2`）的字級回
  `modern HD panel: text exceeds safe rectangle: %q`。
- 量寬經 `Font::measure` 的 `bound_string`。Go 這裡用 `font.BoundString`（x/image v0.12.0 `font/font.go:245`）再取
  `Floor`／`Ceil`，量測器的契約是 Ebiten `text.BoundString`（v2.6.6 `text/text.go:281`）。兩者對單行字串的聯集相同：
  `fixed.Rectangle26_6.Union` 本身略過空矩形，x/image 另以 `!b.Empty()` 略過；只在字串含 `\n` 或全是空白字形時不同，面板字串
  （名稱、`LV·`、`HP`、`MP` 與數字）都不是（強推論，依兩份原始碼）。
- 找到字級之後的點陣化（陰影與前景兩次 `font.Drawer.DrawString`）是 R5。Rust 不畫字、直接回成功，不寫成 `todo!`，否則忠實模式照樣
  停住；`raster` 的畫布與 `key` 因此沒有文字墨跡（KNOWN_DIFFERENCES `panel-ink-measure-only`，只有 HD 繪製讀它們）。
- 沒有注入量測器卻有 TTF 時停在 `todo!("R5: …")`，與 A 組 `Font::width` 相同。

### E 組：交易

第一章重播只執行到城鎮選單、出口提示、商店與教會服務選單的開框，以及酒店選單；購買、販賣、裝備、轉移、復活與轉職的交易函式都標
`r8`（`handleNativeShopInput`、`handleNativeChurch*Input` 等），R4 沒有移植，也沒有重寫規則層。R8 移植這些分支時呼叫 `fd2_campaign`
既有的 `reserve_good`、`finalize_good`、`sell_slot`、`equip_item`、`recompute_equipment`、`revive_unit`、`native_class_change_target`。

### E 組：共用變更需求

本工作樹沒有改 `Game`、`support.rs` 或別組的型別。動到別組檔的地方（都只限必要的行）：

| 檔 | 改動 | 原因 |
|---|---|---|
| `main.rs`（A 組 `update_frame`） | 四處 `native_modifier_held()` → `native_modifier_held(&input)` | 簽名改收本幀輸入 |
| `player_poster_controls.rs`（A） | `native_modifier_held(&self.tick_input)` | 同上 |
| `modern_hd_battle_panel.rs`（A） | `draw_modern_battle_panel_ink` 本體與檔尾 `mod modern_hd_battle_panel_ink_test;` | 主代理指派 |
| `title/title_test.rs`、`main/main_a_test.rs` | 拿掉三個「等 E」的 ignore 屬性行 | `--ignored` 跑過通過 |
| `save/save_test.rs` | 兩個「等 E」改標「等 D 合併（停在 cancel_native_command32_presentation）」 | 實際停點 |

需要主代理決定：

| 項目 | 原因 | 建議 |
|---|---|---|
| `facility-hd-precompute-faithful-skip` | 見「忠實模式下的 HD 預檢」 | 維持到 R6；R6 建立共享擷取狀態後拿掉 HD 開關的提前返回 |
| `NativeServiceMenuPresentation::shop_job`／`church_job` | Go 存工作指標，HD 繪製以指標比對身分；Rust 的工作由 `Game` 擁有，欄位留 `None` | R6 改存工作的世代號或比對 `raw_frames` |

### E 組：KNOWN_DIFFERENCES

| 名稱 | 差異 | 影響 |
|---|---|---|
| `preparation-roster-map-order` | `setupPreparation` 以 `range g.partyRoster`（隨機）補上不在 JOIN 順序裡的成員；Rust 依 ID 排序 | 只在名冊有 JOIN 順序以外的成員時影響選人順序；Go 本身不可重現 |
| `facility-hd-precompute-faithful-skip` | HD 介面關閉時不建立服務選單的 HD 描述 | 只給 HD 繪製讀 |
| `panel-ink-measure-only` | HD 面板文字只量字級、不畫字 | 只給 HD 繪製讀；預檢的成敗與 Go 相同 |
| `facility-dialogue-key-bytes` | `prepareModernHDFacilityDialogue` 的快取鍵是自訂位元組序列的 SHA-256，不是 Go `json.Marshal` 的 | 鍵只和自己比對 |
| `facility-dialogue-font-identity` | Go 以 `h.font == g.font` 比字型指標，Rust 以 `same_panel_font` | 同 C 組 `panel-font-identity` |
| `shop-assets-map-order` | `loadNativeShopUIAssets` 以 `range` map 載入三個變體，Rust 依變體排序 | 只在同時有多個缺件時回報的項目不同 |
| `town-source-value-copies` | `nativeTownFrameSource` 的 `strings`／`font` 在 Go 是指標，Rust 存複本 | 素材載入後不再改寫 |

`NORMALIZATIONS`：無（E 組沒有黃金差分）。

### E 組：測試

`tools/rust_game_test.sh`（`FD2_REQUIRE_ORIGINAL=1`，輸出沒有「略過」）：全 crate 234 通過、13 ignore（含 2 個由黃金腳本執行）。
E 組新增 33 個測試（Go 移植 27、Rust 補充 6），31 通過、2 個 `ignore`；另外原本等 E 的 5 個：3 個通過、拿掉 ignore，2 個改標等 D。

| 檔 | Go 來源 | 測試 |
|---|---|---|
| `native_class_ui_lifecycle/native_class_ui_lifecycle_test.rs` | `native_class_ui_lifecycle_test.go` 7 個 | 7 |
| `native_preparation_ui/native_preparation_cycle_test.rs` | `native_preparation_cycle_test.go` 6 個 | 6 |
| `native_town_ui/native_town_ui_test.rs` | `native_town_ui_test.go` 5 個 | 5 |
| `native_church_ui/native_church_ui_test.rs` | `native_church_ui_test.go` 3 個 | 3 |
| `native_shop_ui/native_shop_ui_test.rs` | `native_shop_ui_test.go:697`、`:1100` | 2 |
| `native_class_ui/native_class_ui_test.rs` | `native_class_ui_test.go:202` | 1 |
| `native_hotel_ui/native_hotel_ui_test.rs` | `hotel_trace_test.go:18` | 1（ignore） |
| `native_town_secret_input/native_town_secret_input_test.rs` | `native_town_secret_input_test.go:5` | 1 個 Go、2 個補充（`TickInput` 修飾鍵與 F 鍵） |
| `native_preparation_input/native_preparation_input_test.rs` | `preparation_test.go:95`（部分：中段的 `restartPreparationSelection` 是 `r8`） | 1 個 Go、2 個補充（`setup_preparation` 的順序與名額；城鎮 → 出口提示 → ESC 回城鎮整段，ignore） |
| `modern_hd_battle_panel/modern_hd_battle_panel_ink_test.rs` | 無（Go 的面板測試都以真 TTF 繪字） | 2 個補充（假量測器驗字級搜尋與兩個錯誤） |

| ignore | 等 | 停點 |
|---|---|---|
| `test_campaign_town_hotel_raw_route_return_trace`、`town_exit_prompt_escape_returns_to_same_town_selector` | D | `enter_node` 經 `reset_action_overlay_lifecycle` 停在 `cancel_native_command32_presentation`；本地把 D 的四個 `cancel_native_command3x_presentation` 暫改成「沒有工作就返回」後兩個都通過（只是驗證，已還原） |
| `save_test.rs` 的 `campaign_save_load_restores_town_boundary_and_party`、`title_json_load_accepts_saved_zero_chapter_and_keeps_bytes` | D | 同上；同一個暫改下兩個都通過 |

Go 以 `t.Setenv("FD2_ASSET_PACK", …)` 指向分離素材包的測試，Rust 沿用 `tools/rust_game_test.sh` 對整個測試行程設定的同一個變數；
`FD2_ORIGINAL_*` 只影響 Go 的封存檔退路，Rust 的載入器只讀分離素材包，所以 Go 以「封存檔不存在」證明不讀封存檔的部分自然成立。

R5 測試清單（需要 `Draw`、GPU 圖片或真 TTF 量寬／繪字，不在本階段移植）：

| 測試 | Go 檔:行 | 原因 |
|---|---|---|
| `TestNativeTownDeparturePreflightAndPresentBoundary`、`TestNativeTownDepartureFromOracle` | `native_town_departure_test.go:22`、`:151` | 以 `Draw` 推進（另需 `r8` 的 `startNativeTownDeparture`；後者是原版對照） |
| `TestDrawNativeClassListUsesPlayerOriginalAssets` | `native_class_ui_test.go:13` | 斷言 `Draw` 的畫面（另走轉職、復活的 `r8` 函式） |
| `TestNativeChurchRuntimeUsesPlayerOriginalSceneAssets` | `native_church_ui_test.go:156` | `drawNativeChurchMenu` |
| `TestNativeShopProductionOwnerDrawsOriginalMenuAndPurchaseList` | `native_shop_ui_test.go:20` | 以 `Draw` 推進（另走購買、販賣、裝備、轉移的 `r8` 函式） |
| `TestOfficialShopItemNamesFitNativeSafeRectangle`、`TestOfficialCharacterNamesFitNativeRosterRectangle`、`TestOfficialBattleNamesFitNativePanelRectangle`、`TestOfficialCommandNamesFitNativeGridCell`、`TestOfficialClassNamesFitClassChangeTargetRectangle`、`TestOfficialShopMessagesFitNativeDialogueRectangle` | `native_shop_locale_test.go:10`、`:41`、`:74`、`:109`、`:141`、`:163` | 以真 TTF 量寬核對四語系譯名與商店訊息 |
| `TestModernHDTextRequiresCompleteHDAndKeepsLogicalMetrics` | `modern_hd_text_test.go:10` | `Draw` |
| `TestModernHDFacilityDialogueRuntimeGPU`、`TestModernHDServiceMenuSourceAndFallback` | `modern_hd_facility_dialogue_test.go:166`、`modern_hd_service_menu_transition_test.go:31` | GPU 圖片或 HD 繪製（R6） |
| `TestModernHDPreparationTransitionRecipesMatchNativeFrames`、`TestModernHDPreparationBackgroundFrozenSourceAndFallback`、`TestModernHDPreparationTransitionInvalidRecipeKeepsNativeJob`、`TestModernHDPreparationTownBackgroundRuntimeGPU`、`TestModernHDPreparationDialogueRuntimeGPU`、`TestModernHDPreparationCursorRuntimeGPU` | `modern_hd_preparation_transition_test.go:19`、`:145`、`:619`、`:354`、`:361`、`modern_hd_preparation_cursor_test.go:266` | `Draw` 或 GPU（R6） |

其他沒有移植的 E 組主題 Go 測試（不需要畫面，所以不列 R5）：

- 受測函式是 `r8`：`native_town_ui_test.go:188`、`native_church_ui_test.go:14`、`native_shop_ui_test.go:785`、`:862`、`:940`、`:1027`
  （截圖狀態）；`native_church_ui_test.go:189`（`handleNativeChurchMenuInput`）；`native_preparation_ui_test.go` 4 個（記錄槽、
  `composeNativePreparationFrame`）；`preparation_test.go` 其餘 4 個（`acceptTownDeparturePrompt`、`restartPreparationSelection`、
  `togglePreparationSelection`）；`hotel_trace_test.go:62`、`:89`、`:112`、`:140`（`confirmNativeHotelLoadSlot`）；
  `modern_hd_facility_dialogue_test.go:83`、`modern_hd_preparation_transition_test.go:514`、`:576`、`modern_hd_preparation_cursor_test.go:273`
  （HD 的 `r8` 函式）。
- 別組主題或整段流程：`hotel_trace_test.go:172`（傳聞劇情）、`:209`（同源探針）、`native_shop_locale_test.go:248`（A 組語系目錄）、
  `native_church_*_path_test.go`、`native_shop_*_path_test.go`、`town_shop_early_mid_late_e2_test.go`、`native_title_to_town_test.go` 等
  走完整交易或章節的測試，在 R8 或階段 3 的重播移植。

黃金差分：E 組沒有加。城鎮、轉職與教會的脈衝計數、選單換位都是小型純函式，Go 測試的期望值已逐條覆蓋（含有號 BIOS 字繞回）；開框、
收框與延續要呈現回報才前進，適合在階段 3 以重播檢查點比對。

### E 組：fd2-rules／fd2-campaign 缺口

沒有改那兩個 crate。`campaign::compose_native_church_scene` 的 `strings`／`font` 收借用（Go 是可為 nil 的指標），Rust 在 nil 時回
`false`，與 campaign 回錯誤相同。

### 整合紀錄（D＋E，階段 2 收尾，2026-11-15）

- D（`8ff1e1eb`、`3c12fbfe`、`dd99b7cc`）與 E（`3341a052`）合併；只有 `PORTING.md` 衝突，以 D 版為底接上 E 段落，E 的 11 列簽名變更併入表格。程式自動合併後直接通過。
- 4 個「等 D」測試通過並拿掉 ignore。`tools/rust_game_test.sh`：304 通過、3 ignore（2 個存檔黃金由腳本執行；`test_native_story_advance_accepts_escape_like_enter` 停在 B 組 R8 的 `dlgWrap`，第一章不會走到）。
- `PORTING.json` 函式狀態依本體重新判定：骨架範圍 565 個中 553 個 `ported`、2 個 `r5`、10 個 `r6`；另 78 個原標 `r8` 的函式隨呼叫端移植為 `ported`。全套 1922 個：ported 631、r5 2、r6 17、r8 1272。
- crate 層的骨架 allow 暫留：`Game::update` 還沒有公開入口，拿掉會出現約 900 個未使用警告。階段 3 接上無畫面驅動端後清理。
- 主代理的決定：
  - E 組 `draw_modern_battle_panel_ink` 只量字級、不畫字，回成功（`panel-ink-measure-only`），接受；忠實模式流程與 Go 相同。
  - E 組服務選單的 HD 預先描述在 HD 介面關閉時跳過（`facility-hd-precompute-faithful-skip`），接受；只有 HD 繪製與輸入稽核讀它。
  - D 組待決留給階段 3：A 的 `step_attack_presentation_tick` 先取出 `native_scene` 的時機（沒有原生地圖素材時與 Go 不同，第一章有素材不受影響）；`begin_native_transient_phases` 的參數可變性；`NativeTransientPreviewSaved` 的還原清單要隨別組新增欄位同步。

## R4 階段 3：呈現回報與第一章無畫面重播（2026-11-15）

代號 `r4i`，合併基準 `92bb5116`。

### 呈現回報介面

`src/game/present.rs`（Rust 新增，沒有單一 Go 檔對應）。契約寫在模組文件：

- `Game::acknowledge_present() -> PresentOwner`：外殼（`fd2-app`，R5）每畫完一幀呼叫一次；無畫面驅動端在 Go 呼叫
  `g.Draw(screen)` 的地方直接呼叫。它照 Go `Draw` 的分支優先序選出本幀的呈現擁有者，做那個擁有者的 `draw*` 函式裡的
  狀態改寫，回傳擁有者讓外殼畫對應的畫面。不畫像素。
- 前置條件只看狀態（幀索引、緩衝長度、調色盤長度），與 Go `draw*` 的 `return false` 條件逐條相同；不成立時走 Go `Draw`
  的失敗路徑。
- 唯一建立的畫面是 `NativePhysicalBodyPlayback::image`：Go 把它快取在演出工作裡，`stepNativePhysicalBodyMillis` 以「有沒有
  快取」決定換幀（`native_physical_body.go:433`、`:484`），所以照 Go 以 `picture_from_indexed` 建立同一份畫面。
- `begin_present_frame` 是 Go `Draw` 開頭的重設，可單獨呼叫；`Game::draw`（A 組，R5）的開頭與它相同；R5 階段 1 起 `draw` 直接呼叫它。

| A 組表 | Go 位置 | Rust |
|---|---|---|
| 1 | `main.go:9356`..`9368` 每幀重設 | `begin_present_frame` |
| 2 | `main.go:9369` HD 橫幅名稱 | R6（`modern_hd_map_gpu` 存在時 `todo!`） |
| 3 | `main.go:9372` `beginModernHDText` | `begin_present_frame` 清 `modern_hd_text`；HD 介面 R6 |
| 4 | `main.go:9386`..`9388` 白閃 defer | `finish_present_frame`，每條 return 路徑都執行 |
| 5 | `main.go:9390` 說明畫面清 HD 文字 | R6 |
| 6 | `captureShot` | fd2-tools，不移植 |
| 7 | 各 `fail*` | END 回收、AI 閒置回收已移植；其他擁有者的 `fail*` 還是 `r8`，見下表 |
| 8 | 演出工作的 `drawn` | 見下表 |
| 9 | `main.go:9768` | `cancel_native_end_turn_recovery("native END recovery: Draw unavailable")` |
| 10 | `main.go:9779`..`9783` | 復原 range、清 job、`loadErr`、`ai_busy = false`（與 Go 同一則訊息） |
| 11、14 | `storyView`、`dlgGrad` 快取 | 畫面資源，R5 |
| 12、16、17 | 地圖整幀、指令環 `markActionOverlayDrawn`、物理面板失敗 | Go `Draw` 尾段（`main.go:9804`..`10188`），R5 `todo!` |
| 13、15 | HD 對話框外框、HD 覆蓋層 | R6 |

逐分支（Go `Draw` 的 if 鏈）：

| 擁有者 | Go draw 函式 | Rust |
|---|---|---|
| `startupBlocked`、`m == nil` | — | 直接回傳 |
| `nativeDefeat` | `native_defeat.go:160` | `drawn`，有斜坡時 `ramp.drawn` |
| `titlePhase` | `title.go:716` | 沒有流程旗標；HD 文字與截圖不在流程 |
| `indexedTransition` | `native_indexed_transition.go:331` | 非 HD：vga 與 256 色齊全才 `drawn`；HD R6 |
| `nativeTownDeparture` | `native_town_departure.go:86` | 非 HD 一律 `drawn`（越界照 Go panic）；HD 介面開啟時 R6 |
| `nativePaletteRamp` | `native_palette_ramp.go:164` | vga 與調色盤齊全才 `drawn`；HD 物理返回 R6 |
| `nativePalettePulse` | `native_palette_pulse.go:158` | 同上 |
| `spawnIntroTransition` | `native_spawn_intro.go:360` | 同上 |
| `nativeEndTurnRecovery` | `native_end_turn_recovery.go:162` | 幀索引、VGA 長度、768 位元組 DAC 與調色盤可解才 `drawn`；否則取消 |
| `nativeAIIdleRecovery` | `native_ai_idle_recovery.go:328` | 幀索引、素材調色盤 256、VGA 長度；否則失敗收尾 |
| `atk`（有原生場景） | `main.go:11231` → `native_physical_scene.go:372`、`native_physical_body.go:414` | 前奏 `prelude_drawn`、出發與捲動 `lead_drawn`、本體 `p.drawn` 與 `p.image`；cue-only 項目不設；本體錯誤寫 `loadErr`。前奏與本體之後、沒有原生場景的近似畫面 R5；HD R6 |
| 結局、ch20／ch23／2189A／單位登場／ch28／24B4D、補血、修飾、AI 修飾、指令 0～35、AI 物品、indexed 回合演出、事件 61 | 見 `present.rs` 各 `todo!` 的 Go 行號 | R8：對應的 `fail*` 還沒移植，`drawn` 條件各自不同（例如補血的 FrontHold／DigitHold 不設 `drawn`、`drawn` 先於長度檢查），第一章重播沒有執行到 |

涵蓋：開頭重設、結尾 defer 兩項加 12 個分支已移植；`todo!` 共 26 個（R5 3、R6 9、R8 14）。單元測試
`present/present_test.rs` 5 個（重設、斜坡前置條件、END 回收取消、白閃 defer、成長觀察者交給戰場狀態）。

### 無畫面重播驅動端

`src/game/parity_replay.rs`（`#[cfg(test)]`）與子模組 `parity_replay/`，一個 Go 測試檔一個：

| Rust | Go | 函式 |
|---|---|---|
| `chapter_parity_replay_test.rs` | `chapter_parity_replay_test.go` | `TestChapterParityReplay` 與 `parityReplay` 的方法、輔助函式，共 40 個 |
| `chapter_parity_start_test.rs` | `chapter_parity_start_test.go` | `verifyParityStartSource`、`startFromNormalTitle`、`verifyChapterSlotSave` |
| `chapter_parity_hp_lock_test.rs` | `chapter_parity_hp_lock_test.go` | `parityUpdateGame`；`parityPreservesEnemy` 只有無鎖血政策的分支 |
| `chapter_parity_legacy_end_test.rs` | `chapter_parity_legacy_end_test.go` | `recoverParityLegacyENDSources` 只有「沒有缺 `before_seq`」的分支 |
| `native_chapter_one_play_test.rs` | `native_chapter_one_play_test.go` | `pump`、`closeRing`、`ackPresents`、`answerNativeTreasurePrompt`、`ch01Blockers`、`pendingOwn` |
| `native_title_to_town_test.rs` | `native_title_to_town_test.go` | `journeyTrace` 與 `observe`、`milestone`、`press`、`storyEnterReady`、`storySignature`、`progressSignature`、`driveStory`、`resultInputReachable`、`writeJourney` 等 12 個 |

共 64 個函式；`todo!` 16 個（R5 2：畫面擷取；R8 14：其他章、LOAD 入口、鎖血、#115 拒收攻擊、舊 oracle 的 `before_seq`、
第一章沒有的 mark 與動作種類、寶物提示的 `confirmNativeSystemEndTurn`、第二十四章舞台）。

- `Driver` 持有 `Game` 與 `VirtualFrameClock`；Go 每次 `g.Update()` 讀的牆鐘改成在 `update` 前以 `advance_virtual_frame`
  推進一個邏輯幀。
- `ackPresents` 逐項照 Go：Go 呼叫 `g.Draw` 的三處（城鎮出發、有原生場景的攻擊演出、END 回收）改呼叫 `acknowledge_present`，
  其餘照 Go 只設 `drawn`；地圖快取失效時照 Go 呼叫 `compose_native_map_frame`。
- Go `t.Setenv` 的八個變數（`FD2_TITLE`、`FD2_SEED`、`FD2_MUTE`、`FD2_CAMPAIGN`、`FD2_SHOT_AI`、`FD2_NATIVE_SAVE`、
  `FD2_CUTSCENE_LOG`、`XDG_DATA_HOME`）以 F 組的 `f_child_env` 在子行程設定；父行程先把建構槽複製成 `<out>/FD2.SAV`。
- AI 與升級成長觀察者以 `Arc<Mutex<…>>` 共用游標；AI 觀察者分岔時寫 `ai_order` checkpoint，所以 `checkpoints.jsonl` 的
  寫入端也共用。
- `checkpoints.jsonl` 照 Go `encoding/json` 的欄位順序、`omitempty` 與 HTML 跳脫手寫序列化；`replay-settings.json` 與
  `start-entry.json` 照 `json.MarshalIndent`（兩格與一格縮排、map 鍵排序）。
- 驅動端的 `KNOWN_DIFFERENCES`（都不影響這次的輸出檔）：`journey-known-unit-index`（Go 以 `*battle.Unit` 指標記已見單位，
  Rust 記 `st.units` 索引並在節點改變時清空；只影響援軍紀錄 `spawns`，章重播的 `start-entry.json` 在第一戰取得操作權時就寫出）、
  `ch01-blockers-no-reflection`（失敗訊息只列流程欄位）、`debug-ai-log`（Go `battle.DebugAI = log.Printf` 只是記錄，沒有移植）、
  `ai-diagnostics-subset`（`FD2_PARITY_AI_DIAGNOSTICS` 只印入口與亂數）。

### 比對（`tools/rust_parity_ch01.sh`）

`tools/rust_parity_ch01.sh <輸出目錄>`：在 `fd2-rust:1.99.1-r3` 以 `--release` 跑重播（建構槽、原版輸出、Go 黃金唯讀掛載），
以 `tools/rust_parity_compare.py` 比對，再以 `tools/verify_chapter_parity.py`（`--frame-policy advisory --save-policy
ch01-start-residual-20261017 --oracle-save state-full-r4/FD2.SAV`）各判 Rust 與 Go 輸出一次，收據寫到
`work/rust-build/golden/r4i-<輸出目錄名>-{rust,go}.json`。在工作樹副本中執行時設 `FD2_WORK=<主工作區>/work`。

| 輸出 | 結果 |
|---|---|
| `checkpoints.jsonl` | 40 筆，與 Go 黃金逐位元組相同，SHA-256 `1c846f573434658cabb3cfa486edd5826b401673c3bff411e30cbdc3e18ac0cc` |
| `FD2.SAV` | 逐位元組相同，SHA-256 `f3942ff2288a76d1c736a3a6f6ff8b7661efea0fc9870c323bc49da8805f1860` |
| `start-entry.json`、`replay-settings.json` | 逐位元組相同（不是要求項目，順帶核對） |
| `hotel-save.json` | 階段 3 時位元組不同（Rust `baca933d…`、Go `190b9f8f…`），只在 `party_roster` 的 `OffX`／`OffY`／`Exp` 整數值與 `Spells` 的 nil；#319 修正後逐位元組相同（`190b9f8f…`），比對工具不再正規化 |
| `verify_chapter_parity.py` | 兩份都是 `status=passed`、behavior／nodes／transaction 通過、frames 依政策不要求、保存 `accepted_disclosed_difference`，結論相同 |

竄改反向檢查：把 Rust `checkpoints.jsonl` 第 12 筆的 `rng_word` 改一個值，比對工具列出該筆該欄並以結束碼 1 失敗。

### R4 階段 3 整合紀錄

| 症狀 | 真因 | Go 行號 | 修正 |
|---|---|---|---|
| 設了升級成長觀察者時 `bind_command_learn` 停在 `todo!` | `Game::growth_roll_observer` 的型別（收 `&mut Game` 的 `FnMut`）與 `State::native_growth_roll_observer` 不同，無法照 Go 交給戰場狀態（A 組共用變更需求） | `main.go:4087`..`4088` | 欄位改成 `Option<battle::Callback<dyn Fn(&battle::Unit, u16) -> u16 + Send + Sync>>`，`bind_command_learn` 直接 `clone` 給 `st`；回歸測試 `growth_roll_observer_reaches_battle_state` |

除此之外，第一次完整重播就與 Go 黃金逐位元組相同，沒有其他移植錯誤要修；`fd2-rules`、`fd2-campaign` 沒有改動。
檢查點層的 `KNOWN_DIFFERENCES`：無。

crate 層 `allow(unused, dead_code, unreachable_code, clippy::todo)` 仍保留：驅動端只在測試編譯，正式建置仍沒有呼叫 `update` 的
公開入口。2026-11-15 試算拿掉後 `cargo clippy --all-targets` 有 882 則警告（lib 819、lib test 另 63 則不重複）。

## R5 階段 1：CPU 畫布、Ebiten 語意子集與 Draw 骨架（2026-11-15）

代號 `r5a`，合併基準 `eeb024d7`（含 R5 階段 0）。計畫見 [121](../../../docs/goal/121-goal-rust-r5-original-art-shell-20261115.md)。

### CPU 畫布

`src/game/canvas.rs`（Rust 新增，沒有 Go 檔對應），測試在 `src/game/canvas/canvas_test.rs`。放在 `game` 底下而不是 crate 層：
R4 已經把 `*ebiten.Image` 定成 `Picture`，每個繪圖函式的簽名都收 `&mut Picture`／`&Arc<Picture>`；畫布操作寫成 `Picture`
的方法，`screen.DrawImage(img, op)` 一對一寫成 `screen.draw_image(&img, &op)`。`fd2-app` 只讀畫格位元組（`Picture` 的欄位
是 `pub`），之後的 GPU 模式以這裡的輸出當黃金，都不需要畫布操作本身。

**`Picture` 改存預乘 alpha。** R4 的 doc 寫「不預乘」，但 Ebiten 影像在 `NewImageFromImage` 時就預乘，之後混色也在預乘空間，
`ReadPixels` 讀回的也是預乘值；照存才能和 Go 黃金逐位元組比。沒有另立畫布型別。R4 的轉換函式逐一對照：

| 函式 | R4 | R5 階段 1 | 影響 |
|---|---|---|---|
| `picture_from_indexed`（`*image.Paletted`） | 調色盤解除預乘後存 | 原值（`ColorRgba` 本來就是預乘） | 調色盤都是不透明色，位元組不變 |
| `picture_from_paletted`（`color.NRGBA` 調色盤） | 原值 | 依 `color.RGBAModel` 預乘 | 同上 |
| `decode_png_picture`（Go `*image.NRGBA`） | 不預乘 | `uint8((v*a*0x101/0xff)>>8)` | 只有半透明像素會變；原版美術模式的 PNG（舊 tileset、sprite）是全透明或不透明 |
| `masked_rgba`（Go 寫 NRGBA 後再 `NewImageFromImage`） | 原值 | 再預乘一次（與 Go 兩步相同） | 調色盤不透明，位元組不變 |
| `picture_from_rgba`、`native_attack_assets`、`title` 的 AFM 畫格 | 原值 | 不變（Go 本來就是 `*image.RGBA` 或不透明） | 無 |

Go `*image.NRGBA` 對應的 `Picture` 存的是 `NewImageFromImage` 之後的值；Go 直接讀 NRGBA 像素做計算的地方只在 HD，R6 處理。

子集與 Go 對照：

| Go（Ebiten v2.6.6） | Rust |
|---|---|
| `ebiten.NewImage(w, h)` | `Picture::new_image(w, h)`（寬高非正時 panic，同 Ebiten） |
| `ebiten.NewImageFromImage(*image.Paletted)`／`(*image.NRGBA)` | `Picture::from_indexed`（即 `picture_from_indexed`）／`Picture::from_nrgba` |
| `img.Clear()`、`img.Fill(c)` | `clear`、`fill`（覆蓋，不疊合） |
| `img.Set(x, y, c)` | `set`（`color.NRGBA` 先 `ColorNrgba::premultiplied`） |
| `img.WritePixels`、`img.ReadPixels` | `write_pixels`（長度不符 panic）、`read_pixels` |
| `img.SubImage(r)` | `sub_image`（來源）、`sub_image_mut`（目標：`fill`、`clear`、`set`、`draw_image`、`draw_sub_image`、`sub_image_mut`） |
| `dst.DrawImage(src, op)` | `draw_image`、`draw_sub_image` |
| `ebiten.GeoM`（`Translate`、`Scale`、`Element`） | `GeoM`（同樣存 `a-1`、`d-1`，`elements32` 轉 float32） |
| `ebiten.ColorScale`（`Scale`、`ScaleWithColor`） | `ColorScale`（float32 存 `係數-1`） |
| `ebiten.DrawImageOptions` | `DrawImageOptions { geo_m, color_scale }`（source-over、最近鄰、`ColorM` 單位矩陣） |
| `image.Rect`、`r.Intersect` | `rect`、`rect_intersect`（`r.Empty()` 沿用 `font.rs` 的 `rect_empty`） |

公式與出處寫在 `canvas.rs` 的模組文件。旋轉、負縮放、`ColorM`、線性濾鏡、其他混色模式不實作，用到時 panic，不輸出近似值。

測試向量 `src/game/canvas/ebiten_vectors.json`（93 KB，SHA-256 `5ade78bf…c339`）由 `tools/rust_r5_canvas_vectors.sh` 從兩份
Go 實測輸出抽出，輸入檔雜湊寫在檔內 `inputs`：

- 每個實驗保留重建輸入的參數與整張讀回的 SHA-256。測試照 Go 工具重建同一份輸入，以畫布畫一次再比雜湊，雜湊相同即每個通道值都相同：
  階段 0 混色 30 組（每組 196608 個通道值）、階段 1 取樣 89 組（640×400）、語意 11 組。
- 另外逐筆比對常見公式會算錯的值（階段 0 每個非逐值相同候選列出的不符，去重後 546 筆），包括 121 列出的 d=191、a=2 → 189 與
  30×0.55 → 16。
- 反向檢查：`common_rounding_formula_is_rejected` 以 `(d*(255-a)+127)/255` 算 `alpha_sweep_black` 必須得到不同雜湊；另手動把
  `adjust_destination_pixel` 的 11/16 改成 1/2、`mul255` 改成 `(x+127)/255` 各跑一次，取樣、混色與抽樣值測試都失敗（語意實驗
  `over_translucent_dst` 兩式結果相同，不能分辨），改回後通過。

KNOWN_DIFFERENCES：

| 名稱 | 內容 |
|---|---|
| `indexed-out-of-palette-transparent` | 索引超出調色盤時 Ebiten `imageToBytes` 會 panic；Rust 沿用 R4 寫成透明黑。正式素材的調色盤都是 256 色 |
| `png-16bit-premultiply` | 16 位元 PNG 先去掉低位元組再預乘，Go 在 16 位元預乘後才取高位元組，半透明時可能差 1。原版美術模式沒有 16 位元 PNG，未實測 |
| `sampling-out-of-range-texel` | 對齊後的頂點讓覆蓋像素的來源索引不會越界（89 組實測都沒有）；萬一越界，Ebiten 會取到圖集鄰格，Rust 當作沒有覆蓋 |

### Ebiten 實測補充（取樣與 Fill）

命令：`FD2_GOCACHE=<主工作區>/work/gocache tools/rust_r5_ebiten_sampling.sh <主工作區>/work/r5-stage1-20261115/sampling`。
只執行 R5 新增的 `remake/cmd/fd2-r5-ebiten-rounding`（加 `-sampling-out`，實驗在 `sampling.go`），環境同階段 0（`fd2-go-test-local`、
Go 1.22.12、Ebiten v2.6.6、Xvfb 下 Mesa llvmpipe）。兩次輸出逐位元組相同，`ebiten-sampling.json` SHA-256
`096f7972595345223ebc3e4335cbed53da5c96ca1429a3d99b4c20a3e17bdce8`。工具改成每個 Update 執行一步後，重跑階段 0 的 `-out`
仍是 `1dc87b3a…`。

- **取樣**：來源圖以座標編碼，讀回先確認可拆成兩軸對應表，再與候選規則逐值比。89 組（原生整幀與 storyView 的 2 倍、標題捲動
  的 2 倍加整數平移、倍率 1 的小數平移 32 組、圖塊鏡頭 4 組、舊對話框 `Scale(2, 2*sc)` 14 組、2.1 倍頭像 2 組、1×198 漸層
  `Scale(604, 182/198)` 2 組、標題 logo 縮放 30 組）全部可拆成兩軸，下列規則逐值相同：頂點先經 Ebiten
  `adjustDestinationPixel` 對齊到 0、5/16、11/16、1（`internal/graphics/vertex.go:113`），像素中心 `p+0.5` 在 `[x0, x1)` 內才
  覆蓋，來源索引 `floor((p+0.5-x0)*n/(x1-x0))`。float32 與 float64 內插都逐值相同，畫布用 float32。不經對齊的像素中心公式在平移小數
  約 0.3、0.5 時不符（`frac_translate_02`、`_07`、`tile_cam_37.5`，每軸 24 到 49 個）與舊對話框垂直縮放 k=1、3、4、6（6 到 8 個），
  logo 縮放與 2.1 倍頭像則碰巧相同；左緣四捨五入的公式連整數 2 倍都不符。
- **Fill**：直接 `Fill` 半透明或 R/G/B 大於 A 的非預乘色是**覆蓋**（BlendCopy），讀回等於 `color.RGBA` 的四個位元組
  （256 個值 × 13 個 alpha）。`SubImage(r)` 與父影像相交，座標是父影像座標，`Fill`、`DrawImage` 都只寫入相交矩形（含出界矩形
  與子影像的子影像）。`Clear` 全部為 0。
- **其他語意**：`Set` 寫入 `clr.RGBA()>>8`；`NewImageFromImage(*image.Paletted)` 是 `color.RGBAModel.Convert`；
  `NewImageFromImage(*image.NRGBA)` 是 `image/draw` 的預乘；`WritePixels` 後 `ReadPixels` 原樣（含 R/G/B 大於 A）；
  疊到半透明底時四個通道（含 alpha）都是 `min(255, s+mul255(d*(255-a)))`。

### PORTING.json 狀態與骨架

`tools/rust_r5_skeleton.sh`：Go 容器以 `tools/rust_r4_skeleton -r5` 讀階段 0 盤點的受版控複本
[`docs/data/rust-r5-executed-draw-functions-20261115.json`](../../../docs/data/rust-r5-executed-draw-functions-20261115.json)
（與 `work/r5-stage0-20261115/inventory/executed-draw-functions.json` 逐位元組相同，`60e0a421…`），以 R4 的型別閉包與命名產生簽名，
輸出 `r5-plan.json`；Python 容器以 `tools/rust_r5_skeleton_apply.py` 依 crate 現況套用；最後 `cargo fmt`。規則：

- 已是 `ported` 的不動；非 HD 的改 `r5`，crate 裡沒有對應函式的補 `todo!("r5:<檔>.go:<函式名>")` 骨架，附在該 Go 檔子模組末尾的
  「R5 階段 1 骨架」區段；HD 檔的 `r8` 改 `r6`，不產生骨架（Draw 裡的 HD 分支 `todo!("r6:…")`）。
- 例外：更正錯誤的 `ported`（逐筆記在 `PORTING.json` 的 `r5_stage1.corrections`）。

| Go | 原狀態 | 改為 | 原因 |
|---|---|---|---|
| `font.go:175` `Font.Draw` | ported | r5（補骨架） | crate 裡沒有這個函式：R4 階段 2 的狀態工具以名稱比對，把 `Game.Draw` 的狀態套到 `Font.Draw` |
| `native_end_turn_recovery.go:162` `drawNativeEndTurnRecovery` | ported | r5 | 本體只有 `todo!("R5: …")`，狀態工具把註解算成本體 |
| `native_physical_body.go:414` `drawNativePhysicalBody` | ported | r5 | 同上（另有 `let _ = …;`） |
| `native_physical_scene.go:372` `drawNativePhysicalPrelude` | ported | r5 | 同上 |
| `native_story_dialogue_locale.go:160` `drawIndexedLocalizedText` | ported | r5 | 同上 |

只有部分分支是 `todo!("R5…")` 的仍算 ported，列在 `r5_stage1.ported_with_r5_todo_branch`：`Font.faceFor`、`Font.Width`
（等外殼注入 `TextMeasure`）、`Game.Draw`（本節的分派骨架）、`drawModernBattlePanelInk`（HD）。

結果（`r5_stage1.result`）：盤點 196 個函式，非 HD 127 個中 ported 69、r5 58（新骨架 52）；HD 69 個中 ported 27、r6 42。
全 crate 狀態 ported 626、r5 58、r6 59、r8 1179。重跑第二次不改任何檔；從 `eeb024d7` 的 crate 重新套一次，`PORTING.json` 與
20 個子模組的骨架區段與這次逐位元組相同。

### `Game::draw` 分派骨架

`main.rs` 的 `Game::draw` = `begin_present_frame`（R4，Go 9356..9372）→ `draw_frame`（Go 的 if 鏈）→ `draw_deferred`（Go 第二個
`defer`：`nativeFullDACWhite`／`Black` 與黑 DAC 的 `Fill`、白閃 `drawn`、說明畫面）。分支是否執行到依階段 0 的 `Draw` 覆蓋率
（opening＋chapter 合計次數）：

| Go 分支 | 行 | 執行次數 | Rust |
|---|---|---|---|
| `startupBlocked`、`nativeDefeat`、`nativeEnding`、ch20／ch23／2189A／單位登場、24B4D、`m == nil` | 9395..9488 | 0 | `todo!("r8:…")`（需要 Fill 的先 Fill 黑） |
| `titlePhase` → `drawTitle` | 9406 | 1964 | `draw_title`（r5） |
| `nativeCh28PostPresent` | 9461 | 1064 | `draw_native_ch28_post_present`（r5）；失敗路徑 r8 |
| `indexedTransition`、補血、修飾、AI 修飾、指令 0～35、AI 物品、`nativeTownDeparture` | 9490..9714 | 0 | `todo!("r8:…")` |
| `nativePaletteRamp` | 9715 | 5934 | `draw_native_palette_ramp`（r5）；DebugPrint r8 |
| `nativePalettePulse` | 9725 | 0 | r8 |
| `spawnIntroTransition` | 9735 | 48 | `draw_native_spawn_intro`（r5） |
| indexed 回合演出、事件 61 | 9745..9763 | 0 | r8 |
| `nativeEndTurnRecovery` | 9765 | 42 | `draw_native_end_turn_recovery`（r5）；失敗時 `cancel_native_end_turn_recovery` 已移植，DebugPrint r8 |
| `nativeAIIdleRecovery` | 9776 | 3 | `draw_native_ai_idle_recovery`（r5）；失敗收尾與 `present.rs` 相同 |
| `atk` → `drawBattleScene` | 9792 | 6488 | `draw_battle_scene`（r5） |
| 尾段 `draw_field` | 9804..10188 | 14812 | 見下 |

尾段：`modernHDMapFrameAdmission` 在 `modern_story_portraits` 為 `None` 時一律 false（忠實模式），否則 `todo!("r6:…")`；
正規化地圖層 9814..9950（storyView 10941 次、圖塊、移動範圍格 135 次、單位、游標 3828 次、`drawNativeMapHUD` 3200 次）是
Draw 本體的 r5 `todo!`；`drawNativeMapFrame` 6371 次；`drawNativeMovementPanel`、`drawPhaseBanner`、`drawNativeStoryDialogue`；
TTF 短訊 2650 次與勝負字樣 1 次照 Go 寫好、呼叫 `Font::draw`（r5）；選中單位中文名與舊對話框 0 次（r8）；`drawCampaignUI`、
`drawNativeClassUIJob`、`drawNativeSystemEndTurn`、`markActionOverlayDrawn`（10 次）或 `drawRing`、指令格、法術與物品選單、
`drawNativeSystemInfoUI`、`drawNativePlayerStatus`；舊 fade 疊層 106 次是 Draw 本體的 r5 `todo!`；`captureShot` 是 r8（截圖模式）。

Draw 裡改流程狀態的地方（R4 A 組表）在 `draw` 路徑由各 `draw*` 函式自己做，與 Go 相同；`acknowledge_present` 照舊是無畫面重播
不擷取畫格時的版本。兩者不能在同一幀都呼叫（旗標會設兩次，Go 黃金是「pump 擷取點呼叫 Draw，`ackPresents` 另外只在三處呼叫
Draw」，見下節）。`finish_present_frame` 改成 `pub(crate)` 讓 `draw_deferred` 共用。

### R5 對照表（階段 2 的工作單）

依階段 0 分組；「局部黃金」是該函式在哪一種 Go 黃金畫格裡執行到：`opening`、`chapter` 是 `tools/rust_r5_golden_frames.sh` 的兩種
模式；`FD2_SHOT_STATE` 表示城鎮與設施在 `chapter` 模式只擷到 1 到 21 幀（多半不經 `pump`），要照 121「驗證」第 2 點另補。
畫布操作取自盤點的 `ebiten_api`（函式本體直接用到的）。表由 `PORTING.json` 狀態與盤點檔產生。`Game.Draw` 本體另有兩段 r5
`todo!`（正規化地圖層 9814..9950 約 137 行、fade 疊層 12 行）。

| 分組 | 函式（r5／ported／HD r6） | 行數（r5） | 畫布操作 | 局部黃金 |
|---|---|---|---|---|
| 城鎮與設施 | 28（16／12／0） | 859 | DrawImage、平移、縮放、NewImageFromImage、NewImage、Fill | chapter、chapter＋FD2_SHOT_STATE、opening＋chapter |
| Game.Draw 本體（分派、正規化地圖層、舊對話框與 fade） | 1（0／1／0） | 0 | DrawImage、平移、縮放、NewImageFromImage、NewImage、Fill、Set、DebugPrint | — |
| HUD 與選單 | 35（20／15／0） | 495 | DrawImage、平移、縮放、NewImageFromImage、NewImage、Fill、SubImage | chapter、opening＋chapter |
| 對話與文字 | 16（5／11／0） | 205 | 平移、ColorScale、TTF | chapter、opening＋chapter |
| 戰鬥演出 | 7（5／2／0） | 297 | DrawImage、平移、縮放、NewImageFromImage、Fill | chapter |
| 地圖與單位 | 14（7／7／0） | 136 | DrawImage、平移、縮放、NewImageFromImage、ColorScale | opening＋chapter |
| 標題與開場 | 7（4／3／0） | 198 | DrawImage、平移、縮放、NewImage、Fill、Clear、ColorScale | opening、opening＋chapter |
| 轉場與調色盤 | 4（1／3／0） | 25 | DrawImage、縮放、NewImageFromImage | chapter |
| 素材路徑與時鐘（非繪圖） | 14（0／14／0） | 0 | — | — |
| 共用與其他 | 1（0／1／0） | 0 | — | — |
| HD 檔（原版模式下有執行，多為守門） | 69（0／27／42） | 0 | DrawImage、平移、縮放、NewImageFromImage、NewImage、WritePixels、TTF、Dispose | — |

| 分組 | Go 位置 | 函式 | 行數 | 畫布操作 | 局部黃金 |
|---|---|---|---|---|---|
| 城鎮與設施 | `main.go:10638..11123` | `drawCampaignUI` | 486 | DrawImage、平移、NewImage、Fill | opening＋chapter |
| 城鎮與設施 | `native_church_ui.go:214..235` | `drawNativeChurchUIJob` | 22 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_class_ui.go:206..208` | `presentNativeClassFrame` | 3 | — | opening＋chapter |
| 城鎮與設施 | `native_class_ui.go:210..223` | `presentNativeClassFrameWithPalette` | 14 | DrawImage、縮放、NewImageFromImage | opening＋chapter |
| 城鎮與設施 | `native_class_ui_lifecycle.go:179..240` | `drawNativeClassUIJob` | 62 | — | chapter |
| 城鎮與設施 | `native_hotel_ui.go:82..102` | `composeNativeHotelStable` | 21 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_hotel_ui.go:106..157` | `composeNativeHotelFrame` | 52 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_hotel_ui.go:159..164` | `drawNativeHotel` | 6 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_preparation_ui.go:244..263` | `composeNativePreparationPromptFrame` | 20 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_preparation_ui.go:429..456` | `drawNativePreparation` | 28 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_preparation_ui.go:458..460` | `composeNativePreparationStableFrame` | 3 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_preparation_ui.go:462..464` | `composeNativePreparationStableFrameLayers` | 3 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_preparation_ui.go:466..477` | `composeNativePreparationStableFrameParts` | 12 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_shop_ui.go:1495..1541` | `drawNativeShopUIJob` | 47 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_shop_ui.go:1543..1611` | `drawNativeShop` | 69 | — | chapter＋FD2_SHOT_STATE |
| 城鎮與設施 | `native_town_ui.go:224..234` | `drawNativeTown` | 11 | — | chapter＋FD2_SHOT_STATE |
| HUD 與選單 | `action_overlay_runtime.go:823..873` | `drawNativeSystemEndTurn` | 51 | — | opening＋chapter |
| HUD 與選單 | `main.go:10194..10197` | `shouldDrawTransientBattleMessage` | 4 | — | opening＋chapter |
| HUD 與選單 | `main.go:10221..10303` | `drawRing` | 83 | DrawImage、平移、縮放、NewImage、Fill | opening＋chapter |
| HUD 與選單 | `main.go:10537..10572` | `drawSpellMenu` | 36 | DrawImage、平移、NewImage、Fill | opening＋chapter |
| HUD 與選單 | `main.go:10577..10613` | `drawItemMenu` | 37 | DrawImage、平移、NewImage、Fill | opening＋chapter |
| HUD 與選單 | `main.go:10617..10635` | `drawNativeCommandGrid` | 19 | — | opening＋chapter |
| HUD 與選單 | `main.go:11571..11584` | `drawCursor` | 14 | DrawImage、平移、NewImage、Fill | opening＋chapter |
| HUD 與選單 | `main.go:12148..12150` | `phaseBannerElapsedMillis` | 3 | — | chapter |
| HUD 與選單 | `main.go:12154..12171` | `phaseBannerSlideOffset` | 18 | — | opening＋chapter |
| HUD 與選單 | `main.go:12203..12217` | `blitPhaseBannerGlyphs` | 15 | — | chapter |
| HUD 與選單 | `main.go:12225..12246` | `blitPhaseBannerGlyph` | 22 | — | chapter |
| HUD 與選單 | `main.go:12249..12252` | `phaseBannerGlyphsAvailable` | 4 | — | chapter |
| HUD 與選單 | `main.go:12268..12301` | `phaseBannerStep` | 34 | — | opening＋chapter |
| HUD 與選單 | `main.go:12317..12334` | `drawPhaseBanner` | 18 | — | opening＋chapter |
| HUD 與選單 | `main.go:12421..12444` | `drawNativeMapHUD` | 24 | DrawImage、縮放、NewImageFromImage | opening＋chapter |
| HUD 與選單 | `native_end_turn_recovery.go:162..179` | `drawNativeEndTurnRecovery` | 18 | DrawImage、縮放、NewImageFromImage | chapter |
| HUD 與選單 | `native_map_unit_window.go:14..17` | `nativeActionOverlayInMapFrame` | 4 | — | opening＋chapter |
| HUD 與選單 | `native_player_input.go:214..237` | `drawNativePlayerStatus` | 24 | — | opening＋chapter |
| HUD 與選單 | `native_player_movement.go:100..118` | `drawNativeMovementPanel` | 19 | DrawImage、平移、縮放 | opening＋chapter |
| HUD 與選單 | `native_system_info_ui.go:186..233` | `drawNativeSystemInfoUI` | 48 | DrawImage、平移、縮放、NewImageFromImage、SubImage | opening＋chapter |
| 對話與文字 | `font.go:175..191` | `Font.Draw` | 17 | 平移、ColorScale、TTF | chapter |
| 對話與文字 | `native_confirmation_mouth.go:71..81` | `composeNativeConfirmationMouth` | 11 | — | chapter |
| 對話與文字 | `native_story_dialogue.go:339..351` | `nativeStoryOpeningFrameIndex` | 13 | — | opening＋chapter |
| 對話與文字 | `native_story_dialogue.go:485..586` | `drawNativeStoryDialogue` | 102 | — | opening＋chapter |
| 對話與文字 | `native_story_dialogue_locale.go:160..221` | `drawIndexedLocalizedText` | 62 | — | chapter |
| 戰鬥演出 | `main.go:11231..11387` | `drawBattleScene` | 157 | DrawImage、平移、縮放、Fill | chapter |
| 戰鬥演出 | `native_ai_idle_recovery.go:328..350` | `drawNativeAIIdleRecovery` | 23 | DrawImage、縮放、NewImageFromImage | chapter |
| 戰鬥演出 | `native_ch28_post_present.go:339..363` | `drawNativeCh28PostPresent` | 25 | DrawImage、縮放、NewImageFromImage | chapter |
| 戰鬥演出 | `native_physical_body.go:414..454` | `drawNativePhysicalBody` | 41 | DrawImage、縮放、NewImageFromImage | chapter |
| 戰鬥演出 | `native_physical_scene.go:372..422` | `drawNativePhysicalPrelude` | 51 | DrawImage、縮放 | chapter |
| 地圖與單位 | `main.go:8363..8368` | `tileAt` | 6 | — | opening＋chapter |
| 地圖與單位 | `main.go:9338..9353` | `nativeMapFrameAdmission` | 16 | — | opening＋chapter |
| 地圖與單位 | `main.go:11514..11521` | `mapSpriteGroup` | 8 | — | opening＋chapter |
| 地圖與單位 | `main.go:11528..11548` | `drawUnitSprite` | 21 | DrawImage、平移、ColorScale | opening＋chapter |
| 地圖與單位 | `main.go:12450..12464` | `drawNativeMapFrame` | 15 | DrawImage、縮放、NewImageFromImage | opening＋chapter |
| 地圖與單位 | `main.go:12469..12525` | `nativeMapFrameImage` | 57 | — | opening＋chapter |
| 地圖與單位 | `native_spawn_intro.go:360..372` | `drawNativeSpawnIntro` | 13 | DrawImage、縮放、NewImageFromImage | opening＋chapter |
| 標題與開場 | `title.go:45..47` | `titleMenuY` | 3 | — | opening＋chapter |
| 標題與開場 | `title.go:456..467` | `titlePublisherBrightness` | 12 | — | opening |
| 標題與開場 | `title.go:486..493` | `titleScrollUsesPulsePalette` | 8 | — | opening |
| 標題與開場 | `title.go:716..890` | `drawTitle` | 175 | DrawImage、平移、縮放、NewImage、Fill、Clear、ColorScale | opening＋chapter |
| 轉場與調色盤 | `native_palette_ramp.go:164..188` | `drawNativePaletteRamp` | 25 | DrawImage、縮放、NewImageFromImage | chapter |

### R5 階段 1：畫格擷取介面

`src/game/parity_replay/rust_golden_frames_test.rs`（對應 Go `remake/cmd/fd2/rust_golden_frames_test.go` 的 chapter 模式）。
設 `FD2_GOLDEN_FRAMES_MODE=chapter` 與 `FD2_GOLDEN_FRAMES_OUT=<目錄>` 才建立 `FrameCapture`，掛在 `Driver::frame_capture`；兩者都沒
設時是 `None`，重播與 R4 完全相同。只設一個或模式不是 `chapter` 時 panic。

| Go | Rust |
|---|---|
| `pump` 每一幀開頭的 `ch01FrameObserver(g)`（`native_chapter_one_play_test.go:305`，在上一個 `Update` 之後、`done` 與 `ackPresents` 之前） | `Driver::pump` 每一幀開頭的 `FrameCapture::observe`，同一位置 |
| `goldenFrameRecorder.observe`：`pinClock` → 記 gap → `screen.Clear()` → `g.Draw(screen)` → `ReadPixels` → SHA-256 | 同順序：`pin_clock` → gap → `Picture::clear` → `Game::draw` → `read_pixels` → SHA-256 |
| 連續相同雜湊併成一列（`count`）；序列雜湊是每次擷取 SHA-256 串接後的 SHA-256 | 同（`rows`、`sequence_sha256`） |
| 重播結束寫 `frames.json` | `ParityReplay` 的 `end` checkpoint 之後 `FrameCapture::finish` 寫 `<目錄>/frames.json`，欄位與 Go 同名 |
| `ackPresents` 在城鎮出發、有原生場景的攻擊演出、END 回收三處另外呼叫 `g.Draw` 到暫存影像 | 照舊呼叫 `acknowledge_present`（R4），不畫像素 |

目前 r5 函式都是 `todo!`，打開擷取會停在第一個擷取點；本輪只驗證關閉時重播結果不變。下列是重現 Go 黃金畫格的規格：

| 項目 | Go 黃金（`rust_golden_frames_test.go`） | Rust | 狀態 |
|---|---|---|---|
| 60 Hz 虛擬時鐘 | 第一次擷取時把 `playerClock.wall` 設在遠未來（`sample` 只回傳 `game`），`wall` 為零時 `game` 從 `time.Unix(1<<30, 0)` 起；之後每次擷取前把 `game` 推進 `Δg.frame × 16666666 ns × normalizePlayerSpeed(speed)`。pump 以外的 `Update` 期間遊戲時間凍結，到下一個 pump 幀才補上；pump 的 `Update` 看到的是上一次擷取的時間（比 `Draw` 晚一幀推進） | `FrameCapture::pin_clock` 照寫：`player_clock.wall = GameInstant(i64::MAX/2)`，`game += d × 16666666 × speed`。`VirtualFrameClock` 照舊每幀推進 `virtual_now`，但 `sample` 的差值為負，只回傳凍結的 `game`，與 Go 相同。擷取打開時遊戲時間改由擷取點推進，與 R4 重播（每個 `update` 前推進 1/60 秒，誤差小於 speed 奈秒）不同；Go 黃金在這個模型下檢查點仍是 `1c846f57…` | 已實作 |
| 第一個 pump 之前 | Go 以真實牆鐘跑 `startFromNormalTitle`（`clock = "virtual-60hz-from-first-pump"`），只有差值影響畫面 | 以 R4 的 `VirtualFrameClock` 跑，`clock` 同名 | 同 |
| 全域亂數 | 測試開頭 `rand.Seed(4)`，之後 `math/rand` 全域來源依程式順序被取用：故事對白嘴型 `rand.Intn(30)`（`native_story_dialogue.go:430`、`442`、`453`）、確認框嘴型第一次用到時 `rand.New(rand.NewSource(rand.Int63()))`（`native_confirmation_mouth.go:38`）、結局預覽（R8） | 擷取打開時要改成：`Game` 另存一個 `GoMathRand::new(4)` 代表全域來源，在重播開始前建立；`story_mouth_intn30` 改取它的 `intn(30)`；`native_confirmation_mouth_rng` 惰性建立時改用 `GoMathRand::new(global.int63())`；擷取關閉時維持 R4 的兩個固定種子（KNOWN_DIFFERENCES `cosmetic-mouth-rng-fixed-seed`） | **未實作**；`frames.json` 的 `global_rand` 寫明，階段 3 比對前必須接上 |
| gaps | pump 以外迴圈的 `Update` 不擷取，記 `{after_capture, uncaptured_updates, from_update_frame, to_update_frame, node}`；第一章 7 段 9282 個（`driveStory` 8430、武器店直接迴圈 817、`closeRing`／`settleActionOverlay` 5 段各 7） | 同一格式記在 `gaps`；Rust 的迴圈結構照 Go 移植（R4），段數與長度應相同，階段 3 先比 `gaps` 再比畫格 | 已實作 |
| 調色盤與像素格式 | `ReadPixels` 的預乘 RGBA8 | `Picture` 預乘（本節「CPU 畫布」） | 已實作 |

### R5 階段 1 驗證

| 檢查 | 結果 |
|---|---|
| `tools/rust_game_test.sh` | fmt、clippy `-D warnings` 通過；320 passed、4 ignored（R4 312 項加畫布 8 項） |
| `tools/rust_campaign_test.sh`、`tools/rust_rules_test.sh` | 530 passed、654 passed（兩個 crate 沒有改動） |
| `tools/rust_parity_ch01.sh`（擷取關閉，`work/r5-stage1-20261115/r5a-parity`） | `checkpoints.jsonl` `1c846f57…`、`FD2.SAV` `f3942ff2…`、`hotel-save.json` `190b9f8f…`，三者與 Go 黃金逐位元組相同；`verify_chapter_parity.py` 兩份結論相同 |
| `Cargo.lock` | 沒有改動；沒有新增套件 |
| 實驗重跑 | `tools/rust_r5_ebiten_sampling.sh` 兩次輸出相同；`tools/rust_r5_canvas_vectors.sh`、`tools/rust_r5_skeleton.sh` 重跑結果相同 |
