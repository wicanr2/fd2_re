//! 移植自 Go `battlepresent/native_command8_test.go`。

use super::*;
use crate::battlepresent::test_util::{anim, px1, surface};

pub(in crate::battlepresent) fn command8_compositor_effect() -> Animation {
    Animation {
        frames: (0..NATIVE_COMMAND8_EFFECT_FRAME_COUNT).map(|i| px1(0, 0, i as u8 + 1, 0)).collect(),
        header_byte2: NATIVE_COMMAND8_EFFECT_FRAME_COUNT as u8,
        ..Default::default()
    }
}

#[test]
fn test_compose_native_command8_target_draws_effect_after_target() {
    let target = px1(30, 10, 99, 0);
    let frame =
        NativeCommand8Frame { layers: vec![NativeCommand8Layer { channel: 0, frame: 8 }], ..Default::default() };
    let got =
        compose_native_command8_target_frame(&surface(), &target, Some(&command8_compositor_effect()), &frame).unwrap();
    assert_eq!(got[0], 9, "pixel want effect frame8 after target");
}

#[test]
fn test_compose_native_command8_transition_uses_selected_offset() {
    let actor = anim(vec![px1(0, 0, 7, 0)]);
    let target = anim(vec![px1(0, 0, 9, 0)]);
    let got = compose_native_command8_transition_frame(
        &surface(),
        Some(&actor),
        Some(&target),
        Some(&command8_compositor_effect()),
        &NativeCommand8TransitionFrame { target_offset_x: 4, ..Default::default() },
    );
    assert!(matches!(&got, Ok(p) if p[4] == 9), "target pixel err={:?}", got.as_ref().err());
}

#[test]
fn test_compose_native_command8_rejects_invalid_layer() {
    let frame = NativeCommand8Frame {
        layers: vec![NativeCommand8Layer { frame: NATIVE_COMMAND8_EFFECT_FRAME_COUNT, ..Default::default() }],
        ..Default::default()
    };
    assert!(
        compose_native_command8_target_frame(
            &surface(),
            &Frame::default(),
            Some(&command8_compositor_effect()),
            &frame
        )
        .is_err(),
        "out-of-range command8 layer accepted"
    );
}
