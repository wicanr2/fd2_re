//! 移植自 `remake/internal/campaign/shop_test.go`。
//!
//! Go 的 `BuyGood`／`SellGood` 失敗時另回傳原本的 gold；Rust 回 `Err`，測試以 [`go_gold`]
//! 還原 Go 的兩個回傳值再比對同樣的期望。

use super::*;
use crate::campaign::church::church_test::unit;
use crate::test_support::remake_asset;

/// 把 `Result<i64>` 還原成 Go 的 `(gold, err)`：失敗時 gold 是呼叫時傳入的值。
fn go_gold(input: i64, r: Result<i64>) -> (i64, Option<CampaignError>) {
    match r {
        Ok(g) => (g, None),
        Err(e) => (input, Some(e)),
    }
}

fn three_item_stats() -> BTreeMap<i64, ItemStats> {
    BTreeMap::from([
        (0, ItemStats { r#type: 1, ap: 10, hit: 95, ..Default::default() }),
        (1, ItemStats { r#type: 1, ap: 20, hit: 95, ..Default::default() }),
        (132, ItemStats { r#type: 22, dp: 8, ..Default::default() }),
    ])
}

/// Go：`TestBuyGoodUsesSelectedInventoryAndIsAtomicOnFailure`
#[test]
fn buy_good_uses_selected_inventory_and_is_atomic_on_failure() {
    let good = Good { id: 0xc0, name: "藥草".into(), price: 10 };
    let mut receiver = unit! { inventory: vec![1, 2], };
    let (gold, err) = go_gold(50, buy_good(50, Some(&mut receiver), &good));
    assert!(
        err.is_none() && gold == 40 && receiver.inventory == [1, 2, 0xc0],
        "purchase gold={gold} err={err:?} inventory={:?}",
        receiver.inventory
    );
    let mut full = unit! { inventory: vec![0; 8], };
    let (got, err) = go_gold(50, buy_good(50, Some(&mut full), &good));
    assert!(
        err.is_some() && got == 50 && full.inventory.len() == 8,
        "full inventory changed gold={got} err={err:?} inventory={:?}",
        full.inventory
    );
    let (got, err) = go_gold(9, buy_good(9, Some(&mut receiver), &good));
    assert!(
        err.is_some() && got == 9 && receiver.inventory.len() == 3,
        "insufficient gold changed state gold={got} err={err:?} inventory={:?}",
        receiver.inventory
    );
}

/// Go：`TestReserveGoodDefersGoldUntilEquipDecision`
#[test]
fn reserve_good_defers_gold_until_equip_decision() {
    let mut receiver = Unit::default();
    let good = Good { id: 0x80, price: 25, name: "劍".into() };
    let slot = reserve_good(30, Some(&mut receiver), &good);
    assert!(
        slot.as_ref().is_ok_and(|&s| s == 0)
            && receiver.inventory.len() == 1
            && receiver.equipped.len() == 1
            && !receiver.equipped[0],
        "reserve slot={slot:?} inventory={:?} equipped={:?}",
        receiver.inventory,
        receiver.equipped
    );
    assert_eq!(finalize_good(30, &good), 5, "finalize gold");
}

/// Go：`TestSellGoodPaysThreeQuartersAndRemovesSlot`
#[test]
fn sell_good_pays_three_quarters_and_removes_slot() {
    let mut u = unit! { inventory: vec![7, 8], equipped: vec![true, false], };
    let (gold, err) = go_gold(10, sell_good(10, Some(&mut u), 7, 101));
    assert!(
        err.is_none() && gold == 85 && u.inventory == [8] && u.equipped == [false],
        "sell gold={gold} err={err:?} inventory={:?} equipped={:?}",
        u.inventory,
        u.equipped
    );
}

/// Go：`TestSellSlotTargetsDuplicateInventoryEntry`
#[test]
fn sell_slot_targets_duplicate_inventory_entry() {
    let mut u = unit! { inventory: vec![7, 7], equipped: vec![true, false], };
    let (gold, err) = go_gold(0, sell_slot(0, Some(&mut u), 1, 100));
    assert!(
        err.is_none() && u.inventory.len() == 1 && u.equipped == [true] && gold == 75,
        "slot sell gold={gold} err={err:?} inventory={:?} equipped={:?}",
        u.inventory,
        u.equipped
    );
}

/// Go：`TestSellNativeSlotCompactsRawCellsAndCreditsAfterValidation`
#[test]
fn sell_native_slot_compacts_raw_cells_and_credits_after_validation() {
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
        equipped: vec![true, false, true],
        inventory_slots: vec![0, 1, 132, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80],
    };
    let gold = sell_native_slot(100, Some(&mut u), 1, 200, &three_item_stats()).unwrap();
    assert!(
        gold == 250
            && u.inventory == [0, 132]
            && u.inventory_slots == [0, 132, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff]
            && u.native_inventory_flags == [0x40, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        "native sell = gold {gold} inventory {:?} slots {:?} flags {:?}",
        u.inventory,
        u.inventory_slots,
        u.native_inventory_flags
    );
}

/// Go：`TestEquipNativeCompactSlotPreservesRawHoleMapping`
#[test]
fn equip_native_compact_slot_preserves_raw_hole_mapping() {
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
        equipped: vec![true, false, true],
        inventory_slots: vec![0, 0x77, 1, 132, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0x80, 0, 0x40, 0x80, 0x80, 0x80, 0x80],
    };
    equip_native_compact_slot(Some(&mut u), 1, &three_item_stats()).unwrap();
    assert!(
        u.equipped == [false, true, true]
            && u.native_inventory_flags == [0, 0x80, 0x40, 0x40, 0x80, 0x80, 0x80, 0x80]
            && u.ap == 26
            && u.dp == 12,
        "native equip = equipped {:?} flags {:?} AP/DP {}/{}",
        u.equipped,
        u.native_inventory_flags,
        u.ap,
        u.dp
    );
}

/// Go：`TestEquipNativeCompactSlotRejectsDivergentProjectionAtomically`
#[test]
fn equip_native_compact_slot_rejects_divergent_projection_atomically() {
    let mut u = unit! {
        inventory: vec![0, 1],
        equipped: vec![true, false],
        inventory_slots: vec![0, 2, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
    };
    let before = u.clone();
    let stats = BTreeMap::from([(1, ItemStats { r#type: 1, ..Default::default() })]);
    assert!(
        equip_native_compact_slot(Some(&mut u), 1, &stats).is_err(),
        "divergent raw/compact projection was accepted"
    );
    assert_eq!(u, before, "failed native equip mutated unit");
}

/// Go：`TestLoadItemPricesFromRuntimeBundle`
#[test]
fn load_item_prices_from_runtime_bundle() {
    let Some(path) = remake_asset("data/item.json") else { return };
    let prices = load_item_prices(&path).unwrap();
    assert!(
        prices.get(&0) == Some(&50) && prices.get(&0x80) == Some(&50),
        "unexpected item prices: id0={:?} id80={:?}",
        prices.get(&0),
        prices.get(&0x80)
    );
}

/// Go：`TestLoadShopEligibilityUsesOriginalTables`
#[test]
fn load_shop_eligibility_uses_original_tables() {
    let (Some(items), Some(equip)) = (remake_asset("data/item.json"), remake_asset("data/class_equip_types.json"))
    else {
        return;
    };
    let (types, equip) = load_shop_eligibility(&items, &equip).unwrap();
    let t80 = types.get(&0x80).copied().unwrap_or(0);
    assert!(
        t80 == 21 && can_equip(1, t80, &equip) && !can_equip(25, t80, &equip),
        "eligibility tables type={t80} equip={:?}",
        equip.get(&1)
    );
}

/// Go：`TestCanEquipUsesOriginalClassTypeWhitelist`
#[test]
fn can_equip_uses_original_class_type_whitelist() {
    let table = BTreeMap::from([(1, vec![1, 21, 22, 255, 255, 255]), (25, vec![8, 27, 255, 255, 255, 255])]);
    assert!(
        can_equip(1, 21, &table) && !can_equip(1, 8, &table) && can_equip(25, 27, &table) && !can_equip(0, 1, &table),
        "class/type whitelist mismatch: {table:?}"
    );
}
