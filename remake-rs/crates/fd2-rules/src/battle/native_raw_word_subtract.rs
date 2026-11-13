//! Go 來源：`remake/internal/battle/native_raw_word_subtract.go`（28 行）。

use super::*;

/// ApplyNativeRawWordSubtract reproduces the arithmetic core used by 0x1ca89:
/// subtract a caller-supplied byte-sized value from a selected record word and
/// store the low 16 bits. The word offset remains explicit because the native
/// command-record lookup is caller-owned. It is not part of 0x1cac7, which is
/// an indexed presentation helper.
///
/// Go：`native_raw_word_subtract.go:13`，`func ApplyNativeRawWordSubtract(records []byte, unitIndex, wordOffset, amount int) (uint16, error)`
pub fn apply_native_raw_word_subtract(
    records: &mut [u8],
    unit_index: i64,
    word_offset: i64,
    amount: i64,
) -> Result<u16> {
    todo!("battle/native_raw_word_subtract.go:ApplyNativeRawWordSubtract")
}
