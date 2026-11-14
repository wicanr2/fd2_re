//! 移植自 Go `battlepresent/native_command2.go`。

use super::native_command0::{
    NATIVE_COMMAND0_SURFACE_SIZE, blit_command0_work_frame, command0_present, command0_work_from,
};
use super::{GoResult, NativeCommandEffectSource, NativeCommandSources, Result, at, err, len64};
use fd2_formats::figani::{
    Animation, Frame, NATIVE_COMMAND2_EFFECT_FRAME_COUNT, NATIVE_COMMAND2_SOUND_RESOURCE, NativeCommand2HelperFrame,
    NativeCommand2PresentationSchedule,
};

/// Go 各合成函式內的 `draw` 閉包：畫一格 effect 並依 before 記進來源。
fn draw_effect(
    work: &mut [u8],
    sources: &mut NativeCommandSources,
    effect: &Animation,
    frame: i64,
    x: i64,
    y: i64,
    before: bool,
) -> Result<()> {
    blit_command0_work_frame(work, &effect.frames[at(frame)], x, y)?;
    let layer = NativeCommandEffectSource { frame, x, y };
    if before {
        sources.before.push(layer);
    } else {
        sources.after.push(layer);
    }
    Ok(())
}

/// ComposeNativeCommand2TargetFrame 重現 0x26528 單張
/// mode4→target→mode5 合成；狀態推進與戰鬥數值發布由外層 owner 負責。
///
/// Go：`native_command2.go:12`
pub fn compose_native_command2_target_frame(
    base: &[u8],
    target: &Frame,
    effect: Option<&Animation>,
    schedule: NativeCommand2PresentationSchedule,
    raw_side: u8,
    state_frame: i64,
) -> Result<Vec<u8>> {
    compose_native_command2_target_frame_with_sources(base, target, effect, schedule, raw_side, state_frame)
        .map(|(pixels, _)| pixels)
}

/// Go：`native_command2.go:17`
pub fn compose_native_command2_target_frame_with_sources(
    base: &[u8],
    target: &Frame,
    effect: Option<&Animation>,
    schedule: NativeCommand2PresentationSchedule,
    raw_side: u8,
    state_frame: i64,
) -> Result<(Vec<u8>, NativeCommandSources)> {
    let mut sources =
        NativeCommandSources { base: base.to_vec(), actor_frame: -1, phase: "target".into(), ..Default::default() };
    let want_resource = if raw_side == 0 { 27 } else { 26 };
    let Some(effect) = effect.filter(|e| {
        len64(base) == NATIVE_COMMAND0_SURFACE_SIZE
            && len64(&e.frames) == NATIVE_COMMAND2_EFFECT_FRAME_COUNT
            && state_frame >= 0
            && state_frame < len64(&e.frames)
    }) else {
        return Err(err("battlepresent: incomplete command2 target frame input"));
    };
    if schedule.effect_resource != want_resource || schedule.sound_resource != NATIVE_COMMAND2_SOUND_RESOURCE {
        return Err(err("battlepresent: command2 schedule does not match raw side"));
    }
    let mut work = command0_work_from(base);
    let draw = |work: &mut [u8], sources: &mut NativeCommandSources, frame: i64, x: i64, y: i64, before: bool| {
        draw_effect(work, sources, effect, frame, x, y, before).wrap(format!("battlepresent: command2 frame{frame}: "))
    };
    if raw_side != 0 {
        draw(&mut work, &mut sources, 15, 1, -1, true)?;
    }
    blit_command0_work_frame(&mut work, target, 0, 0).wrap("battlepresent: command2 target: ")?;
    if raw_side == 0 {
        draw(&mut work, &mut sources, 15, -1, -1, false)?;
    }
    draw(&mut work, &mut sources, state_frame, 1, -1, false)?;
    Ok((command0_present(&work), sources))
}

/// ComposeNativeCommand2OrbitFrame 保存 front/tail 的 mode1/7→actor/target→
/// mode2/8 圖層順序。raw side決定 effect 位於角色層之前或之後。
///
/// Go：`native_command2.go:70`
pub fn compose_native_command2_orbit_frame(
    base: &[u8],
    actor_effect: Option<&Animation>,
    target_idle: Option<&Animation>,
    effect: Option<&Animation>,
    helper: NativeCommand2HelperFrame,
    raw_side: u8,
) -> Result<Vec<u8>> {
    compose_native_command2_orbit_frame_with_sources(base, actor_effect, target_idle, effect, helper, raw_side)
        .map(|(pixels, _)| pixels)
}

/// Go：`native_command2.go:75`
pub fn compose_native_command2_orbit_frame_with_sources(
    base: &[u8],
    actor_effect: Option<&Animation>,
    target_idle: Option<&Animation>,
    effect: Option<&Animation>,
    helper: NativeCommand2HelperFrame,
    raw_side: u8,
) -> Result<(Vec<u8>, NativeCommandSources)> {
    let mut sources = NativeCommandSources {
        base: base.to_vec(),
        actor_frame: -1,
        phase: "handler-orbit".into(),
        ..Default::default()
    };
    let (Some(actor_effect), Some(target_idle), Some(effect)) = (actor_effect, target_idle, effect) else {
        return Err(err("battlepresent: command2 orbit inputs unavailable"));
    };
    if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE
        || actor_effect.frames.is_empty()
        || target_idle.frames.is_empty()
        || helper.effect_frame < 0
        || helper.effect_frame >= len64(&effect.frames)
    {
        return Err(err("battlepresent: command2 orbit inputs unavailable"));
    }
    sources.actor_frame = len64(&actor_effect.frames) - 1;
    let mut pixels = base.to_vec();
    let draw_effect = |pixels: &mut [u8], sources: &mut NativeCommandSources| -> Result<()> {
        effect.frames[at(helper.effect_frame)].blit_at(pixels, 320).go()?;
        let layer = NativeCommandEffectSource { frame: helper.effect_frame, x: 0, y: 0 };
        if raw_side != 0 {
            sources.before.push(layer);
        } else {
            sources.after.push(layer);
        }
        Ok(())
    };
    if raw_side != 0 {
        draw_effect(&mut pixels, &mut sources)?;
    }
    actor_effect.frames[actor_effect.frames.len() - 1].blit_at(&mut pixels, 320).go()?;
    target_idle.frames[0].blit_at(&mut pixels, 320).go()?;
    if raw_side == 0 {
        draw_effect(&mut pixels, &mut sources)?;
    }
    Ok((pixels, sources))
}

/// Go：`native_command2.go:117`
pub fn compose_native_command2_transition_frame(
    base: &[u8],
    actor_effect: Option<&Animation>,
    target_idle: Option<&Animation>,
    effect: Option<&Animation>,
    helper: NativeCommand2HelperFrame,
    raw_side: u8,
    offset_x: i64,
) -> Result<Vec<u8>> {
    compose_native_command2_transition_frame_with_sources(
        base,
        actor_effect,
        target_idle,
        effect,
        helper,
        raw_side,
        offset_x,
    )
    .map(|(pixels, _)| pixels)
}

/// Go：`native_command2.go:122`
pub fn compose_native_command2_transition_frame_with_sources(
    base: &[u8],
    actor_effect: Option<&Animation>,
    target_idle: Option<&Animation>,
    effect: Option<&Animation>,
    helper: NativeCommand2HelperFrame,
    raw_side: u8,
    offset_x: i64,
) -> Result<(Vec<u8>, NativeCommandSources)> {
    let mut sources = NativeCommandSources {
        base: base.to_vec(),
        actor_frame: -1,
        target_offset_x: offset_x,
        phase: "transition".into(),
        ..Default::default()
    };
    let (Some(actor_effect), Some(target_idle), Some(effect)) = (actor_effect, target_idle, effect) else {
        return Err(err("battlepresent: command2 transition inputs unavailable"));
    };
    if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE
        || actor_effect.frames.is_empty()
        || target_idle.frames.is_empty()
        || helper.effect_frame < 0
        || helper.effect_frame >= len64(&effect.frames)
    {
        return Err(err("battlepresent: command2 transition inputs unavailable"));
    }
    sources.actor_frame = len64(&actor_effect.frames) - 1;
    let mut work = command0_work_from(base);
    if raw_side != 0 {
        draw_effect(&mut work, &mut sources, effect, 15, 1, -1, true)?;
    }
    blit_command0_work_frame(&mut work, &actor_effect.frames[actor_effect.frames.len() - 1], 0, 0)?;
    blit_command0_work_frame(&mut work, &target_idle.frames[0], offset_x, 0)?;
    if raw_side == 0 {
        draw_effect(&mut work, &mut sources, effect, 15, -1, -1, false)?;
    }
    draw_effect(&mut work, &mut sources, effect, helper.effect_frame, 1, -1, false)?;
    Ok((command0_present(&work), sources))
}

#[cfg(test)]
mod native_command2_test;
