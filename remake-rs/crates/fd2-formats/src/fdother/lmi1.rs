//! 移植自 Go `lmi1.go`：FDOTHER #5 等 LMI1 UI 容器與 `0x4e916` 高位連續段編碼。

use super::{FdotherError, Result, read_le16, read_le32, read_resource_i64};
use std::path::Path;

/// FDOTHER.DAT 資源 #5 的一個索引色 UI 圖格。像素以列為主序解碼。透明是呼叫端繪圖
/// 原語的性質，不是 `0x4e916` 解碼器的。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LMI1Entry {
    pub width: i64,
    pub height: i64,
    pub pixels: Vec<u8>,
}

/// 解碼 `0x4e8af` 呼叫端使用的 width/height + `0x4e916` 高位連續段資料。刻意與
/// [`parse_lmi1`] 分開：原生場景資源可能外包一層 LLLLLL 目錄，但每格的編碼相同。
pub fn parse_opaque_run_cell(data: &[u8]) -> Result<LMI1Entry> {
    if data.len() < 4 {
        return Err(FdotherError::fixed("fdother: opaque-run cell is too short"));
    }
    let w = read_le16(data, 0);
    let h = read_le16(data, 2);
    if w <= 0 || h <= 0 {
        return Err(FdotherError::fixed("fdother: opaque-run cell has empty dimensions"));
    }
    let pixels = decode_lmi1_pixels(&data[4..], w * h).map_err(FdotherError::fixed)?;
    Ok(LMI1Entry { width: w, height: h, pixels })
}

/// 讀一筆玩家提供的 FDOTHER 封存項目並當成 LMI1 UI 容器解碼。封存邊界保持明確，
/// 與 [`decode_resource`](super::decode_resource) 對圖格表的做法相同。
pub fn decode_lmi1_resource(dat_path: impl AsRef<Path>, resource: i64) -> Result<Vec<LMI1Entry>> {
    parse_lmi1(&read_resource_i64(dat_path.as_ref(), resource)?)
}

impl LMI1Entry {
    fn check_geometry(&self) -> Result<()> {
        if self.width <= 0 || self.height <= 0 || self.pixels.len() as i64 != self.width.wrapping_mul(self.height) {
            return Err(FdotherError::fixed("fdother: invalid LMI1 entry geometry"));
        }
        Ok(())
    }

    /// `x < 0 || y < 0 || stride <= 0 || stride < x+W || y > len/stride || H > (len-y*stride)/stride`
    /// 的反面。
    fn destination_ok(&self, n: i64, stride: i64, x: i64, y: i64) -> bool {
        !(x < 0
            || y < 0
            || stride <= 0
            || stride < x.wrapping_add(self.width)
            || y > n / stride
            || self.height > n.wrapping_sub(y.wrapping_mul(stride)) / stride)
    }

    fn source(&self, row: i64, col: i64, mirror: bool) -> u8 {
        let src = if mirror { self.width - 1 - col } else { col };
        self.pixels[(row * self.width + src) as usize]
    }

    /// 複製一格 LMI1，調色盤索引 0 保留目的端。只給已還原的繪圖原語確實有這條規則的呼叫端
    /// 使用；原生 `0x4e8af` 本身是不透明的，用下面的 [`blit_opaque_at`](Self::blit_opaque_at)。
    pub fn blit_at(&self, dst: &mut [u8], stride: i64, x: i64, y: i64, mirror: bool) -> Result<()> {
        self.check_geometry()?;
        if !self.destination_ok(dst.len() as i64, stride, x, y) {
            return Err(FdotherError::fixed("fdother: LMI1 destination is too small"));
        }
        for row in 0..self.height {
            for col in 0..self.width {
                let v = self.source(row, col, mirror);
                if v != 0 {
                    dst[(y + row).wrapping_mul(stride).wrapping_add(x + col) as usize] = v;
                }
            }
        }
        Ok(())
    }

    /// 重現 `0x4e8af` 直接寫入解碼像素的行為。調色盤索引 0 也會覆寫目的端。
    pub fn blit_opaque_at(&self, dst: &mut [u8], stride: i64, x: i64, y: i64, mirror: bool) -> Result<()> {
        self.check_geometry()?;
        if !self.destination_ok(dst.len() as i64, stride, x, y) {
            return Err(FdotherError::fixed("fdother: LMI1 destination is too small"));
        }
        for row in 0..self.height {
            for col in 0..self.width {
                dst[(y + row).wrapping_mul(stride).wrapping_add(x + col) as usize] = self.source(row, col, mirror);
            }
        }
        Ok(())
    }

    /// 明確的裁邊版本，給起點落在索引視窗外的原生滑動呼叫端。一般 UI 素材的嚴格
    /// [`blit_at`](Self::blit_at) 契約不變。
    pub fn blit_at_clipped(&self, dst: &mut [u8], stride: i64, x: i64, y: i64, mirror: bool) -> Result<()> {
        self.check_geometry()?;
        let n = dst.len() as i64;
        if stride <= 0 || y >= n / stride || y.wrapping_add(self.height) <= 0 {
            return Err(FdotherError::fixed("fdother: clipped LMI1 destination is outside surface"));
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
                let v = self.source(row, col, mirror);
                if v != 0 {
                    dst[(dy * stride + dx) as usize] = v;
                }
            }
        }
        Ok(())
    }
}

/// 解碼原生 UI 圖庫使用的小型索引色子資源容器。目錄從位元組 6 開始（`LMI1` 與 u16 筆數
/// 之後），每筆是 {u16 width, u16 height, 0xc0 編碼串流}。編碼刻意與 FDOTHER 圖格表的
/// RLE 分開：`<= 0xc0` 的位元組是逐字像素，`> 0xc0` 表示下一個像素位元組重複
/// (byte-0xc0) 次。
pub fn parse_lmi1(data: &[u8]) -> Result<Vec<LMI1Entry>> {
    if data.len() < 6 || &data[..4] != b"LMI1" {
        return Err(FdotherError::fixed("fdother: missing LMI1 magic"));
    }
    let len = data.len() as i64;
    let count = read_le16(data, 4);
    let table_end = 6 + count * 4;
    if count == 0 || table_end > len {
        return Err(FdotherError::fixed("fdother: invalid LMI1 directory"));
    }
    let mut entries = Vec::with_capacity(count as usize);
    let mut previous = table_end;
    for i in 0..count {
        let off = read_le32(data, 6 + i * 4);
        let end = if i + 1 < count { read_le32(data, 6 + (i + 1) * 4) } else { len };
        if off < previous || off + 4 > end || end > len {
            return Err(FdotherError::message(format!("fdother: LMI1 entry {i} bounds invalid")));
        }
        let w = read_le16(data, off);
        let h = read_le16(data, off + 2);
        if w <= 0 || h <= 0 {
            return Err(FdotherError::message(format!("fdother: LMI1 entry {i} has empty dimensions")));
        }
        // LMI1 目錄給的是項目起點，不是壓縮串流的嚴格終點。原生 0x4e916 一直讀到
        // width×height 的目的端迴圈填滿為止，重複段可能越過下一個目錄偏移。這裡只以所在
        // 資源為界，與原生行為相同。
        let pixels = decode_lmi1_pixels(&data[(off + 4) as usize..], w * h)
            .map_err(|e| FdotherError::message(format!("fdother: LMI1 entry {i}: {e}")))?;
        entries.push(LMI1Entry { width: w, height: h, pixels });
        previous = off;
    }
    Ok(entries)
}

fn decode_lmi1_pixels(mut stream: &[u8], want: i64) -> std::result::Result<Vec<u8>, &'static str> {
    let want = want as usize;
    // Go 直接 make(0, want)；這裡以串流能產生的上限（每位元組最多 63 個像素）為容量，
    // 避免宣稱巨大尺寸的短串流先配置數 GB。結果相同。
    let mut pixels = Vec::with_capacity(want.min(stream.len().saturating_mul(63)));
    while pixels.len() < want {
        let Some((&c, rest)) = stream.split_first() else {
            return Err("codec stream ends before dimensions are filled");
        };
        stream = rest;
        if c <= 0xc0 {
            pixels.push(c);
            continue;
        }
        let mut run = c as usize - 0xc0;
        let Some((&v, rest)) = stream.split_first() else {
            return Err("repeat command lacks pixel value");
        };
        stream = rest;
        // 0x4e916 的重複計數器跨掃描線保留。最後一個重複段越過圖格邊界時，原生
        // width×height 迴圈只是停止消費像素，多餘的重複狀態隨這次 blit 丟棄。這裡照樣以
        // 目的端為界。
        run = run.min(want - pixels.len());
        pixels.resize(pixels.len() + run, v);
    }
    Ok(pixels)
}
