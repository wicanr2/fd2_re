//! 移植自 Go `battlepresent/native_command3_test.go`。

use super::*;
use crate::battlepresent::test_util::{anim, px1, surface};
use fd2_formats::figani::{NATIVE_COMMAND3_EFFECT_FRAME_COUNT, NativeCommand3Frame, NativeCommand3Layer};

pub(in crate::battlepresent) fn command3_compositor_effect() -> Animation {
    Animation {
        frames: (0..NATIVE_COMMAND3_EFFECT_FRAME_COUNT).map(|i| px1(0, 0, i as u8 + 1, 0)).collect(),
        header_byte2: NATIVE_COMMAND3_EFFECT_FRAME_COUNT as u8,
        ..Default::default()
    }
}

#[test]
fn test_compose_native_command3_target_draws_effect_after_target() {
    let target = px1(30, 10, 99, 0);
    let frame = NativeCommand3Frame {
        layers: vec![NativeCommand3Layer { channel: 0, frame: 22, x: 30, y: 10 }],
        ..Default::default()
    };
    let got =
        compose_native_command3_target_frame(&surface(), &target, Some(&command3_compositor_effect()), &frame).unwrap();
    assert_eq!(got[10 * 320 + 30], 23, "pixel want effect frame22 after target");
}

#[test]
fn test_compose_native_command3_transition_uses_selected_offset() {
    let actor = anim(vec![px1(0, 0, 7, 0)]);
    let target = anim(vec![px1(0, 0, 9, 0)]);
    let got = compose_native_command3_transition_frame(
        &surface(),
        Some(&actor),
        Some(&target),
        Some(&command3_compositor_effect()),
        &NativeCommand3TransitionFrame { target_offset_x: 4, ..Default::default() },
    );
    assert!(matches!(&got, Ok(p) if p[4] == 9), "target pixel err={:?}", got.as_ref().err());
}

#[test]
fn test_compose_native_command3_rejects_invalid_layer() {
    let frame = NativeCommand3Frame {
        layers: vec![NativeCommand3Layer { frame: NATIVE_COMMAND3_EFFECT_FRAME_COUNT, ..Default::default() }],
        ..Default::default()
    };
    assert!(
        compose_native_command3_target_frame(
            &surface(),
            &Frame::default(),
            Some(&command3_compositor_effect()),
            &frame
        )
        .is_err(),
        "out-of-range command3 layer accepted"
    );
}
