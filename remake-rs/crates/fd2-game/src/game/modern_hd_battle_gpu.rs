//! Go 來源：`remake/cmd/fd2/modern_hd_battle_gpu.go`（385 行）。
//!
//! 骨架：Go 檔的 9 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_battle_gpu.go:43`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDBattleGPU {
    pub(crate) owner: Option<OwnerToken>,
    pub(crate) theme: Option<ModernStoryPortraitSet>,
    pub(crate) camera_owner: Option<OwnerToken>,
    pub(crate) camera_theme: Option<ModernStoryPortraitSet>,
    pub(crate) camera_start_frame: i64,
    pub(crate) camera_active: bool,
}

/// Go：`modern_hd_battle_gpu.go:202`，`func modernHDBattlePalette(palette color.Palette) ([]byte, error)`
pub(crate) fn modern_hd_battle_palette(palette: &ColorPalette) -> Result<Vec<u8>> {
    todo!("modern_hd_battle_gpu.go:modernHDBattlePalette")
}
