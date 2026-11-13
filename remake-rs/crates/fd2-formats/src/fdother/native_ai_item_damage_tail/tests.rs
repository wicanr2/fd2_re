//! 移植自 `native_ai_item_damage_tail_test.go`。

use crate::fdother::*;

// Go: TestBuildNativeAIItemDamageTailSchedulePreservesRawContract
#[test]
fn build_native_ai_item_damage_tail_schedule_preserves_raw_contract() {
    for command_id in [0, 2, 3] {
        let s = build_native_ai_item_damage_tail_schedule(command_id).unwrap();
        assert!(
            s.effect_resource == 6
                && s.effect_start == 0x31
                && s.effect_frames == 8
                && s.effect_sample == 6
                && s.blend_frames == 10
                && s.raw_base == 0x20
                && s.blend == [7, 6, 5, 4, 3, 2, 1, 0, 7, 6]
                && s.digit_resource == 5
                && s.digit_bias == 0x5e
                && s.digit_frames == 22
                && s.digit_hold_ms == 500,
            "command {command_id} schedule={s:?}"
        );
    }
}

// Go: TestBuildNativeAIItemDamageTailScheduleRejectsUnprovenCommands
#[test]
fn build_native_ai_item_damage_tail_schedule_rejects_unproven_commands() {
    for command_id in [-1, 1, 4, 32] {
        assert!(build_native_ai_item_damage_tail_schedule(command_id).is_err(), "command {command_id} accepted");
    }
}
