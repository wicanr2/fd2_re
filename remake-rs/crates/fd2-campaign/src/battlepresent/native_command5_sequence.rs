//! 移植自 Go `battlepresent/native_command5_sequence.go`。

use super::native_command0::NATIVE_COMMAND0_SURFACE_SIZE;
use super::{
    GoResult, Result, at, compose_native_command5_orbit_frame, compose_native_command5_target_frame,
    compose_native_command5_transition_frame, err, len64,
};
use fd2_formats::figani::{
    self, Animation, NATIVE_COMMAND5_DAMAGE_STAGES, NativeCommand5Layer, NativeCommand5PresentationSchedule,
};

/// NativeCommand5Sources records the inputs already consumed by this present.
/// It never replays the RNG walk or resolves another target.
///
/// Go：`native_command5_sequence.go:12`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand5Sources {
    pub base: Vec<u8>,
    pub actor_frame: i64,
    pub target_index: i64,
    pub target_frame: i64,
    pub target_offset_x: i64,
    pub layers: Vec<NativeCommand5Layer>,
    pub phase: String,
}

/// Go：`native_command5_sequence.go:20`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand5RenderedFrame {
    pub sources: NativeCommand5Sources,
    pub pixels: Vec<u8>,
    pub hp_stage: i64,
    pub play_primary: bool,
    pub play_secondary: bool,
}

/// Go：`native_command5_sequence.go:28`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand5EffectSequence {
    pub front: Vec<NativeCommand5RenderedFrame>,
    pub targets: Vec<Vec<NativeCommand5RenderedFrame>>,
    pub transitions: Vec<Vec<NativeCommand5RenderedFrame>>,
    pub tail: Vec<NativeCommand5RenderedFrame>,
    pub visual_rng: u16,
    pub next_idle_frame: i64,
    pub next_idle_repeat: i64,
}

/// ResolveTarget 的型別（Go `func(index int, rng uint16) (next uint16, hit bool, err error)`）。
pub type NativeCommand5ResolveTarget<'a> = &'a mut dyn FnMut(i64, u16) -> Result<(u16, bool)>;

/// Go：`native_command5_sequence.go:37`。含 `&mut` callback，所以不 derive `Clone`／`Debug`。
#[derive(Default)]
pub struct NativeCommand5EffectInput<'a> {
    pub front_base: &'a [u8],
    pub tail_base: &'a [u8],
    pub target_bases: Vec<Vec<&'a [u8]>>,
    pub transition_bases: Vec<&'a [u8]>,
    pub actor_effect: Option<&'a Animation>,
    pub target_idle: Vec<Option<&'a Animation>>,
    pub effect: Option<&'a Animation>,
    pub schedule: NativeCommand5PresentationSchedule,
    pub raw_side: u8,
    pub visual_rng: u16,
    /// ResolveTarget 在每個目標的 mode 3 之後、第一張目標畫格之前被呼叫，對應
    /// 0x2B114 的 0x1C75E 擲骰；回傳值成為該目標畫格序列起點的亂數。nil 表示
    /// 呼叫端已另行擲骰（六槽相位仍沿 VisualRNG 走）。
    pub resolve_target: Option<NativeCommand5ResolveTarget<'a>>,
}

/// BuildNativeCommand5EffectSequence 先建立完整 handler-owned batch，並讓
/// 六通道 state 依 front→targets→boundaries→tail 原順序持續；任何晚期缺件
/// 都不回傳部分畫格。VisualRNG 是全域 0x4E893 的同一條序列：mode 0 初始化、
/// 目標畫格的 mode 5 重置、0x2AF40 marker 抖動與 ResolveTarget 的命中／傷害
/// 擲骰依原版順序交錯，最後的 VisualRNG 就是整段演出結束時的全域亂數。
///
/// Go：`native_command5_sequence.go:59`
pub fn build_native_command5_effect_sequence(
    mut in_: NativeCommand5EffectInput,
) -> Result<NativeCommand5EffectSequence> {
    let Some(actor_effect) = in_.actor_effect.filter(|a| {
        len64(in_.front_base) == NATIVE_COMMAND0_SURFACE_SIZE
            && len64(in_.tail_base) == NATIVE_COMMAND0_SURFACE_SIZE
            && !a.frames.is_empty()
            && in_.effect.is_some()
            && !in_.target_idle.is_empty()
            && in_.target_bases.len() == in_.target_idle.len()
            && in_.transition_bases.len() + 1 == in_.target_idle.len()
            && in_.schedule.effect_resource != 0
    }) else {
        return Err(err("battlepresent: incomplete command5 effect sequence input"));
    };
    let mut idles = Vec::with_capacity(in_.target_idle.len());
    for (target_index, idle) in in_.target_idle.iter().enumerate() {
        let Some(idle) = idle.filter(|i| {
            !i.frames.is_empty() && len64(&in_.target_bases[target_index]) == NATIVE_COMMAND5_DAMAGE_STAGES + 1
        }) else {
            return Err(err(format!("battlepresent: command5 target {target_index} inputs unavailable")));
        };
        for (stage, base) in in_.target_bases[target_index].iter().enumerate() {
            if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE {
                return Err(err(format!(
                    "battlepresent: command5 target {target_index} stage {stage} base unavailable"
                )));
            }
        }
        idles.push(idle);
    }
    for (index, base) in in_.transition_bases.iter().enumerate() {
        if len64(base) != NATIVE_COMMAND0_SURFACE_SIZE {
            return Err(err(format!("battlepresent: command5 transition {index} base unavailable")));
        }
    }
    let last_actor = len64(&actor_effect.frames) - 1;

    let mut out = NativeCommand5EffectSequence { targets: vec![Vec::new(); idles.len()], ..Default::default() };
    let mut state = figani::new_native_command5_state_for_schedule(in_.visual_rng, in_.schedule);
    for _ in 0..in_.schedule.front_frames {
        let planned = figani::plan_native_command5_draw_frame(state, in_.schedule, in_.raw_side).go()?;
        // Go 這裡直接回傳錯誤，不加 front frame 前綴（native_command5_sequence.go:96）。
        let pixels = compose_native_command5_orbit_frame(
            in_.front_base,
            in_.actor_effect,
            Some(idles[0]),
            in_.effect,
            &planned,
        )?;
        out.front.push(NativeCommand5RenderedFrame {
            sources: NativeCommand5Sources {
                base: in_.front_base.to_vec(),
                actor_frame: last_actor,
                target_index: 0,
                layers: planned.layers.clone(),
                phase: "handler-front".into(),
                ..Default::default()
            },
            pixels,
            play_primary: planned.play_primary,
            play_secondary: planned.play_secondary,
            ..Default::default()
        });
        state = planned.next;
    }

    let (mut last_idle_frame, mut last_idle_repeat) = (0usize, 0i64);
    for (target_index, &idle) in idles.iter().enumerate() {
        let mut hit = true;
        if let Some(resolve) = in_.resolve_target.as_mut() {
            let (next, resolved_hit) = resolve(target_index as i64, state.rng)?;
            (state.rng, hit) = (next, resolved_hit);
        }
        let (planned, next) =
            figani::build_native_command5_target_sequence_hit(state, in_.schedule, in_.raw_side, hit).go()?;
        let (mut idle_frame, mut idle_repeat, mut hp_stage) = (0usize, 0i64, 0i64);
        for (frame_index, frame) in planned.iter().enumerate() {
            if frame.hp_stage != 0 {
                hp_stage = frame.hp_stage;
            }
            let base = in_.target_bases[target_index][at(hp_stage)];
            let pixels = compose_native_command5_target_frame(base, &idle.frames[idle_frame], in_.effect, frame)
                .wrap(format!("battlepresent: command5 target {target_index} frame {frame_index}: "))?;
            out.targets[target_index].push(NativeCommand5RenderedFrame {
                sources: NativeCommand5Sources {
                    base: base.to_vec(),
                    actor_frame: -1,
                    target_index: target_index as i64,
                    target_frame: idle_frame as i64,
                    layers: frame.layers.clone(),
                    phase: "target".into(),
                    ..Default::default()
                },
                pixels,
                hp_stage: frame.hp_stage,
                play_primary: frame.play_primary,
                play_secondary: frame.play_secondary,
            });
            if idle.frames[idle_frame].delay <= 0 {
                return Err(err(format!("battlepresent: command5 target {target_index} idle delay unavailable")));
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
                figani::build_native_command5_transition_sequence(state, in_.schedule, in_.raw_side).go()?;
            let mut frames = Vec::with_capacity(transition.len());
            for (frame_index, frame) in transition.iter().enumerate() {
                let target = if frame.use_next_target { idles[target_index + 1] } else { idle };
                let pixels = compose_native_command5_transition_frame(
                    in_.transition_bases[target_index],
                    in_.actor_effect,
                    Some(target),
                    in_.effect,
                    frame,
                )
                .wrap(format!("battlepresent: command5 transition {target_index} frame {frame_index}: "))?;
                let which = target_index as i64 + i64::from(frame.use_next_target);
                frames.push(NativeCommand5RenderedFrame {
                    sources: NativeCommand5Sources {
                        base: in_.transition_bases[target_index].to_vec(),
                        actor_frame: last_actor,
                        target_index: which,
                        target_offset_x: frame.target_offset_x,
                        layers: frame.layers.clone(),
                        phase: "transition".into(),
                        ..Default::default()
                    },
                    pixels,
                    play_primary: frame.play_primary,
                    play_secondary: frame.play_secondary,
                    ..Default::default()
                });
            }
            out.transitions.push(frames);
            state = after;
        }
    }

    let (tail, last) = figani::build_native_command5_tail_sequence(state, in_.schedule, in_.raw_side).go()?;
    let last_idle = idles[idles.len() - 1];
    for (frame_index, frame) in tail.iter().enumerate() {
        let tail_idle = Animation { frames: vec![last_idle.frames[last_idle_frame].clone()], ..Default::default() };
        let pixels =
            compose_native_command5_orbit_frame(in_.tail_base, in_.actor_effect, Some(&tail_idle), in_.effect, frame)
                .wrap(format!("battlepresent: command5 tail frame {frame_index}: "))?;
        out.tail.push(NativeCommand5RenderedFrame {
            sources: NativeCommand5Sources {
                base: in_.tail_base.to_vec(),
                actor_frame: last_actor,
                target_index: len64(&idles) - 1,
                target_frame: last_idle_frame as i64,
                layers: frame.layers.clone(),
                phase: "handler-tail".into(),
                ..Default::default()
            },
            pixels,
            play_primary: frame.play_primary,
            play_secondary: frame.play_secondary,
            ..Default::default()
        });
        if last_idle.frames[last_idle_frame].delay <= 0 {
            return Err(err("battlepresent: command5 tail idle delay unavailable"));
        }
        last_idle_repeat += 1;
        if last_idle_repeat >= last_idle.frames[last_idle_frame].delay {
            last_idle_repeat = 0;
            last_idle_frame = (last_idle_frame + 1) % last_idle.frames.len();
        }
    }
    out.visual_rng = last.rng;
    (out.next_idle_frame, out.next_idle_repeat) = (last_idle_frame as i64, last_idle_repeat);
    Ok(out)
}

#[cfg(test)]
mod native_command5_sequence_test;
