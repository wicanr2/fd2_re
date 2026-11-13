//! Go `native_command8_test.go`。

use super::*;
use crate::archive::read_nested_resource;

fn command8_test_animation() -> Animation {
    test_animation(NATIVE_COMMAND8_EFFECT_FRAME_COUNT as usize, 1)
}

fn schedule() -> NativeCommand8PresentationSchedule {
    build_native_command8_presentation_schedule(1, Some(&command8_test_animation())).unwrap()
}

// Go: TestNativeCommand8ScheduleAndMode0
#[test]
fn native_command8_schedule_and_mode0() {
    let s = schedule();
    assert!(
        s.effect_resource == 28
            && s.sound_resource == 90
            && s.frame_bases == native_command8::NATIVE_COMMAND8_FRAME_BASES,
        "schedule={s:?}"
    );
    let z = build_native_command8_presentation_schedule(0, Some(&command8_test_animation())).unwrap();
    assert_eq!(z.effect_resource, 30);
    assert_eq!(
        new_native_command8_state().counters,
        [0, -2, -4, -6, -8, -10, -12, -14, -16, -18, -20, -22, -24, -26, -28, -30]
    );
}

// Go: TestNativeCommand8FirstTargetMarkersAndSamples
#[test]
fn native_command8_first_target_markers_and_samples() {
    let s = schedule();
    let mut state = new_native_command8_state();
    for _ in 0..NATIVE_COMMAND8_FRONT_FRAMES {
        state = plan_native_command8_draw_frame(state, s).unwrap().next;
    }
    let (frames, _) = build_native_command8_target_sequence(state, s).unwrap();
    let markers = |i: usize| i <= 30 && i.is_multiple_of(2);
    let sub1 = |i: usize| (1..=27).contains(&i) && i % 2 == 1;
    let sub2 = |i: usize| (1..=31).contains(&i) && i % 2 == 1;
    let mut stages = 0;
    for (i, f) in frames.iter().enumerate() {
        assert!(f.numeric_marker == markers(i) && f.play_sub1 == sub1(i) && f.play_sub2 == sub2(i), "frame {i}={f:?}");
        if f.hp_stage != 0 {
            stages += 1;
            assert_eq!(f.hp_stage, stages, "frame {i} stage");
        }
    }
    assert_eq!(stages, 16);
}

// Go: TestNativeCommand8TransitionAndTail
#[test]
fn native_command8_transition_and_tail() {
    let s = schedule();
    let (transition, state) = build_native_command8_transition_sequence(new_native_command8_state(), s, 1).unwrap();
    assert!(
        transition.len() == 9 && transition[0].target_offset_x == -35 && transition[8].target_offset_x == 0,
        "transition={transition:?}"
    );
    let (tail, _) = build_native_command8_tail_sequence(state, s).unwrap();
    assert_eq!(tail.len(), 2);
}

// Go: TestOriginalFDOTHERCommand8ResourcesMatchRecoveredShapes
#[test]
fn original_fdother_command8_resources_match_recovered_shapes() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    for (resource, side) in [(28, 1u8), (30, 0)] {
        let animation = decode_resource(&path, resource).unwrap();
        let schedule = build_native_command8_presentation_schedule(side, Some(&animation)).unwrap();
        assert_eq!(schedule.effect_resource, resource);
    }
    for sample in [0, 1, 2] {
        let raw = read_nested_resource(&path, NATIVE_COMMAND8_SOUND_RESOURCE as usize, sample).unwrap();
        assert!(!raw.is_empty(), "resource 90 sample {sample}");
    }
}
