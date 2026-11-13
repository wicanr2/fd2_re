//! 移植自 `native_command_heal_tail_test.go`。

use crate::fdother::*;

// Go: TestBuildNativeCommandHealTailSchedulePreservesRawTables
#[test]
fn build_native_command_heal_tail_schedule_preserves_raw_tables() {
    for id in 13..=16 {
        let s = build_native_command_heal_tail_schedule(id).unwrap();
        assert!(
            s.effect_resource == 6
                && s.effect_start == 0x39
                && s.effect_frames == 7
                && s.effect_sample == 12
                && s.effect_frame_delay_ticks == 1
                && s.mask_sample == 1
                && s.mask_index == 0xc0
                && s.mask_pairs == 5
                && s.digit_resource == 5
                && s.digit_bias == 0x69
                && s.digit_frames == 22
                && s.digit_frame_delay_ms == 2
                && s.digit_hold_ms == 500,
            "id {id} schedule={s:?}"
        );
        let want = [15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15];
        assert_eq!(s.digit_vertical, want, "id {id} vertical");
    }
}

// Go: TestBuildNativeCommandHealTailScheduleRejectsOtherCommands
#[test]
fn build_native_command_heal_tail_schedule_rejects_other_commands() {
    for id in [-1, 12, 17, 40] {
        assert!(build_native_command_heal_tail_schedule(id).is_err(), "id {id} unexpectedly accepted");
    }
}
