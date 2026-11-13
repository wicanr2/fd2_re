//! 移植自 `remake/internal/battle/native_item_capacity_unit_test.go`。

use super::*;

/// Go：`TestApplyNativeItemCapacityToUnitPreservesCurrentAndEXP`
#[test]
fn apply_native_item_capacity_to_unit_preserves_current_and_exp() {
    let mut units = vec![Unit {
        hp: 40,
        max_hp: 100,
        mp: 7,
        max_mp: 20,
        mv: 255,
        exp: 73.0,
        equipment_base_set: true,
        base_mv: 255,
        inventory: vec![96, 0x20],
        equipped: vec![false, false],
        inventory_slots: vec![96, 0x20, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0, 0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        ..Default::default()
    }];
    let (route, _) = native_item_capacity_step_route_for_type(19, 1);
    let got = apply_native_item_capacity_to_unit(&mut units, Some(0), Some(0), route, 0).unwrap();
    let u = &units[0];
    assert!(
        got == 0
            && u.mv == 0
            && u.base_mv == 0
            && u.exp == 73.0
            && u.hp == 40
            && u.max_hp == 100
            && u.mp == 7
            && u.max_mp == 20,
        "capacity projection mismatch: {u:?}"
    );
    assert!(
        u.inventory.len() == 1 && u.inventory[0] == 0x20 && u.inventory_slots[0] == 0x20,
        "capacity item was not compact-removed: {u:?}"
    );
}

/// Go：`TestApplyNativeItemCapacityToUnitAddsMaximumOnly`
#[test]
fn apply_native_item_capacity_to_unit_adds_maximum_only() {
    let mut units = vec![Unit {
        hp: 40,
        max_hp: 100,
        inventory: vec![94],
        inventory_slots: vec![94, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        ..Default::default()
    }];
    let (route, _) = native_item_capacity_step_route_for_type(17, 20);
    let got = apply_native_item_capacity_to_unit(&mut units, Some(0), Some(0), route, 0);
    let u = &units[0];
    assert!(matches!(got, Ok(120)) && u.max_hp == 120 && u.hp == 40, "got={got:?} unit={u:?}");
}
