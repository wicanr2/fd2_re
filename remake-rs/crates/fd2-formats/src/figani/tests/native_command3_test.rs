//! Go `native_command3_test.go`。

use super::*;
use crate::archive::read_nested_resource;

fn command3_test_animation() -> Animation {
    test_animation(NATIVE_COMMAND3_EFFECT_FRAME_COUNT as usize, 1)
}

fn schedule() -> NativeCommand3PresentationSchedule {
    build_native_command3_presentation_schedule(1, Some(&command3_test_animation())).unwrap()
}

fn after_front() -> NativeCommand3State {
    let mut state = new_native_command3_state();
    for _ in 0..NATIVE_COMMAND3_FRONT_FRAMES {
        state = plan_native_command3_draw_frame(state, schedule(), 1).unwrap().next;
    }
    state
}

// Go: TestNativeCommand3SchedulePreservesResourcesAndTables
#[test]
fn native_command3_schedule_preserves_resources_and_tables() {
    let nonzero = schedule();
    assert!(
        nonzero.effect_resource == 39
            && nonzero.sound_resource == 84
            && nonzero.x_offsets == [30, 0, 70, 40, 130, 70, -30, 30, 110, 80, -10, 30]
            && nonzero.y_offsets == [0, 10, 0, 20, 5, 20, 0, 10, 0, 18, 0, 10]
            && nonzero.frame_bases == [22, 0, 0, 0, 0, 0, 11, 0, 22, 0, 0, 0],
        "nonzero schedule={nonzero:?}"
    );
    let zero = build_native_command3_presentation_schedule(0, Some(&command3_test_animation())).unwrap();
    assert_eq!(zero.effect_resource, 43);
}

// Go: TestNativeCommand3Mode0IsDeterministic
#[test]
fn native_command3_mode0_is_deterministic() {
    let state = new_native_command3_state();
    assert!(
        state.counters == [0, -2, -4, -6, -8, -10, -12, -14, -16, -18, -20, -22]
            && state.positions == [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11]
            && state.next_position == 12
            && !state.stop
            && state.toggle == 0,
        "state={state:?}"
    );
}

// Go: TestNativeCommand3ToggleRepeatsBeforeAdvance
#[test]
fn native_command3_toggle_repeats_before_advance() {
    let first = plan_native_command3_draw_frame(new_native_command3_state(), schedule(), 1).unwrap();
    assert!(first.next.counters[0] == 0 && first.next.toggle == 1 && first.layers.len() == 1, "first={first:?}");
    let second = plan_native_command3_draw_frame(first.next, schedule(), 1).unwrap();
    assert!(
        second.next.counters[0] == 1 && second.next.counters[1] == -1 && second.next.toggle == 0 && second.play_sub2,
        "second={second:?}"
    );
}

// Go: TestNativeCommand3FirstTargetMarkersAndSamples
#[test]
fn native_command3_first_target_markers_and_samples() {
    let (frames, _) = build_native_command3_target_sequence(after_front(), schedule(), 1).unwrap();
    let want_markers = [3, 7, 11, 15, 19, 23, 25, 27, 29, 31, 33, 35, 37, 39];
    let want_sub2 = [23, 31];
    let want_sub1 = [7, 11, 15, 19, 23, 25, 29, 31, 33, 37, 39];
    let mut published = 0;
    for (index, frame) in frames.iter().enumerate() {
        assert!(
            frame.numeric_marker == want_markers.contains(&index)
                && frame.play_sub1 == want_sub1.contains(&index)
                && frame.play_sub2 == want_sub2.contains(&index),
            "frame {index}={frame:?}"
        );
        if frame.hp_stage != 0 {
            published += 1;
            assert_eq!(frame.hp_stage, published, "frame {index} stage");
        }
    }
    assert!(published == NATIVE_COMMAND3_DAMAGE_STAGES && frames[39].hp_stage == 0, "published={published}");
}

// Go: TestNativeCommand3TransitionAndTailPreserveState
#[test]
fn native_command3_transition_and_tail_preserve_state() {
    let (_, state) = build_native_command3_target_sequence(after_front(), schedule(), 1).unwrap();
    let (transition, state) = build_native_command3_transition_sequence(state, schedule(), 1).unwrap();
    assert_eq!(transition.len() as i64, NATIVE_COMMAND3_TRANSITION_FRAMES);
    let (tail, last) = build_native_command3_tail_sequence(state, schedule(), 1).unwrap();
    assert!(tail.len() as i64 == NATIVE_COMMAND3_TAIL_FRAMES && last.stop, "tail={} final={last:?}", tail.len());
}

// Go: TestOriginalFDOTHERCommand3ResourcesMatchRecoveredSignatures
#[test]
fn original_fdother_command3_resources_match_recovered_signatures() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    for (resource, side) in [(39, 1u8), (43, 0)] {
        let animation = decode_resource(&path, resource).unwrap();
        let schedule = build_native_command3_presentation_schedule(side, Some(&animation)).unwrap();
        assert_eq!(schedule.effect_resource, resource);
    }
    for sample in [0, 1, 2] {
        let raw = read_nested_resource(&path, NATIVE_COMMAND3_SOUND_RESOURCE as usize, sample).unwrap();
        assert!(!raw.is_empty(), "resource 84 sample {sample}");
    }
}
