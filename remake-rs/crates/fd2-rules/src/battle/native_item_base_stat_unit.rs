//! Go 來源：`remake/internal/battle/native_item_base_stat_unit.go`（80 行）。

use super::*;

/// ApplyNativeItemBaseStatDeltaToUnit is the editable Unit projection of the
/// closed 0x21082 transaction used by item types 8/9/10. It preserves native
/// 16-bit wrapping and 0x1b8e7 compact source removal. AP/DP require an
/// initialized equipment base; DX is already the independently established raw
/// +0x3e field. The caller remains responsible for 0x1145a recomputation.
///
/// Go：`native_item_base_stat_unit.go:10`，`func ApplyNativeItemBaseStatDeltaToUnit( target, source *Unit, route NativeItemWordDeltaRoute, delta uint16, sourceRawSlot int, ) (int, error)`
/// 所有權：`units` 是 Rust 新增的參數。Go 允許這幾個 `*Unit` 指向同一個單位（例如對自己使用道具），Rust 改傳同一陣列的索引。
/// 所有權：`target`、`source` 是 `units` 的索引（Go 傳 `*Unit`）。
pub fn apply_native_item_base_stat_delta_to_unit(
    units: &mut [Unit],
    target: Option<UnitIdx>,
    source: Option<UnitIdx>,
    route: NativeItemWordDeltaRoute,
    delta: u16,
    source_raw_slot: i64,
) -> Result<i64> {
    todo!("battle/native_item_base_stat_unit.go:ApplyNativeItemBaseStatDeltaToUnit")
}

/// Go：`native_item_base_stat_unit.go:62`，`func nativeItemSourceCompactIndex(source *Unit, sourceRawSlot int) (int, error)`
pub(crate) fn native_item_source_compact_index(source: Option<&Unit>, source_raw_slot: i64) -> Result<i64> {
    todo!("battle/native_item_base_stat_unit.go:nativeItemSourceCompactIndex")
}
