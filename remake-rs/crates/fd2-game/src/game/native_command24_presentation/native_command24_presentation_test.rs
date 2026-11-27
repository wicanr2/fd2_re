//! Go 來源：`remake/cmd/fd2/native_command24_presentation_test.go`。
//!
//! Go 另以 `FD2_ORIGINAL_*` 把 FIGANI 指到不存在的檔，證明只讀分離素材包；Rust 的指令演出本來就只讀分離素材包，
//! 不再設這些變數。HD 面板（Go `maybeAttachCommandPanelHDTest`）屬 R6，不在這裡。

use super::*;
use crate::test_support;

/// Go：`native_command24_presentation_test.go:12` `newNativeCommand24PresentationTestGame`
fn command24_test_game() -> Option<(Game, UnitIdx, UnitIdx)> {
    for resource in ["FIGANI_000", "FIGANI_096", "FIGANI_098"] {
        test_support::input("FD2_ASSET_PACK", &format!("animations/{resource}/animation.json"))?;
    }
    let mut actor = battle::Unit::default();
    (actor.camp, actor.on_field, actor.lv, actor.ap) = (battle::OWN, true, 4, 100);
    (actor.mp, actor.max_mp, actor.hp, actor.max_hp) = (30, 30, 120, 120);
    (actor.battle_fig, actor.has_battle_fig) = (32, true);
    (actor.has_native_record_byte6, actor.has_native_record_byte8) = (true, true);
    let mut target = battle::Unit::default();
    (target.camp, target.on_field, target.x, target.lv, target.dp) = (battle::ENEMY, true, 1, 5, 20);
    (target.hp, target.max_hp, target.has_battle_fig) = (200, 200, true);
    (target.native_record_byte6, target.has_native_record_byte6) = (1, true);
    (target.native_record_byte8, target.has_native_record_byte8) = (1, true);
    let mut book: Vec<battle::NativeCommandRecord> =
        (0..36).map(|id| battle::NativeCommandRecord { id, ..Default::default() }).collect();
    book[24] = battle::NativeCommandRecord {
        id: 24,
        selection_mode: 5,
        effect_mode: 1,
        mp_cost: 22,
        target_code: 0,
        ..Default::default()
    };
    let mut st = battle::State::default();
    (st.w, st.h, st.units) = (2, 1, vec![actor, target]);
    (st.native_composition_event_bytes, st.native_command_book) = (vec![0; 2], book);
    let mut g = Game {
        m: Some(MapData {
            w: 2,
            h: 1,
            tiles: vec![0, 0],
            native_terrain_control: vec![0, 0, 4, 0],
            ..Default::default()
        }),
        st: Some(st),
        rng: Some(battle::GoMathRand::new(2)),
        native_ui_palette: load_native_ui_palette(),
        ..Default::default()
    };
    g.locale_id = String::from("zh-Hant");
    g.locale_catalog = load_official_locale("zh-Hant").expect("locale");
    g.locale_content = load_official_locale_content("zh-Hant").expect("content");
    g.locale_entities = load_official_locale_entities("zh-Hant").expect("entities");
    Some((g, 0, 1))
}

fn unit(g: &Game, index: UnitIdx) -> &battle::Unit {
    &g.st.as_ref().unwrap().units[index]
}

/// (actor MP, actor acted, target HP)
fn snapshot(g: &Game, actor: UnitIdx, target: UnitIdx) -> (i64, bool, i64) {
    (unit(g, actor).mp, unit(g, actor).acted, unit(g, target).hp)
}

fn job(g: &Game) -> &NativeCommand24PresentationJob {
    g.native_cmd24_presentation.as_ref().unwrap()
}

fn noop() -> Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send> {
    Box::new(|_, _| {})
}

macro_rules! child {
    ($name:literal) => {
        if f_child_env(module_path!(), $name, |_| vec![("FD2_MUTE", "1".into())]).is_none() {
            return;
        }
    };
}

/// Go：`native_command24_presentation_test.go:55` `TestNativeCommand24PresentationPublishesAtRawMarkers`
#[test]
fn test_native_command24_presentation_publishes_at_raw_markers() {
    child!("test_native_command24_presentation_publishes_at_raw_markers");
    let Some((mut g, actor, target)) = command24_test_game() else { return };
    let done = std::sync::Arc::new(std::sync::atomic::AtomicI64::new(0));
    let counter = done.clone();
    g.start_native_command24_presentation(
        actor,
        Some(target),
        Box::new(move |_, _| {
            counter.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
        }),
    )
    .unwrap();
    assert_eq!(snapshot(&g, actor, target), (30, false, 200), "start crossed mutation boundary");
    let mut screen = Picture::new_image(640, 400);
    while g.native_cmd24_presentation.as_ref().is_some_and(|j| j.frame < 4) {
        assert!(g.present_native_command24(Some(&mut screen)), "actor prelude draw failed");
        g.step_native_command24_presentation();
        assert_eq!(snapshot(&g, actor, target), (30, false, 200), "mutation before actor marker");
    }
    assert!(g.present_native_command24(Some(&mut screen)), "actor marker draw failed");
    g.step_native_command24_presentation();
    assert_eq!(snapshot(&g, actor, target), (8, false, 200), "actor marker state");
    while g.native_cmd24_presentation.as_ref().is_some_and(|j| j.frame < 10) {
        assert!(g.present_native_command24(Some(&mut screen)), "target prelude draw failed");
        g.step_native_command24_presentation();
        let (_, acted, hp) = snapshot(&g, actor, target);
        assert!(hp == 200 && !acted, "damage before target marker");
    }
    assert!(g.present_native_command24(Some(&mut screen)), "target marker draw failed");
    g.step_native_command24_presentation();
    let (_, acted, hp) = snapshot(&g, actor, target);
    assert!(hp < 200 && !acted, "target marker state hp={hp} acted={acted}");
    for _ in 0..128 {
        if g.native_cmd24_presentation.is_none() {
            break;
        }
        assert!(g.present_native_command24(Some(&mut screen)), "tail draw failed");
        g.step_native_command24_presentation();
    }
    assert!(g.native_cmd24_presentation.is_none() && unit(&g, actor).acted, "presentation did not complete");
    assert_eq!(done.load(std::sync::atomic::Ordering::SeqCst), 1);
}

/// Go：`native_command24_presentation_test.go:108` `TestNativeCommand24PresentationDrawsNine29164FramesBeforeActorEffect`
#[test]
fn test_native_command24_presentation_draws_nine29164_frames_before_actor_effect() {
    child!("test_native_command24_presentation_draws_nine29164_frames_before_actor_effect");
    let Some((mut g, actor, target)) = command24_test_game() else { return };
    g.start_native_command24_presentation(actor, Some(target), noop()).unwrap();
    assert!(job(&g).prelude_frames.len() == 9 && job(&g).prelude_frame == 0 && job(&g).frame == 0);
    let mut screen = Picture::new_image(640, 400);
    for frame in 0..9 {
        assert!(job(&g).prelude_frame == frame && g.present_native_command24(Some(&mut screen)), "frame {frame}");
        g.step_native_command24_presentation();
        assert_eq!(snapshot(&g, actor, target), (30, false, 200), "0x29164 frame {frame} crossed gameplay boundary");
        assert_eq!(job(&g).frame, 0);
    }
    assert!(job(&g).prelude_frame == 9 && job(&g).frame == 0 && g.present_native_command24(Some(&mut screen)));
}

/// Go：`native_command24_presentation_test.go:132` `TestNativeCommand24PresentationRejectsUnprovenActorWithoutMutation`
#[test]
fn test_native_command24_presentation_rejects_unproven_actor_without_mutation() {
    child!("test_native_command24_presentation_rejects_unproven_actor_without_mutation");
    let Some((mut g, actor, target)) = command24_test_game() else { return };
    g.st.as_mut().unwrap().units[actor].battle_fig = 4;
    assert!(g.start_native_command24_presentation(actor, Some(target), noop()).is_err());
    assert_eq!(snapshot(&g, actor, target), (30, false, 200));
    assert!(g.native_cmd24_presentation.is_none());
}

/// Go：`native_command24_presentation_test.go:143` `TestNativeCommand24PresentationRunsTwentyBackgroundDrawsBeforeDamage`
#[test]
fn test_native_command24_presentation_runs_twenty_background_draws_before_damage() {
    child!("test_native_command24_presentation_runs_twenty_background_draws_before_damage");
    let Some((mut g, actor, target)) = command24_test_game() else { return };
    g.start_native_command24_presentation(actor, Some(target), noop()).unwrap();
    let mut screen = Picture::new_image(640, 400);
    for _ in 0..128 {
        if job(&g).transition_frame >= 0 {
            break;
        }
        assert!(g.present_native_command24(Some(&mut screen)), "pre-transition draw failed");
        g.step_native_command24_presentation();
    }
    assert_eq!(job(&g).transition_frame, 0);
    assert_eq!(snapshot(&g, actor, target), (8, false, 200));
    for frame in 0..20 {
        assert!(job(&g).transition_frame == frame && g.present_native_command24(Some(&mut screen)), "frame {frame}");
        g.step_native_command24_presentation();
        let (_, acted, hp) = snapshot(&g, actor, target);
        assert!(hp == 200 && !acted, "transition frame {frame} crossed damage boundary");
    }
    let j = job(&g);
    assert_eq!((j.transition_frame, j.frame, j.target_frame, j.target_repeat), (-1, 9, 0, 0));
}

/// Go：`native_command24_presentation_test.go:173` `TestNativeCommand24PresentationRejectsMissingTerrainControlWithoutMutation`
#[test]
fn test_native_command24_presentation_rejects_missing_terrain_control_without_mutation() {
    child!("test_native_command24_presentation_rejects_missing_terrain_control_without_mutation");
    let Some((mut g, actor, target)) = command24_test_game() else { return };
    g.m.as_mut().unwrap().native_terrain_control.clear();
    assert!(g.start_native_command24_presentation(actor, Some(target), noop()).is_err());
    assert_eq!(snapshot(&g, actor, target), (30, false, 200));
    assert!(g.native_cmd24_presentation.is_none());
}

/// Go：`native_command24_presentation_test.go:184` `TestNativeCommand24PresentationRejectsMissingPanelRawNameWithoutMutation`
#[test]
fn test_native_command24_presentation_rejects_missing_panel_raw_name_without_mutation() {
    child!("test_native_command24_presentation_rejects_missing_panel_raw_name_without_mutation");
    let Some((mut g, actor, target)) = command24_test_game() else { return };
    g.st.as_mut().unwrap().units[target].has_native_record_byte8 = false;
    assert!(g.start_native_command24_presentation(actor, Some(target), noop()).is_err());
    assert_eq!(snapshot(&g, actor, target), (30, false, 200));
    assert!(g.native_cmd24_presentation.is_none());
}

/// Go：`native_command24_presentation_test.go:195` `TestNativeCommand24PresentationUsesSeparatedActorTAIForNonzeroRawSide`
#[test]
fn test_native_command24_presentation_uses_separated_actor_tai_for_nonzero_raw_side() {
    child!("test_native_command24_presentation_uses_separated_actor_tai_for_nonzero_raw_side");
    let Some((mut g, actor, target)) = command24_test_game() else { return };
    g.st.as_mut().unwrap().units[actor].native_record_byte6 = 1;
    g.start_native_command24_presentation(actor, Some(target), noop()).unwrap();
}

/// Go：`native_command24_presentation_test.go:204` `TestNativeCommand24PresentationDoesNotRequireLegacyRGBABases`。
/// Rust 沒有 Go 的 `g.bg`／`g.tai`／`g.panel` RGBA 底圖欄位；預設 `Game` 本來就沒有它們，建立仍須成功。
#[test]
fn test_native_command24_presentation_does_not_require_legacy_rgba_bases() {
    child!("test_native_command24_presentation_does_not_require_legacy_rgba_bases");
    let Some((mut g, actor, target)) = command24_test_game() else { return };
    g.start_native_command24_presentation(actor, Some(target), noop()).unwrap();
}

/// 玩家入口：`start_native_player_command(24, …)` 演完後扣 MP、結算傷害、清選取，訊息照 Go `main.go:7853`。
#[test]
fn test_player_native_command24_settles_through_presentation() {
    child!("test_player_native_command24_settles_through_presentation");
    let Some((mut g, actor, target)) = command24_test_game() else { return };
    (g.sel, g.cur_x, g.cur_y) = (Some(actor), 1, 0);
    assert!(g.start_native_player_command(24, Some(target)));
    assert!(g.native_cmd24_presentation.is_some(), "msg={} err={}", g.msg, g.load_err);
    let mut screen = Picture::new_image(640, 400);
    for _ in 0..256 {
        if g.native_cmd24_presentation.is_none() {
            break;
        }
        assert!(g.present_native_command24(Some(&mut screen)));
        g.step_native_command24_presentation();
    }
    let (mp, acted, hp) = snapshot(&g, actor, target);
    assert!(g.native_cmd24_presentation.is_none() && acted && mp == 8 && hp < 200, "mp={mp} hp={hp}");
    assert_eq!(g.msg, format!("原始指令 24：傷害 {}", 200 - hp));
    assert!(g.sel.is_none());
}
