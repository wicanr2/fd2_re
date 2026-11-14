//! 移植自 Go `battlepresent/native_command_compound_test.go`。

use super::*;
use crate::battlepresent::test_util::{anim, px1, surface};
use fd2_formats::figani;

fn native_compound_test_frame(value: u8) -> Frame {
    px1(0, 0, value, 1)
}

#[test]
fn test_build_native_compound_common_frames_preserves_layer_order() {
    let effect =
        anim(vec![native_compound_test_frame(10), native_compound_test_frame(11), native_compound_test_frame(12)]);
    let schedule = figani::build_native_compound_presentation_schedule(32, Some(&effect)).unwrap();
    let frames =
        build_native_compound_common_frames(&surface(), &native_compound_test_frame(7), Some(&effect), schedule)
            .unwrap();
    assert!(
        frames.actor_slide.len() == 8
            && frames.actor_slide[0][0] == 7
            && frames.actor_slide[7][140] == 7
            && frames.effect_slide.len() == 9
            && frames.effect_slide[0][240] == 10
            && frames.effect_slide[8][0] == 10
            && frames.main.len() == 2
            && frames.main[0][0] == 11
            && frames.main[1][0] == 12
            && frames.tail.len() == 11
            && frames.tail[0][0] == 11
            && frames.tail[1][0] == 12
            && frames.tail[10][0] == 11,
        "unexpected compound frame schedule"
    );
}

#[test]
fn test_build_native_compound_common_frames_overlay_is_command_specific() {
    let mut second = native_compound_test_frame(11);
    second.x = 1;
    let effect = anim(vec![native_compound_test_frame(10), second]);
    let base = vec![3u8; 320 * 200];
    let s32 = figani::build_native_compound_presentation_schedule(32, Some(&effect)).unwrap_or_default();
    let s33 = figani::build_native_compound_presentation_schedule(33, Some(&effect)).unwrap_or_default();
    let f32 = build_native_compound_common_frames(&base, &native_compound_test_frame(7), Some(&effect), s32).unwrap();
    let f33 = build_native_compound_common_frames(&base, &native_compound_test_frame(7), Some(&effect), s33).unwrap();
    assert!(
        f32.main[0][0] == 3 && f33.main[0][0] == 10 && f32.main[0][1] == 11 && f33.main[0][1] == 11,
        "command-specific steady overlay corrupted main frame"
    );
}

#[test]
fn test_build_native_compound_common_frames_rejects_without_partial_output() {
    let effect = anim(vec![native_compound_test_frame(1)]);
    let schedule = NativeCompoundPresentationSchedule { command_id: 32, effect_resource: 65, ..Default::default() };
    assert!(
        build_native_compound_common_frames(&surface(), &native_compound_test_frame(2), Some(&effect), schedule)
            .is_err(),
        "malformed compound effect was accepted"
    );
}

#[test]
fn test_build_native_compound_actor_frames_preserves_delay_and_layer_order() {
    let mut actor = native_compound_test_frame(7);
    actor.delay = 2;
    let target = native_compound_test_frame(9);
    let actor_animation = Animation { header_byte4: 2, frames: vec![actor], ..Default::default() };
    let target_animation = anim(vec![target]);
    let base = surface();
    let right = build_native_compound_actor_frames(&base, Some(&actor_animation), Some(&target_animation), 0).unwrap();
    let left = build_native_compound_actor_frames(&base, Some(&actor_animation), Some(&target_animation), 1).unwrap();
    assert!(
        right.len() == 2 && right[0][0] == 7 && left[0][0] == 9,
        "compound actor right={} left={}",
        right[0][0],
        left[0][0]
    );
}

#[test]
fn test_build_native_compound_actor_frames_rejects_mp_marker_branch() {
    let actor = Animation { header_byte4: 1, frames: vec![native_compound_test_frame(1)], ..Default::default() };
    let target = anim(vec![native_compound_test_frame(2)]);
    assert!(
        build_native_compound_actor_frames(&surface(), Some(&actor), Some(&target), 0).is_err(),
        "compound actor accepted unimplemented MP marker branch"
    );
}
