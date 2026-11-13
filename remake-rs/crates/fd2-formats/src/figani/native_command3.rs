//! 指令 3（`0x26795`）的十二通道狀態機。移植自 Go `native_command3.go`。

use super::{Animation, FiganiError, StateError, fail, msg};
use std::ops::Deref;

pub const NATIVE_COMMAND3_EFFECT_FRAME_COUNT: i64 = 33;
pub const NATIVE_COMMAND3_CHANNEL_COUNT: usize = 12;
pub const NATIVE_COMMAND3_POSITION_COUNT: usize = 12;
pub const NATIVE_COMMAND3_SOUND_RESOURCE: i64 = 84;
pub const NATIVE_COMMAND3_FRONT_FRAMES: i64 = 2;
pub const NATIVE_COMMAND3_TARGET_FRAMES: i64 = 40;
pub const NATIVE_COMMAND3_TRANSITION_FRAMES: i64 = 9;
pub const NATIVE_COMMAND3_TAIL_FRAMES: i64 = 20;
pub const NATIVE_COMMAND3_DAMAGE_STAGES: i64 = 13;

const NATIVE_COMMAND3_X_OFFSETS: [i64; 12] = [30, 0, 70, 40, 130, 70, -30, 30, 110, 80, -10, 30];
const NATIVE_COMMAND3_Y_OFFSETS: [i64; 12] = [0, 10, 0, 20, 5, 20, 0, 10, 0, 18, 0, 10];
const NATIVE_COMMAND3_FRAME_BASES: [i64; 12] = [22, 0, 0, 0, 0, 0, 11, 0, 22, 0, 0, 0];

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand3PresentationSchedule {
    pub effect_resource: i64,
    pub sound_resource: i64,
    pub x_offsets: [i64; NATIVE_COMMAND3_POSITION_COUNT],
    pub y_offsets: [i64; NATIVE_COMMAND3_POSITION_COUNT],
    pub frame_bases: [i64; NATIVE_COMMAND3_POSITION_COUNT],
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand3State {
    pub counters: [i64; NATIVE_COMMAND3_CHANNEL_COUNT],
    pub positions: [i64; NATIVE_COMMAND3_CHANNEL_COUNT],
    pub next_position: i64,
    pub stop: bool,
    pub toggle: i64,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand3Layer {
    pub channel: i64,
    pub frame: i64,
    pub x: i64,
    pub y: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand3Frame {
    pub layers: Vec<NativeCommand3Layer>,
    pub numeric_marker: bool,
    pub play_sub1: bool,
    pub play_sub2: bool,
    pub hp_stage: i64,
    pub next: NativeCommand3State,
}

/// Go 以嵌入 `NativeCommand3Frame` 組成；這裡以 `frame` 欄位加 `Deref` 對應欄位提升。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand3TransitionFrame {
    pub frame: NativeCommand3Frame,
    pub use_next_target: bool,
    pub target_offset_x: i64,
}

impl Deref for NativeCommand3TransitionFrame {
    type Target = NativeCommand3Frame;
    fn deref(&self) -> &NativeCommand3Frame {
        &self.frame
    }
}

pub fn build_native_command3_presentation_schedule(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<NativeCommand3PresentationSchedule, FiganiError> {
    if !effect.is_some_and(|e| e.frames.len() as i64 == NATIVE_COMMAND3_EFFECT_FRAME_COUNT) {
        return Err(FiganiError::Static("figani: command3 effect must contain 33 frames"));
    }
    let resource = if raw_side == 0 { 43 } else { 39 };
    Ok(NativeCommand3PresentationSchedule {
        effect_resource: resource,
        sound_resource: NATIVE_COMMAND3_SOUND_RESOURCE,
        x_offsets: NATIVE_COMMAND3_X_OFFSETS,
        y_offsets: NATIVE_COMMAND3_Y_OFFSETS,
        frame_bases: NATIVE_COMMAND3_FRAME_BASES,
    })
}

pub fn new_native_command3_state() -> NativeCommand3State {
    let mut state = NativeCommand3State { next_position: NATIVE_COMMAND3_POSITION_COUNT as i64, ..Default::default() };
    for channel in 0..NATIVE_COMMAND3_CHANNEL_COUNT {
        state.counters[channel] = -2 * channel as i64;
        state.positions[channel] = channel as i64;
    }
    state
}

pub fn plan_native_command3_draw_frame(
    state: NativeCommand3State,
    schedule: NativeCommand3PresentationSchedule,
    raw_side: u8,
) -> Result<NativeCommand3Frame, FiganiError> {
    if schedule.effect_resource == 0 || schedule.sound_resource != NATIVE_COMMAND3_SOUND_RESOURCE {
        return Err(FiganiError::Static("figani: command3 schedule unavailable"));
    }
    let mut next = state;
    next.toggle = next.toggle.wrapping_add(1) % 2;
    let mut frame = NativeCommand3Frame { next, ..Default::default() };
    for channel in 0..NATIVE_COMMAND3_CHANNEL_COUNT {
        let position = state.positions[channel];
        if position < 0 || position >= NATIVE_COMMAND3_POSITION_COUNT as i64 {
            return Err(msg(format!("figani: command3 channel {channel} position {position} invalid")));
        }
        let counter = state.counters[channel];
        if (0..11).contains(&counter) {
            let mut x = schedule.x_offsets[position as usize];
            if raw_side == 0 {
                x = x.wrapping_add(20);
            }
            frame.layers.push(NativeCommand3Layer {
                channel: channel as i64,
                frame: schedule.frame_bases[position as usize].wrapping_add(counter),
                x,
                y: schedule.y_offsets[position as usize].wrapping_neg(),
            });
        }
    }
    if next.toggle != 0 {
        frame.next = next;
        return Ok(frame);
    }
    for channel in 0..NATIVE_COMMAND3_CHANNEL_COUNT {
        // Go 在此不再檢查 next.Positions 的範圍；上一個迴圈已確認 state.Positions，而這個迴圈只在
        // 同一通道的 counter 重置時改寫位置，所以索引仍在範圍內。
        let position = next.positions[channel] as usize;
        let mut counter = next.counters[channel];
        if counter == 0 && schedule.frame_bases[position] != 0 {
            frame.play_sub2 = true;
        }
        counter = counter.wrapping_add(1);
        next.counters[channel] = counter;
        if counter == 3 {
            frame.numeric_marker = true;
            if schedule.frame_bases[position] == 0 {
                frame.play_sub1 = true;
            }
        }
        if counter == 11 && !next.stop {
            next.next_position = next.next_position.wrapping_add(1) % NATIVE_COMMAND3_POSITION_COUNT as i64;
            next.positions[channel] = next.next_position;
            next.counters[channel] = 0;
        }
    }
    frame.next = next;
    Ok(frame)
}

pub fn build_native_command3_target_sequence(
    mut state: NativeCommand3State,
    schedule: NativeCommand3PresentationSchedule,
    raw_side: u8,
) -> Result<(Vec<NativeCommand3Frame>, NativeCommand3State), StateError<NativeCommand3State>> {
    let mut frames = Vec::with_capacity(NATIVE_COMMAND3_TARGET_FRAMES as usize);
    let mut hp_stage = 0;
    for _ in 0..NATIVE_COMMAND3_TARGET_FRAMES {
        let mut frame = plan_native_command3_draw_frame(state, schedule, raw_side).map_err(|e| fail(state, e))?;
        if frame.numeric_marker && hp_stage < NATIVE_COMMAND3_DAMAGE_STAGES {
            hp_stage += 1;
            frame.hp_stage = hp_stage;
        }
        state = frame.next;
        frames.push(frame);
    }
    if hp_stage != NATIVE_COMMAND3_DAMAGE_STAGES {
        return Err(fail(state, msg(format!("figani: command3 HP marker count incomplete: {hp_stage}"))));
    }
    Ok((frames, state))
}

pub fn build_native_command3_transition_sequence(
    mut state: NativeCommand3State,
    schedule: NativeCommand3PresentationSchedule,
    raw_side: u8,
) -> Result<(Vec<NativeCommand3TransitionFrame>, NativeCommand3State), StateError<NativeCommand3State>> {
    let direction: i64 = if raw_side == 0 { 1 } else { -1 };
    let mut frames = Vec::with_capacity(NATIVE_COMMAND3_TRANSITION_FRAMES as usize);
    let steps = (1..=4).map(|s| (false, s)).chain((0..=4).rev().map(|s| (true, s)));
    for (use_next, step) in steps {
        let planned = plan_native_command3_draw_frame(state, schedule, raw_side).map_err(|e| fail(state, e))?;
        state = planned.next;
        frames.push(NativeCommand3TransitionFrame {
            frame: planned,
            use_next_target: use_next,
            target_offset_x: direction * 35 * step,
        });
    }
    Ok((frames, state))
}

pub fn build_native_command3_tail_sequence(
    mut state: NativeCommand3State,
    schedule: NativeCommand3PresentationSchedule,
    raw_side: u8,
) -> Result<(Vec<NativeCommand3Frame>, NativeCommand3State), StateError<NativeCommand3State>> {
    state.stop = true;
    let mut frames = Vec::with_capacity(NATIVE_COMMAND3_TAIL_FRAMES as usize);
    for _ in 0..NATIVE_COMMAND3_TAIL_FRAMES {
        let frame = plan_native_command3_draw_frame(state, schedule, raw_side).map_err(|e| fail(state, e))?;
        state = frame.next;
        frames.push(frame);
    }
    Ok((frames, state))
}
