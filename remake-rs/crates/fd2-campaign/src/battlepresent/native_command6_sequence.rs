//! 移植自 Go `battlepresent/native_command6_sequence.go`。

use super::native_command_sources::native_command6_source_layers;
use super::native_command0::NATIVE_COMMAND0_SURFACE_SIZE;
use super::{
    GoResult, NativeCommandSources, Result, at, compose_native_command6_orbit_frame,
    compose_native_command6_target_frame, compose_native_command6_transition_frame, err, len64,
};
use fd2_formats::figani::{self, Animation, NATIVE_COMMAND6_DAMAGE_STAGES, NativeCommand6PresentationSchedule};

/// Go：`native_command6_sequence.go:10`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand6TargetSequence {
    pub frames: Vec<Vec<u8>>,
    pub hp_stages: Vec<i64>,
    pub sources: Vec<NativeCommandSources>,
}

/// Go：`native_command6_sequence.go:16`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand6EffectSequence {
    pub rng_after: u16,
    pub front: Vec<Vec<u8>>,
    pub targets: Vec<NativeCommand6TargetSequence>,
    pub transitions: Vec<Vec<Vec<u8>>>,
    pub tail: Vec<Vec<u8>>,
    pub front_sources: Vec<NativeCommandSources>,
    pub tail_sources: Vec<NativeCommandSources>,
    pub transition_sources: Vec<Vec<NativeCommandSources>>,
}

/// Go：`native_command6_sequence.go:26`
#[derive(Debug, Clone, Default)]
pub struct NativeCommand6EffectInput<'a> {
    pub target_hits: Vec<bool>,
    pub target_numeric_rng: Vec<u16>,
    pub front_base: &'a [u8],
    pub tail_base: &'a [u8],
    pub target_bases: Vec<Vec<&'a [u8]>>,
    pub transition_bases: Vec<&'a [u8]>,
    pub actor_effect: Option<&'a Animation>,
    pub target_idle: Vec<Option<&'a Animation>>,
    pub effect: Option<&'a Animation>,
    pub schedule: NativeCommand6PresentationSchedule,
    pub raw_side: u8,
}

/// BuildNativeCommand6EffectSequence preflights every handler-owned command6
/// frame before the caller publishes MP. Common 0x29164 and 0x2B659 phases are
/// intentionally supplied by their existing scene owner.
///
/// Go：`native_command6_sequence.go:42`
pub fn build_native_command6_effect_sequence(in_: &NativeCommand6EffectInput) -> Result<NativeCommand6EffectSequence> {
    let Some(actor_effect) = in_.actor_effect.filter(|a| {
        len64(in_.front_base) == NATIVE_COMMAND0_SURFACE_SIZE
            && len64(in_.tail_base) == NATIVE_COMMAND0_SURFACE_SIZE
            && !a.frames.is_empty()
            && in_.effect.is_some()
            && !in_.target_idle.is_empty()
            && in_.target_bases.len() == in_.target_idle.len()
            && in_.target_hits.len() == in_.target_idle.len()
            && in_.target_numeric_rng.len() == in_.target_idle.len()
            && in_.transition_bases.len() + 1 == in_.target_idle.len()
            && in_.schedule.effect_resource != 0
    }) else {
        return Err(err("battlepresent: incomplete command6 effect sequence input"));
    };
    let mut idles = Vec::with_capacity(in_.target_idle.len());
    for (index, idle) in in_.target_idle.iter().enumerate() {
        let Some(idle) = idle
            .filter(|i| !i.frames.is_empty() && len64(&in_.target_bases[index]) == NATIVE_COMMAND6_DAMAGE_STAGES + 1)
        else {
            return Err(err(format!("battlepresent: command6 target {index} inputs unavailable")));
        };
        for (stage, base) in in_.target_bases[index].iter().enumerate() {
            if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE {
                return Err(err(format!("battlepresent: command6 target {index} stage {stage} base unavailable")));
            }
        }
        idles.push(idle);
    }
    for (index, base) in in_.transition_bases.iter().enumerate() {
        if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE {
            return Err(err(format!("battlepresent: command6 transition {index} base unavailable")));
        }
    }
    let last_actor = actor_effect.frames.len() - 1;
    let mut out = NativeCommand6EffectSequence {
        targets: vec![NativeCommand6TargetSequence::default(); idles.len()],
        ..Default::default()
    };

    let mut front_base = in_.front_base.to_vec();
    idles[0].frames[0].blit_at(&mut front_base, 320).wrap("battlepresent: command6 front target: ")?;
    let mut radius = 0;
    for _ in 0..7 {
        let planned = figani::plan_native_command6_prelude_frame(radius, in_.raw_side, in_.schedule).go()?;
        let pixels = compose_native_command6_orbit_frame(&front_base, in_.actor_effect, None, in_.effect, &planned)?;
        out.front.push(pixels);
        out.front_sources.push(NativeCommandSources {
            base: in_.front_base.to_vec(),
            actor_frame: last_actor as i64,
            target_index: 0,
            target_before_effects: true,
            before: native_command6_source_layers(&planned.first),
            after: native_command6_source_layers(&planned.second),
            phase: "handler-front".into(),
            ..Default::default()
        });
        radius = planned.next_radius;
    }
    if radius != 42 {
        return Err(err(format!("battlepresent: command6 front radius={radius}")));
    }
    let points = figani::native_command6_target_coordinates(
        figani::native_command6_coordinates(36, in_.schedule.base_byte),
        radius,
    );
    let mut state = figani::new_native_command6_target_state();

    let mut last_target_frame = 0usize;
    let mut display_state = figani::new_native_command6_display_state();
    for (target_index, &idle) in idles.iter().enumerate() {
        let planned =
            figani::build_native_command6_target_sequence_from_state(state, in_.schedule, points, in_.raw_side).go()?;

        let displays = figani::build_native_command6_target_display_frames(
            display_state,
            &planned,
            in_.target_hits[target_index],
            in_.target_numeric_rng[target_index],
        )
        .go()?;
        // Go 對空的 displays 取 displays[len-1] 會 panic；Rust 的索引同樣 panic。
        display_state = displays[displays.len() - 1].next;
        out.rng_after = displays[displays.len() - 1].rng_after;
        let (mut idle_frame, mut idle_repeat, mut hp_stage) = (0usize, 0i64, 0i64);
        for (frame_index, frame) in planned.iter().enumerate() {
            let base = in_.target_bases[target_index][at(hp_stage)];
            let pixels = compose_native_command6_target_frame(
                base,
                &actor_effect.frames[last_actor],
                &idle.frames[idle_frame],
                in_.effect,
                frame,
                displays[frame_index],
            )
            .wrap(format!("battlepresent: command6 target {target_index} frame {frame_index}: "))?;
            out.targets[target_index].frames.push(pixels);
            let mut sources = NativeCommandSources {
                base: base.to_vec(),
                actor_frame: last_actor as i64,
                target_index: target_index as i64,
                target_frame: idle_frame as i64,
                target_offset_x: displays[frame_index].offset_x,
                target_offset_y: displays[frame_index].offset_y,
                before: native_command6_source_layers(&frame.mode4),
                after: native_command6_source_layers(&frame.mode5),
                phase: "target".into(),
                ..Default::default()
            };
            if displays[frame_index].shader != -1 {
                sources.packed_target_shader = displays[frame_index].shader;
            }
            out.targets[target_index].sources.push(sources);
            out.targets[target_index].hp_stages.push(frame.hp_stage);
            if frame.hp_stage != 0 {
                hp_stage = frame.hp_stage;
            }
            idle_repeat += 1;
            if idle.frames[idle_frame].delay <= 0 {
                return Err(err(format!("battlepresent: command6 target {target_index} idle delay unavailable")));
            }
            if idle_repeat >= idle.frames[idle_frame].delay {
                idle_repeat = 0;
                idle_frame = (idle_frame + 1) % idle.frames.len();
            }
        }
        last_target_frame = idle_frame;
        state = planned[planned.len() - 1].next;
        if target_index + 1 < idles.len() {
            let transition_plan =
                figani::build_native_command6_transition_sequence(state, in_.schedule, points, in_.raw_side).go()?;
            let mut frames = Vec::with_capacity(transition_plan.len());
            let mut sources = Vec::with_capacity(transition_plan.len());
            for (frame_index, frame) in transition_plan.iter().enumerate() {
                let target = if frame.use_next_target { idles[target_index + 1] } else { idle };
                let pixels = compose_native_command6_transition_frame(
                    in_.transition_bases[target_index],
                    in_.actor_effect,
                    Some(target),
                    in_.effect,
                    frame,
                )
                .wrap(format!("battlepresent: command6 transition {target_index} frame {frame_index}: "))?;
                frames.push(pixels);
                let which = target_index as i64 + i64::from(frame.use_next_target);
                sources.push(NativeCommandSources {
                    base: in_.transition_bases[target_index].to_vec(),
                    actor_frame: last_actor as i64,
                    target_index: which,
                    target_offset_x: frame.target_offset_x,
                    before: native_command6_source_layers(&frame.mode4),
                    after: native_command6_source_layers(&frame.mode5),
                    phase: "transition".into(),
                    ..Default::default()
                });
            }
            out.transitions.push(frames);
            out.transition_sources.push(sources);
            state = transition_plan[transition_plan.len() - 1].next;
        }
    }

    radius = 42;
    let last_idle = idles[idles.len() - 1];
    for frame_index in 0..7 {
        let planned = figani::plan_native_command6_tail_frame(radius, in_.raw_side, in_.schedule).go()?;
        let pixels = compose_native_command6_orbit_frame(
            in_.tail_base,
            in_.actor_effect,
            Some(&last_idle.frames[last_target_frame]),
            in_.effect,
            &planned,
        )
        .wrap(format!("battlepresent: command6 tail frame {frame_index}: "))?;
        out.tail.push(pixels);
        out.tail_sources.push(NativeCommandSources {
            base: in_.tail_base.to_vec(),
            actor_frame: last_actor as i64,
            target_index: len64(&idles) - 1,
            target_frame: last_target_frame as i64,
            omit_target: !planned.draw_target,
            before: native_command6_source_layers(&planned.first),
            after: native_command6_source_layers(&planned.second),
            phase: "handler-tail".into(),
            ..Default::default()
        });
        radius = planned.next_radius;
    }
    if radius != 0 {
        return Err(err(format!("battlepresent: command6 tail radius={radius}")));
    }
    Ok(out)
}

#[cfg(test)]
mod native_command6_sequence_test;
