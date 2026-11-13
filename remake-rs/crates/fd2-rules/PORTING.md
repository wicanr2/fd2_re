# fd2-rules 移植說明（R2 階段 1 骨架）

`fd2-rules::battle` 對應凍結 Go 版 `remake/internal/battle`。階段 1 把型別、常數、表格與全部函式簽名搬過來，函式本體是 `todo!("battle/<file>.go:<FuncName>")`；階段 2 由多個子代理依下方分組平行補上本體、移植測試並加黃金差分。單位的所有權設計與其他型別決定寫在 [`src/battle.rs`](src/battle.rs) 的模組文件，階段 2 照用，不重新決定。

逐檔、逐函式的機器可讀清單在 [`PORTING.json`](PORTING.json)：Rust 路徑、Go 正式行數、測試檔與行數、每個函式的 Rust 路徑與狀態、檔內函式的呼叫對象、檔案相依，以及每檔的階段 2 組別。

## 骨架現況

以凍結 Go 版 `dccfd895` 的 `remake/internal/battle` 量測（2026-11-14）。

| 項目 | 數量 |
|---|---|
| Go 正式檔／行數 | 129 檔／20931 行 |
| Go 測試檔／行數 | 133 檔／16355 行 |
| 型別 | Go 具名型別 204 個，另有 4 個 Go 匿名 struct 在 Rust 給了名稱 |
| 常數 | 83 |
| 套件層變數 | 11（8 張表格、3 個可改寫的全域 func／亂數源） |
| 函式與方法 | 632，其中 29 個本體已直接移植，603 個是 `todo!()` |

- 直接移植的只有一行委派、常數或欄位運算（例如 `Unit::alive`、`State::claim_treasure`），PORTING.json 標 `ported`。Go 會繞回的整數運算一律用 `wrapping_*`。
- 每個函式的 doc 保留 Go 註解，接著一行 `Go：<檔>:<行>` 與原始 Go 簽名；有所有權轉換時另有「所有權：」說明索引指向哪個陣列。
- Go 的 JSON 標籤保留在欄位 doc 的「Go 標籤：」。
- `cargo build`、`cargo clippy -- -D warnings` 通過。`lib.rs` 的 crate 層 `allow(unused, dead_code, …)` 是骨架專用，全部組完成後移除。
- 骨架由一次性的 Go `go/types` 分析工具產生（參數可變性、nil 檢查、回傳值是否新建、單位指標的流向都由型別資訊推導）。產生器沒有進版控；從現在起這些檔案改為人工維護。

## 階段 2 分組

依函式呼叫圖與「每個測試檔實際需要哪些函式」（測試呼叫的函式及其遞移呼叫）分組：先把呼叫圖拓撲排序，再固定波次做局部搜尋，壓低「測試要等同波次其他組或後面波次」的行數與呼叫到未完成組的次數，並讓各組行數相近。Go 的函式之間的呼叫沒有環，但以檔案為單位時有一個 33 檔、約 19800 行的循環區（`model`、`event`、`combat`、`growth`、指令與 AI 互相呼叫），所以無法讓每一組的測試都只依賴前面的組；剩下的跨組測試列在下方「延後的測試」。

| 組 | 主題 | 波次 | Go 檔 | 正式行數 | 測試檔 | 測試行數 | 函式 |
|---|---|---|---|---|---|---|---|
| A | 單位、記錄與物品欄基礎（已移植，見「A 組完成紀錄」） | 1 | 41 | 4627 | 42 | 3040 | 152 |
| B | 指令、目標與死亡程式 | 2 | 22 | 4411 | 25 | 3337 | 165 |
| C | 物品效果與面板繪製 | 2 | 28 | 3087 | 27 | 2367 | 82 |
| D | 敵我 AI | 3 | 21 | 3608 | 19 | 3017 | 90 |
| E | 戰鬥、成長與法術 | 3 | 9 | 2606 | 11 | 2102 | 86 |
| F | 劇本事件、回合與快照 | 3 | 8 | 2592 | 9 | 2492 | 57 |

順序：

1. **波次 1：A。** 其他每一組的測試都要用到 A（`Unit`、`State` 基本操作、物品欄與原版記錄）。
2. **波次 2：B、C 平行**，A 合併後開始驗證。C 只有一個測試要等 B。
3. **波次 3：D、E、F 平行**，A、B、C 合併後開始驗證。D 與 E 互相依賴（AI 執行期的測試要跑戰鬥，`combat_test` 要跑 AI 計畫），兩組都合併後再跑彼此的延後測試；F 最後收尾。
4. **整合：** 全部合併後跑下表的延後測試，再移除 `lib.rs` 的骨架 allow。

各組檔案：

- **A**：model、command_learn、native_ai_destination、native_ai_record、native_class_compat、native_command、native_command_availability、native_command_mp、native_command_record、native_coordinate_write、native_equipment、native_equipped_item、native_future_constructor、native_inventory_{assign,available,flags,records,remove,search,slots,transfer}、native_item_panel_record、native_item_row、native_map_presentation、native_marker_rewrite、native_movement_cost、native_persistent_{copy,lookup}、native_raw_{application,damage,flag_restore,pair_step,restore,word_step,word_subtract}、native_record_{byte5,byte5_bit0,flags}、native_transient、native_unit_byte_clear、native_word_delta_remove
- **B**：balance、move、native_ai_path、native_ai_scored_candidates、native_command0／13／17／20／20_ai_plan／25／26／32／33／34／35、native_command_{compound,damage,modifier,target}、native_death_program、native_item_target、native_map_view
- **C**：native_battle_panel、native_ch28_post_{prelude,present}、native_command_{grid,overlay_render,palette_flash,scene}、native_item_{ap_dp_step,base_stat_unit,capacity_step,capacity_unit,command_damage,dispatch,hit_ev_step,hp_restore,marker_application,marker_clear_restore,mp_restore,panel_data_render,panel_render,relocation,selector,table}、native_map_unit_window、native_physical_scene、native_presentation_queue、native_revive_list_render、native_transfer_item_render
- **D**：native_ai_14ef0_dispatch、native_ai_1598a_score、native_ai_action_runtime、native_ai_choice、native_ai_idle_recovery、native_ai_item_{preselection,score}、native_ai_mode11、native_ai_mode11_runtime、native_ai_mode5、native_ai_mode_{record,write}、native_ai_phase_diagnostic、native_ai_physical_{candidates,item_source,score}、native_ai_runtime、native_ai_scored_group_score、native_ai_spell_score、native_command24、native_player_movement
- **E**：combat、growth、magic、native_battle_result、native_physical_{attack,damage,experience}、native_player_attack_field、terrain
- **F**：event、native_end_turn_recovery、native_field_event、native_result、native_system_group_march、native_turn_pair_mutation、native_turn_progression、snapshot

測試檔歸屬：`x_test.go` 歸 `x.go`；名稱不對應的歸最長前綴相符的正式檔，再不行就歸測試呼叫最多函式的那個檔（例如 `treasure_test.go` 歸 `model.go`、`native_combat_tables_test.go` 歸 `terrain.go`）。

### 延後的測試

這些測試檔由所屬組移植，但要等「等待」欄的組合併後才會通過；移植時先標 `#[ignore = "等 <組> 合併"]`，整合時拿掉。合計 2572 行，約占測試的 16%。

| 所屬組 | 測試檔 | 行數 | 等待 |
|---|---|---|---|
| A | `treasure_test.go`（9 個測試中 4 個 ignore：兩個 `PlayerTreasure*`、`Map25NativeEventTreasure*`、`SkyKeyMaterialAssets*`） | 273 | B、D、F |
| A | `native_command_test.go`（7 個測試中 2 個 `StateGainExp*` ignore） | 117 | B、E |
| B | `balance_test.go` | 95 | E |
| B | `move_test.go` | 183 | F |
| B | `native_death_program_test.go` | 318 | E、F |
| C | `native_item_command_damage_test.go` | 60 | B |
| D | `native_ai_mode5_test.go` | 289 | E |
| D | `native_ai_runtime_test.go` | 649 | E |
| D | `native_ai_target_terrain_test.go` | 38 | E |
| E | `combat_test.go` | 396 | D |
| F | `native_system_group_march_test.go` | 154 | D、E |

組內函式呼叫到「尚未完成的組」（同波次其他組或後面波次）的次數：A→B 1、A→D 3、C→B 2、D→E 8、E→D 8、F→D 2；B 沒有。這些呼叫在對方完成前會碰到 `todo!()`，所屬測試已列在上表。

## 階段 2 的工作規則

- 每組只改自己擁有的 `src/battle/<file>.rs`，以及本檔與 PORTING.json 裡自己那幾列的狀態（`todo` → `ported`）。
- 測試放在所屬正式檔的子模組：`x.go` 名下的 `t_test.go` 寫成 `src/battle/x/t_test.rs`，在 `x.rs` 加 `#[cfg(test)] mod t_test;`。這樣各組不會改到同一個檔。
- 型別、常數與簽名是共同契約。真的要改別組擁有的簽名時，先在本檔「簽名變更紀錄」追加一列（日期、函式、原因），再通知該組；不要直接改別組的檔。
- 錯誤訊息用 `BattleError`（`battle_err!` 巨集），`Display` 與 Go 錯誤字串逐字相同。Go 的 `%q`、`%x`、`%#v` 等格式要逐處對照 Go 輸出。
- 黃金差分沿用 R1 的做法：在 `remake/cmd/` 新增只輸出資料的 Go 工具，Rust 測試逐欄比對；種子與輸入雜湊寫進收據。

## 開工前的決定（2026-11-14）

1. **Go `math/rand` 相容序列：移植，放在 `src/battle/rng.rs`，由 A 組實作。** 119 的 R2 完成條件是「Go 的戰鬥決定性測試以相同種子產生相同結果」，約 90 處 Go 測試的期望值綁在 `rand.NewSource(seed)` 的序列上，只有重現序列才能不改期望值。這是 `rust-port-no-go-stdlib` 教訓唯一具名的例外，範圍限定為：
   - `GoSource`：Go `rngSource` 的種子化（`seedrand`）、607 項加法延遲費氏產生器與 `Int63`／`Uint64`，以及 `rngCooked` 表。表格由容器內 Go 1.22.12 的 `$GOROOT/src/math/rand/rng.go` 機械轉出，不手抄；檔頭保留 Go 的 BSD 授權聲明。
   - `GoRandImpl`（名稱由 A 組定）：Go `(*rand.Rand).Intn`、`Int31n`、`Int63n`、`Int31` 的取樣，實作 `GoRand`。`battle` 只用 `Intn`，其他方法不移植。
   - 黃金差分：`remake/cmd/fd2-golden-gorand` 輸出多個種子（含負數、0、`seed` 迴圈用到的 0～999）在不同 `n`（含 2 的冪、1、`1<<31` 以上）的 `Intn` 序列，Rust 逐值比對。
   - `with_engine_rand`（Go 的 `engineRand`，以時間播種）用同一個型別，種子取 `SystemTime`；它只服務舊版相容介面，不要求跨版本一致。
   - 原版自己的亂數不經過這裡，仍是 `fd2_formats::native_rng`。
2. **JSON：`serde` + `serde_json`，`[]byte` 欄位用工作區已鎖定的 `base64 =0.23.1`。** 依賴寫法照 `fd2-formats/Cargo.toml`（`=` 釘版）。共用的 serde 輔助放在 `src/battle/support.rs`，由 A 組建立，其他組只引用：
   - `go_bytes`：Go `[]byte` ↔ 標準 base64 字串（含 `null` 讀成空或 `None`，依欄位是否 `Option`）。
   - `null_as_default`：Go 把 nil 切片／map 寫成 `null`，Rust 讀成空值。
   - `omitempty` → `skip_serializing_if`；`json:"-"` → `#[serde(skip)]`；Go 接受未知欄位，serde 預設也接受，不加 `deny_unknown_fields`。
   - 刻意不照做（列入各組的 `KNOWN_DIFFERENCES`）：Go 解碼時欄位名不分大小寫，Rust 只認標籤原文；損壞 JSON 只比接受或拒收，不比錯誤文字。重製端資產都由 Go 以標籤原文寫出，不受影響。
   - `fd2-formats` 的 `NativeMapSpriteCycleState`、`NativeBinaryTickState`、`NativeTerrainPhaseState` 補 serde，`fdother::Frame` 補 `Debug`，`fdtxt::Font` 補 `Debug`，由 F 組在 `fd2-formats` 內補（只加 derive 與必要的 serde 屬性，欄位名對照 Go 標籤），不改既有行為；改完 `tools/rust_formats_test.sh` 必須照舊全過。
3. **DebugAI 掛鉤：** Go `func(format string, args ...any)` 在 Rust 收 `fmt::Arguments`；`cmd/fd2` 端的對應在 R4 處理。

## 格式層缺口

`src/battle/format_deps.rs` 在編譯期列出 `battle` 用到的全部格式層項目，都已有公開的 Rust 對應。形狀不同、呼叫端要注意的地方：

| Go | Rust | 差異 |
|---|---|---|
| `fdother.ReadResource(path, int)` | `fdother::read_resource(path, usize)` | 資源索引是 `usize`，Go 負索引無法傳入 |
| `fdicon.NewNativeSelectorCacheFromKeys` | `fdicon::NativeSelectorCache::from_keys` | 關聯函式 |
| `fdother.Frame` | 只有 `Clone`、`Default` | 沒有 `Debug`、`PartialEq` |
| `fdtxt.Font` | 只有 `Clone` | 沒有 `Debug`、`Default` |
| `fdtxt.Strings` | 沒有 `Default` | |
| 上列三個狀態型別 | 沒有 serde | 見上節第 2 點 |

因此 `NativeItemPanelDataAssets`（含 `fdother::Frame` 與 `fdtxt::Font`）沒有 `Debug`；它把兩者包在 `Option` 裡，所以仍有 `Default`。

## Go 寫法對應時的注意事項

- **單位指標**：規則見 `battle.rs`。另有兩類函式的 Rust 簽名多了開頭參數 `units`：經由結構欄位裡的 `UnitIdx` 讀寫單位的自由函式（例如 `apply_native_command_damage_mp`、`complete_native_command_damage`，共 34 個），以及兩個以上 `*Unit` 參數可能指向同一單位並被改寫的函式（`transfer_native_inventory_item`、`apply_native_item_base_stat_delta_to_unit`、`apply_native_item_capacity_to_unit`）。
- **nil 接收者**：Go 有不少 `if s == nil` 的防護，Rust 的 `self` 不會是 null，這些分支與「對 nil 呼叫方法」的 Go 測試不移植；函式 doc 有註明。
- **nil 參數**：Go 會檢查 nil 的指標、func 參數包成 `Option`；nil map／切片讀起來等於空的，除非函式明寫 `== nil`，否則用 `&BTreeMap`／`&[T]`。
- **map 迭代順序**：Go 隨機、Rust 有序；會受順序影響的 7 處列在 `battle.rs`。
- **淺複製**：Go `candidate := *s` 共用子指標，Rust `clone()` 是深複製；移植時逐處確認沒有依賴共用的副作用。
- **函式內的區域型別**（例如 `native_death_program.go` 的 `type setter func()`）與函式內的匿名 struct 不在骨架裡，由移植本體的組自行處理。

## 簽名變更紀錄

階段 2 若必須修改骨架既定的型別或簽名，在此追加。

| 日期 | 項目 | 原因 | 組 |
|---|---|---|---|
| 2026-11-14 | `record_bounds_error` 回傳 `Result<()>` → `BattleError` | Go `recordBoundsError` 回傳的是 `error` 值本身（永不為 nil），呼叫端寫成 `Err(record_bounds_error(i))` | A |
| 2026-11-14 | `native_ai_physical_destinations`、`native_movement_destination_field`、`native_movement_budget_field` 的 `base_flags: &mut [u8]` → `&[u8]` | Go 先複製一份再改，不改呼叫端的切片；`&mut [u8]` 仍可自動轉成 `&[u8]`，既有呼叫端不用改 | A |
| 2026-11-14 | `magic.rs` 的 `with_engine_rand`（E 組檔）由 A 組實作本體，簽名不變 | PORTING.md 決定 1 指定與 `rng.rs` 一起完成 | A |
| 2026-11-14 | `native_path_directions` 的 `flags`、`terrain_move_codes`、`cost_row`：`&mut [u8]` → `&[u8]` | Go 只讀不寫（存進搜尋結構的是副本）；`&mut` 呼叫端仍可自動轉型 | B |
| 2026-11-14 | `resolve_native_command_damage_plan` 的 `targets: &[Unit]` → `units: &[Unit], targets: &[UnitIdx]` | 結果的 `target` 必須是 `state.units` 的索引；子集合切片給不出這個索引。呼叫端都在 `native_command0.rs` | B |
| 2026-11-14 | `native_command34_states`、`native_command35_states`、`native_ai_command2022_states`、`native_command_modifier_records`、`native_compound_command_target_records`：`targets: &[Unit]` → `units: &[Unit], targets: &[UnitIdx]`；`publish_native_command_modifier_records` → `(units: &mut [Unit], targets: &[UnitIdx], records)`；`native_ai_command2022_states_from_records` 的 `targets` → `&[UnitIdx]` | 同上：Go 的 `[]*Unit` 是 `state.Units` 的子集合，狀態快照要存原陣列索引、發布要寫回原單位。全部是 B 組內部函式，呼叫端都在 B 組 | B |
| 2026-11-14 | 新增 `native_command_modifier_shadow_states`（`native_command17.rs`，`pub(crate)`） | Go 的「淺複製目標→寫記錄→取狀態→換回 Target」在 4 處重複，Rust 以深複製副本陣列集中做 | B |
| 2026-11-14 | `BalanceOptions`、`GrowthRates`、`BalanceStats`、`NativeCommandResistanceRow`、`NativeActionWhen`、`NativeDeathOp` 加 serde | 欄位名對照 Go 標籤；沒有標籤的欄位用 Go 欄位名（`HP`、`MP`…）。`model.rs` 的 `go_balance_stats` 鏡像可改用 `BalanceStats` 自己的 serde，A 組檔未改 | B |
| 2026-11-14 | `StatRoll`：`dyn FnMut(i64) -> i64` → `dyn FnMut(&Unit, i64) -> i64`；`StatRange::roll(self, roll)` → `roll(self, unit: &Unit, roll)` | Go 的 `AwardExpNative` 在擲骰閉包裡把正在升級的 `*Unit` 傳給 `NativeGrowthRollObserver`；Rust 的閉包不能同時借用 `applyGrowthRow` 正在改的單位，改由呼叫端傳入擲骰當下的單位。全部是 E 組內部項目 | E |
| 2026-11-14 | `gain_exp_inner`、`gain_exp_for_caller`：`(u, amount, roll, learn, row_for[, friendly_only])` → `(host: Option<&mut dyn GainExpHost>, amount, roll[, friendly_only])`，新增 `pub(crate) trait GainExpHost` | `State.AwardExp` 的兩個回呼是 `State` 方法（A 組 `learn_native_commands_at_level(&mut self, idx)`、`native_growth_row_for`），要在單位被改寫的過程中讀寫 `State`；把「單位在哪裡」與回呼合成一個 host，每次需要時才取單位。骨架的 `learn` 收 `&Unit`，Go 的 learn 會改單位，也一併修正 | E |
| 2026-11-14 | `State::ai_targets` 回傳 `(UnitIdx, UnitIdx)` → `(Option<UnitIdx>, Option<UnitIdx>)` | Go 的兩個回傳值都可能是 nil（沒有敵對目標、或全部估計傷害 ≤ 2） | E |
| 2026-11-14 | `Spell` 加 serde | `LoadSpells` 讀 `spells.json`；欄位名照 Go 標籤，`Name` 沒有標籤，寫出 `Name`、讀取另收 `name` | E |
| 2026-11-14 | `build_native_ai_physical_attack_candidates` 的 `movement_flags: &mut [u8]` → `&[u8]` | Go 只把它交給 `NativeAIPhysicalDestinations`（A 組已改成 `&[u8]`），不寫入；`&mut` 呼叫端仍可自動轉型 | D |
| 2026-11-14 | `apply_native_command28_target_stage` 的 `units: &[Unit]` → `&mut [Unit]` | Go 經由 plan 的 `*Unit` 寫目標 HP 與 raw +5；骨架給了唯讀切片，無法寫入。Go 端的呼叫者只在 `native_command24.go` 與 `cmd/fd2` | D |
| 2026-11-14 | `NativePlayerMovement::path` 的 `&mut self` → `&self` | Go 只讀 `Reach` 與路徑輸入 | D |
| 2026-11-14 | D 組 runtime 的 crate 私有方法 `next_native_ai14_ef0_plan`、`next_native_ai_mode_fallback_plan`、`next_native_ai_mode11_plan`、`next_native_ai_physical_plan`、`native_ai_plan_for_destination`、`native_ai_movement_approach_point`、`native_ai_plan_toward_raw_destination`、`native_ai_mode11_blocked_or_recovery`：`&mut self` → `&self`，其中的 `base_flags`／`runtime_flags`／`cost_row: &mut [u8]` → `&[u8]`；`native_ai_mode_blocked_search_flags` 的 `base_flags` → `&[u8]` | Go 不改 `State`，也只讀這些切片（`nativeAIModeBlockedSearchFlags` 先複製再改）。E 組 `combat.rs`、F 組 `native_system_group_march.rs` 的呼叫端持有 `&mut self`／`&mut Vec<u8>` 時照樣可以呼叫 | D |
| 2026-11-14 | 新增 `native_ai_reborrow_skip`（`native_ai_scored_group_score.rs`，`pub(crate)`） | Go 的 `func(int) bool` 可以在迴圈裡重複傳遞；Rust 的 `Option<&mut dyn FnMut>` 每次要重新借出，`as_deref_mut` 因 trait object 生命週期不變性編不過 | D |

## A 組完成紀錄（2026-11-14）

- 41 個檔的 143 個 `todo!()` 全部移植（PORTING.json 標 `ported`），另完成 `rng.rs`（`GoSource`、`GoMathRand`）、`support.rs` 的 serde 輔助（`go_bytes`、`go_bytes::option`、`null_as_default`、`go_is_zero`）與 `magic.rs` 的 `with_engine_rand`。`rngCooked` 表在 `src/battle/rng/cooked.rs`，由 `tools/rust_gen_gorand_cooked.sh` 產生。
- A 組型別加了 serde：`Unit`、`Camp`、`Cell`、`NativeMapPresentationState`、`NativePositionRecord`、`DeathEffect`、`NativeConstructorTable`、地圖 JSON 的各列型別與 `Load*` 的檔案型別。`Unit` 沒有 Go 標籤的欄位用 Go 欄位名（例如 `HP`、`MaxHP`），與 Go `json.Marshal(Unit)` 相同；`Cell` 寫出 `X`／`Y`、讀取另收 `x`／`y`（`units.json` 的 `own_deploy`）。B 組的 `BalanceStats` 沒有 serde，`Unit::enemy_balance_base` 在 `model.rs` 以鏡像結構轉換。
- 測試：A 組 42 個 Go 測試檔逐條移植（`src/battle/<x>/<t>_test.rs`），只用到 A 組的測試現在就跑；上表列出的 6 個延後測試標 `#[ignore = "等 … 合併"]`。共用輸入由 `src/test_support.rs` 提供（變數名同 fd2-formats）。`model_test` 的 `TestLoadSerial0` 讀不入庫的 `remake/assets/map0_units.json`，在工作樹副本執行 `tools/rust_rules_test.sh` 時要把 `FD2_REMAKE_ASSETS` 指向主工作區。
- 黃金差分：`tools/rust_golden_gorand.sh`（Go `math/rand` 序列）與 `tools/rust_golden_rules_a.sh`（A 組規則）。後者的合成寶箱領取前會關掉事件格網，因為格網的驗證與推進屬 D 組 `native_ai_mode5`。

## B 組完成紀錄（2026-11-14）

- 22 個檔的 158 個 `todo!()` 全部移植（PORTING.json 標 `ported`），B 組 165 個函式沒有剩下的 `todo!()`。B 組沒有呼叫到其他未完成組的函式。
- B 組型別加了 serde：`BalanceOptions`、`GrowthRates`、`BalanceStats`、`NativeCommandResistanceRow`、`NativeActionWhen`、`NativeDeathOp`，欄位名對照 Go 標籤（沒有標籤的用 Go 欄位名）。`model.rs` 的 `go_balance_stats` 鏡像與 `BalanceStats` 的 serde 輸出相同，之後可以改成直接用 `BalanceStats`。
- 測試：B 組 25 個 Go 測試檔逐條移植，共 136 個測試；4 個標 `#[ignore]`：`balance_growth_keeps_original_roll_sequence_and_zero_growth`、`kill_with_exp_cancel_awards_no_experience`（等 E）、`load_reads_cost_from_map_json`、`native_death_programs_cover_every_battle_map`（等 F）。`tools/rust_rules_test.sh` 在 `FD2_REQUIRE_ORIGINAL=1` 下 272 通過、10 ignore（A 組 6、B 組 4）、沒有略過。
- 黃金差分：`tools/rust_golden_rules_b.sh`，Go 端 `remake/cmd/fd2-golden-rules-b`。33 張真實地圖（排序後的清單與 SHA-256 寫進黃金檔）各出原樣與補齊原版 provenance 兩份戰場，另加 24 份合成戰場，以固定種子產生 8541 個案例，涵蓋 22 個檔的公開函式（含錯誤路徑，B 組錯誤文字逐字比對）。沒有涵蓋：crate 私有的 `balance_growth_row`、`NATIVE_FOCUS_TRACE_HOOK` 掛鉤。
- 順序風險：`BalanceOptions::validate` 迭代 `skill_hit`，同時有「角色無效」與「招式無效」時 Go 報哪一則隨 map 順序而定，Rust 依鍵升冪；黃金每組最多放一筆不合法。`native_command_target_cells`、`reachable` 回傳 map，黃金依 (y, x) 排序比集合。B 組沒有用 `sort.Slice`。
- `DEBUG_AI` 診斷文字：Go 以 `%+v` 印出含單位指標的結果，Rust 印 `Debug` 格式，只供診斷，不比對。

## C 組完成紀錄（2026-11-14）

- 28 個檔的 82 個 `todo!()` 全部移植（PORTING.json 標 `ported`），沒有改骨架的型別或簽名，也沒有改 C 組以外的檔。C 組型別加了 serde：`NativeItemEffectRowJSON`、`NativeCommandSceneTable`／`NativeCommandSceneSource`、`NativeCommandPaletteFlashTable`／`NativeCommandFlashSource`，欄位名就是 Go 標籤，`chapter_initial_background` 用 `go_bytes`。
- C 組自己的共用輔助放在自己的檔裡：`native_item_dispatch.rs` 的 `go_sharp_v`／`GoV`（Go `%#v` 對道具路由結構的輸出，路由驗證失敗的錯誤訊息用它組成）、`native_item_panel_data_render.rs` 的 `blit_native_item_panel_text_opt` 與 `NativeItemPanelDataAssets::raw_cells_or_empty`／`frames_or_empty`。Go 的面板函式多半不先檢查 `assets.Strings`／`Font`／`RawCells`／`Frames` 是否為 nil，nil 指標與 nil map 在查表時才報錯；Rust 以 `Option` 表示 nil，錯誤文字照 Go 套件的字串組成，所以缺件時的錯誤與 Go 逐字相同。
- `native_ch28_post_*` 的三個呈現函式依 `battle.rs` 單位所有權第 2 點就地修改 `units` 與 `native_runtime_records`；Go 先複製再整批換掉，但所有檢查都在改寫前完成，結果相同。C 組沒有 `sort.Slice` 或會受 map 迭代順序影響的程式（`nativeCh28PostFrontiers` 只查表）。
- 測試：C 組 27 個 Go 測試檔共 96 個測試逐條移植，另加 1 個 Rust 補充測試（路由錯誤訊息的 `%#v` 格式）。`native_item_command_damage_test` 的兩個測試會呼叫 B 組的 `native_command_resistance_class`／`apply_native_command_damage`，標 `#[ignore = "等 B 合併"]`；同檔另外兩個只用路由的測試現在就跑。`native_map_unit_window.go` 沒有 Go 測試，由黃金差分涵蓋。
- 黃金差分：`tools/rust_golden_rules_c.sh`（`remake/cmd/fd2-golden-rules-c`、`tests/golden_rules_c.rs`）。真實輸入除 `remake/assets` 外還掛原版 `FDOTHER.DAT`／`FDTXT.DAT` 與分離素材包（`ui/fdother_005_item_panel`、`text/FDTXT_000`、`fonts`、用到的 `portraits`），檔案清單排序後連同大小與 SHA-256 寫進黃金檔；繪製結果比 320×200 目的緩衝的 SHA-256。`KNOWN_DIFFERENCES` 與比對正規化寫在 `tests/golden_rules_c.rs` 檔頭。

## E 組完成紀錄（2026-11-14）

- 9 個檔的 78 個 `todo!()` 全部移植（PORTING.json 標 `ported`），E 組 86 個函式沒有剩下的 `todo!()`。`magic.rs` 的 `with_engine_rand` 沿用 A 組本體。E 組呼叫 D 組 8 次（`next_ai_plan_for_unit` 的四個 native 規劃器、`native_ai_preselected` 的 `native_ai_mode_runtime_context`、`score_native_ai1598_a`、`score_native_ai1567_e`、`native_ai1_f183_target_gate`），照 Go 寫好呼叫；D 的四個規劃器在 Rust 回 `Result<(plan, handled)>`，Go 只在 handled 為真時帶錯誤，所以 `Err` 一律視為 handled。
- 簽名變更見上表（`StatRoll`、`GainExpHost`、`ai_targets`、`Spell` 的 serde）。`growth.rs` 另有 crate 私有的 `LegacyGainExpHost`、`StateGainExpHost` 與成長表 JSON 列型別 `NativeGrowthRowJson`（讀小寫 `ap`…`mp`，另收 Go 欄位名）。
- 測試：E 組 11 個 Go 測試檔共 83 個測試逐條移植，期望值不改。`combat_test` 的 10 個 `NextAIPlan` 測試標 `#[ignore = "等 D 合併"]`，`--include-ignored` 時都停在 D 組 `nextNativeAIMode11Plan` 的 `todo!()`；其餘 73 個現在就跑。`newTestState`／`mkUnit` 放在 `magic/magic_test.rs`（`pub(crate)`），`combat_test`、`growth_test` 引用；`completeNativeAIScoringUnit` 照其他組慣例複製一份。原本等 E 的 B 組 `balance_growth_keeps_original_roll_sequence_and_zero_growth`、`kill_with_exp_cancel_awards_no_experience` 與 A 組兩個 `state_gain_exp_*` 在 `--include-ignored` 下都通過（ignore 標記留給整合時處理）。
- 黃金差分：`tools/rust_golden_rules_e.sh`（`remake/cmd/fd2-golden-rules-e`、`tests/golden_rules_e.rs`）。33 張真實地圖（排序後的清單與 SHA-256 寫進黃金檔）各一份戰場底稿，另加 16 份合成戰場，以固定種子產生 5188 個案例：`AttackWithRNG`（含 1～100 的固定種子迴圈）、`AttackWithNativeExperience`、`AttackNativePhysical`／`AttackNativePhysicalWithExperience`（含反擊、連擊、狀態武器、經驗累計、升級成長擲原版 RNG 與 observer）、反擊資格、`RollNativePhysicalDamage`、`NativeBattleResultCode205B4`、玩家攻擊射程、地形表與逐格地形、`CastArea`／`InCastRange`、`AIAvailableSpells`／`AISpellCandidates`、`Result`、`GainExp`／`State.GainExp`／`AwardExp`／`AwardExpNative`／`NativeGrowthRowFor`、成長表載入（含合成損壞檔）與經驗公式。帶 `math/rand` 的案例另比呼叫後的下一個 `Int63`。黃金檔 `rules-e-v1.json` SHA-256 `30b4c98af80c868634a3065eba30b74c22b695b4ca2048abbf728c310d83f8d8`，Go 端重跑三次相同；竄改一個反擊揮擊的 `RNGState` 後差分精確報出該欄位並失敗。
- 黃金不涵蓋：會走到 D 組的 `NextAIPlan`、`NextAllyAIPlan` 及只經由它們抵達的 `nextNativeScannedAIPlan`、`nativeAIScanEligible`、`nativeAIPreselected`、`nextAIPlanForUnit`、`nextAISpellPlan`、`aiSpellOptions`、`aiSpellTargetScore`、`aiSpellPath`、`aiSpellPriority`、`aiTargets`、`aiApproachPath`、`estDamage`、`nativeAIPlanScoredCommands`、`hostile`（整合後補）；以時間播種的 `Attack`、`Cast`、`AITurn`／`aiActUnit`（只能由單元測試涵蓋）。Go 端沒有呼叫者的 `applyLevelUpGrowth` 照移植，沒有黃金。
- 順序風險：`combat.go` 202、235、621、921 行對應的 Rust 依 `Cell` 升冪迭代 `reachable` 的結果，距離平手時取第一個，列入 `KNOWN_DIFFERENCES`；621 行（`aiSpellPath`）另有 (Y, X) 決勝，與迭代順序無關。這四處只經由 D 組路徑抵達，`combat_test` 裡走到這幾個迴圈的 `NextAIPlan` 測試逐一檢查過，最近格都唯一，期望值不依賴 Go 的迭代順序；D 合併後以實際執行再確認。E 組沒有 `sort.Slice`。
- 小數讀取：serde_json 預設讀小數可能差 1 ulp，與 Go `strconv` 的正確捨入不同；主代理整合時在 `fd2-rules` 開啟 `serde_json` 的 `float_roundtrip`，黃金不再需要「JSON 文字往返」正規化，已移除。

## D 組完成紀錄（2026-11-14）

- 21 個檔的 87 個 `todo!()` 全部移植（PORTING.json 標 `ported`），D 組 90 個函式沒有剩下的 `todo!()`。另加幾個 Rust 端的小工具，放在 D 組自己的檔裡：`native_ai_action_runtime.rs` 的 `debug_ai`（取 `DEBUG_AI`，寫法同 B 組）與 `go_hex`（Go `%x` 對 `[]byte`，只供診斷文字）、`native_ai_mode5.rs` 的格網表頭檢查與 tile word 讀取、`native_ai_spell_score.rs` 的 `native_ai_spell_record`、`native_ai_scored_group_score.rs` 的 `native_ai_reborrow_skip`。簽名變更見上表。
- D 組呼叫 E 組的地方照 Go 寫好：`State::native_ai_plan_scored_commands`（`combat.go`）與 `native_terrain_apdp_pct`（`terrain.go`）。D 組沒有新增 serde：AI 結果型別在 Go 端沒有 JSON 標籤，也沒有 JSON 讀寫。
- `DEBUG_AI` 診斷文字：Go 以 `%+v` 印候選、物品與幾何結構，Rust 印 `Debug` 格式，只供診斷，不比對（同 B 組）。
- Go map 迭代順序：`ScoreNativePhysicalAttackCandidate`／`SelectNativePhysicalAttackCandidate`（`native_ai_physical_score.go` 46、82 行）多欄同時越界時，Rust 依鍵名升冪檢查；`BuildNativeAIPhaseDiagnosticPlan`（`native_ai_phase_diagnostic.go` 127 行）多筆不合格輸入時，Rust 依單位索引升冪。黃金每案最多放一個不合法欄位或輸入。D 組沒有用 `sort.Slice`。
- 測試：D 組 19 個 Go 測試檔共 108 個測試逐條移植；22 個標 `#[ignore = "等 E 合併"]`：`native_ai_runtime_test` 20 個中的 19 個與 `native_ai_mode5_test` 7 個中的 2 個經由 E 組 `State::next_ai_plan`，`native_ai_target_terrain_test` 的 1 個經由 E 組 `native_terrain_apdp_pct`；`-- --include-ignored` 時這 22 個都停在 E 的 `todo!()`。D 組共用的測試輔助在 `src/battle/native_ai_runtime/d_test_support.rs`（Go 散在各測試檔的 `completeNativeAIScoringUnit`、`nativeCompositionBaseFlagsForTest` 等）。
- 與 Go 測試的差異：`ApplyNativeAIIdleRecoveryWithPresentation` 的兩個測試在 Go 是由 callback 直接讀寫外層 `records`；Rust 的 `records` 在呼叫期間被可變借用，callback 碰不到，「callback 改了記錄」這條回滾分支無法從 safe code 觸發，改測該分支依賴的 decision 重算比對。`ApplyNativeAIMode5EventWithAudioCue` 的 callback 同理不讀外層 `state`，改在呼叫後檢查事件狀態。
- A 組 `treasure_test` 的 4 個延後測試中，兩個 `PlayerTreasure*` 與 `Map25NativeEventTreasure*` 在 D 合併後已可通過（`--include-ignored` 實測），`SkyKeyMaterialAssets*` 仍停在 F 組 `event.go:LoadScenario`。這些 ignore 標記屬 A 組檔，D 組沒有改，整合時拿掉。
- 黃金差分：`tools/rust_golden_rules_d.sh`，Go 端 `remake/cmd/fd2-golden-rules-d`、Rust 端 `tests/golden_rules_d.rs`。33 張真實地圖（排序後的清單、`map.json` 與資料表的 SHA-256 寫進黃金檔；32 張可產生原始記錄，map32 的單位缺 provenance）加 40 份合成戰場，以固定種子產生 5570 個案例，涵蓋 D 組不經過 E 的公開函式（含錯誤路徑，錯誤文字逐字比對）。`KNOWN_DIFFERENCES` 與比對正規化寫在 `tests/golden_rules_d.rs` 檔頭。原版的 0x1F183 目標閘門與地形 AP/DP 修正是 crate 私有或 E 組函式，黃金工具改用自己的確定性閘門與 score-input 來源。
- 整合後才能涵蓋（經由 E 組或只有 crate 私有入口）：`next_native_ai14_ef0_plan`、`next_native_ai_mode_fallback_plan`、`next_native_ai_mode11_plan`、`next_native_ai_physical_plan` 及其私有輔助（`native_ai14_ef0_physical_selection`、`native_ai_plan_for_stationary_selection`、`native_ai_plan_for_destination`、`native_ai_plan_toward_raw_destination`、`native_ai_movement_approach_point`、`native_ai_mode_runtime_context`、`native_ai_mode11_blocked_or_recovery`、`native_ai_idle_recovery_only_plan`、`attach_native_fallback_idle_recovery`、`native_ai_mode2_idle_recovery_plan`、`native_ai_physical_score_input`、`native_ai_terrain_adjusted_words`、`native_ai_physical_helper1_debe`、`native_ai_cost_row_for_record`、`native_ai1_f183_target_gate`、`native_ai_raw_record8_index`、`native_ai_active_unit_at_raw_cell`、`native_ai_mode_blocked_search_flags`）。其中純函式的部分已由單元測試與上面的公開函式間接涵蓋。
