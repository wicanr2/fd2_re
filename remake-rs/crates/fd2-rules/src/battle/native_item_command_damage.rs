//! Go 來源：`remake/internal/battle/native_item_command_damage.go`（80 行）。

use super::*;

/// NativeItemCommandDamageRoute closes item types 20/21/24. The item row word
/// is reused as a 0x4e516 command ID by 0x1c75e; 20/24 use the ten-frame
/// 0x1cd17 presentation while 21 reaches 0x1cac7 through 0x2111a. None removes
/// the source slot.
///
/// Go：`native_item_command_damage.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemCommandDamageRoute {
    pub item_type: u8,
    pub command_id: i64,
    pub presentation: u32,
    pub consumes_source: bool,
}

/// Go：`native_item_command_damage.go:18`，`func NativeItemCommandDamageRouteForType(itemType byte, rowWord uint16) (NativeItemCommandDamageRoute, bool)`
pub fn native_item_command_damage_route_for_type(item_type: u8, row_word: u16) -> (NativeItemCommandDamageRoute, bool) {
    todo!("battle/native_item_command_damage.go:NativeItemCommandDamageRouteForType")
}

/// ApplyNativeItemCommandDamage preserves the shared post-presentation target
/// loop. It intentionally performs no 0x1ca89 MP debit and no inventory
/// removal: neither call exists in these dispatcher branches.
///
/// Go：`native_item_command_damage.go:40`，`func ApplyNativeItemCommandDamage( targets []*Unit, route NativeItemCommandDamageRoute, commandBook []NativeCommandRecord, resistances map[int]int, rngState uint16, ) ([]NativeCommandDamage, uint16, error)`
pub fn apply_native_item_command_damage(
    targets: &mut [Unit],
    route: NativeItemCommandDamageRoute,
    command_book: &[NativeCommandRecord],
    resistances: &BTreeMap<i64, i64>,
    rng_state: u16,
) -> Result<(Vec<NativeCommandDamage>, u16)> {
    todo!("battle/native_item_command_damage.go:ApplyNativeItemCommandDamage")
}
