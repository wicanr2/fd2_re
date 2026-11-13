//! 移植自 `remake/internal/battle/native_equipped_item_test.go`。

use super::*;

/// Go：`TestNativeEquippedInventorySlotMatches1B83D`
#[test]
fn native_equipped_inventory_slot_matches1_b83_d() {
    let flags = [0x40, 0x40, 0, 0x40, 0x80, 0x80, 0, 0x80];
    let slots = [0x90, 0x44, 0x55, 0xc0, 0xff, 0xff, 0x20, 0xff];
    assert_eq!(native_equipped_inventory_slot(&flags, &slots, 0), Ok(1), "low-item first equipped slot");
    assert_eq!(native_equipped_inventory_slot(&flags, &slots, 1), Ok(0), "high-item first equipped slot");
    assert!(native_equipped_inventory_slot(&flags, &slots, 0).unwrap() >= 0, "repeated lookup unexpectedly missing");
    assert!(native_equipped_inventory_slot(&flags[..7], &slots, 0).is_err(), "short input unexpectedly accepted");
}
