//! Go `native_command7_test.go`。

use super::*;
use crate::archive::read_nested_resource;

fn command7_test_animation() -> Animation {
    let mut a = test_animation(NATIVE_COMMAND7_EFFECT_FRAME_COUNT as usize, 1);
    for (frame, delay) in a.frames.iter_mut().zip([2, 2, 2, 2, 4]) {
        frame.delay = delay;
    }
    a
}

fn schedule() -> NativeCommand7PresentationSchedule {
    build_native_command7_presentation_schedule(1, Some(&command7_test_animation())).unwrap()
}

fn after_front() -> NativeCommand7State {
    let mut state = new_native_command7_state();
    for _ in 0..NATIVE_COMMAND7_FRONT_FRAMES {
        state = plan_native_command7_draw_frame(state, schedule(), 1).unwrap().next;
    }
    state
}

// Go: TestNativeCommand7SchedulePreservesResourcesAndOffsets
#[test]
fn native_command7_schedule_preserves_resources_and_offsets() {
    let nonzero = schedule();
    assert!(
        nonzero.effect_resource == 37
            && nonzero.sound_resource == 88
            && nonzero.sound_indices == [0, 1]
            && nonzero.x_offsets == [30, -10, 70, 20, 100, 130, 40, 80, 110, 60],
        "nonzero schedule={nonzero:?}"
    );
    let zero = build_native_command7_presentation_schedule(0, Some(&command7_test_animation())).unwrap();
    assert_eq!(zero.effect_resource, 38);
}

// Go: TestNativeCommand7Mode0PreservesUnusedFourthGroup
#[test]
fn native_command7_mode0_preserves_unused_fourth_group() {
    let state = new_native_command7_state();
    assert!(
        state.counters == [0, -3, -6, -9]
            && state.positions == [0, 1, 2, 3]
            && state.next_position == 4
            && !state.stop
            && state.toggle == 0,
        "state={state:?}"
    );
}

// Go: TestNativeCommand7ToggleRepeatsBeforeAdvance
#[test]
fn native_command7_toggle_repeats_before_advance() {
    let first = plan_native_command7_draw_frame(new_native_command7_state(), schedule(), 1).unwrap();
    assert!(
        first.next.counters == [0, -3, -6, -9] && first.next.toggle == 1 && first.layers.len() == 1,
        "first={first:?}"
    );
    let second = plan_native_command7_draw_frame(first.next, schedule(), 1).unwrap();
    assert!(
        second.next.counters == [1, -2, -5, -9] && second.next.toggle == 0 && second.layers.len() == 1,
        "second={second:?}"
    );
}

// Go: TestNativeCommand7FirstTargetMarkersAndSamples
#[test]
fn native_command7_first_target_markers_and_samples() {
    let (frames, _) = build_native_command7_target_sequence(after_front(), schedule(), 1).unwrap();
    let want_markers = [1, 7, 13, 15, 21, 27, 29];
    let want_a = [1, 15, 29];
    let want_b = [7, 21];
    let mut published = 0;
    for (index, frame) in frames.iter().enumerate() {
        assert!(
            frame.numeric_marker == want_markers.contains(&index)
                && frame.play_handle_a == want_a.contains(&index)
                && frame.play_handle_b == want_b.contains(&index),
            "frame {index}={frame:?}"
        );
        if frame.hp_stage != 0 {
            published += 1;
            assert_eq!(frame.hp_stage, published, "frame {index} stage");
        }
    }
    assert_eq!(published, NATIVE_COMMAND7_DAMAGE_STAGES);
}

// Go: TestNativeCommand7TransitionAndTailPreserveState
#[test]
fn native_command7_transition_and_tail_preserve_state() {
    let (_, state) = build_native_command7_target_sequence(after_front(), schedule(), 1).unwrap();
    let (transition, state) = build_native_command7_transition_sequence(state, schedule(), 1).unwrap();
    let want_offsets = [-35, -70, -105, -140, -140, -105, -70, -35, 0];
    for (index, frame) in transition.iter().enumerate() {
        assert!(
            frame.target_offset_x == want_offsets[index] && frame.use_next_target == (index >= 4),
            "transition {index}={frame:?}"
        );
    }
    let (tail, last) = build_native_command7_tail_sequence(state, schedule(), 1).unwrap();
    assert!(tail.len() as i64 == NATIVE_COMMAND7_TAIL_FRAMES && last.stop, "tail={} final={last:?}", tail.len());
}

// Go: TestOriginalFDOTHERCommand7ResourcesMatchRecoveredSignatures
#[test]
fn original_fdother_command7_resources_match_recovered_signatures() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    for (resource, side) in [(37, 1u8), (38, 0)] {
        let animation = decode_resource(&path, resource).unwrap();
        let schedule = build_native_command7_presentation_schedule(side, Some(&animation)).unwrap();
        assert_eq!(schedule.effect_resource, resource);
    }
    for sample in [0, 1] {
        let raw = read_nested_resource(&path, NATIVE_COMMAND7_SOUND_RESOURCE as usize, sample).unwrap();
        assert!(!raw.is_empty(), "resource 88 sample {sample}");
    }
}

// Go: TestNativeCommand7RNGWalkRejectsUnavailableInputs
#[test]
fn native_command7_rng_walk_rejects_unavailable_inputs() {
    let schedule = schedule();
    for (count, side, with_resolver) in [(0, 1u8, true), (1, 0, true), (1, 1, false)] {
        let mut resolve = |_: i64, rng: u16| -> Resolved { Ok((rng, false)) };
        let resolver: Resolver<'_> = if with_resolver { Some(&mut resolve) } else { None };
        assert!(
            walk_native_command7_rng(10947, schedule, side, count, resolver).is_err(),
            "unavailable inputs accepted: count={count} side={side}"
        );
    }
    let mut failing = |_: i64, _: u16| -> Resolved { Err("resolver unavailable".into()) };
    let err = walk_native_command7_rng(10947, schedule, 1, 2, Some(&mut failing)).unwrap_err();
    assert!(
        matches!(&err.error, FiganiError::Resolver(e) if e.to_string() == "resolver unavailable"),
        "resolver error lost: {err}"
    );
}
