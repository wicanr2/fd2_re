//! 移植自 `native_command_heal_presentation_test.go`。

use crate::fdother::*;

// Go: TestBuildNativeCommandHealPresentationSchedulePreserves21EB1Loops
#[test]
fn build_native_command_heal_presentation_schedule_preserves_21eb1_loops() {
    for (id, first, last) in [(13, 1, 17), (14, 2, 34), (15, 8, 40), (16, 6, 54)] {
        let s = build_native_command_heal_presentation_schedule(id).unwrap();
        assert!(
            s.frames.len() == 16
                && s.mid_frame == 9
                && s.mid_delay_ms == 200
                && s.tail_delay_ms == 200
                && s.sample_index == 11,
            "id {id} schedule={s:?}"
        );
        let f = s.frames[0];
        assert!(f.lut_index == 9 && f.radius == first && f.frame_delay_ms == 5, "id {id} first={f:?}");
        let f = s.frames[8];
        assert!(f.lut_index == 1 && f.radius == last, "id {id} expansion tail={f:?}");
        let f = s.frames[9];
        assert!(f.lut_index == 3 && f.radius == last, "id {id} release start={f:?}");
        let f = s.frames[15];
        assert!(f.lut_index == 9 && f.radius == last, "id {id} release tail={f:?}");
    }
}

// Go: TestBuildNativeCommandHealPresentationScheduleRejectsOtherCommands
#[test]
fn build_native_command_heal_presentation_schedule_rejects_other_commands() {
    for id in [-1, 12, 17, 255] {
        assert!(build_native_command_heal_presentation_schedule(id).is_err(), "id {id} unexpectedly accepted");
    }
}
