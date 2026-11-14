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
