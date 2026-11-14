//! Go 來源：`remake/cmd/fd2/modern_hd_compound_map.go`（317 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// A scene belongs to one frozen pre/post-transaction map, not the live state.
///
/// Go：`modern_hd_compound_map.go:14`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Clone, Default)]
pub(crate) struct ModernHDCompoundMapScene {
    pub(crate) map_index: i64,
    pub(crate) stage: Option<NativeCh23AdapterState>,
    pub(crate) signature: [u8; 32],
    pub(crate) sealed: bool,
    pub(crate) input: indexedmap::NativeFrameInput<'static>,
    pub(crate) work: Vec<u8>,
    pub(crate) vga: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) err: Option<GameError>,
}

/// Go：`modern_hd_compound_map.go:25`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernHDCompoundMapSource {}

/// Go：`modern_hd_compound_map.go:44`，`func newModernHDCompoundMapScene(in indexedmap.NativeFrameInput, work, vga []byte, palette color.Palette, inputErr error) *modernHDCompoundMapScene`
pub(crate) fn new_modern_hd_compound_map_scene(
    r#in: &indexedmap::NativeFrameInput<'_>,
    work: &[u8],
    vga: &[u8],
    palette: &ColorPalette,
    input_err: Option<&GameError>,
) -> Option<ModernHDCompoundMapScene> {
    todo!("modern_hd_compound_map.go:newModernHDCompoundMapScene")
}
