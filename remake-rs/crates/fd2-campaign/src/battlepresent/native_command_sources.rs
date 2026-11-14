//! 移植自 Go `battlepresent/native_command_sources.go`。

use fd2_formats::figani::{NativeCommand3Layer, NativeCommand6Layer, NativeCommand7Layer, NativeCommand8Layer};

/// NativeCommandSources 保存同次合成已消費的來源，不重新規劃狀態或亂數。
/// 預設順序為 Before、可選actor、target、After；明示旗標可前置或省略target。
///
/// Go：`native_command_sources.go:7`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandSources {
    pub base: Vec<u8>,
    pub actor_frame: i64,
    pub actor_offset_x: i64,
    pub target_index: i64,
    pub target_frame: i64,
    pub target_offset_x: i64,
    pub target_offset_y: i64,
    pub before: Vec<NativeCommandEffectSource>,
    pub after: Vec<NativeCommandEffectSource>,
    pub phase: String,
    pub target_before_effects: bool,
    pub omit_target: bool,
    pub packed_target_shader: i64,
}

/// Go：`native_command_sources.go:19`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandEffectSource {
    pub frame: i64,
    pub x: i64,
    pub y: i64,
}

/// Go：`native_command_sources.go:21`
pub(super) fn native_command6_source_layers(layers: &[NativeCommand6Layer]) -> Vec<NativeCommandEffectSource> {
    layers.iter().map(|l| NativeCommandEffectSource { frame: l.frame, x: l.x, y: l.y }).collect()
}

/// Go：`native_command_sources.go:29`
pub(super) fn native_command3_source_layers(layers: &[NativeCommand3Layer]) -> Vec<NativeCommandEffectSource> {
    layers.iter().map(|l| NativeCommandEffectSource { frame: l.frame, x: l.x, y: l.y }).collect()
}

/// Go：`native_command_sources.go:37`
pub(super) fn native_command7_source_layers(layers: &[NativeCommand7Layer]) -> Vec<NativeCommandEffectSource> {
    layers.iter().map(|l| NativeCommandEffectSource { frame: l.frame, x: l.x, y: 0 }).collect()
}

/// Go：`native_command_sources.go:45`
pub(super) fn native_command8_source_layers(layers: &[NativeCommand8Layer]) -> Vec<NativeCommandEffectSource> {
    layers.iter().map(|l| NativeCommandEffectSource { frame: l.frame, x: 0, y: 0 }).collect()
}
