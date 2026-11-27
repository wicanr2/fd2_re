//! Go 來源：`remake/cmd/fd2/native_command32_presentation_test.go`。
//!
//! Go 另以 `FD2_ORIGINAL_*` 證明 FIGANI 只讀分離素材包；Rust 的指令演出本來就只讀分離素材包。
//! HD 面板（Go `maybeAttachCommandPanelHDTest`）屬 R6。

use super::*;
use crate::game::native_command_target_ui::native_command_target_ui_test::ordinary_command_test_game;
use crate::test_support;

fn command32_record() -> battle::NativeCommandRecord {
    battle::NativeCommandRecord {
        id: 32,
        damage: 50,
        hit: 100,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 76,
        target_code: 0,
    }
}

/// Go：`native_command32_presentation_test.go:25` `nativeCommand32PresentationFixture`。map0 的施放者與目標，
/// selector 4／5、指令 32 的紀錄。
fn command32_test_game() -> Option<(Game, UnitIdx, UnitIdx)> {
    let (mut g, actor, target) = ordinary_command_test_game()?;
    let st = g.st.as_mut().unwrap();
    st.units[actor].battle_fig = 4;
    st.units[target].battle_fig = 5;
    st.native_command_book[32] = command32_record();
    g.compose_native_map_frame().unwrap();
    Some((g, actor, target))
}

/// Go：`native_command32_presentation_test.go:126` `nativeCompound32FixtureForMain` 加 `nativeCommand32PresentationPlan`。
fn command32_plan() -> (battle::State, battle::NativeCompoundCommand32Plan) {
    let mut actor = battle::Unit::default();
    (actor.camp, actor.on_field, actor.hp, actor.max_hp, actor.mp, actor.lv) = (battle::OWN, true, 100, 100, 80, 20);
    (actor.class_id, actor.battle_fig, actor.has_battle_fig) = (5, 4, true);
    (actor.native_record_class, actor.has_native_record_class) = (19, true);
    let mut target = battle::Unit::default();
    (target.camp, target.on_field, target.x, target.hp, target.max_hp, target.lv) =
        (battle::ENEMY, true, 1, 100, 100, 12);
    (target.class_id, target.battle_fig, target.has_battle_fig) = (5, 5, true);
    (target.native_record_class, target.has_native_record_class) = (2, true);
    let mut book: Vec<battle::NativeCommandRecord> = (0..battle::NATIVE_COMMAND_RECORD_COUNT)
        .map(|id| battle::NativeCommandRecord { id, ..Default::default() })
        .collect();
    book[32] = command32_record();
    let mut st = battle::State::default();
    (st.w, st.h, st.units) = (2, 1, vec![actor, target]);
    (st.native_composition_event_bytes, st.native_command_book) = (vec![0; 2], book);
    st.native_command_resistances = BTreeMap::from([(2, 10)]);
    let plan = st.plan_native_compound_command32(Some(0), Some(1), 7).unwrap();
    (st, plan)
}

/// 只帶兩格空白畫面的工作：第 `publish_at` 格開始前寫入交易。
fn step_only_game(publish_at: i64, published: bool, frames: usize) -> (Game, battle::NativeCompoundCommand32Plan) {
    let (st, plan) = command32_plan();
    let baseline_work = vec![0u8; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
    let baseline_vga = vec![0u8; indexedmap::NATIVE_MAP_VGA_SIZE as usize];
    let mut post_vga = baseline_vga.clone();
    post_vga[0] = 9;
    let mut g = Game {
        st: Some(st),
        native_rng_state: 7,
        native_map_work: baseline_work.clone(),
        native_map_vga: baseline_vga.clone(),
        ..Default::default()
    };
    let image = Some(Arc::new(Picture::new_image(1, 1)));
    g.native_cmd32_presentation = Some(NativeCommand32PresentationJob {
        actor: Some(0),
        plan: Some(plan.clone()),
        frames: (0..frames)
            .map(|_| NativeCompoundPresentedFrame { image: image.clone(), delay: 1, ..Default::default() })
            .collect(),
        publish_at,
        published,
        rng_before: 7,
        baseline_work: baseline_work.clone(),
        baseline_vga,
        post_transaction_work: baseline_work,
        post_vga,
        ..Default::default()
    });
    (g, plan)
}

fn unit(g: &Game, index: UnitIdx) -> &battle::Unit {
    &g.st.as_ref().unwrap().units[index]
}

/// Go：`native_command32_presentation_test.go:94` `TestStartNativeCommand32PresentationUsesOriginalAssetsEndToEnd`
#[test]
fn test_start_native_command32_presentation_uses_original_assets_end_to_end() {
    if test_support::input("FD2_ASSET_PACK", "surfaces/BG_000/resource.json").is_none() {
        return;
    }
    if f_child_env(module_path!(), "test_start_native_command32_presentation_uses_original_assets_end_to_end", |_| {
        vec![("FD2_MUTE", "1".into())]
    })
    .is_none()
    {
        return;
    }
    let Some((mut g, actor, target)) = command32_test_game() else { return };
    let done = std::sync::Arc::new(std::sync::Mutex::new(Vec::new()));
    let sink = done.clone();
    g.start_native_command32_presentation(
        actor,
        Some(target),
        Box::new(move |_, result| sink.lock().unwrap().push(result.targets.len())),
    )
    .unwrap();
    let j = g.native_cmd32_presentation.as_mut().unwrap();
    assert!(j.frames.len() >= 100 && j.publish_at > 0, "frames={} publish_at={}", j.frames.len(), j.publish_at);
    for frame in &mut j.frames {
        frame.delay = 1;
    }
    let mut screen = Picture::new_image(640, 400);
    for steps in 0..1000 {
        if g.native_cmd32_presentation.is_none() {
            break;
        }
        assert!(g.present_native_command32(Some(&mut screen)), "command32 draw failed at {steps}: {}", g.load_err);
        g.step_native_command32_presentation();
        if let Some(j) = g.native_cmd32_presentation.as_mut().filter(|j| j.holding) {
            j.hold = 0;
        }
    }
    assert!(g.native_cmd32_presentation.is_none(), "err={}", g.load_err);
    let (a, t) = (unit(&g, actor), unit(&g, target));
    assert!(a.acted && t.hp < 100 && a.mp == 80, "actor mp={} target hp={}", a.mp, t.hp);
    assert_eq!(*done.lock().unwrap(), [1]);
}

/// Go：`native_command32_presentation_test.go:146` `TestNativeCommand32PresentationPublishesOnlyAfterDrawBoundary`
#[test]
fn test_native_command32_presentation_publishes_only_after_draw_boundary() {
    let (mut g, plan) = step_only_game(1, false, 2);
    let called = std::sync::Arc::new(std::sync::atomic::AtomicI64::new(0));
    let count = called.clone();
    g.native_cmd32_presentation.as_mut().unwrap().then = Some(Box::new(move |_, _| {
        count.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
    }));
    g.step_native_command32_presentation();
    assert!(unit(&g, 1).hp == 100 && !unit(&g, 0).acted && g.native_rng_state == 7, "published without Draw");
    g.native_cmd32_presentation.as_mut().unwrap().drawn = true;
    g.step_native_command32_presentation();
    assert_eq!(unit(&g, 1).hp, plan.result.targets[0].hp_after);
    assert!(!unit(&g, 0).acted && g.native_rng_state == plan.result.rng_state && g.native_map_vga[0] == 9);
    g.native_cmd32_presentation.as_mut().unwrap().drawn = true;
    g.step_native_command32_presentation();
    assert!(g.native_cmd32_presentation.as_ref().is_some_and(|j| j.holding) && !unit(&g, 0).acted);
    g.native_cmd32_presentation.as_mut().unwrap().hold = 0;
    g.step_native_command32_presentation();
    assert!(g.native_cmd32_presentation.is_none() && unit(&g, 0).acted);
    assert_eq!(called.load(std::sync::atomic::Ordering::SeqCst), 1);
}

/// Go：`native_command32_presentation_test.go:183` `TestNativeCommand32PresentationFailureRollsBackPublishedState`
#[test]
fn test_native_command32_presentation_failure_rolls_back_published_state() {
    let (mut g, _) = step_only_game(0, true, 1);
    let units = presentation_units(&mut g.st);
    let j = g.native_cmd32_presentation.as_mut().unwrap();
    battle::apply_native_compound_command32_target(units, j.plan.as_mut(), 0).unwrap();
    g.native_rng_state = j.plan.as_ref().unwrap().result.rng_state;
    g.fail_native_command32_presentation(&game_err!("late draw failure"));
    assert!(g.native_cmd32_presentation.is_none() && unit(&g, 1).hp == 100 && !unit(&g, 0).acted);
    assert!(g.native_rng_state == 7 && !g.load_err.is_empty());
}

/// Go：`native_command32_presentation_test.go:202` `TestResetActionOverlayCancelsNativeCommand32Transaction`
#[test]
fn test_reset_action_overlay_cancels_native_command32_transaction() {
    let (mut g, _) = step_only_game(0, false, 0);
    let units = presentation_units(&mut g.st);
    let j = g.native_cmd32_presentation.as_mut().unwrap();
    battle::apply_native_compound_command32_target(units, j.plan.as_mut(), 0).unwrap();
    g.reset_action_overlay_lifecycle();
    assert!(g.native_cmd32_presentation.is_none() && unit(&g, 1).hp == 100 && !unit(&g, 0).acted);
}
