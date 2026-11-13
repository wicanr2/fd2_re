//! 移植自 Go `frame.go`：FDOTHER 圖格表、單幀與 `0x4e63d`／`0x4e583` 四模式 RLE。
//!
//! 容器目錄（`ArchiveEntry` 等）已拆到 [`crate::archive`]，模組根轉出同名函式。

use crate::archive::{ArchiveError, read_resource};
use crate::fdicon::FdiconError;
use std::fmt;
use std::path::Path;
use std::sync::Arc;

/// fdother 套件的錯誤。`Display` 與 Go 版的錯誤字串逐字相同，例外只有 [`External`]
/// （Go 包裝標準函式庫錯誤的地方）與 `Archive(ArchiveError::Io)`：前綴相同，內層是 Rust
/// 函式庫的文字。
///
/// [`External`]: FdotherError::External
#[derive(Debug)]
pub enum FdotherError {
    /// Go 版自己的錯誤文字（`errors.New`、`fmt.Errorf`，含包裝本套件錯誤的 `%w`）。
    Contract(String),
    /// 原樣傳回的封存讀取錯誤（Go `ReadResource`／`ArchiveEntry`）。
    Archive(ArchiveError),
    /// 原樣傳回的 fdicon 錯誤（Go `fdicon.Parse`、`fdicon.Sprite.BlitAt`）。
    Fdicon(FdiconError),
    /// Go 以 `%w` 包裝 `os`、`encoding/json`、`image/png` 錯誤的地方。`context` 是 Go 版自己的
    /// 前綴（含結尾的 `: `，可能是空字串），`detail` 是 Rust 函式庫的錯誤文字。
    External { context: String, detail: String },
}

impl FdotherError {
    pub(super) fn fixed(message: &'static str) -> FdotherError {
        FdotherError::Contract(message.to_string())
    }

    pub(super) fn message(message: String) -> FdotherError {
        FdotherError::Contract(message)
    }

    pub(super) fn external(context: impl Into<String>, detail: String) -> FdotherError {
        FdotherError::External { context: context.into(), detail }
    }

    /// Go 版自己的錯誤文字前綴；整段都是 Go 文字時就是完整訊息。
    pub fn go_prefix(&self) -> String {
        match self {
            FdotherError::External { context, .. } => context.clone(),
            FdotherError::Archive(ArchiveError::Io { .. }) => String::new(),
            other => other.to_string(),
        }
    }
}

impl fmt::Display for FdotherError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            FdotherError::Contract(message) => f.write_str(message),
            FdotherError::Archive(e) => write!(f, "{e}"),
            FdotherError::Fdicon(e) => write!(f, "{e}"),
            FdotherError::External { context, detail } => write!(f, "{context}{detail}"),
        }
    }
}

impl std::error::Error for FdotherError {}

impl From<ArchiveError> for FdotherError {
    fn from(e: ArchiveError) -> Self {
        FdotherError::Archive(e)
    }
}

impl From<FdiconError> for FdotherError {
    fn from(e: FdiconError) -> Self {
        FdotherError::Fdicon(e)
    }
}

pub type Result<T> = std::result::Result<T, FdotherError>;

/// Go `ReadResource(path, resource int)`：負的 resource 在 Go 是讀檔、驗目錄之後才回報
/// `archive resource is out of range`；轉成 `usize::MAX` 走同一條路徑，錯誤順序相同。
pub(super) fn read_resource_i64(path: &Path, resource: i64) -> Result<Vec<u8>> {
    Ok(read_resource(path, usize::try_from(resource).unwrap_or(usize::MAX))?)
}

/// Go `Frame.onBlit`：`TextPlaceholder` 留下的展示回呼，參數是寫入點的 x／y。
pub type BlitCallback = Arc<dyn Fn(i64, i64) + Send + Sync>;

/// 一個 FDOTHER 圖格表描述子。X/Y 是資源內嵌的目的端偏移；`pixels` 以小端序
/// Width/Height 開頭。
///
/// `indexed` 與 `mask` 是分離素材 loader 使用的已解碼表示。`mask` 只接受 0（保留目的端）
/// 或 255（寫入 `indexed`）；原始封存解析器只填 `pixels`，兩種表示不可同時存在。
///
/// Go 的函式值欄位 `onBlit` 改成 `Option<Arc<dyn Fn>>`；Go 的 `Frame` 因含函式值而不能用
/// `==` 比較，這裡同樣不實作 `PartialEq`。
#[derive(Clone, Default)]
pub struct Frame {
    pub x: i64,
    pub y: i64,
    pub width: i64,
    pub height: i64,
    pub pixels: Vec<u8>,
    pub indexed: Vec<u8>,
    pub mask: Vec<u8>,
    on_blit: Option<BlitCallback>,
}

impl fmt::Debug for Frame {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("Frame")
            .field("x", &self.x)
            .field("y", &self.y)
            .field("width", &self.width)
            .field("height", &self.height)
            .field("pixels", &self.pixels.len())
            .field("indexed", &self.indexed.len())
            .field("mask", &self.mask.len())
            .field("on_blit", &self.on_blit.is_some())
            .finish()
    }
}

fn le16(data: &[u8], at: usize) -> i64 {
    u16::from_le_bytes([data[at], data[at + 1]]) as i64
}

fn le32(data: &[u8], at: usize) -> i64 {
    u32::from_le_bytes([data[at], data[at + 1], data[at + 2], data[at + 3]]) as i64
}

/// 讀 `data[at..]` 開頭的小端序 u16／u32；給同模組其他檔案用。
pub(super) fn read_le16(data: &[u8], at: i64) -> i64 {
    le16(data, at as usize)
}

pub(super) fn read_le32(data: &[u8], at: i64) -> i64 {
    le32(data, at as usize)
}

impl Frame {
    /// 已解碼的分離表示（`indexed`／`mask`）。
    pub fn separated(x: i64, y: i64, width: i64, height: i64, indexed: Vec<u8>, mask: Vec<u8>) -> Frame {
        Frame { x, y, width, height, pixels: Vec::new(), indexed, mask, on_blit: None }
    }

    /// 原始 `0x4e63d` 位元組表示。
    pub fn raw(x: i64, y: i64, width: i64, height: i64, pixels: Vec<u8>) -> Frame {
        Frame { x, y, width, height, pixels, indexed: Vec::new(), mask: Vec::new(), on_blit: None }
    }

    /// 是否帶有 `TextPlaceholder` 的展示回呼。
    pub fn has_blit_callback(&self) -> bool {
        self.on_blit.is_some()
    }

    /// 先驗證原圖，再把墨色換成有範圍的展示回呼；不改序列化的素材資料。
    ///
    /// Go 版傳 nil 回呼時拒收；Rust 以型別排除 nil，該分支不存在。驗證時呼叫的是本身的
    /// `blit_at`，所以對已是占位圖的 frame 再做一次，原本的回呼會在驗證時被呼叫一次（同 Go）。
    pub fn text_placeholder(&self, paint: impl Fn(i64, i64) + Send + Sync + 'static) -> Result<Frame> {
        let (x, y, w, h) = (self.x, self.y, self.width, self.height);
        if x < 0 || y < 0 || w <= 0 || h <= 0 || x.wrapping_add(w) > 320 || y.wrapping_add(h) > 200 {
            return Err(FdotherError::fixed("fdother: invalid text placeholder"));
        }
        // 上面的加法在 Go 會繞回（例如 Y 接近 int 上限）而通過檢查；這裡的乘加照樣繞回，
        // 負長度時與 Go 的 make 一樣 panic。
        let stride = x.wrapping_add(w);
        self.blit_at(&mut vec![0u8; stride.wrapping_mul(y.wrapping_add(h)) as usize], stride, 0, -1)?;
        let n = w.wrapping_mul(h) as usize;
        Ok(Frame {
            x,
            y,
            width: w,
            height: h,
            pixels: Vec::new(),
            indexed: vec![0; n],
            mask: vec![0; n],
            on_blit: Some(Arc::new(paint)),
        })
    }

    /// 對 dst 套用 FD2.EXE `0x4e63d` 的透明（-1）分支。原版另有兩種非透明的調色盤重映射
    /// 模式；目前沒有已驗證的結局呼叫端使用，所以拒收而不是自創規則。
    pub fn blit(&self, dst: &mut [u8], stride: i64, transparent: i64) -> Result<()> {
        self.blit_at(dst, stride, 0, transparent)
    }

    /// 同一個已驗證的透明解碼器，加上目的端位元組原點（用於 640 寬的原生結局工作緩衝）。
    pub fn blit_at(&self, dst: &mut [u8], stride: i64, base: i64, transparent: i64) -> Result<()> {
        if transparent != -1 {
            return Err(FdotherError::fixed("fdother: only verified transparent blit is supported"));
        }
        if !self.geometry_ok(stride) {
            return Err(FdotherError::fixed("fdother: invalid destination geometry"));
        }
        if !self.destination_ok(dst.len() as i64, stride, base) {
            return Err(FdotherError::fixed("fdother: destination is too small"));
        }
        if self.is_separated() {
            self.blit_separated(dst, stride, base, |v| v)?;
            if let Some(paint) = &self.on_blit {
                paint(base.wrapping_rem(stride).wrapping_add(self.x), base.wrapping_div(stride).wrapping_add(self.y));
            }
            return Ok(());
        }
        self.blit_rle(dst, stride, base, "", |v| v)
    }

    /// 把已驗證的 0x4E63D 串流轉成不遺失透明語意的索引色與遮罩。兩個不同的哨兵值可以
    /// 區分「保留目的端」與不透明的調色盤 254／255。
    pub fn indexed_layers(&self) -> Result<(Vec<u8>, Vec<u8>)> {
        if self.is_separated() {
            self.check_separated()?;
            return Ok((self.indexed.clone(), self.mask.clone()));
        }
        // Go `make([]byte, w*h)`：負長度在 Go 是執行期 panic，這裡轉成 usize 後同樣 panic。
        let n = self.width.wrapping_mul(self.height) as usize;
        let mut a = vec![0xffu8; n];
        let mut b = vec![0xfeu8; n];
        let mut flat = self.clone();
        flat.x = 0;
        flat.y = 0;
        flat.blit(&mut a, self.width, -1)?;
        flat.blit(&mut b, self.width, -1)?;
        let mut indexed = vec![0u8; n];
        let mut mask = vec![0u8; n];
        for i in 0..n {
            if a[i] == 0xff && b[i] == 0xfe {
                continue;
            }
            if a[i] != b[i] {
                return Err(FdotherError::fixed("fdother: inconsistent decoded indexed pixel"));
            }
            indexed[i] = a[i];
            mask[i] = 0xff;
        }
        Ok((indexed, mask))
    }

    /// 重現 `0x4e583`：解同一種四模式串流，但每個寫入的調色盤索引都經過完整的 256 位元組
    /// 對照表。透明跳過仍保留目的端，抖色仍只寫每對的第二個像素。
    pub fn blit_lut_at(&self, dst: &mut [u8], stride: i64, base: i64, lut: &[u8]) -> Result<()> {
        if lut.len() != 256 {
            return Err(FdotherError::fixed("fdother: LUT blit requires 256 entries"));
        }
        if !self.geometry_ok(stride) {
            return Err(FdotherError::fixed("fdother: invalid LUT destination geometry"));
        }
        if !self.destination_ok(dst.len() as i64, stride, base) {
            return Err(FdotherError::fixed("fdother: LUT destination is too small"));
        }
        if self.is_separated() {
            return self.blit_separated(dst, stride, base, |v| lut[v as usize]);
        }
        self.blit_rle(dst, stride, base, "LUT ", |v| lut[v as usize])
    }

    fn is_separated(&self) -> bool {
        !self.indexed.is_empty() || !self.mask.is_empty()
    }

    fn geometry_ok(&self, stride: i64) -> bool {
        !(self.x < 0 || self.y < 0 || self.width <= 0 || self.height <= 0 || stride < self.x.wrapping_add(self.width))
    }

    /// `base < 0 || base > len(dst) || Y > (len-base-X)/stride || H > (len-base)/stride-Y` 的反面。
    /// 通過 [`geometry_ok`](Self::geometry_ok) 後 stride 仍可能因 Go 的 int 繞回而是 0，Go 在此
    /// 除以零 panic，Rust 一樣。
    fn destination_ok(&self, n: i64, stride: i64, base: i64) -> bool {
        !(base < 0
            || base > n
            || self.y > (n - base).wrapping_sub(self.x) / stride
            || self.height > ((n - base) / stride).wrapping_sub(self.y))
    }

    fn check_separated(&self) -> Result<()> {
        if !self.pixels.is_empty()
            || self.indexed.len() as i64 != self.width.wrapping_mul(self.height)
            || self.mask.len() != self.indexed.len()
        {
            return Err(FdotherError::fixed("fdother: invalid separated indexed frame"));
        }
        Ok(())
    }

    fn blit_separated(&self, dst: &mut [u8], stride: i64, base: i64, map: impl Fn(u8) -> u8) -> Result<()> {
        self.check_separated()?;
        for y in 0..self.height {
            for x in 0..self.width {
                let i = (y * self.width + x) as usize;
                match self.mask[i] {
                    0 => {}
                    255 => {
                        let at =
                            base.wrapping_add(self.y.wrapping_add(y).wrapping_mul(stride)).wrapping_add(self.x + x);
                        dst[at as usize] = map(self.indexed[i]);
                    }
                    _ => return Err(FdotherError::fixed("fdother: separated mask is not binary")),
                }
            }
        }
        Ok(())
    }

    /// `0x4e63d`／`0x4e583` 共用的四模式解碼；`tag` 是 LUT 版錯誤訊息多出的 `LUT `。
    fn blit_rle(&self, dst: &mut [u8], stride: i64, base: i64, tag: &str, map: impl Fn(u8) -> u8) -> Result<()> {
        let pixels = &self.pixels;
        let mut pos = 4usize; // 0x4e63d 先從資料吃掉 Width/Height 再進 RLE。
        for y in 0..self.height {
            let row = base.wrapping_add(self.y.wrapping_add(y).wrapping_mul(stride)).wrapping_add(self.x);
            let mut written = 0i64;
            while written < self.width {
                if pos >= pixels.len() {
                    return Err(FdotherError::message(format!("fdother: {tag}frame RLE ends in row {y}")));
                }
                let ctrl = pixels[pos];
                pos += 1;
                let count = (ctrl & 0x3f) as i64 + 1;
                let mode = ctrl >> 6;
                let advance = if mode == 1 { count * 2 } else { count };
                if written + advance > self.width {
                    return Err(FdotherError::message(format!("fdother: {tag}frame RLE overruns row {y}")));
                }
                let at = |i: i64| row.wrapping_add(written + i) as usize;
                match mode {
                    // 色彩連續段
                    0 => {
                        if pos >= pixels.len() {
                            return Err(FdotherError::message(format!("fdother: {tag}colour run lacks a value")));
                        }
                        let v = map(pixels[pos]);
                        pos += 1;
                        for i in 0..count {
                            dst[at(i)] = v;
                        }
                    }
                    // 抖色：保留、上色、保留、上色……
                    1 => {
                        if pos >= pixels.len() {
                            return Err(FdotherError::message(format!("fdother: {tag}dither run lacks a value")));
                        }
                        let v = map(pixels[pos]);
                        pos += 1;
                        for i in 0..count {
                            dst[at(2 * i + 1)] = v;
                        }
                    }
                    // 逐字
                    2 => {
                        let c = count as usize;
                        if pos + c > pixels.len() {
                            return Err(FdotherError::message(format!("fdother: {tag}literal run exceeds frame data")));
                        }
                        for i in 0..count {
                            dst[at(i)] = map(pixels[pos + i as usize]);
                        }
                        pos += c;
                    }
                    // 透明跳過：目的端不變
                    _ => {}
                }
                written += advance;
            }
        }
        Ok(())
    }
}

/// 開啟玩家提供的 LLLLLL .DAT 封存並解析一筆原始 FDOTHER 資源。對應 `0x111ba` 只讀容器的
/// 邊界：這裡不轉換也不解壓。
pub fn decode_resource(dat_path: impl AsRef<Path>, resource: i64) -> Result<Vec<Frame>> {
    parse_frames(&read_resource_i64(dat_path.as_ref(), resource)?)
}

/// 讀一筆內容是 width/height 標頭直接接 `0x4e63d` 四模式 RLE 的封存項目。BG.DAT 用這種
/// 形式；與 [`parse_frames`] 分開，以免在原生 loader 沒有偏移表的地方自創一張。
pub fn decode_archive_single_frame(dat_path: impl AsRef<Path>, resource: i64) -> Result<Frame> {
    parse_single_frame(&read_resource_i64(dat_path.as_ref(), resource)?)
}

/// 解析 FD2 `0x111ba` loader 回傳的原始封存項目。只接受 `0x2935b` 會用到的欄位；
/// 偏移與圖格範圍錯誤一律拒收，不猜測。
pub fn parse_frames(data: &[u8]) -> Result<Vec<Frame>> {
    if data.len() < 8 {
        return Err(FdotherError::fixed("fdother: frame table is too short"));
    }
    let len = data.len() as i64;
    let n = le16(data, 0);
    if n == 0 || 8 + 4 * n > len {
        return Err(FdotherError::fixed("fdother: invalid frame count/table"));
    }
    let mut frames = Vec::with_capacity(n as usize);
    let mut previous = 8 + 4 * n;
    for i in 0..n {
        let off = le32(data, (8 + 4 * i) as usize);
        let end = if i + 1 < n { le32(data, (8 + 4 * (i + 1)) as usize) } else { len };
        if off < previous || off + 13 > end || end > len {
            return Err(FdotherError::message(format!("fdother: frame {i} offset is invalid")));
        }
        let o = off as usize;
        let width = le16(data, o + 9);
        let height = le16(data, o + 11);
        if width == 0 || height == 0 {
            return Err(FdotherError::message(format!("fdother: frame {i} has empty dimensions")));
        }
        frames.push(Frame::raw(le16(data, o), le16(data, o + 2), width, height, data[o + 9..end as usize].to_vec()));
        previous = off;
    }
    Ok(frames)
}

/// 解析偏移 0 就是 Width/Height、沒有偏移表的原始 `0x4e63d` 資料（結局 FDOTHER #56 的形式）。
pub fn parse_single_frame(data: &[u8]) -> Result<Frame> {
    if data.len() < 4 {
        return Err(FdotherError::fixed("fdother: single frame is too short"));
    }
    let (w, h) = (le16(data, 0), le16(data, 2));
    if w == 0 || h == 0 {
        return Err(FdotherError::fixed("fdother: single frame has empty dimensions"));
    }
    Ok(Frame::raw(0, 0, w, h, data.to_vec()))
}
