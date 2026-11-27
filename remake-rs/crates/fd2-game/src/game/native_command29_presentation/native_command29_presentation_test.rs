//! Go 來源：`remake/cmd/fd2/native_command29_presentation_test.go`、`native_command30_presentation_test.go`。
//!
//! Go 另以 `FD2_ORIGINAL_*` 證明 FIGANI 只讀分離素材包；Rust 的指令演出本來就只讀分離素材包。
//! HD 面板（Go `maybeAttachCommandPanelHDTest`）屬 R6。

use super::*;
use crate::test_support;

pub(crate) type Then = Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>;

pub(crate) fn noop() -> Then {
    Box::new(|_, _| {})
}

pub(crate) fn counter() -> (std::sync::Arc<std::sync::atomic::AtomicI64>, Then) {
    let done = std::sync::Arc::new(std::sync::atomic::AtomicI64::new(0));
    let count = done.clone();
    (done, Box::new(move |_, _| _ = count.fetch_add(1, std::sync::atomic::Ordering::SeqCst)))
}

pub(crate) fn spells() -> Option<Vec<battle::NativeCommandRecord>> {
    Some(battle::load_native_command_records(&test_support::remake_asset("spells.json")?).expect("spells"))
}

/// 子行程裡跑，設 `FD2_MUTE=1`（Go `t.Setenv("FD2_MUTE", "1")`）。
macro_rules! child {
    ($name:literal) => {
        if f_child_env(module_path!(), $name, |_| vec![("FD2_MUTE", "1".into())]).is_none() {
            return;
        }
    };
}
pub(crate) use child;

/// 我方施放者 `(0,0)`、敵方目標 `(1,0)`、`(2,0)`。回傳 `(g, actor, [target0, target1])`。
///
/// Go：`native_command29_presentation_test.go:80` `newNativeCommand29PresentationTestGame`
pub(crate) fn command29_test_game() -> Option<(Game, UnitIdx, [UnitIdx; 2])> {
    for resource in ["FIGANI_000", "FIGANI_009", "FIGANI_102", "FIGANI_104"] {
        test_support::input("FD2_ASSET_PACK", &format!("animations/{resource}/animation.json"))?;
    }
    let mut actor = battle::Unit::default();
    (actor.camp, actor.on_field, actor.lv, actor.ap) = (battle::OWN, true, 20, 120);
    (actor.mp, actor.max_mp, actor.hp, actor.max_hp) = (40, 40, 180, 180);
    (actor.battle_fig, actor.has_battle_fig) = (34, true);
    (actor.has_native_record_byte6, actor.has_native_record_byte8) = (true, true);
    let target = |x, lv, dp, hp, fig, byte8| {
        let mut t = battle::Unit::default();
        (t.camp, t.on_field, t.x, t.lv, t.dp, t.hp, t.max_hp) = (battle::ENEMY, true, x, lv, dp, hp, hp);
        (t.battle_fig, t.has_battle_fig) = (fig, true);
        (t.native_record_byte6, t.has_native_record_byte6) = (1, true);
        (t.native_record_byte8, t.has_native_record_byte8) = (byte8, true);
        t.has_native_record_byte5 = true;
        t
    };
    let mut book: Vec<battle::NativeCommandRecord> =
        (0..36).map(|id| battle::NativeCommandRecord { id, ..Default::default() }).collect();
    book[29] = battle::NativeCommandRecord {
        id: 29,
        selection_mode: 5,
        effect_mode: 2,
        mp_cost: 26,
        target_code: 0,
        ..Default::default()
    };
    let mut st = battle::State::default();
    (st.w, st.h) = (3, 1);
    st.units = vec![actor, target(1, 12, 20, 210, 0, 1), target(2, 13, 25, 220, 3, 2)];
    (st.native_composition_event_bytes, st.native_command_book) = (vec![0; 3], book);
    let mut g = Game {
        m: Some(MapData {
            w: 3,
            h: 1,
            tiles: vec![0, 0, 0],
            native_terrain_control: vec![0, 0, 4, 0],
            ..Default::default()
        }),
        st: Some(st),
        rng: Some(battle::GoMathRand::new(29)),
        native_ui_palette: load_native_ui_palette(),
        ..Default::default()
    };
    g.locale_id = String::from("zh-Hant");
    g.locale_catalog = load_official_locale("zh-Hant").expect("locale");
    g.locale_content = load_official_locale_content("zh-Hant").expect("content");
    g.locale_entities = load_official_locale_entities("zh-Hant").expect("entities");
    Some((g, 0, [1, 2]))
}

pub(crate) fn unit(g: &Game, index: UnitIdx) -> &battle::Unit {
    &g.st.as_ref().unwrap().units[index]
}

pub(crate) fn unit_mut(g: &mut Game, index: UnitIdx) -> &mut battle::Unit {
    &mut g.st.as_mut().unwrap().units[index]
}

pub(crate) fn units(g: &Game) -> Vec<battle::Unit> {
    g.st.as_ref().unwrap().units.clone()
}

fn job(g: &Game) -> &NativeCommand29PresentationJob {
    g.native_cmd29_presentation.as_ref().unwrap()
}

/// 畫一格再步進一格；畫不出來就失敗。
pub(crate) fn draw_step(g: &mut Game, screen: &mut Picture) {
    assert!(g.present_native_command29(Some(screen)), "draw failed: {}", g.load_err);
    g.step_native_command29_presentation();
}

/// Go：`native_command29_presentation_test.go:13` `TestNativeCommand29FormalSelfCenterThroughConfirm`
#[test]
fn test_native_command29_formal_self_center_through_confirm() {
    child!("test_native_command29_formal_self_center_through_confirm");
    for side in [0u8, 1] {
        let Some((mut g, actor, targets)) = command29_test_game() else { return };
        let Some(book) = spells() else { return };
        let a = unit_mut(&mut g, actor);
        a.has_native_record_byte5 = true;
        (a.native_record_byte6, a.native_record_byte8) = (side, 2);
        let st = g.st.as_mut().unwrap();
        st.native_command_book = book;
        st.has_native_map_view_state = true;
        st.native_tile_blit_modes = vec![0; 3];
        (g.sel, g.cur_x, g.cur_y) = (Some(actor), 0, 0);
        g.begin_native_command_target_selection(29).unwrap();
        let st = g.st.as_ref().unwrap();
        assert!(st.native_map_range_mode == 1 && st.native_tile_blit_modes[1] != 0xff, "side{side} effect field");
        g.cur_x = 1; // Confirmation may move; the effect list stays actor-centered.
        g.confirm();
        let j = g.native_cmd29_presentation.as_ref().unwrap_or_else(|| panic!("side{side} 沒進演出：{}", g.msg));
        assert_eq!(j.targets.len(), 2);
        let results = &j.plan.as_ref().unwrap().results;
        assert!(results[0].target == Some(targets[0]) && results[1].target == Some(targets[1]));
        g.step_native_command29_presentation();
        assert_eq!(
            (unit(&g, actor).mp, unit(&g, actor).acted, unit(&g, targets[0]).hp, unit(&g, targets[1]).hp),
            (40, false, 210, 220),
            "confirm or an undrawn frame published the transaction"
        );
        let mut screen = Picture::new_image(640, 400);
        for _ in 0..1000 {
            if g.native_cmd29_presentation.is_none() {
                break;
            }
            draw_step(&mut g, &mut screen);
        }
        let a = unit(&g, actor);
        assert!(g.native_cmd29_presentation.is_none() && a.mp == 14 && a.acted, "side{side} err={}", g.load_err);
        assert!(unit(&g, targets[0]).hp < 210 && unit(&g, targets[1]).hp < 220);
        assert!(!g.native_command0_targeting && g.sel.is_none());
        assert_eq!(g.st.as_ref().unwrap().native_map_range_mode, 1);
    }
}

/// Go：`native_command29_presentation_test.go:63` `TestNativeCommand29MissingNativeSourcesRejectsBeforeDamageRNG`
#[test]
fn test_native_command29_missing_native_sources_rejects_before_damage_rng() {
    child!("test_native_command29_missing_native_sources_rejects_before_damage_rng");
    let Some((mut g, actor, targets)) = command29_test_game() else { return };
    let Some(book) = spells() else { return };
    g.st.as_mut().unwrap().native_command_book = book;
    unit_mut(&mut g, actor).has_native_record_byte5 = true;
    unit_mut(&mut g, targets[1]).battle_fig = 99999;
    assert!(g.start_native_command29_presentation_at_cursor(actor, battle::Cell { x: 0, y: 0 }, noop()).is_err());
    assert!(g.native_cmd29_presentation.is_none());
    assert_eq!(
        (unit(&g, actor).mp, unit(&g, actor).acted, unit(&g, targets[0]).hp, unit(&g, targets[1]).hp),
        (40, false, 210, 220)
    );
    assert_eq!(g.rng.as_mut().unwrap().int63(), battle::GoMathRand::new(29).int63(), "damage RNG consumed");
}

/// Go：`native_command29_presentation_test.go:121` `TestNativeCommand29PresentationPublishesMultipleTargetsInOrder`
#[test]
fn test_native_command29_presentation_publishes_multiple_targets_in_order() {
    child!("test_native_command29_presentation_publishes_multiple_targets_in_order");
    let Some((mut g, actor, targets)) = command29_test_game() else { return };
    let results = std::sync::Arc::new(std::sync::Mutex::new(None));
    let sink = results.clone();
    g.start_native_command29_presentation(
        actor,
        Some(targets[0]),
        Box::new(move |_, got| *sink.lock().unwrap() = Some(got.len())),
    )
    .unwrap();
    let j = job(&g);
    assert!(j.targets.len() == 2 && j.targets.iter().all(|t| t.transition_frames.len() == 20));
    let mut screen = Picture::new_image(640, 400);
    let (mut seen_first, mut seen_second) = (false, false);
    for _ in 0..1000 {
        if g.native_cmd29_presentation.is_none() {
            break;
        }
        draw_step(&mut g, &mut screen);
        seen_first |= unit(&g, targets[0]).hp < 210;
        if unit(&g, targets[1]).hp < 220 {
            seen_second = true;
            assert!(seen_first, "second target published before first target");
        }
        if seen_first && !seen_second {
            assert_eq!(unit(&g, actor).mp, 14, "actor MP was not published exactly once");
        }
    }
    assert!(g.native_cmd29_presentation.is_none() && seen_first && seen_second);
    assert!(unit(&g, actor).acted && unit(&g, actor).mp == 14);
    assert_eq!(*results.lock().unwrap(), Some(2));
}

/// Go：`native_command29_presentation_test.go:162` `TestNativeCommand29PresentationRollsBackAllTargetsOnLateFailure`
#[test]
fn test_native_command29_presentation_rolls_back_all_targets_on_late_failure() {
    child!("test_native_command29_presentation_rolls_back_all_targets_on_late_failure");
    let Some((mut g, actor, targets)) = command29_test_game() else { return };
    g.start_native_command29_presentation(actor, Some(targets[0]), noop()).unwrap();
    let mut screen = Picture::new_image(640, 400);
    for _ in 0..600 {
        if g.native_cmd29_presentation.as_ref().is_none_or(|j| j.targets[0].damage_published) {
            break;
        }
        draw_step(&mut g, &mut screen);
    }
    assert!(job(&g).targets[0].damage_published, "first target marker was not reached");
    unit_mut(&mut g, targets[1]).hp -= 1;
    for _ in 0..600 {
        if g.native_cmd29_presentation.is_none() {
            break;
        }
        draw_step(&mut g, &mut screen);
    }
    assert!(g.native_cmd29_presentation.is_none() && !g.load_err.is_empty());
    assert_eq!(
        (unit(&g, actor).mp, unit(&g, actor).acted, unit(&g, targets[0]).hp, unit(&g, targets[1]).hp),
        (40, false, 210, 220),
        "late failure was not atomic"
    );
}

/// Go：`native_command29_presentation_test.go:189` `TestNativeCommand29PresentationRejectsUnprovenSelectorWithoutMutation`
#[test]
fn test_native_command29_presentation_rejects_unproven_selector_without_mutation() {
    child!("test_native_command29_presentation_rejects_unproven_selector_without_mutation");
    let Some((mut g, actor, targets)) = command29_test_game() else { return };
    unit_mut(&mut g, actor).battle_fig = 33;
    let before = units(&g);
    assert!(g.start_native_command29_presentation(actor, Some(targets[0]), noop()).is_err());
    assert!(g.native_cmd29_presentation.is_none() && units(&g) == before);
}

/// 兩個敵人的受控指令 30 場景。
///
/// Go：`native_command30_presentation_test.go:15` `newNativeCommand30PresentationTestGame`
fn command30_test_game() -> Option<(Game, UnitIdx, [UnitIdx; 2])> {
    let (mut g, actor, targets) = command29_test_game()?;
    let book = spells()?;
    let a = unit_mut(&mut g, actor);
    (a.battle_fig, a.native_record_byte8) = (18, 18);
    (a.lv, a.mp, a.max_mp) = (4, 29, 29);
    a.native_record_byte6 = 2;
    a.has_native_record_byte5 = true;
    g.st.as_mut().unwrap().native_command_book = book;
    g.rng = Some(battle::GoMathRand::new(30));
    Some((g, actor, targets))
}

/// Go：`native_command30_presentation_test.go:31` `TestNativeCommand30PresentationMatchesStateControlAndDrawMarkers`
#[test]
fn test_native_command30_presentation_matches_state_control_and_draw_markers() {
    child!("test_native_command30_presentation_matches_state_control_and_draw_markers");
    let Some((mut g, actor, targets)) = command30_test_game() else { return };
    let Some((mut control, ca, _)) = command30_test_game() else { return };
    let (saved, confirmed) = (battle::Cell { x: 0, y: 0 }, battle::Cell { x: 1, y: 0 });
    let rng = control.rng.as_mut().map(|rng| rng as &mut dyn battle::GoRand);
    let want = control.st.as_mut().unwrap().execute_native_command30(Some(ca), saved, confirmed, rng).unwrap();
    let got = std::sync::Arc::new(std::sync::Mutex::new(None));
    let sink = got.clone();
    g.start_native_command30_presentation_at_cursor(
        actor,
        saved,
        confirmed,
        Box::new(move |_, results| *sink.lock().unwrap() = Some(results)),
    )
    .unwrap();
    let j = job(&g);
    assert_eq!(j.plan.as_ref().unwrap().command_id, 30);
    assert!(j.prelude_frames.len() == 9 && j.effect_frames.len() == 14 && j.targets.len() == 2);
    assert_eq!(j.targets[1].transition_frames.len(), 20);
    let cues = j.actor_cues.as_ref().unwrap();
    assert!(cues.len() == 2 && cues.contains_key(&2), "early non-impact sound cue was lost");
    for _ in 0..3 {
        g.step_native_command29_presentation();
    }
    assert!(job(&g).prelude_frame == 0 && unit(&g, actor).mp == 29 && !unit(&g, actor).acted, "advanced without Draw");
    let mut screen = Picture::new_image(640, 400);
    let mut seen = [false; 2];
    for _ in 0..1000 {
        let Some(j) = g.native_cmd29_presentation.as_ref() else { break };
        let (prelude, transition, frame, index) = (j.prelude_frame, j.transition_frame, j.frame, j.target_index);
        let mp = unit(&g, actor).mp;
        let hp = [unit(&g, targets[0]).hp, unit(&g, targets[1]).hp];
        assert!(g.present_native_command29(Some(&mut screen)) && g.present_native_command29(Some(&mut screen)));
        assert!(
            unit(&g, actor).mp == mp && [unit(&g, targets[0]).hp, unit(&g, targets[1]).hp] == hp,
            "Draw mutated rules"
        );
        g.step_native_command29_presentation();
        if unit(&g, actor).mp != mp {
            assert!(prelude == 9 && transition < 0 && frame == 4 && unit(&g, actor).mp == 5, "MP outside marker4");
        }
        for i in 0..2 {
            if unit(&g, targets[i]).hp != hp[i] {
                assert!(prelude == 9 && transition < 0 && frame == 11 && index == i as i64 && !seen[i]);
                assert!(i == 0 || seen[0], "damage order differs");
                seen[i] = true;
            }
        }
    }
    assert!(g.native_cmd29_presentation.is_none() && seen == [true, true]);
    let got = got.lock().unwrap().take().expect("continuation did not run once");
    assert_eq!(got.len(), want.len());
    for (i, (got, want)) in got.iter().zip(&want).enumerate() {
        assert!(got.target == Some(targets[i]) && got.amount == want.amount && got.damage == want.damage);
    }
    assert_eq!(units(&g), units(&control), "final transaction differs from state-only control");
    assert_eq!(g.rng.as_mut().unwrap().int63(), control.rng.as_mut().unwrap().int63());
}

/// Go：`native_command30_presentation_test.go:97` `TestNativeCommand30PresentationColdRejectAndLateRollback`
#[test]
fn test_native_command30_presentation_cold_reject_and_late_rollback() {
    child!("test_native_command30_presentation_cold_reject_and_late_rollback");
    for field in ["actor-selector", "second-target-record", "second-target-selector", "MP"] {
        let Some((mut g, actor, targets)) = command30_test_game() else { return };
        match field {
            "actor-selector" => unit_mut(&mut g, actor).battle_fig = 19,
            "second-target-record" => unit_mut(&mut g, targets[1]).has_native_record_byte8 = false,
            "second-target-selector" => unit_mut(&mut g, targets[1]).battle_fig = 4096,
            _ => unit_mut(&mut g, actor).mp = 23,
        }
        let before = units(&g);
        let (saved, confirmed) = (battle::Cell { x: 0, y: 0 }, battle::Cell { x: 1, y: 0 });
        assert!(g.start_native_command30_presentation_at_cursor(actor, saved, confirmed, noop()).is_err(), "{field}");
        assert!(g.native_cmd29_presentation.is_none() && units(&g) == before, "{field}");
        assert_eq!(g.rng.as_mut().unwrap().int63(), battle::GoMathRand::new(30).int63(), "{field} consumed RNG");
    }
    let Some((mut g, actor, targets)) = command30_test_game() else { return };
    unit_mut(&mut g, actor).ap = 1000;
    unit_mut(&mut g, targets[0]).hp = 5;
    let before = units(&g);
    let (done, then) = counter();
    g.start_native_command30_presentation_at_cursor(
        actor,
        battle::Cell { x: 0, y: 0 },
        battle::Cell { x: 1, y: 0 },
        then,
    )
    .unwrap();
    let mut screen = Picture::new_image(640, 400);
    for _ in 0..600 {
        if g.native_cmd29_presentation.as_ref().is_none_or(|j| j.targets[0].damage_published) {
            break;
        }
        draw_step(&mut g, &mut screen);
    }
    assert!(job(&g).targets[0].damage_published, "first marker not reached");
    unit_mut(&mut g, targets[1]).hp -= 1;
    for _ in 0..600 {
        if g.native_cmd29_presentation.is_none() {
            break;
        }
        draw_step(&mut g, &mut screen);
    }
    assert!(g.native_cmd29_presentation.is_none() && !g.load_err.is_empty());
    assert_eq!(done.load(std::sync::atomic::Ordering::SeqCst), 0);
    assert_eq!(units(&g), before, "late failure did not restore actor and every target");
}

/// Go：`native_command30_presentation_test.go:145` `TestNativeCommand30FixedSourceScheduleIncludesEarlyCue`
#[test]
fn test_native_command30_fixed_source_schedule_includes_early_cue() {
    if command30_test_game().is_none() {
        return;
    }
    let a = figani::load_separated_resource(&separated_asset_path("animations"), 56).unwrap();
    let s = figani::build_native_command_derived_strike_schedule(Some(&a), 30).unwrap();
    assert_eq!(
        (
            s.target_start,
            s.actor_impact_frame,
            s.target_impact_frame,
            s.audio_resource,
            s.actor_sample,
            s.target_sample
        ),
        (9, 4, 11, 49, 4, 3)
    );
    let cues: Vec<(i64, i64)> = s.actor_cues.iter().map(|cue| (cue.frame, cue.sample)).collect();
    assert_eq!(cues, [(2, 2), (4, 4)]);
}

/// Go：`native_command30_presentation_test.go:157` `TestNativeCommand30PlayerTargetModalSavesCursorAndCancels`
#[test]
fn test_native_command30_player_target_modal_saves_cursor_and_cancels() {
    child!("test_native_command30_player_target_modal_saves_cursor_and_cancels");
    let Some((mut g, actor, targets)) = command30_test_game() else { return };
    (g.sel, g.cur_x, g.cur_y) = (Some(actor), 0, 0);
    g.st.as_mut().unwrap().native_tile_blit_modes = vec![0; 3];
    let before = units(&g);
    g.begin_native_command_target_selection(30).unwrap();
    assert!(g.native_command0_targeting && g.native_command30_saved_cursor == Some(battle::Cell { x: 0, y: 0 }));
    g.cur_x = 1;
    g.cancel_native_command_target_selection();
    assert!(!g.native_command0_targeting && g.native_command_open && g.native_command30_saved_cursor.is_none());
    assert_eq!(units(&g), before);
    // Re-enter from the original actor cell, then move the confirmed cursor.
    // Using the current cursor as the line origin would omit target0.
    g.cur_x = 0;
    g.begin_native_command_target_selection(30).unwrap();
    g.cur_x = 1;
    g.confirm();
    let j =
        g.native_cmd29_presentation.as_ref().unwrap_or_else(|| panic!("formal confirm lost saved origin: {}", g.msg));
    assert_eq!(j.plan.as_ref().unwrap().command_id, 30);
    assert!(j.targets.len() == 2 && j.targets[0].unit == Some(targets[0]) && j.targets[1].unit == Some(targets[1]));
    assert_eq!(units(&g), before);
}
