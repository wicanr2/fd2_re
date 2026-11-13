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
    todo!("battle/native_raw_pair_step.go:ApplyNativeRawPairStep")
}
