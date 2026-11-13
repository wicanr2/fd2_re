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
    if item_type != 23 {
        return (NativeItemRelocationRoute::default(), false);
    }
    (
        NativeItemRelocationRoute {
            item_type,
            row_word,
            command_id: 23,
            required_identity: 24,
            required_max_mp: 20,
            presentation: 0x22253,
            consumes_source: false,
        },
        true,
    )
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
    let (verified, ok) = native_item_relocation_route_for_type(route.item_type, route.row_word);
    if !ok || verified != route {
        return Err(battle_err!("native item relocation route is invalid: {}", route.go_sharp_v()));
    }
    let actor_base = actor_unit.wrapping_mul(NATIVE_RECORD_SIZE);
    if actor_unit < 0 || actor_base < 0 || actor_base.wrapping_add(NATIVE_RECORD_SIZE) > records.len() as i64 {
        return Err(record_bounds_error(actor_unit));
    }
    let actor = &records[actor_base as usize..(actor_base + NATIVE_RECORD_SIZE) as usize];
    if actor[0x08] != route.required_identity || u16::from_le_bytes([actor[0x46], actor[0x47]]) < route.required_max_mp
    {
        return Err(battle_err!("native item relocation identity/max MP gate failed"));
    }
    apply_native_command23_relocation_inner(
        records,
        actor_unit,
        targets,
        destination_x,
        destination_y,
        command_book,
        false,
    )
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
    apply_native_command23_relocation_inner(
        records,
        actor_unit,
        targets,
        destination_x,
        destination_y,
        command_book,
        true,
    )
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
    let Some(&first) = targets.first() else {
        return Err(battle_err!("native item relocation has no target"));
    };
    let target_unit = first as i64;
    let actor_base = actor_unit.wrapping_mul(NATIVE_RECORD_SIZE);
    let target_base = target_unit * NATIVE_RECORD_SIZE;
    if actor_unit < 0 || actor_base < 0 || actor_base.wrapping_add(NATIVE_RECORD_SIZE) > records.len() as i64 {
        return Err(record_bounds_error(actor_unit));
    }
    if target_base < 0 || target_base + NATIVE_RECORD_SIZE > records.len() as i64 {
        return Err(record_bounds_error(target_unit));
    }
    if command_book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT || command_book[23].id != 23 {
        return Err(battle_err!("native command record23 is unavailable"));
    }
    let (actor_base, target_base) = (actor_base as usize, target_base as usize);
    let cost = command_book[23].mp_cost;
    if !(0..=0xff).contains(&cost) {
        return Err(battle_err!("native command23 MP cost={cost} is invalid"));
    }
    let actor_mp = u16::from_le_bytes([records[actor_base + 0x44], records[actor_base + 0x45]]);
    if require_mp && (actor_mp as i64) < cost {
        return Err(battle_err!("native command23 insufficient MP"));
    }

    let remaining_mp = actor_mp.wrapping_sub(cost as u16);
    let target = &records[target_base..target_base + NATIVE_RECORD_SIZE as usize];
    let mut effective_level = target[0x21] as i64;
    let class_id = target[0x20];
    if class_id > 8 && class_id < 25 {
        effective_level += 30;
    }
    let result = NativeItemRelocationResult {
        target_unit,
        from_x: target[0],
        from_y: target[1],
        destination_x,
        destination_y,
        remaining_mp,
        accumulator_delta: effective_level * 10,
    };
    records[actor_base + 0x44..actor_base + 0x46].copy_from_slice(&remaining_mp.to_le_bytes());
    (records[target_base], records[target_base + 1]) = (destination_x, destination_y);
    Ok(result)
}

impl NativeItemRelocationRoute {
    /// Go `%#v`。
    pub(crate) fn go_sharp_v(&self) -> String {
        go_sharp_v(
            "NativeItemRelocationRoute",
            &[
                ("ItemType", GoV::U(self.item_type as u64)),
                ("RowWord", GoV::U(self.row_word as u64)),
                ("CommandID", GoV::I(self.command_id)),
                ("RequiredIdentity", GoV::U(self.required_identity as u64)),
                ("RequiredMaxMP", GoV::U(self.required_max_mp as u64)),
                ("Presentation", GoV::U(self.presentation as u64)),
                ("ConsumesSource", GoV::B(self.consumes_source)),
            ],
        )
    }
}

#[cfg(test)]
mod native_item_relocation_test;
