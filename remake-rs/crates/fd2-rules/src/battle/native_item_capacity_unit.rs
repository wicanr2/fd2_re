//! Go 來源：`remake/internal/battle/native_item_capacity_unit.go`（56 行）。

use super::*;

/// ApplyNativeItemCapacityToUnit projects the closed type17/18/19 0x21082
/// transaction onto editable Unit fields. Max HP/MP use native signed-word
/// wrapping while leaving current HP/MP unchanged. MV uses the low byte of the
/// +0x3b word add and preserves EXP exactly, matching the caller save/restore.
///
/// Go：`native_item_capacity_unit.go:9`，`func ApplyNativeItemCapacityToUnit( target, source *Unit, route NativeItemCapacityStepRoute, sourceRawSlot int, ) (int, error)`
/// 所有權：`units` 是 Rust 新增的參數。Go 允許這幾個 `*Unit` 指向同一個單位（例如對自己使用道具），Rust 改傳同一陣列的索引。
/// 所有權：`target`、`source` 是 `units` 的索引（Go 傳 `*Unit`）。
pub fn apply_native_item_capacity_to_unit(
    units: &mut [Unit],
    target: Option<UnitIdx>,
    source: Option<UnitIdx>,
    route: NativeItemCapacityStepRoute,
    source_raw_slot: i64,
) -> Result<i64> {
    todo!("battle/native_item_capacity_unit.go:ApplyNativeItemCapacityToUnit")
}
