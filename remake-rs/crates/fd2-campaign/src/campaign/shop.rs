//! Go 來源：`remake/internal/campaign/shop.go`（183 行）。

use super::*;

/// Go：`shop.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ShopItemRow {
    /// Go 標籤：`json:"id"`
    pub id: i64,
    /// Go 標籤：`json:"type"`
    pub r#type: i64,
}

/// Go：`shop.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ShopEquipRow {
    /// Go 標籤：`json:"cls"`
    pub class_id: i64,
    /// Go 標籤：`json:"types"`
    pub types: Vec<i64>,
}

/// LoadShopEligibility loads the editable EXE-derived item type and class whitelist tables.
///
/// Go：`shop.go:21`，`func LoadShopEligibility(itemPath, equipPath string) (map[int]int, map[int][]int, error)`
pub fn load_shop_eligibility(
    item_path: &str,
    equip_path: &str,
) -> Result<(BTreeMap<i64, i64>, BTreeMap<i64, Vec<i64>>)> {
    todo!("campaign/shop.go:LoadShopEligibility")
}

/// LoadItemPrices loads the editable list-price column used by the original
/// shop sell table. SellGood applies the fixed 3/4 resale rule to this value.
///
/// Go：`shop.go:56`，`func LoadItemPrices(path string) (map[int]int, error)`
pub fn load_item_prices(path: &str) -> Result<BTreeMap<i64, i64>> {
    todo!("campaign/shop.go:LoadItemPrices")
}

/// CanEquip mirrors original 0x1c1c3: an equip item is allowed exactly when
/// its item.type appears in this class record's six type slots (0xff is empty).
///
/// Go：`shop.go:80`，`func CanEquip(classID, itemType int, classEquip map[int][]int) bool`
pub fn can_equip(class_id: i64, item_type: i64, class_equip: &BTreeMap<i64, Vec<i64>>) -> bool {
    todo!("campaign/shop.go:CanEquip")
}

/// BuyGood is the atomic part of original shop purchase: the selected receiver
/// gets the item in its first free inventory slot, then (and only then) gold is
/// deducted. Confirmation, eligible-recipient selection and equip prompting are
/// UI concerns layered above this operation.
///
/// Go：`shop.go:93`，`func BuyGood(gold int, receiver *battle.Unit, good Good) (int, error)`
pub fn buy_good(gold: i64, receiver: Option<&mut Unit>, good: &Good) -> Result<i64> {
    todo!("campaign/shop.go:BuyGood")
}

/// ReserveGood performs the original shop's inventory insertion before the
/// optional equipment prompt. Gold is intentionally not deducted until the
/// caller resolves that prompt via FinalizeGood.
///
/// Go：`shop.go:103`，`func ReserveGood(gold int, receiver *battle.Unit, good Good) (int, error)`
pub fn reserve_good(gold: i64, receiver: Option<&mut Unit>, good: &Good) -> Result<i64> {
    todo!("campaign/shop.go:ReserveGood")
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
    todo!("campaign/shop.go:SellGood")
}

/// SellSlot is the slot-addressed form used by the UI, so duplicate item IDs
/// cannot cause the wrong inventory entry to be sold.
///
/// Go：`shop.go:146`，`func SellSlot(gold int, receiver *battle.Unit, slot, listPrice int) (int, error)`
pub fn sell_slot(gold: i64, receiver: Option<&mut Unit>, slot: i64, list_price: i64) -> Result<i64> {
    todo!("campaign/shop.go:SellSlot")
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
    todo!("campaign/shop.go:SellNativeSlot")
}
