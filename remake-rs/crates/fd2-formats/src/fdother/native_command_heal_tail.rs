//! 移植自 Go `native_command_heal_tail.go`。

use super::native_flow_error::{NativeFlowError, NativeFlowResult};

pub const NATIVE_COMMAND_HEAL_TAIL_EFFECT_RESOURCE: i64 = 6;
pub const NATIVE_COMMAND_HEAL_TAIL_DIGIT_RESOURCE: i64 = 5;
pub const NATIVE_COMMAND_HEAL_TAIL_EFFECT_START: i64 = 0x39;
pub const NATIVE_COMMAND_HEAL_TAIL_EFFECT_FRAMES: i64 = 7;
pub const NATIVE_COMMAND_HEAL_TAIL_EFFECT_SAMPLE: i64 = 12;
pub const NATIVE_COMMAND_HEAL_TAIL_MASK_SAMPLE: i64 = 1;
pub const NATIVE_COMMAND_HEAL_TAIL_MASK_INDEX: i64 = 0xc0;
pub const NATIVE_COMMAND_HEAL_TAIL_MASK_PAIRS: i64 = 5;
pub const NATIVE_COMMAND_HEAL_TAIL_DIGIT_BIAS: i64 = 0x69;
pub const NATIVE_COMMAND_HEAL_TAIL_DIGIT_FRAMES: i64 = 22;
pub const NATIVE_COMMAND_HEAL_TAIL_DIGIT_DELAY_MS: i64 = 2;
pub const NATIVE_COMMAND_HEAL_TAIL_DIGIT_HOLD_MS: i64 = 500;

/// `0x1c4cc -> 0x1c2da -> 0x1e0db/0x1df58` 呼叫端專屬的具型別契約。raw 調色盤與
/// descriptor 值刻意不命名。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandHealTailSchedule {
    pub effect_resource: i64,
    pub effect_start: i64,
    pub effect_frames: i64,
    pub effect_sample: i64,
    pub effect_frame_delay_ticks: i64,
    pub mask_sample: i64,
    pub mask_index: i64,
    pub mask_pairs: i64,
    pub digit_resource: i64,
    pub digit_bias: i64,
    pub digit_frames: i64,
    pub digit_frame_delay_ms: i64,
    pub digit_hold_ms: i64,
    pub digit_vertical: [i64; 25],
}

pub fn build_native_command_heal_tail_schedule(command_id: i64) -> NativeFlowResult<NativeCommandHealTailSchedule> {
    if !(13..=16).contains(&command_id) {
        return Err(NativeFlowError::HealTail(command_id));
    }
    Ok(NativeCommandHealTailSchedule {
        effect_resource: NATIVE_COMMAND_HEAL_TAIL_EFFECT_RESOURCE,
        effect_start: NATIVE_COMMAND_HEAL_TAIL_EFFECT_START,
        effect_frames: NATIVE_COMMAND_HEAL_TAIL_EFFECT_FRAMES,
        effect_sample: NATIVE_COMMAND_HEAL_TAIL_EFFECT_SAMPLE,
        effect_frame_delay_ticks: 1,
        mask_sample: NATIVE_COMMAND_HEAL_TAIL_MASK_SAMPLE,
        mask_index: NATIVE_COMMAND_HEAL_TAIL_MASK_INDEX,
        mask_pairs: NATIVE_COMMAND_HEAL_TAIL_MASK_PAIRS,
        digit_resource: NATIVE_COMMAND_HEAL_TAIL_DIGIT_RESOURCE,
        digit_bias: NATIVE_COMMAND_HEAL_TAIL_DIGIT_BIAS,
        digit_frames: NATIVE_COMMAND_HEAL_TAIL_DIGIT_FRAMES,
        digit_frame_delay_ms: NATIVE_COMMAND_HEAL_TAIL_DIGIT_DELAY_MS,
        digit_hold_ms: NATIVE_COMMAND_HEAL_TAIL_DIGIT_HOLD_MS,
        digit_vertical: [15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15],
    })
}

#[cfg(test)]
mod tests;
