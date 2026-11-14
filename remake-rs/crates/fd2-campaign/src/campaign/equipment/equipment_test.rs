//! 移植自 `remake/internal/campaign/equipment_test.go`。

use super::*;
use crate::campaign::church::church_test::unit;

fn stats(rows: &[(i64, ItemStats)]) -> BTreeMap<i64, ItemStats> {
    rows.iter().copied().collect()
}

/// Go：`TestEquipItemRecomputesFromBaseAndReplacesEXECategory`
#[test]
fn equip_item_recomputes_from_base_and_replaces_exe_category() {
    let mut u = unit! {
        ap: 16,
        dp: 12,
        hit: 97,
        ev: 2,
        mv: 4,
        base_ap: 6,
        base_dp: 4,
        base_hit: 2,
        base_ev: 2,
        base_mv: 4,
        equipment_base_set: true,
        inventory: vec![0, 1, 132],
        inventory_slots: vec![0, 0xff, 1, 132, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0, 0x80, 0, 0, 0x80, 0x80, 0x80, 0x80],
    };
    let stats = stats(&[
        (0, ItemStats { r#type: 1, ap: 10, hit: 95, ..Default::default() }),
        (1, ItemStats { r#type: 1, ap: 20, hit: 95, ..Default::default() }),
        (132, ItemStats { r#type: 22, dp: 8, ..Default::default() }),
    ]);
    let err = equip_item(Some(&mut u), 0, &stats);
    assert!(err.is_ok() && u.ap == 16 && u.hit == 97, "initial equip err={err:?} AP={} HIT={}", u.ap, u.hit);
    let err = equip_item(Some(&mut u), 1, &stats);
    assert!(
        err.is_ok() && u.ap == 26 && u.hit == 97 && !u.equipped[0] && u.equipped[1],
        "replacement err={err:?} AP={} HIT={} flags={:?}",
        u.ap,
        u.hit,
        u.equipped
    );
    assert!(
        u.native_inventory_flags[0] & 0x40 == 0
            && u.native_inventory_flags[2] & 0x40 != 0
            && u.native_inventory_flags[3] & 0x40 == 0,
        "raw equipped flags did not follow compact replacement: {:?}",
        u.native_inventory_flags
    );
}

/// Go：`TestInitializeEquipmentBaseSubtractsAuthoredStartingGearOnce`
#[test]
fn initialize_equipment_base_subtracts_authored_starting_gear_once() {
    let mut u = unit! {
        ap: 16,
        dp: 12,
        hit: 97,
        ev: 2,
        inventory: vec![0, 132],
        equipped: vec![true, true],
    };
    let stats = stats(&[
        (0, ItemStats { r#type: 1, ap: 10, hit: 95, ..Default::default() }),
        (132, ItemStats { r#type: 22, dp: 8, ..Default::default() }),
    ]);
    initialize_equipment_base(Some(&mut u), &stats);
    assert!(
        u.equipment_base_set
            && u.base_ap == 6
            && u.base_dp == 4
            && u.base_hit == 2
            && u.ap == 16
            && u.dp == 12
            && u.hit == 97,
        "base conversion = base({},{},{}) effective({},{},{})",
        u.base_ap,
        u.base_dp,
        u.base_hit,
        u.ap,
        u.dp,
        u.hit
    );
    initialize_equipment_base(Some(&mut u), &stats);
    assert!(u.base_ap == 6 && u.ap == 16, "base conversion repeated: base={} AP={}", u.base_ap, u.ap);
}

/// Go：`TestRecomputeAfterClassChangeUsesDXForHitAndEVBase`
#[test]
fn recompute_after_class_change_uses_dx_for_hit_and_ev_base() {
    let mut u = unit! {
        ap: 10,
        dp: 8,
        dx: 12,
        hit: 99,
        ev: 77,
        mv: 4,
        inventory: vec![1],
        equipped: vec![true],
    };
    let stats = stats(&[(1, ItemStats { ap: 3, dp: 2, hit: 5, ev: 7, ..Default::default() })]);
    recompute_after_class_change(Some(&mut u), &stats);
    assert!(
        u.base_hit == 12 && u.base_ev == 12 && u.hit == 17 && u.ev == 19,
        "class base synthesis hit/ev base={}/{} effective={}/{}",
        u.base_hit,
        u.base_ev,
        u.hit,
        u.ev
    );
    assert!(u.ap == 13 && u.dp == 10, "class base AP/DP effective={}/{}", u.ap, u.dp);
}

/// Go：`TestRecomputeAfterClassChangeDoesNotDoubleCountExistingEquipment`
#[test]
fn recompute_after_class_change_does_not_double_count_existing_equipment() {
    let mut u = unit! {
        ap: 18,
        dp: 12,
        dx: 14,
        mv: 5,
        inventory: vec![1],
        equipped: vec![true],
        equipment_base_set: true,
        base_ap: 15,
        base_dp: 10,
        base_mv: 5,
    };
    let stats = stats(&[(1, ItemStats { r#type: 1, ap: 3, dp: 2, ..Default::default() })]);
    recompute_after_class_change(Some(&mut u), &stats);
    assert!(
        u.base_ap == 15 && u.base_dp == 10 && u.ap == 18 && u.dp == 12,
        "existing equipment double-counted: base={}/{} effective={}/{}",
        u.base_ap,
        u.base_dp,
        u.ap,
        u.dp
    );
}

/// Go：`TestApplyEquippedAttackRangeUsesEquippedWeaponOnly`
#[test]
fn apply_equipped_attack_range_uses_equipped_weapon_only() {
    let stats = stats(&[
        (0x14, ItemStats { r#type: 3, min: 1, max: 2, ..Default::default() }),
        (0x80, ItemStats { r#type: 5, ..Default::default() }),
    ]);
    let mut u = unit! { inventory: vec![0x14, 0x80], equipped: vec![true, true], ap: 91, };
    apply_equipped_attack_range(Some(&mut u), &stats);
    assert!(
        u.atk_min == 1 && u.atk_max == 2 && u.base_atk_min == 1 && u.base_atk_max == 2 && u.ap == 91,
        "equipped spear range = [{},{}] base [{},{}] AP={}",
        u.atk_min,
        u.atk_max,
        u.base_atk_min,
        u.base_atk_max,
        u.ap
    );
    u.equipped[0] = false;
    apply_equipped_attack_range(Some(&mut u), &stats);
    assert!(
        u.atk_min == 0 && u.atk_max == 0,
        "unequipped weapon must fall back to the default range, got [{},{}]",
        u.atk_min,
        u.atk_max
    );
}

/// Go：`TestApplyEquippedAttackRangeIgnoresArmorRange`
///
/// 洛娜（第八章 FDFIELD group 0）裝槍 22（range 1..2）與防具 135（item.json range 1..1）：
/// 射程只看 ID < 0x80 的武器，防具不能把 2 格蓋回 1。
#[test]
fn apply_equipped_attack_range_ignores_armor_range() {
    let stats = stats(&[
        (22, ItemStats { r#type: 3, ap: 50, min: 1, max: 2, ..Default::default() }),
        (135, ItemStats { r#type: 22, dp: 48, min: 1, max: 1, ..Default::default() }),
    ]);
    let mut u = unit! { inventory: vec![22, 135], equipped: vec![true, true], };
    apply_equipped_attack_range(Some(&mut u), &stats);
    assert!(u.atk_min == 1 && u.atk_max == 2, "射程 {}..{}，要 1..2", u.atk_min, u.atk_max);
    let mut u = unit! { inventory: vec![135, 22], equipped: vec![true, true], equipment_base_set: true, };
    recompute_equipment(Some(&mut u), &stats);
    assert!(u.atk_min == 1 && u.atk_max == 2, "RecomputeEquipment 射程 {}..{}，要 1..2", u.atk_min, u.atk_max);
}
