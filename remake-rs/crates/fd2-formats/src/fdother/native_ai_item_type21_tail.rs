//! 移植自 Go `native_ai_item_type21_tail.go`。

use super::{FdotherError, LMI1Entry, Result};

pub const NATIVE_AI_ITEM_TYPE21_EFFECT_RESOURCE: i64 = 6;
pub const NATIVE_AI_ITEM_TYPE21_TOGGLE_A: i64 = 0x4a;
pub const NATIVE_AI_ITEM_TYPE21_TOGGLE_B: i64 = 0x4b;
pub const NATIVE_AI_ITEM_TYPE21_TOGGLE_PAIRS: i64 = 4;
pub const NATIVE_AI_ITEM_TYPE21_TOGGLE_DELAY_MS: i64 = 90;
pub const NATIVE_AI_ITEM_TYPE21_SOUND_RESOURCE: i64 = 80;
pub const NATIVE_AI_ITEM_TYPE21_DIGIT_RESOURCE: i64 = 5;
pub const NATIVE_AI_ITEM_TYPE21_DIGIT_BIAS: i64 = 0x5e;
pub const NATIVE_AI_ITEM_TYPE21_DIGIT_FRAMES: i64 = 22;
pub const NATIVE_AI_ITEM_TYPE21_DIGIT_DELAY_MS: i64 = 2;
pub const NATIVE_AI_ITEM_TYPE21_DIGIT_HOLD_MS: i64 = 500;

/// 保存 type21 呼叫端專屬的 `0x1c4cc -> 0x1cac7 -> 結果佇列` 契約。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAIItemType21TailSchedule {
    pub command_id: i64,
    pub effect_resource: i64,
    pub effect_start: i64,
    pub effect_frames: i64,
    pub sound_resource: i64,
    pub effect_sample: i64,
    pub effect_delay_ticks: i64,
    pub toggle_a: i64,
    pub toggle_b: i64,
    pub toggle_pairs: i64,
    pub toggle_delay_ms: i64,
    pub digit_resource: i64,
    pub digit_bias: i64,
    pub digit_frames: i64,
    pub digit_delay_ms: i64,
    pub digit_hold_ms: i64,
    pub digit_vertical: [i64; 25],
}

/// LMI1 描述子的共用幾何檢查；`width*height` 與 Go 一樣以 64 位元繞回。
pub(super) fn lmi1_entry_valid(entry: &LMI1Entry) -> bool {
    entry.width > 0 && entry.height > 0 && entry.pixels.len() as i64 == entry.width.wrapping_mul(entry.height)
}

pub(super) const NATIVE_DIGIT_VERTICAL: [i64; 25] =
    [15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15];

pub fn build_native_ai_item_type21_tail_schedule(
    command_id: i64,
    effect: &[LMI1Entry],
    digits: &[LMI1Entry],
) -> Result<NativeAIItemType21TailSchedule> {
    let (start, frames, sample) = match command_id {
        1 => (0x31, 8, 6),
        6 | 7 => (0x40, 10, 9),
        _ => {
            return Err(FdotherError::message(format!("native AI item type21 tail: unsupported command {command_id}")));
        }
    };
    if effect.len() as i64 <= NATIVE_AI_ITEM_TYPE21_TOGGLE_B || start + frames > effect.len() as i64 {
        return Err(FdotherError::fixed("native AI item type21 effect descriptors unavailable"));
    }
    for index in start..start + frames {
        if !lmi1_entry_valid(&effect[index as usize]) {
            return Err(FdotherError::message(format!("native AI item type21 effect descriptor {index:#x} invalid")));
        }
    }
    for index in [NATIVE_AI_ITEM_TYPE21_TOGGLE_A, NATIVE_AI_ITEM_TYPE21_TOGGLE_B] {
        if !lmi1_entry_valid(&effect[index as usize]) {
            return Err(FdotherError::message(format!("native AI item type21 toggle descriptor {index:#x} invalid")));
        }
    }
    if digits.len() as i64 <= NATIVE_AI_ITEM_TYPE21_DIGIT_BIAS + 9 {
        return Err(FdotherError::fixed("native AI item type21 digit descriptors unavailable"));
    }
    Ok(NativeAIItemType21TailSchedule {
        command_id,
        effect_resource: NATIVE_AI_ITEM_TYPE21_EFFECT_RESOURCE,
        effect_start: start,
        effect_frames: frames,
        sound_resource: NATIVE_AI_ITEM_TYPE21_SOUND_RESOURCE,
        effect_sample: sample,
        effect_delay_ticks: 1,
        toggle_a: NATIVE_AI_ITEM_TYPE21_TOGGLE_A,
        toggle_b: NATIVE_AI_ITEM_TYPE21_TOGGLE_B,
        toggle_pairs: NATIVE_AI_ITEM_TYPE21_TOGGLE_PAIRS,
        toggle_delay_ms: NATIVE_AI_ITEM_TYPE21_TOGGLE_DELAY_MS,
        digit_resource: NATIVE_AI_ITEM_TYPE21_DIGIT_RESOURCE,
        digit_bias: NATIVE_AI_ITEM_TYPE21_DIGIT_BIAS,
        digit_frames: NATIVE_AI_ITEM_TYPE21_DIGIT_FRAMES,
        digit_delay_ms: NATIVE_AI_ITEM_TYPE21_DIGIT_DELAY_MS,
        digit_hold_ms: NATIVE_AI_ITEM_TYPE21_DIGIT_HOLD_MS,
        digit_vertical: NATIVE_DIGIT_VERTICAL,
    })
}
