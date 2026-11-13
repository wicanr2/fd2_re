//! 移植自 Go `native_command35_tail.go`。

use super::native_ai_item_type21_tail::NATIVE_DIGIT_VERTICAL;
use super::{FdotherError, Result};

/// 保存三次 `0x22D1B` 呼叫消費的一列 raw 呈現資料。幀索引是原生從零起算的計數。
///
/// Go 版沒有額外取樣幀的列，`ExtraSampleFrameIndices` 是 nil slice，Rust 是空 `Vec`；長度都是 0。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand35StageSchedule {
    pub command_id: i64,
    pub effect_start: i64,
    pub effect_frames: i64,
    pub effect_sample: i64,
    pub extra_sample_frame_indices: Vec<i64>,
    pub extra_sample: i64,
    pub mask_index: i64,
    pub mask_pairs: i64,
    pub digit_bias: i64,
    pub digit_frames: i64,
    pub digit_hold_milliseconds: i64,
    pub digit_vertical: [i64; 25],
}

/// 驗證條件以固定資料不可能成立，Go 版仍保留這條錯誤分支；這裡照樣保留。
pub fn build_native_command35_tail_schedule() -> Result<Vec<NativeCommand35StageSchedule>> {
    let row = |command_id, effect_start, effect_frames, effect_sample, extra: &[i64], extra_sample, mask_index| {
        NativeCommand35StageSchedule {
            command_id,
            effect_start,
            effect_frames,
            effect_sample,
            extra_sample_frame_indices: extra.to_vec(),
            extra_sample,
            mask_index,
            ..Default::default()
        }
    };
    let mut rows = vec![
        row(26, 0x8a, 9, 3, &[], 0, 0xc0),
        row(22, 0xaa, 13, 3, &[7], 3, 0x23),
        row(27, 0x9e, 12, 2, &[], 0, 0xc0),
    ];
    for (index, r) in rows.iter_mut().enumerate() {
        r.mask_pairs = 5;
        r.digit_bias = 0x5e;
        r.digit_frames = 22;
        r.digit_hold_milliseconds = 500;
        r.digit_vertical = NATIVE_DIGIT_VERTICAL;
        if r.effect_frames <= 0 || r.effect_sample <= 0 || r.effect_start < 0 {
            return Err(FdotherError::message(format!("native command35 malformed stage {index}")));
        }
    }
    Ok(rows)
}
