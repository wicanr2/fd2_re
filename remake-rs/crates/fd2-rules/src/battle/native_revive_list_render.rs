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
    todo!("battle/native_revive_list_render.go:RenderNativeReviveRows")
}
