//! ANI.DAT 的 AFM 動畫解碼與離線匯出的分離動畫載入。
//!
//! 移植自 Go `remake/internal/afm`（`afm.go`、`separated.go`，凍結於 v.1.1.11）。
//!
//! AFM（Animation File Manager v1.00，Lo Yuan Tsung 1993）不是逐幀點陣圖，而是 10-opcode 的
//! 增量繪圖 VM（反組譯 doc39）：每幀是一段 script，對「上一幀遺留的 framebuffer／palette」
//! 疊加（填色、RLE、局部貼圖、局部調色盤），不清空重畫。[`decode_resource`] 只供離線匯入器
//! 與 archive oracle 測試使用；正式遊戲必須使用 [`load_separated_resource`]。
//!
//! 反組譯位址（FD2.EXE linear）：播放器 `0x020421`、VM 派發 `0x36c9e`、跳表 `0x5276a`、
//! framebuffer＝VGA `0xA0000`（無雙緩衝）。VGA 6-bit palette 以 `(v<<2)|(v>>4)` 還原 8-bit。
//!
//! afm 自己的錯誤訊息（`afm: ...`）與 Go 版逐字相同，以便黃金差分直接比對；包在裡面的
//! 檔案系統、JSON、PNG 錯誤改用 Rust 函式庫的原文，不追 Go 標準庫的字串。Go 的 `string`
//! 是位元組序列，ANI.DAT 標頭的標題可能不是合法 UTF-8，所以 [`Clip::title`] 保留原始位元組。

mod separated;

pub use separated::{ANI_FILE_SIZE, ANI_MD5, ANI_SHA256, load_separated_resource};

use std::fmt;
use std::path::Path;

pub const WIDTH: usize = 320;
pub const HEIGHT: usize = 200;
const SCR_W: usize = WIDTH;
const SCR_H: usize = HEIGHT;
const FRAME_BYTES: usize = SCR_W * SCR_H; // 64000

/// Go `*image.RGBA`，範圍固定從 (0,0) 起，stride 為 `4*width`。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RgbaImage {
    pub width: usize,
    pub height: usize,
    pub pix: Vec<u8>,
}

/// 一個 AFM 資源解出的完整影格序列（已套 palette 的 RGBA）。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Clip {
    /// Go `string` 的原始位元組；分離動畫的標題來自 JSON，必為合法 UTF-8。
    pub title: Vec<u8>,
    pub header_frames: usize,
    pub frames: Vec<RgbaImage>,
    /// 320x200 palette index，每幀一份不可變快照。
    pub indexed_frames: Vec<Vec<u8>>,
    /// 對應的 768 位元組 VGA 6-bit palette 快照。
    pub palettes: Vec<Vec<u8>>,
}

/// 檔案讀取失敗：`<op> <路徑>: <std::io::Error>`。格式仿 Go `*fs.PathError`，訊息本身是 Rust 的。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IoError {
    pub op: &'static str,
    pub path: String,
    pub message: String,
}

impl IoError {
    fn new(op: &'static str, path: &Path, err: &std::io::Error) -> Self {
        IoError { op, path: path.display().to_string(), message: err.to_string() }
    }
}

impl fmt::Display for IoError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{} {}: {}", self.op, self.path, self.message)
    }
}

/// Go `os.ReadFile` 的對應：開檔失敗標 `open`，讀取失敗（例如目錄）標 `read`。
fn read_file(path: &Path) -> std::result::Result<Vec<u8>, IoError> {
    use std::io::Read;
    let mut file = std::fs::File::open(path).map_err(|e| IoError::new("open", path, &e))?;
    let mut out = Vec::new();
    file.read_to_end(&mut out).map_err(|e| IoError::new("read", path, &e))?;
    Ok(out)
}

/// 分離動畫錯誤包住的下層錯誤（Go 以 `%w` 包裝）。文字來自 Rust 函式庫，不與 Go 比對。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Cause {
    Os(IoError),
    /// `serde_json` 的語法或型別錯誤。
    Json(String),
    /// `png` crate 的解碼錯誤。
    Image(String),
}

impl fmt::Display for Cause {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Cause::Os(e) => e.fmt(f),
            Cause::Json(s) | Cause::Image(s) => f.write_str(s),
        }
    }
}

/// 錯誤種類；afm 自有訊息的 `Display` 與 Go 版的錯誤字串逐字相同。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum AfmError {
    /// 讀 .DAT 失敗（Go 版原樣回傳 `os.ReadFile` 的錯誤）。
    Os(IoError),
    /// `afm.go` 以 `errors.New` 建立的固定訊息。
    Fixed(&'static str),
    InvalidSeparatedResource(i64),
    SeparatedMetadata {
        resource: i64,
        cause: Cause,
    },
    SeparatedMetadataContract(i64),
    SeparatedFrameContract(usize),
    SeparatedFrame {
        index: usize,
        cause: Cause,
    },
    SeparatedFrameNotIndexed(usize),
    SeparatedPalette {
        index: usize,
        cause: Cause,
    },
    SeparatedPaletteContract(usize),
    SeparatedPaletteComponent(usize),
}

impl fmt::Display for AfmError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AfmError::Os(e) => e.fmt(f),
            AfmError::Fixed(s) => f.write_str(s),
            AfmError::InvalidSeparatedResource(r) => write!(f, "afm: invalid separated resource {r}"),
            AfmError::SeparatedMetadata { resource, cause } => write!(f, "afm: separated metadata {resource}: {cause}"),
            AfmError::SeparatedMetadataContract(r) => write!(f, "afm: separated metadata {r} violates the contract"),
            AfmError::SeparatedFrameContract(i) => write!(f, "afm: separated frame {i} contract is invalid"),
            AfmError::SeparatedFrame { index, cause } => write!(f, "afm: separated frame {index}: {cause}"),
            AfmError::SeparatedFrameNotIndexed(i) => {
                write!(f, "afm: separated frame {i} is not a packed 320x200 indexed PNG")
            }
            AfmError::SeparatedPalette { index, cause } => write!(f, "afm: separated palette {index}: {cause}"),
            AfmError::SeparatedPaletteContract(i) => write!(f, "afm: separated palette {i} violates the contract"),
            AfmError::SeparatedPaletteComponent(i) => write!(f, "afm: separated palette {i} has an invalid component"),
        }
    }
}

impl std::error::Error for AfmError {}

pub type Result<T> = std::result::Result<T, AfmError>;

const ERR_INDEX: &str = "afm: 資源 index 越界";
const ERR_MAGIC: &str = "afm: 缺少 LLLLLL magic";
const ERR_DIR_START: &str = "afm: 目錄起點不合理";
const ERR_DIR_RANGE: &str = "afm: 目錄越界";
const ERR_SHORT: &str = "afm: 資源過短";
const ERR_GEOMETRY: &str = "afm: 畫面幾何不是 320x200";
const ERR_RECORD: &str = "afm: 幀記錄截斷";
const ERR_SCRIPT: &str = "afm: 幀 script 截斷";
const ERR_ZERO: &str = "afm: 零幀";
const ERR_TRAILING: &str = "afm: 幀資料未恰好消費完整資源";
const ERR_RLE: &str = "afm: RLE 越界";
const ERR_RLE_RUN: &str = "afm: RLE run 值越界";
const ERR_RD8: &str = "afm: script 越界";
const ERR_RD16: &str = "afm: script 越界(u16)";
const ERR_OP1: &str = "afm: op1 越界";
const ERR_OP3: &str = "afm: op3 越界";
const ERR_OP3_DATA: &str = "afm: op3 資料越界";
const ERR_OP5: &str = "afm: op5 越界";
const ERR_OP7: &str = "afm: op7 越界";
const ERR_OP8: &str = "afm: op8 越界";
const ERR_OP9: &str = "afm: op9 越界";
const ERR_OP9_DATA: &str = "afm: op9 資料越界";
const ERR_OPCODE: &str = "afm: 未知 opcode";

fn le_u16(b: &[u8]) -> usize {
    u16::from_le_bytes([b[0], b[1]]) as usize
}

fn le_u32(b: &[u8]) -> usize {
    u32::from_le_bytes([b[0], b[1], b[2], b[3]]) as usize
}

fn read_dat(dat_path: &Path) -> Result<Vec<u8>> {
    read_file(dat_path).map_err(AfmError::Os)
}

/// ANI.DAT 的 LLLLLL 目錄項數；只供離線匯入與證據測試。
pub fn archive_resource_count(dat_path: impl AsRef<Path>) -> Result<usize> {
    let raw = read_dat(dat_path.as_ref())?;
    Ok(container_entries(&raw)?.len())
}

/// 讀取單一原始資源；只供離線匯入與證據測試。
pub fn read_archive_resource(dat_path: impl AsRef<Path>, res: i64) -> Result<Vec<u8>> {
    let raw = read_dat(dat_path.as_ref())?;
    let entries = container_entries(&raw)?;
    if res < 0 || res >= entries.len() as i64 {
        return Err(AfmError::Fixed(ERR_INDEX));
    }
    Ok(entries[res as usize].to_vec())
}

/// 開啟一個 .DAT 容器（LLLLLL magic + uint32 offset 目錄），回傳各資源切片。
fn container_entries(data: &[u8]) -> Result<Vec<&[u8]>> {
    if data.len() < 10 || &data[..6] != b"LLLLLL" {
        return Err(AfmError::Fixed(ERR_MAGIC));
    }
    let first = le_u32(&data[6..]);
    if first < 6 || first > data.len() || !(first - 6).is_multiple_of(4) {
        return Err(AfmError::Fixed(ERR_DIR_START));
    }
    let n = (first - 6) / 4;
    let mut offs: Vec<usize> = (0..n).map(|i| le_u32(&data[6 + 4 * i..])).collect();
    offs.push(data.len());
    let mut out = Vec::with_capacity(n);
    for i in 0..n {
        if offs[i] > offs[i + 1] || offs[i + 1] > data.len() {
            return Err(AfmError::Fixed(ERR_DIR_RANGE));
        }
        out.push(&data[offs[i]..offs[i + 1]]);
    }
    Ok(out)
}

/// 解一個 AFM 資源為 [`Clip`]。`res` 是容器內的資源 index。
pub fn decode_resource(dat_path: impl AsRef<Path>, res: i64) -> Result<Clip> {
    let raw = read_dat(dat_path.as_ref())?;
    let entries = container_entries(&raw)?;
    if res < 0 || res >= entries.len() as i64 {
        return Err(AfmError::Fixed(ERR_INDEX));
    }
    decode_afm(entries[res as usize])
}

/// 解單一 AFM 檔（資源）。標頭 `+0xA5`＝frameCount，`+0xAD` 起 N×(8B 幀記錄＋script)。
fn decode_afm(d: &[u8]) -> Result<Clip> {
    if d.len() < 0xad {
        return Err(AfmError::Fixed(ERR_SHORT));
    }
    // Go 版另判 `len(d) >= 0xA1`；上面已保證，標題恆取 +0x51..+0xA1。
    let title = d[0x51..0xa1].to_vec();
    let frame_count = le_u16(&d[0xa5..]);
    let width = le_u16(&d[0xa7..]);
    let height = le_u16(&d[0xa9..]);
    if width != SCR_W || height != SCR_H {
        return Err(AfmError::Fixed(ERR_GEOMETRY));
    }
    let mut clip = Clip {
        title,
        header_frames: frame_count,
        frames: Vec::new(),
        indexed_frames: Vec::new(),
        palettes: Vec::new(),
    };
    let mut pal = vec![0u8; 768];
    let mut fb = vec![0u8; FRAME_BYTES];
    let mut pos = 0xad;
    for _ in 0..frame_count {
        if pos + 8 > d.len() {
            return Err(AfmError::Fixed(ERR_RECORD));
        }
        let comp_size = le_u16(&d[pos..]);
        let cmd_count = le_u16(&d[pos + 2..]);
        pos += 8;
        if pos + comp_size > d.len() {
            return Err(AfmError::Fixed(ERR_SCRIPT));
        }
        let script = &d[pos..pos + comp_size];
        pos += comp_size;
        run_vm(script, cmd_count, &mut pal, &mut fb).map_err(AfmError::Fixed)?;
        clip.indexed_frames.push(fb.clone());
        clip.palettes.push(pal.clone());
        clip.frames.push(to_rgba(&fb, &pal));
    }
    if frame_count == 0 || clip.frames.len() != frame_count {
        return Err(AfmError::Fixed(ERR_ZERO));
    }
    if pos != d.len() {
        return Err(AfmError::Fixed(ERR_TRAILING));
    }
    Ok(clip)
}

/// index framebuffer + 6-bit palette 轉成 RGBA。Go 的 `byte` 位移會截去高位，Rust `u8` 相同。
fn to_rgba(fb: &[u8], pal: &[u8]) -> RgbaImage {
    let mut pix = vec![0u8; SCR_W * SCR_H * 4];
    for (i, &ci) in fb.iter().enumerate() {
        let base = ci as usize * 3;
        for (k, &v) in pal[base..base + 3].iter().enumerate() {
            pix[i * 4 + k] = (v << 2) | (v >> 4);
        }
        pix[i * 4 + 3] = 0xff;
    }
    RgbaImage { width: SCR_W, height: SCR_H, pix }
}

/// opcode 2／6 共用的 2-mode RLE：高 2 bit＝0b11 → run（len＝ctrl&0x3F），否則單一 literal。
fn rle2mode(data: &[u8], mut pos: usize, dst: &mut [u8], count: usize) -> std::result::Result<usize, &'static str> {
    let mut written = 0;
    while written < count {
        if pos >= data.len() {
            return Err(ERR_RLE);
        }
        let ctrl = data[pos];
        pos += 1;
        if ctrl & 0xc0 == 0xc0 {
            let n = (ctrl & 0x3f) as usize;
            if pos >= data.len() {
                return Err(ERR_RLE_RUN);
            }
            let val = data[pos];
            pos += 1;
            let mut i = 0;
            while i < n && written < count {
                dst[written] = val;
                written += 1;
                i += 1;
            }
        } else {
            dst[written] = ctrl;
            written += 1;
        }
    }
    Ok(pos)
}

/// Go `runVM` 內的 `rd8`／`rd16` 閉包；失敗時不移動讀取位置。
struct Script<'a> {
    s: &'a [u8],
    pos: usize,
}

impl Script<'_> {
    fn rd8(&mut self) -> std::result::Result<u8, &'static str> {
        if self.pos >= self.s.len() {
            return Err(ERR_RD8);
        }
        let v = self.s[self.pos];
        self.pos += 1;
        Ok(v)
    }

    fn rd16(&mut self) -> std::result::Result<usize, &'static str> {
        if self.pos + 2 > self.s.len() {
            return Err(ERR_RD16);
        }
        let v = le_u16(&self.s[self.pos..]);
        self.pos += 2;
        Ok(v)
    }
}

/// 執行 `cmd_count` 個 AFM 指令，原地改 `pal`（768B）與 `fb`（64000B）。忠實 doc39 opcode 表。
///
/// opcode 9 的偏移大於 64000 而資料足夠時，Go 版 `fb[off:end]` 切片越界會 panic；這裡同樣
/// panic，不改成錯誤回傳（黃金差分以 `recover`／`catch_unwind` 比對）。
fn run_vm(s: &[u8], cmd_count: usize, pal: &mut [u8], fb: &mut [u8]) -> std::result::Result<(), &'static str> {
    let mut sc = Script { s, pos: 0 };
    for _ in 0..cmd_count {
        let op = sc.rd8()?;
        match op {
            0 => {
                // palette 全填
                let v = sc.rd8()?;
                pal.fill(v);
            }
            1 => {
                // palette 字面載入
                if sc.pos + 768 > s.len() {
                    return Err(ERR_OP1);
                }
                pal.copy_from_slice(&s[sc.pos..sc.pos + 768]);
                sc.pos += 768;
            }
            2 => {
                // palette RLE
                sc.pos = rle2mode(s, sc.pos, pal, 768)?;
            }
            3 => {
                // palette 局部貼補：N×(idx,cnt,RGB×cnt)
                let n = sc.rd8()?;
                for _ in 0..n {
                    let idx = sc.rd8();
                    let cnt = sc.rd8();
                    let (Ok(idx), Ok(cnt)) = (idx, cnt) else {
                        return Err(ERR_OP3);
                    };
                    let (off, length) = (idx as usize * 3, cnt as usize * 3);
                    if sc.pos + length > s.len() || off + length > 768 {
                        return Err(ERR_OP3_DATA);
                    }
                    pal[off..off + length].copy_from_slice(&s[sc.pos..sc.pos + length]);
                    sc.pos += length;
                }
            }
            4 => {
                // framebuffer 全填
                let v = sc.rd8()?;
                fb.fill(v);
            }
            5 => {
                // framebuffer 字面載入
                if sc.pos + FRAME_BYTES > s.len() {
                    return Err(ERR_OP5);
                }
                fb.copy_from_slice(&s[sc.pos..sc.pos + FRAME_BYTES]);
                sc.pos += FRAME_BYTES;
            }
            6 => {
                // framebuffer RLE（主力全螢幕）
                sc.pos = rle2mode(s, sc.pos, fb, FRAME_BYTES)?;
            }
            7 => {
                // 單點繪製 ×N
                let n = sc.rd16()?;
                for _ in 0..n {
                    let off = sc.rd16();
                    let val = sc.rd8();
                    let (Ok(off), Ok(val)) = (off, val) else {
                        return Err(ERR_OP7);
                    };
                    if off < FRAME_BYTES {
                        fb[off] = val;
                    }
                }
            }
            8 => {
                // 區段填色 ×N
                let n = sc.rd16()?;
                for _ in 0..n {
                    let off = sc.rd16();
                    let length = sc.rd8();
                    let val = sc.rd8();
                    let (Ok(off), Ok(length), Ok(val)) = (off, length, val) else {
                        return Err(ERR_OP8);
                    };
                    let end = (off + length as usize).min(FRAME_BYTES);
                    // Go 的 `for i := off; i < end` 在 off >= end 時不執行。
                    if off < end {
                        fb[off..end].fill(val);
                    }
                }
            }
            9 => {
                // 區段貼圖 ×N
                let n = sc.rd16()?;
                for _ in 0..n {
                    let off = sc.rd16();
                    let length = sc.rd8();
                    let (Ok(off), Ok(length)) = (off, length) else {
                        return Err(ERR_OP9);
                    };
                    let length = length as usize;
                    let end = (off + length).min(FRAME_BYTES);
                    if sc.pos + length > s.len() {
                        return Err(ERR_OP9_DATA);
                    }
                    assert!(
                        off <= end,
                        "afm: op9 偏移 {off} 超過畫面 {end}（Go 版 fb[off:end] 在此 slice bounds out of range）"
                    );
                    fb[off..end].copy_from_slice(&s[sc.pos..sc.pos + (end - off)]);
                    sc.pos += length;
                }
            }
            _ => return Err(ERR_OPCODE),
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests;
