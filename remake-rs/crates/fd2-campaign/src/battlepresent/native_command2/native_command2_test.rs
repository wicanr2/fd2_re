//! 移植自 Go `battlepresent/native_command2_test.go`。

use super::*;
use crate::battlepresent::test_util::{anim, px1, surface};
use fd2_formats::figani;

fn effect18(x: i64, y: i64, value: impl Fn(usize) -> u8) -> Animation {
    Animation { frames: (0..18).map(|i| px1(x, y, value(i), 0)).collect(), header_byte2: 18, ..Default::default() }
}

#[test]
fn test_compose_native_command2_target_frame_preserves_layer_order() {
    let effect = effect18(-1, 1, |i| i as u8 + 1);
    let schedule = figani::build_native_command2_presentation_schedule(1, Some(&effect)).unwrap();
    let target = px1(0, 0, 99, 0);
    let got = compose_native_command2_target_frame(&surface(), &target, Some(&effect), schedule, 1, 16).unwrap();
    // mode5 state16 is the final draw at the same pixel.
    assert_eq!(got[0], 17, "composed pixel want state16 value17");
}

#[test]
fn test_compose_native_command2_target_frame_rejects_partial_bank() {
    let mut effect = effect18(0, 0, |_| 1);
    let schedule = figani::build_native_command2_presentation_schedule(1, Some(&effect)).unwrap_or_default();
    effect.frames.truncate(17);
    assert!(
        compose_native_command2_target_frame(&surface(), &px1(0, 0, 1, 0), Some(&effect), schedule, 1, 16).is_err(),
        "partial command2 bank accepted"
    );
    let effect = Animation { frames: vec![Frame::default(); 18], header_byte2: 18, ..Default::default() };
    assert!(
        compose_native_command2_target_frame(&surface(), &Frame::default(), Some(&effect), schedule, 0, 16).is_err(),
        "raw side accepted the opposite resource schedule"
    );
}

#[test]
fn test_compose_native_command2_orbit_preserves_side_layer_order() {
    let effect = effect18(0, 0, |i| i as u8 + 1);
    let actor = anim(vec![px1(0, 0, 70, 0)]);
    let target = anim(vec![px1(0, 0, 80, 0)]);
    let helper = NativeCommand2HelperFrame { effect_frame: 7, ..Default::default() };
    let nonzero =
        compose_native_command2_orbit_frame(&surface(), Some(&actor), Some(&target), Some(&effect), helper, 1);
    assert!(matches!(&nonzero, Ok(p) if p[0] == 80), "nonzero {nonzero:?}");
    let zero = compose_native_command2_orbit_frame(&surface(), Some(&actor), Some(&target), Some(&effect), helper, 0);
    assert!(matches!(&zero, Ok(p) if p[0] == 8), "zero {:?}", zero.as_ref().map(|p| p[0]));
}
