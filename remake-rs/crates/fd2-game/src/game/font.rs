//! Go 來源：`remake/cmd/fd2/font.go`（287 行）。
//!
//! font.go — TTF 中文字型渲染(doc 18 字型現代化)。
//!
//! 設計決策:台詞/物品名都已解碼成 UTF-8(extracted/story/*),remake 用現代 TTF
//! (Noto Sans CJK)直接渲染 UTF-8，清晰可縮放，且不受原版 16×16 點陣字索引限制。
//! 實際字形覆蓋仍由載入的字型檔決定；不可假設所有繁體 codepoint 必然存在。
//! 原版自製「特殊代號字模」（機器人篇，知識庫有記錄）仍須逐字建立明確特例。
//!
//! 銳利度:Draw(scale) 不做 GeoM 縮放(非整數縮放重採樣=糊字根因,狀態欄名字踩過),
//! 而是把 scale 換算成目標像素尺寸,用 per-尺寸 face 快取直接 rasterize(scale 1.0 繪製)。
//!
//! 骨架：Go 檔的 13 個函式中產生 11 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Font 包一個 TTF,依目標像素尺寸 lazily 建 face(銳利,無縮放重採樣)。
///
/// Go：`font.go:27`
#[derive(Debug, Clone, Default)]
pub(crate) struct Font {
    pub(crate) sf: Option<Arc<SfntFont>>,
    /// Draw 的 scale=1.0 對應的像素尺寸
    pub(crate) base: f64,
    pub(crate) faces: BTreeMap<i64, Option<Arc<FontFace>>>,
    pub(crate) ascs: BTreeMap<i64, f64>,
    /// userScale 是玩家在設定裡選的字級倍率，套在每一次 Draw／Width 的 scale 上。
    /// 英文與日文譯文比中文長得多，同一個對話框裡常常斷不完；把字級調小一格就能
    /// 多塞一行。0 視同 1.0。索引畫面用的是原版字模，不經過這裡，所以調整字級不會
    /// 動到與原版對拍的那些畫面。
    pub(crate) user_scale: f64,
    pub(crate) hd: Option<ModernHDTextFrame>,
    pub(crate) indexed_hd: Option<ModernHDIndexedText>,
    pub(crate) hd_fonts: BTreeMap<String, Font>,
    /// 僅供HD戰鬥框英數；原版字型與斷行尺度保持。
    pub(crate) hd_heavy: Option<Box<Font>>,
}

impl Font {
    /// SetUserScale 設定字級倍率。超出合理範圍的值會被夾住——字太小讀不到、太大反而
    /// 更容易溢出版面。
    ///
    /// Go：`font.go:45`，`func (f *Font) SetUserScale(scale float64)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn set_user_scale(&mut self, scale: f64) {
        todo!("font.go:SetUserScale")
    }

    /// effectiveScale 把呼叫端要的 scale 乘上玩家設定。
    ///
    /// Go：`font.go:59`，`func (f *Font) effectiveScale(scale float64) float64`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn effective_scale(&self, scale: f64) -> f64 {
        todo!("font.go:effectiveScale")
    }
}

/// 字型搜尋路徑:打包用 assets 優先,否則用系統 Noto CJK(桌面)。
///
/// Go：`font.go:67`
pub(crate) static FONT_PATHS: LazyLock<Vec<String>> = LazyLock::new(|| {
    vec![
        String::from("assets/fonts/cjk.ttc"),
        String::from("assets/fonts/cjk.otf"),
        String::from("/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"),
        String::from("/usr/share/fonts/truetype/arphic/uming.ttc"),
    ]
});

pub(crate) const FONT_SIZE: f64 = 18.0;

/// loadFont 載入 CJK TTF/ttc(scale=1 → 18px)。失敗回 nil。
///
/// Go：`font.go:77`，`func loadFont() *Font`
pub(crate) fn load_font() -> Option<Font> {
    todo!("font.go:loadFont")
}

/// loadFontSized 指定 scale=1 的像素尺寸(狀態欄等固定尺寸用)。
///
/// Go：`font.go:80`，`func loadFontSized(size float64) *Font`
pub(crate) fn load_font_sized(size: f64) -> Option<Font> {
    todo!("font.go:loadFontSized")
}

/// 可攜TTC優先，逐一核對區域家族，缺件沿用既有可讀字型。
///
/// Go：`font.go:127`，`func loadHDRegionalFonts(size float64, weight string, paths []string) map[string]*Font`
pub(crate) fn load_hd_regional_fonts(size: f64, weight: &str, paths: &[String]) -> BTreeMap<String, Font> {
    todo!("font.go:loadHDRegionalFonts")
}

impl Font {
    /// faceFor 取(或建)目標像素尺寸的 face。
    ///
    /// Go：`font.go:157`，`func (f *Font) faceFor(px int) (font.Face, float64)`
    pub(crate) fn face_for(&mut self, px: i64) -> (Option<Arc<FontFace>>, f64) {
        todo!("font.go:faceFor")
    }

    /// Width 估算一行寬(像素)。
    ///
    /// Go：`font.go:194`，`func (f *Font) Width(s string, scale float64) float64`
    pub(crate) fn width(&mut self, s: &str, scale: f64) -> f64 {
        todo!("font.go:Width")
    }

    /// Wrap 依實際 glyph 寬度切行。斷點以 textSegments 為單位：繁中每個字都能斷，
    /// 英文與日文譯文裡的拉丁單字整串搬，不會斷在單字中間。明示換行仍保留。這是
    /// ending／說明頁與商店訊息使用的顯示工具，不改劇本文字。
    ///
    /// Go：`font.go:206`，`func (f *Font) Wrap(s string, scale, maxWidth float64) []string`
    pub(crate) fn wrap(&mut self, s: &str, scale: f64, max_width: f64) -> Vec<String> {
        todo!("font.go:Wrap")
    }
}

/// runeBreakable 判斷這個字元自己就能當一個斷行單位。中日韓文字與全形標點
/// 每個字都可以斷，拉丁字母與數字不行——那會把英文單字切成兩半。
///
/// Go：`font.go:214`，`func runeBreakable(r rune) bool`
pub(crate) fn rune_breakable(r: i32) -> bool {
    todo!("font.go:runeBreakable")
}

/// textSegments 把一段文字切成不可拆的顯示單位。空白是斷點，黏在前一段的尾巴——
/// 留著不砍，把每一行接回去才會等於原文（逐字表示與分頁都靠這個性質）；斷在那裡
/// 時它落在行尾，畫出來本來就看不見。
///
/// Go：`font.go:230`，`func textSegments(paragraph string) []string`
pub(crate) fn text_segments(paragraph: &str) -> Vec<String> {
    todo!("font.go:textSegments")
}

/// Go：`font.go:259`，`func wrapTextByWidth(s string, maxWidth float64, width func(string) float64) []string`
pub(crate) fn wrap_text_by_width(s: &str, max_width: f64, width: &mut dyn FnMut(String) -> f64) -> Vec<String> {
    todo!("font.go:wrapTextByWidth")
}
