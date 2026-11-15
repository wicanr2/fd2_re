//! `golang.org/x/image` v0.12.0 的字型量測慣例（`font/sfnt`、`font/opentype`、`font.Drawer`），以 Cargo.lock
//! 已鎖定的 `skrifa` 0.44.0（表格、cmap、字形外框；`skrifa::raw` 即 read-fonts 0.41.0）與 `swash` 0.2.10
//! 重新匯出的 `zeno`（點陣化）重做。R5 r5f（2026-11-15）新增，沒有對應的 `cmd/fd2` Go 檔。
//!
//! # 為什麼要照 x/image 的慣例
//!
//! 121 定案 TTF 只比文字位置與外框：凍結 Go 版的文字位置、`text.BoundString` 外框與斷行寬度都由 x/image
//! 的數值慣例決定，換成字型套件自己的度量會讓外框與斷行位置不同。這裡只重做那些慣例，**點陣化不照 Go**
//! （Go 用 `golang.org/x/image/vector`，這裡用 zeno）：反鋸齒覆蓋率不同，差異寫在 `fd2-game/PORTING.md`
//! 的 KNOWN_DIFFERENCES `ttf-raster-coverage`。
//!
//! | Go（x/image v0.12.0） | 這裡 |
//! |---|---|
//! | `sfnt.ParseCollection`＋`Font(0)`；失敗再 `sfnt.Parse` | `XImageFont::parse`（`FileRef` 的單一字型或集合第 0 個） |
//! | `opentype.NewFace`：`scale = Int26_6(0.5 + size*dpi*64/72)` | `XImageFace::new` |
//! | `sfnt.scale`：乘 ppem 後除以 unitsPerEm，四捨五入到 1/64 像素（遠離零） | `scale_units` |
//! | `Font.Metrics`（hhea ascender／descender／lineGap；HintingFull 向上取整到整像素） | `XImageFace::metrics` |
//! | `Font.GlyphAdvance`（hmtx；HintingFull 四捨五入到整像素） | `XImageFace::glyph_advance` |
//! | `Font.GlyphBounds`：外框**所有控制點**縮放後的極值，不是曲線的緊外框 | `XImageFace::glyph_bounds` |
//! | `Font.LoadGlyph`：字形座標乘 ppem／unitsPerEm 取整到 1/64，Y 軸翻成向下 | `XImageFont::segments` |
//! | `opentype.Face.Kern`：傳給 `sfnt.Kern` 的 ppem 是 `unitsPerEm`（Go 的寫法），結果等於字型單位原值，再以 HintingFull 四捨五入 | `XImageFace::kern` |
//! | `sfnt` 的 GPOS kern：`latn`（沒有再 `DFLT`）預設語系的 `kern` 功能，PairPos 格式 1／2、ValueFormat 只收 X_ADVANCE，略過 markFilteringSet；第一個認得這一對的子表為準 | `parse_gpos_kern`、`KernSubtable` |
//! | `opentype.Face.Glyph`：`dr` 以外框極值加 dot 取 floor／ceil，偏移 `bias = dot - dr.Min` | `XImageFace::glyph`（像素以 zeno 非零規則填滿） |
//! | `font.Drawer.DrawString`（`Src: image.White`、`draw.Over`） | `draw_string` |
//!
//! 外框來自 skrifa 的未縮放 CFF／glyf 外框（字型單位）。x/image 的 Type 2 解譯器把 16.16 小數運算元取整；
//! 字型單位出現小數時兩者會不同，`XImageFont::fractional_points` 記錄出現次數供測試檢查。
//!
//! Go 版 x/image 的授權（本模組照它的數值慣例重做，沒有逐行搬移程式碼）：
// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

use std::collections::HashMap;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};

use skrifa::MetadataProvider;
use skrifa::instance::{LocationRef, Size};
use skrifa::outline::{DrawSettings, OutlinePen};
use skrifa::raw::tables::gpos::{ExtensionSubtable, PairPos, PositionLookup, ValueFormat};
use skrifa::raw::tables::layout::{ClassDef, CoverageTable, ScriptList};
use skrifa::raw::types::Tag;
use skrifa::raw::{FileRef, FontRef, ReadError, TableProvider};
use skrifa::{GlyphId, GlyphId16};

/// Go `fixed.Int26_6`。
pub(crate) type Int26_6 = i32;

/// Go `fixed.Int26_6.Floor()`。
pub(crate) fn floor26_6(x: Int26_6) -> i64 {
    i64::from(x >> 6)
}

/// Go `fixed.Int26_6.Ceil()`。
pub(crate) fn ceil26_6(x: Int26_6) -> i64 {
    i64::from((x + 0x3f) >> 6)
}

/// Go `fixed.Int26_6.Round()`。
pub(crate) fn round26_6(x: Int26_6) -> i64 {
    i64::from((x + 0x20) >> 6)
}

/// Go `fixed.Rectangle26_6`。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct Rect26_6 {
    pub(crate) min_x: Int26_6,
    pub(crate) min_y: Int26_6,
    pub(crate) max_x: Int26_6,
    pub(crate) max_y: Int26_6,
}

impl Rect26_6 {
    /// Go `fixed.Rectangle26_6.Empty()`。
    pub(crate) fn empty(&self) -> bool {
        self.min_x >= self.max_x || self.min_y >= self.max_y
    }

    /// Go `fixed.Rectangle26_6.Union()`：空矩形不參與。
    pub(crate) fn union(&self, s: &Rect26_6) -> Rect26_6 {
        if self.empty() {
            return *s;
        }
        if s.empty() {
            return *self;
        }
        Rect26_6 {
            min_x: self.min_x.min(s.min_x),
            min_y: self.min_y.min(s.min_y),
            max_x: self.max_x.max(s.max_x),
            max_y: self.max_y.max(s.max_y),
        }
    }
}

/// Go `sfnt.scale`：`x` 是字型單位乘上 ppem（26.6），除以 unitsPerEm 並四捨五入（遠離零）。
fn scale_units(x: Int26_6, units_per_em: i32) -> Int26_6 {
    let x = if x >= 0 { x + units_per_em / 2 } else { x - units_per_em / 2 };
    x / units_per_em
}

/// Go `sfnt.SegmentOp`（變體名照 Go 的 `SegmentOpMoveTo` 等）。
#[allow(clippy::enum_variant_names)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum SegmentOp {
    MoveTo,
    LineTo,
    QuadTo,
    CubeTo,
}

/// Go `sfnt.Segment`：`args` 前 1、2、3 個點有效（依 `op`）。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct Segment {
    pub(crate) op: SegmentOp,
    pub(crate) args: [(i32, i32); 3],
}

impl Segment {
    fn points(&self) -> &[(i32, i32)] {
        match self.op {
            SegmentOp::MoveTo | SegmentOp::LineTo => &self.args[..1],
            SegmentOp::QuadTo => &self.args[..2],
            SegmentOp::CubeTo => &self.args[..3],
        }
    }
}

/// Go `sfnt.Segments.Bounds()`：所有控制點的極值；沒有線段時是零矩形。
pub(crate) fn segments_bounds(segments: &[Segment]) -> Rect26_6 {
    if segments.is_empty() {
        return Rect26_6::default();
    }
    let mut b = Rect26_6 { min_x: i32::MAX, min_y: i32::MAX, max_x: i32::MIN, max_y: i32::MIN };
    for seg in segments {
        for &(x, y) in seg.points() {
            b.max_x = b.max_x.max(x);
            b.min_x = b.min_x.min(x);
            b.max_y = b.max_y.max(y);
            b.min_y = b.min_y.min(y);
        }
    }
    b
}

/// 一個 GPOS PairPos 子表（Go `sfnt.kernFunc`）。
#[derive(Debug, Clone)]
enum KernSubtable {
    /// 格式 1：涵蓋索引 → 依第二個字形遞增排列的 `(secondGlyph, xAdvance)`。
    Glyph { coverage: HashMap<u16, usize>, pair_sets: Vec<Vec<(u16, i16)>> },
    /// 格式 2：涵蓋集合、兩個類別表（沒列出的是類別 0）與 `class1 × class2` 的 xAdvance。
    Class {
        coverage: HashMap<u16, usize>,
        class_def1: HashMap<u16, u16>,
        class_def2: HashMap<u16, u16>,
        class2_count: usize,
        values: Vec<i16>,
    },
}

impl KernSubtable {
    /// Go `kernFunc(a, b)`：`None` 是 `ErrNotFound`。
    fn get(&self, a: u16, b: u16) -> Option<i16> {
        match self {
            KernSubtable::Glyph { coverage, pair_sets } => {
                let idx = *coverage.get(&a)?;
                let set = pair_sets.get(idx)?;
                for &(second, value) in set {
                    if second == b {
                        return Some(value);
                    }
                    if second > b {
                        return None;
                    }
                }
                None
            }
            KernSubtable::Class { coverage, class_def1, class_def2, class2_count, values } => {
                // Go 先查涵蓋，避免選到預設類別 0。
                coverage.get(&a)?;
                let idxa = usize::from(class_def1.get(&a).copied().unwrap_or(0));
                let idxb = usize::from(class_def2.get(&b).copied().unwrap_or(0));
                values.get(idxb + idxa * class2_count).copied()
            }
        }
    }
}

fn coverage_map(coverage: &CoverageTable<'_>) -> HashMap<u16, usize> {
    coverage.iter().enumerate().map(|(i, g)| (g.to_u16(), i)).collect()
}

fn class_map(class_def: &ClassDef<'_>) -> HashMap<u16, u16> {
    class_def.iter().map(|(g, c)| (g.to_u16(), c)).collect()
}

/// Go `parseGPOSScriptFeatures`：只取該 script 的預設語系。
fn default_lang_features(scripts: &ScriptList<'_>, tag: Tag) -> Result<Vec<usize>, ReadError> {
    let Some(record) = scripts.script_records().iter().find(|r| r.script_tag() == tag) else {
        return Ok(Vec::new());
    };
    let script = record.script(scripts.offset_data())?;
    let Some(lang_sys) = script.default_lang_sys() else {
        return Ok(Vec::new());
    };
    Ok(lang_sys?.feature_indices().iter().map(|i| usize::from(i.get())).collect())
}

fn push_pair_pos(out: &mut Vec<KernSubtable>, pair: PairPos<'_>) -> Result<(), ReadError> {
    let x_advance_only =
        |vf1: ValueFormat, vf2: ValueFormat| vf1 == ValueFormat::X_ADVANCE && vf2 == ValueFormat::empty();
    match pair {
        PairPos::Format1(t) => {
            let coverage = coverage_map(&t.coverage()?);
            if !x_advance_only(t.value_format1(), t.value_format2()) {
                return Ok(());
            }
            let mut pair_sets = Vec::new();
            for set in t.pair_sets().iter() {
                let mut pairs = Vec::new();
                for record in set?.pair_value_records().iter() {
                    let record = record?;
                    pairs.push((record.second_glyph().to_u16(), record.value_record1().x_advance().unwrap_or(0)));
                }
                pair_sets.push(pairs);
            }
            out.push(KernSubtable::Glyph { coverage, pair_sets });
        }
        PairPos::Format2(t) => {
            let coverage = coverage_map(&t.coverage()?);
            if !x_advance_only(t.value_format1(), t.value_format2()) {
                return Ok(());
            }
            let class2_count = usize::from(t.class2_count());
            let mut values = Vec::new();
            for class1 in t.class1_records().iter() {
                for class2 in class1?.class2_records().iter() {
                    values.push(class2?.value_record1().x_advance().unwrap_or(0));
                }
            }
            out.push(KernSubtable::Class {
                coverage,
                class_def1: class_map(&t.class_def1()?),
                class_def2: class_map(&t.class_def2()?),
                class2_count,
                values,
            });
        }
    }
    Ok(())
}

/// Go `Font.parseGPOSKern`。沒有 GPOS 時回空表；表格損壞時 Go 整個字型解析失敗，這裡回 `Err`。
fn parse_gpos_kern(font: &FontRef<'_>) -> Result<Vec<KernSubtable>, ReadError> {
    if font.table_data(Tag::new(b"GPOS")).is_none() {
        return Ok(Vec::new());
    }
    let gpos = font.gpos()?;
    let scripts = gpos.script_list()?;
    let mut feature_idxs = default_lang_features(&scripts, Tag::new(b"latn"))?;
    if feature_idxs.is_empty() {
        feature_idxs = default_lang_features(&scripts, Tag::new(b"DFLT"))?;
        if feature_idxs.is_empty() {
            return Ok(Vec::new());
        }
    }
    let features = gpos.feature_list()?;
    let mut lookup_idxs = Vec::new();
    for fidx in feature_idxs {
        let Some(record) = features.feature_records().get(fidx) else {
            return Err(ReadError::OutOfBounds);
        };
        if record.feature_tag() != Tag::new(b"kern") {
            continue;
        }
        let feature = record.feature(features.offset_data())?;
        lookup_idxs.extend(feature.lookup_list_indices().iter().map(|i| usize::from(i.get())));
    }
    let lookups = gpos.lookup_list()?;
    let mut out = Vec::new();
    'lookups: for n in lookup_idxs {
        let mut subtables = Vec::new();
        let flag = match lookups.lookups().get(n)? {
            PositionLookup::Pair(lookup) => {
                for st in lookup.subtables().iter() {
                    subtables.push(st?);
                }
                lookup.lookup_flag().to_bits()
            }
            PositionLookup::Extension(lookup) => {
                // Go：延伸子表的格式必須是 1（否則整個解析失敗）；延伸的 lookup 型別不是 2 就略過整個 lookup。
                for st in lookup.subtables().iter() {
                    match st? {
                        ExtensionSubtable::Pair(ext) => subtables.push(ext.extension()?),
                        _ => continue 'lookups,
                    }
                }
                lookup.lookup_flag().to_bits()
            }
            _ => continue,
        };
        if flag & 0x0010 != 0 {
            // useMarkFilteringSet：Go 不支援，略過。
            continue;
        }
        for pair in subtables {
            push_pair_pos(&mut out, pair)?;
        }
    }
    Ok(out)
}

/// 字型單位的外框（未縮放、Y 軸向上），照 x/image Type 2 解譯器的線段形狀記錄。
#[derive(Default)]
struct UnitPen {
    segments: Vec<Segment>,
    fractional: u64,
}

impl UnitPen {
    fn unit(&mut self, v: f32) -> i32 {
        if v.fract() != 0.0 {
            self.fractional += 1;
        }
        v.round() as i32
    }

    fn push(&mut self, op: SegmentOp, pts: &[f32]) {
        let mut args = [(0, 0); 3];
        for (i, pair) in pts.chunks(2).enumerate() {
            args[i] = (self.unit(pair[0]), self.unit(pair[1]));
        }
        self.segments.push(Segment { op, args });
    }
}

impl OutlinePen for UnitPen {
    fn move_to(&mut self, x: f32, y: f32) {
        self.push(SegmentOp::MoveTo, &[x, y]);
    }
    fn line_to(&mut self, x: f32, y: f32) {
        self.push(SegmentOp::LineTo, &[x, y]);
    }
    fn quad_to(&mut self, cx0: f32, cy0: f32, x: f32, y: f32) {
        self.push(SegmentOp::QuadTo, &[cx0, cy0, x, y]);
    }
    fn curve_to(&mut self, cx0: f32, cy0: f32, cx1: f32, cy1: f32, x: f32, y: f32) {
        self.push(SegmentOp::CubeTo, &[cx0, cy0, cx1, cy1, x, y]);
    }
    fn close(&mut self) {
        // 填滿時每個子路徑都自動封閉；Go 封閉時補的直線終點就是子路徑起點，不改變外框極值。
    }
}

/// Go `*sfnt.Font`（`font/opentype` 的 `Font` 是同一型別）：解析後快取的度量與 kern 子表。
pub(crate) struct XImageFont {
    data: Arc<Vec<u8>>,
    index: u32,
    units_per_em: i32,
    ascent: i32,
    descent: i32,
    line_gap: i32,
    num_glyphs: u32,
    kern: Vec<KernSubtable>,
    /// 每個字形索引的字型單位外框（`None`：Go `LoadGlyph` 會失敗）。
    outlines: Mutex<HashMap<u32, Option<Arc<Vec<Segment>>>>>,
    fractional: AtomicU64,
}

impl std::fmt::Debug for XImageFont {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "XImageFont(index={}, upem={}, glyphs={})", self.index, self.units_per_em, self.num_glyphs)
    }
}

impl XImageFont {
    /// Go `sfnt.ParseCollection(data)` 後 `Font(0)`，集合解析失敗時 `sfnt.Parse(data)`（`font.go:loadFontSized`）。
    pub(crate) fn parse(data: Arc<Vec<u8>>) -> Option<XImageFont> {
        let (index, font) = match FileRef::new(&data).ok()? {
            FileRef::Font(font) => (0, font),
            FileRef::Collection(collection) => (0, collection.get(0).ok()?),
        };
        let head = font.head().ok()?;
        let hhea = font.hhea().ok()?;
        let maxp = font.maxp().ok()?;
        let units_per_em = i32::from(head.units_per_em());
        let kern = parse_gpos_kern(&font).ok()?;
        let (ascent, descent, line_gap) = (
            i32::from(hhea.ascender().to_i16()),
            i32::from(hhea.descender().to_i16()),
            i32::from(hhea.line_gap().to_i16()),
        );
        let num_glyphs = u32::from(maxp.num_glyphs());
        if units_per_em == 0 || font.hmtx().is_err() {
            return None;
        }
        Some(XImageFont {
            data,
            index,
            units_per_em,
            ascent,
            descent,
            line_gap,
            num_glyphs,
            kern,
            outlines: Mutex::new(HashMap::new()),
            fractional: AtomicU64::new(0),
        })
    }

    fn font_ref(&self) -> FontRef<'_> {
        match FileRef::new(&self.data).expect("parse 已驗證") {
            FileRef::Font(font) => font,
            FileRef::Collection(collection) => collection.get(self.index).expect("parse 已驗證"),
        }
    }

    /// Go `Font.UnitsPerEm()`。
    pub(crate) fn units_per_em(&self) -> i32 {
        self.units_per_em
    }

    /// 字型單位外框出現小數座標的次數（x/image 會在解譯時取整，見模組文件）。
    pub(crate) fn fractional_points(&self) -> u64 {
        self.fractional.load(Ordering::Relaxed)
    }

    /// Go `Font.GlyphIndex(b, r)`：沒有對應的字是 0。
    pub(crate) fn glyph_index(&self, r: char) -> u32 {
        self.font_ref().charmap().map(r).map_or(0, |g| g.to_u32())
    }

    /// Go `Font.NumGlyphs()`。
    pub(crate) fn num_glyphs(&self) -> u32 {
        self.num_glyphs
    }

    /// 字型單位外框；`None` 對應 Go `LoadGlyph` 回錯誤。
    fn unit_segments(&self, x: u32) -> Option<Arc<Vec<Segment>>> {
        let mut cache = self.outlines.lock().expect("字形快取");
        if let Some(hit) = cache.get(&x) {
            return hit.clone();
        }
        let font = self.font_ref();
        let loaded = font.outline_glyphs().get(GlyphId::new(x)).and_then(|glyph| {
            let mut pen = UnitPen::default();
            glyph.draw(DrawSettings::unhinted(Size::unscaled(), LocationRef::default()), &mut pen).ok()?;
            self.fractional.fetch_add(pen.fractional, Ordering::Relaxed);
            Some(Arc::new(pen.segments))
        });
        cache.insert(x, loaded.clone());
        loaded
    }

    /// Go `Font.LoadGlyph(b, x, ppem, nil)`：縮放到 26.6、Y 軸翻成向下。
    pub(crate) fn segments(&self, x: u32, ppem: Int26_6) -> Option<Vec<Segment>> {
        if x >= self.num_glyphs {
            return None;
        }
        let units = self.unit_segments(x)?;
        let upem = self.units_per_em;
        Some(
            units
                .iter()
                .map(|seg| {
                    let mut out = *seg;
                    for a in out.args.iter_mut() {
                        *a = (scale_units(a.0 * ppem, upem), -scale_units(a.1 * ppem, upem));
                    }
                    out
                })
                .collect(),
        )
    }

    /// hmtx 的前進寬度（字型單位；超出 numberOfHMetrics 的取最後一筆，同 Go）。
    fn advance_units(&self, x: u32) -> Option<i32> {
        let hmtx = self.font_ref().hmtx().ok()?;
        hmtx.advance(GlyphId::new(x)).map(i32::from)
    }
}

/// Go `opentype.NewFace(f, &FaceOptions{Size, DPI, Hinting: font.HintingFull})`（`font.go:faceFor` 只用 HintingFull）。
#[derive(Debug, Clone, Copy)]
pub(crate) struct XImageFace<'a> {
    pub(crate) font: &'a XImageFont,
    /// Go `Face.scale`（26.6 的 ppem）。
    pub(crate) scale: Int26_6,
}

/// Go `font.Metrics` 的三個欄位。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct XImageMetrics {
    pub(crate) height: Int26_6,
    pub(crate) ascent: Int26_6,
    pub(crate) descent: Int26_6,
}

/// Go `opentype.Face.Glyph` 的結果：`dr` 是目標座標，`mask` 是 `dr` 大小的 alpha。
#[derive(Debug, Clone)]
pub(crate) struct GlyphMask {
    pub(crate) dr: (i64, i64, i64, i64),
    pub(crate) mask: Vec<u8>,
}

impl<'a> XImageFace<'a> {
    pub(crate) fn new(font: &'a XImageFont, size: f64, dpi: f64) -> XImageFace<'a> {
        XImageFace { font, scale: (0.5 + size * dpi * 64.0 / 72.0) as Int26_6 }
    }

    /// Go `Face.Metrics()`（HintingFull：向上取整到整像素）。
    pub(crate) fn metrics(&self) -> XImageMetrics {
        let f = self.font;
        let (ppem, upem) = (self.scale, f.units_per_em);
        let full = |v: Int26_6| (v + 63) & !63;
        XImageMetrics {
            height: full(scale_units((f.ascent - f.descent + f.line_gap) * ppem, upem)),
            ascent: full(scale_units(f.ascent * ppem, upem)),
            descent: full(-scale_units(f.descent * ppem, upem)),
        }
    }

    /// Go `Font.GlyphAdvance`（HintingFull：四捨五入到整像素）；`None` 是 `ErrNotFound`。
    fn advance_of(&self, x: u32) -> Option<Int26_6> {
        if x >= self.font.num_glyphs {
            return None;
        }
        let adv = scale_units(self.font.advance_units(x)? * self.scale, self.font.units_per_em);
        Some((adv + 32) & !63)
    }

    /// Go `Face.GlyphBounds(r)`：`(bounds, advance, ok)`；`ok` 在字型沒有這個字（索引 0）時也是 false，
    /// 但仍回傳 .notdef 的外框與寬度。
    pub(crate) fn glyph_bounds(&self, r: char) -> (Rect26_6, Int26_6, bool) {
        let x = self.font.glyph_index(r);
        let Some(advance) = self.advance_of(x) else {
            return (Rect26_6::default(), 0, false);
        };
        match self.font.segments(x, self.scale) {
            Some(segments) => (segments_bounds(&segments), advance, x != 0),
            None => (Rect26_6::default(), 0, false),
        }
    }

    /// Go `Face.Kern(r0, r1)`：ppem 傳 unitsPerEm，再以 HintingFull 四捨五入到整像素。
    pub(crate) fn kern(&self, r0: char, r1: char) -> Int26_6 {
        let x0 = self.font.glyph_index(r0);
        let x1 = self.font.glyph_index(r1);
        let (Ok(a), Ok(b)) = (u16::try_from(x0), u16::try_from(x1)) else {
            return 0;
        };
        for kf in &self.font.kern {
            if let Some(adv) = kf.get(a, b) {
                let upem = self.font.units_per_em;
                let kern = scale_units(i32::from(adv) * upem, upem);
                return (kern + 32) & !63;
            }
        }
        0
    }

    /// Go `Face.Glyph(dot, r)`：`(dr, mask, advance)`；Go 回 `ok == false` 且 `dr` 為零時是 `None`。
    /// 線段照 Go 加上 `bias` 後以 `float32(v)/64` 交給點陣化器；點陣化是 zeno（非零規則），不是 Go 的 vector。
    pub(crate) fn glyph(&self, dot: (Int26_6, Int26_6), r: char) -> Option<(GlyphMask, Int26_6)> {
        let x = self.font.glyph_index(r);
        let advance = self.advance_of(x)?;
        let segments = self.font.segments(x, self.scale)?;
        let b = segments_bounds(&segments);
        let (min_x, min_y) = (floor26_6(b.min_x + dot.0), floor26_6(b.min_y + dot.1));
        let (max_x, max_y) = (ceil26_6(b.max_x + dot.0), ceil26_6(b.max_y + dot.1));
        let (width, height) = (max_x - min_x, max_y - min_y);
        if width < 0 || height < 0 {
            return None;
        }
        let bias_x = dot.0 - (min_x << 6) as Int26_6;
        let bias_y = dot.1 - (min_y << 6) as Int26_6;
        let mut mask = vec![0u8; (width * height) as usize];
        if width > 0 && height > 0 {
            use swash::zeno::{Command, Mask, Point};
            let p = |a: (i32, i32)| Point::new((a.0 + bias_x) as f32 / 64.0, (a.1 + bias_y) as f32 / 64.0);
            let mut path = Vec::with_capacity(segments.len() + 8);
            for seg in &segments {
                match seg.op {
                    SegmentOp::MoveTo => {
                        if !path.is_empty() {
                            path.push(Command::Close);
                        }
                        path.push(Command::MoveTo(p(seg.args[0])));
                    }
                    SegmentOp::LineTo => path.push(Command::LineTo(p(seg.args[0]))),
                    SegmentOp::QuadTo => path.push(Command::QuadTo(p(seg.args[0]), p(seg.args[1]))),
                    SegmentOp::CubeTo => path.push(Command::CurveTo(p(seg.args[0]), p(seg.args[1]), p(seg.args[2]))),
                }
            }
            if !path.is_empty() {
                path.push(Command::Close);
                Mask::new(&path[..]).size(width as u32, height as u32).render_into(&mut mask, None);
            }
        }
        Some((GlyphMask { dr: (min_x, min_y, max_x, max_y), mask }, advance))
    }
}

/// Go `*image.Alpha`（原點在 0,0）。
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct AlphaImage {
    pub(crate) width: i64,
    pub(crate) height: i64,
    pub(crate) pix: Vec<u8>,
}

impl AlphaImage {
    pub(crate) fn new(width: i64, height: i64) -> AlphaImage {
        AlphaImage { width, height, pix: vec![0; (width.max(0) * height.max(0)) as usize] }
    }
}

/// Go `font.Drawer{Dst: dst, Src: image.White, Face: face, Dot: dot}.DrawString(s)`；回傳最後的 dot。
/// 每個字以 `draw.Over` 疊上：`a = (da*(m-ma) + m*ma)/m`（16 位元，`m = 0xffff`），超出 `dst` 的部分裁掉。
pub(crate) fn draw_string(
    face: &XImageFace<'_>,
    dst: &mut AlphaImage,
    mut dot: (Int26_6, Int26_6),
    s: &str,
) -> (Int26_6, Int26_6) {
    let mut prev: Option<char> = None;
    for c in s.chars() {
        if let Some(p) = prev {
            dot.0 += face.kern(p, c);
        }
        if let Some((glyph, advance)) = face.glyph(dot, c) {
            let (x0, y0, x1, y1) = glyph.dr;
            let width = x1 - x0;
            for y in y0.max(0)..y1.min(dst.height) {
                for x in x0.max(0)..x1.min(dst.width) {
                    let ma = u32::from(glyph.mask[((y - y0) * width + (x - x0)) as usize]) * 0x101;
                    let i = (y * dst.width + x) as usize;
                    let da = u32::from(dst.pix[i]) * 0x101;
                    let m = 0xffff;
                    let out = (da * (m - ma) + m * ma) / m;
                    dst.pix[i] = (out >> 8) as u8;
                }
            }
            dot.0 += advance;
        }
        prev = Some(c);
    }
    dot
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fixed_helpers_follow_go() {
        assert_eq!((floor26_6(-1), ceil26_6(-1), round26_6(-1)), (-1, 0, 0));
        assert_eq!((floor26_6(64), ceil26_6(65), round26_6(96)), (1, 2, 2));
        assert_eq!(scale_units(1500, 1000), 2);
        assert_eq!(scale_units(-1500, 1000), -2);
        assert_eq!(scale_units(1499, 1000), 1);
        let a = Rect26_6 { min_x: 0, min_y: 0, max_x: 10, max_y: 10 };
        assert_eq!(Rect26_6::default().union(&a), a);
        assert_eq!(a.union(&Rect26_6 { min_x: 5, min_y: 5, max_x: 5, max_y: 20 }), a);
    }
}
