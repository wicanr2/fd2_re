//! Go 來源：`remake/internal/battle/native_inventory_transfer.go`（201 行）。

use super::*;

/// ValidateNativeInventoryProjection proves that the compact editable arrays
/// still represent exactly the occupied native cells in raw order. Ignored
/// cells may retain stale item bytes; their signed flag remains authoritative.
///
/// Go：`native_inventory_transfer.go:8`，`func ValidateNativeInventoryProjection(unit *Unit) error`
pub fn validate_native_inventory_projection(unit: Option<&Unit>) -> Result<()> {
    let Some(unit) = unit.filter(|u| {
        u.inventory_slots.len() as i64 == NATIVE_INVENTORY_CELLS
            && u.native_inventory_flags.len() as i64 == NATIVE_INVENTORY_CELLS
            && u.inventory.len() == u.equipped.len()
    }) else {
        return Err(battle_err!("native inventory projection is incomplete"));
    };
    let mut compact = 0;
    for (raw_slot, &flag) in unit.native_inventory_flags.iter().enumerate() {
        if flag & 0x80 != 0 {
            continue;
        }
        if compact >= unit.inventory.len()
            || unit.inventory_slots[raw_slot] != unit.inventory[compact]
            || (flag & 0x40 != 0) != unit.equipped[compact]
        {
            return Err(battle_err!("native inventory projection diverges at raw slot {raw_slot}"));
        }
        compact += 1;
    }
    if compact != unit.inventory.len() {
        return Err(battle_err!("native inventory projection length diverges"));
    }
    Ok(())
}

/// TransferNativeInventoryItem mirrors the mutation topology proven in the
/// 0x2f8ea -> 0x1b8e7 -> 0x1bb8c branch.  The source item is removed from its
/// compact inventory slot and inserted into the first empty destination cell;
/// 0x1bb8c writes an un-equipped destination cell.  The caller must already
/// have applied the native signed-flag eligibility filter (0x2f8ea's list
/// builder); this function does not infer equipped state from an item ID.
///
/// It is deliberately a raw operation rather than a named church service.
///
/// Go：`native_inventory_transfer.go:43`，`func TransferNativeInventoryItem(source *Unit, sourceIndex int, destination *Unit) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 允許這幾個 `*Unit` 指向同一個單位（例如對自己使用道具），Rust 改傳同一陣列的索引。
/// 所有權：`source`、`destination` 是 `units` 的索引（Go 傳 `*Unit`）。
pub fn transfer_native_inventory_item(
    units: &mut [Unit],
    source: Option<UnitIdx>,
    source_index: i64,
    destination: Option<UnitIdx>,
) -> Result<()> {
    let (Some(source), Some(destination)) = (source, destination) else {
        return Err(battle_err!("native inventory transfer: missing unit"));
    };
    validate_native_inventory_projection(Some(&units[source]))?;
    if source != destination {
        validate_native_inventory_projection(Some(&units[destination]))?;
    }
    if source_index < 0 || source_index >= units[source].inventory.len() as i64 {
        return Err(battle_err!("native inventory transfer: source slot {source_index} out of bounds"));
    }
    let item_id = units[source].inventory[source_index as usize];
    if !(0..=0xff).contains(&item_id) {
        return Err(battle_err!("native inventory transfer: invalid item id {item_id}"));
    }
    // Preflight both fixed-slot layouts before changing either unit.  A full
    // destination is the native failure path and must not consume the source.
    let free = units[destination].native_inventory_flags.iter().any(|&flag| flag & 0x80 != 0);
    if !free || units[destination].inventory.len() >= 8 {
        return Err(battle_err!("native inventory transfer: destination inventory full"));
    }
    // Snapshot the compact and source-slot views so an unexpected insertion
    // failure remains atomic even if a future Unit representation changes.
    let snapshot = |u: &Unit| {
        (u.inventory.clone(), u.equipped.clone(), u.inventory_slots.clone(), u.native_inventory_flags.clone())
    };
    let source_before = snapshot(&units[source]);
    let destination_before = snapshot(&units[destination]);
    if !remove_native_compact_inventory_inner(Some(&mut units[source]), source_index)
        || !units[destination].add_inventory_item(item_id, false)
    {
        let s = &mut units[source];
        (s.inventory, s.equipped, s.inventory_slots, s.native_inventory_flags) = source_before;
        let d = &mut units[destination];
        (d.inventory, d.equipped, d.inventory_slots, d.native_inventory_flags) = destination_before;
        return Err(battle_err!("native inventory transfer: mutation failed"));
    }
    Ok(())
}

/// ReplaceNativeFullInventoryReward 對應已證實的死亡獎勵路徑
/// 0x1B722 -> 0x1B8E7 -> 0x1BB8C：丟棄選定舊物品、後續格左移，再把新獎勵
/// 以未裝備狀態附加到尾端。原版 caller 沒有隊友或目的角色，因此此操作刻意與
/// TransferNativeInventoryItem 分開。
///
/// Go：`native_inventory_transfer.go:96`，`func ReplaceNativeFullInventoryReward(unit *Unit, selectedIndex, rewardItem int) error`
pub fn replace_native_full_inventory_reward(
    unit: Option<&mut Unit>,
    selected_index: i64,
    reward_item: i64,
) -> Result<()> {
    let Some(unit) = unit.filter(|_| (0..=0xff).contains(&reward_item)) else {
        return Err(battle_err!("native death reward replacement: invalid unit/item"));
    };
    validate_native_inventory_projection(Some(unit)).map_err(|e| e.wrap("native death reward replacement: "))?;
    if unit.inventory.len() as i64 != NATIVE_INVENTORY_CELLS || !(0..NATIVE_INVENTORY_CELLS).contains(&selected_index) {
        return Err(battle_err!(
            "native death reward replacement: full selection {}/{} is invalid",
            selected_index,
            unit.inventory.len()
        ));
    }
    for (slot, &flag) in unit.native_inventory_flags.iter().enumerate() {
        if flag & 0x80 != 0 {
            return Err(battle_err!("native death reward replacement: raw slot {slot} is not occupied"));
        }
    }
    let inventory = unit.inventory.clone();
    let equipped = unit.equipped.clone();
    let slots = unit.inventory_slots.clone();
    let flags = unit.native_inventory_flags.clone();
    if !remove_native_compact_inventory_inner(Some(unit), selected_index)
        || !unit.add_inventory_item(reward_item, false)
    {
        (unit.inventory, unit.equipped) = (inventory, equipped);
        (unit.inventory_slots, unit.native_inventory_flags) = (slots, flags);
        return Err(battle_err!("native death reward replacement: mutation failed"));
    }
    Ok(())
}

/// Go：`native_inventory_transfer.go:126`，`func firstInventoryHole(slots []int) int`
pub(crate) fn first_inventory_hole(slots: &[i64]) -> i64 {
    slots.iter().position(|&item| item == 0xff).map_or(-1, |i| i as i64)
}

impl Unit {
    /// inventoryCellOccupied 判斷第 i 格有沒有東西。原版消費端（0x1B722 等）看旗標
    /// bit7，空格的 item byte 可以是任何殘值（0x1B8E7 左移後第 8 格就留著舊值），所以
    /// 有 raw 旗標時旗標 bit7 為 1 就是空；沒有旗標、或舊夾具只用 0xff 標空的，仍看 0xff。
    ///
    /// Go：`native_inventory_transfer.go:138`，`func (u *Unit) inventoryCellOccupied(i int) bool`
    pub(crate) fn inventory_cell_occupied(&self, i: i64) -> bool {
        if i < 0 || i >= self.inventory_slots.len() as i64 {
            return false;
        }
        let i = i as usize;
        if self.native_inventory_flags.len() as i64 == NATIVE_INVENTORY_CELLS
            && self.inventory_slots.len() as i64 == NATIVE_INVENTORY_CELLS
            && self.native_inventory_flags[i] & 0x80 != 0
        {
            return false;
        }
        self.inventory_slots[i] != 0xff
    }
}

/// removeNativeCompactInventory combines the compact editable view with the
/// fixed-cell shift performed by native 0x1b8e7.  The latter does not leave a
/// hole: `0x1B924 memmove` 把後面的格子左移一格，然後只把第 8 格的旗標 byte
/// 寫成 0x80（`0x1B92C mov byte [ebx+0x18], 0x80`）；第 8 格的 item byte 不動，
/// 留著移走前第 8 格的值。有 raw 旗標時照這個版面，item byte 只是殘值。
///
/// Go：`native_inventory_transfer.go:154`，`func removeNativeCompactInventory(u *Unit, compactIndex int) bool`
pub(crate) fn remove_native_compact_inventory_inner(u: Option<&mut Unit>, compact_index: i64) -> bool {
    let Some(u) = u.filter(|u| compact_index >= 0 && compact_index < u.inventory.len() as i64) else {
        return false;
    };
    u.normalize_inventory_slots();
    let (mut seen, mut slot) = (0, -1i64);
    for i in 0..u.inventory_slots.len() as i64 {
        if u.inventory_cell_occupied(i) {
            if seen == compact_index {
                slot = i;
                break;
            }
            seen += 1;
        }
    }
    if slot < 0 {
        return false;
    }
    let slot = slot as usize;
    u.inventory_slots.copy_within(slot + 1.., slot);
    if u.native_inventory_flags.len() as i64 == NATIVE_INVENTORY_CELLS {
        u.native_inventory_flags.copy_within(slot + 1.., slot);
        let last = u.native_inventory_flags.len() - 1;
        u.native_inventory_flags[last] = 0x80;
    } else {
        let last = u.inventory_slots.len() - 1;
        u.inventory_slots[last] = 0xff;
    }
    u.inventory.remove(compact_index as usize);
    if (compact_index as usize) < u.equipped.len() {
        u.equipped.remove(compact_index as usize);
    }
    true
}

/// RemoveNativeCompactInventory exposes the 0x1b8e7 compact-to-raw adapter to
/// facility transaction owners. Exact eight-cell provenance is mandatory;
/// unlike Unit.RemoveInventoryIndex this shifts later raw cells left.
///
/// Go：`native_inventory_transfer.go:189`，`func RemoveNativeCompactInventory(u *Unit, compactIndex int) error`
pub fn remove_native_compact_inventory(u: Option<&mut Unit>, compact_index: i64) -> Result<()> {
    let Some(u) = u.filter(|u| {
        u.inventory_slots.len() as i64 == NATIVE_INVENTORY_CELLS
            && u.native_inventory_flags.len() as i64 == NATIVE_INVENTORY_CELLS
    }) else {
        return Err(battle_err!("native inventory remove: missing eight-cell provenance"));
    };
    if !remove_native_compact_inventory_inner(Some(u), compact_index) {
        return Err(battle_err!("native inventory remove: compact slot {compact_index} is invalid"));
    }
    Ok(())
}

#[cfg(test)]
mod native_inventory_transfer_test;
