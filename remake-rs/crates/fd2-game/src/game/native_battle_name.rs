//! Go 來源：`remake/cmd/fd2/native_battle_name.go`（135 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeBattleNameAssets 是全螢幕戰鬥狀態欄的原版姓名字模來源。
/// 原版 0x18c6d→0x15f84 走 FDOTHER#4 的 16×16 1bpp glyph，不與對話／現代
/// UTF-8 字型共用；索引表仍是可編輯資產，未知字元一律拒絕 native 路徑。
///
/// Go：`native_battle_name.go:22`，`func loadNativeBattleNameAssets() (*fdtxt.Font, map[string]int, error)`
pub(crate) fn load_native_battle_name_assets() -> Result<(Option<fdtxt::Font>, BTreeMap<String, i64>)> {
    todo!("native_battle_name.go:loadNativeBattleNameAssets")
}

/// Go：`native_battle_name.go:87`，`func nativePaletteColor(palette color.Palette, index byte, fallback color.RGBA) color.RGBA`
pub(crate) fn native_palette_color(palette: &ColorPalette, index: u8, fallback: ColorRgba) -> ColorRgba {
    todo!("native_battle_name.go:nativePaletteColor")
}
