//! FIGANI 戰鬥動畫資源與原版指令演出排程。
//!
//! 移植自 Go `remake/internal/figani`（凍結於 v.1.1.11）。解碼器保留原始調色盤索引與透明 RLE
//! 區段，讓呼叫端在索引色原生畫面上重現 `0x2935b`，不重用匯出的 RGBA PNG。`native_command*`
//! 子模組保存 `0x2A6BD` 各指令處理器已閉合的資源、計數器與亂數消費契約；位址與證據註解沿用
//! Go 版，欄位不替畫面、音效或遊戲狀態命名。
//!
//! Go 慣用法的對應（逐項也寫在各處註解）：
//! - Go `int` 一律對應 `i64`（amd64 上的寬度），呼叫端可傳入的值以環繞運算重現 Go 的溢位行為。
//! - Go 的 `*Animation` 參數可為 nil，這裡以 `Option<&Animation>` 表示；nil 接收者的方法由 Rust
//!   型別排除。
//! - Go 的 nil 與空 slice 在這個套件沒有可觀察的差別（只看 `len`），一律對應空 `Vec`。
//! - Go 在錯誤時仍回傳的狀態或亂數值（例如 `return nil, state, err`）以 [`StateError`] 保留。
//! - Go 的 `func` resolver 以 [`Resolver`]（可為 `None` 的 `&mut dyn FnMut`）表示，錯誤原樣傳回。
//! - `NativeCommand6Coordinates` 的三角函數照抄 Go 純 Go 實作的多項式（amd64 不用組語、不融合
//!   FMA），超出 int64 的浮點轉整數照 amd64 得到 `i64::MIN`，避免平台 libm 末位差異改變截斷結果。
//!
//! 刻意不照做：分離素材的 JSON、PNG 與檔案系統下層錯誤不重現 Go 標準函式庫的行為與文字（改用
//! `serde_json`、`png` crate 與 `std`）；figani 自己的契約錯誤仍逐字相同。對畸形輸入只保證和 Go 一樣
//! 拒收，見 `doc`、`image` 模組與 `tests/golden_figani.rs` 的已知差異清單。

// 序列建構器在錯誤時要連同 Go 版回傳的狀態一起交回（見 `StateError`），錯誤型別因此較大；
// 這是刻意保留 Go 語意的取捨，不是效能熱點。
#![allow(clippy::result_large_err)]

use crate::archive::{self, ArchiveError};
use std::fmt;
use std::path::Path;

mod doc;
mod image;
mod native_command0;
mod native_command1;
mod native_command2;
mod native_command24;
mod native_command3;
mod native_command5;
mod native_command6;
mod native_command7;
mod native_command8;
mod native_command9;
mod native_command_compound_presentation;
mod native_command_indexed_entries;
mod separated;

pub use native_command_compound_presentation::*;
pub use native_command_indexed_entries::*;
pub use native_command0::*;
pub use native_command1::*;
pub use native_command2::*;
pub use native_command3::*;
pub use native_command5::*;
pub use native_command6::*;
pub use native_command7::*;
pub use native_command8::*;
pub use native_command9::*;
pub use native_command24::*;
pub use separated::*;

/// 呼叫端 resolver 回傳的錯誤；Go 版原樣往上傳，不包裝。
pub type ResolveError = Box<dyn std::error::Error + Send + Sync>;

/// `0x2A6BD` 對每個目標呼叫的命中／傷害 resolver：參數是目標序號與目前亂數，
/// 回傳擲骰後的亂數與是否命中。Go 版是可為 nil 的 `func(int, uint16) (uint16, bool, error)`。
pub type Resolver<'a> = Option<&'a mut dyn FnMut(i64, u16) -> std::result::Result<(u16, bool), ResolveError>>;

/// 包裝下層錯誤的來源（檔案、JSON、PNG）。Go 版這裡包的是 `os`、`encoding/json`、`image/png` 的錯誤；
/// 刻意不照做這些標準函式庫的錯誤文字，黃金資料只比對 figani 自己的前綴與接受／拒收。
#[derive(Debug)]
pub enum Cause {
    Io { path: String, source: std::io::Error },
    Json(String),
    UnknownFormat,
    Png(String),
}

impl fmt::Display for Cause {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Cause::Io { path, source } => write!(f, "{path}: {source}"),
            Cause::Json(e) | Cause::Png(e) => f.write_str(e),
            Cause::UnknownFormat => f.write_str("image: unknown format"),
        }
    }
}

/// RLE 解碼錯誤；Display 與 Go 版 `decodeRLE` 的錯誤字串逐字相同。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum RleError {
    EndsInRow(i64),
    OverrunsRow(i64),
    ColourRunLacksValue,
    DitherRunLacksValue,
    LiteralRunExceedsData,
}

impl fmt::Display for RleError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            RleError::EndsInRow(y) => write!(f, "RLE ends in row {y}"),
            RleError::OverrunsRow(y) => write!(f, "RLE overruns row {y}"),
            RleError::ColourRunLacksValue => f.write_str("colour run lacks value"),
            RleError::DitherRunLacksValue => f.write_str("dither run lacks value"),
            RleError::LiteralRunExceedsData => f.write_str("literal run exceeds data"),
        }
    }
}

/// figani 的錯誤；`Display` 與 Go 版的錯誤字串逐字相同。
#[derive(Debug)]
pub enum FiganiError {
    /// `fdother.ReadResource` 的錯誤原樣傳回（Go 的 `DecodeResource` 不包裝）。
    Archive(ArchiveError),
    /// 固定文字的拒收。
    Static(&'static str),
    /// 已格式化、不包裝下層錯誤的拒收。
    Message(String),
    /// `figani: frame %d: %w`
    FrameRle { frame: usize, source: RleError },
    /// `fmt.Errorf("<prefix>%w", cause)`，prefix 含結尾的 `": "`。
    Wrapped { prefix: String, cause: Cause },
    /// 原樣傳回的下層錯誤（`LoadNativeCommandIndexedEntryTable` 不包裝 os／json 錯誤）。
    Cause(Cause),
    /// `figani: resource %d unavailable (%v); status: %w`
    FallbackUnavailable { resource: i64, primary: Box<FiganiError>, status: Box<FiganiError> },
    /// 呼叫端 resolver 的錯誤原樣傳回。
    Resolver(ResolveError),
}

impl fmt::Display for FiganiError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            FiganiError::Archive(e) => e.fmt(f),
            FiganiError::Static(message) => f.write_str(message),
            FiganiError::Message(message) => f.write_str(message),
            FiganiError::FrameRle { frame, source } => write!(f, "figani: frame {frame}: {source}"),
            FiganiError::Wrapped { prefix, cause } => write!(f, "{prefix}{cause}"),
            FiganiError::Cause(cause) => cause.fmt(f),
            FiganiError::FallbackUnavailable { resource, primary, status } => {
                write!(f, "figani: resource {resource} unavailable ({primary}); status: {status}")
            }
            FiganiError::Resolver(e) => e.fmt(f),
        }
    }
}

impl std::error::Error for FiganiError {}

impl From<ArchiveError> for FiganiError {
    fn from(e: ArchiveError) -> Self {
        FiganiError::Archive(e)
    }
}

pub type Result<T> = std::result::Result<T, FiganiError>;

/// Go 版在錯誤時仍回傳的值（序列建構器的 `state`、亂數走訪的 `rng`）。
#[derive(Debug)]
pub struct StateError<S> {
    pub state: S,
    pub error: FiganiError,
}

impl<S> fmt::Display for StateError<S> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.error.fmt(f)
    }
}

impl<S: fmt::Debug> std::error::Error for StateError<S> {}

/// 亂數走訪的錯誤：`rng` 是 Go 版在錯誤時回傳的亂數。
pub type RngWalkError = StateError<u16>;

pub(crate) fn msg(message: String) -> FiganiError {
    FiganiError::Message(message)
}

pub(crate) fn fail<S>(state: S, error: FiganiError) -> StateError<S> {
    StateError { state, error }
}

/// FIGANI 資源。原生容器以 byte 0 為總幀數、byte 1 為前奏旗標、byte 2 為前奏幀數。
/// `header_byte4` 由 `0x2b659` 消費，遊戲語意尚未命名。保留原始標頭位元組，避免演出端追蹤時
/// 默默丟掉原版輸入。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Animation {
    pub frames: Vec<Frame>,
    pub header_byte1: u8,
    pub header_byte2: u8,
    pub header_byte4: u8,
}

/// `0x2935b` 消費的 13 位元組 FIGANI 幀標頭欄位。X／Y 是原生 320x200 的有號座標；`pixels` 是
/// 解碼後 W×H 的索引色影像，`mask` 區分 codec 的透明輸出與調色盤索引 0。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Frame {
    pub x: i64,
    pub y: i64,
    pub width: i64,
    pub height: i64,
    pub pixels: Vec<u8>,
    pub mask: Vec<u8>,
    pub delay: i64,
    pub raw_byte4: u8,
    pub raw_byte5: u8,
    pub raw_byte7: u8,
}

/// 對應 `0x2b9a1` 的兩位元組全域狀態（`0x540fd` 幀序、`0x540fc` 子幀計數）。先選幀再前進；
/// `0x2935b` 的索引色繪製仍由呼叫端負責。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeScheduler {
    pub frame_index: i64,
    pub subframe: i64,
}

impl NativeScheduler {
    /// 回傳這次呼叫要畫的幀。`advance=false` 是原生 arg4==0 的初始化呼叫：清除子幀、不繪製。
    /// Go 版的 nil 接收者由 Rust 型別排除；nil 動畫以 `None` 表示。
    pub fn step(&mut self, animation: Option<&Animation>, advance: bool) -> Result<(i64, bool)> {
        let frames = match animation {
            Some(a) if !a.frames.is_empty() => &a.frames,
            _ => return Err(FiganiError::Static("figani: invalid native scheduler state")),
        };
        if self.frame_index < 0 || self.frame_index >= frames.len() as i64 || self.subframe < 0 {
            return Err(FiganiError::Static("figani: invalid native scheduler state"));
        }
        if !advance {
            self.subframe = 0;
            return Ok((self.frame_index, false));
        }
        let selected = self.frame_index;
        let delay = frames[selected as usize].delay;
        self.subframe = self.subframe.wrapping_add(1);
        if self.subframe >= delay {
            self.subframe = 0;
            self.frame_index += 1;
            if self.frame_index >= frames.len() as i64 {
                self.frame_index = 0;
            }
        }
        Ok((selected, true))
    }
}

/// 把原生 descriptor delay 轉成呼叫端的顯示 tick。原生函式在目前幀呈現後才前進一個子幀；
/// `ticks_per_native` 讓繪製端保留明確的速度控制而不丟掉原始 +6 delay。不含像素或命中效果語意，
/// 可安全用於只做演出的橋接。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct DisplayScheduler {
    native: NativeScheduler,
    animation: Animation,
    ticks_per_native: i64,
    frame_starts: Vec<i64>,
    body_ticks: i64,
    display_tick: i64,
    current_frame: i64,
    done: bool,
}

impl DisplayScheduler {
    /// Go `NewDisplayScheduler`：依原始 descriptor delay 建立嚴格時間軸。`0x2B9A1` 先呈現目前幀，
    /// 再把子幀加一並比較 delay；因此 delay=0 與 delay=1 都會呈現一次。負值不是 byte descriptor 的
    /// 合法輸入，仍須拒絕。總長以環繞乘加計算，與 Go `int` 溢位相同。
    pub fn new(delays: &[i64], ticks_per_native: i64) -> Result<DisplayScheduler> {
        if delays.is_empty() {
            return Err(FiganiError::Static("figani: display scheduler has no frames"));
        }
        if ticks_per_native <= 0 {
            return Err(FiganiError::Static("figani: display scheduler has invalid tick scale"));
        }
        let mut starts = vec![0i64; delays.len() + 1];
        let mut frames = Vec::with_capacity(delays.len());
        let mut total: i64 = 0;
        for (i, &delay) in delays.iter().enumerate() {
            if delay < 0 {
                return Err(msg(format!("figani: frame {i} has invalid delay {delay}")));
            }
            starts[i] = total;
            frames.push(Frame { delay, ..Frame::default() });
            let effective = if delay == 0 { 1 } else { delay };
            total = total.wrapping_add(effective.wrapping_mul(ticks_per_native));
        }
        starts[delays.len()] = total;
        Ok(DisplayScheduler {
            native: NativeScheduler::default(),
            animation: Animation { frames, ..Animation::default() },
            ticks_per_native,
            frame_starts: starts,
            body_ticks: total,
            display_tick: 0,
            current_frame: 0,
            done: false,
        })
    }

    /// 呈現呼叫端的一個顯示 tick，回傳 `(幀, 是否呈現, 是否結束)`。最後一個 tick 回報結束；
    /// 呼叫端可在另外擁有的尾段停留中繼續畫 [`Self::current_frame`]，不推進這個狀態機。
    pub fn step(&mut self) -> Result<(i64, bool, bool)> {
        if self.animation.frames.is_empty() || self.ticks_per_native <= 0 || self.body_ticks <= 0 {
            return Err(FiganiError::Static("figani: invalid display scheduler state"));
        }
        if self.done {
            return Ok((self.current_frame, false, true));
        }
        if self.display_tick % self.ticks_per_native == 0 {
            let (selected, rendered) = self.native.step(Some(&self.animation), true)?;
            if !rendered {
                return Err(FiganiError::Static("figani: native display step did not present"));
            }
            self.current_frame = selected;
        }
        self.display_tick += 1;
        if self.display_tick >= self.body_ticks {
            self.done = true;
        }
        Ok((self.current_frame, true, self.done))
    }

    /// 呼叫端尾段停留之前的精確縮放長度。
    pub fn body_ticks(&self) -> i64 {
        self.body_ticks
    }

    /// 某幀第一次呈現的縮放 tick。可供另有證據的視覺標記使用，本身不指派攻擊、傷害或音效語意。
    pub fn frame_start(&self, frame: i64) -> Option<i64> {
        if frame < 0 || frame >= self.animation.frames.len() as i64 {
            return None;
        }
        Some(self.frame_starts[frame as usize])
    }

    /// 最後選到的幀，不推進排程。
    pub fn current_frame(&self) -> i64 {
        self.current_frame
    }
}

/// 從 LLLLLL 容器讀一筆資源並解析。Go 的 `int` 資源序號可為負；負值在容器驗證後回報超出範圍，
/// 這裡以 `usize::MAX` 送進共用容器讀取以保留同樣的檢查順序。
pub fn decode_resource(path: impl AsRef<Path>, resource: i64) -> Result<Animation> {
    let index = usize::try_from(resource).unwrap_or(usize::MAX);
    let raw = archive::read_resource(path, index)?;
    parse(&raw)
}

fn le_u16(raw: &[u8], at: usize) -> u16 {
    u16::from_le_bytes([raw[at], raw[at + 1]])
}

fn le_u32(raw: &[u8], at: usize) -> usize {
    u32::from_le_bytes([raw[at], raw[at + 1], raw[at + 2], raw[at + 3]]) as usize
}

/// 解析一筆 FIGANI 資源。
pub fn parse(raw: &[u8]) -> Result<Animation> {
    if raw.len() < 12 {
        return Err(FiganiError::Static("figani: animation is too short"));
    }
    // 0x29409 讀 byte 0，0x29510／0x295c3 另外讀 byte 1 與 byte 2。把 byte 0..1 當 u16 會拒收
    // 每一筆設了原生前奏旗標的原版動畫。
    let n = raw[0] as usize;
    if n == 0 || 8 + 4 * n > raw.len() {
        return Err(FiganiError::Static("figani: invalid frame table"));
    }
    if raw[1] != 0 && raw[2] as usize > n {
        return Err(FiganiError::Static("figani: prelude frame count exceeds total frames"));
    }
    let mut frames = Vec::with_capacity(n);
    let mut previous = 8 + 4 * n;
    for i in 0..n {
        let off = le_u32(raw, 8 + 4 * i);
        let end = if i + 1 < n { le_u32(raw, 8 + 4 * (i + 1)) } else { raw.len() };
        if off < previous || off + 13 > end || end > raw.len() {
            return Err(msg(format!("figani: invalid frame {i} offset")));
        }
        let (w, h) = (le_u16(raw, off + 9) as i64, le_u16(raw, off + 11) as i64);
        if w <= 0 || h <= 0 || w > 1024 || h > 1024 {
            return Err(msg(format!("figani: invalid frame {i} geometry")));
        }
        let (pixels, mask) = decode_rle(&raw[off + 13..end], w as usize, h as usize)
            .map_err(|source| FiganiError::FrameRle { frame: i, source })?;
        // 0x2939d 把 +4、+5、+6、+7 當四個獨立位元組消費；+6 是幀 delay。其餘位元組以原始偏移
        // 保留，不指派未閉合的效果、音效或疊層語意。在 +6 讀 u16 會把 byte7=6 加 delay 38 變成 1574。
        frames.push(Frame {
            x: le_u16(raw, off) as i16 as i64,
            y: le_u16(raw, off + 2) as i16 as i64,
            width: w,
            height: h,
            pixels,
            mask,
            raw_byte4: raw[off + 4],
            raw_byte5: raw[off + 5],
            delay: raw[off + 6] as i64,
            raw_byte7: raw[off + 7],
        });
        previous = off;
    }
    Ok(Animation { frames, header_byte1: raw[1], header_byte2: raw[2], header_byte4: raw[4] })
}

fn decode_rle(src: &[u8], width: usize, height: usize) -> std::result::Result<(Vec<u8>, Vec<u8>), RleError> {
    let mut pixels = vec![0u8; width * height];
    let mut mask = vec![0u8; width * height];
    let mut pos = 0usize;
    for y in 0..height {
        let mut x = 0usize;
        while x < width {
            if pos >= src.len() {
                return Err(RleError::EndsInRow(y as i64));
            }
            let ctrl = src[pos];
            pos += 1;
            let (count, mode) = ((ctrl & 0x3f) as usize + 1, ctrl >> 6);
            let span = if mode == 1 { count * 2 } else { count };
            if x + span > width {
                return Err(RleError::OverrunsRow(y as i64));
            }
            let mut write = |at: usize, value: u8| {
                pixels[y * width + at] = value;
                mask[y * width + at] = 1;
            };
            match mode {
                0 => {
                    if pos >= src.len() {
                        return Err(RleError::ColourRunLacksValue);
                    }
                    let v = src[pos];
                    pos += 1;
                    for i in 0..count {
                        write(x + i, v);
                    }
                }
                1 => {
                    if pos >= src.len() {
                        return Err(RleError::DitherRunLacksValue);
                    }
                    let v = src[pos];
                    pos += 1;
                    for i in 0..count {
                        write(x + 2 * i + 1, v);
                    }
                }
                2 => {
                    if pos + count > src.len() {
                        return Err(RleError::LiteralRunExceedsData);
                    }
                    for (i, &v) in src[pos..pos + count].iter().enumerate() {
                        write(x + i, v);
                    }
                    pos += count;
                }
                // 原生透明區段保留目的地。
                _ => {}
            }
            x += span;
        }
    }
    Ok((pixels, mask))
}

impl Frame {
    /// 重現 `0x2c548` 的 FIGANI 呼叫所用的透明分支。
    pub fn blit_at(&self, dst: &mut [u8], stride: i64) -> Result<()> {
        self.blit_at_base(dst, stride, 0)
    }

    /// 帶明確位元組起點的 [`Self::blit_at`]，起點在較大的原生工作畫面內。`0x29164` 以它做
    /// stage*10 位移到 640 跨距的緩衝區；幀的有號座標不變。
    pub fn blit_at_base(&self, dst: &mut [u8], stride: i64, base: i64) -> Result<()> {
        let len = dst.len() as i64;
        let mut invalid =
            self.width <= 0 || self.height <= 0 || stride <= 0 || base < 0 || base > len || self.x < 0 || self.y < 0;
        if !invalid {
            let pixels = self.pixels.len() as i64;
            invalid = pixels % self.width != 0
                || pixels / self.width != self.height
                || self.mask.len() != self.pixels.len()
                || self.x > stride
                || self.width > stride - self.x;
        }
        if !invalid {
            // #92：檢查實際末列 span，右半畫布的 base 不需要額外尾列 padding。
            let remaining = len - base;
            let row_end = self.x + self.width; // 已由 stride 界線保證不溢位。
            invalid = row_end > remaining;
            if !invalid {
                let last_available_row = (remaining - row_end) / stride;
                invalid = self.y > last_available_row || self.height - 1 > last_available_row - self.y;
            }
        }
        if invalid {
            return Err(msg(format!(
                "figani: frame cannot be blitted to destination: frame=({},{},{},{}) dst={} stride={} base={}",
                self.x,
                self.y,
                self.width,
                self.height,
                dst.len(),
                stride,
                base
            )));
        }
        let (w, h) = (self.width as usize, self.height as usize);
        let (stride, base, fx, fy) = (stride as usize, base as usize, self.x as usize, self.y as usize);
        for y in 0..h {
            for x in 0..w {
                let i = y * w + x;
                if self.mask[i] != 0 {
                    dst[base + (fy + y) * stride + fx + x] = self.pixels[i];
                }
            }
        }
        Ok(())
    }

    /// 重現 `0x2939d` 的暫存緩衝區位移，同時呈現在有界的索引色畫面上。畫面外的像素裁掉，對應原生
    /// 400 跨距暫存邊界之後的 320 寬視窗複製。`opaque_fill` 有值時每個不透明像素都改填該索引，
    /// 透明區段仍保留目的地，對應 `0x4e63d` 的 0..255 分支；`None` 保留來源索引，即原生 -1 分支。
    /// Go 的 `*byte` 以 `Option<u8>` 表示。幾何乘積以環繞運算比較，與 Go `int` 相同。
    pub fn blit_translated(
        &self,
        dst: &mut [u8],
        stride: i64,
        dx: i64,
        dy: i64,
        opaque_fill: Option<u8>,
    ) -> Result<()> {
        if self.width <= 0
            || self.height <= 0
            || self.pixels.len() as i64 != self.width.wrapping_mul(self.height)
            || self.mask.len() != self.pixels.len()
            || stride <= 0
            || dst.len() as i64 % stride != 0
        {
            return Err(FiganiError::Static("figani: translated frame cannot be blitted to destination"));
        }
        let height = dst.len() as i64 / stride;
        for y in 0..self.height {
            let dst_y = self.y.wrapping_add(dy).wrapping_add(y);
            if dst_y < 0 || dst_y >= height {
                continue;
            }
            for x in 0..self.width {
                let dst_x = self.x.wrapping_add(dx).wrapping_add(x);
                if dst_x < 0 || dst_x >= stride {
                    continue;
                }
                let i = (y * self.width + x) as usize;
                if self.mask[i] != 0 {
                    let value = opaque_fill.unwrap_or(self.pixels[i]);
                    dst[(dst_y * stride + dst_x) as usize] = value;
                }
            }
        }
        Ok(())
    }
}

/// 檢查幀的像素與遮罩長度是否符合 W×H；與 Go 版的 `len(frame.Pixels) != frame.Width*frame.Height`
/// 相同，乘積以環繞運算計算。
pub(crate) fn frame_shape_ok(frame: &Frame) -> bool {
    frame.width > 0
        && frame.height > 0
        && frame.pixels.len() as i64 == frame.width.wrapping_mul(frame.height)
        && frame.mask.len() == frame.pixels.len()
}

#[cfg(test)]
mod tests;
