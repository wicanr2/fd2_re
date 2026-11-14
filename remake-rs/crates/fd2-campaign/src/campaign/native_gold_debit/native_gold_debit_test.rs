//! 移植自 `remake/internal/campaign/native_gold_debit_test.go`。

use super::*;

/// Go：`nativeGoldTestStrip`
fn native_gold_test_strip() -> fdother::RawCell {
    let mut pixels = vec![0u8; 6 * 99];
    for row in 0..99 {
        for col in 0..6 {
            pixels[row * 6 + col] = (row + 1) as u8;
        }
    }
    fdother::RawCell { width: 6, height: 99, pixels }
}

const ROLL: usize = NATIVE_SHOP_GOLD_ROLL_OFFSET as usize;

fn assert_inside_digit_windows(frames: &[Vec<u8>]) {
    for (index, got) in frames.iter().enumerate() {
        assert!(got[ROLL - 1] == 0 && got[ROLL + 8 * 6] == 0, "frame{index} wrote outside eight digit windows");
    }
}

/// Go：`TestComposeNativeGoldDebitFramesPreservesOdometerSchedule`
#[test]
fn compose_native_gold_debit_frames_preserves_odometer_schedule() {
    let base = vec![0u8; (NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT) as usize];
    let (frames, next) = compose_native_gold_debit_frames(&base, &native_gold_test_strip(), 1000, 50).unwrap();
    // 00001000 -> 00000950: the slowest digit descends five values, with
    // nine vertical-strip phases per value.
    assert!(next == 950 && frames.len() == 45, "next={next} frames={}, want 950/45", frames.len());
    // Hundreds and tens both roll on the first pass. The first phase begins
    // at row digit*9+8; the ninth finishes at the stable digit row.
    assert_eq!(frames[0][ROLL + 4 * 6], 9, "first hundreds roll row, want strip row8 value9");
    assert_eq!(frames[8][ROLL + 4 * 6], 1, "final hundreds row, want strip row0 value1");
    assert_eq!(frames[8][ROLL + 5 * 6], 82, "final tens row, want strip row81 value82");
    assert_eq!(frames[0][ROLL + 5 * 6], 90, "wrapped tens first row, want strip row89 value90");
    assert_eq!(frames[9][ROLL + 6 * 6], 81, "second roll first row, want strip row80 value81");
    // The units digit never differs and must preserve its background.
    assert_eq!(frames[44][ROLL + 7 * 6], 0, "unchanged units roll pixel, want preserved 0");
    assert_inside_digit_windows(&frames);
    // The animated strip starts one VGA row above the stable digit baseline:
    // 0xa7a90 -> (16,98), while NativeShopGoldOffset is (16,99).
    assert_eq!(
        NATIVE_SHOP_GOLD_OFFSET - NATIVE_SHOP_GOLD_ROLL_OFFSET,
        NATIVE_SHOP_WIDTH,
        "native stable/rolling gold offsets lost their one-row boundary"
    );
    let (cascade, next) = compose_native_gold_debit_frames(&base, &native_gold_test_strip(), 10_000_000, 1).unwrap();
    assert!(
        next == 9_999_999 && cascade.len() == 9,
        "borrow cascade = frames{} next{next}, want 9/9999999",
        cascade.len()
    );
}

/// Go：`TestComposeNativeGoldCreditFramesPreservesOdometerSchedule`
#[test]
fn compose_native_gold_credit_frames_preserves_odometer_schedule() {
    let base = vec![0u8; (NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT) as usize];
    let (frames, next) = compose_native_gold_credit_frames(&base, &native_gold_test_strip(), 0, 37).unwrap();
    // 00000000 -> 00000037: tens and units rise together to 33, then
    // only units continue. Seven value steps, each with nine strip phases.
    assert!(next == 37 && frames.len() == 63, "next={next} frames={}, want 37/63", frames.len());
    assert_eq!(frames[0][ROLL + 6 * 6], 2, "first tens roll row, want strip row1 value2");
    assert_eq!(frames[8][ROLL + 6 * 6], 10, "first tens stable row, want strip row9 value10");
    assert_eq!(frames[26][ROLL + 6 * 6], 28, "third tens stable row, want strip row27 value28");
    assert_eq!(frames[62][ROLL + 7 * 6], 64, "final units row, want strip row63 value64");
    // Tens stop after 33 and must remain unchanged while units reach 37.
    assert_eq!(frames[62][ROLL + 6 * 6], frames[26][ROLL + 6 * 6], "stable tens changed after 33");
    assert_inside_digit_windows(&frames);
    let (wrap, next) = compose_native_gold_credit_frames(&base, &native_gold_test_strip(), 9_999_999, 1).unwrap();
    assert!(next == 10_000_000 && wrap.len() == 9, "carry cascade = frames{} next{next}, want 9/10000000", wrap.len());
}

/// Go：`TestComposeNativeGoldDebitFramesRejectsInvalidState`
#[test]
fn compose_native_gold_debit_frames_rejects_invalid_state() {
    let base = vec![0u8; (NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT) as usize];
    let strip = native_gold_test_strip();
    let short = fdother::RawCell { width: 6, height: 98, ..Default::default() };
    for (i, (b, s, old, amount)) in [
        (&base[..1], &strip, 100, 1),
        (&base[..], &short, 100, 1),
        (&base[..], &strip, -1, 1),
        (&base[..], &strip, 100, 101),
        (&base[..], &strip, 100_000_000, 1),
    ]
    .into_iter()
    .enumerate()
    {
        assert!(compose_native_gold_debit_frames(b, s, old, amount).is_err(), "invalid state #{i} accepted");
    }
    let (frames, next) = compose_native_gold_debit_frames(&base, &strip, 100, 0).unwrap();
    assert!(next == 100 && frames.is_empty(), "zero debit = frames{} next{next}", frames.len());
}

/// Go：`TestComposeNativeGoldCreditFramesRejectsInvalidState`
#[test]
fn compose_native_gold_credit_frames_rejects_invalid_state() {
    let base = vec![0u8; (NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT) as usize];
    let strip = native_gold_test_strip();
    let short = fdother::RawCell { width: 6, height: 98, ..Default::default() };
    for (i, (b, s, old, amount)) in [
        (&base[..1], &strip, 100, 1),
        (&base[..], &short, 100, 1),
        (&base[..], &strip, -1, 1),
        (&base[..], &strip, 99_999_999, 1),
        (&base[..], &strip, 100, -1),
    ]
    .into_iter()
    .enumerate()
    {
        assert!(compose_native_gold_credit_frames(b, s, old, amount).is_err(), "invalid state #{i} accepted");
    }
    let (frames, next) = compose_native_gold_credit_frames(&base, &strip, 100, 0).unwrap();
    assert!(next == 100 && frames.is_empty(), "zero credit = frames{} next{next}", frames.len());
}

/// Go：`TestNativeGoldRollWindowPlanCarriesDirectionAndStationaryDigits`
#[test]
fn native_gold_roll_window_plan_carries_direction_and_stationary_digits() {
    for (old, next, first, last, count, credit) in [
        (1000, 950, 89, 45, 45, false),
        (990, 997, 1, 63, 63, true),
        (10_000_000, 9_999_999, 89, 81, 9, false),
        (9_999_999, 10_000_000, 82, 90, 9, true),
    ] {
        let windows = plan_native_gold_roll(old, next, credit).unwrap();
        assert_eq!(windows.len(), count, "plan old={old} next={next}: count");
        let index = if old == 1000 { 6 } else { 7 };
        assert!(
            windows[0].rows[index] == first && windows[windows.len() - 1].rows[index] == last,
            "wrong literal window endpoints for old={old} next={next}"
        );
    }
    for (old, next, credit) in [(-1, 0, true), (0, 100_000_000, true), (10, 9, true), (9, 10, false)] {
        assert!(plan_native_gold_roll(old, next, credit).is_err(), "invalid direction/range accepted");
    }
    let windows = plan_native_gold_roll(12, 12, true);
    assert!(windows.is_ok_and(|w| w.is_empty()), "zero transaction generated animation");
}
