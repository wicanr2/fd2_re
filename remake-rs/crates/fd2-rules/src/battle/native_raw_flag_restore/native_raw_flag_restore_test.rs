//! 移植自 `remake/internal/battle/native_raw_flag_restore_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestApplyNativeRawFlagRestoreMatches22AF6`
#[test]
fn apply_native_raw_flag_restore_matches22_af6() {
    let mut records = vec![0u8; 2 * RS];
    for i in 0..2 {
        let base = i * RS;
        put16(&mut records, base + 0x40, 10);
        put16(&mut records, base + 0x42, 100);
        (records[base + 0x20], records[base + 0x21]) = (9, 2);
    }
    records[0x25] = 1;
    let (got, state, score) = apply_native_raw_flag_restore(&mut records, &[0, 1], 0x25, 0).unwrap();
    assert!(
        got.len() == 2
            && got[0].applied
            && !got[1].applied
            && state != 0
            && score == 128
            && records[0x25] == 0
            && records[RS + 0x25] == 0
            && get16(&records, 0x40) == 19,
        "results={got:?} state={state:#x} score={score} marker={:#x} hp={}",
        records[0x25],
        get16(&records, 0x40)
    );
}

/// Go：`TestApplyNativeRawFlagRestorePreflights`
#[test]
fn apply_native_raw_flag_restore_preflights() {
    assert!(
        apply_native_raw_flag_restore(&mut [0u8; RS], &[0], NATIVE_RECORD_SIZE, 0).is_err(),
        "invalid marker offset unexpectedly accepted"
    );
    assert!(
        apply_native_raw_flag_restore(&mut [0u8; RS], &[1], 0x25, 0).is_err(),
        "out-of-range target unexpectedly accepted"
    );
}
