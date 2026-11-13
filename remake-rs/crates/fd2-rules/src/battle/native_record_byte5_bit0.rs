//! Go 來源：`remake/internal/battle/native_record_byte5_bit0.go`（13 行）。

use super::*;

/// NativeRecordByte5Bit0 reproduces 0x3453e(index): return the selected
/// runtime record byte +0x05 masked with bit0. It is a raw predicate only; no
/// acted/alive/active name is assigned.
///
/// Go：`native_record_byte5_bit0.go:8`，`func NativeRecordByte5Bit0(records []byte, index int) (byte, error)`
pub fn native_record_byte5_bit0(records: &[u8], index: i64) -> Result<u8> {
    if index < 0 || index >= records.len() as i64 / NATIVE_RECORD_SIZE {
        return Err(battle_err!("native record index {index} is out of bounds"));
    }
    Ok(records[(index * NATIVE_RECORD_SIZE + 5) as usize] & 1)
}

#[cfg(test)]
mod native_record_byte5_bit0_test;
