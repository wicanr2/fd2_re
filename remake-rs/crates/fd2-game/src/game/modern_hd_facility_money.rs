//! Go 來源：`remake/cmd/fd2/modern_hd_facility_money.go`（178 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_facility_money.go:18`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDFacilityMoneyPanel {
    pub(crate) owner: String,
    pub(crate) variant: i64,
    pub(crate) resource: i64,
    pub(crate) presented: bool,
    pub(crate) rolling: bool,
    pub(crate) art_hd: bool,
}
