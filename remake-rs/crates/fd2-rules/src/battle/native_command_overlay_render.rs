//! Go 來源：`remake/internal/battle/native_command_overlay_render.go`（57 行）。

use super::*;

/// RenderNativeCommandOverlay reproduces 0x1ceed over an already composed
/// 0x17eef/0x17fc0 panel. Commands remain in 0x1c269 bit order; no normalized
/// spell list or effect semantics are consulted.
///
/// Go：`native_command_overlay_render.go:11`，`func RenderNativeCommandOverlay( assets NativeItemPanelDataAssets, commandIDs []int, book []NativeCommandRecord, selected int, dst []byte, ) error`
pub fn render_native_command_overlay(
    assets: &NativeItemPanelDataAssets,
    command_ids: &[i64],
    book: &[NativeCommandRecord],
    selected: i64,
    dst: &mut [u8],
) -> Result<()> {
    todo!("battle/native_command_overlay_render.go:RenderNativeCommandOverlay")
}
