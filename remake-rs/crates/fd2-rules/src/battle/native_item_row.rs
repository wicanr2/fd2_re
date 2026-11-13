//! Go 來源：`remake/internal/battle/native_item_row.go`（57 行）。

use super::*;

/// NativeItemEffectTableBase is the linear address used by 0x4e56c after
/// relocation into the EXE data image.  The helper below returns an offset
/// relative to that table; it does not assign names to the row fields.
pub const NATIVE_ITEM_EFFECT_TABLE_BASE: i64 = 0x602ad;

pub const NATIVE_ITEM_EFFECT_ROW_SIZE: i64 = 0x17;

/// NativeItemBaseStat identifies the three persistent base words consumed by
/// 0x1145a. Their meanings are closed by the 215-row AP/HIT/DP/EV cross-check,
/// not inferred from presentation codes.
///
/// Go：`native_item_row.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeItemBaseStat(pub &'static str);

pub const NATIVE_ITEM_BASE_AP: NativeItemBaseStat = NativeItemBaseStat("ap");

pub const NATIVE_ITEM_BASE_DP: NativeItemBaseStat = NativeItemBaseStat("dp");

pub const NATIVE_ITEM_BASE_DX: NativeItemBaseStat = NativeItemBaseStat("dx");

/// NativeItemWordDeltaRoute is the raw dispatch contract for item types 8, 9
/// and 0xa in 0x20c6f.  0x21082 receives the item row's +0xe word as delta,
/// permanently adds it to the target record's base AP/DP/DX word, and receives
/// a separate presentation selector whose meaning remains opaque.
///
/// Go：`native_item_row.go:28`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemWordDeltaRoute {
    pub item_type: i64,
    pub base_stat: NativeItemBaseStat,
    pub field_offset: i64,
    pub presentation_code: i64,
}

/// Go：`native_item_row.go:35`，`func NativeItemWordDeltaRouteForType(itemType int) (NativeItemWordDeltaRoute, bool)`
pub fn native_item_word_delta_route_for_type(item_type: i64) -> (NativeItemWordDeltaRoute, bool) {
    todo!("battle/native_item_row.go:NativeItemWordDeltaRouteForType")
}

/// NativeItemEffectRowOffset reproduces 0x4e56c(item): item rows are selected
/// with a 23-byte stride from the table base.  The native routine has no bounds
/// check, so this adapter validates the byte-sized selector and returns only
/// the proven table-relative offset.
///
/// Go：`native_item_row.go:52`，`func NativeItemEffectRowOffset(item int) (int, error)`
pub fn native_item_effect_row_offset(item: i64) -> Result<i64> {
    todo!("battle/native_item_row.go:NativeItemEffectRowOffset")
}
