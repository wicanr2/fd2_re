//! Go 來源：`remake/cmd/fd2/separated_ui_assets.go`（45 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`separated_ui_assets.go:10`，`func loadNativeBattlePalette() ([]byte, color.Palette, error)`
pub(crate) fn load_native_battle_palette() -> Result<(Vec<u8>, ColorPalette)> {
    todo!("separated_ui_assets.go:loadNativeBattlePalette")
}

/// Go：`separated_ui_assets.go:14`，`func loadNativeUIPalette() color.Palette`
pub(crate) fn load_native_ui_palette() -> ColorPalette {
    todo!("separated_ui_assets.go:loadNativeUIPalette")
}

/// loadNativeActionCellsWithRaw 同時回傳 indexed 原格（給 indexed 整幀的指令環用）。
///
/// Go：`separated_ui_assets.go:28`，`func loadNativeActionCellsWithRaw(palette color.Palette) ([]*ebiten.Image, []fdother.RawCell)`
pub(crate) fn load_native_action_cells_with_raw(
    palette: &ColorPalette,
) -> (Vec<Option<Arc<Picture>>>, Vec<fdother::RawCell>) {
    todo!("separated_ui_assets.go:loadNativeActionCellsWithRaw")
}
