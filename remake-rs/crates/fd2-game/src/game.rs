//! 遊戲狀態機，對應凍結 Go 版 `remake/cmd/fd2`（`package main`）的流程部分（R4 骨架）。
//!
//! 每個 Go 檔 `x.go` 對應子模組 `game/x.rs`，各自寫 `impl Game`；只有含骨架項目的 Go 檔才有子模組。
//! 型別、常數與套件層變數已完整產生；函式只有階段 0 清單（第一章受控流程執行到的
//! 565 個），本體是 `todo!("<file>.go:<FuncName>")`。同檔其餘函式在 `PORTING.json` 標 `r8`。
//! 骨架由 `tools/rust_r4_skeleton.sh` 產生，對應規則見
//! `docs/goal/120-goal-rust-r4-game-state-split-20261114.md`「階段 1 骨架的對應規則」，實際落地與偏離見
//! crate 根目錄的 `PORTING.md`。
//!
//! # 所有權
//!
//! - `Game` 保留單一型別，Go 的 538 個欄位全部保留（重設類函式幾乎碰到每個欄位）；只替換帶
//!   繪圖資源的欄位型別。
//! - **單位**：`*battle.Unit` 欄位 → `Option<UnitIdx>`，指向 `Game` 持有的 `battle::State::units`。
//!   `Game` 的方法與帶 `*battle.State` 參數的函式，`*battle.Unit` 參數也是 `UnitIdx`；其他函式收
//!   `&battle::Unit`／`&mut battle::Unit`（同 R2、R3）。容器裡的 `*battle.Unit` 是 `UnitIdx`。
//! - **畫面**：`*ebiten.Image`（含容器元素）→ `Option<Arc<Picture>>`；要畫上去的參數收 `&mut Picture`，
//!   只讀的收 `&Arc<Picture>`（可以直接存進狀態）。演出工作保存索引色畫格（`IndexedFrame` 或 `Vec<u8>`）
//!   與調色盤；GPU 貼圖由 `fd2-app` 建立。
//! - **延續閉包**：Go 的 func 欄位捕捉 `g`，Rust 的第一個參數是 `&mut Game`。`then`／`after`／`rollback`
//!   這類工作結束時只呼叫一次的延續是 `Option<Box<dyn FnOnce(&mut Game, ..) + Send>>`，其餘（觀察者、
//!   交易、時鐘注入）是 `FnMut`。呼叫前先 `take()` 出來，避免同時借用 `Game`。函式參數的 func 是
//!   `&mut dyn FnMut(..)`。
//! - **其他指標**：結構欄位的指標是 `Option<T>`（遞迴型別用 `Option<Box<T>>`）；參數的指標是 `&T`／`&mut T`，
//!   Go 會檢查 nil 的包成 `Option`。指標回傳值若指向接收者的狀態，回傳借用；新建的回傳擁有的值。
//!
//! # 其他型別決定
//!
//! - Go `int` → `i64`；會溢位繞回的運算用 `wrapping_*`。具名整數型別 → 單欄 newtype，常數是模組層 `const`。
//! - Go `map` → `BTreeMap`，`map[K]struct{}` → `BTreeSet`；Go 的 map 迭代順序隨機、Rust 有序。
//! - Go `error` → [`GameError`]（`support.rs`）；`(T, error)` → `Result<T>`。錯誤欄位是 `Option<GameError>`。
//! - Go `*rand.Rand` 參數 → `&mut dyn GoRand`，欄位 → `Option<battle::GoMathRand>`。
//! - Go `time.Time`／`time.Duration` → [`GameInstant`]／[`GoDuration`]；收 `now time.Time` 的函式照 Go
//!   保留參數，來源改成虛擬時鐘。
//! - Go `image.Point`／`image.Rectangle`／`image.RGBA` → `fd2_campaign::campaign` 的 `Point`／`Rectangle`／
//!   `Rgba`（R3 已定義）。
//! - 套件層變數：只讀的表格 → `LazyLock<T>`；Go 執行期會改寫的 → `LazyLock<RwLock<T>>`；`errors.New`
//!   哨兵 → `static GameError`。
//! - Go 的 JSON 標籤保留在欄位 doc（`Go 標籤：…`）。骨架不 derive serde，由擁有該檔的組在階段 2 加
//!   （`[]byte` 用 base64、nil 切片的 `null` 等，比照 `fd2_rules::battle::go_bytes`／`null_as_default`）。
//! - HD（`modern_*`、`player_poster*`、`player_toolbar*`）型別保留型別名，欄位只留第一章忠實模式執行到的
//!   函式用到的，其餘 R6。

mod support;

pub use support::*;

pub(crate) use crate::game_err;
pub(crate) use fd2_campaign::campaign::{Point, Rectangle, Rgba};
pub(crate) use fd2_campaign::{
    battlepresent, campaign, editorcanonical, ending, indexedmap, localization, musiccatalog,
};
pub(crate) use fd2_formats::save as fdsave;
pub(crate) use fd2_formats::{afm, archive, dato, fdicon, fdother, fdtxt, figani};
pub(crate) use fd2_rules::battle::{self, GoRand, UnitIdx};
pub(crate) use std::collections::{BTreeMap, BTreeSet};
pub(crate) use std::sync::{Arc, LazyLock, RwLock};

// ---- 每個 Go 檔一個子模組（產生器輸出，依檔名排序）----
mod action_overlay_runtime;
mod assets;
mod assets_glob_native;
mod audio;
mod balance_save;
mod battle_protection;
mod battle_scene_layers;
mod ending_preview;
mod font;
mod input_audit_trace;
mod locale;
mod main;
mod modern_gold_action;
mod modern_hd_battle_assets;
mod modern_hd_battle_camera;
mod modern_hd_battle_gpu;
mod modern_hd_battle_panel;
mod modern_hd_battle_panel_chrome;
mod modern_hd_battle_runtime;
mod modern_hd_battle_scroll;
mod modern_hd_battle_surface;
mod modern_hd_ch23;
mod modern_hd_church_confirmation_transition;
mod modern_hd_command0;
mod modern_hd_command1;
mod modern_hd_command5;
mod modern_hd_commands10_12_map;
mod modern_hd_compound_map;
mod modern_hd_compound_map_source;
mod modern_hd_confirmation;
mod modern_hd_death_stage;
mod modern_hd_dialogue_chrome;
mod modern_hd_facility_dialogue;
mod modern_hd_facility_item_rows;
mod modern_hd_facility_list_transition;
mod modern_hd_facility_money;
mod modern_hd_facility_roster;
mod modern_hd_facility_scene;
mod modern_hd_facility_scene_pair;
mod modern_hd_facility_success;
mod modern_hd_facility_text;
mod modern_hd_field_event61;
mod modern_hd_full_reward;
mod modern_hd_gold_roll;
mod modern_hd_heal_map;
mod modern_hd_indexed_transition;
mod modern_hd_item_text;
mod modern_hd_map_foreground_plan;
mod modern_hd_map_gpu;
mod modern_hd_map_hud_panel;
mod modern_hd_map_overlay_assets;
mod modern_hd_map_runtime;
mod modern_hd_map_unit_plan;
mod modern_hd_movement_panel;
mod modern_hd_ordinary;
mod modern_hd_panel_transition;
mod modern_hd_physical_sfx;
mod modern_hd_player9_map;
mod modern_hd_preparation_background;
mod modern_hd_preparation_counts;
mod modern_hd_preparation_cursor;
mod modern_hd_preparation_dialogue;
mod modern_hd_preparation_roster;
mod modern_hd_preparation_status;
mod modern_hd_preparation_transition;
mod modern_hd_presentation;
mod modern_hd_row_shake;
mod modern_hd_service_menu_transition;
mod modern_hd_shop_comparison;
mod modern_hd_shop_digits;
mod modern_hd_sky_key;
mod modern_hd_split_strike;
mod modern_hd_status_text;
mod modern_hd_system_confirmation;
mod modern_hd_system_info;
mod modern_hd_terrain_plan;
mod modern_hd_text;
mod modern_hd_title_interface;
mod modern_hd_transient_expiry;
mod modern_hd_unit_present;
mod modern_story_portrait;
mod native_action_death_present;
mod native_ai_idle_recovery;
mod native_ai_item_presentation;
mod native_ai_modifier_presentation;
mod native_attack_assets;
mod native_battle_name;
mod native_battle_panel_locale;
mod native_battle_panel_values;
mod native_battle_staging_present;
mod native_bios_clock;
mod native_ch20_sky_key;
mod native_ch22_post;
mod native_ch22_reload;
mod native_ch23_post;
mod native_ch28_post_present;
mod native_chapter_result;
mod native_chapter_slot_save;
mod native_church_ui;
mod native_class_ui;
mod native_class_ui_lifecycle;
mod native_command0_presentation;
mod native_command10_12_presentation;
mod native_command1_presentation;
mod native_command24_presentation;
mod native_command28_presentation;
mod native_command29_presentation;
mod native_command2_presentation;
mod native_command32_presentation;
mod native_command33_presentation;
mod native_command34_presentation;
mod native_command35_presentation;
mod native_command3_presentation;
mod native_command5_presentation;
mod native_command6_presentation;
mod native_command7_presentation;
mod native_command8_presentation;
mod native_command9_ai_presentation;
mod native_command9_player_presentation;
mod native_command_heal_presentation;
mod native_command_locale;
mod native_command_modifier_presentation;
mod native_confirmation_mouth;
mod native_continue_handoff;
mod native_current_save;
mod native_death_program_runtime;
mod native_death_reward_message;
mod native_death_reward_ui;
mod native_defeat;
mod native_end_turn_recovery;
mod native_event_dialogue;
mod native_field_event61;
mod native_field_event75;
mod native_field_event_pending;
mod native_hotel_ui;
mod native_indexed_transition;
mod native_item_panel_ui;
mod native_level_up_dialogue;
mod native_load_slots_ui;
mod native_map_assets;
mod native_map_frame_input;
mod native_map_unit_window;
mod native_palette_pulse;
mod native_palette_ramp;
mod native_physical_body;
mod native_physical_scene;
mod native_player_input;
mod native_player_movement;
mod native_preparation_input;
mod native_preparation_ui;
mod native_shop_locale;
mod native_shop_transaction_ui;
mod native_shop_ui;
mod native_spawn_intro;
mod native_staging_present;
mod native_story_dialogue;
mod native_story_dialogue_locale;
mod native_story_input;
mod native_story_map;
mod native_system_info_locale;
mod native_system_info_ui;
mod native_town_departure;
mod native_town_secret_input;
mod native_town_ui;
mod native_transient_phase;
mod native_transient_presentation;
mod native_treasure_input;
mod native_turn_event76;
mod native_turn_staging;
mod native_unit_present;
mod packaged_startup;
mod player_balance;
mod player_battle_speed;
mod player_game_clock;
mod player_hd_quality;
mod player_interface;
mod player_poster_assets;
mod player_poster_controls;
mod player_theme_loader;
mod run_log;
mod save;
mod separated_asset_path_native;
mod separated_ui_assets;
mod settings;
mod title;
mod title_menu_state;

pub(crate) use action_overlay_runtime::*;
pub(crate) use assets::*;
pub(crate) use assets_glob_native::*;
pub(crate) use audio::*;
pub(crate) use balance_save::*;
pub(crate) use battle_protection::*;
pub(crate) use battle_scene_layers::*;
pub(crate) use ending_preview::*;
pub(crate) use font::*;
pub(crate) use input_audit_trace::*;
pub(crate) use locale::*;
pub(crate) use main::*;
pub(crate) use modern_gold_action::*;
pub(crate) use modern_hd_battle_assets::*;
pub(crate) use modern_hd_battle_camera::*;
pub(crate) use modern_hd_battle_gpu::*;
pub(crate) use modern_hd_battle_panel::*;
pub(crate) use modern_hd_battle_panel_chrome::*;
pub(crate) use modern_hd_battle_runtime::*;
pub(crate) use modern_hd_battle_scroll::*;
pub(crate) use modern_hd_battle_surface::*;
pub(crate) use modern_hd_ch23::*;
pub(crate) use modern_hd_church_confirmation_transition::*;
pub(crate) use modern_hd_command0::*;
pub(crate) use modern_hd_command1::*;
pub(crate) use modern_hd_command5::*;
pub(crate) use modern_hd_commands10_12_map::*;
pub(crate) use modern_hd_compound_map::*;
pub(crate) use modern_hd_compound_map_source::*;
pub(crate) use modern_hd_confirmation::*;
pub(crate) use modern_hd_death_stage::*;
pub(crate) use modern_hd_dialogue_chrome::*;
pub(crate) use modern_hd_facility_dialogue::*;
pub(crate) use modern_hd_facility_item_rows::*;
pub(crate) use modern_hd_facility_list_transition::*;
pub(crate) use modern_hd_facility_money::*;
pub(crate) use modern_hd_facility_roster::*;
pub(crate) use modern_hd_facility_scene::*;
pub(crate) use modern_hd_facility_scene_pair::*;
pub(crate) use modern_hd_facility_success::*;
pub(crate) use modern_hd_facility_text::*;
pub(crate) use modern_hd_field_event61::*;
pub(crate) use modern_hd_full_reward::*;
pub(crate) use modern_hd_gold_roll::*;
pub(crate) use modern_hd_heal_map::*;
pub(crate) use modern_hd_indexed_transition::*;
pub(crate) use modern_hd_item_text::*;
pub(crate) use modern_hd_map_foreground_plan::*;
pub(crate) use modern_hd_map_gpu::*;
pub(crate) use modern_hd_map_hud_panel::*;
pub(crate) use modern_hd_map_overlay_assets::*;
pub(crate) use modern_hd_map_runtime::*;
pub(crate) use modern_hd_map_unit_plan::*;
pub(crate) use modern_hd_movement_panel::*;
pub(crate) use modern_hd_ordinary::*;
pub(crate) use modern_hd_panel_transition::*;
pub(crate) use modern_hd_physical_sfx::*;
pub(crate) use modern_hd_player9_map::*;
pub(crate) use modern_hd_preparation_background::*;
pub(crate) use modern_hd_preparation_counts::*;
pub(crate) use modern_hd_preparation_cursor::*;
pub(crate) use modern_hd_preparation_dialogue::*;
pub(crate) use modern_hd_preparation_roster::*;
pub(crate) use modern_hd_preparation_status::*;
pub(crate) use modern_hd_preparation_transition::*;
pub(crate) use modern_hd_presentation::*;
pub(crate) use modern_hd_row_shake::*;
pub(crate) use modern_hd_service_menu_transition::*;
pub(crate) use modern_hd_shop_comparison::*;
pub(crate) use modern_hd_shop_digits::*;
pub(crate) use modern_hd_sky_key::*;
pub(crate) use modern_hd_split_strike::*;
pub(crate) use modern_hd_status_text::*;
pub(crate) use modern_hd_system_confirmation::*;
pub(crate) use modern_hd_system_info::*;
pub(crate) use modern_hd_terrain_plan::*;
pub(crate) use modern_hd_text::*;
pub(crate) use modern_hd_title_interface::*;
pub(crate) use modern_hd_transient_expiry::*;
pub(crate) use modern_hd_unit_present::*;
pub(crate) use modern_story_portrait::*;
pub(crate) use native_action_death_present::*;
pub(crate) use native_ai_idle_recovery::*;
pub(crate) use native_ai_item_presentation::*;
pub(crate) use native_ai_modifier_presentation::*;
pub(crate) use native_attack_assets::*;
pub(crate) use native_battle_name::*;
pub(crate) use native_battle_panel_locale::*;
pub(crate) use native_battle_panel_values::*;
pub(crate) use native_battle_staging_present::*;
pub(crate) use native_bios_clock::*;
pub(crate) use native_ch20_sky_key::*;
pub(crate) use native_ch22_post::*;
pub(crate) use native_ch22_reload::*;
pub(crate) use native_ch23_post::*;
pub(crate) use native_ch28_post_present::*;
pub(crate) use native_chapter_result::*;
pub(crate) use native_chapter_slot_save::*;
pub(crate) use native_church_ui::*;
pub(crate) use native_class_ui::*;
pub(crate) use native_class_ui_lifecycle::*;
pub(crate) use native_command_heal_presentation::*;
pub(crate) use native_command_locale::*;
pub(crate) use native_command_modifier_presentation::*;
pub(crate) use native_command0_presentation::*;
pub(crate) use native_command1_presentation::*;
pub(crate) use native_command2_presentation::*;
pub(crate) use native_command3_presentation::*;
pub(crate) use native_command5_presentation::*;
pub(crate) use native_command6_presentation::*;
pub(crate) use native_command7_presentation::*;
pub(crate) use native_command8_presentation::*;
pub(crate) use native_command9_ai_presentation::*;
pub(crate) use native_command9_player_presentation::*;
pub(crate) use native_command10_12_presentation::*;
pub(crate) use native_command24_presentation::*;
pub(crate) use native_command28_presentation::*;
pub(crate) use native_command29_presentation::*;
pub(crate) use native_command32_presentation::*;
pub(crate) use native_command33_presentation::*;
pub(crate) use native_command34_presentation::*;
pub(crate) use native_command35_presentation::*;
pub(crate) use native_confirmation_mouth::*;
pub(crate) use native_continue_handoff::*;
pub(crate) use native_current_save::*;
pub(crate) use native_death_program_runtime::*;
pub(crate) use native_death_reward_message::*;
pub(crate) use native_death_reward_ui::*;
pub(crate) use native_defeat::*;
pub(crate) use native_end_turn_recovery::*;
pub(crate) use native_event_dialogue::*;
pub(crate) use native_field_event_pending::*;
pub(crate) use native_field_event61::*;
pub(crate) use native_field_event75::*;
pub(crate) use native_hotel_ui::*;
pub(crate) use native_indexed_transition::*;
pub(crate) use native_item_panel_ui::*;
pub(crate) use native_level_up_dialogue::*;
pub(crate) use native_load_slots_ui::*;
pub(crate) use native_map_assets::*;
pub(crate) use native_map_frame_input::*;
pub(crate) use native_map_unit_window::*;
pub(crate) use native_palette_pulse::*;
pub(crate) use native_palette_ramp::*;
pub(crate) use native_physical_body::*;
pub(crate) use native_physical_scene::*;
pub(crate) use native_player_input::*;
pub(crate) use native_player_movement::*;
pub(crate) use native_preparation_input::*;
pub(crate) use native_preparation_ui::*;
pub(crate) use native_shop_locale::*;
pub(crate) use native_shop_transaction_ui::*;
pub(crate) use native_shop_ui::*;
pub(crate) use native_spawn_intro::*;
pub(crate) use native_staging_present::*;
pub(crate) use native_story_dialogue::*;
pub(crate) use native_story_dialogue_locale::*;
pub(crate) use native_story_input::*;
pub(crate) use native_story_map::*;
pub(crate) use native_system_info_locale::*;
pub(crate) use native_system_info_ui::*;
pub(crate) use native_town_departure::*;
pub(crate) use native_town_secret_input::*;
pub(crate) use native_town_ui::*;
pub(crate) use native_transient_phase::*;
pub(crate) use native_transient_presentation::*;
pub(crate) use native_treasure_input::*;
pub(crate) use native_turn_event76::*;
pub(crate) use native_turn_staging::*;
pub(crate) use native_unit_present::*;
pub(crate) use packaged_startup::*;
pub(crate) use player_balance::*;
pub(crate) use player_battle_speed::*;
pub(crate) use player_game_clock::*;
pub(crate) use player_hd_quality::*;
pub(crate) use player_interface::*;
pub(crate) use player_poster_assets::*;
pub(crate) use player_poster_controls::*;
pub(crate) use player_theme_loader::*;
pub(crate) use run_log::*;
pub(crate) use save::*;
pub(crate) use separated_asset_path_native::*;
pub(crate) use separated_ui_assets::*;
pub(crate) use settings::*;
pub(crate) use title::*;
pub(crate) use title_menu_state::*;

// crate 外使用的入口
pub use main::Game;
