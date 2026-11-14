//! 移植自 Go `battlepresent/native_command24_prelude.go`。

use super::native_command24_background::{NATIVE_COMMAND24_WORK_STRIDE, native_command24_viewport};
use super::{GoResult, NATIVE_COMMAND24_BACKGROUND_HEIGHT, NATIVE_COMMAND24_BACKGROUND_WIDTH, Result, at, err, len64};
use fd2_formats::fdother;
use fd2_formats::figani;

/// NativeCommand24PreludeInput is the command-24 caller contract for
/// 0x29164. RawSide is the original runtime record byte +6: zero selects the
/// right viewport, while every non-zero value selects the left viewport and
/// requires the caller-provided TAI frame.
///
/// Go：`native_command24_prelude.go:14`
#[derive(Debug, Clone, Default)]
pub struct NativeCommand24PreludeInput<'a> {
    pub base: &'a [u8],
    pub actor_idle: figani::Frame,
    pub platform: Option<&'a fdother::Frame>,
    pub raw_side: u8,
    pub baseline_dac: &'a [u8],
}

/// NativeCommandPreludeInput is the complete caller contract for 0x29164's
/// battle branches. Mode is the original second argument: zero includes the
/// first target idle and requires the TAI frame on both raw-side branches;
/// one is the command-24 form that omits the first target idle.
///
/// Go：`native_command24_prelude.go:26`
#[derive(Debug, Clone, Default)]
pub struct NativeCommandPreludeInput<'a> {
    pub base: &'a [u8],
    pub actor_idle: figani::Frame,
    pub first_target_idle: figani::Frame,
    pub platform: Option<&'a fdother::Frame>,
    pub raw_side: u8,
    pub mode: u8,
    pub baseline_dac: &'a [u8],
}

/// NativeCommand24PreludeFrame keeps indexed pixels and the corresponding
/// six-bit DAC together. The native helper applies stage*6 palette subtraction
/// after each present; an Ebiten owner must not reuse one fixed RGBA palette.
///
/// Go：`native_command24_prelude.go:39`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand24PreludeFrame {
    pub stage: i64,
    pub pixels: Vec<u8>,
    pub dac: Vec<u8>,
}

/// BuildNativeCommand24PreludeFrames reproduces the command-24 form of
/// 0x29164. It deliberately does not use the ending-specific transparent
/// TAI#3 compositor: battle callers pass the actor's real TAI resource.
///
/// Go：`native_command24_prelude.go:48`
pub fn build_native_command24_prelude_frames(
    in_: &NativeCommand24PreludeInput,
) -> Result<Vec<NativeCommand24PreludeFrame>> {
    build_native_command_prelude_frames(&NativeCommandPreludeInput {
        base: in_.base,
        actor_idle: in_.actor_idle.clone(),
        platform: in_.platform,
        raw_side: in_.raw_side,
        mode: 1,
        baseline_dac: in_.baseline_dac,
        ..Default::default()
    })
}

fn frame_ok(frame: &figani::Frame) -> bool {
    frame.width > 0
        && frame.height > 0
        && len64(&frame.pixels) == frame.width.wrapping_mul(frame.height)
        && frame.mask.len() == frame.pixels.len()
}

/// BuildNativeCommandPreludeFrames reproduces both directly observed battle
/// forms of 0x29164. It returns only frames that have passed complete indexed
/// and DAC validation; presentation owners remain responsible for Draw
/// acknowledgement and for publishing any later numeric transaction.
///
/// Go：`native_command24_prelude.go:59`
pub fn build_native_command_prelude_frames(
    in_: &NativeCommandPreludeInput,
) -> Result<Vec<NativeCommand24PreludeFrame>> {
    const FRAME_BYTES: i64 = NATIVE_COMMAND24_BACKGROUND_WIDTH * NATIVE_COMMAND24_BACKGROUND_HEIGHT;
    if len64(in_.base) != FRAME_BYTES || in_.baseline_dac.len() != 256 * 3 || !frame_ok(&in_.actor_idle) || in_.mode > 1
    {
        return Err(err("battlepresent: native command prelude inputs unavailable"));
    }
    fdother::parse_vga_palette(in_.baseline_dac).go()?;
    let empty = fdother::Frame::default();
    let platform_src = match in_.platform {
        Some(p) => p,
        None if in_.mode == 0 || in_.raw_side != 0 => {
            return Err(err("battlepresent: native command prelude requires TAI"));
        }
        None => &empty,
    };
    if in_.mode == 0 && !frame_ok(&in_.first_target_idle) {
        return Err(err("battlepresent: native command prelude first target unavailable"));
    }

    let mut base = in_.base.to_vec();
    if in_.mode == 0 && in_.raw_side == 0 {
        let mut platform = platform_src.clone();
        (platform.x, platform.y) = (164, 157);
        platform.blit(&mut base, NATIVE_COMMAND24_BACKGROUND_WIDTH, -1).go()?;
    }
    let stride = at(NATIVE_COMMAND24_WORK_STRIDE);
    let mut work = vec![0u8; at(NATIVE_COMMAND24_WORK_STRIDE * NATIVE_COMMAND24_BACKGROUND_HEIGHT)];
    let mut frames = Vec::with_capacity(9);
    for stage in (0..=8i64).rev() {
        work.fill(0);
        let viewport = if in_.raw_side == 0 { NATIVE_COMMAND24_BACKGROUND_WIDTH } else { 0 };
        for y in 0..200usize {
            let start = y * stride + at(viewport);
            work[start..start + 320].copy_from_slice(&base[y * 320..(y + 1) * 320]);
        }

        if in_.raw_side == 0 {
            in_.actor_idle.blit_at_base(&mut work, NATIVE_COMMAND24_WORK_STRIDE, viewport - stage * 10).go()?;
            if in_.mode == 0 {
                in_.first_target_idle.blit_at_base(&mut work, NATIVE_COMMAND24_WORK_STRIDE, viewport).go()?;
            }
        } else {
            if in_.mode == 0 {
                in_.first_target_idle.blit_at_base(&mut work, NATIVE_COMMAND24_WORK_STRIDE, 0).go()?;
            }
            let mut platform = platform_src.clone();
            (platform.x, platform.y) = (164, 157);
            platform.blit_at(&mut work, NATIVE_COMMAND24_WORK_STRIDE, stage * 10, -1).go()?;
            in_.actor_idle.blit_at_base(&mut work, NATIVE_COMMAND24_WORK_STRIDE, stage * 10).go()?;
        }

        let mut dac = in_.baseline_dac.to_vec();
        fdother::apply_vga_palette_subtraction(&mut dac, in_.baseline_dac, 0, 255, stage * 6).go()?;
        frames.push(NativeCommand24PreludeFrame { stage, pixels: native_command24_viewport(&work, viewport), dac });
    }
    Ok(frames)
}

#[cfg(test)]
pub(super) mod native_command24_prelude_test;
