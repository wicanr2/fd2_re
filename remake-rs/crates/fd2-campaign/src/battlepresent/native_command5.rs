//! 移植自 Go `battlepresent/native_command5.go`。

use super::native_command0::{
    NATIVE_COMMAND0_SURFACE_SIZE, blit_command0_work_frame, command0_present, command0_work_from,
};
use super::{GoResult, Result, at, err, len64};
use fd2_formats::figani::{
    Animation, Frame, NATIVE_COMMAND5_EFFECT_FRAME_COUNT, NativeCommand5Frame, NativeCommand5Layer,
    NativeCommand5TransitionFrame,
};

/// Go：`native_command5.go:10`
pub(super) fn command5_work(base: &[u8]) -> Result<Vec<u8>> {
    if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE {
        return Err(err("battlepresent: command5 base unavailable"));
    }
    Ok(command0_work_from(base))
}

/// Go：`native_command5.go:22`
pub(super) fn command5_present(work: &[u8]) -> Vec<u8> {
    command0_present(work)
}

/// Go：`native_command5.go:31`
fn draw_command5_layers(work: &mut [u8], effect: Option<&Animation>, layers: &[NativeCommand5Layer]) -> Result<()> {
    let Some(effect) =
        effect.filter(|e| len64(&e.frames) == NATIVE_COMMAND5_EFFECT_FRAME_COUNT || e.frames.len() == 14)
    else {
        return Err(err("battlepresent: command5 effect unavailable"));
    };
    for layer in layers {
        if layer.frame < 0 || layer.frame >= len64(&effect.frames) {
            return Err(err(format!("battlepresent: command5 frame {} unavailable", layer.frame)));
        }
        blit_command0_work_frame(work, &effect.frames[at(layer.frame)], layer.x, 0)
            .wrap(format!("battlepresent: command5 channel {}: ", layer.channel))?;
    }
    Ok(())
}

/// ComposeNativeCommand5TargetFrame 保存 mode4（無 handler 圖層）→target→
/// mode5 effect 的原始順序；數值與音效發布留給正式 owner。
///
/// Go：`native_command5.go:47`
pub fn compose_native_command5_target_frame(
    base: &[u8],
    target: &Frame,
    effect: Option<&Animation>,
    frame: &NativeCommand5Frame,
) -> Result<Vec<u8>> {
    let mut work = command5_work(base)?;
    blit_command0_work_frame(&mut work, target, 0, 0).wrap("battlepresent: command5 target: ")?;
    draw_command5_layers(&mut work, effect, &frame.layers)?;
    Ok(command5_present(&work))
}

/// ComposeNativeCommand5OrbitFrame 保存 mode1／7（無 handler 圖層）→actor→
/// target→mode2／8 effect 的前導與尾段順序。
///
/// Go：`native_command5.go:63`
pub fn compose_native_command5_orbit_frame(
    base: &[u8],
    actor_effect: Option<&Animation>,
    target_idle: Option<&Animation>,
    effect: Option<&Animation>,
    frame: &NativeCommand5Frame,
) -> Result<Vec<u8>> {
    let (Some(actor_effect), Some(target_idle)) =
        (actor_effect.filter(|a| !a.frames.is_empty()), target_idle.filter(|t| !t.frames.is_empty()))
    else {
        return Err(err("battlepresent: command5 orbit actors unavailable"));
    };
    let mut work = command5_work(base)?;
    blit_command0_work_frame(&mut work, &actor_effect.frames[actor_effect.frames.len() - 1], 0, 0)
        .wrap("battlepresent: command5 orbit actor: ")?;
    blit_command0_work_frame(&mut work, &target_idle.frames[0], 0, 0).wrap("battlepresent: command5 orbit target: ")?;
    draw_command5_layers(&mut work, effect, &frame.layers)?;
    Ok(command5_present(&work))
}

/// Go：`native_command5.go:82`
pub fn compose_native_command5_transition_frame(
    base: &[u8],
    actor_effect: Option<&Animation>,
    target_idle: Option<&Animation>,
    effect: Option<&Animation>,
    frame: &NativeCommand5TransitionFrame,
) -> Result<Vec<u8>> {
    let (Some(actor_effect), Some(target_idle)) =
        (actor_effect.filter(|a| !a.frames.is_empty()), target_idle.filter(|t| !t.frames.is_empty()))
    else {
        return Err(err("battlepresent: command5 transition actors unavailable"));
    };
    let mut work = command5_work(base)?;
    blit_command0_work_frame(&mut work, &actor_effect.frames[actor_effect.frames.len() - 1], 0, 0)
        .wrap("battlepresent: command5 transition actor: ")?;
    blit_command0_work_frame(&mut work, &target_idle.frames[0], frame.target_offset_x, 0)
        .wrap("battlepresent: command5 transition target: ")?;
    draw_command5_layers(&mut work, effect, &frame.layers)?;
    Ok(command5_present(&work))
}

#[cfg(test)]
pub(super) mod native_command5_test;
