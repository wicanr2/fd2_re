//! 移植自 Go `lutbank.go`：FDOTHER #3 的 LMI1 目錄形式 256 位元組重映射表。

use super::{FdotherError, Result, read_le16, read_le32, read_resource_i64};
use std::path::Path;

/// 讀 FDOTHER #3 的 LMI1 目錄形式：magic、u16 筆數、u32 相對偏移，然後是 256 位元組的
/// 索引重映射表。刻意與項目是 width/height UI 圖格的 [`parse_lmi1`](super::parse_lmi1) 分開。
pub fn parse_lut_bank(data: &[u8]) -> Result<Vec<Vec<u8>>> {
    if data.len() < 6 || &data[..4] != b"LMI1" {
        return Err(FdotherError::fixed("fdother: missing LUT-bank LMI1 magic"));
    }
    let len = data.len() as i64;
    let count = read_le16(data, 4);
    if count == 0 || 6 + count * 4 > len {
        return Err(FdotherError::fixed("fdother: invalid LUT-bank directory"));
    }
    let mut luts = Vec::with_capacity(count as usize);
    for i in 0..count {
        let off = read_le32(data, 6 + i * 4);
        if off < 6 + count * 4 || off + 256 > len {
            return Err(FdotherError::message(format!("fdother: LUT {i} bounds invalid")));
        }
        luts.push(data[off as usize..off as usize + 256].to_vec());
    }
    Ok(luts)
}

/// 把玩家提供的一筆 FDOTHER 封存項目當成原生 256 項 LUT 庫開啟。
pub fn decode_lut_resource(dat_path: impl AsRef<Path>, resource: i64) -> Result<Vec<Vec<u8>>> {
    parse_lut_bank(&read_resource_i64(dat_path.as_ref(), resource)?)
}

/// 實體化 `0x22547` 回傳的刻意未對齊的表。這個輔助函式經共用收尾離開，回傳 FDOTHER #3
/// entry0 指標 +1；`0x22046` 仍從那個位址起索引 256 位元組。原生項目是連續的 0x100 位元組
/// 表，所以這個視圖的位元組 255 是 entry1[0]。它既不是一般的 LUT0 也不是 LUT1。
pub fn native_unit_present_bridge_lut(luts: &[Vec<u8>]) -> Result<Vec<u8>> {
    if luts.len() < 2 || luts[0].len() != 256 || luts[1].len() != 256 {
        return Err(FdotherError::fixed("fdother: incomplete unit-present bridge LUT source"));
    }
    let mut bridge = vec![0u8; 256];
    bridge[..255].copy_from_slice(&luts[0][1..]);
    bridge[255] = luts[1][0];
    Ok(bridge)
}
