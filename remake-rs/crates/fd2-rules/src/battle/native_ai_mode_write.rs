//! Go 來源：`remake/internal/battle/native_ai_mode_write.go`（33 行）。

use super::*;

/// SetNativeAIModeRange reproduces 0x3419c. It updates the low nibble of
/// runtime record +0x34 for every inclusive unit index while preserving the
/// high-nibble flags.
///
/// Go：`native_ai_mode_write.go:8`，`func SetNativeAIModeRange( records []byte, count, first, last int, mode byte, ) error`
pub fn set_native_ai_mode_range(records: &mut [u8], count: i64, first: i64, last: i64, mode: u8) -> Result<()> {
    todo!("battle/native_ai_mode_write.go:SetNativeAIModeRange")
}

/// SetNativeAIModeByte preserves the full-byte writer used by the mode-5
/// completion branch at 0x13d20 and several chapter handlers. Unlike
/// SetNativeAIModeRange, it intentionally replaces high-nibble flags.
///
/// Go：`native_ai_mode_write.go:27`，`func SetNativeAIModeByte(records []byte, count, unit int, value byte) error`
pub fn set_native_ai_mode_byte(records: &mut [u8], count: i64, unit: i64, value: u8) -> Result<()> {
    todo!("battle/native_ai_mode_write.go:SetNativeAIModeByte")
}
