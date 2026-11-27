//! Go 來源：`remake/cmd/fd2/native_command28_presentation_test.go`。

use super::*;
use crate::game::native_command29_presentation::native_command29_presentation_test::{
    child, command29_test_game, draw_step, noop, spells, unit, unit_mut,
};

/// Go：`native_command28_presentation_test.go:12` `newNativeCommand28PresentationTestGame`
fn command28_test_game() -> Option<(Game, UnitIdx, [UnitIdx; 2])> {
    let (mut g, actor, targets) = command29_test_game()?;
    g.st.as_mut().unwrap().native_command_book = spells()?;
    let a = unit_mut(&mut g, actor);
    (a.battle_fig, a.native_record_byte8, a.has_native_record_byte5) = (19, 19, true);
    let t = unit_mut(&mut g, targets[0]);
    (t.hp, t.max_hp) = (900, 900);
    Some((g, actor, targets))
}

/// Go：`native_command28_presentation_test.go:26` `TestNativeCommand28FormalConfirmPresentsEightImpactsAndReturns`
#[test]
fn test_native_command28_formal_confirm_presents_eight_impacts_and_returns() {
    child!("test_native_command28_formal_confirm_presents_eight_impacts_and_returns");
    for side in [0u8, 2] {
        let Some((mut g, actor, targets)) = command28_test_game() else { return };
        unit_mut(&mut g, actor).native_record_byte6 = side;
        let st = g.st.as_mut().unwrap();
        (st.has_native_map_view_state, st.native_tile_blit_modes) = (true, vec![0; 3]);
        (g.sel, g.cur_x, g.cur_y) = (Some(actor), 1, 0);
        g.begin_native_command_target_selection(28).unwrap();
        g.confirm();
        let j = g.native_cmd29_presentation.as_ref().unwrap_or_else(|| panic!("side{side}: {}", g.msg));
        let sequence = j.command28.as_ref().expect("command28 sequence");
        assert_eq!(j.plan.as_ref().unwrap().command_id, 28);
        assert!(j.targets.len() == 1 && sequence.frames.len() == 49, "frames={}", sequence.frames.len());
        let damage = j.plan.as_ref().unwrap().results[0].damage;
        g.step_native_command29_presentation();
        assert!(unit(&g, actor).mp == 40 && !unit(&g, actor).acted && unit(&g, targets[0]).hp == 900);
        let (mut mp_markers, mut target_markers, mut prelude_frames) = (0, 0, 0);
        let mut screen = Picture::new_image(640, 400);
        for _ in 0..100 {
            let Some(j) = g.native_cmd29_presentation.as_ref() else { break };
            let sequence = j.command28.as_ref().unwrap();
            let event = sequence.events[sequence.index as usize];
            prelude_frames += i64::from(event.prelude < 9);
            draw_step(&mut g, &mut screen);
            mp_markers += i64::from(event.mp);
            if event.impact > 0 {
                target_markers += 1;
                assert_eq!(event.impact, target_markers, "impact order");
                assert_eq!(unit(&g, targets[0]).hp, 900 - damage * target_markers / 8, "cumulative HP");
            }
        }
        assert!(g.native_cmd29_presentation.is_none(), "side{side} err={}", g.load_err);
        assert_eq!((mp_markers, target_markers, prelude_frames), (1, 8, 9));
        assert!(unit(&g, actor).mp == 18 && unit(&g, actor).acted);
        assert!(!g.native_command0_targeting && g.sel.is_none());
        assert!(g.st.as_ref().unwrap().native_map_range_mode == 1 && unit(&g, targets[1]).hp == 220);
    }
}

/// Go：`native_command28_presentation_test.go:77` `TestNativeCommand28LateImpactFailureRestoresWholeTransaction`
#[test]
fn test_native_command28_late_impact_failure_restores_whole_transaction() {
    child!("test_native_command28_late_impact_failure_restores_whole_transaction");
    let Some((mut g, actor, targets)) = command28_test_game() else { return };
    g.start_native_command28_presentation(actor, Some(targets[0]), noop()).unwrap();
    let mut screen = Picture::new_image(640, 400);
    for _ in 0..60 {
        if g.native_cmd29_presentation.is_none() || unit(&g, targets[0]).hp != 900 {
            break;
        }
        g.present_native_command29(Some(&mut screen));
        g.step_native_command29_presentation();
    }
    assert!(unit(&g, targets[0]).hp != 900 && g.native_cmd29_presentation.is_some(), "first impact not presented");
    unit_mut(&mut g, targets[0]).hp -= 1;
    for _ in 0..60 {
        if g.native_cmd29_presentation.is_none() {
            break;
        }
        g.present_native_command29(Some(&mut screen));
        g.step_native_command29_presentation();
    }
    let (a, t) = (unit(&g, actor), unit(&g, targets[0]));
    assert!(g.native_cmd29_presentation.is_none() && !g.load_err.is_empty());
    assert!(a.mp == 40 && !a.acted && t.hp == 900 && t.native_record_byte5 == 0);
}

/// Go：`native_command28_presentation_test.go:100` `TestNativeCommand28MissingTerrainRejectsBeforeDamageRNG`
#[test]
fn test_native_command28_missing_terrain_rejects_before_damage_rng() {
    child!("test_native_command28_missing_terrain_rejects_before_damage_rng");
    let Some((mut g, actor, targets)) = command28_test_game() else { return };
    g.m.as_mut().unwrap().native_terrain_control.clear();
    assert!(g.start_native_command28_presentation(actor, Some(targets[0]), noop()).is_err());
    assert!(g.native_cmd29_presentation.is_none());
    assert!(unit(&g, actor).mp == 40 && !unit(&g, actor).acted && unit(&g, targets[0]).hp == 900);
    assert_eq!(g.rng.as_mut().unwrap().int63(), battle::GoMathRand::new(29).int63());
}
