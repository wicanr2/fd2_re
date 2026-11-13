//! 指令 7（`0x272B8`）的 toggle 雙張與三通道狀態機。移植自 Go `native_command7.go`。

use super::{Animation, FiganiError, Resolver, RngWalkError, StateError, fail, frame_shape_ok, msg};
use crate::native_rng::native_rng_step;
use std::ops::Deref;

pub const NATIVE_COMMAND7_EFFECT_FRAME_COUNT: i64 = 5;
pub const NATIVE_COMMAND7_INIT_CHANNEL_COUNT: usize = 4;
pub const NATIVE_COMMAND7_DRAW_CHANNEL_COUNT: usize = 3;
pub const NATIVE_COMMAND7_POSITION_COUNT: usize = 10;
pub const NATIVE_COMMAND7_SOUND_RESOURCE: i64 = 88;
pub const NATIVE_COMMAND7_FRONT_FRAMES: i64 = 2;
pub const NATIVE_COMMAND7_TARGET_FRAMES: i64 = 32;
pub const NATIVE_COMMAND7_TRANSITION_FRAMES: i64 = 9;
pub const NATIVE_COMMAND7_TAIL_FRAMES: i64 = 16;
pub const NATIVE_COMMAND7_DAMAGE_STAGES: i64 = 5;

const NATIVE_COMMAND7_X_OFFSETS: [i64; NATIVE_COMMAND7_POSITION_COUNT] = [30, -10, 70, 20, 100, 130, 40, 80, 110, 60];

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand7PresentationSchedule {
    pub effect_resource: i64,
    pub sound_resource: i64,
    pub sound_indices: [i64; 2],
    pub x_offsets: [i64; NATIVE_COMMAND7_POSITION_COUNT],
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand7State {
    pub counters: [i64; NATIVE_COMMAND7_INIT_CHANNEL_COUNT],
    pub positions: [i64; NATIVE_COMMAND7_INIT_CHANNEL_COUNT],
    pub next_position: i64,
    pub stop: bool,
    pub toggle: u8,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand7Layer {
    pub channel: i64,
    pub frame: i64,
    pub x: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand7Frame {
    pub layers: Vec<NativeCommand7Layer>,
    pub next: NativeCommand7State,
    pub numeric_marker: bool,
    pub hp_stage: i64,
    pub play_handle_a: bool,
    pub play_handle_b: bool,
}

/// Go 以嵌入 `NativeCommand7Frame` 組成；這裡以 `frame` 欄位加 `Deref` 對應欄位提升。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand7TransitionFrame {
    pub frame: NativeCommand7Frame,
    pub use_next_target: bool,
    pub target_offset_x: i64,
}

impl Deref for NativeCommand7TransitionFrame {
    type Target = NativeCommand7Frame;
    fn deref(&self) -> &NativeCommand7Frame {
        &self.frame
    }
}

pub fn build_native_command7_presentation_schedule(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<NativeCommand7PresentationSchedule, FiganiError> {
    let resource = if raw_side == 0 { 38 } else { 37 };
    let Some(effect) = effect.filter(|e| {
        e.header_byte1 == 0
            && e.header_byte2 as i64 == NATIVE_COMMAND7_EFFECT_FRAME_COUNT
            && e.header_byte4 == 0
            && e.frames.len() as i64 == NATIVE_COMMAND7_EFFECT_FRAME_COUNT
    }) else {
        return Err(msg(format!("figani: command7 FDOTHER #{resource} signature mismatch")));
    };
    for (index, frame) in effect.frames.iter().enumerate() {
        if !frame_shape_ok(frame) {
            return Err(msg(format!("figani: command7 FDOTHER #{resource} frame {index} malformed")));
        }
    }
    Ok(NativeCommand7PresentationSchedule {
        effect_resource: resource,
        sound_resource: NATIVE_COMMAND7_SOUND_RESOURCE,
        sound_indices: [0, 1],
        x_offsets: NATIVE_COMMAND7_X_OFFSETS,
    })
}

pub fn new_native_command7_state() -> NativeCommand7State {
    let mut state =
        NativeCommand7State { next_position: NATIVE_COMMAND7_INIT_CHANNEL_COUNT as i64, ..Default::default() };
    for channel in 0..NATIVE_COMMAND7_INIT_CHANNEL_COUNT {
        state.counters[channel] = -3 * channel as i64;
        state.positions[channel] = channel as i64;
    }
    state
}

/// 重現 mode 2／5／8 的 toggle 雙張與三通道迴圈。mode 1／4／7 不畫 handler effect。
pub fn plan_native_command7_draw_frame(
    state: NativeCommand7State,
    schedule: NativeCommand7PresentationSchedule,
    raw_side: u8,
) -> Result<NativeCommand7Frame, FiganiError> {
    let (want_resource, x_shift) = if raw_side == 0 { (38, 130) } else { (37, 0) };
    if schedule.effect_resource != want_resource {
        return Err(FiganiError::Static("figani: command7 side mismatch"));
    }
    let mut next = state;
    next.toggle = next.toggle.wrapping_add(1) % 2;
    let mut frame = NativeCommand7Frame::default();
    for channel in 0..NATIVE_COMMAND7_DRAW_CHANNEL_COUNT {
        let counter = next.counters[channel];
        if (0..NATIVE_COMMAND7_EFFECT_FRAME_COUNT).contains(&counter) {
            let position = next.positions[channel];
            if position < 0 || position >= schedule.x_offsets.len() as i64 {
                return Err(msg(format!("figani: command7 position {position} unavailable")));
            }
            frame.layers.push(NativeCommand7Layer {
                channel: channel as i64,
                frame: counter,
                x: schedule.x_offsets[position as usize].wrapping_add(x_shift),
            });
        }
        if next.toggle != 0 {
            continue;
        }
        if counter == 1 && channel == 0 {
            frame.play_handle_a = true;
        }
        if counter == 1 && channel == 1 {
            frame.play_handle_b = true;
        }
        next.counters[channel] = next.counters[channel].wrapping_add(1);
        if next.counters[channel] == 2 {
            frame.numeric_marker = true;
        }
        if next.counters[channel] == 7 && !next.stop {
            next.next_position = next.next_position.wrapping_add(1) % NATIVE_COMMAND7_POSITION_COUNT as i64;
            next.positions[channel] = next.next_position;
            next.counters[channel] = 0;
        }
    }
    frame.next = next;
    Ok(frame)
}

pub fn build_native_command7_target_sequence(
    mut state: NativeCommand7State,
    schedule: NativeCommand7PresentationSchedule,
    raw_side: u8,
) -> Result<(Vec<NativeCommand7Frame>, NativeCommand7State), StateError<NativeCommand7State>> {
    let mut frames = Vec::with_capacity(NATIVE_COMMAND7_TARGET_FRAMES as usize);
    let mut hp_stage = 0;
    for _ in 0..NATIVE_COMMAND7_TARGET_FRAMES {
        let mut frame = plan_native_command7_draw_frame(state, schedule, raw_side).map_err(|e| fail(state, e))?;
        if frame.numeric_marker && hp_stage < NATIVE_COMMAND7_DAMAGE_STAGES {
            hp_stage += 1;
            frame.hp_stage = hp_stage;
        }
        state = frame.next;
        frames.push(frame);
    }
    if hp_stage != NATIVE_COMMAND7_DAMAGE_STAGES {
        return Err(fail(state, msg(format!("figani: command7 HP marker count incomplete: {hp_stage}"))));
    }
    Ok((frames, state))
}

pub fn build_native_command7_transition_sequence(
    mut state: NativeCommand7State,
    schedule: NativeCommand7PresentationSchedule,
    raw_side: u8,
) -> Result<(Vec<NativeCommand7TransitionFrame>, NativeCommand7State), StateError<NativeCommand7State>> {
    let direction: i64 = if raw_side == 0 { 1 } else { -1 };
    let mut frames = Vec::with_capacity(NATIVE_COMMAND7_TRANSITION_FRAMES as usize);
    let steps = (1..=4).map(|s| (false, s)).chain((0..=4).rev().map(|s| (true, s)));
    for (use_next, step) in steps {
        let planned = plan_native_command7_draw_frame(state, schedule, raw_side).map_err(|e| fail(state, e))?;
        state = planned.next;
        frames.push(NativeCommand7TransitionFrame {
            frame: planned,
            use_next_target: use_next,
            target_offset_x: direction * 35 * step,
        });
    }
    Ok((frames, state))
}

pub fn build_native_command7_tail_sequence(
    mut state: NativeCommand7State,
    schedule: NativeCommand7PresentationSchedule,
    raw_side: u8,
) -> Result<(Vec<NativeCommand7Frame>, NativeCommand7State), StateError<NativeCommand7State>> {
    state.stop = true;
    let mut frames = Vec::with_capacity(NATIVE_COMMAND7_TAIL_FRAMES as usize);
    for _ in 0..NATIVE_COMMAND7_TAIL_FRAMES {
        let frame = plan_native_command7_draw_frame(state, schedule, raw_side).map_err(|e| fail(state, e))?;
        state = frame.next;
        frames.push(frame);
    }
    Ok((frames, state))
}

/// 保存 `sub_2A6BD` 每個目標的數值呼叫，之後命中目標的每個 mode 5 marker 各擲一次 `0x2AF40`，
/// 包含五個 HP 分段之後的 marker。未命中仍推進 handler 但略過這些擲骰。
/// 證據：`docs/data/ida/fd2_command7_target_rng_20261004.json`。
pub fn walk_native_command7_rng(
    mut rng: u16,
    schedule: NativeCommand7PresentationSchedule,
    raw_side: u8,
    target_count: i64,
    resolve: Resolver<'_>,
) -> Result<u16, RngWalkError> {
    let Some(resolve) = resolve.filter(|_| target_count > 0) else {
        return Err(fail(rng, FiganiError::Static("figani: command7 RNG walk needs targets and a resolver")));
    };
    let mut state = new_native_command7_state();
    for _ in 0..NATIVE_COMMAND7_FRONT_FRAMES {
        let frame = plan_native_command7_draw_frame(state, schedule, raw_side).map_err(|e| fail(rng, e))?;
        state = frame.next;
    }
    for index in 0..target_count {
        let (next, hit) = resolve(index, rng).map_err(|e| fail(rng, FiganiError::Resolver(e)))?;
        rng = next;
        let (frames, after) =
            build_native_command7_target_sequence(state, schedule, raw_side).map_err(|e| fail(rng, e.error))?;
        state = after;
        if hit {
            for frame in &frames {
                if frame.numeric_marker {
                    rng = native_rng_step(rng);
                }
            }
        }
        if index + 1 < target_count {
            let (_, after) =
                build_native_command7_transition_sequence(state, schedule, raw_side).map_err(|e| fail(rng, e.error))?;
            state = after;
        }
    }
    build_native_command7_tail_sequence(state, schedule, raw_side).map_err(|e| fail(rng, e.error))?;
    Ok(rng)
}
