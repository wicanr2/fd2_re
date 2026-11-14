//! D 組：第 10～12 指令演出的測試。Go 來源：`native_command10_12_presentation_test.go`；夾具在
//! `native_command1_presentation_test.rs`。

use super::*;
use crate::game::native_command1_presentation::native_command1_presentation_test::{
    ACTOR, TARGET, callback_counter, command_record, command_test_state, count, resist, unit,
};

/// Go：`native_command10_12_presentation_test.go:180` `nativeCommand10TestPlan`
fn test_plan() -> (battle::State, battle::NativeCommandDamagePlan) {
    let state = command_test_state(command_record(10, 999, 50, 3, 0, 30), (battle::OWN, 40), (battle::ENEMY, 403));
    let plan = state.plan_native_command_damage(Some(ACTOR), Some(TARGET), 10, &resist(), 3).expect("plan");
    (state, plan)
}

fn job(plan: &battle::NativeCommandDamagePlan, actor: &battle::Unit, target: &battle::Unit) -> NativeCommand1012Job {
    NativeCommand1012Job {
        actor: Some(ACTOR),
        plan: Some(plan.clone()),
        baseline_work: vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize],
        baseline_vga: vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize],
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: vec![target.hp],
        rng_before: plan.rng_before,
        ..Default::default()
    }
}

/// Go：`native_command10_12_presentation_test.go:197` `TestNativeCommand1012PublishesOnlyAtRecoveredBoundaries`
#[test]
fn test_native_command1012_publishes_only_at_recovered_boundaries() {
    let (state, plan) = test_plan();
    let (counter, then) = callback_counter();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd1012 = Some(NativeCommand1012Job {
        schedule: fdother::NativeCommand10To12Schedule { result_hold_ms: 0, ..Default::default() },
        phase: NATIVE_COMMAND1012_MAIN,
        main_frames: vec![vec![0]],
        result_frames: vec![vec![0]],
        then: Some(then),
        ..job(&plan, &actor, &target)
    });
    g.step_native_command1012_presentation();
    assert!(
        unit(&g, ACTOR).mp == plan.mp_before && unit(&g, TARGET).hp == plan.results[0].hp_before,
        "未 Draw 即發布交易"
    );
    g.native_cmd1012.as_mut().expect("job").drawn = true;
    g.step_native_command1012_presentation();
    assert!(
        unit(&g, ACTOR).mp == plan.mp_after
            && unit(&g, TARGET).hp == plan.results[0].hp_after
            && !unit(&g, ACTOR).acted,
        "主演出邊界發布錯誤"
    );
    g.native_cmd1012.as_mut().expect("job").drawn = true;
    g.step_native_command1012_presentation();
    g.step_native_command1012_presentation();
    assert!(
        g.native_cmd1012.is_none()
            && unit(&g, ACTOR).acted
            && g.native_rng_state == plan.rng_after
            && count(&counter) == 1,
        "結果 hold 後未完成交易"
    );
}

/// Go：`native_command10_12_presentation_test.go:225` `TestNativeCommand1012RollsBackLatePublishFailure`
#[test]
fn test_native_command1012_rolls_back_late_publish_failure() {
    let (state, plan) = test_plan();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd1012 = Some(NativeCommand1012Job {
        phase: NATIVE_COMMAND1012_MAIN,
        main_frames: vec![vec![0]],
        drawn: true,
        ..job(&plan, &actor, &target)
    });
    g.st.as_mut().expect("st").units[TARGET].hp -= 1;
    g.step_native_command1012_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        g.native_cmd1012.is_none()
            && a.mp == plan.mp_before
            && !a.acted
            && t.hp == plan.results[0].hp_before
            && g.native_rng_state == plan.rng_before
            && !g.load_err.is_empty(),
        "晚期失敗未完整回復"
    );
}

/// Go：`native_command10_12_presentation_test.go:240` `TestNativeCommand1012PreludeCannotDisappearSilently`
#[test]
fn test_native_command1012_prelude_cannot_disappear_silently() {
    let (state, plan) = test_plan();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd1012 = Some(NativeCommand1012Job { phase: NATIVE_COMMAND1012_PRELUDE, ..job(&plan, &actor, &target) });
    g.step_native_command1012_presentation();
    assert!(
        g.native_cmd1012.is_none()
            && unit(&g, ACTOR).mp == plan.mp_before
            && unit(&g, TARGET).hp == plan.results[0].hp_before
            && !g.load_err.is_empty(),
        "前奏消失未失敗即關閉"
    );
}
