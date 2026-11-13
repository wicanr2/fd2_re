//! 移植自 Go `remap.go`：FD2.EXE `0x219ad`／`0x22046` 的索引緩衝 LUT 重映射。

use super::{FdotherError, Result};

/// 已還原的 FD2.EXE `0x219ad` 幾何 ABI。刻意只是索引緩衝原語：描述子載入、緩衝呈現與
/// 時序歸呼叫端。`clip_width` 是原生可見列寬（目前還原的呼叫端都是 0x138），可以比 stride 小。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct RadialLUTRemap {
    pub center_x: i64,
    pub center_y: i64,
    pub radius: i64,
    pub scale: i64,
    pub start_y: i64,
    /// 不含。
    pub end_y: i64,
    pub clip_width: i64,
}

/// FD2.EXE `0x22046` 最後一次只做 LUT 的處理。水平半徑已由呼叫端還原為
/// trunc(native_radius*1.6)；與 [`RadialLUTRemap`] 不同，它不推論那個換算，也不吸收原生
/// 兩次處理之間的 `0x127a9` 重繪。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct CenteredRectLUTRemap {
    pub center_x: i64,
    pub horizontal_radius: i64,
    pub start_y: i64,
    /// 不含。
    pub end_y: i64,
    pub clip_width: i64,
}

/// 套用 `0x219ad` 的就地調色盤索引重映射。
///
/// 原生使用 `0x4db9c(lut, count, pixels)`，所以每個選到的來源位元組都換成 `lut[source]`。
/// 對嚴格位於 CenterY±Radius 內的每一列，映射裁切後的區間
/// CenterX±trunc(sqrt(radius²-dy²)*scale/10)。原版呼叫 `__CHP`（`0x377a4`），其暫時的 x87
/// 控制字選擇向零捨入；這裡所有量都非負，所以還原的結果就是數學上的 floor。
pub fn apply_radial_lut_remap(dst: &mut [u8], stride: i64, lut: &[u8], spec: RadialLUTRemap) -> Result<()> {
    if lut.len() != 256 {
        return Err(FdotherError::fixed("fdother: LUT must have 256 entries"));
    }
    if stride <= 0
        || spec.clip_width <= 0
        || spec.clip_width > stride
        || spec.center_x < 0
        || spec.center_x >= spec.clip_width
        || spec.center_y < 0
        || spec.radius <= 0
        || spec.radius > 0x7fff
        || spec.scale < 0
        || spec.scale > 0x7fff
        || spec.start_y < 0
        || spec.end_y < spec.start_y
        || spec.end_y > dst.len() as i64 / stride
    {
        return Err(FdotherError::fixed("fdother: invalid radial LUT remap geometry"));
    }
    for y in spec.start_y..spec.end_y {
        // center_y 沒有上限；Go 的 int 減法在極端值會繞回。
        let dy = spec.center_y.wrapping_sub(y).wrapping_abs();
        // 0x219ad 的 jle/jge 讓半徑邊界上的兩列不變。
        if dy >= spec.radius {
            continue;
        }
        let span = (((spec.radius * spec.radius - dy * dy) as f64).sqrt() * spec.scale as f64 / 10.0) as i64;
        let left = (spec.center_x - span).max(0);
        let right = (spec.center_x + span).min(spec.clip_width);
        remap_row(dst, y * stride, spec.clip_width, left, right, lut);
    }
    Ok(())
}

/// `row := dst[y*stride : y*stride+clip]`，再就地映射 `row[left..right]`。
fn remap_row(dst: &mut [u8], start: i64, clip: i64, left: i64, right: i64, lut: &[u8]) {
    let row = &mut dst[start as usize..(start + clip) as usize];
    for x in left..right {
        row[x as usize] = lut[row[x as usize] as usize];
    }
}

/// 套用 `0x22046` 最後的就地 LUT 處理：每個要求的列上，一段裁切後置中的水平區間。
/// 與兩次放射狀處理分開，因為 `0x22046` 在它們之間呼叫不透明重繪；呼叫端不可誤以為
/// 這就是整個原生編排。
pub fn apply_centered_rect_lut_remap(
    dst: &mut [u8],
    stride: i64,
    lut: &[u8],
    spec: CenteredRectLUTRemap,
) -> Result<()> {
    if lut.len() != 256 {
        return Err(FdotherError::fixed("fdother: LUT must have 256 entries"));
    }
    if stride <= 0
        || spec.clip_width <= 0
        || spec.clip_width > stride
        || spec.center_x < 0
        || spec.center_x >= spec.clip_width
        || spec.horizontal_radius < 0
        || spec.horizontal_radius > 0x7fff
        || spec.start_y < 0
        || spec.end_y < spec.start_y
        || spec.end_y > dst.len() as i64 / stride
    {
        return Err(FdotherError::fixed("fdother: invalid centered LUT rectangle"));
    }
    let left = (spec.center_x - spec.horizontal_radius).max(0);
    let right = (spec.center_x + spec.horizontal_radius).min(spec.clip_width);
    for y in spec.start_y..spec.end_y {
        remap_row(dst, y * stride, spec.clip_width, left, right, lut);
    }
    Ok(())
}
