//! 移植自 `remake/internal/battle/native_item_target_test.go`。

use super::*;

fn field_unit(camp: Camp, x: i64, y: i64) -> Unit {
    Unit { camp, x, y, hp: 10, max_hp: 10, on_field: true, ..Default::default() }
}

/// Go：`TestNativeItemTargetPlanFromRowPreserves1BBDCCallFields`
#[test]
fn native_item_target_plan_from_row_preserves_1bbdc_call_fields() {
    let mut row = vec![0u8; NATIVE_ITEM_EFFECT_ROW_SIZE as usize];
    (row[0x0d], row[0x10], row[0x12], row[0x15]) = (0x17, 3, 1, 2);
    let plan = native_item_target_plan_from_row(&row).unwrap();
    assert!(
        plan.item_type == 0x17
            && plan.selection_mode == 3
            && plan.selection_inner_mark == 1
            && plan.effect_mode == 1
            && plan.target_code == 2,
        "plan={plan:?}"
    );
    row[0x0d] = 5;
    let got = native_item_target_plan_from_row(&row);
    assert!(matches!(got, Ok(p) if p.selection_inner_mark == 0), "non-0x17 plan={got:?}");
    assert!(
        native_item_target_plan_from_row(&row[..row.len() - 1]).is_err(),
        "short native item row unexpectedly accepted"
    );
}

/// Go：`TestNativeItemEffectTargetsUsesConfirmedSecondStageOrigin`
#[test]
fn native_item_effect_targets_uses_confirmed_second_stage_origin() {
    let units = vec![field_unit(OWN, 0, 0), field_unit(ENEMY, 2, 0), field_unit(ENEMY, 3, 0)];
    let (actor, confirmed, second) = (0, 1, 2);
    let plan = NativeItemTargetPlan { selection_mode: 2, effect_mode: 1, target_code: 0, ..Default::default() };
    let got = native_item_effect_targets(5, 1, Some(actor), Some(confirmed), plan, &[0; 5], &units);
    assert!(matches!(&got, Ok(t) if t.len() == 2 && t[0] == confirmed && t[1] == second), "targets={got:?}");
}

/// Go：`TestNativeItemEffectTargetsRejectsUnconfirmedCandidate`
#[test]
fn native_item_effect_targets_rejects_unconfirmed_candidate() {
    let units = vec![field_unit(OWN, 0, 0), field_unit(ENEMY, 3, 0)];
    let plan = NativeItemTargetPlan { selection_mode: 1, effect_mode: 1, target_code: 0, ..Default::default() };
    assert!(
        native_item_effect_targets(4, 1, Some(0), Some(1), plan, &[0; 4], &units).is_err(),
        "non-candidate confirmation unexpectedly accepted"
    );
}
