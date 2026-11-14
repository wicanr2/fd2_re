//! Go 來源：`remake/cmd/fd2/native_class_ui_lifecycle_test.go`（7 個測試，全部移植）。
//!
//! Go 以 `drawn = true` 模擬 `Draw` 的呈現回報，Rust 照寫（120「畫過才推進」）。延續的 `completed`
//! 旗標用 `Arc<AtomicBool>`，因為延續要 `Send + 'static`。

use super::*;
use std::sync::atomic::{AtomicBool, Ordering};

fn frames(n: u8) -> Vec<Vec<u8>> {
    (0..n).map(|i| vec![i]).collect()
}

fn completion() -> (Arc<AtomicBool>, Option<Box<dyn FnOnce(&mut Game) + Send>>) {
    let flag = Arc::new(AtomicBool::new(false));
    let set = flag.clone();
    (flag, Some(Box::new(move |_: &mut Game| set.store(true, Ordering::SeqCst))))
}

/// Go：`native_class_ui_lifecycle_test.go:8` `TestNativeClassUILifecycleRequiresDrawAcknowledgment`
#[test]
fn test_native_class_ui_lifecycle_requires_draw_acknowledgment() {
    let mut g = Game {
        native_class_ui_job: Some(NativeClassUIJob { frames: frames(6), ..Default::default() }),
        ..Default::default()
    };
    g.step_native_class_ui_lifecycle(GameInstant::default());
    g.step_native_class_ui_lifecycle(GameInstant::default());
    assert_eq!(g.native_class_ui_job.as_ref().unwrap().frame, 0, "undrawn opening frame advanced");
    for want in 0..6 {
        let job = g.native_class_ui_job.as_mut().unwrap_or_else(|| panic!("present {want} has no job"));
        assert_eq!(job.frame, want, "present {want}");
        job.drawn = true;
        g.step_native_class_ui_lifecycle(GameInstant::default());
    }
    assert!(g.native_class_ui_job.is_none(), "six-frame list opening did not settle");
}

/// Go：`native_class_ui_lifecycle_test.go:30` `TestNativeClassUIClosingDefersContinuationUntilFourthPresent`
#[test]
fn test_native_class_ui_closing_defers_continuation_until_fourth_present() {
    let (completed, after) = completion();
    let mut g = Game {
        native_class_ui_job: Some(NativeClassUIJob { frames: frames(4), after, ..Default::default() }),
        ..Default::default()
    };
    for want in 0..4 {
        let job = g.native_class_ui_job.as_mut().unwrap();
        assert!(!completed.load(Ordering::SeqCst) && job.frame == want, "before present {want}: frame={}", job.frame);
        job.drawn = true;
        g.step_native_class_ui_lifecycle(GameInstant::default());
    }
    assert!(completed.load(Ordering::SeqCst) && g.native_class_ui_job.is_none(), "closing completion");
}

/// Go：`native_class_ui_lifecycle_test.go:48` `TestNativeClassUIClosingPresentsRestoreBeforeContinuation`
#[test]
fn test_native_class_ui_closing_presents_restore_before_continuation() {
    let (completed, after) = completion();
    let mut g = Game {
        native_class_ui_job: Some(NativeClassUIJob {
            frames: frames(2),
            restore: vec![0u8; 320 * 200],
            after,
            ..Default::default()
        }),
        ..Default::default()
    };
    for want in 0..2 {
        let job = g.native_class_ui_job.as_mut().unwrap();
        assert!(!completed.load(Ordering::SeqCst) && job.frame == want, "before frame {want}: frame={}", job.frame);
        job.drawn = true;
        g.step_native_class_ui_lifecycle(GameInstant::default());
    }
    assert!(
        !completed.load(Ordering::SeqCst) && g.native_class_ui_job.as_ref().is_some_and(|j| j.frame == 2),
        "continuation ran before restore"
    );
    g.native_class_ui_job.as_mut().unwrap().drawn = true;
    g.step_native_class_ui_lifecycle(GameInstant::default());
    assert!(completed.load(Ordering::SeqCst) && g.native_class_ui_job.is_none(), "restore completion");
}

/// Go：`native_class_ui_lifecycle_test.go:72` `TestNativeClassUIPulseUsesTwoBIOSTickCadenceAndWrap`
#[test]
fn test_native_class_ui_pulse_uses_two_bios_tick_cadence_and_wrap() {
    let mut g = Game::default();
    for (tick, want) in [(0x7ffd, 0), (0x7ffe, 0), (0x7fff, 1), (-0x8000, 2), (-0x7fff, 2), (-0x7ffe, 3), (-0x7ffc, 0)]
    {
        g.step_native_class_ui_pulse_tick(tick);
        assert_eq!(g.native_class_ui_pulse, want, "tick {tick:#x}");
    }
}

/// Go：`native_class_ui_lifecycle_test.go:90` `TestNativeClassUIPulseMatchesSignedBIOSBranch`
#[test]
fn test_native_class_ui_pulse_matches_signed_bios_branch() {
    for (name, previous, current, pulse, want) in [
        ("same", 9, 9, 2, 2),
        ("one", 9, 10, 2, 2),
        ("two", 9, 11, 2, 3),
        ("large", 9, 100, 2, 3),
        ("backward", 9, 8, 2, 3),
        ("signed-crossing", 0x7fff, 0x8000, 3, 0),
        ("unsigned-wrap-one", -1, 0, 3, 3),
        ("unsigned-wrap-two", -1, 1, 3, 0),
    ] {
        let mut g = Game {
            native_class_ui_pulse: pulse,
            native_class_ui_last_tick: previous,
            native_class_ui_has_tick: true,
            ..Default::default()
        };
        g.step_native_class_ui_pulse_tick(current);
        assert_eq!(g.native_class_ui_pulse, want, "{name}: pulse");
        let want_tick = if want != pulse { current as u16 as i16 as i64 } else { previous };
        assert_eq!(g.native_class_ui_last_tick, want_tick, "{name}: latch");
    }
}

/// Go：`native_class_ui_lifecycle_test.go:121` `TestNativeClassUIPulseUsesSecondBIOSReadForLatch`
#[test]
fn test_native_class_ui_pulse_uses_second_bios_read_for_latch() {
    let mut g = Game { native_class_ui_has_tick: true, native_class_ui_last_tick: 7, ..Default::default() };
    g.step_native_class_ui_pulse_ticks(9, 10);
    assert!(g.native_class_ui_pulse == 1 && g.native_class_ui_last_tick == 10, "second read was not saved");
    g.step_native_class_ui_pulse_ticks(11, 12);
    assert!(g.native_class_ui_pulse == 1 && g.native_class_ui_last_tick == 10, "waiting branch published a new latch");
    g.step_native_class_ui_pulse_ticks(12, 13);
    assert!(
        g.native_class_ui_pulse == 2 && g.native_class_ui_last_tick == 13,
        "second update did not preserve separate samples"
    );
}

/// Go：`native_class_ui_lifecycle_test.go:137` `TestNativeClassUITimelineRequiresFinalFramePresentation`
#[test]
fn test_native_class_ui_timeline_requires_final_frame_presentation() {
    let (completed, after) = completion();
    let start = GameInstant(100 * 1_000_000_000);
    let mut g = Game {
        native_class_ui_job: Some(NativeClassUIJob {
            timeline: vec![
                NativeClassUITimelineStep { frame: vec![1], duration: GoDuration(10_000_000), ..Default::default() },
                NativeClassUITimelineStep { frame: vec![2], ..Default::default() },
            ],
            after,
            ..Default::default()
        }),
        ..Default::default()
    };
    g.step_native_class_ui_lifecycle(start);
    g.native_class_ui_job.as_mut().unwrap().drawn = true;
    g.step_native_class_ui_lifecycle(start + GoDuration(20_000_000));
    assert!(
        !completed.load(Ordering::SeqCst) && g.native_class_ui_job.is_some(),
        "timeline completed before its zero-duration final frame was presented"
    );
    let job = g.native_class_ui_job.as_mut().unwrap();
    job.frame = 1;
    job.drawn = true;
    g.step_native_class_ui_lifecycle(start + GoDuration(20_000_000));
    assert!(completed.load(Ordering::SeqCst) && g.native_class_ui_job.is_none(), "timeline completion");
}
