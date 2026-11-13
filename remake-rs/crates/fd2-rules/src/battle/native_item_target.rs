//! Go 來源：`remake/internal/battle/native_item_target.go`（71 行）。

use super::*;

/// NativeItemTargetPlan preserves the two 0x14818 calls in 0x1bbdc item
/// action case 0.  Field names describe only call roles, not gameplay range,
/// item effects, or normalized spell semantics.
///
/// Go：`native_item_target.go:8`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemTargetPlan {
    pub item_type: u8,
    pub selection_mode: i64,
    pub selection_inner_mark: i64,
    pub effect_mode: i64,
    pub target_code: i64,
}

/// NativeItemTargetPlanFromRow reads only the item-row bytes consumed by the
/// verified 0x1bbdc selector.  The first stage uses +0x10 and +0x15; type 0x17
/// alone supplies inner marker 1.  After 0x115b6 confirms one candidate, the
/// second stage uses +0x12, inner marker 0, and the same +0x15 target code.
///
/// Go：`native_item_target.go:20`，`func NativeItemTargetPlanFromRow(row []byte) (NativeItemTargetPlan, error)`
pub fn native_item_target_plan_from_row(row: &[u8]) -> Result<NativeItemTargetPlan> {
    if row.len() as i64 != NATIVE_ITEM_EFFECT_ROW_SIZE {
        return Err(battle_err!("native item row len={} want {}", row.len(), NATIVE_ITEM_EFFECT_ROW_SIZE));
    }
    let inner = if row[0x0d] == 0x17 { 1 } else { 0 };
    Ok(NativeItemTargetPlan {
        item_type: row[0x0d],
        selection_mode: row[0x10] as i64,
        selection_inner_mark: inner,
        effect_mode: row[0x12] as i64,
        target_code: row[0x15] as i64,
    })
}

/// NativeItemEffectTargets reproduces the mutation-free target transaction
/// around 0x115b6: actor-origin first-stage candidates, confirmed-candidate
/// validation, then confirmed-origin final effect targets.  The caller must
/// provide a row-derived plan and the original grid flags.
///
/// Go：`native_item_target.go:41`，`func NativeItemEffectTargets(w, h int, actor, confirmed *Unit, plan NativeItemTargetPlan, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：`actor`、`confirmed` 是 `units` 的索引（Go 傳 `*Unit`）。
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_item_effect_targets(
    w: i64,
    h: i64,
    actor: Option<UnitIdx>,
    confirmed: Option<UnitIdx>,
    plan: NativeItemTargetPlan,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    let raw_complete = native_target_roster_raw_complete(units);
    let (Some(actor_unit), Some(confirmed_unit)) = (actor.map(|i| &units[i]), confirmed.map(|i| &units[i])) else {
        return Err(battle_err!("invalid native item actor/confirmed unit"));
    };
    if !native_target_actor_usable(Some(actor_unit), raw_complete)
        || !native_target_actor_usable(Some(confirmed_unit), raw_complete)
    {
        return Err(battle_err!("invalid native item actor/confirmed unit"));
    }
    let selection = native_attack_candidates(
        w,
        h,
        Cell { x: actor_unit.x, y: actor_unit.y },
        plan.selection_mode,
        plan.selection_inner_mark,
        plan.target_code,
        flags,
        units,
    )?;
    if !selection.iter().any(|&candidate| Some(candidate) == confirmed) {
        return Err(battle_err!("confirmed unit is not a native item candidate"));
    }
    native_attack_candidates(
        w,
        h,
        Cell { x: confirmed_unit.x, y: confirmed_unit.y },
        plan.effect_mode,
        0,
        plan.target_code,
        flags,
        units,
    )
}

#[cfg(test)]
mod native_item_target_test;
