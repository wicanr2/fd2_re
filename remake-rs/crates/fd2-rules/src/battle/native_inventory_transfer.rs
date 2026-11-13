//! Go 來源：`remake/internal/battle/native_inventory_transfer.go`（201 行）。

use super::*;

/// ValidateNativeInventoryProjection proves that the compact editable arrays
/// still represent exactly the occupied native cells in raw order. Ignored
/// cells may retain stale item bytes; their signed flag remains authoritative.
///
/// Go：`native_inventory_transfer.go:8`，`func ValidateNativeInventoryProjection(unit *Unit) error`
pub fn validate_native_inventory_projection(unit: Option<&Unit>) -> Result<()> {
    todo!("battle/native_inventory_transfer.go:ValidateNativeInventoryProjection")
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
    todo!("battle/native_inventory_transfer.go:TransferNativeInventoryItem")
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
    todo!("battle/native_inventory_transfer.go:ReplaceNativeFullInventoryReward")
}

/// Go：`native_inventory_transfer.go:126`，`func firstInventoryHole(slots []int) int`
pub(crate) fn first_inventory_hole(slots: &[i64]) -> i64 {
    todo!("battle/native_inventory_transfer.go:firstInventoryHole")
}

impl Unit {
    /// inventoryCellOccupied 判斷第 i 格有沒有東西。原版消費端（0x1B722 等）看旗標
    /// bit7，空格的 item byte 可以是任何殘值（0x1B8E7 左移後第 8 格就留著舊值），所以
    /// 有 raw 旗標時旗標 bit7 為 1 就是空；沒有旗標、或舊夾具只用 0xff 標空的，仍看 0xff。
    ///
    /// Go：`native_inventory_transfer.go:138`，`func (u *Unit) inventoryCellOccupied(i int) bool`
    pub(crate) fn inventory_cell_occupied(&self, i: i64) -> bool {
        todo!("battle/native_inventory_transfer.go:Unit.inventoryCellOccupied")
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
    todo!("battle/native_inventory_transfer.go:removeNativeCompactInventory")
}

/// RemoveNativeCompactInventory exposes the 0x1b8e7 compact-to-raw adapter to
/// facility transaction owners. Exact eight-cell provenance is mandatory;
/// unlike Unit.RemoveInventoryIndex this shifts later raw cells left.
///
/// Go：`native_inventory_transfer.go:189`，`func RemoveNativeCompactInventory(u *Unit, compactIndex int) error`
pub fn remove_native_compact_inventory(u: Option<&mut Unit>, compact_index: i64) -> Result<()> {
    todo!("battle/native_inventory_transfer.go:RemoveNativeCompactInventory")
}
