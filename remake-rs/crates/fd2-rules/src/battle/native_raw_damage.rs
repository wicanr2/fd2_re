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
    todo!("battle/native_raw_damage.go:ApplyNativeRawHPDamage")
}
