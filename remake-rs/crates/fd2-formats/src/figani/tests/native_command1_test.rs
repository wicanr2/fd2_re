//! Go `native_command1_test.go`。

use super::*;

fn command1_test_animation() -> Animation {
    test_animation(NATIVE_COMMAND1_EFFECT_FRAME_COUNT as usize, 0)
}

// Go: TestNativeCommand1SchedulePreservesRawTablesAndSides
#[test]
fn native_command1_schedule_preserves_raw_tables_and_sides() {
    for (side, resource, shift) in [(0u8, 21, 148), (1, 19, 0), (2, 19, 0)] {
        let s = build_native_command1_presentation_schedule(side, Some(&command1_test_animation())).unwrap();
        assert!(
            s.effect_resource == resource
                && s.side_x_shift == shift
                && s.anchor_x == 0x50
                && s.x_offsets == native_command1::NATIVE_COMMAND1_X_OFFSETS
                && s.y_offsets == native_command1::NATIVE_COMMAND1_Y_OFFSETS,
            "side={side} schedule={s:?}"
        );
    }
}

// Go: TestNativeCommand1ModeFramesAndMarkers
#[test]
fn native_command1_mode_frames_and_markers() {
    assert_eq!(native_command1_counters(0).unwrap(), [0, -2, -4, -6, -8, -10, -12, -14], "mode3 counters");
    assert_eq!(native_command1_mode_frame(4, 4, 2), (17, 0, true), "mode4 second group");
    assert_eq!(native_command1_mode_frame(5, 0, 2), (17, 4, true), "mode5 first group");
    assert!(!native_command1_mode_frame(5, 0, 15).2, "counter15 must not draw");
    assert!(
        native_command1_counters(NATIVE_COMMAND1_PRESENTATION_FRAMES).is_err(),
        "frame after the 31-frame target budget was accepted"
    );
    let (mut numeric_steps, mut sample_steps) = (Vec::new(), Vec::new());
    for step in 0..NATIVE_COMMAND1_PRESENTATION_FRAMES {
        let (numeric, sample) = native_command1_target_markers(step).unwrap();
        if numeric {
            numeric_steps.push(step);
        }
        if sample {
            sample_steps.push(step);
        }
    }
    assert_eq!(numeric_steps, [8, 10, 12, 14, 16, 18, 20, 22], "numeric markers");
    assert_eq!(sample_steps, [4, 6, 8, 10, 12, 14, 16, 18], "sample markers");
}

// Go: TestOriginalFDOTHERCommand1EffectBanksMatchRecoveredSignatures
#[test]
fn original_fdother_command1_effect_banks_match_recovered_signatures() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    for (resource, side) in [(19, 1u8), (21, 0)] {
        let animation = decode_resource(&path, resource).unwrap();
        let schedule = build_native_command1_presentation_schedule(side, Some(&animation)).unwrap();
        assert_eq!(schedule.effect_resource, resource);
    }
}
