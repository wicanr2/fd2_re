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
    let mut queue = queue.to_vec();
    for (slot, descriptor) in [0x74, 0x75, 0x76, 0x76].into_iter().enumerate() {
        queue.push(NativePresentationDigit { position_code: [2, 8, 12, 17][slot], target, digit: descriptor });
    }
    queue
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
    if !(0..=0xff).contains(&digit_bias) {
        return Err(battle_err!("native presentation: malformed digit bias {digit_bias}"));
    }
    if !(0..=0xff).contains(&target) {
        return Err(battle_err!("native presentation: malformed target {target}"));
    }
    if !in_camera {
        return Ok(queue.to_vec());
    }
    let mut digits = value.to_string().into_bytes();
    if digits.len() > 4 {
        // The native loop still reads v12[0..3] when strlen(value)>3; preserve
        // that fixed-buffer behavior rather than normalizing to a number.
        digits.truncate(4);
    }
    let mut out = queue.to_vec();
    let n = digits.len();
    for i in 0..4usize {
        let mut digit = 0;
        if n > 3 - i {
            digit = digits[i + n - 4] as i64 + digit_bias - b'0' as i64;
        }
        out.push(NativePresentationDigit { position_code: 5 * i as i64 + 2, target, digit });
    }
    Ok(out)
}

#[cfg(test)]
mod native_presentation_queue_test;
