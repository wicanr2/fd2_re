//! Go 來源：`remake/cmd/fd2/modern_hd_map_overlay_assets.go`（196 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_map_overlay_assets.go:17`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernLMIHDDescriptor {}

/// Go：`modern_hd_map_overlay_assets.go:24`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernMapOverlayHDResource {
    pub(crate) frames: BTreeMap<i64, Rgba>,
    pub(crate) native: BTreeMap<i64, ModernLMIHDDescriptor>,
}

/// Go：`modern_hd_map_overlay_assets.go:106`，`func hasModernHDDeathBank(set *modernStoryPortraitSet) bool`
pub(crate) fn has_modern_hd_death_bank(set: Option<&ModernStoryPortraitSet>) -> bool {
    todo!("modern_hd_map_overlay_assets.go:hasModernHDDeathBank")
}
