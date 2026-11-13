//! 戰棋規則核心，對應凍結 Go 版 `remake/internal/battle`（R2 階段 1 骨架）。
//!
//! 每個 Go 檔 `x.go` 對應子模組 `battle/x.rs`；型別、常數、表格已完整移植，函式只有簽名與
//! Go 註解，本體是 `todo!("battle/<file>.go:<FuncName>")`。移植狀態與檔案相依見 crate 根目錄的
//! `PORTING.json`，分組與順序見 `PORTING.md`。
//!
//! # 單位的所有權
//!
//! Go 版以 `*Unit` 在多個切片之間共用同一個單位。Rust 版採用下列唯一設計，階段 2 不再重新決定：
//!
//! 1. **單位本體只存在兩處**：`State::units`（戰場上的執行期陣列）與 `State::roster`（尚未
//!    登場的 FDFIELD 列）。兩者都是 `Vec<Unit>`，彼此不共用；登場時把值從 `roster` 移到
//!    `units`。Go 版 `snapshot.go` 在存檔前檢查兩個陣列沒有共用指標，Rust 由型別保證。
//! 2. **`UnitIdx`（`usize`）是單位的參照**，意思是「`State::units` 裡的位置」。`units` 在戰鬥中
//!    只會 append、不會刪除或重排（陣亡單位仍留在陣列），所以索引在整場戰鬥中穩定；這也和原版
//!    以執行期記錄槽位（0x50 位元組一筆）定址的方式相同。Go 另有幾種整批替換：開戰前
//!    `Scenario.SetupChecked` 把 `Units` 整個移到 `Roster`；`native_ch28_post_*` 的三個呈現函式以
//!    修改後的複本換掉元素，長度與順序不變，Rust 直接就地修改（Go 外部若還握著舊指標會看到舊值，
//!    Rust 的索引沒有這個差別）。讀快照（`snapshot.go`）則整個重建 `units`，之前的索引一併失效。
//! 3. **函式參數的對應規則**（產生器依 Go 簽名套用，每個函式的 doc 都寫明索引指向哪個陣列）：
//!    - `State` 方法或帶 `*State` 參數的函式：Go 的 `*Unit` 參數 → `UnitIdx`；`[]*Unit` 參數 →
//!      `&[UnitIdx]`，都指向 `state.units`。
//!    - 沒有 `State`、但有 `units []*Unit` 的自由函式：第一個 `[]*Unit` 參數 → `&[Unit]` 或
//!      `&mut [Unit]`，同一函式的 `*Unit` 參數 → 指向它的 `UnitIdx`，其餘 `[]*Unit` → `&[UnitIdx]`。
//!    - 只操作單一單位的函式與 `Unit` 方法：`&Unit`／`&mut Unit`。
//!    - Go 把參數 append 進 `Units`／`Roster` 的函式（例如 `State::add_unit`）：取得本體 `Unit`／`Vec<Unit>`。
//!    - Go 會檢查 nil 的參數包成 `Option`。Go 用指標相等（`u == actor`）判斷同一單位，Rust 改比較索引。
//! 4. **結果結構裡的 `Target *Unit`、`Targets []*Unit`** → `Option<UnitIdx>`／`Vec<UnitIdx>`，指向
//!    產生結果時的 `state.units`。
//! 5. **Go 的淺複製**（`candidate := *s`、`shadow := *unit`）在 Rust 是深複製（`clone()`）。Go 版
//!    淺複製後透過共用指標改到原物件的副作用，移植時要逐處確認；目前讀到的幾處（例如
//!    `AppendGroupWithNativePlacement`）都是「先在副本試算、成功才整批寫回」，深複製語意相同。
//! 6. Go 測試把不在 `State.Units` 裡的單位傳給 `State` 方法時，Rust 測試要先 push 進 `units` 再傳索引，
//!    並確認被測方法不會因此多掃到一個單位。
//!
//! # 其他型別決定
//!
//! - Go `int` → `i64`；Go 會溢位繞回的運算，移植時用 `wrapping_*`（同 `fd2-formats`）。
//! - Go 具名整數／字串型別（`Camp`、`NativeItemBaseStat` 等）→ 單欄 newtype，常數是模組層 `const`；
//!   Go 允許任意整數轉型成這些型別，所以不用 Rust `enum`。
//! - Go `map` → `BTreeMap`，`map[K]struct{}` → `BTreeSet`（鍵型別 derive `Ord`）。Go 的 map 迭代順序
//!   是隨機的；正式程式 16 處 `range` map 裡，下列幾處的結果會隨順序改變，Rust 的有序迭代只是
//!   Go 可能結果之一，黃金差分要避開平手或改比集合：
//!   - `combat.go` 202、235、621、921 行：以嚴格 `<` 在 `reach` 裡挑最近的格子，距離平手時取先迭代到的。
//!   - `native_ai_physical_score.go` 46、82 行與 `native_ai_phase_diagnostic.go` 127 行：多個欄位同時
//!     不合法時，錯誤訊息報出哪一個由迭代順序決定。
//!
//!   其餘各處只做驗證、集合運算，或迭代後再排序（`snapshot.go` 246 行），與順序無關。
//! - Go 會以 `== nil` 區分 nil 與空值的切片／map 欄位包成 `Option`（欄位 doc 有註明）；其他切片欄位的
//!   nil 與空值在 Rust 都是空 `Vec`，只影響 JSON 的 `null`／`[]`，由快照分組處理。
//! - 自由函式經由結構欄位裡的 `UnitIdx`（Go 的 `*Unit` 欄位）讀寫單位時，Rust 簽名多一個開頭參數
//!   `units: &[Unit]`／`&mut [Unit]`；兩個以上 `*Unit` 參數可能指向同一單位且會被改寫時（例如對自己
//!   使用道具）也同樣改傳 `units` 與索引。這些函式的 doc 都有「所有權：`units` 是 Rust 新增的參數」。
//! - Go `error` → [`BattleError`]（`Display` 必須與 Go 錯誤字串逐字相同）；`(T, error)` → `Result<T>`。
//! - Go 指標欄位 → `Option<T>`（擁有值）；Go 允許兩個結構共用同一個子結構指標，Rust 改為各自擁有。
//! - Go 的 func 欄位 → `Option<Callback<dyn Fn(..) + Send + Sync>>`（`Arc` 包裝，`State` 可 `Clone`，
//!   可放進 Bevy resource）；func 參數 → `&mut dyn FnMut(..)`。
//! - `*rand.Rand` → `&mut dyn GoRand`。R2 的完成條件要求同種子與 Go 結果相同，所以實作必須重現
//!   Go `math/rand` 的序列（見 `rng.rs`）。
//! - Go 的 JSON 標籤保留在欄位 doc（`Go 標籤：…`）。骨架不 derive serde：`fd2-formats` 的
//!   `NativeMapSpriteCycleState` 等型別沒有 serde 實作，而且 Go `[]byte` 以 base64 編碼、解碼不分
//!   大小寫，需要由負責 `snapshot.rs` 與各 `Load*` 的分組一次決定（見 `PORTING.md`）。

mod error;
mod format_deps;
mod rng;
mod support;

pub use error::*;
pub use rng::*;
pub use support::*;

pub(crate) use fd2_formats::{dato, fdicon, fdother, fdtxt};
pub(crate) use std::collections::{BTreeMap, BTreeSet};
pub(crate) use std::fmt;
#[cfg(test)]
pub(crate) use std::sync::Arc;
pub(crate) use std::sync::{LazyLock, RwLock};

// ---- 每個 Go 檔一個子模組（自動產生；新增 Go 對應檔時依字母順序插入）----
mod balance;
mod combat;
mod command_learn;
mod event;
mod growth;
mod magic;
mod model;
mod r#move;
mod native_ai_14ef0_dispatch;
mod native_ai_1598a_score;
mod native_ai_action_runtime;
mod native_ai_choice;
mod native_ai_destination;
mod native_ai_idle_recovery;
mod native_ai_item_preselection;
mod native_ai_item_score;
mod native_ai_mode11;
mod native_ai_mode11_runtime;
mod native_ai_mode5;
mod native_ai_mode_record;
mod native_ai_mode_write;
mod native_ai_path;
mod native_ai_phase_diagnostic;
mod native_ai_physical_candidates;
mod native_ai_physical_item_source;
mod native_ai_physical_score;
mod native_ai_record;
mod native_ai_runtime;
mod native_ai_scored_candidates;
mod native_ai_scored_group_score;
mod native_ai_spell_score;
mod native_battle_panel;
mod native_battle_result;
mod native_ch28_post_prelude;
mod native_ch28_post_present;
mod native_class_compat;
mod native_command;
mod native_command0;
mod native_command13;
mod native_command17;
mod native_command20;
mod native_command20_ai_plan;
mod native_command24;
mod native_command25;
mod native_command26;
mod native_command32;
mod native_command33;
mod native_command34;
mod native_command35;
mod native_command_availability;
mod native_command_compound;
mod native_command_damage;
mod native_command_grid;
mod native_command_modifier;
mod native_command_mp;
mod native_command_overlay_render;
mod native_command_palette_flash;
mod native_command_record;
mod native_command_scene;
mod native_command_target;
mod native_coordinate_write;
mod native_death_program;
mod native_end_turn_recovery;
mod native_equipment;
mod native_equipped_item;
mod native_field_event;
mod native_future_constructor;
mod native_inventory_assign;
mod native_inventory_available;
mod native_inventory_flags;
mod native_inventory_records;
mod native_inventory_remove;
mod native_inventory_search;
mod native_inventory_slots;
mod native_inventory_transfer;
mod native_item_ap_dp_step;
mod native_item_base_stat_unit;
mod native_item_capacity_step;
mod native_item_capacity_unit;
mod native_item_command_damage;
mod native_item_dispatch;
mod native_item_hit_ev_step;
mod native_item_hp_restore;
mod native_item_marker_application;
mod native_item_marker_clear_restore;
mod native_item_mp_restore;
mod native_item_panel_data_render;
mod native_item_panel_record;
mod native_item_panel_render;
mod native_item_relocation;
mod native_item_row;
mod native_item_selector;
mod native_item_table;
mod native_item_target;
mod native_map_presentation;
mod native_map_unit_window;
mod native_map_view;
mod native_marker_rewrite;
mod native_movement_cost;
mod native_persistent_copy;
mod native_persistent_lookup;
mod native_physical_attack;
mod native_physical_damage;
mod native_physical_experience;
mod native_physical_scene;
mod native_player_attack_field;
mod native_player_movement;
mod native_presentation_queue;
mod native_raw_application;
mod native_raw_damage;
mod native_raw_flag_restore;
mod native_raw_pair_step;
mod native_raw_restore;
mod native_raw_word_step;
mod native_raw_word_subtract;
mod native_record_byte5;
mod native_record_byte5_bit0;
mod native_record_flags;
mod native_result;
mod native_revive_list_render;
mod native_system_group_march;
mod native_transfer_item_render;
mod native_transient;
mod native_turn_pair_mutation;
mod native_turn_progression;
mod native_unit_byte_clear;
mod native_word_delta_remove;
mod snapshot;
mod terrain;

pub use balance::*;
pub use combat::*;
pub use command_learn::*;
pub use event::*;
pub use growth::*;
pub use magic::*;
pub use model::*;
pub use native_ai_14ef0_dispatch::*;
pub use native_ai_1598a_score::*;
pub use native_ai_action_runtime::*;
pub use native_ai_choice::*;
pub use native_ai_destination::*;
pub use native_ai_idle_recovery::*;
pub use native_ai_item_preselection::*;
pub use native_ai_item_score::*;
pub use native_ai_mode_record::*;
pub use native_ai_mode_write::*;
pub use native_ai_mode5::*;
pub use native_ai_mode11::*;
pub use native_ai_path::*;
pub use native_ai_phase_diagnostic::*;
pub use native_ai_physical_candidates::*;
pub use native_ai_physical_item_source::*;
pub use native_ai_physical_score::*;
pub use native_ai_record::*;
pub(crate) use native_ai_runtime::*;
pub use native_ai_scored_candidates::*;
pub use native_ai_scored_group_score::*;
pub use native_ai_spell_score::*;
pub use native_battle_panel::*;
pub use native_battle_result::*;
pub use native_ch28_post_prelude::*;
pub use native_ch28_post_present::*;
pub use native_class_compat::*;
pub use native_command_availability::*;
pub use native_command_compound::*;
pub use native_command_damage::*;
pub use native_command_grid::*;
pub use native_command_modifier::*;
pub use native_command_mp::*;
pub use native_command_overlay_render::*;
pub use native_command_palette_flash::*;
pub use native_command_record::*;
pub use native_command_scene::*;
pub use native_command_target::*;
pub use native_command0::*;
pub use native_command13::*;
pub use native_command17::*;
pub use native_command20::*;
pub use native_command20_ai_plan::*;
pub use native_command24::*;
pub use native_command25::*;
pub use native_command26::*;
pub use native_command32::*;
pub use native_command33::*;
pub use native_command34::*;
pub use native_command35::*;
pub use native_coordinate_write::*;
pub use native_death_program::*;
pub use native_end_turn_recovery::*;
pub use native_equipment::*;
pub use native_equipped_item::*;
pub use native_field_event::*;
pub use native_future_constructor::*;
pub use native_inventory_assign::*;
pub use native_inventory_available::*;
pub use native_inventory_flags::*;
pub use native_inventory_records::*;
pub use native_inventory_remove::*;
pub use native_inventory_search::*;
pub use native_inventory_slots::*;
pub use native_inventory_transfer::*;
pub use native_item_ap_dp_step::*;
pub use native_item_base_stat_unit::*;
pub use native_item_capacity_step::*;
pub use native_item_capacity_unit::*;
pub use native_item_command_damage::*;
pub use native_item_dispatch::*;
pub use native_item_hit_ev_step::*;
pub use native_item_hp_restore::*;
pub use native_item_marker_application::*;
pub use native_item_marker_clear_restore::*;
pub use native_item_mp_restore::*;
pub use native_item_panel_data_render::*;
pub use native_item_panel_record::*;
pub use native_item_panel_render::*;
pub use native_item_relocation::*;
pub use native_item_row::*;
pub use native_item_selector::*;
pub use native_item_table::*;
pub use native_item_target::*;
pub use native_map_presentation::*;
pub use native_map_unit_window::*;
pub use native_map_view::*;
pub use native_marker_rewrite::*;
pub use native_movement_cost::*;
pub use native_persistent_copy::*;
pub use native_persistent_lookup::*;
pub use native_physical_attack::*;
pub use native_physical_damage::*;
pub(crate) use native_physical_experience::*;
pub use native_physical_scene::*;
pub use native_player_movement::*;
pub use native_presentation_queue::*;
pub use native_raw_application::*;
pub use native_raw_damage::*;
pub use native_raw_flag_restore::*;
pub use native_raw_pair_step::*;
pub use native_raw_restore::*;
pub use native_raw_word_step::*;
pub use native_raw_word_subtract::*;
pub use native_record_byte5::*;
pub use native_record_byte5_bit0::*;
pub use native_record_flags::*;
pub use native_result::*;
pub use native_revive_list_render::*;
pub use native_system_group_march::*;
pub use native_transfer_item_render::*;
pub use native_transient::*;
pub use native_turn_pair_mutation::*;
pub use native_turn_progression::*;
pub use native_unit_byte_clear::*;
pub use native_word_delta_remove::*;
pub use snapshot::*;
pub use terrain::*;
