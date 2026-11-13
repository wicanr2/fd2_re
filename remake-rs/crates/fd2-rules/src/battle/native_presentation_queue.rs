//! Go 來源：`remake/internal/battle/native_presentation_queue.go`（56 行）。

use super::*;

/// NativePresentationDigit is one entry emitted by FD2's 0x1e0db numeric
/// presentation queue. PositionCode, Target and Digit are intentionally raw;
/// the renderer owns their eventual screen meaning.
///
/// Go：`native_presentation_queue.go:8`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePresentationDigit {
    pub position_code: i64,
    pub target: i64,
    pub digit: i64,
}

/// AppendNativePresentationMiss preserves sub_1E1DC's raw descriptor table at
/// LE linear 0x5204A. The bytes 74 75 76 76 are hexadecimal, not decimal.
/// See docs/data/ida/fd2_command9_player_ai_presentation_ida.txt.
///
/// Go：`native_presentation_queue.go:17`，`func AppendNativePresentationMiss(queue []NativePresentationDigit, target int) []NativePresentationDigit`
pub fn append_native_presentation_miss(queue: &[NativePresentationDigit], target: i64) -> Vec<NativePresentationDigit> {
    todo!("battle/native_presentation_queue.go:AppendNativePresentationMiss")
}

/// AppendNativePresentationDigits reproduces the data-only part of 0x1e0db.
/// The native caller performs the camera test before entering the formatter;
/// inCamera therefore controls the native no-op path. Digits are right
/// aligned in four slots and each non-leading digit receives digitBias-48.
/// It does not assign HP/MP/damage/heal semantics or render anything.
///
/// Go：`native_presentation_queue.go:31`，`func AppendNativePresentationDigits(queue []NativePresentationDigit, value, digitBias, target int, inCamera bool) ([]NativePresentationDigit, error)`
pub fn append_native_presentation_digits(
    queue: &[NativePresentationDigit],
    value: i64,
    digit_bias: i64,
    target: i64,
    in_camera: bool,
) -> Result<Vec<NativePresentationDigit>> {
    todo!("battle/native_presentation_queue.go:AppendNativePresentationDigits")
}
