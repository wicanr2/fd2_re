//! Go 來源：`remake/cmd/fd2/modern_hd_text.go`（180 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Text uses the same logical coordinates as the game, with a separate raster
/// surface at the selected HD resolution. Game state and line wrapping stay
/// independent of display resolution.
///
/// Go：`modern_hd_text.go:16`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDTextRun {
    pub(crate) font: Option<Font>,
    pub(crate) text: String,
    pub(crate) x: f64,
    pub(crate) y: f64,
    pub(crate) pixels: f64,
    pub(crate) color: ColorRgba,
    pub(crate) clip: Rectangle,
    pub(crate) ink_top: bool,
}

/// Go：`modern_hd_text.go:66`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDTextFrame {
    pub(crate) money: Option<ModernHDFacilityMoneyPanel>,
    pub(crate) scene: Option<Arc<Picture>>,
    pub(crate) runs: Vec<ModernHDTextRun>,
    pub(crate) panels: Vec<ModernHDDialoguePanel>,
}

/// Go：`modern_hd_text.go:75`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDIndexedText {
    pub(crate) runs: Vec<ModernHDTextRun>,
    pub(crate) palette: ColorPalette,
    pub(crate) writes: Option<Vec<ModernHDGlyphWrite>>,
}

/// Go：`modern_hd_text.go:81`，`func modernHDNativeTextColor(palette color.Palette, index byte) color.RGBA`
pub(crate) fn modern_hd_native_text_color(palette: &ColorPalette, index: u8) -> ColorRgba {
    todo!("modern_hd_text.go:modernHDNativeTextColor")
}

impl Game {
    /// Go：`modern_hd_text.go:92`，`func (g *Game) beginModernHDText(screen *ebiten.Image) func()`
    pub(crate) fn begin_modern_hd_text(&mut self, screen: &mut Picture) -> Option<Box<dyn FnMut(&mut Game) + Send>> {
        todo!("modern_hd_text.go:beginModernHDText")
    }

    /// Go：`modern_hd_text.go:166`，`func (g *Game) modernHDShopText() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn modern_hd_shop_text(&self) -> bool {
        todo!("modern_hd_text.go:modernHDShopText")
    }

    /// Go：`modern_hd_text.go:170`，`func (g *Game) clearModernHDShopText()`
    pub(crate) fn clear_modern_hd_shop_text(&mut self) {
        todo!("modern_hd_text.go:clearModernHDShopText")
    }
}
