//! Go 來源：`remake/internal/battle/native_raw_application.go`（60 行）。

use super::*;

/// Go：`native_raw_application.go:9`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeRawApplicationResult {
    pub target_index: i64,
    pub applied: bool,
    pub damage: NativeRawDamageResult,
    pub marker: u8,
}

/// ApplyNativeRawApplication reproduces the mutation/gate portion of 0x22d1b.
/// A target is eligible only when its marker byte is zero, class byte +0x20 is
/// not 0x19/0x1a, and the first RNG remainder mod 100 is below 50. Success
/// calls 0x1c81f with base amount 10 (which advances RNG again and rolls
/// amount*0.9 + RNG*amount/1000), advances RNG a third time, writes marker
/// (RNG%4)+2 at markerOffset, and adds 8*record+0x21 to the accumulator.
/// Presentation and marker/status names remain outside this adapter.
///
/// Go：`native_raw_application.go:23`，`func ApplyNativeRawApplication(records []byte, targetIndices []byte, markerOffset int, rngState uint16) ([]NativeRawApplicationResult, uint16, int, error)`
pub fn apply_native_raw_application(
    records: &mut [u8],
    target_indices: &[u8],
    marker_offset: i64,
    rng_state: u16,
) -> Result<(Vec<NativeRawApplicationResult>, u16, i64)> {
    if !(0..NATIVE_RECORD_SIZE).contains(&marker_offset) {
        return Err(battle_err!("native application marker offset {} is out of bounds", go_sharp_x(marker_offset)));
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
        let index = raw_index as i64;
        let base = index as usize * NATIVE_RECORD_SIZE as usize;
        let mut result = NativeRawApplicationResult { target_index: index, ..Default::default() };
        if records[base + marker_offset] != 0 || records[base + 0x20] == 0x19 || records[base + 0x20] == 0x1a {
            results.push(result);
            continue;
        }
        let gate_state = fdother::native_rng_step(rng_state);
        if gate_state % 100 >= 50 {
            rng_state = gate_state;
            results.push(result);
            continue;
        }
        let damage = apply_native_raw_hp_damage(records, index, 10, gate_state)?;
        rng_state = fdother::native_rng_step(damage.rng_state);
        let marker = (rng_state % 4 + 2) as u8;
        records[base + marker_offset] = marker;
        total = total.wrapping_add(8 * records[base + 0x21] as i64);
        result.applied = true;
        result.damage = damage;
        result.marker = marker;
        results.push(result);
    }
    Ok((results, rng_state, total))
}

#[cfg(test)]
mod native_raw_application_test;
