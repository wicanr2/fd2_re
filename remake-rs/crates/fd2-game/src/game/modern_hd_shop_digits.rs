//! Go 來源：`remake/cmd/fd2/modern_hd_shop_digits.go`（84 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

impl Game {
    /// Only the verified decimal banks are replaced. Overflow and other pictorial
    /// markers keep their source sprites. These copies exist for one Draw only.
    ///
    /// Go：`modern_hd_shop_digits.go:13`，`func (g *Game) beginModernHDShopDigits() func()`
    pub(crate) fn begin_modern_hd_shop_digits(&mut self) -> Option<Box<dyn FnMut(&mut Game) + Send>> {
        todo!("modern_hd_shop_digits.go:beginModernHDShopDigits")
    }

    /// Go：`modern_hd_shop_digits.go:28`，`func (g *Game) modernHDDecimalFrames(source map[int]fdother.Frame) map[int]fdother.Frame`
    pub(crate) fn modern_hd_decimal_frames(
        &mut self,
        source: &BTreeMap<i64, fdother::Frame>,
    ) -> BTreeMap<i64, fdother::Frame> {
        todo!("modern_hd_shop_digits.go:modernHDDecimalFrames")
    }

    /// Go：`modern_hd_shop_digits.go:43`，`func (g *Game) modernHDDecimalSprite(source fdother.Frame, digit int) fdother.Frame`
    pub(crate) fn modern_hd_decimal_sprite(&mut self, source: &fdother::Frame, digit: i64) -> fdother::Frame {
        todo!("modern_hd_shop_digits.go:modernHDDecimalSprite")
    }
}
