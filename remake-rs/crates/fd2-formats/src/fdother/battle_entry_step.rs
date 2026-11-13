//! 移植自 Go `battle_entry_step.go`。

use super::native_flow_error::{NativeFlowResult, fixed, le_u16};

/// 套用已證實的 `0x1a30b` raw 記錄閘門，以及 word+0x40 往 word+0x42 的推進。欄位偏移刻意
/// 不命名。回傳 `(next, eligible, changed)`。
///
/// `current + maximum/5` 與 Go 一樣以 `uint16` 繞回：`current` 接近 0xffff 時和可能繞成小於
/// `maximum` 的值，此時不夾到 `maximum`，直接寫入繞回後的值。
pub fn native_battle_entry_step(raw: &mut [u8]) -> NativeFlowResult<(u16, bool, bool)> {
    if raw.len() < 0x44 {
        return fixed("fdother: battle-entry record is shorter than 0x44 bytes");
    }
    if raw[6] != 2 || raw[5] & 0x81 != 0 || raw[0x25] != 0 || raw[0x26] != 0 {
        return Ok((le_u16(raw, 0x40), false, false));
    }
    let current = le_u16(raw, 0x40);
    let maximum = le_u16(raw, 0x42);
    if current == maximum {
        return Ok((current, false, false));
    }
    let step = maximum / 5;
    let mut next = current.wrapping_add(step);
    if next > maximum {
        next = maximum;
    }
    raw[0x40..0x42].copy_from_slice(&next.to_le_bytes());
    Ok((next, true, next != current))
}

#[cfg(test)]
mod tests;
