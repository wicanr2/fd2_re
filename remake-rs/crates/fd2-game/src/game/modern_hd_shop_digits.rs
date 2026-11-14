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
        if !self.modern_hd_shop_text() || self.native_class_ui.is_none() || self.native_shop_ui.is_none() {
            return Some(Box::new(|_: &mut Game| {}));
        }
        // Go 把共用十進位字模與商店物品字模換成 HD 佔位字模，收尾時換回；佔位字模在繪字時寫回 HD 文字擷取。
        todo!("R6: modern_hd_shop_digits.go:beginModernHDShopDigits（HD 數字佔位）")
    }

    /// Go：`modern_hd_shop_digits.go:28`，`func (g *Game) modernHDDecimalFrames(source map[int]fdother.Frame) map[int]fdother.Frame`
    pub(crate) fn modern_hd_decimal_frames(
        &mut self,
        source: &BTreeMap<i64, fdother::Frame>,
    ) -> BTreeMap<i64, fdother::Frame> {
        let mut frames = source.clone();
        for base in [31i64, 42, 119] {
            for digit in 0..10 {
                if let Some(frame) = frames.get(&(base + digit)).cloned() {
                    let sprite = self.modern_hd_decimal_sprite(&frame, digit);
                    frames.insert(base + digit, sprite);
                }
            }
        }
        frames
    }

    /// Go：`modern_hd_shop_digits.go:43`，`func (g *Game) modernHDDecimalSprite(source fdother.Frame, digit int) fdother.Frame`
    pub(crate) fn modern_hd_decimal_sprite(&mut self, source: &fdother::Frame, digit: i64) -> fdother::Frame {
        // Production separated frames have an explicit binary mask. An unverified
        // or archive-only representation retains its ordinary sprite renderer.
        if source.indexed.len() as i64 != source.width.wrapping_mul(source.height)
            || source.mask.len() != source.indexed.len()
        {
            return source.clone();
        }
        // Go 直接解參考 `g.nativeClassUI`。
        let palette = &self.native_class_ui.as_ref().expect("modernHDDecimalSprite: nativeClassUI is nil").palette;
        let mut best: i64 = -1;
        for (i, &index) in source.indexed.iter().enumerate() {
            if source.mask[i] != 255 {
                continue;
            }
            let c = native_palette_color(palette, index, ColorRgba::default());
            let brightness = c.r as i64 + c.g as i64 + c.b as i64;
            if brightness > best {
                best = brightness;
            }
        }
        if best < 0 {
            return source.clone();
        }
        // Go 以 `source.TextPlaceholder` 建立佔位字模，繪字時把十進位數字的 HD 文字列寫回 `g.font.indexedHD`；
        // 回呼要寫回 `Game` 的擷取狀態，屬於 HD 文字層。
        todo!("R6: modern_hd_shop_digits.go:modernHDDecimalSprite（HD 數字佔位回呼）")
    }
}
