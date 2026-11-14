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
        todo!("modern_hd_facility_text.go:modernHDFacilityFont")
    }

    /// Go：`modern_hd_facility_text.go:62`，`func (g *Game) beginModernHDFacilityText() (*modernHDIndexedText, func())`
    pub(crate) fn begin_modern_hd_facility_text(
        &mut self,
    ) -> (Option<ModernHDIndexedText>, Option<Box<dyn FnMut(&mut Game) + Send>>) {
        todo!("modern_hd_facility_text.go:beginModernHDFacilityText")
    }
}
