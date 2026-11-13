//! 移植自 `remake/internal/battle/native_raw_damage_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestApplyNativeRawHPDamageMatches1C81FCore`
#[test]
fn apply_native_raw_hp_damage_matches1_c81_f_core() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x40, 20);
    let got = apply_native_raw_hp_damage(&mut records, 0, 10, 0).unwrap();
    assert!(
        got.rolled == 9 && got.actual == 9 && got.rng_state != 0 && get16(&records, 0x40) == 11,
        "result={got:?} hp={}",
        get16(&records, 0x40)
    );
}

/// Go：`TestApplyNativeRawHPDamageClampsAndPreflights`
#[test]
fn apply_native_raw_hp_damage_clamps_and_preflights() {
    let mut records = vec![0u8; RS];
    apply_native_raw_hp_damage(&mut records, 0, 10, 0).unwrap();
    assert_eq!(get16(&records, 0x40), 0, "damage did not clamp at zero");
    assert!(apply_native_raw_hp_damage(&mut records[..RS - 1], 0, 1, 0).is_err(), "short record unexpectedly accepted");
}
