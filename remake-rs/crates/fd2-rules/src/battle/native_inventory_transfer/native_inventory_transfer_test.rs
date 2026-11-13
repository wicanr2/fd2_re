//! 移植自 `remake/internal/battle/native_inventory_transfer_test.go`。

use super::*;

fn inv(inventory: &[i64], equipped: &[bool], slots: &[i64], flags: &[i64]) -> Unit {
    Unit {
        inventory: inventory.to_vec(),
        equipped: equipped.to_vec(),
        inventory_slots: slots.to_vec(),
        native_inventory_flags: flags.to_vec(),
        ..Default::default()
    }
}

/// Go：`TestTransferNativeInventoryItemUsesFirstDestinationHole`
#[test]
fn transfer_native_inventory_item_uses_first_destination_hole() {
    let source = inv(
        &[0x44, 0x55],
        &[false, true],
        &[0x44, 0x55, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        &[0, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
    );
    let destination = inv(
        &[0x10],
        &[true],
        &[0xff, 0x10, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        &[0x80, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
    );
    let mut units = vec![source, destination];
    transfer_native_inventory_item(&mut units, Some(0), 0, Some(1)).unwrap();
    let (source, destination) = (&units[0], &units[1]);
    assert!(
        source.inventory == [0x55] && source.inventory_slots[0] == 0x55,
        "source was not compactly removed: inventory={:?} slots={:?}",
        source.inventory,
        source.inventory_slots
    );
    assert!(
        destination.inventory.len() == 2
            && destination.inventory[1] == 0x44
            && destination.inventory_slots[0] == 0x44
            && !destination.equipped[1],
        "destination did not receive unequipped item in first hole: inventory={:?} slots={:?} equipped={:?}",
        destination.inventory,
        destination.inventory_slots,
        destination.equipped
    );
    assert!(
        source.native_inventory_flags[0] == 0x40 && destination.native_inventory_flags[0] == 0,
        "native flags did not shift/clear: source={:?} destination={:?}",
        source.native_inventory_flags,
        destination.native_inventory_flags
    );
}

/// Go：`TestTransferNativeInventoryItemFullDestinationIsAtomic`
#[test]
fn transfer_native_inventory_item_full_destination_is_atomic() {
    let source = inv(
        &[0x44],
        &[false],
        &[0x44, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        &[0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
    );
    let destination = inv(&[1, 2, 3, 4, 5, 6, 7, 8], &[false; 8], &[1, 2, 3, 4, 5, 6, 7, 8], &[0; 8]);
    let mut units = vec![source, destination];
    assert!(
        transfer_native_inventory_item(&mut units, Some(0), 0, Some(1)).is_err(),
        "full destination unexpectedly accepted transfer"
    );
    assert!(
        units[0].inventory == [0x44] && units[1].inventory.len() == 8,
        "failed transfer mutated units: source={:?} destination={:?}",
        units[0].inventory,
        units[1].inventory
    );
}

/// Go：`TestTransferNativeInventoryItemRejectsDivergentProjection`
#[test]
fn transfer_native_inventory_item_rejects_divergent_projection() {
    let source = inv(
        &[0x44],
        &[false],
        &[0x55, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        &[0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
    );
    let destination = inv(&[], &[], &[0xff; 8], &[0x80; 8]);
    let mut units = vec![source, destination];
    assert!(
        transfer_native_inventory_item(&mut units, Some(0), 0, Some(1)).is_err(),
        "divergent native projection was accepted"
    );
    assert!(units[0].inventory[0] == 0x44 && units[1].inventory.is_empty(), "failed strict transfer mutated units");
}

/// Go：`TestTransferNativeInventoryItemAllowsNativeSelfReorder`
#[test]
fn transfer_native_inventory_item_allows_native_self_reorder() {
    let unit = inv(
        &[0x44, 0x55, 0x66],
        &[true, false, true],
        &[0x44, 0x55, 0x66, 0xff, 0xff, 0xff, 0xff, 0xff],
        &[0x40, 0, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80],
    );
    let mut units = vec![unit];
    transfer_native_inventory_item(&mut units, Some(0), 0, Some(0)).unwrap();
    let unit = &units[0];
    assert_eq!(unit.inventory, [0x55, 0x66, 0x44], "self transfer inventory");
    assert_eq!(unit.equipped, [false, true, false], "self transfer equipped");
    let got = &unit.native_inventory_flags;
    assert!(got[0] == 0 && got[1] == 0x40 && got[2] == 0 && got[3] == 0x80, "self transfer flags={got:?}");
}

/// Go `fullNativeRewardInventory`
fn full_native_reward_inventory() -> Unit {
    inv(
        &[0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17],
        &[true, false, true, false, false, true, false, true],
        &[0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17],
        &[0x40, 0, 0x40, 0, 0, 0x40, 0, 0x40],
    )
}

/// Go：`TestReplaceNativeFullInventoryRewardMatchesRemoveThenAppend`（`t.Run` 子測試改成迴圈）
#[test]
fn replace_native_full_inventory_reward_matches_remove_then_append() {
    for selected in [0usize, 3, 7] {
        let mut u = full_native_reward_inventory();
        replace_native_full_inventory_reward(Some(&mut u), selected as i64, 0xd3).unwrap();
        let mut want = vec![0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17];
        want.remove(selected);
        want.push(0xd3);
        for (i, &w) in want.iter().enumerate() {
            assert!(
                u.inventory[i] == w && u.inventory_slots[i] == w,
                "selected={selected} inventory={:?} slots={:?}",
                u.inventory,
                u.inventory_slots
            );
        }
        assert!(
            u.inventory.len() == 8 && u.equipped.len() == 8 && !u.equipped[7] && u.native_inventory_flags[7] == 0,
            "selected={selected} equipped={:?} flags={:?}",
            u.equipped,
            u.native_inventory_flags
        );
    }
}

/// Go：`TestReplaceNativeFullInventoryRewardRejectsInvalidStateAtomically`
#[test]
fn replace_native_full_inventory_reward_rejects_invalid_state_atomically() {
    let mutations: [fn(&mut Unit); 3] =
        [|u| u.inventory.truncate(7), |u| u.native_inventory_flags[4] = 0x80, |u| u.inventory_slots[2] = 0x77];
    for mutate in mutations {
        let mut u = full_native_reward_inventory();
        mutate(&mut u);
        let before = (u.inventory.clone(), u.inventory_slots.clone(), u.native_inventory_flags.clone());
        assert!(
            replace_native_full_inventory_reward(Some(&mut u), 2, 0xd3).is_err(),
            "invalid full inventory unexpectedly accepted replacement"
        );
        assert!(
            (u.inventory.clone(), u.inventory_slots.clone(), u.native_inventory_flags.clone()) == before,
            "rejected replacement mutated unit: inventory={:?} slots={:?} flags={:?}",
            u.inventory,
            u.inventory_slots,
            u.native_inventory_flags
        );
    }
}

/// Go：`TestRemoveNativeCompactInventoryKeepsTailItemByte`——釘住 0x1B8E7：左移之後只把第 8 格
/// 的旗標寫成 0x80，item byte 留著移走前第 8 格的值（第四章 r9 酒店存檔收據 rec0：
/// 賣掉第 1 格後尾格是 `80 c9`，與原本第 8 格相同）。
#[test]
fn remove_native_compact_inventory_keeps_tail_item_byte() {
    let mut u = inv(
        &[0x00, 0x84, 0xc0],
        &[true, true, false],
        &[0x00, 0x84, 0xc0, 0xff, 0xff, 0xff, 0x00, 0xc9],
        &[0x40, 0x40, 0x00, 0x80, 0x80, 0x80, 0x80, 0x80],
    );
    remove_native_compact_inventory(Some(&mut u), 0).unwrap();
    assert!(
        u.inventory_slots == [0x84, 0xc0, 0xff, 0xff, 0xff, 0x00, 0xc9, 0xc9]
            && u.native_inventory_flags == [0x40, 0x00, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        "slots={:x?} flags={:x?}",
        u.inventory_slots,
        u.native_inventory_flags
    );
    assert!(u.inventory.len() == 2 && u.inventory[0] == 0x84, "compact={:x?}", u.inventory);
    // 殘值不算佔用：再放一件會落在第 3 格，不會被尾格的 0xc9 擋住。
    assert!(
        u.add_inventory_item(0x33, false) && u.inventory_slots[2] == 0x33 && u.native_inventory_flags[2] == 0,
        "add after remove: slots={:x?} flags={:x?}",
        u.inventory_slots,
        u.native_inventory_flags
    );
}
