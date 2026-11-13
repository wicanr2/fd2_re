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
    if dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES || selected < -1 || selected >= command_ids.len() as i64 {
        return Err(battle_err!("battle: native command overlay inputs are invalid"));
    }
    if command_ids.len() > 32 {
        return Err(battle_err!("battle: native command overlay exceeds local command buffer"));
    }
    let (raw_cells, frames) = (assets.raw_cells_or_empty(), assets.frames_or_empty());
    let mut staged = dst.to_vec();
    for (i, &command_id) in command_ids.iter().enumerate() {
        if command_id < 0 || command_id >= book.len() as i64 || book[command_id as usize].id != command_id {
            return Err(battle_err!("battle: native command record {command_id} is unavailable"));
        }
        let i = i as i64;
        let (column, row) = (i / 4, i % 4);
        let (x, y) = (18 + 100 * column, 103 + 22 * row);
        let foreground = if i == selected { 201 } else { 205 };
        blit_native_item_panel_text_opt(
            assets.strings.as_ref(),
            assets.font.as_ref(),
            &mut staged,
            NativeItemPanelPoint { x, y },
            441 + command_id,
            foreground,
        )
        .map_err(|e| e.wrap(format!("battle: native command {command_id} label: ")))?;
        blit_native_item_panel_raw_cell(raw_cells, 92, &mut staged, (y + 5) * 320 + x + 50)
            .map_err(|e| e.wrap(format!("battle: native command {command_id} MP icon: ")))?;
        blit_native_item_panel_number(
            frames,
            &mut staged,
            NativeItemPanelPoint { x: x + 73, y: y + 5 },
            book[command_id as usize].mp_cost,
            42,
            2,
        )
        .map_err(|e| e.wrap(format!("battle: native command {command_id} MP cost: ")))?;
    }
    dst.copy_from_slice(&staged);
    Ok(())
}

#[cfg(test)]
mod native_command_overlay_render_test;
