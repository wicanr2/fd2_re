//! 移植自 `remake/internal/battle/native_item_base_stat_unit_test.go`。

use super::*;

/// Go：`TestApplyNativeItemBaseStatDeltaToUnitWrapsAndCompacts`
#[test]
fn apply_native_item_base_stat_delta_to_unit_wraps_and_compacts() {
    let mut units = vec![Unit {
        base_ap: 32766,
        equipment_base_set: true,
        inventory: vec![0xc6, 0x20],
        equipped: vec![false, true],
        inventory_slots: vec![0xc6, 0x20, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        ..Default::default()
    }];
    let (route, _) = native_item_word_delta_route_for_type(8);
    let got = apply_native_item_base_stat_delta_to_unit(&mut units, Some(0), Some(0), route, 3, 0).unwrap();
    let u = &units[0];
    assert!(got == -32767 && u.base_ap == -32767, "wrapped AP={got}/{}", u.base_ap);
    assert!(
        u.inventory.len() == 1
            && u.inventory[0] == 0x20
            && u.inventory_slots[0] == 0x20
            && u.inventory_slots[7] == 0xff
            && u.native_inventory_flags[0] == 0x40
            && u.native_inventory_flags[7] == 0x80,
        "native compact removal failed: inventory={:?} slots={:?} flags={:?}",
        u.inventory,
        u.inventory_slots,
        u.native_inventory_flags
    );
}

/// Go：`TestApplyNativeItemBaseStatDeltaToUnitFailsAtomically`
#[test]
fn apply_native_item_base_stat_delta_to_unit_fails_atomically() {
    let mut units = vec![Unit {
        base_dp: 12,
        inventory: vec![0xc7],
        inventory_slots: vec![0xc7, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        ..Default::default()
    }];
    let (route, _) = native_item_word_delta_route_for_type(9);
    assert!(
        apply_native_item_base_stat_delta_to_unit(&mut units, Some(0), Some(0), route, 9, 0).is_err(),
        "missing equipment-base provenance was accepted"
    );
    let u = &units[0];
    assert!(
        u.base_dp == 12 && u.inventory.len() == 1 && u.inventory_slots[0] == 0xc7,
        "failed transaction mutated unit: {u:?}"
    );
}
