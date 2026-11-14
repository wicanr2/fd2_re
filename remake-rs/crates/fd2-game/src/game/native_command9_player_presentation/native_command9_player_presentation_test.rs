//! D 組：第 9 指令玩家演出的測試。Go 來源：`native_command9_player_presentation_test.go`；夾具在
//! `native_command1_presentation_test.rs`。

use super::*;
use crate::game::native_command1_presentation::native_command1_presentation_test::{
    ACTOR, TARGET, callback_counter, command_record, command_test_state, count, resist, unit,
};

/// Go：`native_command9_player_presentation_test.go:11` `nativeCommand9PlayerTestPlan`
fn test_plan() -> (battle::State, battle::NativeCommandDamagePlan) {
    let state = command_test_state(command_record(9, 999, 50, 3, 0, 30), (battle::OWN, 40), (battle::ENEMY, 403));
    let plan = state.plan_native_command_damage(Some(ACTOR), Some(TARGET), 9, &resist(), 3).expect("plan");
    (state, plan)
}

fn baseline_work() -> Vec<u8> {
    vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize]
}

fn baseline_vga() -> Vec<u8> {
    vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize]
}

/// Go：`native_command9_player_presentation_test.go:28` `TestNativeCommand9PlayerPublishesOnlyAtRecoveredBoundaries`
#[test]
fn test_native_command9_player_publishes_only_at_recovered_boundaries() {
    let (state, plan) = test_plan();
    let (counter, then) = callback_counter();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd9_player = Some(NativeCommand9PlayerJob {
        actor: Some(ACTOR),
        target: Some(TARGET),
        plan: Some(plan.clone()),
        palettes: vec![ColorPalette::new(); 1],
        effect_frames: vec![Vec::new(); 1],
        result_frames: vec![Vec::new(); 1],
        baseline_work: baseline_work(),
        baseline_vga: baseline_vga(),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: target.hp,
        rng_before: plan.rng_before,
        then: Some(then),
        ..Default::default()
    });
    g.step_native_command9_player_presentation();
    assert!(
        unit(&g, ACTOR).mp == plan.mp_before && unit(&g, TARGET).hp == plan.results[0].hp_before,
        "未 Draw 即發布交易"
    );
    g.native_cmd9_player.as_mut().expect("job").drawn = true;
    g.step_native_command9_player_presentation();
    assert!(
        unit(&g, ACTOR).mp == plan.mp_before && unit(&g, TARGET).hp == plan.results[0].hp_before,
        "palette 後過早發布交易"
    );
    g.native_cmd9_player.as_mut().expect("job").drawn = true;
    g.step_native_command9_player_presentation();
    assert!(
        unit(&g, ACTOR).mp == plan.mp_after
            && unit(&g, TARGET).hp == plan.results[0].hp_after
            && !unit(&g, ACTOR).acted,
        "effect 邊界發布錯誤"
    );
    g.native_cmd9_player.as_mut().expect("job").drawn = true;
    g.step_native_command9_player_presentation();
    g.native_cmd9_player.as_mut().expect("job").hold = 0;
    g.step_native_command9_player_presentation();
    assert!(
        g.native_cmd9_player.is_none()
            && unit(&g, ACTOR).acted
            && g.native_rng_state == plan.rng_after
            && count(&counter) == 1,
        "result hold 後未完成交易"
    );
}

/// Go：`native_command9_player_presentation_test.go:62` `TestNativeCommand9PlayerRollsBackLatePublishFailure`
#[test]
fn test_native_command9_player_rolls_back_late_publish_failure() {
    let (state, plan) = test_plan();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd9_player = Some(NativeCommand9PlayerJob {
        actor: Some(ACTOR),
        target: Some(TARGET),
        plan: Some(plan.clone()),
        phase: NATIVE_COMMAND9_PLAYER_EFFECT,
        effect_frames: vec![Vec::new(); 1],
        baseline_work: baseline_work(),
        baseline_vga: baseline_vga(),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: target.hp,
        rng_before: plan.rng_before,
        drawn: true,
        ..Default::default()
    });
    g.st.as_mut().expect("st").units[TARGET].hp -= 1;
    g.step_native_command9_player_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        g.native_cmd9_player.is_none()
            && a.mp == plan.mp_before
            && !a.acted
            && t.hp == plan.results[0].hp_before
            && g.native_rng_state == plan.rng_before
            && !g.load_err.is_empty(),
        "晚期失敗未完整回復"
    );
}
