//! 移植自 Go `battlepresent/native_command1.go`。

use super::native_command0::{
    NATIVE_COMMAND0_SURFACE_SIZE, blit_command0_work_frame, command0_present, command0_work_from,
};
use super::{GoResult, Result, at, err, len64};
use fd2_formats::figani::{
    self, Animation, Frame, NATIVE_COMMAND1_EFFECT_FRAME_COUNT, NATIVE_COMMAND1_PRESENTATION_FRAMES,
    NativeCommand1PresentationSchedule,
};

/// NativeCommand1TargetSequence 是 0x2A6BD 以 mode3 回傳31所擁有的完整單目標
/// 批次。HPStages 只在八個 numeric marker 依序給出1..8；SampleMarkers 保存
/// 0x262EF 直接播放 sample1 的八個畫格。
///
/// Go：`native_command1.go:13`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand1TargetSequence {
    pub frames: Vec<Vec<u8>>,
    pub sources: Vec<NativeCommand1FrameSources>,
    pub hp_stages: Vec<i64>,
    pub sample_markers: Vec<bool>,
    pub next_idle_frame: i64,
    pub next_idle_repeat: i64,
}

/// NativeCommand1FrameSources records the inputs consumed by this same indexed
/// composition. The optional actor precedes mode4, target, then mode5 layers.
/// NextTarget is used only by the nine-frame adjacent-target transition.
///
/// Go：`native_command1.go:25`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand1FrameSources {
    pub base: Vec<u8>,
    pub actor_frame: i64,
    pub target_frame: i64,
    pub target_offset_x: i64,
    pub next_target: bool,
    pub before_target: Vec<NativeCommand1EffectLayer>,
    pub after_target: Vec<NativeCommand1EffectLayer>,
}

/// Go：`native_command1.go:33`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand1EffectLayer {
    pub frame: i64,
    pub x: i64,
    pub y: i64,
}

/// Go：`native_command1.go:37`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand1TransitionSequence {
    pub frames: Vec<Vec<u8>>,
    pub sources: Vec<NativeCommand1FrameSources>,
}

fn frame_ok(frame: &Frame) -> bool {
    frame.width > 0
        && frame.height > 0
        && len64(&frame.pixels) == frame.width.wrapping_mul(frame.height)
        && frame.mask.len() == frame.pixels.len()
}

/// BuildNativeCommand1TargetSequence 在正式 owner 發布任何狀態前預建完整31張。
///
/// Go：`native_command1.go:43`
pub fn build_native_command1_target_sequence(
    bases: &[&[u8]],
    target_idle: Option<&Animation>,
    effect: Option<&Animation>,
    schedule: NativeCommand1PresentationSchedule,
) -> Result<NativeCommand1TargetSequence> {
    let Some(target_idle) = target_idle.filter(|t| bases.len() == 9 && !t.frames.is_empty()) else {
        return Err(err("battlepresent: command1 staged target inputs unavailable"));
    };
    for (index, frame) in target_idle.frames.iter().enumerate() {
        if frame.delay <= 0 || !frame_ok(frame) {
            return Err(err(format!("battlepresent: command1 target idle frame {index} malformed")));
        }
    }
    let n = at(NATIVE_COMMAND1_PRESENTATION_FRAMES);
    let mut sequence = NativeCommand1TargetSequence {
        frames: Vec::with_capacity(n),
        hp_stages: Vec::with_capacity(n),
        sample_markers: Vec::with_capacity(n),
        ..Default::default()
    };
    let (mut hp_stage, mut target_frame, mut target_repeat) = (0usize, 0usize, 0i64);
    for step in 0..NATIVE_COMMAND1_PRESENTATION_FRAMES {
        // Go 在 hpStage 超過 8 時以 bases[hpStage] 越界 panic；Rust 的索引同樣 panic。
        let (pixels, mut sources) = compose_native_command1_target_frame_inner(
            bases[hp_stage],
            &target_idle.frames[target_frame],
            effect,
            schedule,
            step,
        )?;
        let (numeric, sample) = figani::native_command1_target_markers(step).go()?;
        if numeric {
            hp_stage += 1;
        }
        let published_stage = if numeric { hp_stage as i64 } else { 0 };
        sequence.frames.push(pixels);
        sources.target_frame = target_frame as i64;
        sequence.sources.push(sources);
        sequence.hp_stages.push(published_stage);
        sequence.sample_markers.push(sample);
        target_repeat += 1;
        if target_repeat >= target_idle.frames[target_frame].delay {
            target_repeat = 0;
            target_frame = (target_frame + 1) % target_idle.frames.len();
        }
    }
    if hp_stage != 8 {
        return Err(err("battlepresent: command1 HP marker count changed"));
    }
    (sequence.next_idle_frame, sequence.next_idle_repeat) = (target_frame as i64, target_repeat);
    Ok(sequence)
}

/// BuildNativeCommand1TransitionFrames 重現 sub_2BA22 在 command1 target counters
/// 全部離開可繪範圍後的九張相鄰目標轉場。每張仍畫 actor effect 最末幀及目前／
/// 下一目標 idle frame0；command1 的 mode4／5 此時不再產生 effect layer 或 marker。
///
/// Go：`native_command1.go:96`
pub fn build_native_command1_transition_frames(
    base: &[u8],
    actor_effect: Option<&Animation>,
    current_target: Option<&Animation>,
    next_target: Option<&Animation>,
    raw_side: u8,
) -> Result<Vec<Vec<u8>>> {
    build_native_command1_transition_sequence(base, actor_effect, current_target, next_target, raw_side)
        .map(|s| s.frames)
}

/// Go：`native_command1.go:101`
pub fn build_native_command1_transition_sequence(
    base: &[u8],
    actor_effect: Option<&Animation>,
    current_target: Option<&Animation>,
    next_target: Option<&Animation>,
    raw_side: u8,
) -> Result<NativeCommand1TransitionSequence> {
    let (Some(actor_effect), Some(current_target), Some(next_target)) = (actor_effect, current_target, next_target)
    else {
        return Err(err("battlepresent: command1 transition inputs unavailable"));
    };
    if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE
        || actor_effect.frames.is_empty()
        || current_target.frames.is_empty()
        || next_target.frames.is_empty()
    {
        return Err(err("battlepresent: command1 transition inputs unavailable"));
    }
    let direction: i64 = if raw_side == 0 { 1 } else { -1 };
    // Go 的函式內型別 `targetStep{animation, offset, next}`。
    let mut steps: Vec<(&Animation, i64, bool)> = Vec::with_capacity(9);
    for step in 1..=4 {
        steps.push((current_target, direction * 35 * step, false));
    }
    for step in (0..=4).rev() {
        steps.push((next_target, direction * 35 * step, true));
    }
    let mut out = NativeCommand1TransitionSequence::default();
    let last = actor_effect.frames.len() - 1;
    for (animation, offset, next) in steps {
        let mut work = command0_work_from(base);
        blit_command0_work_frame(&mut work, &actor_effect.frames[last], 0, 0)
            .wrap("battlepresent: command1 transition actor: ")?;
        blit_command0_work_frame(&mut work, &animation.frames[0], offset, 0)
            .wrap("battlepresent: command1 transition target: ")?;
        out.frames.push(command0_present(&work));
        out.sources.push(NativeCommand1FrameSources {
            base: base.to_vec(),
            actor_frame: last as i64,
            target_frame: 0,
            target_offset_x: offset,
            next_target: next,
            ..Default::default()
        });
    }
    Ok(out)
}

/// ComposeNativeCommand1TargetFrame 重現 0x262EF 的 mode4→target→mode5
/// 圖層順序。它只合成 indexed pixels，不發布 MP、HP、音效或行動完成狀態。
///
/// Go：`native_command1.go:150`
pub fn compose_native_command1_target_frame(
    base: &[u8],
    target: &Frame,
    effect: Option<&Animation>,
    schedule: NativeCommand1PresentationSchedule,
    step: i64,
) -> Result<Vec<u8>> {
    compose_native_command1_target_frame_inner(base, target, effect, schedule, step).map(|(pixels, _)| pixels)
}

/// Go：`native_command1.go:155`
fn compose_native_command1_target_frame_inner(
    base: &[u8],
    target: &Frame,
    effect: Option<&Animation>,
    schedule: NativeCommand1PresentationSchedule,
    step: i64,
) -> Result<(Vec<u8>, NativeCommand1FrameSources)> {
    let mut sources = NativeCommand1FrameSources { base: base.to_vec(), actor_frame: -1, ..Default::default() };
    let Some(effect) = effect.filter(|e| {
        len64(base) == NATIVE_COMMAND0_SURFACE_SIZE && len64(&e.frames) == NATIVE_COMMAND1_EFFECT_FRAME_COUNT
    }) else {
        return Err(err("battlepresent: incomplete command1 target frame input"));
    };
    let counters = figani::native_command1_counters(step).go()?;
    let mut work = command0_work_from(base);
    let draw_mode = |work: &mut [u8], sources: &mut NativeCommand1FrameSources, mode: i64| -> Result<()> {
        for (slot, &counter) in counters.iter().enumerate() {
            let (frame, offset, ok) = figani::native_command1_mode_frame(mode, slot as i64, counter);
            if !ok {
                continue;
            }
            let x = schedule.anchor_x.wrapping_add(schedule.side_x_shift).wrapping_add(schedule.x_offsets[at(offset)]);
            let y = schedule.y_offsets[at(offset)];
            blit_command0_work_frame(work, &effect.frames[at(frame)], x, y)
                .wrap(format!("battlepresent: command1 mode{mode} slot{slot}: "))?;
            let layer = NativeCommand1EffectLayer { frame, x, y };
            if mode == 4 {
                sources.before_target.push(layer);
            } else {
                sources.after_target.push(layer);
            }
        }
        Ok(())
    };
    draw_mode(&mut work, &mut sources, 4)?;
    blit_command0_work_frame(&mut work, target, 0, 0).wrap("battlepresent: command1 target: ")?;
    draw_mode(&mut work, &mut sources, 5)?;
    Ok((command0_present(&work), sources))
}

#[cfg(test)]
mod native_command1_test;
