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
第 1、3 項，其餘停在 `todo!("R5: …")`；表中屬於別組的旗標由該組的 `draw*` 函式設定。

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
`rune_breakable` 的 Han／Hiragana／Katakana／Hangul 範圍取自 Go 1.22.12 `unicode.Scripts`（Unicode 15.0.0），寫成資料表（Unicode Script 屬性
資料，不是 Go 程式碼；若主代理認定屬於教訓 `rust-port-no-go-stdlib` 的例外範圍，需另徵同意）。

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
| 黃金檔 | `work/rust-build/golden/fd2-game-save-r4f-v1.json`，349 個案例，SHA-256 `48afbbf79107a64fe139b0d6171abb5a94455355ef57226c2ce04080a9212be4`（go1.22.12，Go 端連跑兩次相同） |
| 存檔 196 | Go 寫出的固定與固定種子隨機 `saveData` 164 份（含 JOIN 建構器建立的真實持續隊伍單位）、手寫讀取端輸入 23 份、第一章戰鬥中 F5 存檔 2 份（附戰場快照）、真實 Go 版存檔 7 份 |
| 設定 80 | Go 寫出的 64 份、手寫 14 份、真實設定檔 2 份；比零值起點與 `loadSettings` 起點的解碼及重新寫出的位元組 |
| 附檔 45、附檔鍵 28 | `readBalanceSidecar` 的接受／拒收與錯誤文字、讀回與重新寫出；`balanceSaveKey` 的鍵與錯誤 |
| 存檔位元組 | 191 份接受的存檔裡 64 份 Rust 重新寫出逐位元組相同；127 份只在 battle 套件子樹（名冊單位、戰場快照）不同，以 `NORMALIZATIONS` 比對相同，拿掉這兩個子樹的外層位元組 191 份全部相同。設定與附檔逐位元組相同 |
| 竄改反向檢查 | 測試內對七類比對各改一個欄位（讀回金幣、名冊 HP、外層位元組、設定位元組、設定讀回、附檔錯誤文字、附檔鍵），全部被抓到 |

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
巢狀結構欄位的陣列寫法；存檔與附檔 JSON 的重複鍵。`NORMALIZATIONS`（只用在 battle 子樹與 `party_join_order`）：
null 與空陣列／空物件／空字串相同、數字只比數值、物件鍵順序、`party_join_order` 的 null 與 `[]`。

### fd2-rules 缺口（F 組發現）

Go 寫出的存檔由 Rust 讀回再寫出時，位元組只在 fd2-rules 型別的 serde 上不同（黃金統計，127 份存檔）：

| 型別 | 欄位 | Go | fd2-rules |
|---|---|---|---|
| `Unit` | `OffX`、`OffY`、`Exp`（`float64`） | 整數值寫 `0` | `0.0` |
| `Unit` | `Spells`、`Inventory`、`Equipped`、`InventorySlots`、`NativeInventoryFlags` | nil 寫 `null` | `Vec` 不分 nil，寫 `[]` |
| `StateSnapshot` | `flags` | 非 nil 空 map 寫 `{}` | 空的寫 `null` |
| `BalanceOptions` | `characters`、`skill_hit`（整數鍵） | 依鍵的字串排序 | 依整數排序（F 組寫出時改走 `go_json_balance_options`，已相同） |

讀回的內容都相同（逐欄比對通過）。要讓整份存檔逐位元組相同，需要 fd2-rules 的 `Unit` 以 `Option<Vec<_>>` 或寫出端
規則保留 nil、`float64` 欄位以整數值寫出，並讓 `StateSnapshot` 區分 nil 與空 map；由主代理決定是否在 fd2-rules 補。
