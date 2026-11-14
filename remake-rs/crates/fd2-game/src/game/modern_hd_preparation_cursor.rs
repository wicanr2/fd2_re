//! Go 來源：`remake/cmd/fd2/modern_hd_preparation_cursor.go`（156 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_preparation_cursor.go:14`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernPreparationCursor {}

/// Go：`modern_hd_preparation_cursor.go:20`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernPreparationNativeCursorUnit {}

/// Go：`modern_hd_preparation_cursor.go:36`，`func preparationCursorPalette(palette color.Palette) ([]color.RGBA, bool)`
pub(crate) fn preparation_cursor_palette(palette: &ColorPalette) -> (Vec<ColorRgba>, bool) {
    todo!("modern_hd_preparation_cursor.go:preparationCursorPalette")
}
