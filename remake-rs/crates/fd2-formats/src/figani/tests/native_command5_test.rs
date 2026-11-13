//! Go `native_command5_test.go`。

use super::*;
use crate::archive::read_nested_resource;
use crate::native_rng::native_rng_step;

fn command5_test_animation() -> Animation {
    test_animation(NATIVE_COMMAND5_EFFECT_FRAME_COUNT as usize, 1)
}

fn schedule() -> NativeCommand5PresentationSchedule {
    build_native_command5_presentation_schedule(1, Some(&command5_test_animation())).unwrap()
}

// Go: TestNativeCommand5SchedulePreservesSideResourcesAndOffsets
#[test]
fn native_command5_schedule_preserves_side_resources_and_offsets() {
    let nonzero = schedule();
    assert!(
        nonzero.effect_resource == 24
            && nonzero.sound_resource == 86
            && nonzero.sound_indices == [0, 1]
            && nonzero.x_offsets == [30, 50, 70, 40, 80, 100, 70, 30, 60, 90],
        "nonzero schedule={nonzero:?}"
    );
    let zero = build_native_command5_presentation_schedule(0, Some(&command5_test_animation())).unwrap();
    assert_eq!(zero.effect_resource, 25);
}

// Go: TestNativeCommand5Mode0ConsumesSixRNGSteps
#[test]
fn native_command5_mode0_consumes_six_rng_steps() {
    let state = new_native_command5_state(0x1234);
    let mut want_rng = 0x1234u16;
    let mut want_phases = [0i64; 6];
    for phase in &mut want_phases {
        want_rng = native_rng_step(want_rng);
        *phase = (want_rng % 2) as i64 * 6;
    }
    assert!(
        state.rng == want_rng
            && state.phases == want_phases
            && state.counters == [0, -2, -4, -6, -8, -10]
            && state.positions == [0, 1, 2, 3, 4, 5]
            && state.next_position == 6
            && !state.stop,
        "state={state:?}"
    );
}

// Go: TestNativeCommand5FrontAndTargetPreserveMarkersAndSamples
#[test]
fn native_command5_front_and_target_preserve_markers_and_samples() {
    let front = plan_native_command5_draw_frame(new_native_command5_state(0x4321), schedule(), 1).unwrap();
    assert!(
        front.layers.len() == 1 && front.play_primary && !front.play_secondary && !front.numeric_marker,
        "front={front:?}"
    );
    let (frames, _) = build_native_command5_target_sequence(front.next, schedule(), 1).unwrap();
    let want_markers = [0, 2, 4, 6, 7, 8, 9, 10, 11];
    let mut published = 0;
    for (index, frame) in frames.iter().enumerate() {
        assert_eq!(frame.numeric_marker, want_markers.contains(&index), "frame {index} marker");
        if frame.hp_stage != 0 {
            published += 1;
            assert_eq!(frame.hp_stage, published, "frame {index} stage");
        }
    }
    assert_eq!(published, NATIVE_COMMAND5_DAMAGE_STAGES);
}

// Go: TestNativeCommand5TransitionCarriesStateAndTailStopsReseeding
#[test]
fn native_command5_transition_carries_state_and_tail_stops_reseeding() {
    let front = plan_native_command5_draw_frame(new_native_command5_state(7), schedule(), 1).unwrap();
    let (_, state) = build_native_command5_target_sequence(front.next, schedule(), 1).unwrap();
    let (transition, after_transition) = build_native_command5_transition_sequence(state, schedule(), 1).unwrap();
    let want_offsets = [-35, -70, -105, -140, -140, -105, -70, -35, 0];
    for (index, frame) in transition.iter().enumerate() {
        assert!(
            frame.target_offset_x == want_offsets[index] && frame.use_next_target == (index >= 4),
            "transition {index}={frame:?}"
        );
    }
    let (tail, last) = build_native_command5_tail_sequence(after_transition, schedule(), 1).unwrap();
    assert!(tail.len() as i64 == NATIVE_COMMAND5_TAIL_FRAMES && last.stop, "tail={} final={last:?}", tail.len());
    for (channel, counter) in last.counters.iter().enumerate() {
        assert!(*counter <= 14, "channel {channel} unexpectedly reset after stop: {counter}");
    }
}

// Go: TestOriginalFDOTHERCommand5ResourcesMatchRecoveredSignatures
#[test]
fn original_fdother_command5_resources_match_recovered_signatures() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    for (resource, side) in [(24, 1u8), (25, 0)] {
        let animation = decode_resource(&path, resource).unwrap();
        let schedule = build_native_command5_presentation_schedule(side, Some(&animation)).unwrap();
        assert_eq!(schedule.effect_resource, resource);
    }
    for sample in [0, 1] {
        let raw = read_nested_resource(&path, NATIVE_COMMAND5_SOUND_RESOURCE as usize, sample).unwrap();
        assert!(!raw.is_empty(), "resource 86 sample {sample}");
    }
}
