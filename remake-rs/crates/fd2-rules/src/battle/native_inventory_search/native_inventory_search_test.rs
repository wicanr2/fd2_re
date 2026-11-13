//! 移植自 `remake/internal/battle/native_inventory_search_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestFindNativeInventoryItemUsesRawPrefixAndDoesNotMutate`
#[test]
fn find_native_inventory_item_uses_raw_prefix_and_does_not_mutate() {
    let mut records = vec![0u8; 16 * RS];
    let base = 3 * RS;
    // Three clear flags make the native search prefix length three.
    records[base + 0x0a] = 0;
    records[base + 0x0b] = 0x64;
    records[base + 0x0c] = 0;
    records[base + 0x0d] = 0x21;
    records[base + 0x0e] = 0;
    records[base + 0x0f] = 0x37;
    // A matching item after the native prefix must not be found.
    records[base + 0x10] = 0x80;
    records[base + 0x11] = 0x64;
    let before = records.clone();
    assert_eq!(find_native_inventory_item(&records, 0x64), Ok((3, 0)));
    assert_eq!(records, before, "native inventory search mutated records");
}

/// Go：`TestFindNativeInventoryItemMissingAndBounds`
#[test]
fn find_native_inventory_item_missing_and_bounds() {
    let records = vec![0u8; 16 * RS];
    assert_eq!(find_native_inventory_item(&records, 0x64), Ok((-1, -1)));
    assert!(find_native_inventory_item_in_unit(&[0u8; RS], 1, 1).is_err(), "short unit record unexpectedly accepted");
    assert!(
        find_native_inventory_item(&vec![0u8; 16 * RS - 1], 1).is_err(),
        "short 16-slot buffer unexpectedly accepted"
    );
}

/// Go：`TestFindNativeInventoryItemInUnitRespectsZeroOccupiedCount`
#[test]
fn find_native_inventory_item_in_unit_respects_zero_occupied_count() {
    let mut records = vec![0u8; RS];
    for slot in 0..8 {
        records[0x0a + slot * 2] = 0x80;
        records[0x0b + slot * 2] = 0x64;
    }
    // All flags are reserved-empty => 0x1b8a6 returns zero, so 0x31860 does not scan.
    assert_eq!(find_native_inventory_item_in_unit(&records, 0, 0x64), Ok(-1));
}
