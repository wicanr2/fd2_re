//! 移植自 Go `preparation_timer.go`。

use super::native_flow_error::{NativeFlowResult, at, fixed};

/// 記錄一次 `0x1a941` 的精確轉移。保留來源偏移，因為接下來的原版繪圖端會用由計數器
/// 索引推得的資料；這裡不推論狀態或效果名稱。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePreparationTimerExpiry {
    pub record_index: i64,
    pub counter_index: i64,
    pub counter_offset: i64,
    pub visual_source_offset: i64,
}

/// 在 0x50 位元組記錄上原地執行 raw `0x1a941` 掃描，只回傳由 1 變 0 的計數器。
///
/// `record_count*0x50` 與 Go 一樣以 64 位元繞回；繞回成非負小值騙過檢查時，之後的索引越界
/// panic，與 Go 對應。
pub fn tick_native_preparation_timers(
    records: &mut [u8],
    record_count: i64,
    selector: i64,
) -> NativeFlowResult<Vec<NativePreparationTimerExpiry>> {
    if record_count < 0 || !(0..=0xff).contains(&selector) {
        return fixed("fdother: invalid preparation timer arguments");
    }
    const RECORD_SIZE: i64 = 0x50;
    const COUNTER_BASE: i64 = 0x22;
    const COUNTER_COUNT: i64 = 6;
    let need = record_count.wrapping_mul(RECORD_SIZE);
    if need < 0 || (records.len() as i64) < need {
        return fixed("fdother: preparation timer records are truncated");
    }
    let mut expired = Vec::new();
    for record_index in 0..record_count {
        let base = record_index * RECORD_SIZE;
        if records[at(base + 6)] as i64 != selector || records[at(base + 5)] & 1 != 0 {
            continue;
        }
        for counter_index in 0..COUNTER_COUNT {
            let offset = at(base + COUNTER_BASE + counter_index);
            if records[offset] == 0 {
                continue;
            }
            records[offset] -= 1;
            if records[offset] == 0 {
                expired.push(NativePreparationTimerExpiry {
                    record_index,
                    counter_index,
                    counter_offset: COUNTER_BASE + counter_index,
                    visual_source_offset: 0x1e1 + counter_index,
                });
            }
        }
    }
    Ok(expired)
}

#[cfg(test)]
mod tests;
