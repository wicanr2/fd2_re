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
    todo!("battle/native_unit_byte_clear.go:ClearNativeUnitByte")
}
