//! Go 來源：`remake/internal/battle/native_raw_pair_step.go`（52 行）。

use super::*;

/// NativeRawPairStepResult records the mutation-only portion of 0x22997.
///
/// Go：`native_raw_pair_step.go:10`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeRawPairStepResult {
    pub unit_index: i64,
    pub processed: bool,
    pub marker: u8,
}

/// ApplyNativeRawPairStep reproduces 0x22997: skip a nonzero marker at +0x24;
/// otherwise advance the shared RNG, write marker=(rng%4)+2, and add 0x0f to
/// the derived HIT/EV words at +0x4c/+0x4e. The raw score is
/// 2*effective(+0x21).
///
/// Go：`native_raw_pair_step.go:20`，`func ApplyNativeRawPairStep(records []byte, unitIndices []byte, rngState uint16) ([]NativeRawPairStepResult, uint16, int, error)`
pub fn apply_native_raw_pair_step(
    records: &mut [u8],
    unit_indices: &[u8],
    rng_state: u16,
) -> Result<(Vec<NativeRawPairStepResult>, u16, i64)> {
    for &raw_index in unit_indices {
        let base = raw_index as i64 * NATIVE_RECORD_SIZE;
        if base + NATIVE_RECORD_SIZE > records.len() as i64 {
            return Err(record_bounds_error(raw_index as i64));
        }
    }
    let mut rng_state = rng_state;
    let mut results = Vec::with_capacity(unit_indices.len());
    let mut total = 0i64;
    for &raw_index in unit_indices {
        let unit_index = raw_index as i64;
        let base = unit_index as usize * NATIVE_RECORD_SIZE as usize;
        let record = &mut records[base..base + NATIVE_RECORD_SIZE as usize];
        let mut result = NativeRawPairStepResult { unit_index, ..Default::default() };
        if record[0x24] != 0 {
            results.push(result);
            continue;
        }
        rng_state = fdother::native_rng_step(rng_state);
        record[0x24] = (rng_state % 4 + 2) as u8;
        for off in [0x4c, 0x4e] {
            let value = u16::from_le_bytes([record[off], record[off + 1]]).wrapping_add(0x0f);
            record[off..off + 2].copy_from_slice(&value.to_le_bytes());
        }
        let mut effective = record[0x21] as i64;
        if record[0x20] > 8 && record[0x20] < 0x19 {
            effective += 30;
        }
        result.processed = true;
        result.marker = record[0x24];
        total = total.wrapping_add(2 * effective);
        results.push(result);
    }
    Ok((results, rng_state, total))
}

#[cfg(test)]
mod native_raw_pair_step_test;
