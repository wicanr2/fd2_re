//! 移植自 Go `native_command33_tail.go`。

use super::{FdotherError, NativeCommandHealTailSchedule, Result, build_native_command_heal_tail_schedule};

/// 保存 `sub_211A4` 寫死的 command 13 呈現列，不引入 command 13 另外的 `0x21EB1` 前段。
///
/// 內部呼叫的 [`build_native_command_heal_tail_schedule`] 屬原版流程邏輯那一組，錯誤型別是
/// `NativeFlowError`；Go 版原樣回傳那個錯誤，這裡把它的文字轉成 [`FdotherError`]，`Display` 相同。
/// 以固定的 13 呼叫時那條錯誤分支不會發生，下方的比對失敗分支同樣以固定資料不可能成立。
pub fn build_native_command33_tail_schedule() -> Result<NativeCommandHealTailSchedule> {
    let schedule = build_native_command_heal_tail_schedule(13).map_err(|e| FdotherError::message(e.to_string()))?;
    if schedule.effect_resource != 6
        || schedule.effect_start != 0x39
        || schedule.effect_frames != 7
        || schedule.effect_sample != 12
        || schedule.mask_sample != 1
        || schedule.mask_index != 0xc0
        || schedule.mask_pairs != 5
        || schedule.digit_resource != 5
        || schedule.digit_bias != 0x69
        || schedule.digit_frames != 22
        || schedule.digit_hold_ms != 500
    {
        return Err(FdotherError::fixed("native command33 hard-coded command13 tail unavailable"));
    }
    Ok(schedule)
}
