//! FDOTHER.DAT 的原版資源與 FD2.EXE 的原版流程邏輯，以及 BG／TAI／FDSHAP 等共用的原生格式。
//!
//! 移植自 Go `remake/internal/fdother`（凍結於 v.1.1.11），每個 Go 檔對應一個子模組。
//! 位址與證據註解沿用 Go 版；錯誤訊息與 Go 版逐字相同，以便黃金差分直接比對。
//!
//! 核心與分離載入器（`frame`、`lmi1`、`separated_*` 等）刻意只公開索引像素：調色盤與 UI 排程
//! 歸各自的呼叫端。格式與 RLE 直接轉寫自 FD2.EXE `0x2935b` 與 `0x4e63d`；特別是透明命令保留
//! 目的端，不可像 PNG 匯出那樣轉成調色盤索引 0。這一組的錯誤型別是 [`FdotherError`]，原版
//! 流程邏輯那一組是 [`NativeFlowError`]。
//!
//! 核心與分離載入器的 Go 慣用寫法對應（行為以凍結 Go 版為準）：
//!
//! - Go `int` 一律對應 `i64`；呼叫端給的值可能讓 Go 的乘加繞回，這些地方用 `wrapping_*`。
//!   Go 會因越界索引或除以零 panic 的地方，Rust 也 panic。
//! - Go 接受 nil 指標或 nil 函式值並回傳錯誤的分支（nil bank、nil surface、nil 回呼），
//!   Rust 以參考與泛型閉包排除 nil，該分支不存在。
//! - `map[int]T` 改成 `BTreeMap<i64, T>`；`color.Palette`／`*image.Paletted` 改成
//!   [`Palette`]／[`PalettedImage`]；函式值欄位改成 `Arc<dyn Fn>`。
//! - 分離素材的 JSON、PNG 與路徑改用已鎖定的 serde_json、png crate 與 `std::path`，不逐行移植
//!   Go 標準函式庫；說明見 `go_compat` 模組。格式正確的輸入結果與 Go 相同，包裝的內層錯誤
//!   文字（[`FdotherError::External`]）則是 Rust 函式庫的。
//!
//! 依賴核心與原版流程邏輯的第三組（`action_overlay`、`native_ch23`、`preparation_roster`、
//! `spawn_intro`、`unit_present_schedule` 等）也用 [`FdotherError`]：它們的錯誤多半包裝核心那一組的
//! 繪圖與載入錯誤。Go 的 nil callback 以 `Option` 保留拒收分支；Go 閉包捕捉同一個緩衝再改寫的地方，
//! 改成把緩衝交給 callback，或像 `run_native_ch23_loop` 一樣用 `&[Cell<u8>]` 保留共用別名。
//!
//! 封存目錄已拆到 [`crate::archive`]，這裡轉出同名函式，保留 Go `fdother.ReadResource` 的對應。

mod go_compat;

mod action_overlay;
mod battle_entry_step;
mod ch28_post_present;
mod chapter_aux_surface;
mod dialogue_grid;
mod frame;
mod indexed_transition;
mod indexed_transition_buffer;
mod indexed_transition_lut;
mod indexed_transition_pass;
mod indexed_transition_schedule;
mod lmi1;
mod lmi1_frame;
mod lutbank;
mod map_binding;
mod map_parallax;
mod native_ai_item_damage_tail;
mod native_ai_item_type21_tail;
mod native_ch23;
mod native_command10_12;
mod native_command20_22_tail;
mod native_command32_tail;
mod native_command33_tail;
mod native_command34_tail;
mod native_command35_tail;
mod native_command9;
mod native_command_heal_presentation;
mod native_command_heal_tail;
mod native_compound_palette;
mod native_flow_error;
mod native_hotel_route;
mod native_phase_dispatch;
mod native_postbattle_gate;
mod native_preparation_cap;
mod native_preparation_key;
mod native_shop_route;
mod native_unit_mode;
mod native_unit_mode_branch;
mod palette;
mod pending_code1_interlude;
mod post_resolution_commands;
mod preparation_dispatch;
mod preparation_gate;
mod preparation_input;
mod preparation_roster;
mod preparation_timer;
mod range_overlay;
mod rawcells;
mod remap;
mod separated_action_cells;
mod separated_chapter_aux_surface;
mod separated_church_ui;
mod separated_frame_bank;
mod separated_item_panel;
mod separated_lmi1_bank;
mod separated_load_slots;
mod separated_lutbank;
mod separated_palette;
mod separated_range_overlay;
mod separated_sound;
mod separated_surface;
mod spawn_intro;
mod split_slide;
mod sprite_bank;
mod system_info_transition;
mod terrain_lut;
mod terrain_phase;
mod unit_present_schedule;

#[cfg(test)]
mod tests;

pub use crate::archive::{
    ArchiveError, archive_data_resource_count, archive_entry, archive_resource_count, read_nested_resource,
    read_resource,
};
pub use crate::native_rng::{native_rng_marker, native_rng_step};

pub use action_overlay::*;
pub use battle_entry_step::*;
pub use ch28_post_present::*;
pub use chapter_aux_surface::*;
pub use dialogue_grid::*;
pub use frame::*;
pub use indexed_transition::*;
pub use indexed_transition_buffer::*;
pub use indexed_transition_lut::*;
pub use indexed_transition_pass::*;
pub use indexed_transition_schedule::*;
pub use lmi1::*;
pub use lmi1_frame::*;
pub use lutbank::*;
pub use map_binding::*;
pub use map_parallax::*;
pub use native_ai_item_damage_tail::*;
pub use native_ai_item_type21_tail::*;
pub use native_ch23::*;
pub use native_command_heal_presentation::*;
pub use native_command_heal_tail::*;
pub use native_command9::*;
pub use native_command10_12::*;
pub use native_command20_22_tail::*;
pub use native_command32_tail::*;
pub use native_command33_tail::*;
pub use native_command34_tail::*;
pub use native_command35_tail::*;
pub use native_compound_palette::*;
pub use native_flow_error::*;
pub use native_hotel_route::*;
pub use native_phase_dispatch::*;
pub use native_postbattle_gate::*;
pub use native_preparation_cap::*;
pub use native_preparation_key::*;
pub use native_shop_route::*;
pub use native_unit_mode::*;
pub use native_unit_mode_branch::*;
pub use palette::*;
pub use pending_code1_interlude::*;
pub use post_resolution_commands::*;
pub use preparation_dispatch::*;
pub use preparation_gate::*;
pub use preparation_input::*;
pub use preparation_roster::*;
pub use preparation_timer::*;
pub use range_overlay::*;
pub use rawcells::*;
pub use remap::*;
pub use separated_action_cells::*;
pub use separated_chapter_aux_surface::*;
pub use separated_church_ui::*;
pub use separated_frame_bank::*;
pub use separated_item_panel::*;
pub use separated_lmi1_bank::*;
pub use separated_load_slots::*;
pub use separated_lutbank::*;
pub use separated_palette::*;
pub use separated_range_overlay::*;
pub use separated_sound::*;
pub use separated_surface::*;
pub use spawn_intro::*;
pub use split_slide::*;
pub use sprite_bank::*;
pub use system_info_transition::*;
pub use terrain_lut::*;
pub use terrain_phase::*;
pub use unit_present_schedule::*;
