//! Go 來源：`remake/internal/battle/native_persistent_lookup.go`（24 行）。

use super::*;

/// Go 原式：`nativeRecordSize`
pub(crate) const NATIVE_PERSISTENT_RECORD_SIZE: i64 = 80;

/// FindNativePersistentIdentity reproduces the byte-level lookup shared by
/// 0x24bde/0x33499: scan the caller-supplied persistent record count, using
/// record stride 0x50 and comparing the unsigned byte at record+0x08. It
/// returns the first raw record index and never changes the buffer.
///
/// Go：`native_persistent_lookup.go:11`，`func FindNativePersistentIdentity(records []byte, count int, identity byte) (int, bool, error)`
pub fn find_native_persistent_identity(records: &[u8], count: i64, identity: u8) -> Result<(i64, bool)> {
    todo!("battle/native_persistent_lookup.go:FindNativePersistentIdentity")
}
