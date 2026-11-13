//! 移植自 Go `lmi1_frame.go`：LMI1 目錄中由其他原生原語消費的單筆項目。

use super::{
    FdotherError, Frame, LMI1Entry, RawCell, Result, parse_opaque_run_cell, parse_single_frame, read_le16, read_le32,
    read_resource_i64,
};
use std::path::Path;

/// 驗 LMI1 檔頭並回傳（筆數, 目錄結尾）。索引錯誤用呼叫端指定的訊息。
fn directory(data: &[u8], index: i64, invalid_index: &'static str) -> Result<(i64, i64)> {
    if data.len() < 6 || &data[..4] != b"LMI1" {
        return Err(FdotherError::fixed("fdother: missing LMI1 magic"));
    }
    let count = read_le16(data, 4);
    let table_end = 6 + count * 4;
    if count == 0 || table_end > data.len() as i64 || index < 0 || index >= count {
        return Err(FdotherError::fixed(invalid_index));
    }
    Ok((count, table_end))
}

/// 讀一筆原生呼叫端送進 `0x4e63d` 的 LMI1 目錄項目。LMI1 是目錄，不是通用的像素編碼：
/// 這條路徑刻意與 [`parse_lmi1`](super::parse_lmi1) 的 `0x4e916` 圖格分開。
pub fn decode_lmi1_frame_resource(dat_path: impl AsRef<Path>, resource: i64, index: i64) -> Result<Frame> {
    parse_lmi1_frame_entry(&read_resource_i64(dat_path.as_ref(), resource)?, index)
}

/// 取出單筆 width/height 標頭接 `0x4e63d` 四模式 RLE 的 LMI1 項目。不推論相鄰項目用同一種編碼。
pub fn parse_lmi1_frame_entry(data: &[u8], index: i64) -> Result<Frame> {
    let (_, table_end) = directory(data, index, "fdother: LMI1 frame index is invalid")?;
    let off = read_le32(data, 6 + index * 4);
    if off < table_end || off + 4 > data.len() as i64 {
        return Err(FdotherError::fixed("fdother: LMI1 frame offset is invalid"));
    }
    parse_single_frame(&data[off as usize..])
}

/// 讀另一條 `0x1685c`→`0x4e9bb` 路徑。沿用 LMI1 目錄，但選到的圖格當成原始 width×height
/// 位元組複製；不套用 `0x4e916`／`0x4e63d` 任何連續段編碼。
pub fn decode_lmi1_raw_entry(dat_path: impl AsRef<Path>, resource: i64, index: i64) -> Result<RawCell> {
    parse_lmi1_raw_entry(&read_resource_i64(dat_path.as_ref(), resource)?, index)
}

pub fn parse_lmi1_raw_entry(data: &[u8], index: i64) -> Result<RawCell> {
    let (count, table_end) = directory(data, index, "fdother: LMI1 raw entry index is invalid")?;
    let len = data.len() as i64;
    let off = read_le32(data, 6 + index * 4);
    let end = if index + 1 < count { read_le32(data, 6 + (index + 1) * 4) } else { len };
    if off < table_end || off + 4 > end || end > len {
        return Err(FdotherError::fixed("fdother: LMI1 raw entry bounds are invalid"));
    }
    let w = read_le16(data, off);
    let h = read_le16(data, off + 2);
    if w <= 0 || h <= 0 || w > (end - off - 4) / h {
        return Err(FdotherError::fixed("fdother: LMI1 raw entry geometry is invalid"));
    }
    let start = (off + 4) as usize;
    Ok(RawCell { width: w, height: h, pixels: data[start..start + (w * h) as usize].to_vec() })
}

/// 取出一筆呼叫端送進 `0x4e8af` 的 LMI1 目錄項目。那個原語解 `0x4e916` 高位連續段串流，
/// 每個像素都寫，包括調色盤索引 0。
pub fn parse_lmi1_opaque_entry(data: &[u8], index: i64) -> Result<LMI1Entry> {
    let (_, table_end) = directory(data, index, "fdother: LMI1 opaque entry index is invalid")?;
    let off = read_le32(data, 6 + index * 4);
    if off < table_end || off + 4 > data.len() as i64 {
        return Err(FdotherError::fixed("fdother: LMI1 opaque entry offset is invalid"));
    }
    parse_opaque_run_cell(&data[off as usize..])
}
