//! Go 來源：`remake/cmd/fd2/modern_hd_indexed_transition.go`（198 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_indexed_transition.go:16`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernHDIndexedTransitionSource {}

/// Go：`modern_hd_indexed_transition.go:25`，`func indexedHDInputIdentity(in indexedmap.NativeTransitionFrameInput) ([]any, error)`
pub(crate) fn indexed_hd_input_identity(
    r#in: &indexedmap::NativeTransitionFrameInput<'_>,
) -> Result<Vec<Option<OwnerToken>>> {
    todo!("modern_hd_indexed_transition.go:indexedHDInputIdentity")
}
