//! 移植自 Go `native_command10_12.go`。

use super::native_flow_error::{NativeFlowError, NativeFlowResult};

pub const NATIVE_COMMAND10_TO12_SOUND_RESOURCE: i64 = 80;
pub const NATIVE_COMMAND10_TO12_PRELUDE_SAMPLE: i64 = 2;
pub const NATIVE_COMMAND10_TO12_MAIN_SAMPLE: i64 = 13;
pub const NATIVE_COMMAND10_TO12_FRAMES: i64 = 60;
pub const NATIVE_COMMAND10_TO12_DELAY_MS: i64 = 10;

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand10To12Prelude {
    pub enabled: bool,
    pub initial_radius: i64,
    pub radius_step: i64,
    pub repeat: i64,
    pub sample: i64,
}

/// 只保存 `0x21527/0x2185F/0x21A9E -> 0x21548` 已證實的 raw 表與呼叫排程。等同
/// `0x1F558` 的指標格取樣與狀態原子發布仍由繪圖端負責。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand10To12Schedule {
    pub command_id: i64,
    pub sound_resource: i64,
    pub prelude: NativeCommand10To12Prelude,
    pub x_origins: [i64; 3],
    pub y_origins: [i64; 3],
    pub sampling_increments: [i64; 3],
    pub surface_cycle: [i64; 4],
    pub frames: i64,
    pub delay_ms: i64,
    pub main_sample: i64,
    pub main_sample_frames: [i64; 8],
    pub result_resource: i64,
    pub result_digit_bias: i64,
    pub result_miss_descriptors: [i64; 4],
    pub result_frames: i64,
    pub result_hold_ms: i64,
    pub result_vertical: [i64; 25],
}

pub fn build_native_command10_to12_schedule(command_id: i64) -> NativeFlowResult<NativeCommand10To12Schedule> {
    if !(10..=12).contains(&command_id) {
        return Err(NativeFlowError::Command10To12(command_id));
    }
    let mut prelude = NativeCommand10To12Prelude::default();
    if command_id == 11 {
        prelude = NativeCommand10To12Prelude {
            enabled: true,
            initial_radius: 15,
            radius_step: 10,
            repeat: 10,
            sample: NATIVE_COMMAND10_TO12_PRELUDE_SAMPLE,
        };
    }
    if command_id == 12 {
        prelude = NativeCommand10To12Prelude {
            enabled: true,
            initial_radius: 30,
            radius_step: 16,
            repeat: 10,
            sample: NATIVE_COMMAND10_TO12_PRELUDE_SAMPLE,
        };
    }
    Ok(NativeCommand10To12Schedule {
        command_id,
        sound_resource: NATIVE_COMMAND10_TO12_SOUND_RESOURCE,
        prelude,
        x_origins: [128, 0, -128],
        y_origins: [128, 0, 128],
        sampling_increments: [131, 128, 125],
        surface_cycle: [0, 1, 2, 3],
        frames: NATIVE_COMMAND10_TO12_FRAMES,
        delay_ms: NATIVE_COMMAND10_TO12_DELAY_MS,
        main_sample: NATIVE_COMMAND10_TO12_MAIN_SAMPLE,
        main_sample_frames: [0, 6, 12, 18, 24, 30, 36, 42],
        result_resource: 5,
        result_digit_bias: 94,
        result_miss_descriptors: [116, 117, 118, 118],
        result_frames: 22,
        result_hold_ms: 500,
        result_vertical: [15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15],
    })
}

#[cfg(test)]
mod tests;
