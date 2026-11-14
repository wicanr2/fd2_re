//! 戰役流程、城鎮、商店、教會與劇本處理器，對應凍結 Go 版 `remake/internal/campaign`（R3 骨架）。
//!
//! 每個 Go 檔 `x.go` 對應子模組 `campaign/x.rs`；型別、常數、表格已完整移植，函式只有簽名與
//! Go 註解，本體是 `todo!("campaign/<file>.go:<FuncName>")`。移植狀態與檔案相依見 crate 根目錄的
//! `PORTING.json`，分組與順序見 `PORTING.md`。
//!
//! # 單位與戰場狀態
//!
//! 單位與戰場狀態一律用 `fd2_rules::battle` 的型別，所有權照 `fd2_rules::battle` 模組文件的規則：
//!
//! - 帶 `*battle.State` 參數的函式：Go 的 `*battle.Unit` 參數 → `UnitIdx`，指向 `state.units`；
//!   `[]*battle.Unit` → `&[UnitIdx]`。
//! - 沒有 `State`、只操作單一單位的函式：`&Unit`／`&mut Unit`（Go 會檢查 nil 的包成 `Option`）。
//! - 沒有 `State`、但有 `units []*battle.Unit` 的函式：第一個 `[]*battle.Unit` → `&[Unit]`／`&mut [Unit]`。
//! - Go 以值保存的 `battle.Unit`（例如持續隊伍 `map[int]battle.Unit`、JOIN 建構結果）在 Rust 也是
//!   擁有的 `Unit` 值，和 `State::units` 無關；放進戰場時由呼叫端 push 進 `State::units`。
//! - 這三個套件的正式程式沒有任何結構欄位保存 `*battle.Unit` 或 `[]*battle.Unit`，所以沒有
//!   `UnitIdx` 欄位，也沒有 R2 那種「多一個 `units` 參數」的函式。
//!
//! # 其他型別決定
//!
//! - Go `int` → `i64`；Go 會溢位繞回的運算用 `wrapping_*`。具名整數／字串型別 → 單欄 newtype，常數是
//!   模組層 `const`（同 `fd2_rules::battle`）。
//! - Go `map` → `BTreeMap`，`map[K]struct{}` → `BTreeSet`。Go 的 map 迭代順序隨機、Rust 有序；
//!   會受順序影響的地方列在 `PORTING.md`「Go 寫法難對應處」。
//! - Go `error` → [`CampaignError`]（`error.rs`）；`(T, error)` → `Result<T>`。
//! - Go `*rand.Rand` → `&mut dyn fd2_rules::battle::GoRand`（序列與 Go `math/rand` 相同）。
//! - Go 的 func 欄位 → `Option<Callback<dyn Fn(..) + Send + Sync>>`（`fd2_rules::battle::Callback`）；
//!   func 參數 → `&mut dyn FnMut(..)`。
//! - Go `any`（只出現在 JSON 欄位）→ `serde_json::Value`，nil 對應 `Value::Null`。
//! - Go `image.Point`／`image.Rectangle`／`image.RGBA` → `go_image.rs` 的同形值型別（只有欄位；用到的
//!   方法由階段 2 補，不移植 Go `image` 套件的其餘部分）。`ending` 也用這一份。
//! - Go 的 JSON 標籤保留在欄位 doc（`Go 標籤：…`）。骨架不 derive serde：讀檔函式由負責的組在
//!   階段 2 一次決定 serde 屬性（`[]byte` 用 base64、nil 切片的 `null` 等，比照
//!   `fd2_rules::battle::go_bytes`／`null_as_default`）。

// 函式 doc 逐字保留 Go 註解（含條列與縮排），不改寫成 Markdown 慣例。
#![allow(clippy::doc_lazy_continuation, clippy::doc_overindented_list_items)]
// 簽名逐一對應 Go 版：參數個數與 callback 型別照 Go 保留（同 fd2-rules 的 crate 層設定）。
#![allow(clippy::too_many_arguments, clippy::type_complexity)]

mod error;
mod ext_deps;
pub(crate) mod go_image;

pub use error::*;
pub use go_image::*;

pub(crate) use error::campaign_err;
pub(crate) use fd2_formats::{dato, fdicon, fdother, fdtxt, save};
pub(crate) use fd2_rules::battle::{self, Callback, GoRand, Unit};
pub(crate) use std::collections::{BTreeMap, BTreeSet};
pub(crate) use std::sync::LazyLock;

// ---- 每個 Go 檔一個子模組（自動產生；新增 Go 對應檔時依字母順序插入）----
mod ai_commands;
#[allow(clippy::module_inception)] // Go 檔 campaign.go；保持一個 Go 檔一個子模組
mod campaign;
mod church;
mod class_change_table;
mod equipment;
mod handler_binding;
mod handler_compile;
mod handler_script;
mod menu_state;
mod native_chapter_slot_restore;
mod native_chapter_slot_save;
mod native_church_menu;
mod native_church_revive_success;
mod native_church_scene;
mod native_class_confirm;
mod native_class_list;
mod native_continue_field_boundary;
mod native_continue_handoff;
mod native_continue_map_timing;
mod native_continue_pending_groups;
mod native_continue_runtime_units;
mod native_current_snapshot;
mod native_gold_debit;
mod native_join_base;
mod native_join_constructor;
mod native_load_slots;
mod native_map_view_config;
mod native_persistent_party;
mod native_roster_list;
mod native_shop_equipment_recipient;
mod native_shop_item_list;
mod native_shop_purchase_message;
mod native_shop_purchase_success;
mod native_shop_recipient;
mod native_shop_scene;
mod native_shop_sell_message;
mod native_shop_service_menu;
mod native_shop_transfer_message;
mod native_story_dialogue;
mod native_story_dialogue_coverage;
mod native_story_hd_text;
mod native_system_info;
mod native_town_departure;
mod native_town_scene;
mod native_transient_expiry;
mod separated_shop_assets;
mod shop;
mod story_index_map;

pub use ai_commands::*;
pub use campaign::*;
pub use church::*;
pub use class_change_table::*;
pub use equipment::*;
pub use handler_binding::*;
pub use handler_compile::*;
pub use handler_script::*;
pub use menu_state::*;
pub use native_chapter_slot_restore::*;
pub use native_chapter_slot_save::*;
pub use native_church_menu::*;
pub use native_church_revive_success::*;
pub use native_church_scene::*;
pub use native_class_confirm::*;
pub use native_class_list::*;
pub use native_continue_field_boundary::*;
pub use native_continue_handoff::*;
pub use native_continue_map_timing::*;
pub use native_continue_pending_groups::*;
pub use native_continue_runtime_units::*;
pub use native_current_snapshot::*;
pub use native_gold_debit::*;
pub use native_join_base::*;
pub use native_join_constructor::*;
pub use native_load_slots::*;
pub use native_persistent_party::*;
pub use native_roster_list::*;
pub use native_shop_equipment_recipient::*;
pub use native_shop_item_list::*;
pub use native_shop_purchase_message::*;
pub use native_shop_purchase_success::*;
pub use native_shop_recipient::*;
pub use native_shop_scene::*;
pub use native_shop_sell_message::*;
pub use native_shop_service_menu::*;
pub use native_shop_transfer_message::*;
pub use native_story_dialogue::*;
pub use native_story_dialogue_coverage::*;
pub use native_story_hd_text::*;
pub use native_system_info::*;
pub use native_town_departure::*;
pub use native_town_scene::*;
pub use native_transient_expiry::*;
pub use separated_shop_assets::*;
pub use shop::*;
pub use story_index_map::*;
