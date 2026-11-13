//! 移植自 Go `palette.go`：VGA 6 位元 DAC 調色盤與 `0x11df2`／`0x11d40` 的調色盤增減。
//!
//! Go 的 `color.Palette` 是 `color.Color` 介面的切片；這裡固定為 [`Nrgba`]（Go `color.NRGBA`），
//! 本檔產生的調色盤也只會有這個型別。`image.Paletted` 對應 [`PalettedImage`]。

use super::{FdotherError, RawCell, Result};

/// Go `color.NRGBA`：未預乘 alpha 的 8 位元 RGBA。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Nrgba {
    pub r: u8,
    pub g: u8,
    pub b: u8,
    pub a: u8,
}

impl Nrgba {
    /// Go `color.NRGBA.RGBA()`：16 位元、已預乘 alpha 的分量。
    pub fn rgba(self) -> (u32, u32, u32, u32) {
        let a = self.a as u32;
        let channel = |v: u8| {
            let v = v as u32;
            (v | v << 8) * a / 0xff
        };
        (channel(self.r), channel(self.g), channel(self.b), a | a << 8)
    }
}

/// Go `color.Palette`（元素固定為 `color.NRGBA`）。
pub type Palette = Vec<Nrgba>;

/// Go `*image.Paletted`，原點固定為 (0,0)。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct PalettedImage {
    pub width: i64,
    pub height: i64,
    pub stride: i64,
    pub pix: Vec<u8>,
    pub palette: Palette,
}

impl PalettedImage {
    /// Go `(*image.Paletted).ColorIndexAt`：範圍外回傳 0。
    pub fn color_index_at(&self, x: i64, y: i64) -> u8 {
        if x < 0 || y < 0 || x >= self.width || y >= self.height {
            return 0;
        }
        self.pix[(y * self.stride + x) as usize]
    }
}

/// 把 FDOTHER #0 的 256×RGB 6 位元 DAC 資料轉成不透明的 VGA 調色盤。透明是個別原生 blitter
/// 或遮罩的性質，不是 DAC 索引的；完整的 mode-13h 畫面可以合法地含有可見的索引 0。
pub fn parse_vga_palette(data: &[u8]) -> Result<Palette> {
    if data.len() != 256 * 3 {
        return Err(FdotherError::fixed("fdother: VGA palette must contain 768 bytes"));
    }
    let mut palette = Vec::with_capacity(256);
    for &[r, g, b] in data.as_chunks::<3>().0 {
        if r > 63 || g > 63 || b > 63 {
            return Err(FdotherError::fixed("fdother: VGA palette component exceeds six bits"));
        }
        palette.push(Nrgba { r: (r << 2) | (r >> 4), g: (g << 2) | (g >> 4), b: (b << 2) | (b >> 4), a: 0xff });
    }
    Ok(palette)
}

/// 把呼叫端擁有的 6 位元 VGA DAC 快照轉成主機呈現端使用的不透明調色盤。索引畫面不動；
/// 原生調色盤效果只在兩次呈現之間更新 dac。
pub fn vga_palette_from_dac(dac: &[u8]) -> Result<Palette> {
    parse_vga_palette(dac)
}

fn palette_range_ok(dac: &[u8], baseline: &[u8], start: i64, end: i64, delta: i64) -> bool {
    // Go：len(dac) != 768 || len(baseline) != 768 || start < 0 || end < start || end > 255 ||
    // delta < 0 || delta > 255 的反面。
    dac.len() == 256 * 3
        && baseline.len() == 256 * 3
        && start >= 0
        && (start..=255).contains(&end)
        && (0..=255).contains(&delta)
}

/// 重現 `0x11df2`：每個分量都從 `[0x53a65]` 的不變調色盤讀出、加上 delta，再以 6 位元上限
/// 夾住後寫入呼叫端擁有的 DAC。所以這是以基準推導的範圍寫入，不是在目前 DAC 上累加。
/// 已還原的呼叫端都用非負的位元組大小 delta。特別是 `0x35822` 傳 255，把每個基準分量飽和成
/// 白色；原始引數不可正規化成 63，因為它是處理器證據的一部分。不支援的有號或更寬變體
/// 失敗即關閉。
pub fn apply_vga_palette_delta(dac: &mut [u8], baseline: &[u8], start: i64, end: i64, delta: i64) -> Result<()> {
    if !palette_range_ok(dac, baseline, start, end, delta) {
        return Err(FdotherError::fixed("fdother: invalid VGA palette delta input"));
    }
    for i in (start * 3) as usize..=(end * 3 + 2) as usize {
        dac[i] = (baseline[i] as i64 + delta).min(63) as u8;
    }
    Ok(())
}

/// 重現 `0x11d40`。每個選到的分量從不變的基準讀出、減去位元組大小的 delta，並夾在 0。
/// 輸入不合法時整體不變。
pub fn apply_vga_palette_subtraction(dac: &mut [u8], baseline: &[u8], start: i64, end: i64, delta: i64) -> Result<()> {
    if !palette_range_ok(dac, baseline, start, end, delta) {
        return Err(FdotherError::fixed("fdother: invalid VGA palette subtraction input"));
    }
    for i in (start * 3) as usize..=(end * 3 + 2) as usize {
        dac[i] = (baseline[i] as i64 - delta).max(0) as u8;
    }
    Ok(())
}

impl RawCell {
    /// 回傳直接索引原始圖格的影像形式。索引 0 維持透明，與原生 `0x4e9e4` 保留目的端的
    /// blitter 一致，而不是把透明像素變成不透明的黑色。
    ///
    /// Go 對索引 0 取 `RGBA()` 的預乘值右移 8 位元；調色盤項 alpha 不是 0xff 時顏色因此變暗，
    /// 這裡照做。
    pub fn paletted(&self, palette: &[Nrgba]) -> Result<PalettedImage> {
        if self.width <= 0 || self.height <= 0 || self.pixels.len() as i64 != self.width.wrapping_mul(self.height) {
            return Err(FdotherError::fixed("fdother: invalid raw cell"));
        }
        if palette.len() < 256 {
            return Err(FdotherError::fixed("fdother: palette has fewer than 256 entries"));
        }
        let mut cell_palette = palette.to_vec();
        let (r, g, b, _) = cell_palette[0].rgba();
        cell_palette[0] = Nrgba { r: (r >> 8) as u8, g: (g >> 8) as u8, b: (b >> 8) as u8, a: 0 };
        Ok(PalettedImage {
            width: self.width,
            height: self.height,
            stride: self.width,
            pix: self.pixels.clone(),
            palette: cell_palette,
        })
    }
}
