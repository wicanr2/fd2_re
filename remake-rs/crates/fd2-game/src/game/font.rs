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
//!
//! # R4 的切分
//!
//! Go 以 `golang.org/x/image/font/opentype` 建 face、以 `text.BoundString` 量寬。`fd2-game` 沒有 TTF
//! 解析器（`Cargo.toml` 不加依賴），量寬與 ascent 交給外殼注入的 [`TextMeasure`]（`Font::measure`）。
//! 沒有注入時：沒有字型（`sf` 是 `None`）照 Go 回 0；有字型卻沒有量測器是 `todo!("R5: …")`，
//! 不以猜測的寬度推進分頁（失敗即關閉）。HD 區域字型（`loadHDRegionalFonts`、`hdFonts`）在 R6。

use super::*;

/// Rust 新增：外殼提供的字型量測（Go `opentype.NewFace` 的 ascent 與 `text.BoundString` 的寬）。
pub(crate) trait TextMeasure: Send + Sync {
    /// Go `opentype.NewFace(sf, Size: px, DPI: 72, HintingFull)` 後 `fc.Metrics().Ascent.Round()`；
    /// 建 face 失敗回 `None`（Go `faceFor` 回 `nil, 0`）。
    fn ascent(&self, face: &FontFace) -> Option<f64>;
    /// Go `text.BoundString(face, s)`（像素矩形，原點在基線起點）。
    fn bound_string(&self, face: &FontFace, s: &str) -> Rectangle;
    /// Go `sf.GlyphIndex(&buffer, r)` 成功且不是 0（字型有這個字）。
    fn has_glyph(&self, font: &SfntFont, r: char) -> bool;
}

/// Go `image.Rectangle.Dx()`／`Dy()`／`Empty()`。
pub(crate) fn rect_dx(r: &Rectangle) -> i64 {
    r.max.x - r.min.x
}

pub(crate) fn rect_dy(r: &Rectangle) -> i64 {
    r.max.y - r.min.y
}

pub(crate) fn rect_empty(r: &Rectangle) -> bool {
    r.min.x >= r.max.x || r.min.y >= r.max.y
}

/// `Font::measure` 的容器：讓 `Font` 保留 `Debug`、`Clone`、`Default`。
#[derive(Clone, Default)]
pub(crate) struct TextMeasurer(pub(crate) Option<Arc<dyn TextMeasure>>);

impl std::fmt::Debug for TextMeasurer {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(if self.0.is_some() { "TextMeasurer(Some)" } else { "TextMeasurer(None)" })
    }
}

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
    /// Rust 新增：外殼注入的量測器（見模組文件）。
    pub(crate) measure: TextMeasurer,
}

impl Font {
    /// SetUserScale 設定字級倍率。超出合理範圍的值會被夾住——字太小讀不到、太大反而
    /// 更容易溢出版面。
    ///
    /// Go：`font.go:45`，`func (f *Font) SetUserScale(scale float64)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn set_user_scale(&mut self, scale: f64) {
        // Go 先夾下限再夾上限；NaN 兩個比較都不成立而原樣保留，`f64::clamp` 同樣回傳 NaN。
        self.user_scale = scale.clamp(0.6, 1.6);
    }

    /// effectiveScale 把呼叫端要的 scale 乘上玩家設定。
    ///
    /// Go：`font.go:59`，`func (f *Font) effectiveScale(scale float64) float64`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn effective_scale(&self, scale: f64) -> f64 {
        if self.user_scale <= 0.0 {
            return scale;
        }
        scale * self.user_scale
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
    load_font_sized(FONT_SIZE)
}

/// Go `sfnt.ParseCollection`／`sfnt.Parse` 接受的檔頭：TTC（`ttcf`，至少一個字型）或單一
/// sfnt（`0x00010000`、`OTTO`、`true`）。完整表格驗證由量測器（R5）負責。
fn sfnt_header_ok(data: &[u8]) -> bool {
    if data.len() < 12 {
        return false;
    }
    match &data[..4] {
        b"ttcf" => u32::from_be_bytes([data[8], data[9], data[10], data[11]]) > 0,
        [0, 1, 0, 0] | b"OTTO" | b"true" => true,
        _ => false,
    }
}

/// loadFontSized 指定 scale=1 的像素尺寸(狀態欄等固定尺寸用)。
///
/// Go：`font.go:80`，`func loadFontSized(size float64) *Font`
///
/// Go 對 TTC 取第 0 個字型；Rust 的 `SfntFont` 保存整份檔案，量測器照 Go 取第 0 個。Go 另外依
/// 家族名稱建立的區域字型（`hdFonts`，含襯線 HD 字型）只給 HD 介面用，在 R6。
pub(crate) fn load_font_sized(size: f64) -> Option<Font> {
    let mut data = None;
    for p in FONT_PATHS.iter() {
        if let Ok(d) = std::fs::read(asset_path(p)) {
            data = Some(d);
            break;
        }
    }
    let data = data?;
    if !sfnt_header_ok(&data) {
        return None;
    }
    Some(Font { sf: Some(Arc::new(SfntFont { data })), base: size, ..Default::default() })
}

/// 可攜TTC優先，逐一核對區域家族，缺件沿用既有可讀字型。
///
/// Go：`font.go:127`，`func loadHDRegionalFonts(size float64, weight string, paths []string) map[string]*Font`
pub(crate) fn load_hd_regional_fonts(size: f64, weight: &str, paths: &[String]) -> BTreeMap<String, Font> {
    todo!("R6: font.go:loadHDRegionalFonts（HD 區域襯線字型，需要讀 sfnt name 表）")
}

impl Font {
    /// faceFor 取(或建)目標像素尺寸的 face。
    ///
    /// Go：`font.go:157`，`func (f *Font) faceFor(px int) (font.Face, float64)`
    pub(crate) fn face_for(&mut self, mut px: i64) -> (Option<Arc<FontFace>>, f64) {
        if px < 6 {
            px = 6;
        }
        if let Some(fc) = self.faces.get(&px) {
            return (fc.clone(), self.ascs.get(&px).copied().unwrap_or(0.0));
        }
        let Some(sf) = self.sf.clone() else {
            // Go `opentype.NewFace(nil, …)` 失敗：不快取，回 nil。
            return (None, 0.0);
        };
        let face = FontFace { font: Some(sf), size: px as f64, dpi: 72.0 };
        let Some(measure) = self.measure.0.clone() else {
            todo!("R5: font.go:faceFor 需要外殼注入 TextMeasure（opentype face 的 ascent）")
        };
        let Some(asc) = measure.ascent(&face) else {
            return (None, 0.0);
        };
        let face = Some(Arc::new(face));
        self.faces.insert(px, face.clone());
        self.ascs.insert(px, asc);
        (face, asc)
    }

    /// Width 估算一行寬(像素)。
    ///
    /// Go：`font.go:194`，`func (f *Font) Width(s string, scale float64) float64`
    pub(crate) fn width(&mut self, s: &str, scale: f64) -> f64 {
        let px = (self.base * self.effective_scale(scale)).round() as i64;
        let (face, _) = self.face_for(px);
        let Some(face) = face else {
            return 0.0;
        };
        match self.measure.0.as_ref() {
            Some(measure) => rect_dx(&measure.bound_string(&face, s)) as f64,
            None => todo!("R5: font.go:Width 需要外殼注入 TextMeasure（text.BoundString）"),
        }
    }

    /// Wrap 依實際 glyph 寬度切行。斷點以 textSegments 為單位：繁中每個字都能斷，
    /// 英文與日文譯文裡的拉丁單字整串搬，不會斷在單字中間。明示換行仍保留。這是
    /// ending／說明頁與商店訊息使用的顯示工具，不改劇本文字。
    ///
    /// Go：`font.go:206`，`func (f *Font) Wrap(s string, scale, maxWidth float64) []string`
    pub(crate) fn wrap(&mut self, s: &str, scale: f64, max_width: f64) -> Vec<String> {
        wrap_text_by_width(s, max_width, &mut |line: String| self.width(&line, scale))
    }
}

/// Go `unicode.Han`、`unicode.Hiragana`、`unicode.Katakana`、`unicode.Hangul`（Unicode 15.0.0，
/// Go 1.22.12 的 `unicode.Scripts` 表）：`(lo, hi, stride)`，同 Go `unicode.Range16`／`Range32`。
/// 2026-11-14 以 `fd2-go-test-local` 容器列印 `unicode.Scripts` 取得；這是 Unicode Script 屬性
/// 資料，Rust 標準函式庫沒有對應。
const CJK_SCRIPT_RANGES: &[(u32, u32, u32)] = &[
    // Han
    (0x2E80, 0x2E99, 1),
    (0x2E9B, 0x2EF3, 1),
    (0x2F00, 0x2FD5, 1),
    (0x3005, 0x3007, 2),
    (0x3021, 0x3029, 1),
    (0x3038, 0x303B, 1),
    (0x3400, 0x4DBF, 1),
    (0x4E00, 0x9FFF, 1),
    (0xF900, 0xFA6D, 1),
    (0xFA70, 0xFAD9, 1),
    (0x16FE2, 0x16FE3, 1),
    (0x16FF0, 0x16FF1, 1),
    (0x20000, 0x2A6DF, 1),
    (0x2A700, 0x2B739, 1),
    (0x2B740, 0x2B81D, 1),
    (0x2B820, 0x2CEA1, 1),
    (0x2CEB0, 0x2EBE0, 1),
    (0x2F800, 0x2FA1D, 1),
    (0x30000, 0x3134A, 1),
    (0x31350, 0x323AF, 1),
    // Hiragana
    (0x3041, 0x3096, 1),
    (0x309D, 0x309F, 1),
    (0x1B001, 0x1B11F, 1),
    (0x1B132, 0x1B150, 30),
    (0x1B151, 0x1B152, 1),
    (0x1F200, 0x1F200, 1),
    // Katakana
    (0x30A1, 0x30FA, 1),
    (0x30FD, 0x30FF, 1),
    (0x31F0, 0x31FF, 1),
    (0x32D0, 0x32FE, 1),
    (0x3300, 0x3357, 1),
    (0xFF66, 0xFF6F, 1),
    (0xFF71, 0xFF9D, 1),
    (0x1AFF0, 0x1AFF3, 1),
    (0x1AFF5, 0x1AFFB, 1),
    (0x1AFFD, 0x1AFFE, 1),
    (0x1B000, 0x1B120, 288),
    (0x1B121, 0x1B122, 1),
    (0x1B155, 0x1B164, 15),
    (0x1B165, 0x1B167, 1),
    // Hangul
    (0x1100, 0x11FF, 1),
    (0x302E, 0x302F, 1),
    (0x3131, 0x318E, 1),
    (0x3200, 0x321E, 1),
    (0x3260, 0x327E, 1),
    (0xA960, 0xA97C, 1),
    (0xAC00, 0xD7A3, 1),
    (0xD7B0, 0xD7C6, 1),
    (0xD7CB, 0xD7FB, 1),
    (0xFFA0, 0xFFBE, 1),
    (0xFFC2, 0xFFC7, 1),
    (0xFFCA, 0xFFCF, 1),
    (0xFFD2, 0xFFD7, 1),
    (0xFFDA, 0xFFDC, 1),
];

/// runeBreakable 判斷這個字元自己就能當一個斷行單位。中日韓文字與全形標點
/// 每個字都可以斷，拉丁字母與數字不行——那會把英文單字切成兩半。
///
/// Go：`font.go:214`，`func runeBreakable(r rune) bool`
pub(crate) fn rune_breakable(r: i32) -> bool {
    if r >= 0 {
        let r = r as u32;
        if CJK_SCRIPT_RANGES.iter().any(|&(lo, hi, stride)| r >= lo && r <= hi && (r - lo).is_multiple_of(stride)) {
            return true;
        }
    }
    // 全形標點；全形英數與半形片假名
    (0x3000..=0x303F).contains(&r) || (0xFF00..=0xFFEF).contains(&r)
}

/// textSegments 把一段文字切成不可拆的顯示單位。空白是斷點，黏在前一段的尾巴——
/// 留著不砍，把每一行接回去才會等於原文（逐字表示與分頁都靠這個性質）；斷在那裡
/// 時它落在行尾，畫出來本來就看不見。
///
/// Go：`font.go:230`，`func textSegments(paragraph string) []string`
///
/// Go `range` 字串遇到無效 UTF-8 得到 U+FFFD；Rust 的 `&str` 必是合法 UTF-8。
pub(crate) fn text_segments(paragraph: &str) -> Vec<String> {
    let mut segments: Vec<String> = Vec::new();
    let mut current = String::new();
    fn flush(segments: &mut Vec<String>, current: &mut String) {
        if !current.is_empty() {
            segments.push(std::mem::take(current));
        }
    }
    for r in paragraph.chars() {
        if r == ' ' || r == '\t' {
            if current.is_empty() && !segments.is_empty() {
                segments.last_mut().unwrap().push(r);
            } else {
                current.push(r);
                flush(&mut segments, &mut current);
            }
        } else if rune_breakable(r as i32) {
            flush(&mut segments, &mut current);
            segments.push(r.to_string());
        } else {
            current.push(r);
        }
    }
    flush(&mut segments, &mut current);
    segments
}

/// Go：`font.go:259`，`func wrapTextByWidth(s string, maxWidth float64, width func(string) float64) []string`
pub(crate) fn wrap_text_by_width(s: &str, max_width: f64, width: &mut dyn FnMut(String) -> f64) -> Vec<String> {
    let mut lines = Vec::new();
    for paragraph in s.split('\n') {
        let mut line = String::new();
        for segment in text_segments(paragraph) {
            if !line.is_empty() && width(format!("{line}{segment}")) > max_width {
                lines.push(std::mem::take(&mut line));
            }
            if line.is_empty() && width(segment.clone()) > max_width {
                // 這一段自己就放不下：逐字元硬拆，總比整行溢出好。
                for r in segment.chars() {
                    if !line.is_empty() && width(format!("{line}{r}")) > max_width {
                        lines.push(std::mem::take(&mut line));
                    }
                    line.push(r);
                }
                continue;
            }
            line.push_str(&segment);
        }
        lines.push(line);
    }
    lines
}

#[cfg(test)]
#[path = "font/font_test.rs"]
mod font_test;
