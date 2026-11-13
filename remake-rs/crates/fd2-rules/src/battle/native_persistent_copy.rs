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
    todo!("battle/native_persistent_copy.go:ApplyNativePersistentRecordCopy")
}

/// Go：`native_persistent_copy.go:38`，`func nativePersistentRecordBase(records []byte, index int) (int, error)`
pub(crate) fn native_persistent_record_base(records: &[u8], index: i64) -> Result<i64> {
    todo!("battle/native_persistent_copy.go:nativePersistentRecordBase")
}
