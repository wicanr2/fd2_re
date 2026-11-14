//! D 組：第 7 指令演出的測試。Go 來源：`native_command7_presentation_test.go`；夾具在 `native_command1_presentation_test.rs`。

use super::*;
use crate::game::native_command1_presentation::native_command1_presentation_test::{
    ACTOR, TARGET, callback_counter, command_damage_test_plan, command_record, command_test_state, count, nil_frames,
    publish_mp_spec, resist, unit,
};

fn test_plan() -> (battle::State, battle::NativeCommandDamagePlan) {
    command_damage_test_plan(7, command_record(7, 90, 100, 1, 0, 2))
}

/// Go：`native_command7_presentation_test.go:29` `TestNativeCommand7PresentationPublishesOnlyAfterDraw`
#[test]
fn test_native_command7_presentation_publishes_only_after_draw() {
    let (state, plan) = test_plan();
    let (counter, then) = callback_counter();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    let mut job = NativeCommand7PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan.clone()),
        prelude: nil_frames(1),
        actor_black: nil_frames(1),
        actor_pulse: nil_frames(1),
        actor_specs: publish_mp_spec(),
        tail: nil_frames(1),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: vec![target.hp],
        then: Some(then),
        ..Default::default()
    };
    for stage in 1..=plan.damage_stages {
        job.handler.push(NativeCommand7HandlerFrame { target_index: 0, hp_stage: stage, ..Default::default() });
    }
    g.native_cmd7_presentation = Some(job);
    g.step_native_command7_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(a.mp == plan.mp_before && !a.acted && t.hp == plan.results[0].hp_before, "未繪製的第 7 指令改變了戰鬥狀態");
    while let Some(job) = g.native_cmd7_presentation.as_mut() {
        job.drawn = true;
        g.step_native_command7_presentation();
    }
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        a.mp == plan.mp_after
            && a.acted
            && t.hp == plan.results[0].hp_after
            && g.native_rng_state == plan.rng_after
            && count(&counter) == 1,
        "第 7 指令交易未完成：mp={} acted={} hp={} rng={:#x} callback={}",
        a.mp,
        a.acted,
        t.hp,
        g.native_rng_state,
        count(&counter)
    );
}

/// Go：`native_command7_presentation_test.go:58` `TestNativeCommand7PresentationRollsBackRuntimeFailure`
#[test]
fn test_native_command7_presentation_rolls_back_runtime_failure() {
    let (state, plan) = test_plan();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd7_presentation = Some(NativeCommand7PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan.clone()),
        phase: NATIVE_COMMAND7_ACTOR,
        actor_black: nil_frames(1),
        actor_pulse: nil_frames(1),
        actor_specs: publish_mp_spec(),
        handler: vec![NativeCommand7HandlerFrame { target_index: 0, hp_stage: 2, ..Default::default() }],
        tail: nil_frames(1),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: vec![target.hp],
        drawn: true,
        ..Default::default()
    });
    g.step_native_command7_presentation();
    assert_eq!(unit(&g, ACTOR).mp, plan.mp_after, "MP 標記未發布");
    g.native_cmd7_presentation.as_mut().expect("job").drawn = true;
    g.step_native_command7_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        g.native_cmd7_presentation.is_none()
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
