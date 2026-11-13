//! 移植自 `remake/internal/battle/native_presentation_queue_test.go`。

use super::*;

fn d(position_code: i64, target: i64, digit: i64) -> NativePresentationDigit {
    NativePresentationDigit { position_code, target, digit }
}

/// Go：`TestAppendNativePresentationMissUsesOriginalHexTable`
#[test]
fn append_native_presentation_miss_uses_original_hex_table() {
    let first = d(7, 1, 94);
    let queue = append_native_presentation_miss(&[first], 9);
    let want = [first, d(2, 9, 116), d(8, 9, 117), d(12, 9, 118), d(17, 9, 118)];
    assert_eq!(queue, want);
}

/// Go：`TestAppendNativePresentationDigitsRightAlignsRawDigits`
#[test]
fn append_native_presentation_digits_right_aligns_raw_digits() {
    let queue = append_native_presentation_digits(&[], 12, b'0' as i64, 7, true).unwrap();
    assert_eq!(queue.len(), 4, "queue len");
    let want_pos = [2, 7, 12, 17];
    let want_digit = [0, 0, b'1' as i64, b'2' as i64];
    for (i, entry) in queue.iter().enumerate() {
        assert!(
            entry.position_code == want_pos[i] && entry.target == 7 && entry.digit == want_digit[i],
            "entry {i}={entry:?}, want pos={} digit={}",
            want_pos[i],
            want_digit[i]
        );
    }
}

/// Go：`TestAppendNativePresentationDigitsBiasAndCameraGate`
#[test]
fn append_native_presentation_digits_bias_and_camera_gate() {
    let queue = append_native_presentation_digits(&[], 1234, 5, 2, true).unwrap();
    for (i, want) in [6, 7, 8, 9].into_iter().enumerate() {
        assert_eq!(queue[i].digit, want, "digit {i}");
    }
    let unchanged = append_native_presentation_digits(&queue, 99, b'0' as i64, 2, false);
    assert!(matches!(&unchanged, Ok(u) if u.len() == queue.len()), "off-camera append changed queue: {unchanged:?}");
    assert!(
        append_native_presentation_digits(&[], 1, 256, 0, true).is_err(),
        "invalid digit bias unexpectedly accepted"
    );
}

/// Go：`TestAppendNativePresentationDigitsPreservesNativeLongValueRead`
#[test]
fn append_native_presentation_digits_preserves_native_long_value_read() {
    let queue = append_native_presentation_digits(&[], 12345, b'0' as i64, 1, true).unwrap();
    for (i, want) in b"1234".iter().enumerate() {
        assert_eq!(queue[i].digit, *want as i64, "long value digit {i}");
    }
}
