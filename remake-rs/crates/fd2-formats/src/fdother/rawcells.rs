//! 移植自 Go `rawcells.go`：FDOTHER 無標籤偏移庫中的未壓縮索引色圖格。

use super::{FdotherError, Result, read_le16, read_le32, read_resource_i64};
use std::path::Path;

/// FDOTHER 無標籤偏移庫形式的未壓縮索引色圖格。FDOTHER #2 用這種形式存原生動作覆蓋層。
/// 調色盤索引 0 保留目的端，與 `0x4e9e4` 相同。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RawCell {
    pub width: i64,
    pub height: i64,
    pub pixels: Vec<u8>,
}

/// 解碼一筆 width/height + width*height 逐字索引色圖格。這是 `0x4e9e4` 直接消費的資源項目
/// 形式；調色盤索引 0 在繪圖時仍是透明。
pub fn parse_raw_cell(data: &[u8]) -> Result<RawCell> {
    if data.len() < 4 {
        return Err(FdotherError::fixed("fdother: raw cell is too short"));
    }
    let width = read_le16(data, 0);
    let height = read_le16(data, 2);
    if width <= 0 || height <= 0 || width > (data.len() as i64 - 4) / height {
        return Err(FdotherError::fixed("fdother: raw cell geometry is invalid"));
    }
    Ok(RawCell { width, height, pixels: data[4..4 + (width * height) as usize].to_vec() })
}

/// 讀一筆第一個 u32 是 u32 偏移目錄結尾的原始 FDOTHER 封存項目。刻意不對 LMI1 或
/// 圖格表資源嘗試這個解析器。
pub fn decode_raw_cell_resource(dat_path: impl AsRef<Path>, resource: i64) -> Result<Vec<RawCell>> {
    parse_raw_cell_bank(&read_resource_i64(dat_path.as_ref(), resource)?)
}

/// 解析 FDOTHER #2 使用的無標籤目錄：
///
/// ```text
/// +0  u32 directory_end（= 第一格偏移）
/// +4  u32 下一格偏移
/// ...
/// 圖格：u16 width, u16 height, width*height 個原始索引像素
/// ```
///
/// 原生 `0x4e9e4` 恰好消費 width*height 位元組，所以目錄偏移定義的是圖格起點，
/// 不是壓縮串流的終點。
pub fn parse_raw_cell_bank(data: &[u8]) -> Result<Vec<RawCell>> {
    if data.len() < 4 {
        return Err(FdotherError::fixed("fdother: raw cell bank is too short"));
    }
    let len = data.len() as i64;
    let directory_end = read_le32(data, 0);
    if directory_end < 4 || directory_end % 4 != 0 || directory_end > len {
        return Err(FdotherError::fixed("fdother: invalid raw cell directory"));
    }
    let count = directory_end / 4;
    let mut cells = Vec::with_capacity(count as usize);
    let mut previous = directory_end;
    for i in 0..count {
        let off = read_le32(data, 4 * i);
        if off < previous || off + 4 > len {
            return Err(FdotherError::message(format!("fdother: raw cell {i} offset is invalid")));
        }
        let width = read_le16(data, off);
        let height = read_le16(data, off + 2);
        if width <= 0 || height <= 0 || width > (len - off - 4) / height {
            return Err(FdotherError::message(format!("fdother: raw cell {i} geometry is invalid")));
        }
        let end = off + 4 + width * height;
        if end > len {
            return Err(FdotherError::message(format!("fdother: raw cell {i} pixels exceed resource")));
        }
        cells.push(RawCell { width, height, pixels: data[(off + 4) as usize..end as usize].to_vec() });
        previous = off;
    }
    Ok(cells)
}

impl RawCell {
    fn check(&self) -> Result<()> {
        if self.width <= 0 || self.height <= 0 || self.pixels.len() as i64 != self.width.wrapping_mul(self.height) {
            return Err(FdotherError::fixed("fdother: invalid raw cell"));
        }
        Ok(())
    }

    /// 套用 `0x4e9e4` 的直接索引複製。來源像素 0 保留目的端；這個常式沒有推論出的調色盤轉換。
    ///
    /// Go 在 `stride < x+W` 通過但 stride 為 0 時（x+W 繞回成負值）於 `len(dst)/stride`
    /// 除以零 panic；Rust 一樣 panic。
    pub fn blit_at(&self, dst: &mut [u8], stride: i64, x: i64, y: i64) -> Result<()> {
        self.check()?;
        let n = dst.len() as i64;
        if x < 0
            || y < 0
            || stride < x.wrapping_add(self.width)
            || y > n / stride
            || self.height > n.wrapping_sub(y.wrapping_mul(stride)) / stride
        {
            return Err(FdotherError::fixed("fdother: raw cell destination is too small"));
        }
        for row in 0..self.height {
            for col in 0..self.width {
                let v = self.pixels[(row * self.width + col) as usize];
                if v != 0 {
                    dst[(y + row).wrapping_mul(stride).wrapping_add(x + col) as usize] = v;
                }
            }
        }
        Ok(())
    }

    /// [`blit_at`](Self::blit_at) 的裁邊版本，給起點落在視窗外的原生滑動用。
    ///
    /// 可見游標（`[0x53AB9]`／`[0x53ABD]`）在原版是自由的 dword，八個寫入端沒有一個檢查
    /// 範圍：走行步進在鏡頭還能捲時照樣遞減，收據量到過 -1。對白收框的滑動終點就是那個
    /// 值，所以終點在畫面外是合法狀態，不是錯誤。嚴格的 `blit_at` 契約維持不變。
    pub fn blit_at_clipped(&self, dst: &mut [u8], stride: i64, x: i64, y: i64) -> Result<()> {
        self.check()?;
        let n = dst.len() as i64;
        if stride <= 0 || n < stride {
            return Err(FdotherError::fixed("fdother: clipped raw cell surface is invalid"));
        }
        // 整格落在畫面外是合法的中間狀態，不是錯誤。滑動終點是可見游標，而可見游標可以是
        // 負的；x = 24×(-3)+4 = -68 時整格在左界外，那一幀原版也只是什麼都沒畫。
        if y >= n / stride || y.wrapping_add(self.height) <= 0 || x >= stride || x.wrapping_add(self.width) <= 0 {
            return Ok(());
        }
        for row in 0..self.height {
            let dy = y.wrapping_add(row);
            if dy < 0 || dy >= n / stride {
                continue;
            }
            for col in 0..self.width {
                let dx = x.wrapping_add(col);
                if dx < 0 || dx >= stride {
                    continue;
                }
                let v = self.pixels[(row * self.width + col) as usize];
                if v != 0 {
                    dst[(dy * stride + dx) as usize] = v;
                }
            }
        }
        Ok(())
    }

    /// 重現 `0x4e9bb` 的直接逐列複製。與 [`blit_at`](Self::blit_at) 不同，0 也當逐字索引像素
    /// 寫入，因為這是 FDOTHER#5 對白框路徑，不是透明的 `0x4e9e4` 路徑。
    pub fn blit_opaque_at_offset(&self, dst: &mut [u8], stride: i64, offset: i64) -> Result<()> {
        let n = dst.len() as i64;
        if self.width <= 0
            || self.height <= 0
            || self.pixels.len() as i64 != self.width.wrapping_mul(self.height)
            || stride <= 0
            || offset < 0
            || (offset % stride).wrapping_add(self.width) > stride
            || offset > n
            || self.height > (n - offset) / stride
        {
            return Err(FdotherError::fixed("fdother: opaque raw cell destination is too small"));
        }
        let w = self.width as usize;
        for row in 0..self.height as usize {
            let at = offset as usize + row * stride as usize;
            dst[at..at + w].copy_from_slice(&self.pixels[row * w..(row + 1) * w]);
        }
        Ok(())
    }
}
