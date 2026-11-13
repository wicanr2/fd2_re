//! 移植自 `remake/internal/battle/native_coordinate_write_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestSetNativeUnitCoordinateBytesWritesRawRecordPrefix`
#[test]
fn set_native_unit_coordinate_bytes_writes_raw_record_prefix() {
    let mut records = vec![0u8; 2 * RS];
    set_native_unit_coordinate_bytes(&mut records, 1, 0x21, 0x37).unwrap();
    assert!(records[RS] == 0x21 && records[RS + 1] == 0x37, "coords = {:#x} {:#x}", records[RS], records[RS + 1]);
}

/// Go：`TestSetNativeUnitCoordinateBytesRejectsBadUnitBeforeWrite`
#[test]
fn set_native_unit_coordinate_bytes_rejects_bad_unit_before_write() {
    let mut records = vec![0u8; RS];
    assert!(set_native_unit_coordinate_bytes(&mut records, 1, 1, 2).is_err(), "expected bad unit index");
    assert!(records[0] == 0 && records[1] == 0, "bad input mutated record");
}
