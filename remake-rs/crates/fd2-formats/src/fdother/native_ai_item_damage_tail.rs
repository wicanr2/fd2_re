//! 移植自 Go `native_ai_item_damage_tail.go`。

use super::native_flow_error::{NativeFlowError, NativeFlowResult};

pub const NATIVE_AI_ITEM_DAMAGE_EFFECT_RESOURCE: i64 = 6;
pub const NATIVE_AI_ITEM_DAMAGE_EFFECT_START: i64 = 0x31;
pub const NATIVE_AI_ITEM_DAMAGE_EFFECT_FRAMES: i64 = 8;
pub const NATIVE_AI_ITEM_DAMAGE_EFFECT_SAMPLE: i64 = 6;
pub const NATIVE_AI_ITEM_DAMAGE_BLEND_FRAMES: usize = 10;
pub const NATIVE_AI_ITEM_DAMAGE_RAW_BASE: i64 = 0x20;
pub const NATIVE_AI_ITEM_DAMAGE_DIGIT_RESOURCE: i64 = 5;
pub const NATIVE_AI_ITEM_DAMAGE_DIGIT_BIAS: i64 = 0x5e;
pub const NATIVE_AI_ITEM_DAMAGE_DIGIT_FRAMES: i64 = 22;
pub const NATIVE_AI_ITEM_DAMAGE_DIGIT_HOLD_MS: i64 = 500;

/// type 20/24 的 `0x1c4cc -> 0x1cd17 -> 0x1e0db/0x1e1dc -> 0x1df58` 路徑專屬的資料契約。
/// raw descriptor 與調色盤值刻意不命名。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAIItemDamageTailSchedule {
    pub effect_resource: i64,
    pub effect_start: i64,
    pub effect_frames: i64,
    pub effect_sample: i64,
    pub effect_frame_delay_ticks: i64,
    pub blend_frames: i64,
    pub raw_base: i64,
    pub blend_delay_ticks: i64,
    pub blend: [i64; NATIVE_AI_ITEM_DAMAGE_BLEND_FRAMES],
    pub digit_resource: i64,
    pub digit_bias: i64,
    pub digit_frames: i64,
    pub digit_frame_delay_ticks: i64,
    pub digit_hold_ms: i64,
    pub digit_vertical: [i64; 25],
}

/// 只接受已證實的 command 0、2、3。
pub fn build_native_ai_item_damage_tail_schedule(command_id: i64) -> NativeFlowResult<NativeAIItemDamageTailSchedule> {
    if command_id != 0 && command_id != 2 && command_id != 3 {
        return Err(NativeFlowError::AiItemDamageCommand(command_id));
    }
    Ok(NativeAIItemDamageTailSchedule {
        effect_resource: NATIVE_AI_ITEM_DAMAGE_EFFECT_RESOURCE,
        effect_start: NATIVE_AI_ITEM_DAMAGE_EFFECT_START,
        effect_frames: NATIVE_AI_ITEM_DAMAGE_EFFECT_FRAMES,
        effect_sample: NATIVE_AI_ITEM_DAMAGE_EFFECT_SAMPLE,
        effect_frame_delay_ticks: 1,
        blend_frames: NATIVE_AI_ITEM_DAMAGE_BLEND_FRAMES as i64,
        raw_base: NATIVE_AI_ITEM_DAMAGE_RAW_BASE,
        blend_delay_ticks: 1,
        blend: [7, 6, 5, 4, 3, 2, 1, 0, 7, 6],
        digit_resource: NATIVE_AI_ITEM_DAMAGE_DIGIT_RESOURCE,
        digit_bias: NATIVE_AI_ITEM_DAMAGE_DIGIT_BIAS,
        digit_frames: NATIVE_AI_ITEM_DAMAGE_DIGIT_FRAMES,
        digit_frame_delay_ticks: 1,
        digit_hold_ms: NATIVE_AI_ITEM_DAMAGE_DIGIT_HOLD_MS,
        digit_vertical: [15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15],
    })
}

#[cfg(test)]
mod tests;
