//! Go 來源：`remake/cmd/fd2/modern_hd_map_unit_plan.go`（89 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_map_unit_plan.go:12`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernHDMapUnitDraw {
    pub(crate) group: i64,
    pub(crate) frame: i64,
    pub(crate) rect: Rectangle,
    pub(crate) work_offset: i64,
    pub(crate) palette_band: bool,
}
