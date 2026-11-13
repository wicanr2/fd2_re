//! FDOTHER.DAT 的原版資源與 FD2.EXE 的原版流程邏輯。
//!
//! 移植自 Go `remake/internal/fdother`（凍結於 v.1.1.11），每個 Go 檔對應一個子模組。
//! 位址與證據註解沿用 Go 版；錯誤訊息與 Go 版逐字相同，以便黃金差分直接比對。

mod battle_entry_step;
mod indexed_transition_buffer;
mod indexed_transition_lut;
mod indexed_transition_schedule;
mod map_binding;
mod native_ai_item_damage_tail;
mod native_command10_12;
mod native_command34_tail;
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
mod post_resolution_commands;
mod preparation_dispatch;
mod preparation_gate;
mod preparation_input;
mod preparation_timer;
mod system_info_transition;

pub use battle_entry_step::*;
pub use indexed_transition_buffer::*;
pub use indexed_transition_lut::*;
pub use indexed_transition_schedule::*;
pub use map_binding::*;
pub use native_ai_item_damage_tail::*;
pub use native_command_heal_presentation::*;
pub use native_command_heal_tail::*;
pub use native_command10_12::*;
pub use native_command34_tail::*;
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
pub use post_resolution_commands::*;
pub use preparation_dispatch::*;
pub use preparation_gate::*;
pub use preparation_input::*;
pub use preparation_timer::*;
pub use system_info_transition::*;
