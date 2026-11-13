//! 移植自 `remake/internal/battle/native_ai_mode_write_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestSetNativeAIModeRangePreservesHighNibble`
#[test]
fn set_native_ai_mode_range_preserves_high_nibble() {
    let mut records = vec![0u8; 3 * RS];
    records[0x34] = 0x80;
    records[RS + 0x34] = 0xc2;
    records[2 * RS + 0x34] = 0x45;
    set_native_ai_mode_range(&mut records, 3, 0, 1, 7).unwrap();
    assert!(
        records[0x34] == 0x87 && records[RS + 0x34] == 0xc7 && records[2 * RS + 0x34] == 0x45,
        "mode bytes={:#x}/{:#x}/{:#x}",
        records[0x34],
        records[RS + 0x34],
        records[2 * RS + 0x34]
    );
}

/// Go：`TestSetNativeAIModeByteReplacesWholeByte`
#[test]
fn set_native_ai_mode_byte_replaces_whole_byte() {
    let mut records = vec![0u8; RS];
    records[0x34] = 0xc2;
    set_native_ai_mode_byte(&mut records, 1, 0, 7).unwrap();
    assert!(records[0x34] == 7, "mode byte={:#x}", records[0x34]);
}

/// Go：`TestNativeAIModeWritersRejectBounds`
#[test]
fn native_ai_mode_writers_reject_bounds() {
    let mut records = vec![0u8; RS];
    assert!(set_native_ai_mode_range(&mut records, 1, 0, 1, 2).is_err(), "out-of-range mode span accepted");
    assert!(set_native_ai_mode_range(&mut records, 1, 0, 0, 0x10).is_err(), "wide mode accepted");
    assert!(set_native_ai_mode_byte(&mut records, 1, 1, 0).is_err(), "out-of-range mode unit accepted");
}
