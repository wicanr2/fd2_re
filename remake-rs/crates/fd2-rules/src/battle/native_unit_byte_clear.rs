//! Go 來源：`remake/internal/battle/native_unit_byte_clear.go`（46 行）。

use super::*;

/// NativeUnitByteClearResult is the mutation-only portion of 0x22af6.  The
/// caller supplies unit indices and a raw record offset; gameplay meaning of
/// the byte and the accumulated value is intentionally not named.
///
/// Go：`native_unit_byte_clear.go:8`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeUnitByteClearResult {
    pub unit_index: i64,
    pub cleared: bool,
    pub raw_value: i64,
}

/// ClearNativeUnitByte reproduces 0x22af6's per-unit mutation.  For each
/// supplied index, a nonzero record[fieldOffset] is cleared and contributes
/// 4*effective(record+0x21) to the raw accumulator.  The effective value adds
/// 30 when record+0x20 is in (8,25).  Rendering callbacks and global writes are
/// deliberately outside this pure adapter.
///
/// Go：`native_unit_byte_clear.go:19`，`func ClearNativeUnitByte(records []byte, unitIndices []byte, fieldOffset int) ([]NativeUnitByteClearResult, int, error)`
pub fn clear_native_unit_byte(
    records: &mut [u8],
    unit_indices: &[u8],
    field_offset: i64,
) -> Result<(Vec<NativeUnitByteClearResult>, i64)> {
    if !(0..NATIVE_RECORD_SIZE).contains(&field_offset) {
        return Err(battle_err!("native unit field offset {} is out of bounds", go_sharp_x(field_offset)));
    }
    let field_offset = field_offset as usize;
    let mut results = Vec::with_capacity(unit_indices.len());
    let mut total = 0i64;
    for &raw_index in unit_indices {
        let unit_index = raw_index as i64;
        let base = unit_index * NATIVE_RECORD_SIZE;
        if base + NATIVE_RECORD_SIZE > records.len() as i64 {
            return Err(record_bounds_error(unit_index));
        }
        let record = &mut records[base as usize..(base + NATIVE_RECORD_SIZE) as usize];
        let mut result = NativeUnitByteClearResult { unit_index, ..Default::default() };
        if record[field_offset] != 0 {
            record[field_offset] = 0;
            let mut effective = record[0x21] as i64;
            if record[0x20] > 8 && record[0x20] < 0x19 {
                effective += 30;
            }
            result.cleared = true;
            result.raw_value = 4 * effective;
            total = total.wrapping_add(result.raw_value);
        }
        results.push(result);
    }
    Ok((results, total))
}

#[cfg(test)]
mod native_unit_byte_clear_test;
