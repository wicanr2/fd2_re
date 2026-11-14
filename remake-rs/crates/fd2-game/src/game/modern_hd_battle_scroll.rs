//! Go 來源：`remake/cmd/fd2/modern_hd_battle_scroll.go`（138 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Decode only the verified transparent FDOTHER blit. Two different initial
/// destinations distinguish an opaque index0 from a transparent write omission.
///
/// Go：`modern_hd_battle_scroll.go:14`，`func modernHDBattleSurfaceFrame(frame fdother.Frame) (figani.Frame, error)`
pub(crate) fn modern_hd_battle_surface_frame(frame: &fdother::Frame) -> Result<figani::Frame> {
    todo!("modern_hd_battle_scroll.go:modernHDBattleSurfaceFrame")
}
