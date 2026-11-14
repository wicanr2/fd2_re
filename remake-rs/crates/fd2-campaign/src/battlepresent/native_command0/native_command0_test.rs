//! 移植自 Go `battlepresent/native_command0_test.go`。

use super::*;
use crate::battlepresent::test_util::{fig, px1, surface};

fn effect16(value: impl Fn(usize) -> u8) -> Animation {
    let mut effect =
        Animation { frames: (0..16).map(|i| px1(0, 0, value(i), 0)).collect(), header_byte2: 16, ..Default::default() };
    effect.frames[9].raw_byte4 = 1;
    effect
}

#[test]
fn test_compose_native_command0_target_frame_preserves_back_target_front_order() {
    let effect = effect16(|i| 10 + i as u8);
    let mut schedule = figani::build_native_command0_presentation_schedule(1, Some(&effect)).unwrap();
    // Collapse all active layers at step 12 onto one pixel. Back layers 2/4
    // are covered by the target; front layers 0/1/3/5/6 then cover it in
    // native order, so layer6's frame0 is the final value 10.
    schedule.x_offsets = [0; 7];
    schedule.y_offsets = [0; 7];
    let target = px1(0, 0, 99, 0);
    let got = compose_native_command0_target_frame(&surface(), &target, Some(&effect), schedule, 12).unwrap();
    assert_eq!(got[0], 10, "composed pixel want layer6 frame0 value10");
}

#[test]
fn test_compose_native_command0_target_frame_uses_side_shift_and_rejects_partial_publish() {
    let effect = effect16(|_| 7);
    let schedule = figani::build_native_command0_presentation_schedule(0, Some(&effect)).unwrap();
    let target = px1(1, 1, 9, 0);
    let base = surface();
    let got = compose_native_command0_target_frame(&base, &target, Some(&effect), schedule, 0).unwrap();
    assert!(got[40 + 148] == 7 && got[1 + 320] == 9, "side-shift effect={} target={}", got[40 + 148], got[1 + 320]);
    let mut bad = effect.clone();
    bad.frames.truncate(15);
    assert!(
        compose_native_command0_target_frame(&base, &target, Some(&bad), schedule, 0).is_err(),
        "partial effect bank accepted"
    );
}

#[test]
fn test_compose_native_command0_target_frame_allows_native_work_margin() {
    let base = surface();
    let mut effect = effect16(|_| 7);
    effect.frames[9].raw_byte4 = 0;
    effect.frames[0] = fig(0, 0, 1, 12, vec![9; 12], vec![1; 12], 0);
    let schedule = NativeCommand0PresentationSchedule {
        frames: 28,
        layer_flags: [0; 7],
        y_offsets: [-10, 0, 0, 0, 0, 0, 0],
        ..Default::default()
    };
    let target = px1(1, 1, 7, 0);
    let got = compose_native_command0_target_frame(&base, &target, Some(&effect), schedule, 0).unwrap();
    assert!(
        got[0] == 9 && got[320] == 9 && got[321] == 7,
        "native work-margin pixels top={} next={} target={}",
        got[0],
        got[320],
        got[321]
    );
}
