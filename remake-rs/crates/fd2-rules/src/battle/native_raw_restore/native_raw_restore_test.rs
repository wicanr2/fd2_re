//! 移植自 `remake/internal/battle/native_raw_restore_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestApplyNativeRawHPRestoreMatches1C916Core`
#[test]
fn apply_native_raw_hp_restore_matches1_c916_core() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x40, 10);
    put16(&mut records, 0x42, 100);
    (records[0x20], records[0x21], records[7]) = (9, 2, 0);
    let got = apply_native_raw_hp_restore(&mut records, 0, 20, 0).unwrap();
    assert!(
        got.rolled == 18 && got.actual == 18 && got.score == 230 && got.rng_state != 0 && get16(&records, 0x40) == 28,
        "result={got:?} hp={}",
        get16(&records, 0x40)
    );
}

/// Go：`TestApplyNativeRawHPRestoreClampsAndPreflights`
#[test]
fn apply_native_raw_hp_restore_clamps_and_preflights() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x40, 95);
    put16(&mut records, 0x42, 100);
    let mut before = records.clone();
    let got = apply_native_raw_hp_restore(&mut records, 0, 100, 0x1234).unwrap();
    assert!(got.actual == 5 && get16(&records, 0x40) == 100, "result={got:?} hp={}", get16(&records, 0x40));
    assert!(apply_native_raw_hp_restore(&mut before[..RS - 1], 0, 1, 0).is_err(), "short record unexpectedly accepted");
    assert!(apply_native_raw_hp_restore(&mut records, -1, 1, 0).is_err(), "negative unit unexpectedly accepted");
}

/// Go：`TestApplyNativeRawMPRestoreUsesMPOffsetsAndNoClassBonus`
#[test]
fn apply_native_raw_mp_restore_uses_mp_offsets_and_no_class_bonus() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x44, 10);
    put16(&mut records, 0x46, 100);
    (records[0x20], records[0x21], records[7]) = (9, 2, 0);
    let got = apply_native_raw_mp_restore(&mut records, 0, 20, 0).unwrap();
    assert!(
        got.actual == 18 && got.score == 14 && get16(&records, 0x44) == 28,
        "result={got:?} mp={}",
        get16(&records, 0x44)
    );
}

/// Go：`TestApplyNativeRawHPRestoreListMatches211A4Order`
#[test]
fn apply_native_raw_hp_restore_list_matches211_a4_order() {
    let mut records = vec![0u8; 2 * RS];
    for unit in 0..2 {
        let base = unit * RS;
        put16(&mut records, base + 0x40, 10 + unit as u16);
        put16(&mut records, base + 0x42, 100);
        (records[base + 0x20], records[base + 0x21], records[base + 7]) = (9, 2 + unit as u8, 0);
    }
    let mut want_records = records.clone();
    let first = apply_native_raw_hp_restore(&mut want_records, 1, 20, 0).unwrap();
    let second = apply_native_raw_hp_restore(&mut want_records, 0, 20, first.rng_state).unwrap();
    let got = apply_native_raw_hp_restore_list(&mut records, &[1, 0], 20, 0).unwrap();
    assert!(
        got.rng_state == second.rng_state && got.score == first.score + second.score && got.results.len() == 2,
        "batch={got:?} first={first:?} second={second:?}"
    );
    assert_eq!(records, want_records, "batch mutation order differs from 0x211a4 sequential loop");
}

/// Go：`TestApplyNativeRawHPRestoreListPreflightsAtomically`
#[test]
fn apply_native_raw_hp_restore_list_preflights_atomically() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x40, 10);
    put16(&mut records, 0x42, 100);
    let before = records.clone();
    assert!(
        apply_native_raw_hp_restore_list(&mut records, &[0, 1], 20, 0).is_err(),
        "invalid later target unexpectedly accepted"
    );
    assert_eq!(records, before, "invalid target list partially mutated an earlier record");
}
