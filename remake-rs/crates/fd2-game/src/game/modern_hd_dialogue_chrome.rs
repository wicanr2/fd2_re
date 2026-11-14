//! Go 來源：`remake/cmd/fd2/modern_hd_dialogue_chrome.go`（247 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// This freezes FDOTHER#5's source validation and native panel colors.
/// Status/item panels keep the thin native bevel; dialogue owners explicitly
/// select the approved gold material in the same 18px border band.
///
/// Go：`modern_hd_dialogue_chrome.go:21`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDDialogueChrome {
    pub(crate) background: ColorRgba,
    /// top, right, bottom, left
    pub(crate) edges: [[ColorRgba; 3]; 4],
    /// top-left, top-right, bottom-left, bottom-right
    pub(crate) corners: [[ColorRgba; 9]; 4],
    pub(crate) gpu: Option<Arc<Picture>>,
    pub(crate) gold: Option<Arc<Picture>>,
    pub(crate) gold_gpu: BTreeMap<Point, Option<Arc<Picture>>>,
}

/// Go：`modern_hd_dialogue_chrome.go:30`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDDialoguePanel {
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    /// Frozen native 320x200 coordinates, before clipping.
    pub(crate) rect: Rectangle,
    /// Only dialogue owners use the approved antique-gold material.
    pub(crate) gold: bool,
    /// Legacy story box geometry, snapped to HD pixels.
    pub(crate) physical_rect: Rectangle,
}

/// Go：`modern_hd_dialogue_chrome.go:41`，`func freezeModernHDDialogueChrome(cells []fdother.RawCell, palette color.Palette) (*modernHDDialogueChrome, error)`
pub(crate) fn freeze_modern_hd_dialogue_chrome(
    cells: &[fdother::RawCell],
    palette: &ColorPalette,
) -> Result<Option<ModernHDDialogueChrome>> {
    todo!("modern_hd_dialogue_chrome.go:freezeModernHDDialogueChrome")
}

/// Code-generated from the approved SVG, not original game artwork. The 18px
/// corners stay fixed; the flat center strips repeat without scaling the bevel.
///
/// Go：`modern_hd_dialogue_chrome.go:208`
///
/// Go 以 `//go:embed modern_hd_gold_dialogue.png` 嵌入 `remake/cmd/fd2/modern_hd_gold_dialogue.png`；素材放哪裡由擁有本檔的組決定（`include_bytes!` 或執行期讀檔）。
pub(crate) static MODERN_HD_GOLD_DIALOGUE_TILE: LazyLock<Vec<u8>> = LazyLock::new(|| {
    todo!("modern_hd_dialogue_chrome.go:modernHDGoldDialogueTile go:embed modern_hd_gold_dialogue.png")
});
