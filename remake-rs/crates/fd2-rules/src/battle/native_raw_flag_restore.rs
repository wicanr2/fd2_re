//! Go 來源：`remake/internal/battle/native_raw_flag_restore.go`（47 行）。

use super::*;

/// NativeRawFlagRestoreResult records the raw result of one 0x22af6 flag cell.
///
/// Go：`native_raw_flag_restore.go:6`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeRawFlagRestoreResult {
    pub target_index: i64,
    pub applied: bool,
    pub restore: NativeRawRestoreResult,
}

/// ApplyNativeRawFlagRestore reproduces the proven 0x22af6 branch. For each
/// target, a nonzero byte at record+markerOffset invokes 0x1c916(target,10),
/// is then cleared in that same runtime record, and contributes the native
/// effective-value*4 accumulator. Marker meaning and UI remain caller-owned.
///
/// Go：`native_raw_flag_restore.go:16`，`func ApplyNativeRawFlagRestore(records []byte, targetIndices []byte, markerOffset int, rngState uint16) ([]NativeRawFlagRestoreResult, uint16, int, error)`
pub fn apply_native_raw_flag_restore(
    records: &mut [u8],
    target_indices: &[u8],
    marker_offset: i64,
    rng_state: u16,
) -> Result<(Vec<NativeRawFlagRestoreResult>, u16, i64)> {
    todo!("battle/native_raw_flag_restore.go:ApplyNativeRawFlagRestore")
}
