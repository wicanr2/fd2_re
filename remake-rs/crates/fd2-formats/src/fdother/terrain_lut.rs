//! 移植自 Go `terrain_lut.go`：FD2.EXE 靜態 `0x51a97` 地形 LUT 相位表。

use super::{FdotherError, Result};

/// 讀 FD2.EXE 靜態的 `0x51a97` 表；`0x11eee` 在沒有明確調色盤覆寫時使用。20 個相位依序選
/// FDOTHER #3 的 LUT 項目 0..10 再 9..1；phase 是執行期的 `0x53c1f` 計數器。
pub fn native_terrain_lut_index(phase: i64) -> Result<i64> {
    const TABLE: [i64; 20] = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1];
    if !(0..TABLE.len() as i64).contains(&phase) {
        return Err(FdotherError::fixed("fdother: invalid native terrain LUT phase"));
    }
    Ok(TABLE[phase as usize])
}
