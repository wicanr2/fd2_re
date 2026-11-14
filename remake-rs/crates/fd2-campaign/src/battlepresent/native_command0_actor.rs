//! 移植自 Go `battlepresent/native_command0_actor.go`。

use super::{GoResult, NATIVE_COMMAND24_BACKGROUND_HEIGHT, NATIVE_COMMAND24_BACKGROUND_WIDTH, Result, err, len64};
use fd2_formats::fdother;
use fd2_formats::figani::Animation;

/// NativeCommand0ActorFrame is one actual 0x2B659 present boundary. Pulse is
/// true for the six repetitions that temporarily set DAC entry zero to the
/// command-indexed color for 30 ms and then restore it to black before tick.
///
/// Go：`native_command0_actor.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand0ActorFrame {
    pub pixels: Vec<u8>,
    pub effect_frame: i64,
    pub effect_repeat: i64,
    pub target_idle_frame: i64,
    pub publish_mp: bool,
    pub pulse: bool,
}

/// Go：`native_command0_actor.go:23`
#[derive(Debug, Clone, Default)]
pub struct NativeCommand0ActorInput<'a> {
    pub base_before: &'a [u8],
    pub base_after: &'a [u8],
    pub actor_effect: Option<&'a Animation>,
    pub first_target_idle: Option<&'a Animation>,
    pub raw_side: u8,
    pub background: fdother::Frame,
    pub platform: fdother::Frame,
    pub lut: &'a [u8],
}

/// BuildNativeCommand0ActorFrames reproduces command0's 0x2B659 indexed
/// pixels and exact raw marker placement. The caller owns the command-color
/// DAC subphase, sample0 playback, Draw acknowledgement and MP publication.
///
/// Go：`native_command0_actor.go:36`
pub fn build_native_command0_actor_frames(in_: &NativeCommand0ActorInput) -> Result<Vec<NativeCommand0ActorFrame>> {
    const SURFACE: i64 = NATIVE_COMMAND24_BACKGROUND_WIDTH * NATIVE_COMMAND24_BACKGROUND_HEIGHT;
    let (Some(actor_effect), Some(first_target_idle)) = (in_.actor_effect, in_.first_target_idle) else {
        return Err(err("battlepresent: command0 actor inputs unavailable"));
    };
    if len64(in_.base_before) != SURFACE
        || len64(in_.base_after) != SURFACE
        || actor_effect.frames.is_empty()
        || first_target_idle.frames.is_empty()
        || in_.lut.len() != 256
    {
        return Err(err("battlepresent: command0 actor inputs unavailable"));
    }
    let mut after = in_.base_after.to_vec();
    let mut background = in_.background.clone();
    (background.x, background.y) = (0, 50);
    background
        .blit_lut_at(&mut after, NATIVE_COMMAND24_BACKGROUND_WIDTH, 0, in_.lut)
        .wrap("battlepresent: command0 actor BG LUT: ")?;
    let mut platform = in_.platform.clone();
    (platform.x, platform.y) = (164, 157);
    platform
        .blit_lut_at(&mut after, NATIVE_COMMAND24_BACKGROUND_WIDTH, 0, in_.lut)
        .wrap("battlepresent: command0 actor TAI LUT: ")?;
    for (i, frame) in first_target_idle.frames.iter().enumerate() {
        if frame.delay <= 0
            || frame.width <= 0
            || frame.height <= 0
            || len64(&frame.pixels) != frame.width.wrapping_mul(frame.height)
            || frame.mask.len() != frame.pixels.len()
        {
            return Err(err(format!("battlepresent: command0 target idle frame {i} malformed")));
        }
    }
    let (mut target_frame, mut target_repeat) = (0usize, 0i64);
    let (mut pulse_remaining, mut marker_count) = (0i64, 0i64);
    let mut frames = Vec::new();
    for (effect_index, effect) in actor_effect.frames.iter().enumerate() {
        if effect.delay < 0
            || effect.width <= 0
            || effect.height <= 0
            || len64(&effect.pixels) != effect.width.wrapping_mul(effect.height)
            || effect.mask.len() != effect.pixels.len()
        {
            return Err(err(format!("battlepresent: command0 actor effect frame {effect_index} malformed")));
        }
        let publish = effect.raw_byte4 == 1;
        if publish {
            marker_count += 1;
            pulse_remaining = 6;
            if effect.delay == 0 {
                return Err(err("battlepresent: command0 MP marker has no present boundary"));
            }
        }
        for repeat in 0..effect.delay {
            let base = if marker_count > 0 { &after[..] } else { in_.base_before };
            let mut pixels = base.to_vec();
            let target = &first_target_idle.frames[target_frame];
            if in_.raw_side != 0 {
                effect.blit_at(&mut pixels, NATIVE_COMMAND24_BACKGROUND_WIDTH).go()?;
                target.blit_at(&mut pixels, NATIVE_COMMAND24_BACKGROUND_WIDTH).go()?;
            } else {
                target.blit_at(&mut pixels, NATIVE_COMMAND24_BACKGROUND_WIDTH).go()?;
                effect.blit_at(&mut pixels, NATIVE_COMMAND24_BACKGROUND_WIDTH).go()?;
            }
            frames.push(NativeCommand0ActorFrame {
                pixels,
                effect_frame: effect_index as i64,
                effect_repeat: repeat,
                target_idle_frame: target_frame as i64,
                publish_mp: publish && repeat == 0,
                pulse: pulse_remaining > 0,
            });
            if pulse_remaining > 0 {
                pulse_remaining -= 1;
            }
            target_repeat += 1;
            if target_repeat >= first_target_idle.frames[target_frame].delay {
                target_repeat = 0;
                target_frame = (target_frame + 1) % first_target_idle.frames.len();
            }
        }
    }
    if marker_count != 1 || frames.is_empty() {
        return Err(err(format!("battlepresent: command0 actor MP markers={marker_count} frames={}", frames.len())));
    }
    Ok(frames)
}

#[cfg(test)]
mod native_command0_actor_test;
