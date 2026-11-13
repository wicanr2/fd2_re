//! 移植自 Go `native_command_heal_presentation.go`。

use super::native_flow_error::{NativeFlowError, NativeFlowResult};

/// 一次 `0x21EB1` 呼叫共用 `0x22046` 索引合成器。`lut_index` 保持 raw，因為 FDOTHER #3 是
/// 調色盤索引重映射表，不是具名的法術動畫封存。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandHealPresentationFrame {
    pub lut_index: i64,
    pub radius: i64,
    pub frame_delay_ms: i64,
}

/// 保存直接觀察到的兩個 `0x21EB1` 迴圈。`mid_delay_ms` 接在 descriptor 9..1 之後；
/// `tail_delay_ms` 接在最後的 descriptor 9 與 `0x11CAC(0)` 呈現之後。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandHealPresentationSchedule {
    pub frames: Vec<NativeCommandHealPresentationFrame>,
    pub mid_frame: i64,
    pub mid_delay_ms: i64,
    pub tail_delay_ms: i64,
    pub sample_index: i64,
}

/// 重現 command 13..16 包裝函式的常值與迴圈邊界。第一個迴圈走訪 FDOTHER #3 LUT 9..1 並
/// 推進半徑；第二個迴圈保持最後的半徑，走訪 LUT 3..9。
pub fn build_native_command_heal_presentation_schedule(
    command_id: i64,
) -> NativeFlowResult<NativeCommandHealPresentationSchedule> {
    let (start, step) = match command_id {
        13 => (1, 2),
        14 => (2, 4),
        15 => (8, 4),
        16 => (6, 6),
        _ => return Err(NativeFlowError::HealPresentation(command_id)),
    };
    let mut frames = Vec::with_capacity(16);
    let mut radius = start;
    for descriptor in (1..=9).rev() {
        frames.push(NativeCommandHealPresentationFrame { lut_index: descriptor, radius, frame_delay_ms: 5 });
        radius += step;
    }
    radius -= step;
    for descriptor in 3..10 {
        frames.push(NativeCommandHealPresentationFrame { lut_index: descriptor, radius, frame_delay_ms: 5 });
    }
    Ok(NativeCommandHealPresentationSchedule {
        frames,
        mid_frame: 9,
        mid_delay_ms: 200,
        tail_delay_ms: 200,
        sample_index: 11,
    })
}

#[cfg(test)]
mod tests;
