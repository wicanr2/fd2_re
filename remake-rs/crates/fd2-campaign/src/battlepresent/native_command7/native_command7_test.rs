//! 移植自 Go `battlepresent/native_command7_test.go`。

use super::*;
use crate::battlepresent::test_util::{anim, px1, surface};

pub(in crate::battlepresent) fn command7_compositor_effect() -> Animation {
    Animation {
        frames: (0..NATIVE_COMMAND7_EFFECT_FRAME_COUNT).map(|i| px1(0, 0, i as u8 + 1, 0)).collect(),
        header_byte2: NATIVE_COMMAND7_EFFECT_FRAME_COUNT as u8,
        ..Default::default()
    }
}

#[test]
fn test_compose_native_command7_target_draws_effect_after_target() {
    let target = px1(30, 0, 99, 0);
    let frame =
        NativeCommand7Frame { layers: vec![NativeCommand7Layer { channel: 0, frame: 4, x: 30 }], ..Default::default() };
    let got =
        compose_native_command7_target_frame(&surface(), &target, Some(&command7_compositor_effect()), &frame).unwrap();
    assert_eq!(got[30], 5, "pixel want effect frame4 after target");
}

#[test]
fn test_compose_native_command7_orbit_preserves_layer_order() {
    let actor = anim(vec![px1(1, 0, 7, 0)]);
    let target = anim(vec![px1(2, 0, 9, 0)]);
    let frame = NativeCommand7Frame {
        layers: vec![NativeCommand7Layer { frame: 0, x: 2, ..Default::default() }],
        ..Default::default()
    };
    let got = compose_native_command7_orbit_frame(
        &surface(),
        Some(&actor),
        Some(&target),
        Some(&command7_compositor_effect()),
        &frame,
    )
    .unwrap();
    assert!(got[1] == 7 && got[2] == 1, "actor/effect pixels={},{}", got[1], got[2]);
}

#[test]
fn test_compose_native_command7_transition_uses_selected_offset() {
    let actor = anim(vec![px1(0, 0, 7, 0)]);
    let target = anim(vec![px1(0, 0, 9, 0)]);
    let got = compose_native_command7_transition_frame(
        &surface(),
        Some(&actor),
        Some(&target),
        Some(&command7_compositor_effect()),
        &NativeCommand7TransitionFrame { target_offset_x: 4, ..Default::default() },
    );
    assert!(matches!(&got, Ok(p) if p[4] == 9), "target pixel err={:?}", got.as_ref().err());
}

#[test]
fn test_compose_native_command7_rejects_invalid_layer() {
    let frame = NativeCommand7Frame {
        layers: vec![NativeCommand7Layer { frame: NATIVE_COMMAND7_EFFECT_FRAME_COUNT, ..Default::default() }],
        ..Default::default()
    };
    assert!(
        compose_native_command7_target_frame(
            &surface(),
            &Frame::default(),
            Some(&command7_compositor_effect()),
            &frame
        )
        .is_err(),
        "out-of-range command7 layer accepted"
    );
}
