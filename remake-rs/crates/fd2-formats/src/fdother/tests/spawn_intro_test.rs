//! 移植自 `spawn_intro_test.go`。

use super::original;
use crate::fdother::*;

// Go: TestNativeSpawnIntroSchedulePreservesAllTwelvePasses
#[test]
fn native_spawn_intro_schedule_preserves_all_twelve_passes() {
    let steps = native_spawn_intro_schedule();
    validate_native_spawn_intro_schedule(&steps).unwrap();
    assert!(steps.len() == 12 && steps[0].lmi_entry == 0 && steps[11].lmi_entry == 11, "spawn-intro steps={steps:?}");
    assert!(
        steps[1].sound_resource == 95 && steps[0].sound_resource == -1 && steps[2].sound_resource == -1,
        "pass1 raw resource boundary={steps:?}"
    );
    let got = steps[6];
    assert!(
        got.snapshot_mode == NATIVE_SPAWN_INTRO_SPLIT_UNITS && !got.redraw_terrain && got.new_unit_pixel_lift == -8,
        "pass6={got:?}"
    );
    let got = steps[7];
    assert!(
        got.snapshot_mode == NATIVE_SPAWN_INTRO_SPLIT_UNITS && got.redraw_terrain && got.new_unit_pixel_lift == -5,
        "pass7={got:?}"
    );
    let got = steps[8];
    assert!(
        got.snapshot_mode == NATIVE_SPAWN_INTRO_FULL_FRAME && got.redraw_terrain && got.new_unit_pixel_lift == 0,
        "pass8={got:?}"
    );
    for pass in [0, 1, 2, 3, 4, 5, 9, 10, 11] {
        assert!(
            steps[pass].snapshot_mode == NATIVE_SPAWN_INTRO_KEEP_SNAPSHOT && steps[pass].delay_ticks == 1,
            "ordinary pass {pass}={:?}",
            steps[pass]
        );
    }
}

// Go: TestValidateNativeSpawnIntroScheduleRejectsFormerTwelveTickShortcut
#[test]
fn validate_native_spawn_intro_schedule_rejects_former_twelve_tick_shortcut() {
    let mut steps = native_spawn_intro_schedule();
    steps[6].snapshot_mode = NATIVE_SPAWN_INTRO_KEEP_SNAPSHOT;
    assert!(validate_native_spawn_intro_schedule(&steps).is_err(), "schedule without pass6 snapshot rebuild");
    assert!(validate_native_spawn_intro_schedule(&steps[..11]).is_err(), "short spawn-intro schedule was accepted");
}

// Go: TestFDOTHER009NativeSpawnIntroFrames
#[test]
fn fdother009_native_spawn_intro_frames() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let entries = decode_native_spawn_intro_frames(&path).unwrap();
    assert!(
        entries.len() as i64 == NATIVE_SPAWN_INTRO_PASS_COUNT && entries[0].width == 42 && entries[11].width == 66,
        "FDOTHER#9 spawn-intro entries={entries:?}"
    );
}

// Go: TestNativeSpawnIntroVisibilityAndLMIOriginMatch32999
#[test]
fn native_spawn_intro_visibility_and_lmi_origin_match_32999() {
    assert!(
        native_spawn_intro_visible(9, 20, 10, 20, 12, 7)
            && !native_spawn_intro_visible(9, 19, 10, 20, 12, 7)
            && native_spawn_intro_visible(22, 28, 10, 20, 12, 7)
            && !native_spawn_intro_visible(23, 28, 10, 20, 12, 7),
        "0x32999 camera bounds differ"
    );
    let want = 0x8088 + 24 * (14 - 10 - 1) + 24 * 0x1c8 * (23 - 20) - 0xab0;
    assert_eq!(native_spawn_intro_lmi_origin(14, 23, 10, 20), want);
    let mut work = vec![0xeeu8; 0x25680];
    let entry = LMI1Entry { width: 2, height: 1, pixels: vec![0, 0x44] };
    blit_native_spawn_intro_lmi(&entry, &mut work, 14, 23, 10, 20).unwrap();
    let origin = native_spawn_intro_lmi_origin(14, 23, 10, 20) as usize;
    assert!(
        work[origin] == 0xee && work[origin + 1] == 0x44,
        "transparent LMI={:#x}/{:#x}",
        work[origin],
        work[origin + 1]
    );
}
