//! Go `native_command9_test.go`。

use super::*;
use crate::archive::read_nested_resource;

fn command9_animation(count: usize) -> Animation {
    Animation { frames: vec![Frame::default(); count], header_byte2: count as u8, ..Animation::default() }
}

// Go: TestNativeCommand9AIScheduleFailsClosedByRawSideAndShape
#[test]
fn native_command9_ai_schedule_fails_closed_by_raw_side_and_shape() {
    assert!(
        build_native_command9_ai_schedule(1, Some(&command9_animation(31))).is_err(),
        "unproven nonzero raw side was accepted"
    );
    assert!(build_native_command9_ai_schedule(0, Some(&command9_animation(30))).is_err(), "short effect was accepted");
}

// Go: TestNativeCommand9AITargetMarkersAndSamples
#[test]
fn native_command9_ai_target_markers_and_samples() {
    let s = build_native_command9_ai_schedule(0, Some(&command9_animation(31))).unwrap();
    assert_eq!(s.slide_in_hold_ms, 500, "slide-in hold");
    let (frames, next) = build_native_command9_ai_target_sequence(new_native_command9_ai_state(), s).unwrap();
    let (mut markers, mut stages) = (0, 0);
    for (index, frame) in frames.iter().enumerate() {
        if frame.numeric_marker {
            markers += 1;
        }
        if frame.hp_stage != 0 {
            stages += 1;
        }
        assert!(frame.play_sub1 == (index == 5) && frame.play_sub2 == (index == 35), "frame {index} samples");
    }
    assert!(
        frames.len() == 60 && markers == 27 && stages == 20 && next.counter == 61,
        "frames={} markers={markers} stages={stages} next={next:?}",
        frames.len()
    );
}

// Go: TestNativeCommand9AIOrbitAlternatesFrameZero
#[test]
fn native_command9_ai_orbit_alternates_frame_zero() {
    let s = build_native_command9_ai_schedule(0, Some(&command9_animation(31))).unwrap();
    let mut state = new_native_command9_ai_state();
    for index in 0..4 {
        let frame = plan_native_command9_ai_orbit_frame(state, s).unwrap();
        assert_eq!(frame.effect_frames.len() == 1, index % 2 == 0, "orbit frame {index}={frame:?}");
        state = frame.next;
    }
}

// Go: TestNativeCommand9AIOriginalAssets
#[test]
fn native_command9_ai_original_assets() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let effect = decode_resource(&path, NATIVE_COMMAND9_AI_EFFECT_RESOURCE).unwrap();
    assert_eq!(effect.frames.len(), 31, "FDOTHER #44 frames");
    build_native_command9_ai_schedule(0, Some(&effect)).unwrap();
    for sample in 0..=2 {
        let raw = read_nested_resource(&path, NATIVE_COMMAND9_AI_SOUND_RESOURCE as usize, sample).unwrap();
        assert!(!raw.is_empty(), "FDOTHER #90 sub{sample}");
    }
}
