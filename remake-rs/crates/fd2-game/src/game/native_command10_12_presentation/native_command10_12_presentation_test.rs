//! D 組：第 10～12 指令演出的測試。Go 來源：`native_command10_12_presentation_test.go`；夾具在
//! `native_command1_presentation_test.rs`。

use super::*;
use crate::game::native_command_target_ui::native_command_target_ui_test::ordinary_command_test_game;
use crate::game::native_command1_presentation::native_command1_presentation_test::{
    ACTOR, TARGET, callback_counter, command_record, command_test_state, count, resist, unit,
};
use crate::test_support;

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

/// Go：`modern_hd_commands10_12_map_test.go:15` `command1012HDTestGame` 與 `modern_hd_player9_map_test.go:40`
/// `player9MapTestGame`（命中版）：map0 的施放者 `(1,6)`、目標 `(2,6)`。HD 主題屬 R6。
fn command1012_test_game(id: i64) -> Option<(Game, UnitIdx, UnitIdx)> {
    let (mut g, actor, target) = ordinary_command_test_game()?;
    let st = g.st.as_mut().unwrap();
    let record = battle::NativeCommandRecord {
        id,
        damage: 999,
        hit: 100,
        selection_mode: 3,
        effect_mode: 0,
        mp_cost: 30,
        target_code: 0,
    };
    st.native_command_book[9] = battle::NativeCommandRecord { id: 9, ..record };
    st.native_command_book[id as usize] = record;
    assert!(st.units[actor].set_map_placement(1, 6, 3) && st.units[target].set_map_placement(2, 6, 1));
    st.units[actor].materialize_native_map_presentation().unwrap();
    st.units[target].materialize_native_map_presentation().unwrap();
    st.materialize_native_map_view_state(battle::NativeMapViewState {
        camera_x: 0,
        camera_y: 0,
        cursor_x: 2,
        cursor_y: 6,
        visible_cursor_x: 2,
        visible_cursor_y: 6,
    })
    .unwrap();
    (g.cur_x, g.cur_y) = (2, 6);
    g.compose_native_map_frame().unwrap();
    Some((g, actor, target))
}

/// Go：`native_command10_12_presentation_test.go:17` `TestNativeCommand1012FormalPlayerConfirmAndReturn`
#[test]
fn test_native_command1012_formal_player_confirm_and_return() {
    if test_support::input("FD2_ASSET_PACK", "surfaces/BG_000/resource.json").is_none() {
        return;
    }
    if f_child_env(module_path!(), "test_native_command1012_formal_player_confirm_and_return", |_| {
        vec![("FD2_MUTE", "1".into())]
    })
    .is_none()
    {
        return;
    }
    let Some(spells) = test_support::remake_asset("spells.json") else { return };
    let book = battle::load_native_command_records(&spells).expect("spells");
    for id in [10, 11, 12] {
        let Some((mut g, actor, target)) = command1012_test_game(id) else { return };
        let st = g.st.as_mut().unwrap();
        st.native_command_book = book.clone();
        (st.units[actor].mp, st.units[actor].max_mp) = (150, 150);
        (st.units[target].hp, st.units[target].max_hp) = (900, 900);
        for (index, unit) in st.units.iter_mut().enumerate() {
            unit.has_native_record_byte5 = true;
            if index != actor && index != target {
                unit.native_record_byte5 |= 1;
            }
        }
        let (x, y) = (st.units[actor].x, st.units[actor].y);
        (g.sel, g.cur_x, g.cur_y) = (Some(actor), x, y);
        g.begin_native_command_target_selection(id).unwrap();
        g.confirm();
        let j =
            g.native_cmd1012.as_ref().unwrap_or_else(|| panic!("command{id} did not enter its map owner: {}", g.msg));
        let results = &j.plan.as_ref().unwrap().results;
        assert!(results.len() == 1 && results[0].target == Some(target), "command{id}");
        let unit = |g: &Game, i: UnitIdx| g.st.as_ref().unwrap().units[i].clone();
        assert!(unit(&g, actor).mp == 150 && !unit(&g, actor).acted && unit(&g, target).hp == 900);
        let mut screen = Picture::new_image(640, 400);
        for step in 0..1024 {
            if g.native_cmd1012.is_none() {
                break;
            }
            if g.native2189_a.is_some() {
                assert!(g.draw_native_2189a(Some(&mut screen)), "command{id} prelude did not draw");
                g.step_native_2189a();
            } else {
                assert!(
                    g.present_native_command1012(Some(&mut screen)),
                    "command{id} draw failed at {step}: {}",
                    g.load_err
                );
            }
            g.step_native_command1012_presentation();
        }
        assert!(g.native_cmd1012.is_none() && g.native2189_a.is_none(), "command{id}: {}", g.load_err);
        let a = unit(&g, actor);
        assert!(a.mp == 150 - book[id as usize].mp_cost && a.acted, "command{id} mp={}", a.mp);
        assert!(!g.native_command0_targeting && g.sel.is_none());
        assert_eq!(g.st.as_ref().unwrap().native_map_range_mode, 1);
    }
}
