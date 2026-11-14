//! 移植自 Go `battlepresent/native_command0.go`。

use super::{GoResult, Result, at, err, len64};
use fd2_formats::figani::{
    self, Animation, Frame, NATIVE_COMMAND0_EFFECT_FRAME_COUNT, NATIVE_COMMAND0_PRESENTATION_FRAMES,
    NativeCommand0PresentationSchedule,
};

pub(super) const NATIVE_COMMAND0_SURFACE_SIZE: i64 = 320 * 200;
pub(super) const NATIVE_COMMAND0_WORK_STRIDE: i64 = 640;
/// sub_2A6BD: 0x2A300 bytes / 640-byte stride (#151)
pub(super) const NATIVE_COMMAND0_WORK_HEIGHT: i64 = 270;
pub(super) const NATIVE_COMMAND0_VIEWPORT_BASE: i64 = 19360;

/// 把 320×200 底圖放進 640 stride 工作畫面的 0x4ba0 起點（Go 各合成函式內同一段迴圈）。呼叫端已確認
/// `base` 長度。
pub(super) fn command0_work_from(base: &[u8]) -> Vec<u8> {
    let mut work = vec![0u8; at(NATIVE_COMMAND0_WORK_STRIDE * NATIVE_COMMAND0_WORK_HEIGHT)];
    for y in 0..200usize {
        let start = at(NATIVE_COMMAND0_VIEWPORT_BASE) + y * at(NATIVE_COMMAND0_WORK_STRIDE);
        work[start..start + 320].copy_from_slice(&base[y * 320..(y + 1) * 320]);
    }
    work
}

/// 取出 0x4ba0 起點的 320×200 viewport（Go 各合成函式結尾同一段迴圈）。
pub(super) fn command0_present(work: &[u8]) -> Vec<u8> {
    let mut out = vec![0u8; at(NATIVE_COMMAND0_SURFACE_SIZE)];
    for y in 0..200usize {
        let start = at(NATIVE_COMMAND0_VIEWPORT_BASE) + y * at(NATIVE_COMMAND0_WORK_STRIDE);
        out[y * 320..(y + 1) * 320].copy_from_slice(&work[start..start + 320]);
    }
    out
}

/// ComposeNativeCommand0TargetFrame reproduces sub_26152's mode4 -> target
/// FIGANI -> mode5 ordering on the original 640-stride work surface, then
/// extracts the 320x200 viewport rooted at 0x4ba0. It is a
/// caller-specific compositor primitive; background, panel, target animation,
/// audio and numeric publication remain owned by the 0x2A6BD presentation job.
///
/// Go：`native_command0.go:22`
pub fn compose_native_command0_target_frame(
    base: &[u8],
    target: &Frame,
    effect: Option<&Animation>,
    schedule: NativeCommand0PresentationSchedule,
    step: i64,
) -> Result<Vec<u8>> {
    let Some(effect) = effect.filter(|e| {
        len64(base) == NATIVE_COMMAND0_SURFACE_SIZE
            && len64(&e.frames) == NATIVE_COMMAND0_EFFECT_FRAME_COUNT
            && step >= 0
            && step < schedule.frames
            && schedule.frames == NATIVE_COMMAND0_PRESENTATION_FRAMES
    }) else {
        return Err(err("battlepresent: incomplete command0 target frame input"));
    };
    let mut work = command0_work_from(base);
    let draw_layer = |work: &mut [u8], layer: usize| -> Result<()> {
        let (frame_index, ok) = figani::native_command0_effect_frame(step, layer as i64);
        if !ok {
            return Ok(());
        }
        let frame = &effect.frames[at(frame_index)];
        blit_command0_work_frame(
            work,
            frame,
            schedule.side_x_shift.wrapping_add(schedule.x_offsets[layer]),
            schedule.y_offsets[layer],
        )
    };
    for (layer, &flag) in schedule.layer_flags.iter().enumerate() {
        if flag == 1 {
            draw_layer(&mut work, layer).wrap(format!("battlepresent: command0 back layer {layer}: "))?;
        }
    }
    blit_command0_work_frame(&mut work, target, 0, 0).wrap("battlepresent: command0 target: ")?;
    for (layer, &flag) in schedule.layer_flags.iter().enumerate() {
        if flag == 0 {
            draw_layer(&mut work, layer).wrap(format!("battlepresent: command0 front layer {layer}: "))?;
        }
    }
    Ok(command0_present(&work))
}

/// Go：`native_command0.go:65`
pub(super) fn blit_command0_work_frame(dst: &mut [u8], frame: &Frame, extra_x: i64, extra_y: i64) -> Result<()> {
    if len64(dst) != NATIVE_COMMAND0_WORK_STRIDE * NATIVE_COMMAND0_WORK_HEIGHT
        || frame.width <= 0
        || frame.height <= 0
        || len64(&frame.pixels) != frame.width.wrapping_mul(frame.height)
        || frame.mask.len() != frame.pixels.len()
    {
        return Err(err("malformed indexed frame"));
    }
    let (origin_x, origin_y) = (
        NATIVE_COMMAND0_VIEWPORT_BASE % NATIVE_COMMAND0_WORK_STRIDE,
        NATIVE_COMMAND0_VIEWPORT_BASE / NATIVE_COMMAND0_WORK_STRIDE,
    );
    let (x0, y0) =
        (origin_x.wrapping_add(frame.x).wrapping_add(extra_x), origin_y.wrapping_add(frame.y).wrapping_add(extra_y));
    if x0 < 0
        || y0 < 0
        || x0.wrapping_add(frame.width) > NATIVE_COMMAND0_WORK_STRIDE
        || y0.wrapping_add(frame.height) > NATIVE_COMMAND0_WORK_HEIGHT
    {
        return Err(err(format!("work frame bounds ({x0},{y0} {}x{})", frame.width, frame.height)));
    }
    for y in 0..frame.height {
        for x in 0..frame.width {
            let source = at(y * frame.width + x);
            if frame.mask[source] != 0 {
                dst[at((y0 + y) * NATIVE_COMMAND0_WORK_STRIDE + x0 + x)] = frame.pixels[source];
            }
        }
    }
    Ok(())
}

#[cfg(test)]
mod native_command0_test;
