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
//! # 量測與繪字（R5 r5f）
//!
//! Go 以 `golang.org/x/image/font/opentype` 建 face、以 Ebiten `text.BoundString` 量寬、`text.DrawWithOptions`
//! 繪字。Rust 的 [`TextMeasure`]（`Font::measure`）把這三件事包在一起：`load_font_sized` 讀到字型時注入
//! [`XImageMeasure`]（`crate::go_x_image_font` 照 x/image 的度量慣例，點陣化用 Cargo.lock 已鎖定的
//! `swash`／zeno）；單元測試可以注入固定寬度的量測器。沒有字型（`sf` 是 `None`）照 Go 回 0；有字型卻沒有
//! 量測器只會出現在手動組出的 `Font`，`face_for` 直接 panic，不以猜測的寬度推進分頁（失敗即關閉）。Ebiten
//! `text` 的排版（字形影像快取鍵、整數位置、`\n` 換行）照 v2.6.6 `text/text.go` 寫在本檔末尾。
//! HD 區域字型（`loadHDRegionalFonts`、`hdFonts`）在 R6。

use super::*;
use crate::go_x_image_font::{
    AlphaImage, Rect26_6, XImageFace, XImageFont, ceil26_6, draw_string, floor26_6, round26_6,
};
use std::collections::HashMap;
use std::sync::Mutex;

/// Rust 新增：外殼提供的字型量測（Go `opentype.NewFace` 的 ascent 與 `text.BoundString` 的寬）。
pub(crate) trait TextMeasure: Send + Sync {
    /// Go `opentype.NewFace(sf, Size: px, DPI: 72, HintingFull)` 後 `fc.Metrics().Ascent.Round()`；
    /// 建 face 失敗回 `None`（Go `faceFor` 回 `nil, 0`）。
    fn ascent(&self, face: &FontFace) -> Option<f64>;
    /// Go `text.BoundString(face, s)`（像素矩形，原點在基線起點）。
    fn bound_string(&self, face: &FontFace, s: &str) -> Rectangle;
    /// Go `sf.GlyphIndex(&buffer, r)` 成功且不是 0（字型有這個字）。
    fn has_glyph(&self, font: &SfntFont, r: char) -> bool;
    /// Go Ebiten `text.DrawWithOptions(dst, s, face, op)`（R5 r5f）。只量寬的量測器不必實作。
    fn draw_text(&self, face: &FontFace, dst: &mut Picture, s: &str, op: &DrawImageOptions) {
        panic!("這個 TextMeasure 只能量測，不能繪字（text.DrawWithOptions）");
    }
    /// Go `font.Drawer{Dst: mask, Src: image.White, Face: face, Dot: dot}.DrawString(s)`（R5 r5f；`dot` 是 26.6）。
    fn draw_alpha(&self, face: &FontFace, mask: &mut AlphaImage, dot: (i32, i32), s: &str) {
        panic!("這個 TextMeasure 只能量測，不能繪字（font.Drawer）");
    }
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
/// sfnt（`0x00010000`、`OTTO`、`true`）。完整表格解析在 `XImageFont::parse`。
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
/// Go 對 TTC 取第 0 個字型；Rust 的 `SfntFont` 保存整份檔案，`XImageFont` 照 Go 取第 0 個。Go 另外依
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
    // Go `sfnt.ParseCollection`／`sfnt.Parse` 失敗時回 nil；量測器的解析對應這一步。
    let parsed = Arc::new(XImageFont::parse(Arc::new(data.clone()))?);
    let measure = TextMeasurer(Some(Arc::new(XImageMeasure::new(parsed))));
    Some(Font { sf: Some(Arc::new(SfntFont { data })), base: size, measure, ..Default::default() })
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
        // `load_font_sized` 讀到字型就注入 `XImageMeasure`；只有手動組出、帶 `sf` 卻沒有量測器的 `Font` 會停在這裡
        // （失敗即關閉，不以猜測的 ascent 繼續）。
        let measure = self.measure.0.clone().expect("font.go:faceFor：有字型（sf）卻沒有注入 TextMeasure");
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
        // `face_for` 回了 face 就表示量測器已注入。
        let measure = self.measure.0.as_ref().expect("font.go:Width：face_for 成功時已注入 TextMeasure");
        rect_dx(&measure.bound_string(&face, s)) as f64
    }
    /// LineHeight 一行高(像素,含行距)。
    ///
    /// Go：`font.go:287`，`func (f *Font) LineHeight(scale float64) float64`
    pub(crate) fn line_height(&self, scale: f64) -> f64 {
        self.base * 1.3 * scale
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

/// runeBreakable 判斷這個字元自己就能當一個斷行單位。中日韓文字與全形標點
/// 每個字都可以斷，拉丁字母與數字不行——那會把英文單字切成兩半。
///
/// Go：`font.go:214`，`func runeBreakable(r rune) bool`
pub(crate) fn rune_breakable(r: i32) -> bool {
    use crate::go_unicode::{HAN, HANGUL, HIRAGANA, KATAKANA, is_in};
    if is_in(r, &[HAN, HIRAGANA, KATAKANA, HANGUL]) {
        return true;
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

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Font {
    /// Draw 畫文字(支援 \n)。scale 相對 base 尺寸;內部用整數尺寸 face 直接 rasterize(銳利)。
    ///
    /// Go：`font.go:175`，`func (f *Font) Draw(dst *ebiten.Image, s string, x, y, scale float64, clr color.RGBA) float64`
    ///
    /// `f.hd != nil && f.hd.scene == dst`（HD 文字框）在 R6；R5 沒有建立 `hd` 的路徑（`beginModernHDText` 在
    /// 忠實模式只清掉 `modernHDText`），`hd` 有值時一律停在 `todo!`。
    pub(crate) fn draw(&mut self, dst: &mut Picture, s: &str, x: f64, y: f64, scale: f64, clr: ColorRgba) -> f64 {
        if self.hd.is_some() {
            todo!("r6:font.go:Draw HD 文字框（f.hd.runs）")
        }
        let px = (self.base * self.effective_scale(scale)).round() as i64;
        let (face, asc) = self.face_for(px);
        let Some(face) = face else {
            return y;
        };
        let mut op = DrawImageOptions::default();
        op.geo_m.translate(x.round(), y.round() + asc); // y=頂部 → baseline;座標取整對齊像素格
        op.color_scale.scale_with_color(clr);
        let measure = self.measure.0.clone().expect("face_for 成功時已注入 TextMeasure");
        measure.draw_text(&face, dst, s, &op);
        y
    }
}

// ---- Rust 新增：x/image 度量＋Ebiten text 排版（R5 r5f） ----

/// [`TextMeasure`] 的正式實作：x/image 的度量（`crate::go_x_image_font`）加上 Ebiten v2.6.6 `text` 套件的
/// 排版。字形影像以 Ebiten 的快取鍵（字、x 方向的次像素偏移）快取在每個像素尺寸底下；Ebiten 依存取時間
/// 淘汰舊字形，這裡不淘汰（只影響記憶體，不影響像素）。
pub(crate) struct XImageMeasure {
    font: Arc<XImageFont>,
    glyphs: Mutex<HashMap<(i32, char, i32), Option<Arc<Picture>>>>,
}

impl XImageMeasure {
    pub(crate) fn new(font: Arc<XImageFont>) -> XImageMeasure {
        XImageMeasure { font, glyphs: Mutex::new(HashMap::new()) }
    }

    pub(crate) fn font(&self) -> &XImageFont {
        &self.font
    }

    fn face(&self, face: &FontFace) -> XImageFace<'_> {
        XImageFace::new(&self.font, face.size, face.dpi)
    }

    /// Ebiten `text/text.go` getGlyphImage：外框寬高取 ceil，起點有小數時各加 1；以 `font.Drawer` 把單一字
    /// 畫進 RGBA（白色來源，所以四個通道都是覆蓋率），再 `NewImageFromImage`（RGBA 原值）。
    fn glyph_image(&self, face: &XImageFace<'_>, r: char, offset: (i32, i32)) -> Option<Arc<Picture>> {
        let key = (face.scale, r, offset.0);
        if let Some(hit) = self.glyphs.lock().expect("字形影像快取").get(&key) {
            return hit.clone();
        }
        let (b, _, _) = face.glyph_bounds(r);
        let (mut w, mut h) = (ceil26_6(b.max_x - b.min_x), ceil26_6(b.max_y - b.min_y));
        let image = if w == 0 || h == 0 {
            None
        } else {
            if b.min_x & 63 != 0 {
                w += 1;
            }
            if b.min_y & 63 != 0 {
                h += 1;
            }
            let mut alpha = AlphaImage::new(w, h);
            draw_string(face, &mut alpha, (-b.min_x + offset.0, -b.min_y + offset.1), &r.to_string());
            let mut pic = Picture::new_image(w, h);
            for (i, a) in alpha.pix.iter().enumerate() {
                pic.pix[i * 4..i * 4 + 4].copy_from_slice(&[*a; 4]);
            }
            Some(Arc::new(pic))
        };
        self.glyphs.lock().expect("字形影像快取").insert(key, image.clone());
        image
    }
}

/// Ebiten `text/text.go` adjustOffsetGranularity：x 方向的次像素偏移以 1/4 像素為單位。
fn adjust_offset_granularity(x: i32) -> i32 {
    x / (1 << 4) * (1 << 4)
}

/// Ebiten `text/text.go` fixed26_6ToFloat64。
fn fixed26_6_to_float64(x: i32) -> f64 {
    f64::from(x >> 6) + f64::from(x & ((1 << 6) - 1)) / f64::from(1 << 6)
}

impl TextMeasure for XImageMeasure {
    fn ascent(&self, face: &FontFace) -> Option<f64> {
        // Go `opentype.NewFace` 不會失敗；`fc.Metrics().Ascent.Round()`。
        Some(round26_6(self.face(face).metrics().ascent) as f64)
    }

    /// Ebiten v2.6.6 `text.BoundString`。
    fn bound_string(&self, face: &FontFace, s: &str) -> Rectangle {
        let fc = self.face(face);
        let face_height = fc.metrics().height;
        let (mut fx, mut fy) = (0i32, 0i32);
        let mut prev: Option<char> = None;
        let mut bounds = Rect26_6::default();
        for r in s.chars() {
            if let Some(p) = prev {
                fx += fc.kern(p, r);
            }
            if r == '\n' {
                fx = 0;
                fy += face_height;
                prev = None;
                continue;
            }
            let (mut b, a, _) = fc.glyph_bounds(r);
            b.min_x += fx;
            b.max_x += fx;
            b.min_y += fy;
            b.max_y += fy;
            bounds = bounds.union(&b);
            fx += a;
            prev = Some(r);
        }
        rect(floor26_6(bounds.min_x), floor26_6(bounds.min_y), ceil26_6(bounds.max_x), ceil26_6(bounds.max_y))
    }

    fn has_glyph(&self, _font: &SfntFont, r: char) -> bool {
        self.font.glyph_index(r) != 0
    }

    /// Ebiten v2.6.6 `text.DrawWithOptions`：每個字畫在整數位置（字形影像本身帶次像素偏移），`op` 的
    /// `GeoM` 接在字形平移之後（`drawGlyph` 的 `Concat`）。子集只收純平移的 `op.GeoM`。
    fn draw_text(&self, face: &FontFace, dst: &mut Picture, s: &str, op: &DrawImageOptions) {
        let g = &op.geo_m;
        assert!(
            g.element(0, 0) == 1.0 && g.element(0, 1) == 0.0 && g.element(1, 0) == 0.0 && g.element(1, 1) == 1.0,
            "text.DrawWithOptions：子集只收純平移的 GeoM"
        );
        let fc = self.face(face);
        let face_height = fc.metrics().height;
        let (mut dx, mut dy) = (0i32, 0i32);
        let mut prev: Option<char> = None;
        for r in s.chars() {
            if let Some(p) = prev {
                dx += fc.kern(p, r);
            }
            if r == '\n' {
                dx = 0;
                dy += face_height;
                prev = None;
                continue;
            }
            let (b, a, _) = fc.glyph_bounds(r);
            let offset = ((adjust_offset_granularity(dx) + b.min_x) & ((1 << 6) - 1), b.min_y & ((1 << 6) - 1));
            if let Some(img) = self.glyph_image(&fc, r, offset) {
                let mut op2 = DrawImageOptions { color_scale: op.color_scale, ..Default::default() };
                op2.geo_m.translate(
                    fixed26_6_to_float64(dx + b.min_x - offset.0),
                    fixed26_6_to_float64(dy + b.min_y - offset.1),
                );
                op2.geo_m.translate(g.element(0, 2), g.element(1, 2));
                dst.draw_image(&img, &op2);
            }
            dx += a;
            prev = Some(r);
        }
    }

    fn draw_alpha(&self, face: &FontFace, mask: &mut AlphaImage, dot: (i32, i32), s: &str) {
        draw_string(&self.face(face), mask, dot, s);
    }
}
