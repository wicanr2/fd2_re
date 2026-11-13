//! 移植自 `remake/internal/battle/native_raw_application_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestApplyNativeRawApplicationMatches22D1B`
#[test]
fn apply_native_raw_application_matches22_d1_b() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x40, 20);
    (records[0x20], records[0x21], records[0x25]) = (1, 3, 0);
    let (got, state, score) = apply_native_raw_application(&mut records, &[0], 0x25, 0).unwrap();
    assert!(
        got.len() == 1
            && got[0].applied
            && (2..=5).contains(&got[0].marker)
            && score == 24
            && state != 0
            && get16(&records, 0x40) == 11,
        "results={got:?} state={state:#x} score={score} hp={}",
        get16(&records, 0x40)
    );
}

/// Go：`TestApplyNativeRawApplicationSkipsGates`
#[test]
fn apply_native_raw_application_skips_gates() {
    let mut records = vec![0u8; 2 * RS];
    (records[0x20], records[RS + 0x20]) = (0x19, 1);
    (records[0x25], records[RS + 0x25]) = (1, 1);
    let (got, _, score) = apply_native_raw_application(&mut records, &[0, 1], 0x25, 0).unwrap();
    assert!(got.len() == 2 && !got[0].applied && !got[1].applied && score == 0, "results={got:?} score={score}");
}
