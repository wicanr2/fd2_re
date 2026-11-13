//! 移植自 Go `indexed_transition_schedule.go`。

use super::native_flow_error::{NativeFlowError, NativeFlowResult};

/// 保存一次 `0x24618` 條帶 pass。`lut_index` 是 raw FDOTHER#3 選擇值（原版迴圈走訪 9..1）；
/// `radius` 是呼叫端給的第三個參數，每次加上第四個參數。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeIndexedTransitionFrame {
    pub pass: i64,
    pub lut_index: i64,
    pub radius: i64,
    pub frame_delay_ms: i64,
}

/// `0x24618` 合成器周圍的時序／資料邊界。刻意停在 LUT 載入、雙緩衝寫入、重繪與
/// VGA／Ebiten 呈現之前。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeIndexedTransitionSchedule {
    pub frames: Vec<NativeIndexedTransitionFrame>,
    pub tail_delay_ms: i64,
    pub palette_deltas: Vec<i64>,
    pub palette_delay_ms: i64,
}

/// 重現 `0x24618` 的迴圈邊界與推進：九個 pass（LUT 索引 9 降到 1），接著保持 500ms，
/// 再以每次 4ms 走調色盤差值 0..62（含）、步距 2。
pub fn build_native_indexed_transition_schedule(
    start_radius: i64,
    radius_step: i64,
) -> NativeFlowResult<NativeIndexedTransitionSchedule> {
    if !(0..=0x7fff).contains(&start_radius) || !(-0x7fff..=0x7fff).contains(&radius_step) {
        return Err(NativeFlowError::TransitionRadiusArgs { start_radius, radius_step });
    }
    let mut frames = Vec::with_capacity(9);
    let mut radius = start_radius;
    let (mut pass, mut descriptor) = (0, 9);
    while descriptor > 0 {
        frames.push(NativeIndexedTransitionFrame { pass, lut_index: descriptor, radius, frame_delay_ms: 5 });
        radius += radius_step;
        pass += 1;
        descriptor -= 1;
    }
    let palette_deltas = (0..=62).step_by(2).collect();
    Ok(NativeIndexedTransitionSchedule { frames, tail_delay_ms: 500, palette_deltas, palette_delay_ms: 4 })
}

#[cfg(test)]
mod tests;
