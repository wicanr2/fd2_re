//! 移植自 Go `battlepresent/native_command8_sequence.go`。

use super::native_command_sources::native_command8_source_layers;
use super::native_command0::NATIVE_COMMAND0_SURFACE_SIZE;
use super::{
    GoResult, NativeCommandSources, Result, at, compose_native_command8_orbit_frame,
    compose_native_command8_target_frame, compose_native_command8_transition_frame, err, len64,
};
use fd2_formats::figani::{
    self, Animation, NATIVE_COMMAND8_DAMAGE_STAGES, NATIVE_COMMAND8_FRONT_FRAMES, NativeCommand8PresentationSchedule,
};

/// Go：`native_command8_sequence.go:10`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand8RenderedFrame {
    pub sources: NativeCommandSources,
    pub pixels: Vec<u8>,
    pub hp_stage: i64,
    pub play_sub1: bool,
    pub play_sub2: bool,
}

/// Go：`native_command8_sequence.go:16`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand8EffectSequence {
    pub front: Vec<NativeCommand8RenderedFrame>,
    pub targets: Vec<Vec<NativeCommand8RenderedFrame>>,
    pub transitions: Vec<Vec<NativeCommand8RenderedFrame>>,
    pub tail: Vec<NativeCommand8RenderedFrame>,
    pub next_idle_frame: i64,
    pub next_idle_repeat: i64,
}

/// Go：`native_command8_sequence.go:23`
#[derive(Debug, Clone, Default)]
pub struct NativeCommand8EffectInput<'a> {
    pub front_base: &'a [u8],
    pub tail_base: &'a [u8],
    pub target_bases: Vec<Vec<&'a [u8]>>,
    pub transition_bases: Vec<&'a [u8]>,
    pub actor_effect: Option<&'a Animation>,
    pub target_idle: Vec<Option<&'a Animation>>,
    pub effect: Option<&'a Animation>,
    pub schedule: NativeCommand8PresentationSchedule,
    pub raw_side: u8,
}

/// Go：`native_command8_sequence.go:34`
pub fn build_native_command8_effect_sequence(in_: &NativeCommand8EffectInput) -> Result<NativeCommand8EffectSequence> {
    let Some(actor_effect) = in_.actor_effect.filter(|a| {
        len64(in_.front_base) == NATIVE_COMMAND0_SURFACE_SIZE
            && len64(in_.tail_base) == NATIVE_COMMAND0_SURFACE_SIZE
            && !a.frames.is_empty()
            && in_.effect.is_some()
            && in_.target_idle.len() == 1
            && in_.target_bases.len() == 1
            && in_.transition_bases.is_empty()
            && in_.schedule.effect_resource != 0
    }) else {
        return Err(err("battlepresent: incomplete command8 effect sequence input"));
    };
    let mut idles = Vec::with_capacity(in_.target_idle.len());
    for (target_index, idle) in in_.target_idle.iter().enumerate() {
        let Some(idle) = idle.filter(|i| {
            !i.frames.is_empty() && len64(&in_.target_bases[target_index]) == NATIVE_COMMAND8_DAMAGE_STAGES + 1
        }) else {
            return Err(err(format!("battlepresent: command8 target {target_index} inputs unavailable")));
        };
        for (stage, base) in in_.target_bases[target_index].iter().enumerate() {
            if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE {
                return Err(err(format!(
                    "battlepresent: command8 target {target_index} stage {stage} base unavailable"
                )));
            }
        }
        idles.push(idle);
    }
    for (index, base) in in_.transition_bases.iter().enumerate() {
        if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE {
            return Err(err(format!("battlepresent: command8 transition {index} base unavailable")));
        }
    }
    let last_actor = len64(&actor_effect.frames) - 1;

    let mut out = NativeCommand8EffectSequence { targets: vec![Vec::new(); idles.len()], ..Default::default() };
    let mut state = figani::new_native_command8_state();
    for frame_index in 0..NATIVE_COMMAND8_FRONT_FRAMES {
        let planned = figani::plan_native_command8_draw_frame(state, in_.schedule).go()?;
        let pixels =
            compose_native_command8_orbit_frame(in_.front_base, in_.actor_effect, Some(idles[0]), in_.effect, &planned)
                .wrap(format!("battlepresent: command8 front frame {frame_index}: "))?;
        out.front.push(NativeCommand8RenderedFrame {
            sources: NativeCommandSources {
                base: in_.front_base.to_vec(),
                actor_frame: last_actor,
                target_index: 0,
                after: native_command8_source_layers(&planned.layers),
                phase: "handler-front".into(),
                ..Default::default()
            },
            pixels,
            play_sub1: planned.play_sub1,
            play_sub2: planned.play_sub2,
            ..Default::default()
        });
        state = planned.next;
    }

    let (mut last_idle_frame, mut last_idle_repeat) = (0usize, 0i64);
    for (target_index, &idle) in idles.iter().enumerate() {
        let (planned, next) = figani::build_native_command8_target_sequence(state, in_.schedule).go()?;
        let (mut idle_frame, mut idle_repeat, mut hp_stage) = (0usize, 0i64, 0i64);
        for (frame_index, frame) in planned.iter().enumerate() {
            if frame.hp_stage != 0 {
                hp_stage = frame.hp_stage;
            }
            let base = in_.target_bases[target_index][at(hp_stage)];
            let pixels = compose_native_command8_target_frame(base, &idle.frames[idle_frame], in_.effect, frame)
                .wrap(format!("battlepresent: command8 target {target_index} frame {frame_index}: "))?;
            out.targets[target_index].push(NativeCommand8RenderedFrame {
                sources: NativeCommandSources {
                    base: base.to_vec(),
                    actor_frame: -1,
                    target_index: target_index as i64,
                    target_frame: idle_frame as i64,
                    after: native_command8_source_layers(&frame.layers),
                    phase: "target".into(),
                    ..Default::default()
                },
                pixels,
                hp_stage: frame.hp_stage,
                play_sub1: frame.play_sub1,
                play_sub2: frame.play_sub2,
            });
            if idle.frames[idle_frame].delay <= 0 {
                return Err(err(format!("battlepresent: command8 target {target_index} idle delay unavailable")));
            }
            idle_repeat += 1;
            if idle_repeat >= idle.frames[idle_frame].delay {
                idle_repeat = 0;
                idle_frame = (idle_frame + 1) % idle.frames.len();
            }
        }
        (last_idle_frame, last_idle_repeat) = (idle_frame, idle_repeat);
        state = next;
        if target_index + 1 < idles.len() {
            let (transition, after) =
                figani::build_native_command8_transition_sequence(state, in_.schedule, in_.raw_side).go()?;
            let mut frames = Vec::with_capacity(transition.len());
            for (frame_index, frame) in transition.iter().enumerate() {
                let target = if frame.use_next_target { idles[target_index + 1] } else { idle };
                let pixels = compose_native_command8_transition_frame(
                    in_.transition_bases[target_index],
                    in_.actor_effect,
                    Some(target),
                    in_.effect,
                    frame,
                )
                .wrap(format!("battlepresent: command8 transition {target_index} frame {frame_index}: "))?;
                let which = target_index as i64 + i64::from(frame.use_next_target);
                frames.push(NativeCommand8RenderedFrame {
                    sources: NativeCommandSources {
                        base: in_.transition_bases[target_index].to_vec(),
                        actor_frame: last_actor,
                        target_index: which,
                        target_offset_x: frame.target_offset_x,
                        after: native_command8_source_layers(&frame.layers),
                        phase: "transition".into(),
                        ..Default::default()
                    },
                    pixels,
                    play_sub1: frame.play_sub1,
                    play_sub2: frame.play_sub2,
                    ..Default::default()
                });
            }
            out.transitions.push(frames);
            state = after;
        }
    }

    let (tail, _) = figani::build_native_command8_tail_sequence(state, in_.schedule).go()?;
    let last_idle = idles[idles.len() - 1];
    for (frame_index, frame) in tail.iter().enumerate() {
        let tail_idle = Animation { frames: vec![last_idle.frames[last_idle_frame].clone()], ..Default::default() };
        let pixels =
            compose_native_command8_orbit_frame(in_.tail_base, in_.actor_effect, Some(&tail_idle), in_.effect, frame)
                .wrap(format!("battlepresent: command8 tail frame {frame_index}: "))?;
        out.tail.push(NativeCommand8RenderedFrame {
            sources: NativeCommandSources {
                base: in_.tail_base.to_vec(),
                actor_frame: last_actor,
                target_index: len64(&idles) - 1,
                target_frame: last_idle_frame as i64,
                after: native_command8_source_layers(&frame.layers),
                phase: "handler-tail".into(),
                ..Default::default()
            },
            pixels,
            play_sub1: frame.play_sub1,
            play_sub2: frame.play_sub2,
            ..Default::default()
        });
        if last_idle.frames[last_idle_frame].delay <= 0 {
            return Err(err("battlepresent: command8 tail idle delay unavailable"));
        }
        last_idle_repeat += 1;
        if last_idle_repeat >= last_idle.frames[last_idle_frame].delay {
            last_idle_repeat = 0;
            last_idle_frame = (last_idle_frame + 1) % last_idle.frames.len();
        }
    }
    (out.next_idle_frame, out.next_idle_repeat) = (last_idle_frame as i64, last_idle_repeat);
    Ok(out)
}

#[cfg(test)]
mod native_command8_sequence_test;
