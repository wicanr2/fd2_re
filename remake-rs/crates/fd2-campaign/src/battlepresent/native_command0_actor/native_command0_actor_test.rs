//! 移植自 Go `battlepresent/native_command0_actor_test.go`。

use super::*;
use crate::battlepresent::test_util::{anim, fig, raw_frame, surface};

/// Go `command0ActorTestInput` 的素材；輸入結構借用素材，所以另外保存。
struct Assets {
    base_before: Vec<u8>,
    base_after: Vec<u8>,
    effect: Animation,
    target: Animation,
    lut: Vec<u8>,
}

fn assets() -> Assets {
    let mut base_after = surface();
    base_after[1] = 6;
    let mut marked = fig(10, 10, 1, 1, vec![3], vec![1], 6);
    marked.raw_byte4 = 1;
    let effect = anim(vec![fig(10, 10, 1, 1, vec![2], vec![1], 1), marked]);
    let target = anim(vec![fig(10, 10, 1, 1, vec![8], vec![1], 2), fig(10, 10, 1, 1, vec![9], vec![1], 2)]);
    let mut lut: Vec<u8> = (0..=255).collect();
    (lut[4], lut[5]) = (14, 15);
    Assets { base_before: surface(), base_after, effect, target, lut }
}

fn command0_actor_test_input(a: &Assets, side: u8) -> NativeCommand0ActorInput<'_> {
    NativeCommand0ActorInput {
        base_before: &a.base_before,
        base_after: &a.base_after,
        actor_effect: Some(&a.effect),
        first_target_idle: Some(&a.target),
        raw_side: side,
        lut: &a.lut,
        background: raw_frame(0, 0, 1, 1, vec![1, 0, 1, 0, 0, 4]),
        platform: raw_frame(0, 0, 1, 1, vec![1, 0, 1, 0, 0, 5]),
    }
}

#[test]
fn test_build_native_command0_actor_frames_publishes_once_and_pulses_six() {
    let a = assets();
    let frames = build_native_command0_actor_frames(&command0_actor_test_input(&a, 1)).unwrap();
    assert!(
        frames.len() == 7 && !frames[0].publish_mp && frames[1].publish_mp,
        "actor frames/marker={} {:?} {:?}",
        frames.len(),
        frames[0].publish_mp,
        frames[1].publish_mp
    );
    for (i, frame) in frames.iter().enumerate() {
        let want_pulse = i >= 1;
        assert_eq!(frame.pulse, want_pulse, "frame {i} pulse");
    }
    assert!(
        frames[0].pixels[1] == 0
            && frames[1].pixels[1] == 6
            && frames[1].pixels[50 * 320] == 14
            && frames[1].pixels[157 * 320 + 164] == 15,
        "before/after LUT boundary not preserved"
    );
}

#[test]
fn test_build_native_command0_actor_frames_preserves_raw_side_layer_order() {
    let a = assets();
    let left = build_native_command0_actor_frames(&command0_actor_test_input(&a, 1)).unwrap();
    let right = build_native_command0_actor_frames(&command0_actor_test_input(&a, 0)).unwrap();
    assert!(
        left[0].pixels[10 * 320 + 10] == 8 && right[0].pixels[10 * 320 + 10] == 2,
        "raw-side order left={} right={}",
        left[0].pixels[10 * 320 + 10],
        right[0].pixels[10 * 320 + 10]
    );
}

#[test]
fn test_build_native_command0_actor_frames_fails_without_unique_visible_mp_marker() {
    let mut a = assets();
    a.effect.frames[1].raw_byte4 = 0;
    let r = build_native_command0_actor_frames(&command0_actor_test_input(&a, 1));
    assert!(r.is_err(), "missing marker accepted {r:?}");
    let mut a = assets();
    a.effect.frames[1].delay = 0;
    let r = build_native_command0_actor_frames(&command0_actor_test_input(&a, 1));
    assert!(r.is_err(), "invisible marker accepted {r:?}");
}
