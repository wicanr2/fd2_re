//! 移植自 Go `native_command20_22_tail.go`。

use super::native_ai_item_type21_tail::NATIVE_DIGIT_VERTICAL;
use super::{FdotherError, NativeCommand34StageSchedule, Result};

/// 保存已還原的 `0x22A85..0x22E41` 處理器家族消費的六列 command 索引資料。回傳型別與位元組相同的
/// 尾段原語共用；它的歷史名稱不代表歸 command 34 擁有。
///
/// 驗證條件以固定資料不可能成立，Go 版仍保留這條錯誤分支；這裡照樣保留。
pub fn build_native_command2022_tail_schedule() -> Result<Vec<NativeCommand34StageSchedule>> {
    let row = |command_id, effect_start, effect_frames, effect_sample, extra: &[i64], mask_index, digit_bias| {
        NativeCommand34StageSchedule {
            command_id,
            effect_start,
            effect_frames,
            effect_sample,
            extra_sample_frame_indices: extra.to_vec(),
            mask_index,
            digit_bias,
            ..Default::default()
        }
    };
    let mut rows = vec![
        row(20, 0xcc, 13, 4, &[], 0xc0, 0x69),
        row(21, 0xd9, 13, 4, &[], 0xc0, 0x69),
        row(22, 0xaa, 13, 3, &[7], 0x23, 0x5e),
        row(25, 0xbf, 13, 5, &[], 0xc0, 0x5e),
        row(26, 0x8a, 9, 3, &[], 0xc0, 0x5e),
        row(27, 0x9e, 12, 2, &[], 0xc0, 0x5e),
    ];
    for (index, r) in rows.iter_mut().enumerate() {
        r.mask_pairs = 5;
        r.digit_frames = 22;
        r.digit_hold_milliseconds = 500;
        r.digit_vertical = NATIVE_DIGIT_VERTICAL;
        if r.effect_start < 0 || r.effect_frames <= 0 || r.effect_sample <= 0 || r.digit_bias < 0 {
            return Err(FdotherError::message(format!("native command tail malformed stage {index}")));
        }
    }
    Ok(rows)
}
