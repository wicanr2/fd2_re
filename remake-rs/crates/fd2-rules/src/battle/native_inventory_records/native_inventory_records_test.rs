//! 移植自 `remake/internal/battle/native_inventory_records_test.go`。

use super::*;

/// Go：`TestNativeInventoryRecordsMaterializesOnlyProvenCells`
#[test]
fn native_inventory_records_materializes_only_proven_cells() {
    let u = Unit {
        inventory_slots: vec![0x64, 0xff, 2, 3, 4, 5, 6, 7],
        native_inventory_flags: vec![0, 0x80, 0, 0, 0, 0, 0, 0x80],
        ..Default::default()
    };
    let records = native_inventory_records(&[u], 1).unwrap();
    assert!(
        records[0x0a] == 0 && records[0x0b] == 0x64 && records[0x0a + 2] == 0x80 && records[0x0b + 2] == 0xff,
        "raw inventory cells={:#x} {:#x} {:#x} {:#x}",
        records[0x0a],
        records[0x0b],
        records[0x0a + 2],
        records[0x0b + 2]
    );
    assert_eq!(find_native_inventory_item_in_unit(&records, 0, 0x64), Ok(0), "native search");
}

/// Go：`TestNativeInventoryRecordsFailsClosedWithoutProvenance`
#[test]
fn native_inventory_records_fails_closed_without_provenance() {
    let u = Unit { inventory: vec![0x64], ..Default::default() };
    assert!(native_inventory_records(&[u], 1).is_err(), "normalized-only inventory must not be serialized as raw");
}
