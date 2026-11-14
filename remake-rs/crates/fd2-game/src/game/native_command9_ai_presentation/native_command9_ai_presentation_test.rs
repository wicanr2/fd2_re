//! D 組：第 9 指令 AI 演出的測試。Go 來源：`native_command9_ai_presentation_test.go`；夾具在 `native_command1_presentation_test.rs`。

use super::*;
use crate::game::native_command1_presentation::native_command1_presentation_test::{
    ACTOR, TARGET, callback_counter, command_damage_test_plan, command_record, command_test_state, count, nil_frames,
    publish_mp_spec, resist, unit,
};

fn test_plan() -> (battle::State, battle::NativeCommandDamagePlan) {
    let state = command_test_state(command_record(9, 999, 50, 3, 0, 30), (battle::ENEMY, 40), (battle::OWN, 403));
    let plan =
        state.plan_native_ai_command_damage_single_target(Some(ACTOR), Some(TARGET), 9, &resist(), 3).expect("plan");
    assert!(
        plan.results.len() == 1 && plan.damage_stages == 20,
        "plan results={} stages={}",
        plan.results.len(),
        plan.damage_stages
    );
    (state, plan)
}

/// Go：`native_command9_ai_presentation_test.go:31` `TestNativeCommand9AIPresentationPublishesOnlyAfterDraw`
#[test]
fn test_native_command9_ai_presentation_publishes_only_after_draw() {
    let (state, plan) = test_plan();
    let (counter, then) = callback_counter();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    let mut job = NativeCommand9AIPresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan.clone()),
        prelude: nil_frames(1),
        actor_black: nil_frames(1),
        actor_pulse: nil_frames(1),
        actor_specs: publish_mp_spec(),
        tail: nil_frames(1),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: target.hp,
        then: Some(then),
        ..Default::default()
    };
    for stage in 1..=plan.damage_stages {
        job.handler.push(NativeCommand9AIHandlerFrame { hp_stage: stage, ..Default::default() });
    }
    g.native_cmd9_ai_presentation = Some(job);
    g.step_native_command9_ai_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(a.mp == plan.mp_before && !a.acted && t.hp == plan.results[0].hp_before, "未繪製的第 9 指令改變了戰鬥狀態");
    while let Some(job) = g.native_cmd9_ai_presentation.as_mut() {
        job.drawn = true;
        g.step_native_command9_ai_presentation();
    }
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        a.mp == plan.mp_after
            && a.acted
            && t.hp == plan.results[0].hp_after
            && g.native_rng_state == plan.rng_after
            && count(&counter) == 1,
        "第 9 指令交易未完成：mp={} acted={} hp={} rng={:#x} callback={}",
        a.mp,
        a.acted,
        t.hp,
        g.native_rng_state,
        count(&counter)
    );
}

/// Go：`native_command9_ai_presentation_test.go:52` `TestNativeCommand9AIPresentationRollsBackOutOfOrderStage`
#[test]
fn test_native_command9_ai_presentation_rolls_back_out_of_order_stage() {
    let (state, plan) = test_plan();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd9_ai_presentation = Some(NativeCommand9AIPresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan.clone()),
        phase: NATIVE_COMMAND9_AI_ACTOR,
        actor_black: nil_frames(1),
        actor_pulse: nil_frames(1),
        actor_specs: publish_mp_spec(),
        handler: vec![NativeCommand9AIHandlerFrame { hp_stage: 2, ..Default::default() }],
        tail: nil_frames(1),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: target.hp,
        drawn: true,
        ..Default::default()
    });
    g.step_native_command9_ai_presentation();
    assert_eq!(unit(&g, ACTOR).mp, plan.mp_after, "MP 標記未發布");
    g.native_cmd9_ai_presentation.as_mut().expect("job").drawn = true;
    g.step_native_command9_ai_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        g.native_cmd9_ai_presentation.is_none()
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
