//! Go 來源：`remake/cmd/fd2/native_church_ui_test.go`（R4 E 組，期望值不改）。
//!
//! 未移植：`:15`（截圖狀態，`r8`）、`:157`（`drawNativeChurchMenu`，R5）、`:190`（`handleNativeChurchMenuInput`，`r8`）。

use super::*;
use std::sync::atomic::{AtomicBool, Ordering};

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
