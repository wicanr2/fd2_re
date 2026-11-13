//! Go 來源：`remake/internal/battle/native_raw_word_subtract.go`（28 行）。

use super::*;

/// ApplyNativeRawWordSubtract reproduces the arithmetic core used by 0x1ca89:
/// subtract a caller-supplied byte-sized value from a selected record word and
/// store the low 16 bits. The word offset remains explicit because the native
/// command-record lookup is caller-owned. It is not part of 0x1cac7, which is
/// an indexed presentation helper.
///
/// Go：`native_raw_word_subtract.go:13`，`func ApplyNativeRawWordSubtract(records []byte, unitIndex, wordOffset, amount int) (uint16, error)`
pub fn apply_native_raw_word_subtract(
    records: &mut [u8],
    unit_index: i64,
    word_offset: i64,
    amount: i64,
) -> Result<u16> {
    if unit_index < 0 || unit_index >= records.len() as i64 / NATIVE_RECORD_SIZE {
        return Err(record_bounds_error(unit_index));
    }
    if word_offset < 0 || word_offset.wrapping_add(2) > NATIVE_RECORD_SIZE {
        return Err(battle_err!("native word offset {} is out of bounds", go_sharp_x(word_offset)));
    }
    if !(0..=0xff).contains(&amount) {
        return Err(battle_err!("native subtract amount {amount} is out of byte range"));
    }
    let off = (unit_index * NATIVE_RECORD_SIZE + word_offset) as usize;
    let value = u16::from_le_bytes([records[off], records[off + 1]]);
    let updated = value.wrapping_sub(amount as u16);
    records[off..off + 2].copy_from_slice(&updated.to_le_bytes());
    Ok(updated)
}

#[cfg(test)]
mod native_raw_word_subtract_test;
