//! Go 來源：`remake/internal/battle/native_item_relocation.go`（113 行）。

use super::*;

/// NativeItemRelocationRoute closes the post-confirm type-23 item branch.
/// 0x2218a reuses command 23's MP cost, retains the item, and relocates only
/// the first target through the shared 0x22253 presentation.
///
/// Go：`native_item_relocation.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemRelocationRoute {
    pub item_type: u8,
    pub row_word: u16,
    pub command_id: i64,
    pub required_identity: u8,
    pub required_max_mp: u16,
    pub presentation: u32,
    pub consumes_source: bool,
}

/// Go：`native_item_relocation.go:21`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemRelocationResult {
    pub target_unit: i64,
    pub from_x: u8,
    pub from_y: u8,
    pub destination_x: u8,
    pub destination_y: u8,
    pub remaining_mp: u16,
    pub accumulator_delta: i64,
}

/// Go：`native_item_relocation.go:30`，`func NativeItemRelocationRouteForType(itemType byte, rowWord uint16) (NativeItemRelocationRoute, bool)`
pub fn native_item_relocation_route_for_type(item_type: u8, row_word: u16) -> (NativeItemRelocationRoute, bool) {
    todo!("battle/native_item_relocation.go:NativeItemRelocationRouteForType")
}

/// ApplyNativeItemRelocation preserves the mutation portion after 0x1bbdc has
/// confirmed the type-23 destination. Selector mode-6 terrain/occupancy checks
/// and the 27-present indexed renderer remain caller-owned.
///
/// Go：`native_item_relocation.go:44`，`func ApplyNativeItemRelocation( records []byte, actorUnit int, targets []byte, destinationX, destinationY byte, route NativeItemRelocationRoute, commandBook []NativeCommandRecord, ) (NativeItemRelocationResult, error)`
pub fn apply_native_item_relocation(
    records: &mut [u8],
    actor_unit: i64,
    targets: &[u8],
    destination_x: u8,
    destination_y: u8,
    route: NativeItemRelocationRoute,
    command_book: &[NativeCommandRecord],
) -> Result<NativeItemRelocationResult> {
    todo!("battle/native_item_relocation.go:ApplyNativeItemRelocation")
}

/// ApplyNativeCommand23Relocation is the spell dispatch to sub_2218A.
/// Item-only identity/max-MP guards belong to the item caller, not this handler.
/// The spell caller has already checked current MP; preserve that atomic gate.
///
/// Go：`native_item_relocation.go:70`，`func ApplyNativeCommand23Relocation(records []byte, actorUnit int, targets []byte, destinationX, destinationY byte, commandBook []NativeCommandRecord) (NativeItemRelocationResult, error)`
pub fn apply_native_command23_relocation(
    records: &mut [u8],
    actor_unit: i64,
    targets: &[u8],
    destination_x: u8,
    destination_y: u8,
    command_book: &[NativeCommandRecord],
) -> Result<NativeItemRelocationResult> {
    todo!("battle/native_item_relocation.go:ApplyNativeCommand23Relocation")
}

/// Go：`native_item_relocation.go:74`，`func applyNativeCommand23Relocation(records []byte, actorUnit int, targets []byte, destinationX, destinationY byte, commandBook []NativeCommandRecord, requireMP bool) (NativeItemRelocationResult, error)`
pub(crate) fn apply_native_command23_relocation_inner(
    records: &mut [u8],
    actor_unit: i64,
    targets: &[u8],
    destination_x: u8,
    destination_y: u8,
    command_book: &[NativeCommandRecord],
    require_mp: bool,
) -> Result<NativeItemRelocationResult> {
    todo!("battle/native_item_relocation.go:applyNativeCommand23Relocation")
}
