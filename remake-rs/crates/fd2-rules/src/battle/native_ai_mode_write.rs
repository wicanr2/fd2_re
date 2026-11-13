//! Go 來源：`remake/internal/battle/native_ai_mode_write.go`（33 行）。

use super::*;

/// SetNativeAIModeRange reproduces 0x3419c. It updates the low nibble of
/// runtime record +0x34 for every inclusive unit index while preserving the
/// high-nibble flags.
///
/// Go：`native_ai_mode_write.go:8`，`func SetNativeAIModeRange( records []byte, count, first, last int, mode byte, ) error`
pub fn set_native_ai_mode_range(records: &mut [u8], count: i64, first: i64, last: i64, mode: u8) -> Result<()> {
    if count < 0
        || count > records.len() as i64 / NATIVE_RECORD_SIZE
        || first < 0
        || last < first
        || last >= count
        || mode > 0x0f
    {
        return Err(battle_err!("native AI mode range is out of bounds"));
    }
    for unit in first..=last {
        let offset = (unit * NATIVE_RECORD_SIZE + 0x34) as usize;
        records[offset] = records[offset] & 0xf0 | mode;
    }
    Ok(())
}

/// SetNativeAIModeByte preserves the full-byte writer used by the mode-5
/// completion branch at 0x13d20 and several chapter handlers. Unlike
/// SetNativeAIModeRange, it intentionally replaces high-nibble flags.
///
/// Go：`native_ai_mode_write.go:27`，`func SetNativeAIModeByte(records []byte, count, unit int, value byte) error`
pub fn set_native_ai_mode_byte(records: &mut [u8], count: i64, unit: i64, value: u8) -> Result<()> {
    if count < 0 || count > records.len() as i64 / NATIVE_RECORD_SIZE || unit < 0 || unit >= count {
        return Err(battle_err!("native AI mode unit is out of bounds"));
    }
    records[(unit * NATIVE_RECORD_SIZE + 0x34) as usize] = value;
    Ok(())
}

#[cfg(test)]
mod native_ai_mode_write_test;
