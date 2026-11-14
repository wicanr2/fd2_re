//! 移植自 Go `battlepresent/native_command5_test.go`。

use super::*;
use crate::battlepresent::test_util::{anim, px1, surface};

pub(in crate::battlepresent) fn command5_compositor_effect() -> Animation {
    Animation {
        frames: (0..NATIVE_COMMAND5_EFFECT_FRAME_COUNT).map(|i| px1(0, 0, i as u8 + 1, 0)).collect(),
        header_byte2: NATIVE_COMMAND5_EFFECT_FRAME_COUNT as u8,
        ..Default::default()
    }
}

#[test]
fn test_compose_native_command5_target_draws_effect_after_target() {
    let target = px1(30, 0, 99, 0);
    let frame =
        NativeCommand5Frame { layers: vec![NativeCommand5Layer { channel: 0, frame: 5, x: 30 }], ..Default::default() };
    let got =
        compose_native_command5_target_frame(&surface(), &target, Some(&command5_compositor_effect()), &frame).unwrap();
    assert_eq!(got[30], 6, "pixel want effect frame5 after target");
}

#[test]
fn test_compose_native_command5_orbit_preserves_actor_target_effect_order() {
    let actor = anim(vec![px1(1, 0, 7, 0)]);
    let target = anim(vec![px1(2, 0, 9, 0)]);
    let frame = NativeCommand5Frame {
        layers: vec![NativeCommand5Layer { frame: 0, x: 2, ..Default::default() }],
        ..Default::default()
    };
    let got = compose_native_command5_orbit_frame(
        &surface(),
        Some(&actor),
        Some(&target),
        Some(&command5_compositor_effect()),
        &frame,
    )
    .unwrap();
    assert!(got[1] == 7 && got[2] == 1, "actor/effect pixels={},{}", got[1], got[2]);
}

#[test]
fn test_compose_native_command5_transition_uses_selected_offset() {
    let actor = anim(vec![px1(0, 0, 7, 0)]);
    let target = anim(vec![px1(0, 0, 9, 0)]);
    let got = compose_native_command5_transition_frame(
        &surface(),
        Some(&actor),
        Some(&target),
        Some(&command5_compositor_effect()),
        &NativeCommand5TransitionFrame { target_offset_x: 4, ..Default::default() },
    );
    assert!(matches!(&got, Ok(p) if p[4] == 9), "target pixel err={:?}", got.as_ref().err());
}

#[test]
fn test_compose_native_command5_rejects_invalid_layer() {
    let frame = NativeCommand5Frame {
        layers: vec![NativeCommand5Layer { frame: NATIVE_COMMAND5_EFFECT_FRAME_COUNT, ..Default::default() }],
        ..Default::default()
    };
    assert!(
        compose_native_command5_target_frame(
            &surface(),
            &Frame::default(),
            Some(&command5_compositor_effect()),
            &frame
        )
        .is_err(),
        "out-of-range command5 layer accepted"
    );
}
