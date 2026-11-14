//! D 組：第 33 指令演出的測試。Go 來源：`native_command33_presentation_test.go`。
//!
//! Go 的單位不在 `g.st` 裡；Rust 把建立計畫的 `State` 放進 `g.st`（行動者索引 0、目標索引 1）。

use super::*;

const ACTOR: UnitIdx = 0;
const TARGET: UnitIdx = 1;

/// Go：`native_command33_presentation_test.go:14` `nativeCompound33FixtureForMain`
fn compound33_fixture() -> battle::State {
    let mut actor = battle::Unit::default();
    (actor.camp, actor.on_field, actor.x, actor.y, actor.hp, actor.max_hp, actor.mp, actor.lv) =
        (battle::OWN, true, 0, 0, 40, 100, 60, 20);
    (actor.battle_fig, actor.has_battle_fig, actor.native_record_class, actor.has_native_record_class) =
        (4, true, 19, true);
    actor.native_transient = [1, 2, 3, 4, 5, 6];
    let mut target = battle::Unit::default();
    (target.camp, target.on_field, target.x, target.y, target.hp, target.max_hp, target.lv) =
        (battle::OWN, true, 1, 0, 10, 90, 12);
    (target.battle_fig, target.has_battle_fig, target.native_record_class, target.has_native_record_class) =
        (5, true, 2, true);
    target.native_transient = [7, 8, 9, 10, 11, 12];
    let mut book = vec![battle::NativeCommandRecord::default(); battle::NATIVE_COMMAND_RECORD_COUNT as usize];
    for (id, entry) in book.iter_mut().enumerate() {
        entry.id = id as i64;
    }
    book[33] = battle::NativeCommandRecord {
        id: 33,
        selection_mode: 5,
        effect_mode: 3,
        mp_cost: 52,
        target_code: 1,
        ..Default::default()
    };
    let mut st = battle::State::default();
    (st.w, st.h) = (2, 1);
    st.units = vec![actor, target];
    st.native_composition_event_bytes = vec![0; 2];
    st.native_command_book = book;
    st
}

/// Go：`native_command33_presentation_test.go:36` `nativeCommand33PresentationPlan`
fn test_plan() -> (battle::State, battle::NativeCompoundCommand33Plan) {
    let st = compound33_fixture();
    let plan = st.plan_native_compound_command33(Some(ACTOR), Some(TARGET), 7).expect("plan");
    (st, plan)
}

fn unit(g: &Game, index: UnitIdx) -> &battle::Unit {
    &g.st.as_ref().expect("st").units[index]
}

/// Go 的 `{image: ebiten.NewImage(1, 1), delay: 1}`。
fn one_frame() -> NativeCompoundPresentedFrame {
    NativeCompoundPresentedFrame {
        image: Some(Arc::new(Picture { width: 1, height: 1, pix: vec![0; 4] })),
        delay: 1,
        ..Default::default()
    }
}

/// Go：`native_command33_presentation_test.go:142` `TestNativeCommand33PresentationPublishesOnlyAfterDrawBoundary`
#[test]
fn test_native_command33_presentation_publishes_only_after_draw_boundary() {
    let (st, plan) = test_plan();
    let baseline_work = vec![0u8; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
    let baseline_vga = vec![0u8; indexedmap::NATIVE_MAP_VGA_SIZE as usize];
    let mut post_vga = baseline_vga.clone();
    post_vga[0] = 9;
    let restore_rng = plan.result.restore.rng_state;
    let mut g = Game {
        native_rng_state: 7,
        native_map_work: baseline_work.clone(),
        native_map_vga: baseline_vga.clone(),
        st: Some(st),
        ..Default::default()
    };
    g.native_cmd33_presentation = Some(NativeCommand33PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan),
        frames: vec![one_frame(), one_frame()],
        publish_at: 1,
        rng_before: 7,
        post_transaction_work: baseline_work.clone(),
        post_vga,
        baseline_work,
        baseline_vga,
        ..Default::default()
    });
    g.step_native_command33_presentation();
    assert!(
        unit(&g, ACTOR).hp == 40 && unit(&g, TARGET).hp == 10 && !unit(&g, ACTOR).acted,
        "command33 published without Draw acknowledgement"
    );
    g.native_cmd33_presentation.as_mut().expect("job").drawn = true;
    g.step_native_command33_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        a.hp == a.max_hp
            && t.hp == t.max_hp
            && !a.acted
            && g.native_rng_state == restore_rng
            && g.native_map_vga[0] == 9,
        "command33 publication boundary actor={a:?} target={t:?}"
    );
    g.native_cmd33_presentation.as_mut().expect("job").drawn = true;
    g.step_native_command33_presentation();
    assert!(
        g.native_cmd33_presentation.as_ref().is_some_and(|j| j.holding) && !unit(&g, ACTOR).acted,
        "command33 completed before result hold"
    );
    g.native_cmd33_presentation.as_mut().expect("job").hold = 0;
    g.step_native_command33_presentation();
    assert!(
        g.native_cmd33_presentation.is_none() && unit(&g, ACTOR).acted,
        "command33 completion actor={:?}",
        unit(&g, ACTOR)
    );
}

/// Go：`native_command33_presentation_test.go:175` `TestNativeCommand33PresentationFailureRollsBackPublishedState`
#[test]
fn test_native_command33_presentation_failure_rolls_back_published_state() {
    let (st, mut plan) = test_plan();
    let baseline_work = vec![0u8; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
    let baseline_vga = vec![0u8; indexedmap::NATIVE_MAP_VGA_SIZE as usize];
    let mut g = Game {
        native_rng_state: 7,
        native_map_work: baseline_work.clone(),
        native_map_vga: baseline_vga.clone(),
        st: Some(st),
        ..Default::default()
    };
    battle::publish_native_compound_command33(presentation_units(&mut g.st), Some(&mut plan)).expect("publish");
    g.native_rng_state = plan.result.restore.rng_state;
    g.native_cmd33_presentation = Some(NativeCommand33PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan),
        rng_before: 7,
        baseline_work,
        baseline_vga,
        ..Default::default()
    });
    g.fail_native_command33_presentation(&game_err!("late draw failure"));
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        g.native_cmd33_presentation.is_none()
            && a.hp == 40
            && t.hp == 10
            && !a.acted
            && g.native_rng_state == 7
            && !g.load_err.is_empty(),
        "command33 rollback actor={a:?} target={t:?} rng={:#x} err={:?}",
        g.native_rng_state,
        g.load_err
    );
}

/// Go：`native_command33_presentation_test.go:191` `TestResetActionOverlayCancelsNativeCommand33Transaction`
#[test]
fn test_reset_action_overlay_cancels_native_command33_transaction() {
    let (st, mut plan) = test_plan();
    let mut g = Game {
        native_rng_state: 7,
        native_map_work: vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize],
        native_map_vga: vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize],
        st: Some(st),
        ..Default::default()
    };
    battle::publish_native_compound_command33(presentation_units(&mut g.st), Some(&mut plan)).expect("publish");
    g.native_cmd33_presentation = Some(NativeCommand33PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan),
        rng_before: 7,
        baseline_work: g.native_map_work.clone(),
        baseline_vga: g.native_map_vga.clone(),
        ..Default::default()
    });
    g.reset_action_overlay_lifecycle();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        g.native_cmd33_presentation.is_none() && a.hp == 40 && t.hp == 10 && !a.acted,
        "overlay reset retained command33 actor={a:?} target={t:?}"
    );
}
