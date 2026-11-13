//! Go 來源：`remake/internal/battle/native_persistent_copy.go`（44 行）。

use super::*;

/// ApplyNativePersistentRecordCopy reproduces the mutation core of 0x11506
/// after its caller has selected a runtime/persistent pair. It copies one
/// 0x50-byte runtime record to persistent storage, clears persistent +0x22..27,
/// preserves only byte +0x05 bit0, then copies +0x46 to +0x44 and (unless the
/// preserved byte equals 1) copies +0x42 to +0x40. The unknown 0x3453e
/// zero-identity gate and 0x1145a tail are intentionally caller-owned.
///
/// Go：`native_persistent_copy.go:14`，`func ApplyNativePersistentRecordCopy(runtime, persistent []byte, runtimeIndex, persistentIndex int) error`
pub fn apply_native_persistent_record_copy(
    runtime: &[u8],
    persistent: &mut [u8],
    runtime_index: i64,
    persistent_index: i64,
) -> Result<()> {
    let runtime_base = native_persistent_record_base(runtime, runtime_index).map_err(|e| e.wrap("runtime: "))? as usize;
    let persistent_base =
        native_persistent_record_base(persistent, persistent_index).map_err(|e| e.wrap("persistent: "))? as usize;
    let n = NATIVE_RECORD_SIZE as usize;
    let mut record = [0u8; NATIVE_RECORD_SIZE as usize];
    record.copy_from_slice(&runtime[runtime_base..runtime_base + n]);
    let p = &mut persistent[persistent_base..persistent_base + n];
    p.copy_from_slice(&record);
    for b in &mut p[0x22..=0x27] {
        *b = 0;
    }
    p[5] &= 1;
    if p[5] != 1 {
        p.copy_within(0x42..0x44, 0x40);
    }
    p.copy_within(0x46..0x48, 0x44);
    Ok(())
}

/// Go：`native_persistent_copy.go:38`，`func nativePersistentRecordBase(records []byte, index int) (int, error)`
pub(crate) fn native_persistent_record_base(records: &[u8], index: i64) -> Result<i64> {
    let base = index.wrapping_mul(NATIVE_PERSISTENT_RECORD_SIZE);
    if index < 0 || base < 0 || base.wrapping_add(NATIVE_PERSISTENT_RECORD_SIZE) > records.len() as i64 {
        return Err(battle_err!("record index {index} is out of bounds"));
    }
    Ok(base)
}

#[cfg(test)]
mod native_persistent_copy_test;
