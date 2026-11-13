//! Go `native_command0_test.go`。

use super::*;

pub(super) fn command0_test_animation() -> Animation {
    let mut a = test_animation(NATIVE_COMMAND0_EFFECT_FRAME_COUNT as usize, 0);
    a.frames[9].raw_byte4 = 1;
    a
}

// Go: TestBuildNativeCommand0PresentationSchedulePreservesRawTables
#[test]
fn build_native_command0_presentation_schedule_preserves_raw_tables() {
    for (side, want) in [(0u8, 20), (1, 18), (2, 18)] {
        let s = build_native_command0_presentation_schedule(side, Some(&command0_test_animation())).unwrap();
        let want_shift = if side == 0 { 148 } else { 0 };
        assert!(
            s.effect_resource == want
                && s.sound_resource == 82
                && s.sound_index == 1
                && s.frames == 28
                && s.damage_stages == 7
                && s.side_x_shift == want_shift,
            "side={side} schedule={s:?}"
        );
    }
    let s = build_native_command0_presentation_schedule(1, Some(&command0_test_animation())).unwrap();
    assert_eq!(s.layer_flags, [0, 0, 1, 0, 1, 0, 0]);
    assert_eq!(s.x_offsets, [40, 70, 120, 80, 50, 100, 70]);
    assert_eq!(s.y_offsets, [0, -10, -20, 0, -15, -5, 0]);
}

// Go: TestNativeCommand0StaggerAndImpactMarkers
#[test]
fn native_command0_stagger_and_impact_markers() {
    for (layer, step) in [3, 5, 7, 9, 11, 13, 15].into_iter().enumerate() {
        assert!(native_command0_impact(step, layer as i64), "layer {layer} impact step {step} missing");
    }
    assert!(!native_command0_effect_frame(0, 1).1, "layer1 must still be outside the frame bank at step0");
    assert_eq!(native_command0_effect_frame(27, 6), (15, true), "last staggered frame");
}

// Go: TestBuildNativeCommand0PresentationScheduleRejectsMalformedBank
#[test]
fn build_native_command0_presentation_schedule_rejects_malformed_bank() {
    let mut bad = command0_test_animation();
    bad.frames.truncate(15);
    assert!(build_native_command0_presentation_schedule(0, Some(&bad)).is_err(), "short command0 effect bank accepted");
    let mut bad = command0_test_animation();
    bad.frames[3].delay = 1;
    assert!(
        build_native_command0_presentation_schedule(1, Some(&bad)).is_err(),
        "edited command0 frame signature accepted"
    );
    let mut bad = command0_test_animation();
    bad.frames[9].raw_byte4 = 0;
    assert!(
        build_native_command0_presentation_schedule(1, Some(&bad)).is_err(),
        "edited command0 raw byte4 marker accepted"
    );
}

// Go: TestOriginalFDOTHERCommand0EffectBanksMatchRecoveredSignatures
#[test]
fn original_fdother_command0_effect_banks_match_recovered_signatures() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    for (resource, side) in [(18, 1u8), (20, 0)] {
        let animation = decode_resource(&path, resource).unwrap();
        let schedule = build_native_command0_presentation_schedule(side, Some(&animation)).unwrap();
        assert_eq!(schedule.effect_resource, resource, "resource {resource} selected");
    }
}
