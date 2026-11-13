//! Go 來源：`remake/internal/battle/native_revive_list_render.go`（75 行）。

use super::*;

pub const NATIVE_REVIVE_VISIBLE: i64 = 3;

/// NativeReviveRow is one caller-resolved 0x30a47 candidate.
///
/// Go：`native_revive_list_render.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeReviveRow {
    pub sprite: fdicon::Sprite,
    pub name_text_index: i64,
    pub race_text_index: i64,
    pub class_text_index: i64,
    pub fee: i64,
}

/// RenderNativeReviveRows reproduces 0x30a47's three-row candidate renderer.
///
/// Go：`native_revive_list_render.go:23`，`func RenderNativeReviveRows( assets NativeItemPanelDataAssets, currencyCell fdother.RawCell, rows []NativeReviveRow, selected int, dst []byte, ) error`
pub fn render_native_revive_rows(
    assets: &NativeItemPanelDataAssets,
    currency_cell: &fdother::RawCell,
    rows: &[NativeReviveRow],
    selected: i64,
    dst: &mut [u8],
) -> Result<()> {
    if dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES
        || rows.is_empty()
        || rows.len() as i64 > NATIVE_REVIVE_VISIBLE
        || selected < 0
        || selected >= rows.len() as i64
    {
        return Err(battle_err!("battle: native revive rows/state are invalid"));
    }
    let mut staged = dst.to_vec();
    for (i, row) in rows.iter().enumerate() {
        let i64i = i as i64;
        if row.fee < 0 || row.fee > 0xffff {
            return Err(battle_err!("battle: native revive row {i} fee is invalid"));
        }
        row.sprite
            .blit_at(&mut staged, 320, 14, 117 + 26 * i64i)
            .map_err(|e| battle_err!("battle: native revive row {i} sprite: {e}"))?;
        let foreground = if i64i == selected { 201 } else { 205 };
        let y = 121 + 26 * i64i;
        for (x, index) in [(40, row.name_text_index), (130, row.race_text_index), (175, row.class_text_index)] {
            blit_native_item_panel_text_opt(
                assets.strings.as_ref(),
                assets.font.as_ref(),
                &mut staged,
                NativeItemPanelPoint { x, y },
                index,
                foreground,
            )
            .map_err(|e| e.wrap(format!("battle: native revive row {i} text {index}: ")))?;
        }
        currency_cell
            .blit_opaque_at_offset(&mut staged, 320, (y + 4) * 320 + 220)
            .map_err(|e| battle_err!("battle: native revive row {i} currency: {e}"))?;
        blit_native_item_panel_number(
            assets.frames_or_empty(),
            &mut staged,
            NativeItemPanelPoint { x: 228, y: y + 4 },
            row.fee,
            119,
            5,
        )
        .map_err(|e| e.wrap(format!("battle: native revive row {i} fee: ")))?;
    }
    dst.copy_from_slice(&staged);
    Ok(())
}

#[cfg(test)]
mod native_revive_list_render_test;
