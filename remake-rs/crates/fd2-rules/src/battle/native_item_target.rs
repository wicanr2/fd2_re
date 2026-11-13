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
    todo!("battle/native_item_target.go:NativeItemTargetPlanFromRow")
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
    todo!("battle/native_item_target.go:NativeItemEffectTargets")
}
