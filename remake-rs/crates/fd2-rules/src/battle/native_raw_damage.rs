//! Go 來源：`remake/internal/battle/native_raw_damage.go`（39 行）。

use super::*;

/// NativeRawDamageResult exposes only the proven 0x1c81f HP mutation.
///
/// Go：`native_raw_damage.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeRawDamageResult {
    pub unit_index: i64,
    pub rolled: i64,
    pub actual: i64,
    pub rng_state: u16,
}

/// ApplyNativeRawHPDamage reproduces the raw write portion of 0x1c81f:
/// amount*9/10 + (RNG%100)*amount/1000 is subtracted from +0x40 and clamped at
/// zero. The separate native presentation-score branch is intentionally not
/// inferred here.
///
/// Go：`native_raw_damage.go:22`，`func ApplyNativeRawHPDamage(records []byte, unitIndex, amount int, rngState uint16) (NativeRawDamageResult, error)`
pub fn apply_native_raw_hp_damage(
    records: &mut [u8],
    unit_index: i64,
    amount: i64,
    rng_state: u16,
) -> Result<NativeRawDamageResult> {
    if unit_index < 0 || unit_index >= records.len() as i64 / NATIVE_RECORD_SIZE {
        return Err(record_bounds_error(unit_index));
    }
    if amount < 0 {
        return Err(battle_err!("native damage amount {amount} is negative"));
    }
    let record = &mut records[(unit_index * NATIVE_RECORD_SIZE) as usize..];
    let current = u16::from_le_bytes([record[0x40], record[0x41]]) as i64;
    let next_rng = fdother::native_rng_step(rng_state);
    let rolled = (amount.wrapping_mul(9) / 10).wrapping_add(((next_rng % 100) as i64).wrapping_mul(amount) / 1000);
    let mut next = current.wrapping_sub(rolled);
    if next < 0 {
        next = 0;
    }
    record[0x40..0x42].copy_from_slice(&(next as u16).to_le_bytes());
    Ok(NativeRawDamageResult { unit_index, rolled, actual: current.wrapping_sub(next), rng_state: next_rng })
}

#[cfg(test)]
mod native_raw_damage_test;
