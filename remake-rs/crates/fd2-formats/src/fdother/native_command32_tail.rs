//! 移植自 Go `native_command32_tail.go`。

use super::native_ai_item_type21_tail::{NATIVE_DIGIT_VERTICAL, lmi1_entry_valid};
use super::{FdotherError, LMI1Entry, Result};

pub const NATIVE_COMMAND32_TAIL_EFFECT_RESOURCE: i64 = 6;
pub const NATIVE_COMMAND32_TAIL_EFFECT_START: i64 = 0x40;
pub const NATIVE_COMMAND32_TAIL_EFFECT_FRAMES: i64 = 10;
pub const NATIVE_COMMAND32_TAIL_TOGGLE_A: i64 = 0x4a;
pub const NATIVE_COMMAND32_TAIL_TOGGLE_B: i64 = 0x4b;
pub const NATIVE_COMMAND32_TAIL_TOGGLE_PAIRS: i64 = 4;
pub const NATIVE_COMMAND32_TAIL_TOGGLE_DELAY_MS: i64 = 90;
pub const NATIVE_COMMAND32_TAIL_SOUND_RESOURCE: i64 = 80;
pub const NATIVE_COMMAND32_TAIL_EFFECT_SAMPLE: i64 = 9;
pub const NATIVE_COMMAND32_TAIL_DIGIT_RESOURCE: i64 = 5;
pub const NATIVE_COMMAND32_TAIL_DIGIT_BIAS: i64 = 0x5e;
pub const NATIVE_COMMAND32_TAIL_DIGIT_FRAMES: i64 = 22;
pub const NATIVE_COMMAND32_TAIL_DIGIT_DELAY_MS: i64 = 2;
pub const NATIVE_COMMAND32_TAIL_DIGIT_HOLD_MS: i64 = 500;

/// 保存 `0x2111A` 呼叫端專屬的 `0x1C4CC -> 0x1CAC7 -> 數值結果` 契約。描述子值維持 raw。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand32TailSchedule {
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

/// 效果描述子 0x40..0x4B（含兩個 toggle）全部檢查；數字庫只檢查長度要涵蓋 0x76。
pub fn build_native_command32_tail_schedule(
    effect: &[LMI1Entry],
    digits: &[LMI1Entry],
) -> Result<NativeCommand32TailSchedule> {
    if effect.len() as i64 <= NATIVE_COMMAND32_TAIL_TOGGLE_B {
        return Err(FdotherError::fixed("native command32 tail effect descriptors unavailable"));
    }
    for index in NATIVE_COMMAND32_TAIL_EFFECT_START..=NATIVE_COMMAND32_TAIL_TOGGLE_B {
        if !lmi1_entry_valid(&effect[index as usize]) {
            return Err(FdotherError::message(format!("native command32 tail effect descriptor {index:#x} invalid")));
        }
    }
    if digits.len() <= 0x76 {
        return Err(FdotherError::fixed("native command32 tail digit descriptors unavailable"));
    }
    Ok(NativeCommand32TailSchedule {
        effect_resource: NATIVE_COMMAND32_TAIL_EFFECT_RESOURCE,
        effect_start: NATIVE_COMMAND32_TAIL_EFFECT_START,
        effect_frames: NATIVE_COMMAND32_TAIL_EFFECT_FRAMES,
        sound_resource: NATIVE_COMMAND32_TAIL_SOUND_RESOURCE,
        effect_sample: NATIVE_COMMAND32_TAIL_EFFECT_SAMPLE,
        effect_delay_ticks: 1,
        toggle_a: NATIVE_COMMAND32_TAIL_TOGGLE_A,
        toggle_b: NATIVE_COMMAND32_TAIL_TOGGLE_B,
        toggle_pairs: NATIVE_COMMAND32_TAIL_TOGGLE_PAIRS,
        toggle_delay_ms: NATIVE_COMMAND32_TAIL_TOGGLE_DELAY_MS,
        digit_resource: NATIVE_COMMAND32_TAIL_DIGIT_RESOURCE,
        digit_bias: NATIVE_COMMAND32_TAIL_DIGIT_BIAS,
        digit_frames: NATIVE_COMMAND32_TAIL_DIGIT_FRAMES,
        digit_delay_ms: NATIVE_COMMAND32_TAIL_DIGIT_DELAY_MS,
        digit_hold_ms: NATIVE_COMMAND32_TAIL_DIGIT_HOLD_MS,
        digit_vertical: NATIVE_DIGIT_VERTICAL,
    })
}
