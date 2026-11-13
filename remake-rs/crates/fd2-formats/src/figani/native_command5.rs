//! 指令 4（`0x269D3`）與指令 5（`0x26BFD`）共用的六槽狀態機。移植自 Go `native_command5.go`。

use super::{Animation, FiganiError, Resolver, RngWalkError, StateError, fail, frame_shape_ok, msg};
use crate::native_rng::native_rng_step;
use std::ops::Deref;

pub const NATIVE_COMMAND5_EFFECT_FRAME_COUNT: i64 = 12;
pub const NATIVE_COMMAND5_CHANNEL_COUNT: usize = 6;
pub const NATIVE_COMMAND5_POSITION_COUNT: usize = 10;
pub const NATIVE_COMMAND5_SOUND_RESOURCE: i64 = 86;
pub const NATIVE_COMMAND5_FRONT_FRAMES: i64 = 1;
pub const NATIVE_COMMAND5_TARGET_FRAMES: i64 = 12;
pub const NATIVE_COMMAND5_TRANSITION_FRAMES: i64 = 9;
pub const NATIVE_COMMAND5_TAIL_FRAMES: i64 = 8;
pub const NATIVE_COMMAND5_DAMAGE_STAGES: i64 = 6;

const NATIVE_COMMAND5_X_OFFSETS: [i64; NATIVE_COMMAND5_POSITION_COUNT] = [30, 50, 70, 40, 80, 100, 70, 30, 60, 90];

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand5PresentationSchedule {
    pub command_id: i64,
    pub effect_resource: i64,
    pub sound_resource: i64,
    pub sound_indices: [i64; 2],
    pub x_offsets: [i64; NATIVE_COMMAND5_POSITION_COUNT],
    pub effect_frames: i64,
    pub phase_stride: i64,
    pub active_counters: i64,
    pub front_frames: i64,
    pub target_frames: i64,
    pub tail_frames: i64,
    pub marker_counter: i64,
    pub reset_counter: i64,
    pub sample_every_channel: bool,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand5State {
    pub counters: [i64; NATIVE_COMMAND5_CHANNEL_COUNT],
    pub positions: [i64; NATIVE_COMMAND5_CHANNEL_COUNT],
    pub phases: [i64; NATIVE_COMMAND5_CHANNEL_COUNT],
    pub next_position: i64,
    pub stop: bool,
    pub rng: u16,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand5Layer {
    pub channel: i64,
    pub frame: i64,
    pub x: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand5Frame {
    pub layers: Vec<NativeCommand5Layer>,
    pub next: NativeCommand5State,
    pub numeric_marker: bool,
    pub hp_stage: i64,
    pub play_primary: bool,
    pub play_secondary: bool,
}

/// Go 以嵌入 `NativeCommand5Frame` 組成；這裡以 `frame` 欄位加 `Deref` 對應欄位提升。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand5TransitionFrame {
    pub frame: NativeCommand5Frame,
    pub use_next_target: bool,
    pub target_offset_x: i64,
}

impl Deref for NativeCommand5TransitionFrame {
    type Target = NativeCommand5Frame;
    fn deref(&self) -> &NativeCommand5Frame {
        &self.frame
    }
}

fn check_bank(effect: Option<&Animation>, frames: i64, command: i64, resource: i64) -> Result<(), FiganiError> {
    let Some(effect) = effect.filter(|e| {
        e.header_byte1 == 0 && e.header_byte2 as i64 == frames && e.header_byte4 == 0 && e.frames.len() as i64 == frames
    }) else {
        return Err(msg(format!("figani: command{command} FDOTHER #{resource} signature mismatch")));
    };
    for (index, frame) in effect.frames.iter().enumerate() {
        if !frame_shape_ok(frame) {
            return Err(msg(format!("figani: command{command} FDOTHER #{resource} frame {index} malformed")));
        }
    }
    Ok(())
}

pub fn build_native_command5_presentation_schedule(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<NativeCommand5PresentationSchedule, FiganiError> {
    let resource = if raw_side == 0 { 25 } else { 24 };
    check_bank(effect, NATIVE_COMMAND5_EFFECT_FRAME_COUNT, 5, resource)?;
    Ok(NativeCommand5PresentationSchedule {
        command_id: 5,
        effect_resource: resource,
        sound_resource: NATIVE_COMMAND5_SOUND_RESOURCE,
        sound_indices: [0, 1],
        x_offsets: NATIVE_COMMAND5_X_OFFSETS,
        effect_frames: NATIVE_COMMAND5_EFFECT_FRAME_COUNT,
        phase_stride: 6,
        active_counters: 6,
        front_frames: NATIVE_COMMAND5_FRONT_FRAMES,
        target_frames: NATIVE_COMMAND5_TARGET_FRAMES,
        tail_frames: NATIVE_COMMAND5_TAIL_FRAMES,
        marker_counter: 2,
        reset_counter: 7,
        sample_every_channel: false,
    })
}

/// 保存 `0x269D3` 已證實的六槽變體。型別沿用相同的六槽狀態容器，但所有會不同的原版常數都留在
/// schedule，不把第 5 號資源或 marker 偷渡給第 4 號。
pub fn build_native_command4_presentation_schedule(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<NativeCommand5PresentationSchedule, FiganiError> {
    let resource = if raw_side == 0 { 23 } else { 22 };
    check_bank(effect, 14, 4, resource)?;
    Ok(NativeCommand5PresentationSchedule {
        command_id: 4,
        effect_resource: resource,
        sound_resource: 85,
        sound_indices: [0, 1],
        x_offsets: NATIVE_COMMAND5_X_OFFSETS,
        effect_frames: 14,
        phase_stride: 7,
        active_counters: 7,
        front_frames: 2,
        target_frames: 12,
        tail_frames: 8,
        marker_counter: 3,
        reset_counter: 8,
        sample_every_channel: true,
    })
}

/// 重現 mode 0：六個相位各消耗一次 `0x4E893`，並保留下一個循環位置 6。raw side 只影響畫面水平
/// 錨點，不影響狀態。
pub fn new_native_command5_state_for_schedule(
    rng: u16,
    schedule: NativeCommand5PresentationSchedule,
) -> NativeCommand5State {
    let mut state =
        NativeCommand5State { next_position: NATIVE_COMMAND5_CHANNEL_COUNT as i64, rng, ..Default::default() };
    for channel in 0..NATIVE_COMMAND5_CHANNEL_COUNT {
        state.counters[channel] = -2 * channel as i64;
        state.positions[channel] = channel as i64;
        state.rng = native_rng_step(state.rng);
        state.phases[channel] = ((state.rng % 2) as i64).wrapping_mul(schedule.phase_stride);
    }
    state
}

pub fn new_native_command5_state(rng: u16) -> NativeCommand5State {
    new_native_command5_state_for_schedule(
        rng,
        NativeCommand5PresentationSchedule { phase_stride: 6, ..Default::default() },
    )
}

/// 重現 mode 2／5／8 共用的六通道迴圈。mode 1／4／7 不畫 handler effect，故不在此表示。
pub fn plan_native_command5_draw_frame(
    state: NativeCommand5State,
    schedule: NativeCommand5PresentationSchedule,
    raw_side: u8,
) -> Result<NativeCommand5Frame, FiganiError> {
    if schedule.command_id != 4 && schedule.command_id != 5 {
        return Err(FiganiError::Static("figani: command5 schedule unavailable"));
    }
    let (want_resource, x_shift) = match (schedule.command_id, raw_side == 0) {
        (4, false) => (22, 0),
        (4, true) => (23, 143),
        (_, false) => (24, 0),
        (_, true) => (25, 143),
    };
    if schedule.effect_resource != want_resource {
        return Err(FiganiError::Static("figani: command5 side mismatch"));
    }
    let mut next = state;
    let mut frame = NativeCommand5Frame { next, ..Default::default() };
    for channel in 0..NATIVE_COMMAND5_CHANNEL_COUNT {
        let counter = next.counters[channel];
        if counter >= 0 && counter < schedule.active_counters {
            let position = next.positions[channel];
            if position < 0 || position >= schedule.x_offsets.len() as i64 {
                return Err(msg(format!("figani: command5 position {position} unavailable")));
            }
            let effect_frame = next.phases[channel].wrapping_add(counter);
            if effect_frame < 0 || effect_frame >= schedule.effect_frames {
                return Err(msg(format!("figani: command5 effect frame {effect_frame} unavailable")));
            }
            frame.layers.push(NativeCommand5Layer {
                channel: channel as i64,
                frame: effect_frame,
                x: schedule.x_offsets[position as usize].wrapping_add(x_shift),
            });
            if counter == 0 && (schedule.sample_every_channel || channel == 0) {
                frame.play_primary = true;
            }
            if counter == 0 && !schedule.sample_every_channel && channel == 3 {
                frame.play_secondary = true;
            }
        }
        next.counters[channel] = next.counters[channel].wrapping_add(1);
        if next.counters[channel] == schedule.marker_counter {
            frame.numeric_marker = true;
        }
        if next.counters[channel] == schedule.reset_counter && !next.stop {
            next.positions[channel] = next.next_position;
            next.next_position = next.next_position.wrapping_add(1) % NATIVE_COMMAND5_POSITION_COUNT as i64;
            next.counters[channel] = 0;
            next.rng = native_rng_step(next.rng);
            next.phases[channel] = ((next.rng % 2) as i64).wrapping_mul(schedule.phase_stride);
        }
    }
    frame.next = next;
    Ok(frame)
}

pub fn build_native_command5_target_sequence(
    state: NativeCommand5State,
    schedule: NativeCommand5PresentationSchedule,
    raw_side: u8,
) -> Result<(Vec<NativeCommand5Frame>, NativeCommand5State), StateError<NativeCommand5State>> {
    build_native_command5_target_sequence_hit(state, schedule, raw_side, true)
}

/// 目標畫格序列加上 `0x2A6BD` 的命中分支：`0x1C75E` 回 0（未命中）時 `0x2ADFE` 走 `0x2AF61`，
/// mode 5 照樣呼叫（六槽重置照樣吃亂數）但不看回傳值，沒有 HP 分段也沒有 `0x2AF40` 抖動；命中才走
/// `0x2AEB6` 的 marker 分支（r4 收據 seq 1013：指令 0 未命中只吃 `0x1C7F2` 一步）。
/// `schedule.target_frames` 為負時 Go 的 `make` 會 panic，這裡視為零。
pub fn build_native_command5_target_sequence_hit(
    mut state: NativeCommand5State,
    schedule: NativeCommand5PresentationSchedule,
    raw_side: u8,
    hit: bool,
) -> Result<(Vec<NativeCommand5Frame>, NativeCommand5State), StateError<NativeCommand5State>> {
    let mut frames = Vec::with_capacity(schedule.target_frames.clamp(0, 1 << 16) as usize);
    let mut hp_stage = 0;
    for _ in 0..schedule.target_frames {
        let mut frame = plan_native_command5_draw_frame(state, schedule, raw_side).map_err(|e| fail(state, e))?;
        if hit {
            if frame.numeric_marker && hp_stage < NATIVE_COMMAND5_DAMAGE_STAGES {
                hp_stage += 1;
                frame.hp_stage = hp_stage;
            }
            // 0x2AF40：mode 5 回傳 1 的每一張目標畫格都在 handler 之後再吃一次亂數。
            frame = native_command5_marker_shake(frame);
        }
        state = frame.next;
        frames.push(frame);
    }
    if hit && hp_stage != NATIVE_COMMAND5_DAMAGE_STAGES {
        return Err(fail(state, msg(format!("figani: command5 HP marker count incomplete: {hp_stage}"))));
    }
    Ok((frames, state))
}

pub fn build_native_command5_transition_sequence(
    mut state: NativeCommand5State,
    schedule: NativeCommand5PresentationSchedule,
    raw_side: u8,
) -> Result<(Vec<NativeCommand5TransitionFrame>, NativeCommand5State), StateError<NativeCommand5State>> {
    let direction: i64 = if raw_side == 0 { 1 } else { -1 };
    let mut frames = Vec::with_capacity(NATIVE_COMMAND5_TRANSITION_FRAMES as usize);
    let steps = (1..=4).map(|s| (false, s)).chain((0..=4).rev().map(|s| (true, s)));
    for (use_next, step) in steps {
        let planned = plan_native_command5_draw_frame(state, schedule, raw_side).map_err(|e| fail(state, e))?;
        state = planned.next;
        frames.push(NativeCommand5TransitionFrame {
            frame: planned,
            use_next_target: use_next,
            target_offset_x: direction * 35 * step,
        });
    }
    Ok((frames, state))
}

/// `schedule.tail_frames` 為負時 Go 的 `make` 會 panic，這裡視為零。
pub fn build_native_command5_tail_sequence(
    mut state: NativeCommand5State,
    schedule: NativeCommand5PresentationSchedule,
    raw_side: u8,
) -> Result<(Vec<NativeCommand5Frame>, NativeCommand5State), StateError<NativeCommand5State>> {
    state.stop = true;
    let mut frames = Vec::with_capacity(schedule.tail_frames.clamp(0, 1 << 16) as usize);
    for _ in 0..schedule.tail_frames {
        let frame = plan_native_command5_draw_frame(state, schedule, raw_side).map_err(|e| fail(state, e))?;
        state = frame.next;
        frames.push(frame);
    }
    Ok((frames, state))
}

/// 重現 `0x2A6BD` 目標迴圈在 mode 5 回傳 1 之後的 `0x2AF40 call 0x4E893`：每個回傳 numeric
/// marker 的目標畫格都再吃一次全域亂數（餘數只用於畫面抖動位移 `1 - rng%3`）。前段 mode 2 與尾段
/// mode 8 的呼叫端（`0x2AC8E`／`0x2B1B6`）不看回傳值，所以那兩段沒有這一步。
pub fn native_command5_marker_shake(mut frame: NativeCommand5Frame) -> NativeCommand5Frame {
    if frame.numeric_marker {
        frame.next.rng = native_rng_step(frame.next.rng);
    }
    frame
}

/// 只走 `0x269D3` 六槽狀態機與 `0x2A6BD` 目標迴圈的亂數序列，不合成任何畫格：mode 0 初始化（六次）
/// → 前段畫格（可能的重置）→ 每個目標：`resolve` 在 mode 3 之後、第一張目標畫格之前擲命中／傷害
/// （`0x2B114 call 0x1C75E`），接著目標畫格裡 mode 5 的重置與 marker 抖動，目標之間是過場畫格 →
/// 尾段畫格（stop=1，沒有重置與抖動）。回傳整段結束時的亂數（r4 收據 seq 1451：6 次 `0x26A5A`、
/// `0x1C7F2`、`0x1C86E`、`0x2AF45`／`0x26BCD` 交錯共 18 步）。
///
/// Go 版任何錯誤都回傳呼叫端傳入的原始 `rng`，不是走到一半的值；這裡照樣保留。
pub fn walk_native_command5_rng(
    rng: u16,
    schedule: NativeCommand5PresentationSchedule,
    raw_side: u8,
    target_count: i64,
    resolve: Resolver<'_>,
) -> Result<u16, RngWalkError> {
    let Some(resolve) = resolve.filter(|_| target_count > 0) else {
        return Err(fail(rng, FiganiError::Static("figani: command5 RNG walk needs targets and a resolver")));
    };
    let mut state = new_native_command5_state_for_schedule(rng, schedule);
    for _ in 0..schedule.front_frames {
        let frame = plan_native_command5_draw_frame(state, schedule, raw_side).map_err(|e| fail(rng, e))?;
        state = frame.next;
    }
    for target in 0..target_count {
        let (next, hit) = resolve(target, state.rng).map_err(|e| fail(rng, FiganiError::Resolver(e)))?;
        state.rng = next;
        let (_, after) = build_native_command5_target_sequence_hit(state, schedule, raw_side, hit)
            .map_err(|e| fail(rng, e.error))?;
        state = after;
        if target + 1 < target_count {
            let (_, after) =
                build_native_command5_transition_sequence(state, schedule, raw_side).map_err(|e| fail(rng, e.error))?;
            state = after;
        }
    }
    let (_, last) = build_native_command5_tail_sequence(state, schedule, raw_side).map_err(|e| fail(rng, e.error))?;
    Ok(last.rng)
}
