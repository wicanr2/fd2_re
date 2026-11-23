//! Go 來源：`remake/cmd/fd2/native_church_ui_test.go`（R4 E 組，期望值不改）。
//!
//! 未移植：`:15`（截圖狀態，`r8`）、`:157`（`drawNativeChurchMenu`，R5）、`:190`（`handleNativeChurchMenuInput`，`r8`）。

use super::*;
use std::sync::atomic::{AtomicBool, Ordering};

#[test]
fn chapter_town_church_services_and_save() {
    let Ok(path) = std::env::var("FD2_CHURCH_PROBE_SAVE") else { return };
    let name = "chapter_town_church_services_and_save";
    if f_child_env(module_path!(), name, |dir| {
        vec![
            ("XDG_DATA_HOME", dir.into()),
            ("FD2_TITLE", "1".into()),
            ("FD2_MUTE", "1".into()),
            ("FD2_CAMPAIGN", "canonical".into()),
            ("FD2_SEED", "4".into()),
            ("FD2_NATIVE_SAVE", "".into()),
        ]
    })
    .is_none()
    {
        return;
    }
    let mut g = load_game().unwrap();
    assert!(g.load_err.is_empty(), "{}", g.load_err);
    g.load_native_game_from_slot(&path, 0).unwrap();
    g.title_phase.clear();
    let town = g.camp.as_ref().unwrap().node_id().to_string();
    assert!(matches!(town.as_str(), "town_ch02" | "town_ch03"), "{town}");
    let mut screen = Picture::new_image(640, 400);
    let mut clock = 1_000_000_000;
    let mut advance = |g: &mut Game, key: Key| {
        g.tick_input = TickInput { pressed: [key].into(), ..Default::default() };
        assert!(g.camp_input());
        for _ in 0..600 {
            if g.native_church_ui_job.is_none() && g.native_class_ui_job.is_none() {
                return;
            }
            assert!(g.draw_native_church_ui_job(&mut screen) || g.draw_native_class_ui_job(&mut screen));
            clock += 16_666_667;
            g.step_native_church_ui_lifecycle(GameInstant(clock));
            g.step_native_class_ui_lifecycle(GameInstant(clock));
        }
        panic!("教會轉場沒有結束");
    };
    g.camp_sel = 4;
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "menu", "{}", g.msg);
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "status_roster", "{}", g.msg);
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "status_view", "{}", g.msg);
    assert!(!g.church_status_panel.is_empty());
    advance(&mut g, Key::Escape);
    if g.church_mode == "status_commands" {
        advance(&mut g, Key::Escape);
    }
    assert_eq!(g.church_mode, "status_roster");
    advance(&mut g, Key::Escape);

    let funds = g.gold;
    advance(&mut g, Key::ArrowRight);
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "transfer_source", "{}", g.msg);
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "transfer_item", "{}", g.msg);
    let source = g.church_transfer_source;
    let item = g.party_roster[&source].inventory[g.church_transfer_items[0] as usize];
    let count = g.party_roster[&source].inventory.len();
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "transfer_dest", "{}", g.msg);
    advance(&mut g, Key::ArrowRight);
    let dest = g.church_ids[g.church_sel as usize];
    assert_ne!(source, dest);
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "transfer_source", "{}", g.msg);
    assert_eq!(g.party_roster[&source].inventory.len(), count - 1);
    assert_eq!(g.party_roster[&dest].inventory.last(), Some(&item));
    assert_eq!(g.gold, funds);
    advance(&mut g, Key::Escape);

    advance(&mut g, Key::ArrowRight);
    advance(&mut g, Key::ArrowRight);
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "revive", "{}", g.msg);
    let fallen = g.church_ids[0];
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "revive_confirm", "{}", g.msg);
    let fee = g.church_revive_fee;
    advance(&mut g, Key::Enter);
    assert_eq!(g.gold, funds - fee, "{}", g.msg);
    assert_eq!(g.party_roster[&fallen].hp, g.party_roster[&fallen].max_hp);
    assert!(!campaign::can_revive(g.party_roster.get(&fallen)));
    advance(&mut g, Key::Escape);
    assert_eq!(g.church_mode, "menu");

    // The chapter-boundary party normally arrives below promotion level.
    // Set the required level to exercise the original promotion transaction here.
    let trainee = g.party_join_order[0];
    let unit = g.party_roster.get_mut(&trainee).unwrap();
    unit.lv = 20;
    let (target, valid) = campaign::native_class_change_target(Some(unit), &g.class_change_table);
    assert!(valid);
    advance(&mut g, Key::ArrowLeft);
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "class", "{}", g.msg);
    advance(&mut g, Key::Enter);
    assert_eq!(g.church_mode, "class_confirm", "{}", g.msg);
    advance(&mut g, Key::Enter);
    assert_eq!(g.party_roster[&trainee].portrait, target.portrait, "{}", g.msg);
    assert_eq!(g.gold, funds - fee);
    advance(&mut g, Key::Escape);
    advance(&mut g, Key::Escape);
    assert_eq!(g.camp.as_ref().unwrap().node_id(), town);
    g.save_game_to_slot(0).unwrap();
    let save = g.build_save_data();
    assert!(g.load_json_game_from_slot(0, false));
    assert_eq!(g.build_save_data().gold, save.gold);
    assert_eq!(g.build_save_data().party_roster, save.party_roster);
}

fn frames(n: u8) -> Vec<Vec<u8>> {
    (0..n).map(|i| vec![i]).collect()
}

/// Go：`native_church_ui_test.go:84` `TestNativeChurchUILifecyclePresentsOpeningAndClosingRestore`
#[test]
fn test_native_church_ui_lifecycle_presents_opening_and_closing_restore() {
    let mut g = Game {
        native_church_ui_job: Some(NativeChurchUIJob { frames: frames(4), ..Default::default() }),
        ..Default::default()
    };
    for want in 0..4 {
        let job = g.native_church_ui_job.as_mut().unwrap_or_else(|| panic!("opening present{want} has no job"));
        assert_eq!(job.frame, want, "opening present{want}");
        job.drawn = true;
        g.step_native_church_ui_lifecycle(GameInstant::default());
    }
    assert!(g.native_church_ui_job.is_none(), "opening did not settle after four acknowledged presents");

    let completed = Arc::new(AtomicBool::new(false));
    let set = completed.clone();
    g.native_church_ui_job = Some(NativeChurchUIJob {
        frames: frames(4),
        restore: vec![0u8; 320 * 200],
        after: Some(Box::new(move |_: &mut Game| set.store(true, Ordering::SeqCst))),
        ..Default::default()
    });
    for want in 0..4 {
        let job = g.native_church_ui_job.as_mut().unwrap();
        assert!(!completed.load(Ordering::SeqCst) && job.frame == want, "closing present{want}");
        job.drawn = true;
        g.step_native_church_ui_lifecycle(GameInstant::default());
    }
    assert!(
        !completed.load(Ordering::SeqCst) && g.native_church_ui_job.as_ref().is_some_and(|j| j.frame == 4),
        "closing ran callback before source restore"
    );
    g.native_church_ui_job.as_mut().unwrap().drawn = true;
    g.step_native_church_ui_lifecycle(GameInstant::default());
    assert!(completed.load(Ordering::SeqCst) && g.native_church_ui_job.is_none(), "restore did not hand off");
}

/// Go：`native_church_ui_test.go:124` `TestNativeChurchUILifecycleCannotSkipUndrawnFrame`
#[test]
fn test_native_church_ui_lifecycle_cannot_skip_undrawn_frame() {
    let mut g = Game {
        native_church_ui_job: Some(NativeChurchUIJob { frames: frames(4), ..Default::default() }),
        ..Default::default()
    };
    g.step_native_church_ui_lifecycle(GameInstant::default());
    g.step_native_church_ui_lifecycle(GameInstant::default());
    assert_eq!(g.native_church_ui_job.as_ref().unwrap().frame, 0, "undrawn frame advanced");
}

/// Go：`native_church_ui_test.go:136` `TestNativeChurchUIPulseStartsSelectedAndUsesTwoTicks`
#[test]
fn test_native_church_ui_pulse_starts_selected_and_uses_two_ticks() {
    let mut g = Game::default();
    g.reset_native_church_ui_pulse();
    assert_eq!(g.native_church_ui_pulse / 2, 1, "initial selected variant");
    g.step_native_church_ui_pulse_tick(10);
    g.step_native_church_ui_pulse_tick(11);
    assert_eq!(g.native_church_ui_pulse, 2, "one-tick delta advanced pulse");
    g.step_native_church_ui_pulse_tick(12);
    assert!(
        g.native_church_ui_pulse == 3 && g.native_church_ui_pulse / 2 == 1,
        "two-tick pulse={}",
        g.native_church_ui_pulse
    );
    g.step_native_church_ui_pulse_tick(14);
    assert!(
        g.native_church_ui_pulse == 0 && g.native_church_ui_pulse / 2 == 0,
        "wrapped pulse={}",
        g.native_church_ui_pulse
    );
}
