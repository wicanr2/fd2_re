# fd2-campaign 移植說明

## campaign 組（campaign、ending、editorcanonical；R3 階段 1 骨架）

`fd2_campaign::campaign`、`ending`、`editorcanonical` 分別對應凍結 Go 版 `remake/internal/campaign`、
`ending`、`editorcanonical`。階段 1 把型別、常數、套件層變數、表格與全部函式簽名搬過來，函式本體是
`todo!("<套件>/<file>.go:<FuncName>")`；階段 2 由下方三組平行補上本體、移植測試並加黃金差分。
型別與所有權的決定寫在 [`src/campaign.rs`](src/campaign.rs) 與 [`src/ending.rs`](src/ending.rs) 的模組文件，
階段 2 照用。

逐檔、逐函式的機器可讀清單在 [`PORTING.json`](PORTING.json)：Rust 路徑、Go 正式行數、測試檔與行數、
每個函式的 Rust 路徑與狀態、檔內函式的呼叫對象、檔案相依、每檔的階段 2 組別，以及延後測試。

### 骨架現況

以凍結 Go 版 `8bdcf8c8` 量測（2026-11-14）。

| 項目 | campaign | ending | editorcanonical | 合計 |
|---|---|---|---|---|
| Go 正式檔／行數 | 48／10919 | 14／3547 | 2／285 | 64／14751 |
| Go 測試檔／行數 | 39／11553 | 9／2243 | 2／210 | 50／14006 |
| Go 具名型別 | 106 | 65 | 5 | 176 |
| Go 匿名 struct（Rust 給名稱） | 1 | 0 | 3 | 4 |
| 常數 | 116 | 50 | 0 | 166 |
| 套件層變數 | 14 | 2 | 0 | 16 |
| 函式與方法 | 253 | 97 | 4 | 354 |
| 本體已直接移植 | 31 | 9 | 0 | 40 |
| `todo!()` | 222 | 88 | 4 | 314 |
| Rust 行數（含手寫檔） | 6102 | 2456 | 237 | 8795 |

- 每個 Go 檔 `x.go` 一個子模組 `src/<套件>/x.rs`。手寫、沒有 Go 對應的檔：`campaign/error.rs`（共用錯誤型別）、
  `campaign/go_image.rs`（Go `image` 的三個值型別）、三個 `ext_deps.rs`（外部項目的編譯期清單）。
- 直接移植的只有一行委派、常數回傳、欄位讀取與單一布林式（例如 `RenderNativeRosterRows` 委派給
  `render_native_roster_rows_inner`、`CanRevive`、`FinalizeGood`），PORTING.json 標 `ported`。
- 每個函式的 doc 保留 Go 註解，接著一行 `Go：<檔>:<行>` 與原始 Go 簽名；索引、所有權與 Rust 改寫另有說明行。
  Go 的 JSON 標籤保留在欄位 doc 的「Go 標籤：」。
- `cargo build`、`cargo clippy --offline --locked -p fd2-campaign --all-targets -- -D warnings`、`cargo fmt --check`
  通過。三個模組根的 `allow(unused, dead_code, unreachable_code, clippy::todo)` 是骨架專用，階段 2 全部移植後移除；
  `allow(clippy::too_many_arguments, clippy::type_complexity)` 與兩個 doc lint 照 `fd2-rules` 的 crate 層設定長期保留
  （`lib.rs` 不歸本組，所以放在模組層）。
- 骨架由一次性的 Go `go/types` 分析工具產生（R2 產生器改寫；參數可變性、nil 檢查、回傳值是否新建都由型別資訊
  推導，外部 crate 的可變性與 derive 由掃描 `fd2-rules`／`fd2-formats` 的 Rust 原始碼取得）。產生器沒有進版控；
  從現在起這些檔案改為人工維護。

### 型別與所有權決定

- **單位與戰場狀態**一律用 `fd2_rules::battle` 的型別，照 R2 的所有權規則：帶 `*battle.State` 的函式把 `*Unit`
  改成 `state.units` 的 `UnitIdx`；只操作單一單位的函式收 `&Unit`／`&mut Unit`；`NativeSystemInfoCampCounts`
  的 `[]*battle.Unit` 收 `&[Unit]`（呼叫端傳 `&state.units`）。Go 以值保存的 `battle.Unit`（持續隊伍
  `map[int]battle.Unit`、JOIN 建構與 CONTINUE 物化的結果）在 Rust 是擁有的 `Unit`，由呼叫端 push 進
  `State::units`。三個套件的正式程式沒有任何欄位保存 `*battle.Unit`，所以沒有 `UnitIdx` 欄位，也沒有 R2 那種
  多一個 `units` 參數的函式。
- **錯誤**：`CampaignError`（`campaign/error.rs`），形狀同 `BattleError`（只保存訊息，`%w` 用 `wrap`），三個模組
  共用，`Result<T>` 別名同名。另立型別是因為孤兒規則不允許在本 crate 為 `BattleError` 實作
  `From<SaveError>`、`From<AfmError>`、`From<FiganiError>`。已實作 `From<BattleError>` 與 9 個格式層錯誤。
  三個套件沒有 `errors.Is`／`errors.As` 判斷錯誤種類（`editorcanonical` 的 `errors.Is(err, io.EOF)` 只判斷
  尾端 JSON），只比訊息文字。
- **共享合成器**：Go 的 `*IndexedCompositor` 同時被 `Player`、`MontageCycle`、`MontageTailPlayer`、
  `Phase0Player` 持有並改寫（`cmd/fd2/ending_preview.go` 把 `p.player.Compositor` 傳給後兩者後，`Player` 仍
  繼續使用）。Rust 以 `ending::SharedCompositor = Arc<Mutex<IndexedCompositor>>` 表達：結構欄位與五個存進結構的
  建構函式參數用它，其餘函式照一般規則收 `&`／`&mut IndexedCompositor`，呼叫端先 `lock()`。其他只讀的 Go
  指標欄位（`*afm.Clip`、`*fdtxt.Font`、`*fdtxt.Strings`、`*figani.Animation`、`Runner.C *Campaign`）由結構
  擁有一份 `Option<T>`。
- **Go `image`**：`image.Point`、`image.Rectangle`、`image.RGBA` 在 `campaign/go_image.rs` 定義同形值型別
  （`RGBA` 依 clippy 命名寫成 `Rgba`）。用到的方法（`Rect` 的 min／max 正規化、`Union`、`Dx`／`Dy`）由用到的組
  依 Go 定義補上。
- **`any` 與 `json.RawMessage`**：Go `any`（只在 JSON 欄位與 `handlerTextIndex`）→ `serde_json::Value`；
  `json.RawMessage` → `editorcanonical::GoRawMessage`（暫時是 `serde_json::Value` 的別名，見下方缺口）。
- **serde**：骨架不 derive serde，JSON 標籤留在 doc（同 R2）。階段 2 由擁有該檔的組替自己檔裡的型別加 serde；
  `[]byte` 欄位（`native_chapter_slot_restore.go` 的 `Entries`）用 `fd2_rules::battle::go_bytes`，nil 切片的
  `null` 用 `null_as_default`。Go 的 JSON 欄位都不含外部 crate 的型別，所以不需要 `fd2-formats` 補 serde。
- **改名**：Go 型別 `Option`（`campaign.go` 的 choice 選項）與 Rust prelude 相撞，改為 `ChoiceOption`；
  `ending.Player` 的未匯出欄位 `ani *ANIPlayer` 與匯出欄位 `ANI` 轉 snake_case 後相同，改為 `ani_player`；
  未匯出函式與匯出函式轉 snake_case 後相同的 7 組，未匯出的那個加 `_inner`（例如
  `compose_native_town_frame_inner`）。`campaign.go` 的子模組與父模組同名，加 `allow(clippy::module_inception)`。
- **只讀參數的人工判定**：產生器的可變性分析不分流程，把「先淺複製、改指向新切片，再寫入」誤判成改寫參數。
  下列參數逐處讀過 Go 原始碼確認只讀，簽名用 `&`：`ComposeNativeStoryDialogueProgressiveFrames`／
  `ComposeNativeStoryDialoguePage` 的 `layout`、`cloneNativeDialogueLayout`／`attachNativeDialogueLayout` 的
  `layout`、`CompileHandlerScript`／`compileHandlerScript` 的 `script`、`ExpandNativeDialogueGroups` 的 `beats`；
  `NewMontageCycle` 的 `units` 存進結構後只讀，骨架產生後改為 `&[Vec<u8>]`（結構擁有複本）。
  `append([]byte(nil), x...)` 不再被當成共用 `x`，背景畫面參數都是 `&[u8]`。

### 檔案相依

- 以函式呼叫圖彙總到檔案（64 檔），**沒有循環區**（Tarjan 強連通分量全為單檔），和 R2 `battle` 的 33 檔循環區不同。
- `ending` → `campaign`：只有 `native_dialogue.go` 呼叫 `NativeClassListOpeningFrames`、`NativeClassListClosingFrames`。
- `editorcanonical` → `campaign`：正式程式沒有；`compile_test.go` 用 `campaign.Decode`、`campaign.Load` 驗證編譯結果。
- 外部項目（`ext_deps.rs` 在編譯期以 `use` 檢查，測試檔另外用到的放在 `cfg(test)`；方法以掃描原始碼確認）：

  | 模組 | campaign 正式 | campaign 測試另用 | ending 正式 | ending 測試另用 |
  |---|---|---|---|---|
  | `fd2_rules::battle` | 26 | 6 | 1 | 1 |
  | `fd2_formats::fdother` | 13 | 6 | 12 | 0 |
  | `fd2_formats::save` | 9 | 9 | 4 | 0 |
  | `fd2_formats::fdicon` | 8 | 1 | 2 | 0 |
  | `fd2_formats::fdtxt` | 5 | 5 | 9 | 2 |
  | `fd2_formats::dato` | 1 | 1 | 2 | 1 |
  | `fd2_formats::figani`／`afm` | 0 | 0 | 5／1 | 0／1 |
  | 方法 | 23 | 3 | 13 | 1 |

  Go 用到的外部項目全部都有公開的 Rust 對應，`fd2-rules` 與 `fd2-formats` 已全部移植（沒有 `todo!()`）。

### 階段 2 分組

依函式呼叫圖與「每個測試函式實際需要哪些函式」（測試直接呼叫的函式、同套件測試輔助函式，再取遞移呼叫，
已移植的一行委派不算等待）分組。沒有循環區，所以三組可以在同一波次平行，只有 4 個測試函式要等別組。

| 組 | 主題 | 波次 | Go 檔 | 正式行數 | 測試檔 | 測試行數 | 函式（todo） |
|---|---|---|---|---|---|---|---|
| A | 劇本資料與處理器編譯 | 1 | 13 | 4537 | 12 | 6790 | 71（67） |
| B | 設施畫面與城鎮／商店／教會規則 | 1 | 26 | 4732 | 19 | 3225 | 157（132） |
| C | 存檔、CONTINUE、JOIN 與結局演出 | 1 | 25 | 5482 | 19 | 3991 | 126（115） |

A 的測試行數較多（`handler_compile_test.go` 2920 行、`native_story_dialogue_test.go` 1603 行），但都是表格式
期望值，正式碼與 B、C 相近；把劇本檔拆到別組會讓 1600 行以上的測試整批等待，所以不拆。

各組檔案（`c:` = campaign，`e:` = ending，`ec:` = editorcanonical）：

- **A**：c:campaign、handler_compile、handler_binding、handler_script、story_index_map、native_map_view_config、
  ai_commands、menu_state、native_story_dialogue、native_story_hd_text、native_story_dialogue_coverage；
  ec:bundle、compile
- **B**：c:church、class_change_table、equipment、shop、native_church_{menu,revive_success,scene}、
  native_class_{confirm,list}、native_gold_debit、native_load_slots、native_roster_list、
  native_shop_{equipment_recipient,item_list,purchase_message,purchase_success,recipient,scene,sell_message,service_menu,transfer_message}、
  native_system_info、native_town_{departure,scene}、native_transient_expiry、separated_shop_assets
- **C**：c:native_chapter_slot_{restore,save}、native_continue_{field_boundary,handoff,map_timing,pending_groups,runtime_units}、
  native_current_snapshot、native_join_{base,constructor}、native_persistent_party；
  e:全部 14 檔（compositor、dato_frame、dialogue_frame、figure_fade、finale、montage、montage_cycle、
  montage_portrait、montage_tail、montage_tail_player、native_dialogue、player、separated_fdfield、timeline）

順序：

1. **波次 1：A、B、C 平行。** B 完全不依賴其他組；A 只有一個 B 的函式（`blitNativeDialoguePortraitAt`）；
   C 依賴 B 的兩個職業清單轉場函式。
2. **整合**：三組合併後跑下表的延後測試，再移除三個模組根的骨架 allow。

測試檔歸屬：`x_test.go` 歸 `x.go`；名稱不對應的歸最長前綴相符的正式檔，再不行就歸測試呼叫最多函式的檔
（`native_story_portrait_rect_test.go` 歸 `native_story_dialogue.go`、`separated_town_assets_test.go` 歸
`native_town_scene.go`），只讀素材的 `story_asset_test.go` 歸 `campaign.go`。

#### 延後的測試

移植時先標 `#[ignore = "等 <組> 合併"]`，整合時拿掉。

| 所屬組 | 測試檔 | 延後的測試函式 | 等待 |
|---|---|---|---|
| A | `native_story_dialogue_test.go` | `TestComposeNativeStoryDialoguePageUsesOriginalIndexedAssets` | B |
| C | `native_chapter_slot_restore_test.go` | `TestNativeIntermissionGateBoundsLateEffectiveRouteOverrideToItsSource`、`TestNativeIntermissionGateRoutesEverySaveableChapterToCampaignNode`（要用 `campaign.Load` 讀戰役圖） | A |
| C | `ending/native_dialogue_test.go` | `TestNativeDialogueUsesSixBandOpeningAndFiveBandClosingRestore` | B |

組內函式呼叫到別組未完成函式的次數：A→B 2、C→B 2；B 沒有。

#### 測試與黃金差分規劃

R3 的完成條件是「讀入全部 `remake/assets` 劇本與語系，結果與 Go 黃金資料相同；相關 Go 測試全部移植通過」。
各組在自己的檔案下移植名下的 Go 測試（`src/<套件>/<x>/<t>_test.rs`，`mod` 宣告只加在自己的 `x.rs`），並各做一個
黃金差分（`remake/cmd/fd2-golden-campaign-<組>/`、`tests/golden_campaign_<組>.rs`、`tools/rust_golden_campaign_<組>.sh`）：

- **A**：Go 端以 `campaign.Load` 讀 `assets/scenarios/` 全部戰役圖，並編譯 `assets/cutscenes/{bindings,handlers}` 的
  全部處理器綁定與腳本、`dialogue-index` 的 story index map；輸出解碼後的結構（以 Go JSON 寫出）與編譯問題清單。
  另以 `editorcanonical.CompileCampaignJSON` 編 `assets/editor-canonical`，比解碼後的值。輸入檔排序並記大小與 SHA-256。
- **B**：以分離素材包與 `remake/assets/data` 的表，對固定的選擇、游標與脈衝狀態合成教會、商店、城鎮、職業清單與
  確認對話框畫面，記每格畫面的 SHA-256；規則面（`BuyGood`、`SellSlot`、`ReviveUnit`、`ApplyClassChange`）以固定
  `GoMathRand` 種子跑多組單位。
- **C**：以固定的合成存檔槽跑 `BuildNativeChapterSlotRestorePlan`、`EncodeNativeChapterSlot`、CONTINUE 五段物化與
  全部 JOIN 角色的建構；結局以 `assets/endings` 的時間軸、蒙太奇與片尾跑固定步數，記每步 VGA 與調色盤的 SHA-256。

需要原版或素材的測試走 `crate::test_support`（`docs_file`、`remake_asset`、`input`）；設 `FD2_REQUIRE_ORIGINAL=1`
時不得有略過。

### Go 寫法難對應處

- **map 迭代順序**：正式程式 22 處 `range` map。結果會隨順序改變的只有 `church.go:257`
  `ClassChangeCandidates`：先照 `order`，再把 `roster` 裡不在 `order` 的 id 依 map 迭代順序附加，Go 的輸出順序不固定，
  Rust 是遞增順序，黃金差分要改比集合或只給齊全的 `order`。其餘各處只影響「多個欄位同時不合法時報哪一個錯誤」
  （`campaign.go:735`、`handler_binding.go:126/134/139/180/343`、`native_chapter_slot_save.go:104`、
  `native_continue_pending_groups.go:100`、`ending/montage_tail.go:577`、`montage_tail_player.go:88/96`），
  或寫入順序無關（`native_chapter_slot_save.go:65`、`native_current_snapshot.go:55`、
  `native_continue_pending_groups.go:187/203`、`campaign.go:890/966`、`handler_binding.go:224`、
  `editorcanonical/compile.go:95/103/123`）。`sort.Ints` 只有一處（整數排序，無平手問題）。
- **淺複製**：`prefix := *layout`、`copyPatch := *patch`、`single := beat`、`candidate := *state` 等在 Rust 是 `clone()`
  （深複製）。讀到的幾處都在改寫前先換成新切片，深複製語意相同；`native_continue_pending_groups.go` 的
  `candidate := *state` 是「先在副本試算、成功才 `*state = candidate`」。
- **就地改寫再回傳同一切片**：`ComposeNativeChurchTextAt` 等函式改寫呼叫端的 `frame` 後回傳它。Rust 簽名是
  `frame: &mut [u8]` 並回傳 `Vec<u8>`；移植時確認呼叫端是用回傳值還是原切片。
- **nil 參數**：Go 會檢查 nil 的參數包成 `Option`。只靠被呼叫方法的 nil 接收者防護擋下的參數（例如
  `MaterializeNativeContinueRuntimeUnits` 的 `catalog`，靠 `catalog.validate()` 的 `catalog == nil` 分支）沒有包，
  對它們傳 nil 的 Go 測試分支不移植（同 R2）。
- **`any` 的數字**：Go 把 JSON 數字解碼進 `any` 時一律是 `float64`；`handlerTextIndex` 的 `int` 分支只有 Go 程式自己
  建的值才會走到。`serde_json::Value` 保留整數與小數的區別，移植時要照 Go 的 `float64` 判定（含 `math.Trunc` 與
  `math.MaxInt` 上限）。
- **`json.RawMessage` 與 `json.Marshal`**：`CompileCampaignJSON` 的輸出由 Go `json.Marshal` 產生：map 鍵排序、
  RawMessage 原樣保留巢狀物件鍵順序與數字字面值，並把 `<`、`>`、`&` 跳脫成 `<` 等。`serde_json::Value` 會排序
  巢狀鍵、正規化數字、不做 HTML 跳脫，所以 A 組的黃金差分比「解碼後的值」，不比位元組（列入 `NORMALIZATIONS`）；
  RawMessage 的「欄位缺席」（長度 0）與「明寫 `null`」在 Go 不同，Rust 要用 `Option<GoRawMessage>` 區分。
- **嚴格解碼**：`decodeStrict` 的 `DisallowUnknownFields` 在 serde 由目標型別加 `#[serde(deny_unknown_fields)]`
  表達，尾端多餘 JSON 用 `serde_json::Deserializer::end` 檢查。Go 解碼時欄位名不分大小寫，Rust 只認標籤原文（同 R2，
  列入 `KNOWN_DIFFERENCES`）。
- **`fmt.Sscanf(outcome, "opt%d", &i)`**（`campaign.go:979`）：Go 接受數字後面還有其他字元、也接受正負號；移植時照
  Go 的實際接受範圍寫手動剖析，不用 `str::parse`。
- **`%q`／`%#x`**：三個套件的錯誤訊息有 107 處 `%q`。`fd2_rules::battle` 的 `go_quote`、`go_sharp_x` 是
  `pub(crate)`，見下方缺口。
- **`reflect.DeepEqual`**：比較的都是定長陣列或純值結構，Rust 用 `PartialEq` 結果相同；若遇到切片欄位要注意 Go 的
  nil 與空切片不相等。
- **函式內的區域型別與匿名 struct**（例如 `ValidateBundle` 的 `identity`）不在骨架裡，由移植本體的組自行處理。

### 階段 2 開工前的決定（主代理，2026-11-14）

1. **PNG**：`fd2-campaign/Cargo.toml` 加 `png = "=0.18.1"`（工作區已鎖定版本），`separated_shop_assets` 在本 crate 解碼 indexed 與灰階 PNG；不改 `fd2-formats`。損壞 PNG 只比接受或拒收。
2. **Go 格式化與讀檔輔助**：`fd2_rules::battle` 的 `go_quote`、`go_sharp_x`、`read_file`、`decode_json`、`decode_hex` 改為 `pub`，各組直接引用（`read_file`／`decode_json` 回 `BattleError`，需要時轉成 `CampaignError`）。不要在本 crate 另寫一份。
3. **`json.RawMessage`**：`serde_json` 開 `raw_value`。A 組依 Go 的實際行為選擇表示法：Go `json.Marshal` 會把 `RawMessage` 的輸出壓縮（compact），讀入時則原樣保留位元組；Rust 照 Go 在各呼叫點實際看得到的位元組，能逐位元組比對就逐位元組比對，不要預設列入 `NORMALIZATIONS`。

### fd2-rules／fd2-formats 缺口

Go 用到的具名項目都有公開的 Rust 對應（見 `ext_deps.rs`）。下列是形狀或可見度不足，**沒有改那兩個 crate**，由主代理決定：

| 需要 | 現況 | 暫時做法 |
|---|---|---|
| 解碼 indexed（色彩型態 3）與灰階 PNG：`separated_shop_assets.go` 的 `readSeparatedIndexed`／`readSeparatedBinaryMask`（Go `png.Decode` 後要求 `*image.Paletted`／`*image.Gray`） | `fd2_formats::dato` 的 `indexed_png::decode` 是 `pub(super)`，沒有灰階版；`fd2-campaign` 也沒有依賴 `png` | 二選一：`fd2-formats` 公開 indexed／灰階解碼函式，或 `fd2-campaign/Cargo.toml` 加 `png = "=0.18.1"`（Cargo.lock 已鎖這個版本，不新增套件） |
| Go `%q`、`%#x` 格式化、`os.ReadFile`、`json.Unmarshal`、`hex.DecodeString` 的共用輔助 | `fd2_rules::battle` 的 `go_quote`、`go_sharp_x`、`read_file`、`decode_json`、`decode_hex` 都是 `pub(crate)` | 建議公開；否則 B 組在 `campaign` 內另寫一份同樣的輔助，其他組引用 |
| `json.RawMessage` 逐位元組保留 | 工作區 `serde_json` 沒開 `raw_value`、`preserve_order` | 骨架用 `GoRawMessage = serde_json::Value`；若要逐位元組比對 `CompileCampaignJSON`，在 `fd2-campaign/Cargo.toml` 開 `raw_value`（功能旗標不改 Cargo.lock 的版本），別名改成 `Box<RawValue>` |
| `fdtxt::Font` 沒有 `Default`、`PartialEq`；`fdother::Frame` 沒有 `PartialEq`；`NativeItemPanelDataAssets` 只有 `Clone`、`Default` | 同 R2 | 含這些欄位的結構只 derive 做得到的 trait（骨架已依掃描結果自動決定），測試以欄位或雜湊比較 |
| `fdother::read_resource` 的索引是 `usize` | 同 R2 | Go 負索引在呼叫端先擋下 |

## present 組（2026-11-14）

### 範圍

凍結 Go 版 `remake/internal/` 的四個套件，各自移植成同名模組；每個 Go 檔一個子模組，Go 測試檔逐條移植到
`src/<模組>/<檔>/<t>_test.rs`。

| Go 套件 | Go 正式檔 | Rust 模組 | Go 測試函式 | Rust 測試 |
|---|---|---|---|---|
| `localization` | `content.go`、`entities.go`、`locale.go` | `src/localization.rs`、`src/localization/`（另有 `patterns.rs`） | 16 | 16 |
| `musiccatalog` | `catalog.go` | `src/musiccatalog.rs`、`src/musiccatalog/` | 4 | 4 |
| `indexedmap` | `frame.go`、`hud.go`、`mosaic.go`、`native_command10_12.go`、`native_command_heal_tail.go`、`spawn_intro.go` | `src/indexedmap.rs`、`src/indexedmap/` | 55 | 55 |
| `battlepresent` | 21 檔 | `src/battlepresent.rs`、`src/battlepresent/`（Go 70 個函式全移植，另有 7 個內部小工具） | 70 | 70 |

沒有剩下的 `todo!()`，也沒有 `#[ignore]`：四個套件只依賴格式層與 `fd2_rules::battle`，全部測試現在就能跑。

### 決定

- **錯誤型別**：四個模組都用 `fd2_rules::battle::BattleError`（只保存訊息，`Display` 與 Go `err.Error()` 逐字
  相同），模組裡 `pub use … as Error`。`%w` 包裝格式層錯誤時以 `BattleError::from` 加 `wrap` 串接前綴。
- **輸入結構的所有權**（`indexedmap`）：Go 的 `*fdicon.Bank`、`*NativeSelectorCache`、表面指標是呼叫端擁有的
  資源，Rust 用 `Option<&'a T>` 借用，nil 檢查照 Go 保留；`[]byte` 輸入用 `&'a [u8]`；單位層、前景層與格子用
  `Vec`。Go `ComposeFrame` 成功時把 mode 6 改寫的格子寫回呼叫端共用的 `Cells` 陣列，Rust 的
  `compose_frame`／`compose_native_frame` 收 `&mut` 輸入，寫回 `in_.cells`。
- **callback**：Go func 參數 → `Option<&mut dyn FnMut(..)>`；`FrameInput.SelectionOverlay` 這種結構欄位 →
  `Option<&'a dyn Fn(..)>`。`RunNativeUnitPresentStripBridge` 的逐列延遲 callback 另外收目前的 VGA（唯讀），
  因為 Go 的閉包可以同時讀呼叫端緩衝、Rust 不能；同 `fdother::NativeUnitPresentTick` 的做法。
- **共用陣列的切片**：Go `ComposeNativeUnitPresentStripBridge` 把 `frame[0x8088:]` 交給
  `fdother.ApplyIndexedTransitionPass`，中段重繪閉包改寫整個 `frame`。`fdother` 只公開收單一切片的
  `apply_indexed_transition_pass`（`apply_indexed_transition_pass_at` 是 `pub(super)`），所以 Rust 把 `0x8088`
  之前的位元組另存，重繪時接成完整緩衝再拆回，結果與共用陣列相同（黃金差分逐位元組核對）。
- **Go 常數運算**：`PhaseBannerStepMillis = 1000.0 / PhaseBannerStepTickHz` 是無型別常數，Go 以任意精度
  算出精確商才捨入；Rust 的 `f64` 除法差 1 ulp，所以寫成 Go 捨入後的字面值 `54.925409453960306`。黃金差分
  逐位元核對。
- **JSON**（`localization`、`musiccatalog`）：`DisallowUnknownFields` → `deny_unknown_fields`（巢狀結構也拒收）；
  Go 讀 `null` 或缺欄位時留零值，Rust 每個欄位 `default` 加 `null_as_default`。「解碼後再讀一次要求 EOF」以
  `serde_json` 的 `StreamDeserializer` 實作，區分 Go 的 `trailing JSON` 與 `trailing JSON: <err>`。
- **regexp**：Go 的五個錨定樣式與 `FindAllString` 改成逐字元比對（`src/localization/patterns.rs`），每個函式的
  doc 寫出原樣式；字元類別互不重疊，不需要回溯。`filepath.Join`／`Clean` 以 `/` 做字面清理（`go_join`、
  `go_clean`），錯誤訊息裡的路徑與 Go 相同。
- **`Catalog.Format(key, args ...any)`**：參數改成 `&[FormatArg]`（`Int`、`Str`、`Float`）。`fmt.Sprintf` 只實作
  語系文字可能用到的子集（`go_sprintf`）：`%d %v %x %X %c`（整數）、`%s %v %q %x %X`（字串）、`%f %F`（浮點數），
  旗標 `+ - # 空格 0`、寬度、精度，行為照 Go 1.22（例如 `0` 旗標對字串也補零、`%#06x` 的前綴不算寬度、
  `%#.0f` 保留小數點、精度 0 的 0 什麼都不寫）。不支援的組合輸出 `%!`，`Format` 回「incompatible argument
  types」。目前全部官方語系只用到 `%s`、`%d`、`%.0f`、`%02X`。
- **nil 接收者**：Go 對 nil `*Catalog`／`*ContentCatalog`／`*EntityCatalog`／`*musiccatalog.Catalog` 呼叫方法時回錯；
  Rust 的 `self` 不會是 null，那些分支不移植。
- **battlepresent 的輸入**：Go 的 `*figani.Animation`、`*fdother.Frame` → `Option<&T>`，以值傳入的 Frame → `&Frame`；
  輸入結構帶生命週期，底圖是 `&'a [u8]`，每目標多張底圖是 `Vec<Vec<&'a [u8]>>`，`[]*Animation` 是
  `Vec<Option<&'a Animation>>`。來源紀錄的 `Base` 存一份 `Vec<u8>` 複本（Go 與輸入共用切片，內容相同）。
  指令 5 的 `ResolveTarget` callback → `Option<&'a mut dyn FnMut(i64, u16) -> Result<(u16, bool)>>`，所以
  `build_native_command5_effect_sequence` 收值，其他序列建構函式收 `&input`。指令 32 與 AI type21 尾段的
  內部 `build_native1_cac7_tail_frames` 把 8 個純量參數收成私有結構 `TailContract`，值與順序不變。
- **musiccatalog 的 Go 怪癖**：`inspectVorbis` 以 `append([]byte(nil), packet...)` 存識別封包，空封包得到 nil，
  所以空的第一個封包不算、由下一個封包當識別封包；Rust 照做。sample rate 為 0 時 Go 整數除以零 panic，
  Rust 同樣 panic。

### map 走訪順序

- `localization/entities.go` 以 map 的隨機順序走訪五張表：多筆不合法時回報哪一筆、兩個鍵字串轉成同一個 ID
  （例如 `"7"` 與 `"07"`）時留下哪個名稱由順序決定。Rust 依 JSON 鍵字串排序走訪，只是 Go 可能結果之一。
- `localization/locale.go` 的 `validatePack` 以 map 順序走訪 `entries`，多筆不合法時同上。
- `indexedmap/native_command_heal_tail.go` 的 `byRecord` map 只做查詢，與順序無關。
- `battlepresent/native_command32_tail.go` 以 map 偵測重複目標，Rust 用 `BTreeSet`，只查成員，與順序無關。
- 四個套件都沒有用到 `sort.Slice`。

### 測試

`tools/rust_campaign_test.sh -- --nocapture`（`FD2_REQUIRE_ORIGINAL=1`，掛齊原版、分離素材包與
`remake/assets`）：fmt、clippy `-D warnings` 通過；本組 145 個單元測試全過（Go 測試函式 16＋4＋55＋70＝145），0 ignore，輸出沒有「略過」。
`musiccatalog` 的 render 不入庫，在工作樹副本執行時 `FD2_REMAKE_ASSETS` 指向主工作區的 `remake/assets`。

### 黃金差分

| 黃金 | Go 工具 | Rust 測試 | 腳本 | 案例 | SHA-256 |
|---|---|---|---|---|---|
| `present-loc-v1.json` | `remake/cmd/fd2-golden-present-loc` | `tests/golden_present_loc.rs` | `tools/rust_golden_present_loc.sh` | 2744 | `e446bb6b9d8d6487bfe7651a9a6e1d7533f28eb27393c28b3cfaa51f87207759` |
| `present-map-v1.json` | `remake/cmd/fd2-golden-present-map` | `tests/golden_present_map.rs` | `tools/rust_golden_present_map.sh` | 1196 | `a9aa1288129989508359e37954c217fdea94fb4fd57734443c6f8c7a8fce9030` |
| `present-bp-v1.json` | `remake/cmd/fd2-golden-present-bp` | `tests/golden_present_bp.rs` | `tools/rust_golden_present_bp.sh` | 174 | `b5bdb374863f8cf225aeb50e7e4f80b3c0cd5bd1c0de2fd9682ddb064238cf75` |

- **present-loc**：四個語系的 `pack.json`、`content.json`、`entities.json` 與 `music_catalog.json`、30 個 OGG
  render（清單排序並記大小與 SHA-256）。官方語系的 `LoadOfficial` 結果與每個鍵的 `Format`（含型別錯誤與個數
  錯誤）、`LoadOfficialContent` 每筆 `StoryText`、`LoadOfficialEntities` 各 ID 範圍的查詢；pack／content／entities／
  music 的變造輸入（以 JSON 路徑操作描述，兩邊對同一棵樹套用）；OGG 位元組變造；locale 樣式與兩種佔位符掃描
  （佔位符清單由 Go 工具逐字複製的掃描函式算出、以真實 `Load` 驗證，Rust 以「Load 成立且多一個變數就失敗」
  確認相等）；`Format` 的動詞與旗標組合 1200 個。比對 2420 案例，`Format` 不支援的組合跳過 324 個（Go 成功、
  Rust 不支援，列在 `KNOWN_DIFFERENCES`）。
- **present-map**：33 張真實地圖（`maps/mapN/map.json`）與分離素材包的 FDSHAP、FDICON、範圍圖示、LUT、HUD、
  spawn-intro、指令尾段 LMI、章節輔助表面與山景。每張地圖 3 個固定種子局面，各跑整幀（含 HUD 與觀察者各階段）、
  走行、上行走行、轉場、單位登場全流程、指令 10～12 取樣、補血尾段四種、spawn-intro 12 步、HUD 原語；另有馬賽克、
  ch23 staging 與純量函式。初始緩衝以 xorshift32 由 salt 產生，比工作緩衝與 VGA 的 SHA-256、各階段 SHA-256、
  回傳值與錯誤文字。1196 案例全過。
- **present-bp**：分離素材包 `animations/` 的 FIGANI 與 FDOTHER 動畫、固定種子挑出的 20 個 FIGANI、各指令的
  effect bank（共 1190 個檔，排序並記大小與 SHA-256）。19 種 op 涵蓋每個指令與入口（指令 0／1／2／3／5／6／7／8／9、
  24 背景與前奏、32 尾段、複合指令、物理攻擊本體），其中約 23 個是刻意的拒收案例，逐字比錯誤文字。底圖、工作緩衝、
  合成 RLE 幀與 LMI1 項目由 JSON 記下的種子經同一條 LCG 重建；畫格比 SHA-256，序列比每一步的來源層、HP 段、
  音效旗標與下一個 idle 狀態。174 案例全過。
- Go 端各跑兩次，SHA-256 相同。竄改反向檢查：present-loc 改一個錯誤文字、present-map 改一個 VGA 雜湊、present-bp
  把一個目標的 HP 段改成 4，Rust 測試都在該案例失敗。
- `KNOWN_DIFFERENCES`：present-loc 的 Format 不支援組合、JSON 欄位名大小寫與重複欄位、entities.json 的 map 走訪
  順序；present-map 的逐列延遲 callback 多一個 VGA 參數；present-bp 的 nil 指標參數只以 `Option` 表示、指令 5
  `ResolveTarget` 的錯誤路徑不在黃金案例內。`NORMALIZATIONS`：暫存目錄前綴、標準函式庫錯誤只比本套件前綴、官方對白
  比雜湊、null 與空陣列相同；present-map 的緩衝與格子比雜湊；present-bp 的 null 與空陣列相同。

### 格式層與規則層缺口

- `fd2_formats::fdother::apply_indexed_transition_pass_at` 是 `pub(super)`；`indexedmap` 以前綴另存重組代替
  （見「決定」）。若之後公開，可改成直接呼叫。
- `fd2_rules::battle` 的 `go_quote`、`read_file` 是 `pub(crate)`；`localization` 自己寫了一份 `go_quote`。
- `fd2_rules::battle::BattleError` 沒有 `From<fd2_formats::figani::FiganiError>`；`battlepresent` 以模組內的
  `trait GoResult`（`.go()`、`.wrap(prefix)`，經 `to_string()`）轉換，文字與 Go 相同。
- `fd2_formats::fdother::Frame` 的 `on_blit` 是私有欄位，測試只能用 `Frame::raw()` 建立。

## campaign A 組完成紀錄（2026-11-14）

劇本資料與處理器編譯：campaign 的 campaign、handler_compile、handler_binding、handler_script、story_index_map、
native_map_view_config、ai_commands、menu_state、native_story_dialogue、native_story_hd_text、
native_story_dialogue_coverage，與 editorcanonical 的 bundle、compile。開工於 `83a6c64a`。

### 移植結果

- 71 個函式全部有本體（67 個 `todo!()` 補齊，PORTING.json 對應項改 `ported`）。A 組檔案沒有剩下 `todo!()`。
- A 組呼叫 B 組的 `blit_native_dialogue_portrait_at`（`native_church_scene.rs`）兩處：
  `compose_native_story_dialogue_mouth_frame`、`compose_native_story_dialogue_base_frame`。本體照 Go 呼叫它，
  B 合併前執行到這裡會停在 B 的 `todo!()`。
- serde：A 組檔案裡帶 Go 標籤的結構 derive `Serialize`／`Deserialize`，欄位名照標籤、容器 `#[serde(default)]`；
  字串、切片、map 欄位的 JSON `null` 以 `null_as_default` 讀成空值；`omitempty` 照欄位型別對應
  `String::is_empty`、`Vec::is_empty`、`Option::is_none`、`go_is_zero` 等（以值保存的結構不省略，同 Go）。
  editorcanonical 的型別另加 `deny_unknown_fields`（Go `DisallowUnknownFields`）。
- `json.RawMessage`：`compile.rs` 直接用 `Box<serde_json::value::RawValue>`（模組根的 `GoRawMessage` 別名不歸本組改，
  沿用為 `Summary.diagnostics` 等只看長度的欄位）。`next`／`on_win`／`on_lose`／`asset_ids` 是
  `Option<RawMessage>`：缺席是 `None`，明寫 `null` 是 `Some("null")`，與 Go 的 `UnmarshalJSON("null")` 相同。
  `CompileCampaignJSON` 的輸出與 Go `json.Marshal` 逐位元組相同：map 鍵依位元組排序，`RawMessage` 壓縮，
  `<`、`>`、`&`、U+2028、U+2029 跳脫（`go_compact`、`go_json_string`、`go_marshal_object`）。
- 手寫的 Go 行為輔助：`go_sscanf_opt`（`fmt.Sscanf(outcome, "opt%d")` 的實際接受範圍）、`go_any_int`
  （`any` 數字的 `int`／`float64` 分支）、`go_join_dir`（`filepath.Join(filepath.Dir(p), rel)`，以 `std::path`）、
  `bundle_relative_path`（bundle 逃逸檢查，以路徑元件計深度）、`go_plus_v_counts`（`%+v`）、`go_rect`／
  `go_rect_union`（`image.Rect`／`Union`，自由函式，避免與其他組替 `go_image::Rectangle` 補的方法衝突）。
- Go 的實際行為照做：`0x22253` 分支直接取 `RawArgs[4]`（少於五個引數時 panic，Rust 同樣以索引 panic）；
  `Runner.Visible`／`ShopGoods` 對不存在的節點 panic；`native_any_of` 的 inactive-count 迴圈不 break。

### 測試

12 個 Go 測試檔、177 個測試函式全部逐條移植（`src/<套件>/<檔>/<t>_test.rs`）：

| Go 測試檔 | 測試數 |
|---|---|
| `handler_compile_test.go` | 101 |
| `campaign_test.go`／`story_asset_test.go` | 35／1 |
| `native_story_dialogue_test.go`／`native_story_portrait_rect_test.go` | 22／1 |
| `handler_script_test.go` | 4 |
| `ai_commands_test.go`、`menu_state_test.go`、`native_map_view_config_test.go`、`story_index_map_test.go` | 2、1、2、2 |
| `editorcanonical/bundle_test.go`／`compile_test.go` | 2／4 |

`tools/rust_campaign_test.sh`（`FD2_REQUIRE_ORIGINAL=1`，fmt、clippy `-D warnings`、全部單元測試）：176 通過、
1 個 `#[ignore = "等 B 合併"]`（`compose_native_story_dialogue_page_uses_original_indexed_assets`），沒有略過。
`--include-ignored` 時它停在 `native_church_scene.rs:83` 的 B 組 `todo!("campaign/native_church_scene.go:blitNativeDialoguePortraitAt")`。
期望值都照 Go；Go 以指標同一性比較 `*bool` 的兩三處改比值，`t.Run` 子測試改成同一測試內的迴圈。

### 黃金差分 `campaign-a`

`tools/rust_golden_campaign_a.sh`（`remake/cmd/fd2-golden-campaign-a`、`tests/golden_campaign_a.rs`）。Go 端讀全部
`remake/assets` 劇本，不抽樣（素材在兩端都唯讀掛在 `/assets`，錯誤訊息裡的路徑字樣相同；在工作樹副本執行時
`FD2_REMAKE_ASSETS` 指向主工作區，`bindings/generated-*` 等不入庫的產生檔也一併讀入）：

- `scenarios/*.json` 32 份 `campaign.Load`（2 份戰役圖成功，30 份戰場腳本照 Go 回 `start 節點 "" 不存在`）；
  成功者再對每個帶 beats 的節點跑 `ExpandNativeDialogueGroups`、每個原生對白版面每頁的逐字時鐘與四個進度的可見字形
  （字形清單記 SHA-256），以及每個節點 × 18 個固定結果的 `Runner.Advance`、`Visible`、`ShopGoods`、城鎮暗門判定。
- `cutscenes/handlers/**.json` 64 份 `LoadHandlerScript`，成功者以空 `HandlerBindings` 編譯。
- `cutscenes/bindings/**.json` 368 份 `LoadHandlerBinding` 與 `CompileHandlerBinding`（含 5 個 `_diagnostics.json`
  的拒收），編譯結果再展開並記逐字時鐘。
- `cutscenes/acting/**.json` 12 份 `LoadActingResourceSet`；`dialogue-index` 的 `LoadStoryIndexMap` 與每個映射的
  `Lookup`（含越界與錯誤腳本）。
- `editor-canonical`：`ValidateBundle`、`CompileCampaignJSON`（輸出原文逐位元組比對）與 `Decode`；另有 14 份合成
  canonical 文件（原文存在案例裡）驗 HTML 跳脫、壓縮、轉場解析、嚴格解碼與尾端 JSON。

黃金檔 `campaign-a-v1.json`（11.8 MB）SHA-256 `5958f83f54936ac3dfe4960bfb81e74a038043db5d6a02daa31547550ca71c11`，
492 個案例、545 個輸入檔（排序後的相對路徑、大小與 SHA-256 存進黃金檔，Rust 端先核對），比對 2905 項，全部相同。
Go 端重跑兩次位元組相同（腳本以 `cmp` 檢查）。竄改反向檢查（`FD2_GOLDEN_TAMPER=<案例>|<JSON pointer>`）：戰役圖
`start`、綁定編譯結果的 `beats[1].source`、真實與合成的 `compiled` 位元組、自有錯誤文字各一處，都精確報出該欄位並失敗。
`KNOWN_DIFFERENCES` 與 `NORMALIZATIONS` 列在 `tests/golden_campaign_a.rs`。

### 排序與 map 迭代

- A 組正式程式的 map 迭代（`Decode` 的節點、`LoadHandlerBinding` 的 contexts／overrides／spawn groups、
  `LoadActingResourceSet`、`CompileCampaignJSON` 的轉場欄位）在 Go 只影響「多項同時不合法時先報哪一項」或寫入不同鍵，
  Rust 依鍵排序。黃金案例的錯誤都只有一個不合法項（合成文件刻意只留一個查不到的轉場），所以 Go 端重跑相同。
- `sort.Slice` 沒有用到。

### 缺口與給主代理的事項

- `fd2-rules`／`fd2-formats` 沒有缺口；Cargo.lock 沒有變動。
- 共用檔沒有改。模組根 `editorcanonical.rs` 的 `GoRawMessage` 仍是 `serde_json::Value` 別名（`compile.rs` 另用 `RawValue`）；
  整合時若要統一成 `Box<RawValue>`，`bundle.rs` 的 `Summary` 要一併拿掉 `PartialEq`／`Eq`（`RawValue` 沒有實作）。三個模組根的骨架 `allow(unused, …)` 照計畫在整合時移除。
- 延後測試：C 組 `native_chapter_slot_restore_test.go` 兩條要用 `campaign.Load` 讀戰役圖，A 組已可提供。

## campaign B 組完成紀錄（2026-11-14）

B 組 26 個 Go 檔（`church`、`class_change_table`、`equipment`、`shop`、`native_church_*`、`native_class_*`、
`native_gold_debit`、`native_load_slots`、`native_roster_list`、`native_shop_*`、`native_system_info`、
`native_town_*`、`native_transient_expiry`、`separated_shop_assets`）的 157 個函式全部移植（骨架的 132 個 `todo!()`
補上本體，另 25 個骨架時已直接移植），PORTING.json 對應項目改為 `ported`。B 組不依賴其他組，沒有 ignore 的測試。

### 簽名與寫法的調整

- `ReviveUnit` 改回傳 `(gold, cost, Result<()>)`：Go 失敗時仍回傳有意義的 `cost`（金幣不足時是應付金額），
  `Result<(i64, i64)>` 會丟掉它。其他 Go 失敗時另回傳的值都可由輸入推得（`BuyGood`／`SellGood`／`SellSlot`／
  `SellNativeSlot` 回原 gold、`ReserveGood` 回 -1、`ComposeNativeGold*Frames` 回 oldGold），維持 `Result`，
  測試與黃金依 Go 規則補回。
- `rollClassChangeRange` 的 rng 改收 `&mut Option<&mut dyn GoRand>`：同一個 rng 要連續借給五次擲骰。
- Go 就地改寫 `frame` 再回傳同一切片的函式（`ComposeNativeChurchTextAt` 等）：Rust 改寫呼叫端的 `&mut [u8]`，
  回傳複本。`ComposeNativeShopPurchaseInsufficientGold` 照 Go 改寫呼叫端的 `postChoiceClose`。
- `ComposeNativeTransientExpiryFrame` 的 strings／font 收 `Option`：Go 對 nil 接收者回 `fdtxt: string index N out of
  range` 與 `fdtxt: invalid native glyph blit`，Rust 直接回同樣的 `FdtxtError`（Go 的字型錯誤要畫到第一個字才出現；
  FDTXT 481..486 都不是空字串，結果相同）。
- `NativeSystemInfoCampCounts` 收 `&[Unit]`，Go 的 nil 元素分支不移植。
- Go 在兩、三個函式各寫一份的相同迴圈合併成私有輔助函式（註解寫明 Go 行號）：`native_class_list_slice_frame`、
  `native_roster_panel_frame`、`blit_native_shop_decoration_and_gold`、`native_shop_service_frame`、
  `native_confirmation_spread_frame`，以及 FFFC／十進位展開的 `expand_fffc`、`go_decimal_words`。
- serde：`ReviveFeeTable`、`ClassChangeCurrent`／`ClassChangeTarget`（也 Serialize，omitempty 用 `go_is_zero`）、
  `ShopItemRow`／`ShopEquipRow`、分離素材的三個 JSON 文件型別加 derive；Go 匿名 wire 結構在函式內定義。成長表的
  `AP` 等沒有標籤，Rust 認檔案實際的小寫鍵與 Go 欄位名原文兩種。
- PNG（決定 1）：`separated_shop_assets.rs` 的 `decode_separated_png` 以 `png =0.18.1` 解碼，色彩型態 3 對應 Go
  `*image.Paletted`、灰階 1～8 位元對應 `*image.Gray`（低位元深度依 Go 放大成 0～255），其他型態拒收。
- Go 的 map 順序：`church.go:257` `ClassChangeCandidates` 附加不在 order 的 id 時，Rust 依遞增順序（註解寫明）。
  單元測試的期望值只有一個候選，不受影響。

### 測試

- 19 個 Go 測試檔共 94 個測試函式逐條移植（`separated_town_assets_test.go` 歸 `native_town_scene`），期望值不改；
  `tools/rust_campaign_test.sh`（`FD2_REQUIRE_ORIGINAL=1`、原版與素材包都掛上）94 個全部通過，沒有 ignore、
  沒有略過。
- `fd2_rules::battle::Unit` 有三個私有欄位，外部 crate 不能用 `..Default::default()`；測試以
  `church_test.rs` 的 `unit!` 巨集（先取零值再逐欄指定）表達 Go 的 `battle.Unit{...}` 字面值。
- 共用的測試輔助（`native_class_list_strings`／`font`、原版教會素材載入等）放在各自測試模組，以 `pub(crate) mod`
  供其他 B 組測試引用。

### 黃金差分 campaign-b

- `remake/cmd/fd2-golden-campaign-b`、`tests/golden_campaign_b.rs`、`tools/rust_golden_campaign_b.sh`。
- 真實輸入：原版 FLAME2 四檔、`remake/assets/data` 與 `docs/data/exe_tables` 九個表（各記大小與 SHA-256），分離
  素材包 `fonts`、`shop`、`sprites/fdicon`、`surfaces`、`text`、`ui` 六個目錄（記排序後的檔案數與清單雜湊）。
  Rust 端先核對全部雜湊。
- 2573 個案例：教會、商店（分離包與原版封存兩種來源、四個資源）、城鎮（兩種來源 × 3 變體 × 6 選項 × 4 脈衝、
  離城縮放）、職業清單與名冊、確認框、整備與戰鬥系統問句／回覆、讀檔欄、系統資訊、金額捲動等畫面（每格畫面
  SHA-256），以及游標視窗、表格載入、`BuyGood`／`ReserveGood`／`SellGood`／`SellSlot`／`SellNativeSlot`／
  `EquipItem`／`EquipNativeCompactSlot`／裝備重算／`ReviveUnit`／`ApplyClassChange`（`GoMathRand` 種子另記）／
  `ClassChangeCandidates`／裝備比較等規則（PCG 固定種子產生單位）。1075 個案例是 Go 回錯誤的路徑，錯誤文字逐字比對。
- 2026-11-14 的黃金檔 SHA-256 `b4fc7f005a72efae3c69a96c1d443792dd36c0fe91e4d91753d9bb9fbcc20759`（2,979,426 位元組）；
  Go 端重跑三次 SHA-256 相同，Rust 端 2573 案例全部相同。竄改反向檢查：改一格城鎮畫面的 SHA-256、改一個購買後
  單位的 HP，Rust 端都在該案例失敗。
- `KNOWN_DIFFERENCES`：`ClassChangeCandidates` 不在 order 的 id 的附加順序（order 不齊全的 5 個案例改比「order 部分
  的前綴＋集合」）、JSON 欄位名大小寫、標準函式庫錯誤文字（只比接受或拒收）、nil 切片元素。
  `NORMALIZATIONS`：null 與空陣列／空字串相同、數字只比數值、失敗時的附帶回傳值依 Go 規則重建。

### fd2-rules／fd2-formats 缺口（沒有改那兩個 crate）

| 需要 | 現況 | 暫時做法 |
|---|---|---|
| 測試以結構字面值建 `battle.Unit` | `Unit` 的 `native_constructor_pending`、`native_death_flag_before`、`native_death_flag_valid` 是私有欄位，外部 crate 不能用 `..Default::default()` | 測試用 `unit!` 巨集逐欄指定；正式程式不受影響 |
| Go `image.Rectangle` 的 `In`／`Overlaps` | `go_image.rs` 只有欄位（共用檔，B 組不改） | 只有測試用到，在 `native_town_scene_test.rs` 寫兩個區域函式 |

## campaign C 組完成紀錄（2026-11-14）

範圍：campaign 的 `native_chapter_slot_{restore,save}`、`native_continue_{field_boundary,handoff,map_timing,pending_groups,runtime_units}`、
`native_current_snapshot`、`native_join_{base,constructor}`、`native_persistent_party`，以及 ending 全部 14 檔。

- **函式**：C 組 25 檔共 126 個函式全部移植（階段 1 已直接移植 11 個，本輪補 115 個），C 組檔案沒有 `todo!()`；
  `PORTING.json` 的 C 組函式改為 `ported`（階段 2 分組表的統計數字是規劃快照，未改）。
- **測試**：C 組 Go 測試 112 個函式逐條移植，109 個通過、3 個依「延後的測試」標 `ignore`：
  `native_chapter_slot_restore_test` 的兩個停在 A 組 `campaign.go:Load`，`ending/native_dialogue_test` 的
  `TestNativeDialogueUsesSixBandOpeningAndFiveBandClosingRestore` 停在 B 組 `NativeClassListOpeningFrames`（以
  `--include-ignored` 確認）。`tools/rust_campaign_test.sh`（`FD2_REQUIRE_ORIGINAL=1`）沒有略過。
- **測試移植的對應**：Go 的 `FD2_NATIVE_SAVE_FIXTURE`（使用者原版 FD2.SAV）改讀 `FD2_FLAME2_DIR/FD2.SAV`，即知識庫 `56`
  記載的同一份快照，期望值不變；`TestMaterializeNativeContinueFieldBoundaryRejectsBeforeMutation` 的
  "unvalidated input" 子測試無法建構（`ContinueRuntimeInput` 的 `validated` 欄位私有、沒有 `Default`），同一道閘門由
  "tampered validated input" 覆蓋；`TestRecoveredPrefixStopsAtNativeOnlyGate` 改核對錯誤訊息指向第 2 段（Rust 的
  `Result` 不帶停下的段索引）。`battle::Unit`／`State`、`fdother::Frame` 有私有欄位、`afm::Clip`／
  `save::ChapterSlotSnapshot` 沒有 `Default`，測試一律從零值逐欄設定。
- **所有權**：CONTINUE 轉接器照 Go「先在副本試算、成功才 `*state = candidate`」用 `State::clone()`；JOIN 與持續隊伍的
  `Unit` 是擁有的值。結局的 `Player`、`Phase0Player`、`MontageCycle`、`MontageTailPlayer` 共用 `SharedCompositor`，
  每個步驟只在步驟內持有鎖；`Player` 呼叫 `Phase0Player::advance` 前不持鎖（兩者共用同一個合成器）。Go 把
  `c.Work`／`c.Offscreen` 當參數傳回同一個合成器的地方（`MontageCycle` 的人物淡入），Rust 先把 `work` 移出、事後放回，
  三塊緩衝互不重疊，結果相同。
- **簽名修正**：`MontageTailPlayer::render_source_bound_pair` 的 `opaque_fill` 由骨架的 `&u8` 改為 `Option<u8>`
  （Go `*byte` 可為 nil）。其餘簽名照骨架。
- **map 迭代順序**：`BuildNativeChapterSlot` 建 `byIdentity` 與 `MaterializeNativeContinuePendingGroups` 走 schedule 時，
  多個成員或多列同時不合法時 Go 報哪一個不固定；Rust 是遞增順序。黃金工具對失敗的呼叫重做 64 次，偵測到文字不一致就記為
  `map-order`，Rust 端只比拒收（607 個案例中 4 個）。其餘 map 迭代只影響錯誤選擇，黃金輸入不觸發。
- **黃金差分**：`tools/rust_golden_campaign_c.sh`（`remake/cmd/fd2-golden-campaign-c`、`tests/golden_campaign_c.rs`）。
  真實輸入：`remake/assets` 的 data／maps／scenarios／endings、`docs/data/glyph_map.json`、原版 FD2.SAV／FDOTHER.DAT／
  ANI.DAT 與分離素材包，共 7735 個檔案，排序後連同大小與 SHA-256 寫進黃金檔。以固定種子產生 607 個案例：合成存檔槽的
  `BuildNativeChapterSlotRestorePlan`（80，含真實存檔目前名冊組出的槽）、`BuildNativeChapterSlot`（70）與
  `EncodeNativePersistentRecord`（65）、JOIN 基底（35）與建構（全部 32 個角色加越界值、殘值與三種物品表，各 90）、
  持續紀錄投影（60）、CONTINUE 五段物化（60：真實存檔與變體、map25 的 event61；每段記 State 摘要，最後加
  `BuildNativeCurrentRuntimeRecords`／`BuildNativeCurrentPersistentRecords`）、四個結局資料的解碼與計畫、終章
  `Player`（真實 FDOTHER#54／ANI#2 到 0x2c548，346 步）、兩輪 0x2c548 蒙太奇（含輸入跳過，2437／1469 步）、原版與分離素材
  兩份 0x2c194 片尾（各 987 步）、片尾載入基準、頭像文字框、終章對話框與 0x2c405 第 0 段；每步記 VGA 與調色盤的
  SHA-256。黃金檔 `campaign-c-v1.json` SHA-256 `9004ce7198fcf75f72153b6c022c77ce0c5dea1d384ff2fa653ed91b0ba79064`，
  Go 端重跑兩次相同；竄改片尾第 400 步的 VGA 雜湊、存檔槽 roster 的一個位元組、handoff 後 `Units[1].HP`，差分都精確報出該欄位
  並失敗。`KNOWN_DIFFERENCES`：JSON 欄位名大小寫、純量欄位的 null、handoff 的 nil roster／pending groups、錯誤時的附帶
  回傳值、未驗證的 `ContinueRuntimeInput` 零值；`NORMALIZATIONS`：null 與空值相同、數字只比數值、缺席的鍵與 null 相同、
  map 順序決定的錯誤只比拒收。終章對話框的開收框轉場呼叫 B 組，黃金不涵蓋（由 B 組的職業清單轉場黃金負責）。

## 整合紀錄（2026-11-14）

- present、campaign A／B／C 四組合併後沒有剩下的 `todo!()`；PORTING.json 的 354 個函式全部 `ported`（present 組四個套件沒有列在 PORTING.json，函式全部移植）。
- 延後測試全部拿掉 `#[ignore]`：`tools/rust_campaign_test.sh` 在 `FD2_REQUIRE_ORIGINAL=1` 下 528 項通過、0 ignore、0 略過，與 Go 七個套件的 528 個測試函式數相同。
- 三個模組根的骨架 allow 已移除，clippy `-D warnings` 乾淨。浮出的未使用匯入刪除；`separated_fdfield` 只有 crate 內部使用者，glob 改成 `pub(crate) use`；`montage_tail` 的兩個資源常數在 Go 也沒有使用者，照移植保留並標 `allow(dead_code)`。
- `localization` 原本自寫的 `go_quote` 改用 `fd2_rules::battle::go_quote`（兩份實作相同）。
- 七份黃金差分（present-loc、present-map、present-bp、campaign-a、campaign-b、campaign-c）在整合後的主 checkout 全部重跑通過，SHA 與各組回報相同；`fd2-rules` 647 項、`fd2-formats` 421 項照舊通過。
