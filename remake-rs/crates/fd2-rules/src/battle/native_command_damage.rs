//! Go 來源：`remake/internal/battle/native_command_damage.go`（97 行）。

use super::*;

/// NativeCommandResistanceRow is one editable copy of the original
/// word_51F96 class multiplier.  The original uses ResistRaw directly: base
/// damage is record damage * ResistRaw / 10.  Keep the raw value rather than
/// deriving it from a presentation percentage at runtime.
///
/// Go：`native_command_damage.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCommandResistanceRow {
    /// Go 標籤：`json:"cls"`
    #[serde(rename = "cls")]
    pub class_id: i64,
    /// Go 標籤：`json:"resist_raw"`
    pub resist_raw: i64,
}

/// LoadNativeCommandResistances loads the verified class multiplier table.
/// Invalid IDs or values are rejected so an edited table cannot silently turn
/// a native command into a different formula.
///
/// Go：`native_command_damage.go:23`，`func LoadNativeCommandResistances(path string) (map[int]int, error)`
pub fn load_native_command_resistances(path: &str) -> Result<BTreeMap<i64, i64>> {
    let raw = read_file(path)?;
    let rows: Vec<NativeCommandResistanceRow> = decode_json::<Option<_>>(&raw)?.unwrap_or_default();
    let mut table = BTreeMap::new();
    for row in rows {
        if row.class_id <= 0 || row.resist_raw < 0 || row.resist_raw > 10 {
            return Err(battle_err!("invalid native command resistance class={} raw={}", row.class_id, row.resist_raw));
        }
        if table.contains_key(&row.class_id) {
            return Err(battle_err!("duplicate native command resistance class={}", row.class_id));
        }
        table.insert(row.class_id, row.resist_raw);
    }
    Ok(table)
}

/// NativeCommandDamage is the recovered numeric writer used by verified
/// player-dispatched commands 9..12.
/// Target geometry, animation, and post-resolution processing deliberately do
/// not live here: they are separate native contracts still being recovered.
///
/// Go：`native_command_damage.go:49`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandDamage {
    pub hit: bool,
    pub damage: i64,
}

/// nativeCommandResistanceClass preserves sub_1C75E's target record+0x20
/// reader at IDA LE 0x1C792. ClassID is the compatibility source only for
/// authored units without raw class provenance. An invalid supplied raw class
/// must fail the resistance-table preflight rather than fall back.
/// Evidence: docs/data/ida/fd2_command_damage_raw_class_20261002.json.
///
/// Go：`native_command_damage.go:59`，`func nativeCommandResistanceClass(target *Unit) int`
pub(crate) fn native_command_resistance_class(target: &Unit) -> i64 {
    if target.has_native_record_class {
        return target.native_record_class as i64;
    }
    target.class_id
}

/// ResolveNativeCommandDamage mirrors 0x1c75e -> 0x1c81f for a single target.
/// It consumes exactly two 0x4e893 steps on a hit (hit roll, then damage
/// variance), and only the hit-roll step on a miss. It does not mutate target.
///
/// Go：`native_command_damage.go:69`，`func ResolveNativeCommandDamage(recordDamage, hit, targetResistRaw int, rngState uint16) (NativeCommandDamage, uint16, error)`
pub fn resolve_native_command_damage(
    record_damage: i64,
    hit: i64,
    target_resist_raw: i64,
    rng_state: u16,
) -> Result<(NativeCommandDamage, u16)> {
    if record_damage < 0 || !(0..=100).contains(&hit) || !(0..=10).contains(&target_resist_raw) {
        return Err(battle_err!("invalid native damage={record_damage} hit={hit} resist_raw={target_resist_raw}"));
    }
    let mut rng_state = fdother::native_rng_step(rng_state);
    if (rng_state % 100) as i64 >= hit {
        return Ok((NativeCommandDamage::default(), rng_state));
    }
    let base = record_damage.wrapping_mul(target_resist_raw) / 10;
    rng_state = fdother::native_rng_step(rng_state);
    let damage = base.wrapping_mul(9) / 10 + ((rng_state % 100) as i64).wrapping_mul(base) / 1000;
    Ok((NativeCommandDamage { hit: true, damage }, rng_state))
}

/// ApplyNativeCommandDamage is the state mutation proven at 0x1c81f: subtract
/// the resolved damage from current HP and clamp at zero.  Callers must supply
/// the raw target-class multiplier; this method intentionally does not reuse
/// the legacy normalized spell resolver.
///
/// Go：`native_command_damage.go:87`，`func ApplyNativeCommandDamage(target *Unit, recordDamage, hit, targetResistRaw int, rngState uint16) (NativeCommandDamage, uint16, error)`
pub fn apply_native_command_damage(
    target: Option<&mut Unit>,
    record_damage: i64,
    hit: i64,
    target_resist_raw: i64,
    rng_state: u16,
) -> Result<(NativeCommandDamage, u16)> {
    let Some(target) = target else {
        return Err(battle_err!("nil target"));
    };
    let (result, next_rng) = resolve_native_command_damage(record_damage, hit, target_resist_raw, rng_state)?;
    if !result.hit {
        return Ok((result, next_rng));
    }
    target.apply_hp_damage(result.damage);
    Ok((result, next_rng))
}

#[cfg(test)]
mod native_command_damage_test;
