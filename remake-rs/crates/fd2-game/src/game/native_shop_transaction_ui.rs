//! Go 來源：`remake/cmd/fd2/native_shop_transaction_ui.go`（262 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_shop_transaction_ui.go:10`，`func cloneNativeShopUnit(source battle.Unit) battle.Unit`
pub(crate) fn clone_native_shop_unit(source: &battle::Unit) -> battle::Unit {
    // Go 逐一複製 `Inventory`、`Equipped`、`InventorySlots`、`NativeInventoryFlags`、`Spells` 的底層陣列；
    // Rust 的 `Unit::clone` 本來就是深複製。
    source.clone()
}
