//! Go 來源：`remake/cmd/fd2/modern_hd_player9_map.go`（251 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_player9_map.go:14`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernHDPlayer9Frames {}

impl Game {
    /// Capture the input published with the actual steady snapshot. Legacy direct
    /// fixtures may supply a reconstructed input only if its complete VGA matches.
    ///
    /// Go：`modern_hd_player9_map.go:22`，`func (g *Game) freezeNativeCommand9MapInput() (indexedmap.NativeFrameInput, error)`
    pub(crate) fn freeze_native_command9_map_input(&mut self) -> Result<indexedmap::NativeFrameInput<'static>> {
        todo!("modern_hd_player9_map.go:freezeNativeCommand9MapInput")
    }
}
