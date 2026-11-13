//! Go 來源：`remake/internal/battle/native_raw_restore.go`（126 行）。

use super::*;

/// NativeRawRestoreResult exposes the proven state mutation and raw score from
/// 0x1c916.  Score is kept unnamed: callers may still have to prove how the
/// native presentation accumulator is consumed.
///
/// Go：`native_raw_restore.go:13`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeRawRestoreResult {
    pub unit_index: i64,
    pub rolled: i64,
    pub actual: i64,
    pub score: i64,
    pub rng_state: u16,
}

/// NativeRawRestoreBatch is the mutation-only result of the 0x211a4 target
/// loop.  Presentation queue writes and indexed animation remain outside this
/// adapter.
///
/// Go：`native_raw_restore.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeRawRestoreBatch {
    pub results: Vec<NativeRawRestoreResult>,
    pub rng_state: u16,
    pub score: i64,
}

/// ApplyNativeRawHPRestoreList reproduces the per-target mutation loop in
/// 0x211a4(actor,count,targetBytes,amount).  Official IDA 9.4 confirms that
/// 0x20c6f passes its a3/a4 target count/list unchanged and supplies item-row
/// word +0x0e as amount.  Inputs are preflighted so an invalid later target
/// cannot leave an earlier record partially mutated.
///
/// Go：`native_raw_restore.go:35`，`func ApplyNativeRawHPRestoreList(records []byte, targetIndices []byte, amount int, rngState uint16) (NativeRawRestoreBatch, error)`
pub fn apply_native_raw_hp_restore_list(
    records: &mut [u8],
    target_indices: &[u8],
    amount: i64,
    rng_state: u16,
) -> Result<NativeRawRestoreBatch> {
    if amount < 0 {
        return Err(battle_err!("native restore amount {amount} is negative"));
    }
    for &raw_index in target_indices {
        if raw_index as i64 >= records.len() as i64 / NATIVE_RECORD_SIZE {
            return Err(record_bounds_error(raw_index as i64));
        }
    }
    let mut result = NativeRawRestoreBatch { results: Vec::with_capacity(target_indices.len()), rng_state, score: 0 };
    for &raw_index in target_indices {
        let entry = apply_native_raw_hp_restore(records, raw_index as i64, amount, result.rng_state)?;
        result.results.push(entry);
        result.rng_state = entry.rng_state;
        result.score = result.score.wrapping_add(entry.score);
    }
    Ok(result)
}

/// ApplyNativeRawHPRestore reproduces the mutation core of 0x1c916 on a
/// 0x50-byte runtime record. The native routine advances the shared 16-bit RNG,
/// applies toward-zero integer arithmetic, clamps +0x40 to +0x42, and adds a
/// derived raw score only when record +0x07 is below 0x4b. No gameplay label is
/// assigned to those raw bytes or to the score.
///
/// Go：`native_raw_restore.go:65`，`func ApplyNativeRawHPRestore(records []byte, unitIndex, amount int, rngState uint16) (NativeRawRestoreResult, error)`
pub fn apply_native_raw_hp_restore(
    records: &mut [u8],
    unit_index: i64,
    amount: i64,
    rng_state: u16,
) -> Result<NativeRawRestoreResult> {
    if unit_index < 0 || unit_index >= records.len() as i64 / NATIVE_RECORD_SIZE {
        return Err(record_bounds_error(unit_index));
    }
    if amount < 0 {
        return Err(battle_err!("native restore amount {amount} is negative"));
    }
    let record = &mut records[(unit_index * NATIVE_RECORD_SIZE) as usize..];
    let current = u16::from_le_bytes([record[0x40], record[0x41]]) as i64;
    let max = u16::from_le_bytes([record[0x42], record[0x43]]) as i64;
    let next_rng = fdother::native_rng_step(rng_state);
    let rolled = (amount.wrapping_mul(9) / 10).wrapping_add(((next_rng % 100) as i64).wrapping_mul(amount) / 1000);
    let mut next = current.wrapping_add(rolled);
    if next > max {
        next = max;
    }
    if next < 0 {
        next = 0;
    }
    record[0x40..0x42].copy_from_slice(&(next as u16).to_le_bytes());
    let delta = next.wrapping_sub(current);
    let mut score = 0;
    if record[7] < 0x4b && max != 0 {
        let mut effective = record[0x21] as i64;
        if record[0x20] > 8 && record[0x20] < 0x19 {
            effective += 0x1e;
        }
        score = (40 * effective).wrapping_mul(delta) / max;
    }
    Ok(NativeRawRestoreResult { unit_index, rolled, actual: delta, score, rng_state: next_rng })
}

/// ApplyNativeRawMPRestore is the sibling 0x1c9dd mutation. It uses raw
/// +0x44/+0x46 and the same RNG/amount arithmetic, but its score path uses only
/// byte +0x21 (the HP routine has an additional class-range adjustment).
///
/// Go：`native_raw_restore.go:100`，`func ApplyNativeRawMPRestore(records []byte, unitIndex, amount int, rngState uint16) (NativeRawRestoreResult, error)`
pub fn apply_native_raw_mp_restore(
    records: &mut [u8],
    unit_index: i64,
    amount: i64,
    rng_state: u16,
) -> Result<NativeRawRestoreResult> {
    if unit_index < 0 || unit_index >= records.len() as i64 / NATIVE_RECORD_SIZE {
        return Err(record_bounds_error(unit_index));
    }
    if amount < 0 {
        return Err(battle_err!("native restore amount {amount} is negative"));
    }
    let record = &mut records[(unit_index * NATIVE_RECORD_SIZE) as usize..];
    let current = u16::from_le_bytes([record[0x44], record[0x45]]) as i64;
    let max = u16::from_le_bytes([record[0x46], record[0x47]]) as i64;
    let next_rng = fdother::native_rng_step(rng_state);
    let rolled = (amount.wrapping_mul(9) / 10).wrapping_add(((next_rng % 100) as i64).wrapping_mul(amount) / 1000);
    let mut next = current.wrapping_add(rolled);
    if next > max {
        next = max;
    }
    if next < 0 {
        next = 0;
    }
    record[0x44..0x46].copy_from_slice(&(next as u16).to_le_bytes());
    let delta = next.wrapping_sub(current);
    let mut score = 0;
    if record[7] < 0x4b && max != 0 {
        score = (40 * record[0x21] as i64).wrapping_mul(delta) / max;
    }
    Ok(NativeRawRestoreResult { unit_index, rolled, actual: delta, score, rng_state: next_rng })
}

#[cfg(test)]
mod native_raw_restore_test;
