//! DATO.DAT 的四幀頭像資源。
//!
//! 移植自 Go `remake/internal/dato`（凍結於 v.1.1.11）：`dato.go`、`mouth.go`、
//! `confirmation_mouth.go`、`separated.go`。索引像素與四幀嘴型序列和結局文字排程分開；
//! `DATO_075` 是商店店員。錯誤訊息與 Go 版逐字相同（包裹外部 I/O 或 PNG 解碼錯誤的部分除外，
//! 見 [`DatoError`]），以便黃金差分直接比對。
//!
//! Go 的 `int` 一律對應 `i64`，並以 wrapping 運算重現 Go 的二補數溢位；Go 會 panic 的索引
//! （越界切片、除以零）在 Rust 同樣 panic，不改成錯誤值。

mod indexed_png;

use crate::archive::{ArchiveError, read_resource};
use std::fmt;
use std::io::Read;
use std::path::{Component, Path, PathBuf};

pub use indexed_png::ImageError;

/// Go 版未匯出的 `mouthFrameCount`。
const MOUTH_FRAME_COUNT: usize = 4;

/// 錯誤種類；`Display` 與 Go 版的錯誤字串逐字相同。
///
/// 例外：作業系統 I/O 錯誤（`Archive` 的讀檔錯誤、`SeparatedOpen`、`SeparatedDecode` 的
/// `ImageError::Io`）直接帶出 Rust 端的錯誤文字。Go 版這些位置是 `*PathError`，文字因平台
/// 而異。`ImageError::Png` 是 `png` crate 的錯誤文字，不對應 Go `image/png`（見
/// [`load_separated_resource`]）。外層前綴（`dato: separated portrait R frame M decode: ` 等）
/// 與 `image: unknown format` 仍逐字相同。
#[derive(Debug)]
pub enum DatoError {
    /// `fdother.ReadResource` 的錯誤，Go 版原樣回傳。
    Archive(ArchiveError),
    ShortResource,
    FrameOffset(usize),
    FrameHeader(usize),
    FrameGeometry(usize),
    FramePixels {
        frame: usize,
        source: RleError,
    },
    BlitDestination,
    BlitOffset,
    BlitRightToLeftDestination,
    MouthRandom(i64),
    MouthCountdown(i64),
    ConfirmationRandom(i64),
    ConfirmationState {
        state: ConfirmationMouthState,
        sample: i64,
    },
    SeparatedInput,
    SeparatedOpen {
        resource: i64,
        mouth: usize,
        path: PathBuf,
        source: std::io::Error,
    },
    SeparatedDecode {
        resource: i64,
        mouth: usize,
        source: ImageError,
    },
    SeparatedNotIndexed {
        resource: i64,
        mouth: usize,
    },
    SeparatedGeometry {
        resource: i64,
        mouth: usize,
    },
}

/// `decodePixels` 的兩種錯誤（Go 版以 `%w` 包在 `dato: frame %d: ` 之後）。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RleError {
    StreamEnds,
    RepeatLacksPixel,
}

impl fmt::Display for RleError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match self {
            RleError::StreamEnds => "RLE stream ends before frame is filled",
            RleError::RepeatLacksPixel => "RLE repeat lacks a pixel",
        })
    }
}

impl fmt::Display for DatoError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            DatoError::Archive(e) => write!(f, "{e}"),
            DatoError::ShortResource => f.write_str("dato: resource is shorter than four offsets"),
            DatoError::FrameOffset(i) => write!(f, "dato: frame {i} offset is invalid"),
            DatoError::FrameHeader(i) => write!(f, "dato: frame {i} header exceeds bounds"),
            DatoError::FrameGeometry(i) => write!(f, "dato: frame {i} geometry is invalid"),
            DatoError::FramePixels { frame, source } => write!(f, "dato: frame {frame}: {source}"),
            DatoError::BlitDestination => f.write_str("dato: frame destination is too small"),
            DatoError::BlitOffset => f.write_str("dato: frame offset is invalid"),
            DatoError::BlitRightToLeftDestination => f.write_str("dato: right-to-left frame destination is too small"),
            DatoError::MouthRandom(r) => write!(f, "dato: random mouth value {r} outside [0,30)"),
            DatoError::MouthCountdown(c) => write!(f, "dato: mouth countdown {c} is negative"),
            DatoError::ConfirmationRandom(r) => write!(f, "dato: confirmation random value {r} outside [0,30)"),
            // Go 以 `%+v` 印結構：`{Open:true Countdown:-1}`。
            DatoError::ConfirmationState { state, sample } => write!(
                f,
                "dato: invalid confirmation mouth state/sample: {{Open:{} Countdown:{}}}, {sample}",
                state.open, state.countdown
            ),
            DatoError::SeparatedInput => f.write_str("dato: separated portrait input is invalid"),
            // Go 版內層是 os.Open 的 *PathError：`open <path>: <errno 文字>`。
            DatoError::SeparatedOpen { resource, mouth, path, source } => {
                write!(f, "dato: separated portrait {resource} frame {mouth}: open {}: {source}", path.display())
            }
            DatoError::SeparatedDecode { resource, mouth, source } => {
                write!(f, "dato: separated portrait {resource} frame {mouth} decode: {source}")
            }
            DatoError::SeparatedNotIndexed { resource, mouth } => {
                write!(f, "dato: separated portrait {resource} frame {mouth} is not indexed PNG")
            }
            DatoError::SeparatedGeometry { resource, mouth } => {
                write!(f, "dato: separated portrait {resource} frame {mouth} geometry is invalid")
            }
        }
    }
}

impl std::error::Error for DatoError {}

impl From<ArchiveError> for DatoError {
    fn from(e: ArchiveError) -> Self {
        DatoError::Archive(e)
    }
}

pub type Result<T> = std::result::Result<T, DatoError>;

/// 一幀頭像：寬、高與逐列索引像素。欄位型別沿用 Go 的 `int`，呼叫端可以構造任意值，
/// 繪製函式的檢查也照 Go 的整數語意進行。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Frame {
    pub width: i64,
    pub height: i64,
    pub pixels: Vec<u8>,
}

/// 讀 DATO.DAT 的一筆資源並解出四幀。
///
/// Go 的 `resource` 是 `int`：負值在 `ArchiveEntry` 通過標頭與目錄檢查之後才回報
/// `archive resource is out of range`。這裡把負值轉成 `usize::MAX` 交給同一條路徑，錯誤順序
/// 與 Go 相同。
pub fn decode_resource(dat_path: impl AsRef<Path>, resource: i64) -> Result<Vec<Frame>> {
    let index = usize::try_from(resource).unwrap_or(usize::MAX);
    let entry = read_resource(dat_path, index)?;
    parse_resource(&entry)
}

fn le_u32(data: &[u8], at: usize) -> usize {
    u32::from_le_bytes([data[at], data[at + 1], data[at + 2], data[at + 3]]) as usize
}

fn le_u16(data: &[u8], at: usize) -> i64 {
    i64::from(u16::from_le_bytes([data[at], data[at + 1]]))
}

/// 解 DATO 的 LLLLLL 資源本體：四個 u32 幀偏移，每幀是寬、高與 `0x4e916` 高位游程編碼。
/// 和透明 sprite 編碼不同，0 是實際的頭像像素，由 [`Frame::blit_at`] 寫出。
///
/// 照 Go：先檢查全部四個偏移，再逐幀解碼；每幀的串流到下一幀偏移（最後一幀到資源尾端）為止，
/// 填滿像素後剩下的位元組忽略。相同的偏移可以通過偏移檢查，但會在幀標頭檢查失敗。
pub fn parse_resource(data: &[u8]) -> Result<Vec<Frame>> {
    if data.len() < 16 {
        return Err(DatoError::ShortResource);
    }
    let mut offsets = [0usize; 4];
    for i in 0..offsets.len() {
        offsets[i] = le_u32(data, i * 4);
        if offsets[i] < 16 || offsets[i] >= data.len() || (i > 0 && offsets[i] < offsets[i - 1]) {
            return Err(DatoError::FrameOffset(i));
        }
    }
    let mut frames = Vec::with_capacity(4);
    for (i, &start) in offsets.iter().enumerate() {
        let end = if i + 1 < offsets.len() { offsets[i + 1] } else { data.len() };
        if start + 4 > end {
            return Err(DatoError::FrameHeader(i));
        }
        let w = le_u16(data, start);
        let h = le_u16(data, start + 2);
        if w <= 0 || h <= 0 || w > 256 || h > 256 {
            return Err(DatoError::FrameGeometry(i));
        }
        let pixels = decode_pixels(&data[start + 4..end], (w * h) as usize)
            .map_err(|source| DatoError::FramePixels { frame: i, source })?;
        frames.push(Frame { width: w, height: h, pixels });
    }
    Ok(frames)
}

/// `0x4e916` 游程：`code <= 0xc0` 是字面像素（含 0xc0 本身），否則重複下一個位元組
/// `code - 0xc0` 次；超出這幀剩餘像素的部分截掉。
fn decode_pixels(mut stream: &[u8], want: usize) -> std::result::Result<Vec<u8>, RleError> {
    let mut pixels = Vec::with_capacity(want);
    while pixels.len() < want {
        let Some((&code, rest)) = stream.split_first() else {
            return Err(RleError::StreamEnds);
        };
        stream = rest;
        if code <= 0xc0 {
            pixels.push(code);
            continue;
        }
        let Some((&value, rest)) = stream.split_first() else {
            return Err(RleError::RepeatLacksPixel);
        };
        stream = rest;
        let run = (usize::from(code) - 0xc0).min(want - pixels.len());
        pixels.resize(pixels.len() + run, value);
    }
    Ok(pixels)
}

/// Go 的 `s[i:]` 起點：負值或超過長度時 panic，與 Go 的切片越界相同。
fn go_index(i: i64, len: usize) -> usize {
    match usize::try_from(i) {
        Ok(u) if u <= len => u,
        _ => panic!("dato: slice bounds out of range [{i}:{len}]"),
    }
}

/// Go 的內建 `copy`：複製 `min(len(dst), len(src))` 個位元組。
fn go_copy(dst: &mut [u8], src: &[u8]) {
    let n = dst.len().min(src.len());
    dst[..n].copy_from_slice(&src[..n]);
}

impl Frame {
    /// 不透明繪製：每列整段覆寫目的緩衝區，0 也照寫。
    ///
    /// 檢查式照 Go 的求值順序與 wrapping 溢位；`x` 極大使 `x+Width` 溢位成負數時，Go 會通過
    /// 檢查後在切片或除法上 panic，Rust 這裡同樣 panic。
    pub fn blit_at(&self, dst: &mut [u8], stride: i64, x: i64, y: i64) -> Result<()> {
        let len = dst.len() as i64;
        if self.width <= 0
            || self.height <= 0
            || self.pixels.len() as i64 != self.width.wrapping_mul(self.height)
            || x < 0
            || y < 0
            || stride < x.wrapping_add(self.width)
            || y > len.wrapping_div(stride)
            || self.height > len.wrapping_sub(y.wrapping_mul(stride)).wrapping_div(stride)
        {
            return Err(DatoError::BlitDestination);
        }
        for row in 0..self.height {
            let start = go_index(y.wrapping_add(row).wrapping_mul(stride).wrapping_add(x), dst.len());
            let lo = go_index(row.wrapping_mul(self.width), self.pixels.len());
            let hi = go_index((row + 1).wrapping_mul(self.width), self.pixels.len());
            go_copy(&mut dst[start..], &self.pixels[lo..hi]);
        }
        Ok(())
    }

    /// 原版 `0x4e8af` 呼叫端使用的偏移形式。暫存區錨點由呼叫端負責證明，這裡不自行假設。
    pub fn blit_at_offset(&self, dst: &mut [u8], stride: i64, offset: i64) -> Result<()> {
        if stride <= 0 || offset < 0 || (offset % stride).wrapping_add(self.width) > stride {
            return Err(DatoError::BlitOffset);
        }
        self.blit_at(dst, stride, offset % stride, offset / stride)
    }

    /// 重現 `0x4E8E1`：`offset` 是每列最右端，來源像素依序寫入後遞減目的位址；下一列再從
    /// 原始 `offset` 加 `stride` 開始。
    ///
    /// 照 Go：`offset/stride+Height` 溢位時檢查會通過，迴圈寫到越界才 panic，已寫的列保留。
    pub fn blit_right_to_left_at_offset(&self, dst: &mut [u8], stride: i64, offset: i64) -> Result<()> {
        let len = dst.len() as i64;
        if self.width <= 0
            || self.height <= 0
            || self.pixels.len() as i64 != self.width.wrapping_mul(self.height)
            || stride <= 0
            || offset < 0
            || offset % stride < self.width - 1
            || (offset / stride).wrapping_add(self.height) > len / stride
        {
            return Err(DatoError::BlitRightToLeftDestination);
        }
        for row in 0..self.height {
            let right = offset.wrapping_add(row.wrapping_mul(stride));
            for column in 0..self.width {
                // Go 的 `dst[i]` 在 i<0 或 i>=len 時 panic；go_index 擋負值，Rust 索引擋上界。
                let d = go_index(right.wrapping_sub(column), dst.len());
                let s = go_index(row.wrapping_mul(self.width).wrapping_add(column), self.pixels.len());
                dst[d] = self.pixels[s];
            }
        }
        Ok(())
    }
}

/// 原版對白迴圈的小狀態機（`sub_16C57`）。閉嘴時顯示第 0 幀，張嘴的短暫一拍顯示第 3 幀；
/// 其他 DATO 幀是資源變體，不是時序狀態。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct MouthState {
    pub open: bool,
    pub countdown: i64,
}

impl MouthState {
    /// 目前狀態對應的原版 DATO 幀號。
    pub fn frame_index(self) -> usize {
        if self.open { 3 } else { 0 }
    }

    /// 前進一拍。`random_mod30` 是已取樣的 `rand()%0x1e`，只在張嘴後閉嘴時使用；取樣留在外面，
    /// 測試才能決定性，也不宣稱與執行檔全域亂數序列一致。
    pub fn tick(self, random_mod30: i64) -> Result<MouthState> {
        if !(0..30).contains(&random_mod30) {
            return Err(DatoError::MouthRandom(random_mod30));
        }
        if self.countdown < 0 {
            return Err(DatoError::MouthCountdown(self.countdown));
        }
        if self.open {
            return Ok(MouthState { open: false, countdown: random_mod30 + 2 });
        }
        let mut next = self;
        // sub_16C57 採用 post-decrement：只有舊值為零才張嘴；正值則減一後存回。
        if next.countdown == 0 {
            next.open = true;
            return Ok(next);
        }
        next.countdown -= 1;
        Ok(next)
    }
}

/// `sub_19953` 自有的確認嘴型狀態，與 `sub_16C57` 無關。
/// 證據與規格：`docs/data/ida/fd2_confirmation_mouth_runtime_20261005.json`。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ConfirmationMouthState {
    pub open: bool,
    pub countdown: i64,
}

impl ConfirmationMouthState {
    /// Go: `NewConfirmationMouthState`。
    pub fn new(random_mod30: i64) -> Result<ConfirmationMouthState> {
        if !(0..30).contains(&random_mod30) {
            return Err(DatoError::ConfirmationRandom(random_mod30));
        }
        Ok(ConfirmationMouthState { open: false, countdown: random_mod30 + 2 })
    }

    pub fn frame_index(self) -> usize {
        if self.open { 3 } else { 0 }
    }

    /// 只有張嘴後閉嘴時才消耗新的 `random_mod30`。原版 post-decrement 在張嘴那一拍留下
    /// EBP=-1，這裡保留同一狀態。
    pub fn tick(self, random_mod30: i64) -> Result<ConfirmationMouthState> {
        if !(0..30).contains(&random_mod30)
            || (self.open && self.countdown != -1)
            || (!self.open && !(0..=39).contains(&self.countdown))
        {
            return Err(DatoError::ConfirmationState { state: self, sample: random_mod30 });
        }
        if self.open {
            return Ok(ConfirmationMouthState { open: false, countdown: random_mod30 + 10 });
        }
        let mut s = self;
        s.countdown -= 1;
        if s.countdown == -1 {
            s.open = true;
        }
        Ok(s)
    }
}

/// Go `filepath.Join` 會對結果做 `Clean`：`..` 以字面方式消去，不經作業系統解析。
/// 例如根目錄是 `missing/../pack` 時，Go 開的是 `pack/...`，即使 `missing` 不存在。
fn go_join_clean(root: &Path, name: &str) -> PathBuf {
    let joined = root.join(name);
    let mut out: Vec<Component<'_>> = Vec::new();
    for c in joined.components() {
        match c {
            Component::CurDir => {}
            Component::ParentDir => match out.last() {
                Some(Component::Normal(_)) => {
                    out.pop();
                }
                Some(Component::RootDir) | Some(Component::Prefix(_)) => {}
                _ => out.push(c),
            },
            _ => out.push(c),
        }
    }
    if out.is_empty() { PathBuf::from(".") } else { out.iter().collect() }
}

/// 讀離線匯出的四張 indexed PNG（`DATO_%03d_m%d.png`）。只接受解碼成 Go `*image.Paletted`
/// 的檔案，也就是 PNG 色彩型態 3（任何位元深度），避免由 RGBA 顏色猜回原始 VGA 索引。
///
/// Go 版的轉譯：
/// - `image.Decode` 只認得連結進執行檔的格式；凍結 Go 版的 `cmd/fd2` 與黃金匯出工具都只註冊
///   `image/png`，所以這裡只認 PNG，其他內容回報 `image: unknown format`。
/// - Go 先檢查解碼錯誤再檢查 `Close` 錯誤；Rust 的 `File` 關閉錯誤無法取得，那條分支不存在。
///
/// 刻意不照做：畸形 PNG 的判定。
/// - 原版怎麼做：`FD2.EXE` 不讀 PNG，頭像只來自 DATO.DAT；分離頭像包是重製端工具匯出的產物。
///   凍結 Go 版用標準庫 `image/png` 解碼，哪些畸形檔被接受（例如 zlib 串流後的垃圾位元組是否
///   已被 bufio 讀走、調色盤外的索引補黑色）與錯誤文字都由 Go 標準庫的實作細節決定。
/// - remake 怎麼做：用工作區已鎖定的 `png` crate 解碼，驗 CRC 與 Adler-32 並讀到 IEND。合法
///   PNG 的寬、高與每個索引像素與 Go 相同；非 PNG 輸入、目錄、非 indexed 等 dato 自身契約的
///   錯誤文字與 Go 相同；畸形 PNG 只保證多數情形同樣拒收，接受與否及錯誤文字可能不同。
/// - 為什麼：分離頭像由我們自己的工具產生並以雜湊驗證，畸形檔的取捨是 Go 標準庫的副作用，
///   不是原版遊戲行為；為重現它而移植標準庫會帶來授權標示與維護負擔。黃金差分把已知不同的
///   案例類別逐一列在 `tests/golden_dato.rs` 的 `KNOWN_DIFFERENCES`。
pub fn load_separated_resource(portrait_root: impl AsRef<Path>, resource: i64) -> Result<Vec<Frame>> {
    let root = portrait_root.as_ref();
    if root.as_os_str().is_empty() || resource < 0 {
        return Err(DatoError::SeparatedInput);
    }
    let mut frames = Vec::with_capacity(MOUTH_FRAME_COUNT);
    for mouth in 0..MOUTH_FRAME_COUNT {
        let path = go_join_clean(root, &format!("DATO_{resource:03}_m{mouth}.png"));
        let mut file = match std::fs::File::open(&path) {
            Ok(file) => file,
            Err(source) => return Err(DatoError::SeparatedOpen { resource, mouth, path, source }),
        };
        let mut data = Vec::new();
        let decoded = match file.read_to_end(&mut data) {
            Ok(_) => indexed_png::decode(&data),
            Err(e) => Err(indexed_png::read_failure(&data, e)),
        };
        let decoded = decoded.map_err(|source| DatoError::SeparatedDecode { resource, mouth, source })?;
        let Some(image) = decoded else {
            return Err(DatoError::SeparatedNotIndexed { resource, mouth });
        };
        // Go 的 Paletted 一定是 Stride==width、len(Pix)==width*height，這個檢查不會失敗；照樣保留。
        let (width, height, stride) = (image.width, image.height, image.width);
        if width <= 0 || height <= 0 || stride < width || (image.pix.len() as i64) < stride * height {
            return Err(DatoError::SeparatedGeometry { resource, mouth });
        }
        let mut pixels = vec![0u8; (width * height) as usize];
        for row in 0..height as usize {
            let (w, s) = (width as usize, stride as usize);
            pixels[row * w..(row + 1) * w].copy_from_slice(&image.pix[row * s..row * s + w]);
        }
        frames.push(Frame { width, height, pixels });
    }
    Ok(frames)
}

#[cfg(test)]
mod tests;
