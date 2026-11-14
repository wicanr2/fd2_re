//! D 組：第 6 指令演出的測試。Go 來源：`native_command6_presentation_test.go`；夾具在 `native_command1_presentation_test.rs`。

use super::*;
use crate::game::native_command1_presentation::native_command1_presentation_test::{
    ACTOR, TARGET, callback_counter, command_damage_test_plan, command_record, command_test_state, count, nil_frames,
    publish_mp_spec, resist, unit,
};

fn test_plan() -> (battle::State, battle::NativeCommandDamagePlan) {
    let state = command_test_state(command_record(6, 90, 100, 1, 0, 2), (battle::OWN, 5), (battle::ENEMY, 103));
    let schedule = figani::NativeCommand6PresentationSchedule {
        effect_resource: 33,
        base_byte: 90,
        dword_table: [-10, -8, -3, 0, 0],
        ..Default::default()
    };
    // Go `figani.WalkNativeCommand6RNG(3, schedule, 0, count, resolve)`；規則層與格式層的錯誤型別在此轉接。
    let mut walk =
        |count: i64, resolve: &mut dyn FnMut(i64, u16) -> battle::Result<(u16, bool)>| -> battle::Result<u16> {
            let mut bridge =
                |index: i64, rng: u16| resolve(index, rng).map_err(|err| Box::new(err) as figani::ResolveError);
            figani::walk_native_command6_rng(3, schedule, 0, count, Some(&mut bridge))
                .map_err(|err| battle::BattleError::new(err.error.to_string()))
        };
    let plan = state
        .plan_native_command_damage_walk(Some(ACTOR), Some(TARGET), 6, &resist(), 3, Some(&mut walk))
        .expect("plan");
    (state, plan)
}

/// Go：`native_command6_presentation_test.go:54` `TestNativeCommand6PresentationPublishesOnlyAfterDraw`
#[test]
fn test_native_command6_presentation_publishes_only_after_draw() {
    let (state, plan) = test_plan();
    let (counter, then) = callback_counter();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    let mut job = NativeCommand6PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan.clone()),
        prelude: nil_frames(1),
        actor_black: nil_frames(1),
        actor_pulse: nil_frames(1),
        actor_specs: publish_mp_spec(),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: vec![target.hp],
        then: Some(then),
        ..Default::default()
    };
    for stage in 1..=plan.damage_stages {
        job.handler.push(NativeCommand6HandlerFrame { target_index: 0, hp_stage: stage, ..Default::default() });
    }
    g.native_cmd6_presentation = Some(job);
    g.step_native_command6_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(a.mp == plan.mp_before && !a.acted && t.hp == plan.results[0].hp_before, "未繪製的第 6 指令改變了戰鬥狀態");
    while let Some(job) = g.native_cmd6_presentation.as_mut() {
        job.drawn = true;
        g.step_native_command6_presentation();
    }
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        a.mp == plan.mp_after
            && a.acted
            && t.hp == plan.results[0].hp_after
            && g.native_rng_state == plan.rng_after
            && count(&counter) == 1,
        "第 6 指令交易未完成：mp={} acted={} hp={} rng={:#x} callback={}",
        a.mp,
        a.acted,
        t.hp,
        g.native_rng_state,
        count(&counter)
    );
}

/// Go：`native_command6_presentation_test.go:86` `TestNativeCommand6PresentationRollsBackRuntimeFailure`
#[test]
fn test_native_command6_presentation_rolls_back_runtime_failure() {
    let (state, plan) = test_plan();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd6_presentation = Some(NativeCommand6PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan.clone()),
        phase: NATIVE_COMMAND6_ACTOR,
        actor_black: nil_frames(1),
        actor_pulse: nil_frames(1),
        actor_specs: publish_mp_spec(),
        handler: vec![NativeCommand6HandlerFrame { target_index: 0, hp_stage: 2, ..Default::default() }],
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: vec![target.hp],
        drawn: true,
        ..Default::default()
    });
    g.step_native_command6_presentation();
    assert_eq!(unit(&g, ACTOR).mp, plan.mp_after, "MP 標記未發布");
    g.native_cmd6_presentation.as_mut().expect("job").drawn = true;
    g.step_native_command6_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        g.native_cmd6_presentation.is_none()
            && a.mp == plan.mp_before
            && !a.acted
            && t.hp == plan.results[0].hp_before
            && g.native_rng_state == plan.rng_before
            && !g.load_err.is_empty(),
        "失敗回復不完整：mp={} acted={} hp={} rng={:#x} err={:?}",
        a.mp,
        a.acted,
        t.hp,
        g.native_rng_state,
        g.load_err
    );
}
