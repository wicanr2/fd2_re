//! Go 來源：`remake/cmd/fd2/modern_hd_map_runtime.go`（349 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// A frame is prepared inside the original timing transaction and published
/// only with its native work/VGA/state. TerrainSeed never contains objects/UI.
///
/// Go：`modern_hd_map_runtime.go:18`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDMapFrame {
    pub(crate) units: Vec<ModernHDMapUnitDraw>,
    pub(crate) foreground: Vec<ModernHDMapForegroundDraw>,
    pub(crate) terrain_seed: Vec<u8>,
    pub(crate) terrain: Vec<ModernHDTerrainDraw>,
    pub(crate) overlays: Vec<ModernHDMapOverlayDraw>,
}

/// Go：`modern_hd_map_runtime.go:40`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDMapOverlayDraw {
    pub(crate) entry: i64,
    pub(crate) image: Option<Rgba>,
    pub(crate) rect: Rectangle,
}
