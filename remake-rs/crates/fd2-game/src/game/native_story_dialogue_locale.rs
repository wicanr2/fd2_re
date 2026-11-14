//! Go 來源：`remake/cmd/fd2/native_story_dialogue_locale.go`（221 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_story_dialogue_locale.go:160`，`func drawIndexedLocalizedText( dst []byte, displayFont *Font, text string, x, y, maxWidth, height int, scale float64, foreground, shadow byte, ) error`
pub(crate) fn draw_indexed_localized_text(
    dst: &mut [u8],
    display_font: Option<&mut Font>,
    text: &str,
    x: i64,
    y: i64,
    max_width: i64,
    height: i64,
    scale: f64,
    foreground: u8,
    shadow: u8,
) -> Result<()> {
    // R4 B 組：非繁中語系把 TTF 字形點陣化寫進索引色畫格（`font.Drawer`、`sfnt.GlyphIndex`、HD 文字層），
    // 屬於 R5 的字型繪製；忠實模式繁中走 FDTXT 點陣字，不經過這裡。
    todo!("R5: native_story_dialogue_locale.go:drawIndexedLocalizedText（TTF 點陣化）")
}
