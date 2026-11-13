//! Go 來源：`remake/internal/battle/native_item_dispatch.go`（62 行）。

use super::*;

/// NativeItemEffectRoute is the raw call-topology result of 0x20c6f.  The
/// fields intentionally retain addresses and integer arguments instead of
/// naming an item type as potion, status, damage, or equipment behavior.
///
/// Go：`native_item_dispatch.go:6`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemEffectRoute {
    pub item_type: u8,
    /// item row +0x0e, passed to callees that consume it
    pub word: u16,
    pub primary: u32,
    pub secondary: u32,
    pub arg2: i64,
    pub arg5: i64,
    pub arg7: i64,
}

/// NativeItemEffectRouteForType reproduces the verified dispatch table in
/// 0x20c6f without executing any callee or mutating runtime state.  A false
/// result means the type is not covered by the recovered table.
///
/// Go：`native_item_dispatch.go:19`，`func NativeItemEffectRouteForType(itemType byte, word uint16) (NativeItemEffectRoute, bool)`
pub fn native_item_effect_route_for_type(item_type: u8, word: u16) -> (NativeItemEffectRoute, bool) {
    todo!("battle/native_item_dispatch.go:NativeItemEffectRouteForType")
}
