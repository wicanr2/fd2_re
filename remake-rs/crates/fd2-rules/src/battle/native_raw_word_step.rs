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
    todo!("battle/native_raw_word_step.go:ApplyNativeRawWordStep")
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
    todo!("battle/native_raw_word_step.go:ApplyNativeRawWordStepAtOffsets")
}
