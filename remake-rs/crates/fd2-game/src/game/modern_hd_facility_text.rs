//! Go 來源：`remake/cmd/fd2/modern_hd_facility_text.go`（147 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

impl Game {
    /// A caller uses this only around a fresh composition, then publishes the
    /// returned runs alongside that exact frame. Cached animation frames need their
    /// own explicit descriptors and must not use this adapter.
    ///
    /// Go：`modern_hd_facility_text.go:17`，`func (g *Game) modernHDFacilityFont(source *fdtxt.Font, capture *modernHDIndexedText) *fdtxt.Font`
    /// Go 第 1 個回傳值是指向參數 `source` 的指標；骨架回傳擁有的值，移植時再決定借用方式。
    pub(crate) fn modern_hd_facility_font(
        &mut self,
        source: Option<&fdtxt::Font>,
        capture: Option<&mut ModernHDIndexedText>,
    ) -> Option<fdtxt::Font> {
        if self.modern_hd_text.is_none() || self.font.is_none() || source.is_none() || capture.is_none() {
            return source.cloned();
        }
        // Go 以 `source.WithNativePresenter` 換上記錄 HD 文字的 presenter：以 `sfnt.GlyphIndex` 檢查字形、
        // 在繪字時把文字列寫回 `capture`。presenter 是 `Arc<dyn Fn>`，要寫回 `capture` 須改成共享擷取狀態，
        // 屬於 HD 文字層。
        todo!("R6: modern_hd_facility_text.go:modernHDFacilityFont（HD 文字擷取 presenter）")
    }

    /// Go：`modern_hd_facility_text.go:62`，`func (g *Game) beginModernHDFacilityText() (*modernHDIndexedText, func())`
    pub(crate) fn begin_modern_hd_facility_text(
        &mut self,
    ) -> (Option<ModernHDIndexedText>, Option<Box<dyn FnMut(&mut Game) + Send>>) {
        if self.modern_hd_text.is_none() || self.font.is_none() || self.native_class_ui.is_none() {
            return (None, Some(Box::new(|_: &mut Game| {})));
        }
        // Go 建立擷取、把 `nativeClassUI` 換成字型改用 `modernHDFacilityFont` 的複本，回傳換回的收尾。
        todo!("R6: modern_hd_facility_text.go:beginModernHDFacilityText（HD 文字擷取）")
    }
}
