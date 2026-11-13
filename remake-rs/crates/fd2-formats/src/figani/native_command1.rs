//! 指令 1（`0x262EF`）的演出排程。移植自 Go `native_command1.go`。

use super::{Animation, FiganiError, frame_shape_ok, msg};

pub const NATIVE_COMMAND1_EFFECT_FRAME_COUNT: i64 = 30;
pub const NATIVE_COMMAND1_CHANNEL_COUNT: usize = 8;
pub const NATIVE_COMMAND1_ANCHOR_X: i64 = 0x50;
pub const NATIVE_COMMAND1_PRESENTATION_FRAMES: i64 = 31;

pub(crate) const NATIVE_COMMAND1_X_OFFSETS: [i64; NATIVE_COMMAND1_CHANNEL_COUNT] = [-59, -39, 0, 39, 55, 39, 0, -39];
pub(crate) const NATIVE_COMMAND1_Y_OFFSETS: [i64; NATIVE_COMMAND1_CHANNEL_COUNT] = [-10, -24, -30, -24, -10, 4, 10, 4];

/// 保存 `0x262EF` 的 mode 3/4/5 純繪製契約。直接 sample1 與 numeric marker 由
/// [`native_command1_target_markers`] 分開保存。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand1PresentationSchedule {
    pub effect_resource: i64,
    pub side_x_shift: i64,
    pub anchor_x: i64,
    pub x_offsets: [i64; NATIVE_COMMAND1_CHANNEL_COUNT],
    pub y_offsets: [i64; NATIVE_COMMAND1_CHANNEL_COUNT],
}

pub fn build_native_command1_presentation_schedule(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<NativeCommand1PresentationSchedule, FiganiError> {
    let (resource, shift) = if raw_side == 0 { (21, 148) } else { (19, 0) };
    let Some(effect) = effect.filter(|e| {
        e.header_byte1 == 0
            && e.header_byte2 as i64 == NATIVE_COMMAND1_EFFECT_FRAME_COUNT
            && e.header_byte4 == 0
            && e.frames.len() as i64 == NATIVE_COMMAND1_EFFECT_FRAME_COUNT
    }) else {
        return Err(msg(format!("figani: command1 FDOTHER #{resource} signature mismatch")));
    };
    for (index, frame) in effect.frames.iter().enumerate() {
        if !frame_shape_ok(frame) {
            return Err(msg(format!("figani: command1 FDOTHER #{resource} frame {index} malformed")));
        }
    }
    Ok(NativeCommand1PresentationSchedule {
        effect_resource: resource,
        side_x_shift: shift,
        anchor_x: NATIVE_COMMAND1_ANCHOR_X,
        x_offsets: NATIVE_COMMAND1_X_OFFSETS,
        y_offsets: NATIVE_COMMAND1_Y_OFFSETS,
    })
}

/// 回傳 mode 3 寫入的八個交錯 counter。
pub fn native_command1_counters(step: i64) -> Result<[i64; NATIVE_COMMAND1_CHANNEL_COUNT], FiganiError> {
    if !(0..NATIVE_COMMAND1_PRESENTATION_FRAMES).contains(&step) {
        return Err(msg(format!("figani: command1 step {step} is unavailable")));
    }
    let mut counters = [0i64; NATIVE_COMMAND1_CHANNEL_COUNT];
    for (slot, counter) in counters.iter_mut().enumerate() {
        *counter = step - 2 * slot as i64;
    }
    Ok(counters)
}

/// 保存 mode 4/5 的兩組 frame／index 交換規則，回傳 `(frame, offset_index, ok)`。
/// ok=false 表示 counter 不在原版的 [0,15) 繪製範圍。
pub fn native_command1_mode_frame(mode: i64, slot: i64, counter: i64) -> (i64, i64, bool) {
    if slot < 0 || slot >= NATIVE_COMMAND1_CHANNEL_COUNT as i64 || !(0..15).contains(&counter) {
        return (0, 0, false);
    }
    match mode {
        4 if slot < 4 => (counter, slot, true),
        4 => (counter + 15, slot - 4, true),
        5 if slot < 4 => (counter + 15, slot + 4, true),
        5 => (counter, slot, true),
        _ => (0, 0, false),
    }
}

/// 保存 mode 5 遞增八個交錯 counter 後的直接消費端：等於 9 時回傳 numeric marker，等於 5 時
/// 播放 sample1。回傳 `(numeric, sample)`。
pub fn native_command1_target_markers(step: i64) -> Result<(bool, bool), FiganiError> {
    let counters = native_command1_counters(step)?;
    let numeric = counters.iter().any(|c| c + 1 == 9);
    let sample = counters.iter().any(|c| c + 1 == 5);
    Ok((numeric, sample))
}
