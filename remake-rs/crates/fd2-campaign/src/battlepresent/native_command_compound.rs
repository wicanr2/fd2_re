//! 移植自 Go `battlepresent/native_command_compound.go`。

use super::{GoResult, Result, at, err, len64};
use fd2_formats::figani::{Animation, Frame, NativeCompoundPresentationSchedule};

/// Go：`native_command_compound.go:10`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundCommonFrames {
    pub actor_slide: Vec<Vec<u8>>,
    pub effect_slide: Vec<Vec<u8>>,
    pub main: Vec<Vec<u8>>,
    pub tail: Vec<Vec<u8>>,
}

/// Go：`native_command_compound.go:17`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundActorPresent {
    pub pixels: Vec<u8>,
    pub actor_index: i64,
    pub target_index: i64,
}

fn frame_ok(frame: &Frame) -> bool {
    frame.delay > 0
        && frame.width > 0
        && frame.height > 0
        && len64(&frame.pixels) == frame.width.wrapping_mul(frame.height)
        && frame.mask.len() == frame.pixels.len()
}

/// BuildNativeCompoundActorFrames preserves the no-MP-marker class19 branch of
/// sub_2B659. Each FIGANI delay tick becomes one frame and target idle timing
/// advances across the complete actor phase.
///
/// Go：`native_command_compound.go:25`
pub fn build_native_compound_actor_frames(
    base: &[u8],
    actor_effect: Option<&Animation>,
    first_target_idle: Option<&Animation>,
    raw_side: u8,
) -> Result<Vec<Vec<u8>>> {
    let presents = build_native_compound_actor_presents(base, actor_effect, first_target_idle, raw_side)?;
    Ok(presents.into_iter().map(|p| p.pixels).collect())
}

/// BuildNativeCompoundActorPresents retains the original source indices at each
/// present boundary; the indexed renderer and optional HD renderer share timing.
///
/// Go：`native_command_compound.go:39`
pub fn build_native_compound_actor_presents(
    base: &[u8],
    actor_effect: Option<&Animation>,
    first_target_idle: Option<&Animation>,
    raw_side: u8,
) -> Result<Vec<NativeCompoundActorPresent>> {
    const FRAME_BYTES: i64 = 320 * 200;
    let (Some(actor_effect), Some(first_target_idle)) = (actor_effect, first_target_idle) else {
        return Err(err("battlepresent: native compound actor contract unavailable"));
    };
    if len64(base) != FRAME_BYTES
        || actor_effect.frames.is_empty()
        || first_target_idle.frames.is_empty()
        || actor_effect.header_byte4 == 1
    {
        return Err(err("battlepresent: native compound actor contract unavailable"));
    }
    for (index, frame) in first_target_idle.frames.iter().enumerate() {
        if !frame_ok(frame) {
            return Err(err(format!("battlepresent: compound target idle frame {index} malformed")));
        }
    }
    let (mut target_frame, mut target_repeat) = (0usize, 0i64);
    let mut frames = Vec::new();
    for (index, actor) in actor_effect.frames.iter().enumerate() {
        if !frame_ok(actor) {
            return Err(err(format!("battlepresent: compound actor frame {index} malformed")));
        }
        for _ in 0..actor.delay {
            let mut pixels = base.to_vec();
            let target = &first_target_idle.frames[target_frame];
            if raw_side != 0 {
                actor.blit_at(&mut pixels, 320).go()?;
                target.blit_at(&mut pixels, 320).go()?;
            } else {
                target.blit_at(&mut pixels, 320).go()?;
                actor.blit_at(&mut pixels, 320).go()?;
            }
            frames.push(NativeCompoundActorPresent {
                pixels,
                actor_index: index as i64,
                target_index: target_frame as i64,
            });
            target_repeat += 1;
            if target_repeat >= first_target_idle.frames[target_frame].delay {
                target_repeat = 0;
                target_frame = (target_frame + 1) % first_target_idle.frames.len();
            }
        }
    }
    if frames.is_empty() {
        return Err(err("battlepresent: native compound actor has no present boundary"));
    }
    Ok(frames)
}

/// BuildNativeCompoundCommonFrames reproduces the framebuffer-only portion of
/// 0x27FC9 after 0x29164/0x2B659 and before the command-specific state tail.
/// Audio markers and palette deltas stay in the typed schedule and are owned by
/// the Draw-acknowledged Game transaction.
///
/// Go：`native_command_compound.go:96`
pub fn build_native_compound_common_frames(
    base: &[u8],
    actor: &Frame,
    effect: Option<&Animation>,
    schedule: NativeCompoundPresentationSchedule,
) -> Result<NativeCompoundCommonFrames> {
    const FRAME_BYTES: i64 = 320 * 200;
    let Some(effect) = effect.filter(|e| {
        len64(base) == FRAME_BYTES
            && e.frames.len() >= 2
            && (32..=35).contains(&schedule.command_id)
            && schedule.effect_resource == schedule.command_id + 33
            && schedule.actor_slide_frames == 8
            && schedule.actor_slide_step_x == 20
            && schedule.effect_slide_frames == 9
            && schedule.effect_slide_step_x == 30
            && schedule.main_frame_count == len64(&e.frames) - 1
    }) else {
        return Err(err("battlepresent: native compound common contract unavailable"));
    };
    let mut out = NativeCompoundCommonFrames {
        actor_slide: Vec::with_capacity(at(schedule.actor_slide_frames)),
        effect_slide: Vec::with_capacity(at(schedule.effect_slide_frames)),
        main: Vec::with_capacity(at(schedule.main_frame_count)),
        tail: Vec::new(),
    };
    for i in 0..schedule.actor_slide_frames {
        let mut pixels = base.to_vec();
        actor
            .blit_translated(&mut pixels, 320, schedule.actor_slide_step_x * i, 0, None)
            .wrap(format!("battlepresent: compound actor slide {i}: "))?;
        out.actor_slide.push(pixels);
    }
    for j in (0..schedule.effect_slide_frames).rev() {
        let mut pixels = base.to_vec();
        effect.frames[0]
            .blit_translated(&mut pixels, 320, schedule.effect_slide_step_x * j, 0, None)
            .wrap(format!("battlepresent: compound effect slide {j}: "))?;
        out.effect_slide.push(pixels);
    }
    let mut main_base = base.to_vec();
    if schedule.overlay_first_frame {
        effect.frames[0].blit_at(&mut main_base, 320).wrap("battlepresent: compound steady overlay: ")?;
    }
    for frame in 1..effect.frames.len() {
        let mut pixels = main_base.clone();
        effect.frames[frame].blit_at(&mut pixels, 320).wrap(format!("battlepresent: compound main frame {frame}: "))?;
        out.main.push(pixels);
    }
    if schedule.tail_enabled {
        if schedule.tail_frames != 11 {
            return Err(err("battlepresent: native compound tail count unavailable"));
        }
        out.tail = Vec::with_capacity(at(schedule.tail_frames));
        for frame in 0..at(schedule.tail_frames) {
            let mut pixels = base.to_vec();
            let index = effect.frames.len() - 2 + frame % 2;
            effect.frames[index]
                .blit_at(&mut pixels, 320)
                .wrap(format!("battlepresent: compound tail frame {frame}: "))?;
            out.tail.push(pixels);
        }
    }
    Ok(out)
}

#[cfg(test)]
mod native_command_compound_test;
