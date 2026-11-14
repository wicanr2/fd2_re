//! Go 來源：`remake/cmd/fd2/separated_ui_assets.go`（45 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`separated_ui_assets.go:10`，`func loadNativeBattlePalette() ([]byte, color.Palette, error)`
pub(crate) fn load_native_battle_palette() -> Result<(Vec<u8>, ColorPalette)> {
    let (dac, palette) = fdother::load_separated_fdother_palette(separated_asset_path("palette"), 0)?;
    Ok((dac, color_palette_from_nrgba(&palette)))
}

/// Go：`separated_ui_assets.go:14`，`func loadNativeUIPalette() color.Palette`
pub(crate) fn load_native_ui_palette() -> ColorPalette {
    match load_native_battle_palette() {
        Ok((_, palette)) => palette,
        Err(_) => ColorPalette::new(),
    }
}

/// loadNativeActionCellsWithRaw 同時回傳 indexed 原格（給 indexed 整幀的指令環用）。
///
/// Go：`separated_ui_assets.go:28`，`func loadNativeActionCellsWithRaw(palette color.Palette) ([]*ebiten.Image, []fdother.RawCell)`
pub(crate) fn load_native_action_cells_with_raw(
    palette: &ColorPalette,
) -> (Vec<Option<Arc<Picture>>>, Vec<fdother::RawCell>) {
    if palette.len() != 256 {
        return (Vec::new(), Vec::new());
    }
    let cells = match fdother::load_separated_action_cells(separated_asset_path("ui")) {
        Ok(cells) if cells.len() as i64 == NATIVE_ACTION_OVERLAY_CELL_COUNT => cells,
        _ => return (Vec::new(), Vec::new()),
    };
    let nrgba = nrgba_palette_from_color(palette);
    let mut images = Vec::with_capacity(cells.len());
    for cell in &cells {
        let Ok(decoded) = cell.paletted(&nrgba) else {
            return (Vec::new(), Vec::new());
        };
        images.push(Some(Arc::new(picture_from_paletted(&decoded))));
    }
    (images, cells)
}

#[cfg(test)]
#[path = "separated_ui_assets/separated_ui_assets_test.rs"]
mod separated_ui_assets_test;
