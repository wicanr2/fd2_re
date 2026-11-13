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
    todo!("battle/native_raw_restore.go:ApplyNativeRawHPRestoreList")
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
    todo!("battle/native_raw_restore.go:ApplyNativeRawHPRestore")
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
    todo!("battle/native_raw_restore.go:ApplyNativeRawMPRestore")
}
