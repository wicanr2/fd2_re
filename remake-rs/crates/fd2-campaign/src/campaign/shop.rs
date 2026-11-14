//! Go 來源：`remake/internal/campaign/shop.go`（183 行）。

use super::*;

/// Go：`shop.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default)]
pub(crate) struct ShopItemRow {
    /// Go 標籤：`json:"id"`
    pub id: i64,
    /// Go 標籤：`json:"type"`
    pub r#type: i64,
}

/// Go：`shop.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default)]
pub(crate) struct ShopEquipRow {
    /// Go 標籤：`json:"cls"`
    #[serde(rename = "cls")]
    pub class_id: i64,
    /// Go 標籤：`json:"types"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub types: Vec<i64>,
}

/// LoadShopEligibility loads the editable EXE-derived item type and class whitelist tables.
///
/// Go：`shop.go:21`，`func LoadShopEligibility(itemPath, equipPath string) (map[int]int, map[int][]int, error)`
pub fn load_shop_eligibility(
    item_path: &str,
    equip_path: &str,
) -> Result<(BTreeMap<i64, i64>, BTreeMap<i64, Vec<i64>>)> {
    let item_raw = battle::read_file(item_path)?;
    let equip_raw = battle::read_file(equip_path)?;
    let items: Vec<ShopItemRow> = battle::decode_json(&item_raw)?;
    let rows: Vec<ShopEquipRow> = battle::decode_json(&equip_raw)?;
    let (mut types, mut equip) = (BTreeMap::new(), BTreeMap::new());
    for item in items {
        if item.id < 0 || item.id > 0xff {
            return Err(campaign_err!("invalid item id {}", item.id));
        }
        types.insert(item.id, item.r#type);
    }
    for row in rows {
        if row.types.len() != 6 {
            return Err(campaign_err!("class {} has {} equipment slots", row.class_id, row.types.len()));
        }
        equip.insert(row.class_id, row.types);
    }
    Ok((types, equip))
}

/// LoadItemPrices loads the editable list-price column used by the original
/// shop sell table. SellGood applies the fixed 3/4 resale rule to this value.
///
/// Go：`shop.go:56`，`func LoadItemPrices(path string) (map[int]int, error)`
pub fn load_item_prices(path: &str) -> Result<BTreeMap<i64, i64>> {
    /// Go 的匿名列結構（`shop.go:62`）。
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Row {
        id: i64,
        price: i64,
    }
    let raw = battle::read_file(path)?;
    let rows: Vec<Row> = battle::decode_json(&raw)?;
    let mut prices = BTreeMap::new();
    for row in rows {
        if row.id < 0 || row.id > 0xff || row.price < 0 {
            return Err(campaign_err!("invalid item price row id={} price={}", row.id, row.price));
        }
        prices.insert(row.id, row.price);
    }
    Ok(prices)
}

/// CanEquip mirrors original 0x1c1c3: an equip item is allowed exactly when
/// its item.type appears in this class record's six type slots (0xff is empty).
///
/// Go：`shop.go:80`，`func CanEquip(classID, itemType int, classEquip map[int][]int) bool`
pub fn can_equip(class_id: i64, item_type: i64, class_equip: &BTreeMap<i64, Vec<i64>>) -> bool {
    class_equip.get(&class_id).is_some_and(|allowed| allowed.contains(&item_type))
}

/// BuyGood is the atomic part of original shop purchase: the selected receiver
/// gets the item in its first free inventory slot, then (and only then) gold is
/// deducted. Confirmation, eligible-recipient selection and equip prompting are
/// UI concerns layered above this operation.
///
/// Go：`shop.go:93`，`func BuyGood(gold int, receiver *battle.Unit, good Good) (int, error)`
pub fn buy_good(gold: i64, receiver: Option<&mut Unit>, good: &Good) -> Result<i64> {
    // Go 失敗時回傳原本的 gold；Rust 的 `Err` 由呼叫端保留原值。
    reserve_good(gold, receiver, good)?;
    Ok(gold.wrapping_sub(good.price))
}

/// ReserveGood performs the original shop's inventory insertion before the
/// optional equipment prompt. Gold is intentionally not deducted until the
/// caller resolves that prompt via FinalizeGood.
///
/// Go：`shop.go:103`，`func ReserveGood(gold int, receiver *battle.Unit, good Good) (int, error)`
pub fn reserve_good(gold: i64, receiver: Option<&mut Unit>, good: &Good) -> Result<i64> {
    // Go 失敗時回傳 -1。
    let Some(receiver) = receiver else {
        return Err(campaign_err!("shop receiver missing"));
    };
    if good.id < 0 || good.id > 0xff || good.price < 0 {
        return Err(campaign_err!("invalid shop good"));
    }
    if gold < good.price {
        return Err(campaign_err!("insufficient gold"));
    }
    if receiver.inventory.len() >= 8 {
        return Err(campaign_err!("inventory full"));
    }
    if !receiver.add_inventory_item(good.id, false) {
        return Err(campaign_err!("inventory full"));
    }
    Ok(receiver.inventory.len() as i64 - 1)
}

/// FinalizeGood deducts the already-confirmed good's price.
///
/// Go：`shop.go:123`，`func FinalizeGood(gold int, good Good) int`
pub fn finalize_good(gold: i64, good: &Good) -> i64 {
    gold.wrapping_sub(good.price)
}

/// SellGood mirrors the original 75%-of-list-price transaction. The slot is
/// removed only after validation; its equipped marker follows the item out.
///
/// Go：`shop.go:129`，`func SellGood(gold int, receiver *battle.Unit, itemID, listPrice int) (int, error)`
pub fn sell_good(gold: i64, receiver: Option<&mut Unit>, item_id: i64, list_price: i64) -> Result<i64> {
    let Some(receiver) = receiver else {
        return Err(campaign_err!("shop receiver missing"));
    };
    if !(0..=0xff).contains(&item_id) || list_price < 0 {
        return Err(campaign_err!("invalid shop item"));
    }
    match receiver.inventory.iter().position(|&id| id == item_id) {
        Some(slot) => sell_slot(gold, Some(receiver), slot as i64, list_price),
        None => Err(campaign_err!("item not found")),
    }
}

/// SellSlot is the slot-addressed form used by the UI, so duplicate item IDs
/// cannot cause the wrong inventory entry to be sold.
///
/// Go：`shop.go:146`，`func SellSlot(gold int, receiver *battle.Unit, slot, listPrice int) (int, error)`
pub fn sell_slot(gold: i64, receiver: Option<&mut Unit>, slot: i64, list_price: i64) -> Result<i64> {
    let Some(receiver) = receiver else {
        return Err(campaign_err!("shop receiver missing"));
    };
    if slot < 0 || slot >= receiver.inventory.len() as i64 || list_price < 0 {
        return Err(campaign_err!("invalid shop slot"));
    }
    receiver.remove_inventory_index(slot);
    Ok(gold.wrapping_add(list_price.wrapping_mul(3) / 4))
}

/// SellNativeSlot applies the proven 0x2d3ff→0x1b8e7→0x1b750 sell mutation
/// as one fail-closed high-level commit. The caller owns confirmation and
/// success presentation; listPrice is the original row+0x13 value.
///
/// Go：`shop.go:160`，`func SellNativeSlot( gold int, receiver *battle.Unit, slot, listPrice int, stats map[int]ItemStats, ) (int, error)`
pub fn sell_native_slot(
    gold: i64,
    receiver: Option<&mut Unit>,
    slot: i64,
    list_price: i64,
    stats: &BTreeMap<i64, ItemStats>,
) -> Result<i64> {
    let Some(receiver) = receiver else {
        return Err(campaign_err!("invalid native shop sell state"));
    };
    if slot < 0 || slot >= receiver.inventory.len() as i64 || list_price < 0 {
        return Err(campaign_err!("invalid native shop sell state"));
    }
    // Go `staged := *receiver` 淺複製後換掉四個切片；Rust clone 是深複製，結果相同。
    let mut staged = receiver.clone();
    battle::remove_native_compact_inventory(Some(&mut staged), slot)?;
    recompute_equipment(Some(&mut staged), stats);
    *receiver = staged;
    Ok(gold.wrapping_add(list_price.wrapping_mul(3) / 4))
}

#[cfg(test)]
pub(crate) mod shop_test;
