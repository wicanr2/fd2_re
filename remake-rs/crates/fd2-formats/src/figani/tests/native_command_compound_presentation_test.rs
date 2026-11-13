//! Go `native_command_compound_presentation_test.go`。

use super::*;
use crate::archive::read_nested_resource;

// Go: TestBuildNativeCompoundPresentationSchedulePreservesRawTables
#[test]
fn build_native_compound_presentation_schedule_preserves_raw_tables() {
    let effect = Animation { frames: vec![Frame::default(); 8], ..Animation::default() };
    let want_colors: [[u8; 3]; 4] = [[63, 63, 63], [51, 57, 63], [53, 0, 0], [53, 58, 9]];
    for id in 32..=35 {
        let s = build_native_compound_presentation_schedule(id, Some(&effect)).unwrap();
        assert!(
            s.effect_resource == id + 33
                && s.sound_resource == id + 59
                && s.actor_slide_frames == 8
                && s.actor_slide_step_x == 20
                && s.effect_slide_frames == 9
                && s.effect_slide_step_x == 30
                && s.prelude_hold_ticks == 6
                && s.main_delay_ticks == 2
                && s.main_frame_count == 7
                && s.palette_rgb == want_colors[(id - 32) as usize],
            "command{id} schedule={s:?}"
        );
    }
    let get = |id| build_native_compound_presentation_schedule(id, Some(&effect)).unwrap();
    let (s32, s33, s34, s35) = (get(32), get(33), get(34), get(35));
    assert!(
        s32.tail_enabled
            && s32.mixer_sample2_frame == 1
            && s33.overlay_first_frame
            && s33.sample1_frame == 6
            && s34.overlay_first_frame
            && s34.sample1_frame == 2
            && s35.tail_enabled
            && s35.mixer_sample2_frame == 1,
        "command-specific markers: 32={s32:?} 33={s33:?} 34={s34:?} 35={s35:?}"
    );
}

// Go: TestNativeCompoundPresentationOriginalResources
#[test]
fn native_compound_presentation_original_resources() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    for id in 32..=35i64 {
        let effect = decode_resource(&path, id + 33).unwrap_or_else(|e| panic!("command{id} effect: {e}"));
        build_native_compound_presentation_schedule(id, Some(&effect))
            .unwrap_or_else(|e| panic!("command{id} schedule: {e}"));
        let mut samples = vec![1];
        if id == 32 || id == 35 {
            samples.push(2);
        }
        for sample in samples {
            let raw = read_nested_resource(&path, (id + 59) as usize, sample).unwrap();
            assert!(!raw.is_empty(), "command{id} sound{sample}");
        }
    }
}
