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
    if row_word as i64 >= NATIVE_COMMAND_RECORD_COUNT {
        return (NativeItemCommandDamageRoute::default(), false);
    }
    let presentation = match item_type {
        20 | 24 => 0x1cd17,
        21 => 0x1cac7,
        _ => return (NativeItemCommandDamageRoute::default(), false),
    };
    (
        NativeItemCommandDamageRoute { item_type, command_id: row_word as i64, presentation, consumes_source: false },
        true,
    )
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
    let (verified, ok) = native_item_command_damage_route_for_type(route.item_type, route.command_id as u16);
    if !ok || verified != route {
        return Err(battle_err!("native item command-damage route is invalid: {}", route.go_sharp_v()));
    }
    let command_id = route.command_id;
    if command_id >= command_book.len() as i64 || command_book[command_id as usize].id != command_id {
        return Err(battle_err!("native command record {command_id} is unavailable"));
    }
    let record = command_book[command_id as usize];
    if record.damage < 0 || record.hit < 0 || record.hit > 100 {
        return Err(battle_err!("native command record {command_id} has invalid damage/hit"));
    }
    // Go 的 `target == nil` 檢查在 Rust 不存在：`targets` 是單位本體的切片。
    for target in targets.iter() {
        let class = native_command_resistance_class(target);
        match resistances.get(&class) {
            Some(&r) if (0..=10).contains(&r) => {}
            _ => return Err(battle_err!("native resistance class {class} is unavailable")),
        }
    }

    let mut rng_state = rng_state;
    let mut results = Vec::with_capacity(targets.len());
    for target in targets.iter_mut() {
        let resistance = resistances[&native_command_resistance_class(target)];
        let (result, next_rng) =
            apply_native_command_damage(Some(target), record.damage, record.hit, resistance, rng_state)?;
        rng_state = next_rng;
        results.push(result);
    }
    Ok((results, rng_state))
}

impl NativeItemCommandDamageRoute {
    /// Go `%#v`。
    pub(crate) fn go_sharp_v(&self) -> String {
        go_sharp_v(
            "NativeItemCommandDamageRoute",
            &[
                ("ItemType", GoV::U(self.item_type as u64)),
                ("CommandID", GoV::I(self.command_id)),
                ("Presentation", GoV::U(self.presentation as u64)),
                ("ConsumesSource", GoV::B(self.consumes_source)),
            ],
        )
    }
}

#[cfg(test)]
mod native_item_command_damage_test;
