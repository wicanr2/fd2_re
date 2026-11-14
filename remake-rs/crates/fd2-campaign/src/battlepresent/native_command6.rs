//! 移植自 Go `battlepresent/native_command6.go`。

use super::native_command0::{
    NATIVE_COMMAND0_SURFACE_SIZE, NATIVE_COMMAND0_WORK_HEIGHT, NATIVE_COMMAND0_WORK_STRIDE, blit_command0_work_frame,
    command0_present, command0_work_from,
};
use super::{GoResult, Result, at, err, len64};
use fd2_formats::figani::{
    Animation, Frame, NATIVE_COMMAND6_EFFECT_FRAME_COUNT, NativeCommand6Layer, NativeCommand6OrbitFrame,
    NativeCommand6TargetDisplayFrame, NativeCommand6TargetFrame, NativeCommand6TransitionFrame,
};
use sha2::{Digest, Sha256};

/// #154：只識別固定 FDOTHER #32 的完整 typed bytes，不依檔名授權越界。
/// isolation_spec 記錄與原始資源雜湊的對照；這不是原版 heap 的重製。
///
/// Go：`native_command6.go:15`
fn command6_bank32_fingerprint(effect: Option<&Animation>) -> bool {
    let Some(effect) = effect.filter(|e| len64(&e.frames) == NATIVE_COMMAND6_EFFECT_FRAME_COUNT) else {
        return false;
    };
    let mut h = Sha256::new();
    h.update(b"fd2-command6-typed-v1\x00");
    h.update([effect.header_byte1, effect.header_byte2, effect.header_byte4]);
    h.update((effect.frames.len() as u32).to_le_bytes());
    for frame in &effect.frames {
        if frame.width <= 0
            || frame.height <= 0
            || len64(&frame.pixels) != frame.width.wrapping_mul(frame.height)
            || frame.mask.len() != frame.pixels.len()
        {
            return false;
        }
        for value in [frame.x, frame.y, frame.width, frame.height, frame.delay] {
            if value as i32 as i64 != value {
                return false;
            }
            h.update((value as i32 as u32).to_le_bytes());
        }
        h.update([frame.raw_byte4, frame.raw_byte5, frame.raw_byte7]);
        h.update(&frame.pixels);
        h.update(&frame.mask);
    }
    let sum: String = h.finalize().iter().map(|b| format!("{b:02x}")).collect();
    sum == "3ebd12dfb0cb4fec35e968551672d6c644e1475dc2ac8a3353b523e3d05005d3"
}

/// 保留唯一已觀測的 -422 opaque 寫入於獨立前置列，其餘界限沿嚴格 blit。
///
/// Go：`native_command6.go:45`
fn blit_command6_effect_work_frame(
    work: &mut [u8],
    prefix: &mut [u8],
    effect: &Animation,
    layer: NativeCommand6Layer,
) -> Result<()> {
    let frame = &effect.frames[at(layer.frame)];
    let y0 = 30i64.wrapping_add(frame.y).wrapping_add(layer.y);
    if y0 >= 0 {
        return blit_command0_work_frame(work, frame, layer.x, layer.y);
    }
    let x0 = 160i64.wrapping_add(frame.x).wrapping_add(layer.x);
    if len64(work) != NATIVE_COMMAND0_WORK_STRIDE * NATIVE_COMMAND0_WORK_HEIGHT
        || len64(prefix) != NATIVE_COMMAND0_WORK_STRIDE
        || layer.mode != 5
        || layer.channel != 2
        || layer.frame != 9
        || !layer.secondary
        || x0 != 141
        || y0 != -1
        || frame.width != 143
        || frame.height != 111
        || !command6_bank32_fingerprint(Some(effect))
    {
        return Err(err(format!("unreviewed command6 signed work frame ({x0},{y0} {}x{})", frame.width, frame.height)));
    }
    let mut negative = 0;
    for x in 0..frame.width {
        if frame.mask[at(x)] != 0 {
            if -NATIVE_COMMAND0_WORK_STRIDE + x0 + x != -422 {
                return Err(err("unreviewed command6 prefix write"));
            }
            negative += 1;
        }
    }
    if negative != 1 {
        return Err(err("command6 prefix writer unavailable"));
    }
    for y in 0..frame.height {
        for x in 0..frame.width {
            let idx = at(y * frame.width + x);
            if frame.mask[idx] == 0 {
                continue;
            }
            let logical = (y0 + y) * NATIVE_COMMAND0_WORK_STRIDE + x0 + x;
            if logical < 0 {
                prefix[at(len64(prefix) + logical)] = frame.pixels[idx];
            } else {
                work[at(logical)] = frame.pixels[idx];
            }
        }
    }
    Ok(())
}

/// ComposeNativeCommand6TargetFrame draws one typed mode4→target→mode5 plan
/// onto the native work surface, preserving caller actor and packed target.
/// It never advances or publishes battle state.
///
/// Go：`native_command6.go:91`
pub fn compose_native_command6_target_frame(
    base: &[u8],
    actor: &Frame,
    target: &Frame,
    effect: Option<&Animation>,
    frame: &NativeCommand6TargetFrame,
    display: NativeCommand6TargetDisplayFrame,
) -> Result<Vec<u8>> {
    let Some(effect) = effect.filter(|e| {
        len64(base) == NATIVE_COMMAND0_SURFACE_SIZE && len64(&e.frames) == NATIVE_COMMAND6_EFFECT_FRAME_COUNT
    }) else {
        return Err(err("battlepresent: incomplete command6 target frame input"));
    };
    let mut work = command0_work_from(base);
    let mut prefix = vec![0u8; at(NATIVE_COMMAND0_WORK_STRIDE)];
    let draw = |work: &mut [u8], prefix: &mut [u8], layers: &[NativeCommand6Layer]| -> Result<()> {
        for &layer in layers {
            if layer.frame < 0 || layer.frame >= len64(&effect.frames) {
                return Err(err(format!("battlepresent: command6 frame {} unavailable", layer.frame)));
            }
            blit_command6_effect_work_frame(work, prefix, effect, layer)
                .wrap(format!("battlepresent: command6 mode{} channel{}: ", layer.mode, layer.channel))?;
        }
        Ok(())
    };
    draw(&mut work, &mut prefix, &frame.mode4)?;

    blit_command0_work_frame(&mut work, actor, 0, 0).wrap("battlepresent: command6 actor: ")?;
    let shaded;
    let mut target = target;
    if display.shader != -1 {
        let shade = display.shader >> 8;
        if display.shader & 0xff != 0xb0 || !(2..=8).contains(&shade) {
            return Err(err("battlepresent: command6 packed shader unavailable"));
        }
        if target.width <= 0
            || target.height <= 0
            || len64(&target.pixels) != target.width.wrapping_mul(target.height)
            || target.mask.len() != target.pixels.len()
        {
            return Err(err("battlepresent: command6 target malformed"));
        }
        let mut copy = target.clone();
        for value in &mut copy.pixels {
            *value = ((*value as i64 + shade) & 7) as u8 + 0xb0;
        }
        shaded = copy;
        target = &shaded;
    }
    blit_command0_work_frame(&mut work, target, display.offset_x, display.offset_y)
        .wrap("battlepresent: command6 target: ")?;
    draw(&mut work, &mut prefix, &frame.mode5)?;
    Ok(command0_present(&work))
}

/// ComposeNativeCommand6OrbitFrame preserves the caller's first-mode,
/// actor-effect, second-mode order for one front or tail iteration.
///
/// Go：`native_command6.go:144`
pub fn compose_native_command6_orbit_frame(
    base: &[u8],
    actor_effect: Option<&Animation>,
    target: Option<&Frame>,
    effect: Option<&Animation>,
    frame: &NativeCommand6OrbitFrame,
) -> Result<Vec<u8>> {
    let (Some(actor_effect), Some(effect)) = (actor_effect, effect) else {
        return Err(err("battlepresent: incomplete command6 orbit frame input"));
    };
    if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE
        || actor_effect.frames.is_empty()
        || len64(&effect.frames) != NATIVE_COMMAND6_EFFECT_FRAME_COUNT
    {
        return Err(err("battlepresent: incomplete command6 orbit frame input"));
    }
    let mut work = command0_work_from(base);
    let draw = |work: &mut [u8], layers: &[NativeCommand6Layer]| -> Result<()> {
        for layer in layers {
            if layer.frame != 4 {
                return Err(err(format!("battlepresent: command6 orbit frame {} unavailable", layer.frame)));
            }
            blit_command0_work_frame(work, &effect.frames[at(layer.frame)], layer.x, layer.y)
                .wrap(format!("battlepresent: command6 mode{} channel{}: ", layer.mode, layer.channel))?;
        }
        Ok(())
    };
    draw(&mut work, &frame.first)?;
    blit_command0_work_frame(&mut work, &actor_effect.frames[actor_effect.frames.len() - 1], 0, 0)
        .wrap("battlepresent: command6 orbit actor: ")?;
    if frame.draw_target {
        let Some(target) = target else {
            return Err(err("battlepresent: command6 tail target unavailable"));
        };
        blit_command0_work_frame(&mut work, target, 0, 0).wrap("battlepresent: command6 orbit target: ")?;
    }
    draw(&mut work, &frame.second)?;
    Ok(command0_present(&work))
}

/// ComposeNativeCommand6TransitionFrame reproduces one command6 sub_2BA22
/// frame: mode4, actor-effect last frame, displaced target idle, then mode5.
///
/// Go：`native_command6.go:189`
pub fn compose_native_command6_transition_frame(
    base: &[u8],
    actor_effect: Option<&Animation>,
    target_idle: Option<&Animation>,
    effect: Option<&Animation>,
    frame: &NativeCommand6TransitionFrame,
) -> Result<Vec<u8>> {
    let (Some(actor_effect), Some(target_idle), Some(effect)) = (actor_effect, target_idle, effect) else {
        return Err(err("battlepresent: incomplete command6 transition input"));
    };
    if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE
        || actor_effect.frames.is_empty()
        || target_idle.frames.is_empty()
        || len64(&effect.frames) != NATIVE_COMMAND6_EFFECT_FRAME_COUNT
    {
        return Err(err("battlepresent: incomplete command6 transition input"));
    }
    let mut work = command0_work_from(base);
    let mut prefix = vec![0u8; at(NATIVE_COMMAND0_WORK_STRIDE)];
    let draw = |work: &mut [u8], prefix: &mut [u8], layers: &[NativeCommand6Layer]| -> Result<()> {
        for &layer in layers {
            if layer.frame < 0 || layer.frame >= len64(&effect.frames) {
                return Err(err(format!(
                    "battlepresent: command6 transition effect frame {} unavailable",
                    layer.frame
                )));
            }
            blit_command6_effect_work_frame(work, prefix, effect, layer)?;
        }
        Ok(())
    };
    draw(&mut work, &mut prefix, &frame.mode4)?;
    blit_command0_work_frame(&mut work, &actor_effect.frames[actor_effect.frames.len() - 1], 0, 0)
        .wrap("battlepresent: command6 transition actor: ")?;
    blit_command0_work_frame(&mut work, &target_idle.frames[0], frame.target_offset_x, 0)
        .wrap("battlepresent: command6 transition target: ")?;
    draw(&mut work, &mut prefix, &frame.mode5)?;
    Ok(command0_present(&work))
}

#[cfg(test)]
mod native_command6_test;
