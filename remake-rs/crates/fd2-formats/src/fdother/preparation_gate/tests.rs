//! 移植自 `preparation_gate_test.go`。

use crate::fdother::*;

// Go: TestNativePreparationRecordGatesRemainRaw
#[test]
fn native_preparation_record_gates_remain_raw() {
    let mut raw = vec![0u8; 0x50];
    raw[0x25] = 1;
    raw[5] = 0;
    raw[6] = 7;
    raw[7] = 3;
    raw[0x40..0x42].copy_from_slice(&12u16.to_le_bytes());
    raw[0x42..0x44].copy_from_slice(&25u16.to_le_bytes());
    let mut record = parse_native_preparation_record(&raw).unwrap();
    assert!(native_preparation_eligible(record, 7) && !native_preparation_eligible(record, 8), "selector gates");
    assert_eq!(native_preparation_adjusted_word40(record), (10, 2));
    record.raw25 = 0;
    assert!(!native_preparation_eligible(record, 7), "raw +0x25 zero must reject");
    record.raw25 = 1;
    record.raw5 = 1;
    assert!(!native_preparation_eligible(record, 7), "raw +5 bit0 must reject");
}

// Go: TestNativePreparationAdjustedWord40Clamps
#[test]
fn native_preparation_adjusted_word40_clamps() {
    let record = NativePreparationRecord { word40: 3, word42: 99, ..Default::default() };
    assert_eq!(native_preparation_adjusted_word40(record), (0, 9));
}
