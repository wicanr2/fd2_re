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
    todo!("battle/native_raw_application.go:ApplyNativeRawApplication")
}
