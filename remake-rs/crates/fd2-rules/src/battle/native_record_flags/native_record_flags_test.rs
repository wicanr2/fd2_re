//! 移植自 `remake/internal/battle/native_record_flags_test.go`。

use super::*;

/// Go：`TestNativeRecordBit7SetAndClearPreserveOtherBits`
#[test]
fn native_record_bit7_set_and_clear_preserve_other_bits() {
    let mut records = vec![0u8; 2 * 0x50];
    (records[5], records[0x50 + 5]) = (0x01, 0xff);
    set_native_record_bit7(&mut records, 1).unwrap();
    assert!(
        records[5] == 0x01 && records[0x50 + 5] == 0xff,
        "set touched wrong bytes: {:#x} {:#x}",
        records[5],
        records[0x55]
    );
    clear_native_record_bit7_all(&mut records, 2).unwrap();
    assert!(records[5] == 0x01 && records[0x50 + 5] == 0x7f, "clear result={:#x} {:#x}", records[5], records[0x55]);
}

/// Go：`TestNativeRecordFlagsFailClosedOnShortBuffers`
#[test]
fn native_record_flags_fail_closed_on_short_buffers() {
    assert!(set_native_record_bit7(&mut [0u8; 0x50], 1).is_err(), "out-of-range set must fail");
    assert!(clear_native_record_bit7_all(&mut [0u8; 0x4f], 1).is_err(), "short clear buffer must fail");
}
