//! Go 來源：`remake/cmd/fd2/modern_hd_commands10_12_map.go`（294 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_commands10_12_map.go:15`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernHDCommand1012Frames {}

/// These producers build an unadorned typed input. Selection closures belong
/// to the separately frozen steady snapshot, never the sampled terrain.
///
/// Go：`modern_hd_commands10_12_map.go:23`，`func freezeModernHDCommandGeometry(in indexedmap.NativeFrameInput) (indexedmap.NativeFrameInput, error)`
pub(crate) fn freeze_modern_hd_command_geometry(
    r#in: &indexedmap::NativeFrameInput<'_>,
) -> Result<indexedmap::NativeFrameInput<'static>> {
    todo!("modern_hd_commands10_12_map.go:freezeModernHDCommandGeometry")
}
