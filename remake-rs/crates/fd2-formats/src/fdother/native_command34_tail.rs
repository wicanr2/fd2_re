//! 移植自 Go `native_command34_tail.go`。

use super::native_flow_error::{NativeFlowError, NativeFlowResult};

/// 保存 `0x22721`、`0x22866`、`0x22997` 選出的三列 raw 呈現資料之一。名稱刻意保持 raw。
///
/// Go 版第一列的 `ExtraSampleFrameIndices` 是 nil slice，Rust 是空 `Vec`；兩者長度都是 0。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand34StageSchedule {
    pub command_id: i64,
    pub effect_start: i64,
    pub effect_frames: i64,
    pub effect_sample: i64,
    pub extra_sample_frame_indices: Vec<i64>,
    pub mask_index: i64,
    pub mask_pairs: i64,
    pub digit_bias: i64,
    pub digit_frames: i64,
    pub digit_hold_milliseconds: i64,
    pub digit_vertical: [i64; 25],
}

/// 回傳 command 34 在 `0x27FC9` 共用段之後使用的 17→18→19 順序。
pub fn build_native_command34_tail_schedule() -> NativeFlowResult<Vec<NativeCommand34StageSchedule>> {
    let vertical = [15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15];
    let row = |command_id, effect_start, effect_frames, effect_sample, extra: &[i64], mask_index| {
        NativeCommand34StageSchedule {
            command_id,
            effect_start,
            effect_frames,
            effect_sample,
            extra_sample_frame_indices: extra.to_vec(),
            mask_index,
            ..Default::default()
        }
    };
    let mut rows =
        vec![row(17, 0xb7, 8, 6, &[], 0x92), row(18, 0x7e, 12, 7, &[4], 0x48), row(19, 0x93, 11, 8, &[3, 6], 0xd8)];
    for (index, r) in rows.iter_mut().enumerate() {
        r.mask_pairs = 5;
        r.digit_bias = 0x69;
        r.digit_frames = 22;
        r.digit_hold_milliseconds = 500;
        r.digit_vertical = vertical;
        if r.effect_start < 0 || r.effect_frames <= 0 || r.effect_sample <= 0 {
            return Err(NativeFlowError::Command34Stage(index));
        }
    }
    Ok(rows)
}

#[cfg(test)]
mod tests;
