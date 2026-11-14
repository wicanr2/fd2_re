//! Go 來源：`remake/cmd/fd2/modern_hd_confirmation.go`（280 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_confirmation.go:18`
pub(crate) static MODERN_HD_CONFIRMATION_INDICES: LazyLock<[i64; 6]> = LazyLock::new(|| [16, 17, 48, 49, 51, 52]);

/// Go：`modern_hd_confirmation.go:20`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernHDConfirmationCell {
    pub(crate) height: i64,
    pub(crate) background: ColorRgba,
    pub(crate) ink: ColorRgba,
    pub(crate) shadow: ColorRgba,
}

/// Go：`modern_hd_confirmation.go:25`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDConfirmation {
    pub(crate) cells: BTreeMap<i64, ModernHDConfirmationCell>,
    pub(crate) labels: BTreeMap<String, [String; 2]>,
    pub(crate) fonts: BTreeMap<String, Font>,
    pub(crate) bounds: BTreeMap<String, [Rectangle; 2]>,
    pub(crate) images: BTreeMap<ModernHDConfirmationImageKey, Option<Arc<Picture>>>,
}

/// Go：`modern_hd_confirmation.go:33`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct ModernHDConfirmationImageKey {}

/// Go：`modern_hd_confirmation.go:51`，`func freezeModernHDConfirmation(cells []fdother.RawCell, palette color.Palette, font *Font) (*modernHDConfirmation, error)`
pub(crate) fn freeze_modern_hd_confirmation(
    cells: &[fdother::RawCell],
    palette: &ColorPalette,
    font: Option<&mut Font>,
) -> Result<Option<ModernHDConfirmation>> {
    todo!("modern_hd_confirmation.go:freezeModernHDConfirmation")
}
