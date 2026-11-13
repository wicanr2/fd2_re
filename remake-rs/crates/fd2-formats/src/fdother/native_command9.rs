//! 移植自 Go `native_command9.go`。

use super::native_ai_item_type21_tail::{NATIVE_DIGIT_VERTICAL, lmi1_entry_valid};
use super::{FdotherError, LMI1Entry, Result};

pub const NATIVE_COMMAND9_PLAYER_EFFECT_RESOURCE: i64 = 6;
pub const NATIVE_COMMAND9_PLAYER_EFFECT_START: i64 = 87;
pub const NATIVE_COMMAND9_PLAYER_EFFECT_FRAMES: i64 = 27;
pub const NATIVE_COMMAND9_PLAYER_SOUND_RESOURCE: i64 = 80;
pub const NATIVE_COMMAND9_PLAYER_INITIAL_SAMPLE: i64 = 14;
pub const NATIVE_COMMAND9_PLAYER_REPEAT_SAMPLE: i64 = 15;
pub const NATIVE_COMMAND9_RESULT_RESOURCE: i64 = 5;
pub const NATIVE_COMMAND9_RESULT_DIGIT_BIAS: i64 = 94;
pub const NATIVE_COMMAND9_RESULT_FRAMES: i64 = 22;
pub const NATIVE_COMMAND9_RESULT_DELAY_MS: i64 = 2;
pub const NATIVE_COMMAND9_RESULT_HOLD_MS: i64 = 500;

/// 保存玩家路徑 `0x1D6C8 -> 0x214AD -> 0x1C4CC -> 0x1DF58` 的原始表值。它不描述敵方
/// `0x2A6BD -> 0x275D6` 的另一套畫面排程。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand9PlayerSchedule {
    pub effect_resource: i64,
    pub effect_start: i64,
    pub effect_frames: i64,
    pub sound_resource: i64,
    pub initial_sample: i64,
    pub repeat_sample: i64,
    pub repeat_sample_frames: [i64; 2],
    pub effect_frame_delay_ticks: i64,
    pub result_resource: i64,
    pub result_digit_bias: i64,
    pub result_miss_descriptors: [i64; 4],
    pub result_frames: i64,
    pub result_delay_ms: i64,
    pub result_hold_ms: i64,
    pub result_vertical: [i64; 25],
}

pub fn build_native_command9_player_schedule(
    effect: &[LMI1Entry],
    result: &[LMI1Entry],
) -> Result<NativeCommand9PlayerSchedule> {
    if (effect.len() as i64) < NATIVE_COMMAND9_PLAYER_EFFECT_START + NATIVE_COMMAND9_PLAYER_EFFECT_FRAMES {
        return Err(FdotherError::fixed("native command9 player effect descriptors unavailable"));
    }
    for index in
        NATIVE_COMMAND9_PLAYER_EFFECT_START..NATIVE_COMMAND9_PLAYER_EFFECT_START + NATIVE_COMMAND9_PLAYER_EFFECT_FRAMES
    {
        if !lmi1_entry_valid(&effect[index as usize]) {
            return Err(FdotherError::message(format!("native command9 player effect descriptor {index} invalid")));
        }
    }
    const REQUIRED: [usize; 13] = [94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 116, 117, 118];
    for index in REQUIRED {
        if index >= result.len() || !lmi1_entry_valid(&result[index]) {
            return Err(FdotherError::message(format!("native command9 result descriptor {index} unavailable")));
        }
    }
    Ok(NativeCommand9PlayerSchedule {
        effect_resource: NATIVE_COMMAND9_PLAYER_EFFECT_RESOURCE,
        effect_start: NATIVE_COMMAND9_PLAYER_EFFECT_START,
        effect_frames: NATIVE_COMMAND9_PLAYER_EFFECT_FRAMES,
        sound_resource: NATIVE_COMMAND9_PLAYER_SOUND_RESOURCE,
        initial_sample: NATIVE_COMMAND9_PLAYER_INITIAL_SAMPLE,
        repeat_sample: NATIVE_COMMAND9_PLAYER_REPEAT_SAMPLE,
        repeat_sample_frames: [15, 19],
        effect_frame_delay_ticks: 1,
        result_resource: NATIVE_COMMAND9_RESULT_RESOURCE,
        result_digit_bias: NATIVE_COMMAND9_RESULT_DIGIT_BIAS,
        result_miss_descriptors: [116, 117, 118, 118],
        result_frames: NATIVE_COMMAND9_RESULT_FRAMES,
        result_delay_ms: NATIVE_COMMAND9_RESULT_DELAY_MS,
        result_hold_ms: NATIVE_COMMAND9_RESULT_HOLD_MS,
        result_vertical: NATIVE_DIGIT_VERTICAL,
    })
}
