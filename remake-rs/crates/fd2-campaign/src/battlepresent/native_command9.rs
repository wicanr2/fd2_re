//! 移植自 Go `battlepresent/native_command9.go`。

use super::native_command0::{
    NATIVE_COMMAND0_SURFACE_SIZE, blit_command0_work_frame, command0_present, command0_work_from,
};
use super::{GoResult, NativeCommandEffectSource, NativeCommandSources, Result, at, err, len64};
use fd2_formats::figani::{
    self, Animation, Frame, NATIVE_COMMAND9_AI_DAMAGE_STAGES, NATIVE_COMMAND9_AI_EFFECT_RESOURCE,
    NativeCommand9AIFrame, NativeCommand9AISchedule,
};

/// Go：`native_command9.go:10`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand9RenderedFrame {
    pub sources: NativeCommandSources,
    pub pixels: Vec<u8>,
    pub hp_stage: i64,
    pub play_sub1: bool,
    pub play_sub2: bool,
}

/// Go：`native_command9.go:17`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand9AISequence {
    pub slide_in: Vec<NativeCommand9RenderedFrame>,
    pub front: Vec<NativeCommand9RenderedFrame>,
    pub target: Vec<NativeCommand9RenderedFrame>,
    pub tail: Vec<NativeCommand9RenderedFrame>,
    pub slide_out: Vec<NativeCommand9RenderedFrame>,
    pub next_idle_frame: i64,
    pub next_idle_repeat: i64,
}

/// Go：`native_command9.go:22`
#[derive(Debug, Clone, Default)]
pub struct NativeCommand9AIInput<'a> {
    pub before_base: &'a [u8],
    pub after_base: &'a [u8],
    pub target_bases: Vec<&'a [u8]>,
    pub actor: Option<&'a Animation>,
    pub target: Option<&'a Animation>,
    pub effect: Option<&'a Animation>,
    pub schedule: NativeCommand9AISchedule,
}

/// Go：`native_command9.go:30`
fn command9_work(base: &[u8]) -> Result<Vec<u8>> {
    if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE {
        return Err(err("battlepresent: command9 base unavailable"));
    }
    Ok(command0_work_from(base))
}

/// Go：`native_command9.go:42`
fn command9_present(work: &[u8]) -> Vec<u8> {
    command0_present(work)
}

/// Go：`native_command9.go:51`
fn command9_effect(work: &mut [u8], effect: Option<&Animation>, index: i64) -> Result<()> {
    let Some(effect) = effect.filter(|e| index >= 0 && index < len64(&e.frames)) else {
        return Err(err(format!("battlepresent: command9 effect frame {index} unavailable")));
    };
    blit_command0_work_frame(work, &effect.frames[at(index)], 0, 0)
}

/// Go：`native_command9.go:58`
pub fn compose_native_command9_slide_frame(
    base: &[u8],
    actor: &Frame,
    target: &Frame,
    actor_shift_x: i64,
    draw_actor: bool,
) -> Result<Vec<u8>> {
    let mut work = command9_work(base)?;
    if draw_actor {
        blit_command0_work_frame(&mut work, actor, actor_shift_x, 0).wrap("battlepresent: command9 slide actor: ")?;
    }
    blit_command0_work_frame(&mut work, target, 0, 0).wrap("battlepresent: command9 slide target: ")?;
    Ok(command9_present(&work))
}

/// Go：`native_command9.go:73`
pub fn compose_native_command9_orbit_frame(
    base: &[u8],
    target: &Frame,
    effect: Option<&Animation>,
    frame: &NativeCommand9AIFrame,
) -> Result<Vec<u8>> {
    let mut work = command9_work(base)?;
    for &index in &frame.effect_frames {
        command9_effect(&mut work, effect, index)?;
    }
    blit_command0_work_frame(&mut work, target, 0, 0).wrap("battlepresent: command9 orbit target: ")?;
    Ok(command9_present(&work))
}

/// Go：`native_command9.go:89`
pub fn compose_native_command9_target_frame(
    base: &[u8],
    target: &Frame,
    effect: Option<&Animation>,
    frame: &NativeCommand9AIFrame,
) -> Result<Vec<u8>> {
    if frame.effect_frames.len() != 2 || frame.effect_frames[0] != 0 {
        return Err(err("battlepresent: command9 target layer order unavailable"));
    }
    let mut work = command9_work(base)?;
    command9_effect(&mut work, effect, frame.effect_frames[0])?;
    blit_command0_work_frame(&mut work, target, 0, 0).wrap("battlepresent: command9 target: ")?;
    command9_effect(&mut work, effect, frame.effect_frames[1])?;
    Ok(command9_present(&work))
}

/// Go：`native_command9.go:107`
pub fn build_native_command9_ai_sequence(in_: &NativeCommand9AIInput) -> Result<NativeCommand9AISequence> {
    let (Some(actor_anim), Some(target_anim), Some(_)) = (
        in_.actor.filter(|a| !a.frames.is_empty()),
        in_.target.filter(|t| !t.frames.is_empty()),
        in_.effect.filter(|e| e.frames.len() >= 31),
    ) else {
        return Err(err("battlepresent: incomplete command9 AI sequence input"));
    };
    if len64(in_.before_base) != NATIVE_COMMAND0_SURFACE_SIZE
        || len64(in_.after_base) != NATIVE_COMMAND0_SURFACE_SIZE
        || len64(&in_.target_bases) != NATIVE_COMMAND9_AI_DAMAGE_STAGES + 1
        || in_.schedule.effect_resource != NATIVE_COMMAND9_AI_EFFECT_RESOURCE
    {
        return Err(err("battlepresent: incomplete command9 AI sequence input"));
    }
    for (stage, base) in in_.target_bases.iter().enumerate() {
        if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE {
            return Err(err(format!("battlepresent: command9 target stage {stage} unavailable")));
        }
    }
    let mut out = NativeCommand9AISequence::default();
    let (actor, target) = (&actor_anim.frames[0], &target_anim.frames[0]);
    for k in 0..in_.schedule.actor_slide_in {
        let last = k == in_.schedule.actor_slide_in - 1;
        let pixels = compose_native_command9_slide_frame(in_.before_base, actor, target, -10 * k, !last)?;
        out.slide_in.push(NativeCommand9RenderedFrame {
            pixels,
            sources: NativeCommandSources {
                base: in_.before_base.to_vec(),
                actor_frame: if last { -1 } else { 0 },
                actor_offset_x: -10 * k,
                target_frame: 0,
                phase: "command9-slide-in".into(),
                ..Default::default()
            },
            ..Default::default()
        });
    }
    let mut state = figani::new_native_command9_ai_state();
    for _ in 0..in_.schedule.front_frames {
        let frame = figani::plan_native_command9_ai_orbit_frame(state, in_.schedule).go()?;
        let pixels = compose_native_command9_orbit_frame(in_.before_base, target, in_.effect, &frame)?;
        out.front.push(NativeCommand9RenderedFrame {
            pixels,
            sources: NativeCommandSources {
                base: in_.before_base.to_vec(),
                actor_frame: -1,
                target_frame: 0,
                before: native_command9_source_layers(&frame.effect_frames),
                phase: "command9-front".into(),
                ..Default::default()
            },
            ..Default::default()
        });
        state = frame.next;
    }
    let (planned, next) = figani::build_native_command9_ai_target_sequence(state, in_.schedule).go()?;
    state = next;
    let (mut idle_frame, mut idle_repeat, mut hp_stage) = (0usize, 0i64, 0i64);
    for (index, frame) in planned.iter().enumerate() {
        if frame.hp_stage != 0 {
            hp_stage = frame.hp_stage;
        }
        let base = in_.target_bases[at(hp_stage)];
        let pixels = compose_native_command9_target_frame(base, &target_anim.frames[idle_frame], in_.effect, frame)
            .wrap(format!("battlepresent: command9 target frame {index}: "))?;
        out.target.push(NativeCommand9RenderedFrame {
            pixels,
            hp_stage: frame.hp_stage,
            play_sub1: frame.play_sub1,
            play_sub2: frame.play_sub2,
            sources: NativeCommandSources {
                base: base.to_vec(),
                actor_frame: -1,
                target_frame: idle_frame as i64,
                before: native_command9_source_layers(&frame.effect_frames[..1]),
                after: native_command9_source_layers(&frame.effect_frames[1..]),
                phase: "command9-target".into(),
                ..Default::default()
            },
        });
        if target_anim.frames[idle_frame].delay <= 0 {
            return Err(err("battlepresent: command9 target idle delay unavailable"));
        }
        idle_repeat += 1;
        if idle_repeat >= target_anim.frames[idle_frame].delay {
            idle_repeat = 0;
            idle_frame = (idle_frame + 1) % target_anim.frames.len();
        }
    }
    for _ in 0..in_.schedule.tail_frames {
        let frame = figani::plan_native_command9_ai_orbit_frame(state, in_.schedule).go()?;
        let pixels =
            compose_native_command9_orbit_frame(in_.after_base, &target_anim.frames[idle_frame], in_.effect, &frame)?;
        out.tail.push(NativeCommand9RenderedFrame {
            pixels,
            sources: NativeCommandSources {
                base: in_.after_base.to_vec(),
                actor_frame: -1,
                target_frame: idle_frame as i64,
                before: native_command9_source_layers(&frame.effect_frames),
                phase: "command9-tail".into(),
                ..Default::default()
            },
            ..Default::default()
        });
        state = frame.next;
    }
    for k in (0..in_.schedule.actor_slide_out).rev() {
        let pixels =
            compose_native_command9_slide_frame(in_.after_base, actor, &target_anim.frames[idle_frame], -10 * k, true)?;
        out.slide_out.push(NativeCommand9RenderedFrame {
            pixels,
            sources: NativeCommandSources {
                base: in_.after_base.to_vec(),
                actor_frame: 0,
                actor_offset_x: -10 * k,
                target_frame: idle_frame as i64,
                phase: "command9-slide-out".into(),
                ..Default::default()
            },
            ..Default::default()
        });
    }
    (out.next_idle_frame, out.next_idle_repeat) = (idle_frame as i64, idle_repeat);
    Ok(out)
}

/// Go：`native_command9.go:200`
fn native_command9_source_layers(indices: &[i64]) -> Vec<NativeCommandEffectSource> {
    indices.iter().map(|&frame| NativeCommandEffectSource { frame, x: 0, y: 0 }).collect()
}

#[cfg(test)]
mod native_command9_test;
