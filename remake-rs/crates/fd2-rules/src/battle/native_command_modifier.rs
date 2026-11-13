//! Go 來源：`remake/internal/battle/native_command_modifier.go`（46 行）。

use super::*;

/// NativeCommandModifierResult is the raw-only result of command IDs 17..19.
/// The union fields preserve the distinct native branch shapes without naming
/// the mutated words as gameplay stats.
///
/// Go：`native_command_modifier.go:8`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandModifierResult {
    pub command_id: i64,
    pub word_steps: Vec<NativeRawWordStepResult>,
    pub pair_steps: Vec<NativeRawPairStepResult>,
    pub rng_state: u16,
    pub accumulator: i64,
}

/// ApplyNativeCommandModifier dispatches the proven command table branches:
/// ID17→0x22721 (+0x22/+0x48), ID18→0x22866 (+0x23/+0x4a), and
/// ID19→0x22997 (+0x24/+0x4c/+0x4e). It deliberately excludes command MP
/// debit, target selection, presentation, and any stat/effect naming.
///
/// Go：`native_command_modifier.go:20`，`func ApplyNativeCommandModifier(records []byte, unitIndices []byte, commandID int, rngState uint16) (NativeCommandModifierResult, error)`
pub fn apply_native_command_modifier(
    records: &mut [u8],
    unit_indices: &[u8],
    command_id: i64,
    rng_state: u16,
) -> Result<NativeCommandModifierResult> {
    if !(17..=19).contains(&command_id) {
        return Err(battle_err!("native modifier: unsupported command id {command_id}"));
    }
    let mut result = NativeCommandModifierResult { command_id, ..Default::default() };
    match command_id {
        17 => {
            let (steps, state, score) =
                apply_native_raw_word_step_at_offsets(records, unit_indices, rng_state, 0x22, 0x48)?;
            (result.word_steps, result.rng_state, result.accumulator) = (steps, state, score);
        }
        18 => {
            let (steps, state, score) =
                apply_native_raw_word_step_at_offsets(records, unit_indices, rng_state, 0x23, 0x4a)?;
            (result.word_steps, result.rng_state, result.accumulator) = (steps, state, score);
        }
        _ => {
            let (steps, state, score) = apply_native_raw_pair_step(records, unit_indices, rng_state)?;
            (result.pair_steps, result.rng_state, result.accumulator) = (steps, state, score);
        }
    }
    Ok(result)
}

#[cfg(test)]
mod native_command_modifier_test;
