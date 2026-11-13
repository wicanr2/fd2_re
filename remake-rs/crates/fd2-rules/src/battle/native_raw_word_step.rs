//! Go 來源：`remake/internal/battle/native_raw_word_step.go`（70 行）。

use super::*;

/// NativeRawWordStepResult records the mutation-only portion of 0x22721.
///
/// Go：`native_raw_word_step.go:12`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeRawWordStepResult {
    pub unit_index: i64,
    pub processed: bool,
    pub marker: u8,
    pub delta: u16,
}

/// ApplyNativeRawWordStep reproduces 0x22721's per-unit branch.  Each index
/// addresses one 0x50-byte record. A nonzero record+0x22 is skipped. Otherwise
/// the shared 0x4e893 RNG advances, marker+0x22 receives (rng%4)+2, and the
/// derived AP at +0x48 gains trunc(AP*0.15+1) with native toward-zero semantics.
/// The returned score is the raw 2*effective(+0x21) accumulator. Renderer and
/// presentation calls are deliberately excluded.
///
/// Go：`native_raw_word_step.go:25`，`func ApplyNativeRawWordStep(records []byte, unitIndices []byte, rngState uint16) ([]NativeRawWordStepResult, uint16, int, error)`
pub fn apply_native_raw_word_step(
    records: &mut [u8],
    unit_indices: &[u8],
    rng_state: u16,
) -> Result<(Vec<NativeRawWordStepResult>, u16, i64)> {
    apply_native_raw_word_step_at_offsets(records, unit_indices, rng_state, 0x22, 0x48)
}

/// ApplyNativeRawWordStepAtOffsets is the shared implementation used by
/// 0x22721 (+0x22/derived AP +0x48) and
/// 0x22866 (+0x23/derived DP +0x4a).
///
/// Go：`native_raw_word_step.go:32`，`func ApplyNativeRawWordStepAtOffsets(records []byte, unitIndices []byte, rngState uint16, markerOffset, wordOffset int) ([]NativeRawWordStepResult, uint16, int, error)`
pub fn apply_native_raw_word_step_at_offsets(
    records: &mut [u8],
    unit_indices: &[u8],
    rng_state: u16,
    marker_offset: i64,
    word_offset: i64,
) -> Result<(Vec<NativeRawWordStepResult>, u16, i64)> {
    if !(0..NATIVE_RECORD_SIZE).contains(&marker_offset)
        || word_offset < 0
        || word_offset.wrapping_add(2) > NATIVE_RECORD_SIZE
    {
        return Err(battle_err!(
            "native raw word offsets marker={} word={} out of bounds",
            go_sharp_x(marker_offset),
            go_sharp_x(word_offset)
        ));
    }
    for &raw_index in unit_indices {
        let base = raw_index as i64 * NATIVE_RECORD_SIZE;
        if base + NATIVE_RECORD_SIZE > records.len() as i64 {
            return Err(record_bounds_error(raw_index as i64));
        }
    }
    let (marker_offset, word_offset) = (marker_offset as usize, word_offset as usize);
    let mut rng_state = rng_state;
    let mut results = Vec::with_capacity(unit_indices.len());
    let mut total = 0i64;
    for &raw_index in unit_indices {
        let unit_index = raw_index as i64;
        let base = unit_index as usize * NATIVE_RECORD_SIZE as usize;
        let record = &mut records[base..base + NATIVE_RECORD_SIZE as usize];
        let mut result = NativeRawWordStepResult { unit_index, ..Default::default() };
        if record[marker_offset] != 0 {
            results.push(result);
            continue;
        }
        rng_state = fdother::native_rng_step(rng_state);
        record[marker_offset] = (rng_state % 4 + 2) as u8;
        let current = u16::from_le_bytes([record[word_offset], record[word_offset + 1]]);
        let increment = (current as f64 * 0.15 + 1.0).trunc() as i64;
        let updated = current.wrapping_add(increment as u16);
        record[word_offset..word_offset + 2].copy_from_slice(&updated.to_le_bytes());
        let mut effective = record[0x21] as i64;
        if record[0x20] > 8 && record[0x20] < 0x19 {
            effective += 30;
        }
        result.processed = true;
        result.marker = record[marker_offset];
        result.delta = increment as u16;
        total = total.wrapping_add(2 * effective);
        results.push(result);
    }
    Ok((results, rng_state, total))
}

#[cfg(test)]
mod native_raw_word_step_test;
