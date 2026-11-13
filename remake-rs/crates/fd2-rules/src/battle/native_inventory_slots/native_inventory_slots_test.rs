//! 移植自 `remake/internal/battle/native_inventory_slots_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeInventoryOccupiedCountUsesFlagBytesOnly`
#[test]
fn native_inventory_occupied_count_uses_flag_bytes_only() {
    let mut records = vec![0u8; RS];
    for slot in 0..8 {
        records[0x0a + slot * 2 + 1] = 0xff; // item bytes must not affect availability
    }
    (records[0x0a], records[0x0a + 4], records[0x0a + 14]) = (0x80, 0x80, 0x80);
    assert_eq!(native_inventory_occupied_count(&records, 0), Ok(5));
}

/// Go：`TestNativeInventoryOccupiedCountFailsClosed`
#[test]
fn native_inventory_occupied_count_fails_closed() {
    assert!(native_inventory_occupied_count(&[0u8; RS], 1).is_err(), "missing record must fail");
}
