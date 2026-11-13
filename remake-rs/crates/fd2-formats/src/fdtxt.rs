//! 原版 FDTXT 字表與 FDOTHER #4 的 16×16 點陣字型。
//!
//! 移植自 Go `remake/internal/fdtxt`（`fdtxt.go`、`separated.go`，凍結於 v.1.1.11）。
//! 解碼刻意停在場景控制碼之前：FFxx 控制字原樣交給呼叫端，不把原版的換頁或頭像切換
//! 悄悄轉成一般文字。本套件自己的錯誤（`fdtxt: ...`）與 Go 版逐字相同，以便黃金差分直接比對。
//!
//! Go 慣用寫法的對應（行為以凍結 Go 版為準）：
//!
//! - Go `int` 參數一律用 `i64`：負值與溢位照 Go 的檢查與二補數回繞處理；Go 會因負索引 panic
//!   的寫入，Rust 也以越界索引 panic。
//! - nil 接收者（`*Strings`、`*Font` 為 nil）在 Rust 無法表示，Go 對應的拒收分支不存在。
//! - `Words` 在 Go 對空字串回傳 nil slice，Rust 回傳空 `Vec`；兩者長度都是 0。
//! - `tokens` 的 nil 與空陣列以 `Option<Vec<_>>` 區分（`null`、缺欄為 `None`）。
//! - `WithNativePresenter` 的 Go 函式值改為 `Arc<dyn Fn>`；傳 `None` 等於 Go 傳 nil。
//! - `image.Image.At().RGBA()` 由 [`decode_png_rgba`] 重現：調色盤、tRNS 與 NRGBA 先乘 alpha 的換算
//!   和 Go 相同，所以 alpha 為 0 的白點也讀成 0 位元。
//!
//! 刻意不照做：分離素材由本專案工具產生並以雜湊核對，所以 JSON、PNG 與路徑改用 serde_json、
//! png crate 與 `std::path`，不逐行移植 Go 標準函式庫。格式正確的輸入結果與 Go 相同；Go 標準
//! 函式庫對壞輸入的寬鬆之處（鍵名不分大小寫、重複鍵合併、無效 UTF-8 換成 U+FFFD、`a/../b`
//! 先字面化簡、IDAT 尾端多餘位元組等）不是原版遊戲行為，Rust 版改為拒收或交給作業系統解析。
//! 這類輸入的黃金差分只比對接受或拒收，差異逐項列在 `tests/golden_fdtxt.rs` 的已知差異清單。
//! 包裝的內層錯誤文字（作業系統、serde_json、png crate）也與 Go 不同，只有外層前綴相同。

#[cfg(test)]
mod tests;

use serde::Deserialize;
use std::fmt;
use std::io::Read;
use std::path::Path;
use std::sync::Arc;

pub const GLYPH_WIDTH: usize = 16;
pub const GLYPH_HEIGHT: usize = 16;
pub const GLYPH_BYTES: usize = 32;
pub const STRING_END: u16 = 0xffff;
pub const CONTROL_MIN: u16 = 0xff00;
pub const OPENING_GLYPH: u16 = 557;

const FDTXT_SOURCE_SIZE: i64 = 120502;
const FDTXT_SOURCE_MD5: &str = "fe5c487ce4313485f1da9d48d35b05f9";
const FDTXT_SOURCE_SHA256: &str = "a4555f8a0e61e884b4f504d56a8bdde11672583bbbbc6506281ae10dcdfb1f69";
const FDOTHER_SOURCE_SIZE: i64 = 3382481;
const FDOTHER_SOURCE_MD5: &str = "22f56e5027edc7c766ad34ca4e5aca93";
const FDOTHER_SOURCE_SHA256: &str = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce";
const SEPARATED_FONT_GLYPHS: i64 = 1824;
const SEPARATED_FONT_COLUMNS: i64 = 32;

/// 錯誤種類；`Display` 的 `fdtxt: ...` 部分與 Go 版逐字相同，包裝（`%w`）的內層錯誤由
/// `source()` 取得，內層文字來自 Rust 函式庫。
#[derive(Debug)]
pub enum FdtxtError {
    ShortResource,
    InvalidFirstOffset(usize),
    InvalidOffset(usize),
    StringIndex(i64),
    LogicalIndex(i64),
    FontLength(usize),
    GlyphCoordinate,
    InvalidBlit,
    BlitExceeds,
    InvalidNativeBlit,
    NativeBlitExceeds,
    InvalidSeparatedRequest,
    SeparatedIo {
        resource: i64,
        source: std::io::Error,
    },
    SeparatedJson {
        resource: i64,
        source: serde_json::Error,
    },
    SeparatedMetadata(i64),
    StringIdentity {
        resource: i64,
        index: usize,
    },
    InvalidGlyph {
        resource: i64,
        index: usize,
        token: usize,
    },
    InvalidControl {
        resource: i64,
        index: usize,
        token: usize,
    },
    UnknownKind {
        resource: i64,
        index: usize,
        token: usize,
    },
    Projection {
        resource: i64,
        index: usize,
    },
    FontRootEmpty,
    FontMetadataIo(std::io::Error),
    FontJson(serde_json::Error),
    FontMetadata,
    AtlasOpen(std::io::Error),
    /// 開檔成功但讀取失敗（例如圖集路徑是目錄）；Go 在 png.Decode 途中回報，前綴相同。
    AtlasRead(std::io::Error),
    AtlasDecode(png::DecodingError),
    AtlasGeometry,
    NonBinaryPixel,
}

impl fmt::Display for FdtxtError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        use FdtxtError::*;
        match self {
            ShortResource => f.write_str("fdtxt: resource shorter than first offset"),
            InvalidFirstOffset(first) => write!(f, "fdtxt: invalid first offset {first:#x}"),
            InvalidOffset(i) => write!(f, "fdtxt: invalid offset at string {i}"),
            StringIndex(i) => write!(f, "fdtxt: string index {i} out of range"),
            LogicalIndex(i) => write!(f, "fdtxt: logical utterance index {i} out of range"),
            FontLength(n) => write!(f, "fdtxt: invalid 16x16 font length {n}"),
            GlyphCoordinate => f.write_str("fdtxt: glyph coordinate out of range"),
            InvalidBlit => f.write_str("fdtxt: invalid glyph blit"),
            BlitExceeds => f.write_str("fdtxt: glyph blit exceeds destination"),
            InvalidNativeBlit => f.write_str("fdtxt: invalid native glyph blit"),
            NativeBlitExceeds => f.write_str("fdtxt: native glyph blit exceeds destination"),
            InvalidSeparatedRequest => f.write_str("fdtxt: invalid separated resource request"),
            SeparatedIo { resource, source } => write!(f, "fdtxt: separated resource {resource}: {source}"),
            SeparatedJson { resource, source } => write!(f, "fdtxt: separated resource {resource} JSON: {source}"),
            SeparatedMetadata(r) => write!(f, "fdtxt: separated resource {r} metadata mismatch"),
            StringIdentity { resource, index } => {
                write!(f, "fdtxt: separated resource {resource} string {index} identity mismatch")
            }
            InvalidGlyph { resource, index, token } => {
                write!(f, "fdtxt: resource {resource} string {index} token {token} invalid glyph")
            }
            InvalidControl { resource, index, token } => {
                write!(f, "fdtxt: resource {resource} string {index} token {token} invalid control")
            }
            UnknownKind { resource, index, token } => {
                write!(f, "fdtxt: resource {resource} string {index} token {token} unknown kind")
            }
            Projection { resource, index } => {
                write!(f, "fdtxt: separated resource {resource} string {index} UTF-8 projection mismatch")
            }
            FontRootEmpty => f.write_str("fdtxt: separated font root is empty"),
            FontMetadataIo(source) => write!(f, "fdtxt: separated font metadata: {source}"),
            FontJson(source) => write!(f, "fdtxt: separated font JSON: {source}"),
            FontMetadata => f.write_str("fdtxt: separated font metadata mismatch"),
            AtlasOpen(source) => write!(f, "fdtxt: separated font atlas: {source}"),
            AtlasRead(source) => write!(f, "fdtxt: separated font atlas decode: {source}"),
            AtlasDecode(source) => write!(f, "fdtxt: separated font atlas decode: {source}"),
            AtlasGeometry => f.write_str("fdtxt: separated font atlas geometry mismatch"),
            NonBinaryPixel => f.write_str("fdtxt: separated font atlas contains a non-binary pixel"),
        }
    }
}

impl std::error::Error for FdtxtError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        use FdtxtError::*;
        match self {
            SeparatedIo { source, .. } | FontMetadataIo(source) | AtlasOpen(source) | AtlasRead(source) => Some(source),
            SeparatedJson { source, .. } | FontJson(source) => Some(source),
            AtlasDecode(source) => Some(source),
            _ => None,
        }
    }
}

pub type Result<T> = std::result::Result<T, FdtxtError>;

fn le_u16(data: &[u8], at: usize) -> u16 {
    u16::from_le_bytes([data[at], data[at + 1]])
}

fn be_u16(data: &[u8], at: usize) -> u16 {
    u16::from_be_bytes([data[at], data[at + 1]])
}

// ---------------------------------------------------------------------------
// FDTXT 字表
// ---------------------------------------------------------------------------

/// 位移表格式的 FDTXT 資源。每個字串不含結尾的 0xffff，但保留全部 FFxx 控制字。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Strings {
    words: Vec<Vec<u16>>,
}

/// 指出實體 FDTXT 位移表字串中的一句可見台詞。原生選擇子可以指向這個邏輯序列，
/// 但不可假設它就是實體位移表的索引。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct LogicalLocator {
    pub raw_string_index: usize,
    pub raw_utterance_index: usize,
}

/// 解析位移表：第一個位移同時決定字串數，之後的位移必須為偶數、遞增且不超出資源。
pub fn parse(data: &[u8]) -> Result<Strings> {
    if data.len() < 2 {
        return Err(FdtxtError::ShortResource);
    }
    let first = usize::from(le_u16(data, 0));
    if first == 0 || !first.is_multiple_of(2) || first > data.len() {
        return Err(FdtxtError::InvalidFirstOffset(first));
    }
    let count = first / 2;
    let mut offsets = Vec::with_capacity(count);
    for i in 0..count {
        let offset = usize::from(le_u16(data, i * 2));
        if offset < first || offset > data.len() || !offset.is_multiple_of(2) || (i > 0 && offset < offsets[i - 1]) {
            return Err(FdtxtError::InvalidOffset(i));
        }
        offsets.push(offset);
    }
    let words = offsets
        .iter()
        .enumerate()
        .map(|(i, &start)| {
            let end = offsets.get(i + 1).copied().unwrap_or(data.len());
            // Go 以 `p < end; p += 2` 逐字讀取。最後一個字串延伸到奇數長度的資源結尾、且在結尾字
            // 之前讀到最後一個位元組時，Go 的 `binary.LittleEndian.Uint16` 越界 panic；這裡的
            // `le_u16` 同樣越界 panic，不改成錯誤或截斷（黃金資料以 panic 案例驗證）。
            (start..end).step_by(2).map(|p| le_u16(data, p)).take_while(|&w| w != STRING_END).collect()
        })
        .collect();
    Ok(Strings { words })
}

impl Strings {
    pub fn count(&self) -> usize {
        self.words.len()
    }

    /// 回傳獨立複本，讓可編輯的重製劇本無法改動原版資源證據。
    pub fn words(&self, index: i64) -> Result<Vec<u16>> {
        usize::try_from(index).ok().and_then(|i| self.words.get(i)).cloned().ok_or(FdtxtError::StringIndex(index))
    }

    /// 依與字數對齊劇情匯出器相同的結構規則找台詞：非控制字段落以說話者運算元開頭、第二個字是
    /// 開引號字形時，即一句台詞的開始。FFxx 只當分段，刻意不在這裡重新解讀。
    pub fn locate_logical_utterance(&self, index: i64) -> Result<LogicalLocator> {
        if index < 0 {
            return Err(FdtxtError::LogicalIndex(index));
        }
        let mut seen = 0i64;
        for (raw_string_index, words) in self.words.iter().enumerate() {
            let mut chunk: Vec<u16> = Vec::new();
            let mut local = 0;
            for &word in words.iter().chain(std::iter::once(&STRING_END)) {
                if word < CONTROL_MIN {
                    chunk.push(word);
                    continue;
                }
                if chunk.len() >= 2 && chunk[1] == OPENING_GLYPH {
                    if seen == index {
                        return Ok(LogicalLocator { raw_string_index, raw_utterance_index: local });
                    }
                    seen += 1;
                    local += 1;
                }
                chunk.clear();
            }
        }
        Err(FdtxtError::LogicalIndex(index))
    }
}

// ---------------------------------------------------------------------------
// 16×16 點陣字型
// ---------------------------------------------------------------------------

/// `0x4ea2a` ABI 中可見的顏色部分。來源位元為 1 時寫入前景色；陰影寫在下一列的 x-1 與 x；
/// 背景色不為 0 時才填滿 16×16 字格。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct NativeGlyphStyle {
    pub foreground: u8,
    pub shadow: u8,
    pub background: u8,
}

/// 原生字形的呈現轉接：參數依序為 stride、base、字形索引與樣式；回傳 true 表示已自行繪製字墨。
pub type NativePresenter = Arc<dyn Fn(i64, i64, i64, NativeGlyphStyle) -> bool + Send + Sync>;

/// FDOTHER #4：緊密排列的 16×16、1bpp 字形，每列最高位元在左。
#[derive(Clone)]
pub struct Font {
    data: Arc<[u8]>,
    native_presenter: Option<NativePresenter>,
}

impl fmt::Debug for Font {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("Font")
            .field("glyphs", &self.glyph_count())
            .field("native_presenter", &self.native_presenter.is_some())
            .finish()
    }
}

pub fn parse_font(data: &[u8]) -> Result<Font> {
    if data.is_empty() || !data.len().is_multiple_of(GLYPH_BYTES) {
        return Err(FdtxtError::FontLength(data.len()));
    }
    Ok(Font { data: Arc::from(data), native_presenter: None })
}

impl Font {
    /// 回傳只共用不可變字型位元組的呈現轉接。呈現函式回傳 true 時，在一般檢查與背景填色之後
    /// 取代字墨；回傳 false 則保留原生字形與陰影。疊圖順序由呼叫端負責：之後若有不透明面板
    /// 蓋住，呼叫端必須丟棄描述，這個轉接不追蹤目的地身分。
    pub fn with_native_presenter(&self, present: Option<NativePresenter>) -> Font {
        Font { data: Arc::clone(&self.data), native_presenter: present }
    }

    pub fn glyph_count(&self) -> usize {
        self.data.len() / GLYPH_BYTES
    }

    fn row(&self, index: i64, y: i64) -> u16 {
        be_u16(&self.data, index as usize * GLYPH_BYTES + y as usize * 2)
    }

    fn valid_index(&self, index: i64) -> bool {
        index >= 0 && (index as u64) < self.glyph_count() as u64
    }

    /// 回報原始 1bpp 位元，不套調色盤也不解讀 FFxx 控制字。
    pub fn glyph_bit(&self, index: i64, x: i64, y: i64) -> Result<bool> {
        if !self.valid_index(index) || !(0..GLYPH_WIDTH as i64).contains(&x) || !(0..GLYPH_HEIGHT as i64).contains(&y) {
            return Err(FdtxtError::GlyphCoordinate);
        }
        Ok(self.row(index, y) & (0x8000u16 >> x) != 0)
    }

    /// 只把來源為 1 的位元寫進索引色目的地，0 位元透明，與原版 1bpp 遮罩語意相同。調色盤由
    /// 尚未還原的 `0x4ea2a` 呼叫 ABI 決定，所以必須由呼叫端明確指定，不沿用猜測的介面色。
    ///
    /// 與 Go 相同，逐列檢查邊界：前面的列已寫入後才可能回報超出目的地。
    pub fn blit_glyph(&self, dst: &mut [u8], stride: i64, base: i64, index: i64, color: u8) -> Result<()> {
        if !self.valid_index(index) || stride < GLYPH_WIDTH as i64 || base < 0 {
            return Err(FdtxtError::InvalidBlit);
        }
        for y in 0..GLYPH_HEIGHT as i64 {
            let start = base.wrapping_add(y.wrapping_mul(stride));
            if start.wrapping_add(GLYPH_WIDTH as i64) > dst.len() as i64 {
                return Err(FdtxtError::BlitExceeds);
            }
            let row = self.row(index, y);
            for x in 0..GLYPH_WIDTH as i64 {
                if row & (0x8000u16 >> x) != 0 {
                    dst[start.wrapping_add(x) as usize] = color;
                }
            }
        }
        Ok(())
    }

    /// 重現 `0x4ea2a` 的 16×16 迴圈。邊界檢查刻意比原版的裸指標寫入嚴格：字形左側要留一個像素，
    /// 字格下方要留一列陰影。
    pub fn blit_native_glyph(
        &self,
        dst: &mut [u8],
        stride: i64,
        base: i64,
        index: i64,
        style: NativeGlyphStyle,
    ) -> Result<()> {
        if !self.valid_index(index) || stride < GLYPH_WIDTH as i64 || base < 1 {
            return Err(FdtxtError::InvalidNativeBlit);
        }
        let at = |y: i64, x: i64| base.wrapping_add(y.wrapping_mul(stride)).wrapping_add(x);
        if at(GLYPH_HEIGHT as i64, GLYPH_WIDTH as i64) > dst.len() as i64 {
            return Err(FdtxtError::NativeBlitExceeds);
        }
        if style.background != 0 {
            for y in 0..GLYPH_HEIGHT as i64 {
                for x in 0..GLYPH_WIDTH as i64 {
                    dst[at(y, x) as usize] = style.background;
                }
            }
        }
        if let Some(present) = &self.native_presenter
            && present(stride, base, index, style)
        {
            return Ok(());
        }
        for y in 0..GLYPH_HEIGHT as i64 {
            let row = self.row(index, y);
            for x in 0..GLYPH_WIDTH as i64 {
                if row & (0x8000u16 >> x) == 0 {
                    continue;
                }
                let pos = at(y, x);
                dst[pos as usize] = style.foreground;
                dst[pos.wrapping_add(stride).wrapping_sub(1) as usize] = style.shadow;
                dst[pos.wrapping_add(stride) as usize] = style.shadow;
            }
        }
        Ok(())
    }
}

// ---------------------------------------------------------------------------
// 分離素材（fd2-asset-import 的輸出）
// ---------------------------------------------------------------------------

// 欄位缺漏時取零值（同 Go），未知欄位略過。鍵名大小寫、重複鍵與 `null` 純量依 serde 規則拒收，
// 不沿用 Go encoding/json 的寬鬆處理（見模組說明）。

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedSource {
    file: String,
    resource: i64,
    size: i64,
    md5: String,
    sha256: String,
    raw_size: i64,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedToken {
    kind: String,
    glyph_index: Option<i64>,
    control: String,
    text: String,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedString {
    string_id: String,
    source_index: i64,
    text: String,
    /// Go 以 nil 判斷缺欄與 `null`，空陣列則合法。
    tokens: Option<Vec<SeparatedToken>>,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedResource {
    schema_version: i64,
    kind: String,
    asset_id: String,
    status: String,
    evidence: String,
    /// Go 版只解碼不檢查；保留欄位讓型別不符時同樣拒收。
    #[serde(rename = "reason_code")]
    _reason_code: String,
    source: SeparatedSource,
    strings: Vec<SeparatedString>,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedFontDocument {
    schema_version: i64,
    kind: String,
    asset_id: String,
    status: String,
    evidence: String,
    glyph_count: i64,
    cell_width: i64,
    cell_height: i64,
    columns: i64,
    rows: i64,
    atlas: String,
    source: SeparatedSource,
}

/// 載入 fd2-asset-import 輸出的無損可編輯字元投影。絕不退回讀 FDTXT.DAT。
pub fn load_separated_resource(root: impl AsRef<Path>, resource: i64) -> Result<Strings> {
    let root = root.as_ref();
    if root.as_os_str().is_empty() || resource < 0 {
        return Err(FdtxtError::InvalidSeparatedRequest);
    }
    let path = root.join(format!("FDTXT_{resource:03}")).join("resource.json");
    let raw = std::fs::read(path).map_err(|source| FdtxtError::SeparatedIo { resource, source })?;
    let document: SeparatedResource =
        serde_json::from_slice(&raw).map_err(|source| FdtxtError::SeparatedJson { resource, source })?;
    let src = &document.source;
    if document.schema_version != 1
        || document.kind != "fdtxt_word_table"
        || document.asset_id != format!("text/FDTXT_{resource:03}")
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || src.file != "FDTXT.DAT"
        || src.resource != resource
        || src.size != FDTXT_SOURCE_SIZE
        || src.md5 != FDTXT_SOURCE_MD5
        || src.sha256 != FDTXT_SOURCE_SHA256
        || src.raw_size <= 0
        || document.strings.is_empty()
    {
        return Err(FdtxtError::SeparatedMetadata(resource));
    }
    let mut words = vec![Vec::new(); document.strings.len()];
    for (index, entry) in document.strings.iter().enumerate() {
        let want_id = format!("FDTXT_{resource:03}/string_{index:04}");
        let tokens = match &entry.tokens {
            Some(tokens) if entry.source_index == index as i64 && entry.string_id == want_id => tokens,
            _ => return Err(FdtxtError::StringIdentity { resource, index }),
        };
        let mut projection = String::new();
        for (token, t) in tokens.iter().enumerate() {
            match t.kind.as_str() {
                "glyph" => match t.glyph_index {
                    Some(g) if g >= 0 && g < i64::from(CONTROL_MIN) && t.control.is_empty() && !t.text.is_empty() => {
                        words[index].push(g as u16);
                        projection.push_str(&t.text);
                    }
                    _ => return Err(FdtxtError::InvalidGlyph { resource, index, token }),
                },
                "control" => {
                    let invalid = || FdtxtError::InvalidControl { resource, index, token };
                    if t.glyph_index.is_some() || !t.text.is_empty() || t.control.len() != 4 {
                        return Err(invalid());
                    }
                    // strconv.ParseUint(control, 16, 16)：base 16 不接受正負號、底線或 0x 前綴。
                    if !t.control.bytes().all(|b| b.is_ascii_hexdigit()) {
                        return Err(invalid());
                    }
                    let value = u16::from_str_radix(&t.control, 16).map_err(|_| invalid())?;
                    if !(CONTROL_MIN..STRING_END).contains(&value) {
                        return Err(invalid());
                    }
                    words[index].push(value);
                }
                _ => return Err(FdtxtError::UnknownKind { resource, index, token }),
            }
        }
        if projection != entry.text {
            return Err(FdtxtError::Projection { resource, index });
        }
    }
    Ok(Strings { words })
}

/// 解碼後的 PNG，像素值為 Go `img.At(x, y).RGBA()` 的結果（16 位元、已乘 alpha）。
#[derive(Debug, Clone)]
pub struct DecodedPng {
    pub width: usize,
    pub height: usize,
    /// Go `png.Decode` 會回傳的影像型別（`fmt.Sprintf("%T", img)`）。
    pub go_type: &'static str,
    pixels: Vec<[u32; 4]>,
}

impl DecodedPng {
    pub fn rgba(&self, x: usize, y: usize) -> [u32; 4] {
        self.pixels[y * self.width + x]
    }
}

/// 以 png crate 解碼，並換算成 Go `color.Color.RGBA()` 的值：8 位元樣本乘 0x101；NRGBA／NRGBA64
/// （有 alpha 或 tRNS 的影像）先乘 alpha 再除以 0xff／0xffff；調色盤經展開後與 Go 的
/// `color.RGBA`／`color.NRGBA` 調色盤項同值。只保證格式正確的 PNG 與 Go 相同。
pub fn decode_png_rgba(data: &[u8]) -> std::result::Result<DecodedPng, png::DecodingError> {
    use png::{BitDepth, ColorType};
    let mut decoder = png::Decoder::new(std::io::Cursor::new(data));
    decoder.set_transformations(png::Transformations::EXPAND);
    let mut reader = decoder.read_info()?;
    let info = reader.info();
    let (width, height) = (info.width as usize, info.height as usize);
    let (trns, wide) = (info.trns.is_some(), info.bit_depth == BitDepth::Sixteen);
    let go_type = match info.color_type {
        ColorType::Indexed => "*image.Paletted",
        ColorType::Grayscale if trns => {
            if wide {
                "*image.NRGBA64"
            } else {
                "*image.NRGBA"
            }
        }
        ColorType::Grayscale => {
            if wide {
                "*image.Gray16"
            } else {
                "*image.Gray"
            }
        }
        ColorType::Rgb if !trns => {
            if wide {
                "*image.RGBA64"
            } else {
                "*image.RGBA"
            }
        }
        _ => {
            if wide {
                "*image.NRGBA64"
            } else {
                "*image.NRGBA"
            }
        }
    };
    // 與 png crate 預設的 64 MiB 限制一致，避免依 IHDR 宣告的尺寸配置過大的緩衝。
    let size = reader
        .output_buffer_size()
        .filter(|&n| n <= png::Limits::default().bytes)
        .ok_or(png::DecodingError::LimitsExceeded)?;
    let mut buf = vec![0u8; size];
    reader.next_frame(&mut buf)?;
    let (color, depth) = reader.output_color_type();
    let channels = color.samples();
    let sample_bytes = if depth == BitDepth::Sixteen { 2 } else { 1 };
    let mut pixels = Vec::with_capacity(width * height);
    for px in buf.chunks_exact(channels * sample_bytes).take(width * height) {
        let s = |c: usize| {
            if sample_bytes == 2 {
                u32::from(u16::from_be_bytes([px[2 * c], px[2 * c + 1]]))
            } else {
                u32::from(px[c]) * 0x101
            }
        };
        let (max, [r, g, b, a]) = match color {
            ColorType::Grayscale => (0, [s(0), s(0), s(0), 0xffff]),
            ColorType::GrayscaleAlpha => (0xffff, [s(0), s(0), s(0), s(1)]),
            ColorType::Rgb | ColorType::Indexed => (0, [s(0), s(1), s(2), 0xffff]),
            ColorType::Rgba => (0xffff, [s(0), s(1), s(2), s(3)]),
        };
        pixels.push(if max == 0 {
            [r, g, b, a]
        } else if sample_bytes == 1 {
            // Go NRGBA：(v|v<<8) * A8 / 0xff。
            let a8 = a / 0x101;
            [r * a8 / 0xff, g * a8 / 0xff, b * a8 / 0xff, a]
        } else {
            [r * a / 0xffff, g * a / 0xffff, b * a / 0xffff, a]
        });
    }
    Ok(DecodedPng { width, height, go_type, pixels })
}

/// 從標準圖集重建 FDOTHER#4 的 1bpp 字形庫。PNG 只是傳輸格式；只接受 0／255 的二值像素，
/// 避免編輯器悄悄把反鋸齒帶進原版繪字契約。
pub fn load_separated_font(root: impl AsRef<Path>) -> Result<Font> {
    let root = root.as_ref();
    if root.as_os_str().is_empty() {
        return Err(FdtxtError::FontRootEmpty);
    }
    let directory = root.join("fdother_004");
    let raw = std::fs::read(directory.join("font.json")).map_err(FdtxtError::FontMetadataIo)?;
    let document: SeparatedFontDocument = serde_json::from_slice(&raw).map_err(FdtxtError::FontJson)?;
    let want_rows = (SEPARATED_FONT_GLYPHS + SEPARATED_FONT_COLUMNS - 1) / SEPARATED_FONT_COLUMNS;
    let src = &document.source;
    if document.schema_version != 1
        || document.kind != "bitmap_font"
        || document.asset_id != "font/FDOTHER_004"
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.glyph_count != SEPARATED_FONT_GLYPHS
        || document.cell_width != GLYPH_WIDTH as i64
        || document.cell_height != GLYPH_HEIGHT as i64
        || document.columns != SEPARATED_FONT_COLUMNS
        || document.rows != want_rows
        || document.atlas != "atlas.png"
        || src.file != "FDOTHER.DAT"
        || src.resource != 4
        || src.size != FDOTHER_SOURCE_SIZE
        || src.md5 != FDOTHER_SOURCE_MD5
        || src.sha256 != FDOTHER_SOURCE_SHA256
        || src.raw_size != SEPARATED_FONT_GLYPHS * GLYPH_BYTES as i64
    {
        return Err(FdtxtError::FontMetadata);
    }
    let mut file = std::fs::File::open(directory.join(&document.atlas)).map_err(FdtxtError::AtlasOpen)?;
    let mut bytes = Vec::new();
    file.read_to_end(&mut bytes).map_err(FdtxtError::AtlasRead)?;
    let atlas = decode_png_rgba(&bytes).map_err(FdtxtError::AtlasDecode)?;
    let (columns, rows) = (document.columns as usize, document.rows as usize);
    if atlas.width != columns * GLYPH_WIDTH || atlas.height != rows * GLYPH_HEIGHT {
        return Err(FdtxtError::AtlasGeometry);
    }
    let glyphs = document.glyph_count as usize;
    let mut packed = vec![0u8; glyphs * GLYPH_BYTES];
    for glyph in 0..glyphs {
        let base_x = (glyph % columns) * GLYPH_WIDTH;
        let base_y = (glyph / columns) * GLYPH_HEIGHT;
        for y in 0..GLYPH_HEIGHT {
            let mut row: u16 = 0;
            for x in 0..GLYPH_WIDTH {
                let [r, g, b, _] = atlas.rgba(base_x + x, base_y + y);
                if (r != 0 || g != 0 || b != 0) && (r != 0xffff || g != 0xffff || b != 0xffff) {
                    return Err(FdtxtError::NonBinaryPixel);
                }
                if r == 0xffff {
                    row |= 0x8000 >> x;
                }
            }
            packed[glyph * GLYPH_BYTES + y * 2..][..2].copy_from_slice(&row.to_be_bytes());
        }
    }
    parse_font(&packed)
}
