//! Go 來源：`remake/internal/battle/native_record_byte5.go`（14 行）。

use super::*;

/// SetNativeRecordByte5One reproduces 0x32975(index): it overwrites (rather
/// than ORs) byte +0x05 of the selected 0x50-byte runtime record with 1. The
/// byte's higher-level action meaning is intentionally not assigned here.
///
/// Go：`native_record_byte5.go:8`，`func SetNativeRecordByte5One(records []byte, index int) error`
pub fn set_native_record_byte5_one(records: &mut [u8], index: i64) -> Result<()> {
    if index < 0 || index >= records.len() as i64 / NATIVE_RECORD_SIZE {
        return Err(battle_err!("native record index {index} is out of bounds"));
    }
    records[(index * NATIVE_RECORD_SIZE + 5) as usize] = 1;
    Ok(())
}

#[cfg(test)]
mod native_record_byte5_test;
