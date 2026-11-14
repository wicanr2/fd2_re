//! 移植自 Go `battlepresent/native_command7.go`。

use super::native_command0::blit_command0_work_frame;
use super::native_command5::{command5_present, command5_work};
use super::{GoResult, Result, at, err, len64};
use fd2_formats::figani::{
    Animation, Frame, NATIVE_COMMAND7_EFFECT_FRAME_COUNT, NativeCommand7Frame, NativeCommand7Layer,
    NativeCommand7TransitionFrame,
};

/// Go：`native_command7.go:10`
fn draw_command7_layers(work: &mut [u8], effect: Option<&Animation>, layers: &[NativeCommand7Layer]) -> Result<()> {
    let Some(effect) = effect.filter(|e| len64(&e.frames) == NATIVE_COMMAND7_EFFECT_FRAME_COUNT) else {
        return Err(err("battlepresent: command7 effect unavailable"));
    };
    for layer in layers {
        if layer.frame < 0 || layer.frame >= len64(&effect.frames) {
            return Err(err(format!("battlepresent: command7 frame {} unavailable", layer.frame)));
        }
        blit_command0_work_frame(work, &effect.frames[at(layer.frame)], layer.x, 0)
            .wrap(format!("battlepresent: command7 channel {}: ", layer.channel))?;
    }
    Ok(())
}

/// Go：`native_command7.go:25`
pub fn compose_native_command7_target_frame(
    base: &[u8],
    target: &Frame,
    effect: Option<&Animation>,
    frame: &NativeCommand7Frame,
) -> Result<Vec<u8>> {
    let mut work = command5_work(base)?;
    blit_command0_work_frame(&mut work, target, 0, 0).wrap("battlepresent: command7 target: ")?;
    draw_command7_layers(&mut work, effect, &frame.layers)?;
    Ok(command5_present(&work))
}

/// Go：`native_command7.go:39`
pub fn compose_native_command7_orbit_frame(
    base: &[u8],
    actor_effect: Option<&Animation>,
    target_idle: Option<&Animation>,
    effect: Option<&Animation>,
    frame: &NativeCommand7Frame,
) -> Result<Vec<u8>> {
    let (Some(actor_effect), Some(target_idle)) =
        (actor_effect.filter(|a| !a.frames.is_empty()), target_idle.filter(|t| !t.frames.is_empty()))
    else {
        return Err(err("battlepresent: command7 orbit actors unavailable"));
    };
    let mut work = command5_work(base)?;
    blit_command0_work_frame(&mut work, &actor_effect.frames[actor_effect.frames.len() - 1], 0, 0)
        .wrap("battlepresent: command7 orbit actor: ")?;
    blit_command0_work_frame(&mut work, &target_idle.frames[0], 0, 0).wrap("battlepresent: command7 orbit target: ")?;
    draw_command7_layers(&mut work, effect, &frame.layers)?;
    Ok(command5_present(&work))
}

/// Go：`native_command7.go:59`
pub fn compose_native_command7_transition_frame(
    base: &[u8],
    actor_effect: Option<&Animation>,
    target_idle: Option<&Animation>,
    effect: Option<&Animation>,
    frame: &NativeCommand7TransitionFrame,
) -> Result<Vec<u8>> {
    let (Some(actor_effect), Some(target_idle)) =
        (actor_effect.filter(|a| !a.frames.is_empty()), target_idle.filter(|t| !t.frames.is_empty()))
    else {
        return Err(err("battlepresent: command7 transition actors unavailable"));
    };
    let mut work = command5_work(base)?;
    blit_command0_work_frame(&mut work, &actor_effect.frames[actor_effect.frames.len() - 1], 0, 0)
        .wrap("battlepresent: command7 transition actor: ")?;
    blit_command0_work_frame(&mut work, &target_idle.frames[0], frame.target_offset_x, 0)
        .wrap("battlepresent: command7 transition target: ")?;
    draw_command7_layers(&mut work, effect, &frame.layers)?;
    Ok(command5_present(&work))
}

#[cfg(test)]
pub(super) mod native_command7_test;
