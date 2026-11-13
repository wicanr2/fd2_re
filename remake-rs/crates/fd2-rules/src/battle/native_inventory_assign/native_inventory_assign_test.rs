//! 移植自 `remake/internal/battle/native_inventory_assign_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestAssignNativeReservedItemConsumesFirstReservedCell`
#[test]
fn assign_native_reserved_item_consumes_first_reserved_cell() {
    let mut records = vec![0u8; RS];
    (records[0x0a], records[0x0e]) = (0x80, 0x80);
    (records[0x0b], records[0x0f]) = (0x11, 0x22);
    let ok = assign_native_reserved_item(&mut records, 0, 0xc4);
    assert_eq!(ok, Ok(true), "assign");
    assert!(
        records[0x0a] == 0 && records[0x0b] == 0xc4 && records[0x0e] == 0x80 && records[0x0f] == 0x22,
        "cells={:#x} {:#x} {:#x} {:#x}",
        records[0x0a],
        records[0x0b],
        records[0x0e],
        records[0x0f]
    );
}

/// Go：`TestAssignNativeReservedItemReturnsFalseWhenNoReservedCell`
#[test]
fn assign_native_reserved_item_returns_false_when_no_reserved_cell() {
    assert_eq!(assign_native_reserved_item(&mut [0u8; RS], 0, 9), Ok(false));
}
