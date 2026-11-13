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
    if !(0..NATIVE_RECORD_SIZE).contains(&marker_offset) {
        return Err(battle_err!("native flag restore marker offset {} is out of bounds", go_sharp_x(marker_offset)));
    }
    for &raw_index in target_indices {
        if raw_index as i64 >= records.len() as i64 / NATIVE_RECORD_SIZE {
            return Err(record_bounds_error(raw_index as i64));
        }
    }
    let mut rng_state = rng_state;
    let marker_offset = marker_offset as usize;
    let mut results = Vec::with_capacity(target_indices.len());
    let mut total = 0i64;
    for &raw_index in target_indices {
        let mut result = NativeRawFlagRestoreResult { target_index: raw_index as i64, ..Default::default() };
        let base = raw_index as usize * NATIVE_RECORD_SIZE as usize;
        if records[base + marker_offset] != 0 {
            let restore = apply_native_raw_hp_restore(records, raw_index as i64, 10, rng_state)?;
            rng_state = restore.rng_state;
            let mut effective = records[base + 0x21] as i64;
            if records[base + 0x20] > 8 && records[base + 0x20] < 0x19 {
                effective += 0x1e;
            }
            records[base + marker_offset] = 0;
            total = total.wrapping_add(effective * 4);
            result.applied = true;
            result.restore = restore;
        }
        results.push(result);
    }
    Ok((results, rng_state, total))
}

#[cfg(test)]
mod native_raw_flag_restore_test;
