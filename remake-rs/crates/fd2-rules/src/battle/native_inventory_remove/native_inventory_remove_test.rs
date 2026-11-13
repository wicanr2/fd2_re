//! 移植自 `remake/internal/battle/native_inventory_remove_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestRemoveNativeInventorySlotShiftsRawCellsAndMarksTailFree`
#[test]
fn remove_native_inventory_slot_shifts_raw_cells_and_marks_tail_free() {
    let mut records = vec![0u8; RS];
    for slot in 0..8 {
        let off = 0x0a + slot * 2;
        records[off] = slot as u8;
        records[off + 1] = 0xa0 + slot as u8;
    }
    remove_native_inventory_slot(&mut records, 0, 2).unwrap();
    for slot in 2..7 {
        let off = 0x0a + slot * 2;
        assert!(
            records[off] == slot as u8 + 1 && records[off + 1] == 0xa0 + slot as u8 + 1,
            "slot {slot} = {:#x} {:#x}",
            records[off],
            records[off + 1]
        );
    }
    assert!(records[0x18] == 0x80 && records[0x19] == 0xa7, "tail = {:#x} {:#x}", records[0x18], records[0x19]);
}

/// Go：`TestRemoveNativeInventorySlotKeepsTailItemByteForLastSlot`
#[test]
fn remove_native_inventory_slot_keeps_tail_item_byte_for_last_slot() {
    let mut records = vec![0u8; RS];
    (records[0x18], records[0x19]) = (0x40, 0xd3);
    remove_native_inventory_slot(&mut records, 0, 7).unwrap();
    assert!(records[0x18] == 0x80 && records[0x19] == 0xd3, "tail = {:#x} {:#x}", records[0x18], records[0x19]);
}

/// Go：`TestRemoveNativeInventorySlotRejectsInvalidInput`
#[test]
fn remove_native_inventory_slot_rejects_invalid_input() {
    assert!(remove_native_inventory_slot(&mut [0u8; RS], 0, 8).is_err(), "expected invalid slot to fail closed");
    assert!(remove_native_inventory_slot(&mut [0u8; 0x1a - 1], 0, 0).is_err(), "expected short record to fail closed");
}
