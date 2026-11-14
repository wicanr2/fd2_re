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
