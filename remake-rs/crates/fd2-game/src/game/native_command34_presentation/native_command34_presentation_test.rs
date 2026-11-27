//! D 組：第 34 指令演出的測試。Go 來源：`native_command34_presentation_test.go`。
//!
//! Go 的單位不在 `g.st` 裡；Rust 把建立計畫的 `State` 放進 `g.st`（行動者索引 0、目標索引 1）。

use super::*;
use crate::game::native_command_target_ui::native_command_target_ui_test::ordinary_command_test_game;
use crate::test_support;

const ACTOR: UnitIdx = 0;
const TARGET: UnitIdx = 1;

/// Go：`native_command34_presentation_test.go:14` `nativeCommand34PlanFixture`
fn test_plan() -> (battle::State, battle::NativeCompoundCommand34Plan) {
    let mut actor = battle::Unit::default();
    (actor.camp, actor.on_field, actor.x, actor.y, actor.hp, actor.max_hp, actor.mp, actor.lv) =
        (battle::OWN, true, 0, 0, 40, 100, 60, 20);
    (actor.ap, actor.dp, actor.hit, actor.ev) = (100, 80, 60, 50);
    (actor.battle_fig, actor.has_battle_fig, actor.native_record_class, actor.has_native_record_class) =
        (4, true, 19, true);
    let mut target = battle::Unit::default();
    (target.camp, target.on_field, target.x, target.y, target.hp, target.max_hp, target.mp, target.lv) =
        (battle::OWN, true, 1, 0, 10, 90, 0, 12);
    (target.ap, target.dp, target.hit, target.ev) = (70, 60, 50, 40);
    (target.battle_fig, target.has_battle_fig, target.native_record_class, target.has_native_record_class) =
        (5, true, 2, true);
    let mut book = vec![battle::NativeCommandRecord::default(); battle::NATIVE_COMMAND_RECORD_COUNT as usize];
    for (id, entry) in book.iter_mut().enumerate() {
        entry.id = id as i64;
    }
    book[34] = battle::NativeCommandRecord {
        id: 34,
        selection_mode: 5,
        effect_mode: 3,
        mp_cost: 28,
        target_code: 1,
        ..Default::default()
    };
    let mut st = battle::State::default();
    (st.w, st.h) = (2, 1);
    st.units = vec![actor, target];
    st.native_composition_event_bytes = vec![0; 2];
    st.native_command_book = book;
    let plan = st.plan_native_compound_command34(Some(ACTOR), Some(TARGET), 7).expect("plan");
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

fn game(st: battle::State) -> Game {
    Game {
        native_rng_state: 7,
        native_map_work: vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize],
        native_map_vga: vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize],
        st: Some(st),
        ..Default::default()
    }
}

/// Go：`native_command34_presentation_test.go:119` `TestNativeCommand34PresentationPublishesStagesOnlyAfterDraw`
#[test]
fn test_native_command34_presentation_publishes_stages_only_after_draw() {
    let (st, plan) = test_plan();
    let mut g = game(st);
    let (base_work, base_vga) = (g.native_map_work.clone(), g.native_map_vga.clone());
    let mut j = NativeCommand34PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan),
        frames: vec![one_frame(), one_frame()],
        publish_at: [1, 9, 10],
        stage_end: [2, 9, 10],
        rng_before: 7,
        baseline_work: base_work.clone(),
        baseline_vga: base_vga.clone(),
        ..Default::default()
    };
    (j.stage_work[0], j.stage_vga[0]) = (base_work, base_vga);
    g.native_cmd34_presentation = Some(j);
    g.step_native_command34_presentation();
    assert_eq!(unit(&g, ACTOR).native_transient[0], 0, "published without draw");
    g.native_cmd34_presentation.as_mut().expect("job").drawn = true;
    g.step_native_command34_presentation();
    let a = unit(&g, ACTOR);
    assert!(a.native_transient[0] != 0 && a.native_transient[1] == 0 && !a.acted, "stage boundary actor={a:?}");
}

/// Go：`native_command34_presentation_test.go:137` `TestNativeCommand34PresentationLateFailureRollsBackStages`
#[test]
fn test_native_command34_presentation_late_failure_rolls_back_stages() {
    let (st, mut plan) = test_plan();
    let mut g = game(st);
    battle::publish_native_compound_command34_stage(presentation_units(&mut g.st), Some(&mut plan), 0)
        .expect("publish");
    g.native_rng_state = plan.result.stages[0].rng_state;
    g.native_cmd34_presentation = Some(NativeCommand34PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan),
        rng_before: 7,
        baseline_work: g.native_map_work.clone(),
        baseline_vga: g.native_map_vga.clone(),
        ..Default::default()
    });
    g.fail_native_command34_presentation(&game_err!("late failure"));
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        g.native_cmd34_presentation.is_none()
            && a.native_transient[0] == 0
            && t.native_transient[0] == 0
            && !a.acted
            && g.native_rng_state == 7
            && !g.load_err.is_empty(),
        "rollback actor={a:?} target={t:?}"
    );
}

/// Go：`native_command34_presentation_test.go:152` `TestResetActionOverlayCancelsNativeCommand34Stages`
#[test]
fn test_reset_action_overlay_cancels_native_command34_stages() {
    let (st, mut plan) = test_plan();
    let mut g = game(st);
    battle::publish_native_compound_command34_stage(presentation_units(&mut g.st), Some(&mut plan), 0)
        .expect("publish");
    g.native_cmd34_presentation = Some(NativeCommand34PresentationJob {
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
        g.native_cmd34_presentation.is_none()
            && a.native_transient[0] == 0
            && t.native_transient[0] == 0
            && !a.acted
            && g.native_rng_state == 7,
        "overlay reset retained command34 actor={a:?} target={t:?}"
    );
}

/// Go：`native_command34_presentation_test.go:31` `nativeCommand34PresentationFixture`：map0 的施放者 `(0,0)` 與
/// 同陣營目標 `(1,0)`，其他我方單位下場。
fn command34_test_game() -> Option<(Game, UnitIdx, UnitIdx)> {
    let (mut g, actor, target) = ordinary_command_test_game()?;
    let st = g.st.as_mut().unwrap();
    for index in 1..4 {
        st.units[index].on_field = false;
    }
    assert!(st.units[target].set_map_placement(1, 0, 3));
    st.units[target].materialize_native_map_presentation().unwrap();
    let a = &mut st.units[actor];
    (a.camp, a.mp, a.ap, a.dp, a.hit, a.ev, a.battle_fig) = (battle::OWN, 60, 100, 80, 60, 50, 4);
    let t = &mut st.units[target];
    (t.camp, t.ap, t.dp, t.hit, t.ev, t.battle_fig) = (battle::OWN, 70, 60, 50, 40, 5);
    st.native_command_book[34] = battle::NativeCommandRecord {
        id: 34,
        selection_mode: 5,
        effect_mode: 3,
        mp_cost: 28,
        target_code: 1,
        ..Default::default()
    };
    g.compose_native_map_frame().unwrap();
    Some((g, actor, target))
}

/// Go：`native_command34_presentation_test.go:87` `TestStartNativeCommand34PresentationUsesOriginalAssetsEndToEnd`
#[test]
fn test_start_native_command34_presentation_uses_original_assets_end_to_end() {
    if test_support::input("FD2_ASSET_PACK", "surfaces/BG_000/resource.json").is_none() {
        return;
    }
    if f_child_env(module_path!(), "test_start_native_command34_presentation_uses_original_assets_end_to_end", |_| {
        vec![("FD2_MUTE", "1".into())]
    })
    .is_none()
    {
        return;
    }
    let Some((mut g, actor, target)) = command34_test_game() else { return };
    let done = std::sync::Arc::new(std::sync::Mutex::new(Vec::new()));
    let sink = done.clone();
    g.start_native_command34_presentation(
        actor,
        Some(target),
        Box::new(move |_, result| sink.lock().unwrap().push(result.stage_states.len())),
    )
    .unwrap();
    let j = g.native_cmd34_presentation.as_mut().unwrap();
    assert!(j.frames.len() >= 150 && j.publish_at[0] > 0, "frames={} publish_at={:?}", j.frames.len(), j.publish_at);
    for frame in &mut j.frames {
        frame.delay = 1;
    }
    let mut screen = Picture::new_image(640, 400);
    for steps in 0..2000 {
        if g.native_cmd34_presentation.is_none() {
            break;
        }
        assert!(g.present_native_command34(Some(&mut screen)), "draw failed at {steps}: {}", g.load_err);
        g.step_native_command34_presentation();
        if let Some(j) = g.native_cmd34_presentation.as_mut().filter(|j| j.holding) {
            j.hold = 0;
        }
    }
    assert!(g.native_cmd34_presentation.is_none(), "err={}", g.load_err);
    let a = &g.st.as_ref().unwrap().units[actor];
    assert!(a.acted && a.native_transient[..3].iter().all(|&b| b != 0) && a.mp == 60, "actor={a:?}");
    assert_eq!(*done.lock().unwrap(), [3]);
}
