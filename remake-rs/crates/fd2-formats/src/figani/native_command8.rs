//! 指令 8（`0x274B0`）的十六通道狀態機。移植自 Go `native_command8.go`。

use super::{Animation, FiganiError, StateError, fail, msg};
use std::ops::Deref;

pub const NATIVE_COMMAND8_EFFECT_FRAME_COUNT: i64 = 32;
pub const NATIVE_COMMAND8_CHANNEL_COUNT: usize = 16;
pub const NATIVE_COMMAND8_SOUND_RESOURCE: i64 = 90;
pub const NATIVE_COMMAND8_FRONT_FRAMES: i64 = 3;
pub const NATIVE_COMMAND8_TARGET_FRAMES: i64 = 34;
pub const NATIVE_COMMAND8_TRANSITION_FRAMES: i64 = 9;
pub const NATIVE_COMMAND8_TAIL_FRAMES: i64 = 2;
pub const NATIVE_COMMAND8_DAMAGE_STAGES: i64 = 16;

pub(crate) const NATIVE_COMMAND8_FRAME_BASES: [i64; NATIVE_COMMAND8_CHANNEL_COUNT] =
    [0, 8, 24, 16, 8, 0, 24, 16, 0, 8, 24, 16, 8, 0, 24, 16];

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand8PresentationSchedule {
    pub effect_resource: i64,
    pub sound_resource: i64,
    pub frame_bases: [i64; NATIVE_COMMAND8_CHANNEL_COUNT],
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand8State {
    pub counters: [i64; NATIVE_COMMAND8_CHANNEL_COUNT],
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand8Layer {
    pub channel: i64,
    pub frame: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand8Frame {
    pub layers: Vec<NativeCommand8Layer>,
    pub numeric_marker: bool,
    pub play_sub1: bool,
    pub play_sub2: bool,
    pub hp_stage: i64,
    pub next: NativeCommand8State,
}

/// Go 以嵌入 `NativeCommand8Frame` 組成；這裡以 `frame` 欄位加 `Deref` 對應欄位提升。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand8TransitionFrame {
    pub frame: NativeCommand8Frame,
    pub use_next_target: bool,
    pub target_offset_x: i64,
}

impl Deref for NativeCommand8TransitionFrame {
    type Target = NativeCommand8Frame;
    fn deref(&self) -> &NativeCommand8Frame {
        &self.frame
    }
}

pub fn build_native_command8_presentation_schedule(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<NativeCommand8PresentationSchedule, FiganiError> {
    if !effect.is_some_and(|e| e.frames.len() as i64 == NATIVE_COMMAND8_EFFECT_FRAME_COUNT) {
        return Err(FiganiError::Static("figani: command8 effect must contain 32 frames"));
    }
    let resource = if raw_side == 0 { 30 } else { 28 };
    Ok(NativeCommand8PresentationSchedule {
        effect_resource: resource,
        sound_resource: NATIVE_COMMAND8_SOUND_RESOURCE,
        frame_bases: NATIVE_COMMAND8_FRAME_BASES,
    })
}

pub fn new_native_command8_state() -> NativeCommand8State {
    let mut state = NativeCommand8State::default();
    for (channel, counter) in state.counters.iter_mut().enumerate() {
        *counter = -2 * channel as i64;
    }
    state
}

pub fn plan_native_command8_draw_frame(
    state: NativeCommand8State,
    schedule: NativeCommand8PresentationSchedule,
) -> Result<NativeCommand8Frame, FiganiError> {
    if schedule.effect_resource == 0 || schedule.sound_resource != NATIVE_COMMAND8_SOUND_RESOURCE {
        return Err(FiganiError::Static("figani: command8 schedule unavailable"));
    }
    let mut frame = NativeCommand8Frame { next: state, ..Default::default() };
    for (channel, &counter) in state.counters.iter().enumerate() {
        if (0..8).contains(&counter) {
            let index = schedule.frame_bases[channel].wrapping_add(counter);
            if !(0..NATIVE_COMMAND8_EFFECT_FRAME_COUNT).contains(&index) {
                return Err(msg(format!("figani: command8 channel {channel} frame {index} invalid")));
            }
            frame.layers.push(NativeCommand8Layer { channel: channel as i64, frame: index });
        }
        if counter == 0 {
            frame.play_sub1 = true;
        }
        if counter == 4 {
            frame.play_sub2 = true;
        }
        let counter = counter.wrapping_add(1);
        frame.next.counters[channel] = counter;
        if counter == 4 {
            frame.numeric_marker = true;
        }
    }
    Ok(frame)
}

pub fn build_native_command8_target_sequence(
    mut state: NativeCommand8State,
    schedule: NativeCommand8PresentationSchedule,
) -> Result<(Vec<NativeCommand8Frame>, NativeCommand8State), StateError<NativeCommand8State>> {
    let mut frames = Vec::with_capacity(NATIVE_COMMAND8_TARGET_FRAMES as usize);
    let mut stage = 0;
    for _ in 0..NATIVE_COMMAND8_TARGET_FRAMES {
        let mut frame = plan_native_command8_draw_frame(state, schedule).map_err(|e| fail(state, e))?;
        if frame.numeric_marker && stage < NATIVE_COMMAND8_DAMAGE_STAGES {
            stage += 1;
            frame.hp_stage = stage;
        }
        state = frame.next;
        frames.push(frame);
    }
    if stage != NATIVE_COMMAND8_DAMAGE_STAGES {
        return Err(fail(state, msg(format!("figani: command8 HP marker count incomplete: {stage}"))));
    }
    Ok((frames, state))
}

pub fn build_native_command8_transition_sequence(
    mut state: NativeCommand8State,
    schedule: NativeCommand8PresentationSchedule,
    raw_side: u8,
) -> Result<(Vec<NativeCommand8TransitionFrame>, NativeCommand8State), StateError<NativeCommand8State>> {
    let direction: i64 = if raw_side == 0 { 1 } else { -1 };
    let mut frames = Vec::with_capacity(NATIVE_COMMAND8_TRANSITION_FRAMES as usize);
    let steps = (1..=4).map(|s| (false, s)).chain((0..=4).rev().map(|s| (true, s)));
    for (use_next, step) in steps {
        let frame = plan_native_command8_draw_frame(state, schedule).map_err(|e| fail(state, e))?;
        state = frame.next;
        frames.push(NativeCommand8TransitionFrame {
            frame,
            use_next_target: use_next,
            target_offset_x: direction * 35 * step,
        });
    }
    Ok((frames, state))
}

pub fn build_native_command8_tail_sequence(
    mut state: NativeCommand8State,
    schedule: NativeCommand8PresentationSchedule,
) -> Result<(Vec<NativeCommand8Frame>, NativeCommand8State), StateError<NativeCommand8State>> {
    let mut frames = Vec::with_capacity(NATIVE_COMMAND8_TAIL_FRAMES as usize);
    for _ in 0..NATIVE_COMMAND8_TAIL_FRAMES {
        let frame = plan_native_command8_draw_frame(state, schedule).map_err(|e| fail(state, e))?;
        state = frame.next;
        frames.push(frame);
    }
    Ok((frames, state))
}
