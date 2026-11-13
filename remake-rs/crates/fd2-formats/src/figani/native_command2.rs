//! 指令 2（`0x26528`）的 helper 幀序列與亂數走訪。移植自 Go `native_command2.go`。

use super::{Animation, FiganiError, Resolver, RngWalkError, fail, frame_shape_ok, msg};
use crate::native_rng::native_rng_step;

pub const NATIVE_COMMAND2_EFFECT_FRAME_COUNT: i64 = 18;
pub const NATIVE_COMMAND2_FRONT_FRAMES: i64 = 29;
pub const NATIVE_COMMAND2_TARGET_FRAMES: i64 = 12;
pub const NATIVE_COMMAND2_DAMAGE_STAGES: i64 = 6;
pub const NATIVE_COMMAND2_TRANSITION_FRAMES: i64 = 9;
pub const NATIVE_COMMAND2_TAIL_FRAMES: i64 = 10;
pub const NATIVE_COMMAND2_SOUND_RESOURCE: i64 = 83;
pub const NATIVE_COMMAND2_SOUND_MODE2_INDEX: i64 = 1;
pub const NATIVE_COMMAND2_SOUND_MODE5_INDEX: i64 = 2;
pub const NATIVE_COMMAND2_SOUND_MODE6_INDEX: i64 = 3;

/// helper 的兩個原始位元組；遞增時與 Go `byte` 一樣環繞。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand2HelperState {
    pub frame: u8,
    pub repeat: u8,
}

/// 保存 `sub_2A6BD` 每個目標的 `0x2B114` 數值呼叫，之後命中目標的每個 mode 5 HP marker 各吃
/// 一次 `0x2AF40` 亂數。未命中在 `0x2AE0C` 略過這些 marker；mode 0/3 與過場不擲骰。
/// 證據：`docs/data/ida/fd2_command2_target_rng_20261002.json`。
pub fn walk_native_command2_rng(mut rng: u16, target_count: i64, resolve: Resolver<'_>) -> Result<u16, RngWalkError> {
    let Some(resolve) = resolve.filter(|_| target_count > 0) else {
        return Err(fail(rng, FiganiError::Static("figani: command2 RNG walk needs targets and a resolver")));
    };
    let (frames, _) = build_native_command2_target_sequence().map_err(|e| fail(rng, e))?;
    for index in 0..target_count {
        let (next, hit) = resolve(index, rng).map_err(|e| fail(rng, FiganiError::Resolver(e)))?;
        rng = next;
        if hit {
            for frame in &frames {
                if frame.numeric_marker {
                    rng = native_rng_step(rng);
                }
            }
        }
    }
    Ok(rng)
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand2HelperFrame {
    pub effect_frame: i64,
    pub next: NativeCommand2HelperState,
    pub sample1: bool,
    pub sample2: bool,
    pub numeric_marker: bool,
    pub hp_stage: i64,
}

/// 保存 `0x26528` 已閉合的資源契約；各 phase 的 frame／repeat 排程由同檔的建構函式產生。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand2PresentationSchedule {
    pub effect_resource: i64,
    pub sound_resource: i64,
    pub sound_indices: [i64; 3],
}

fn advance_native_command2_helper(
    state: NativeCommand2HelperState,
    effect: Option<&Animation>,
) -> Result<NativeCommand2HelperFrame, FiganiError> {
    let delay = effect.and_then(|e| e.frames.get(state.frame as usize)).map(|f| f.delay).filter(|&d| d > 0);
    let Some(delay) = delay else {
        return Err(msg(format!("figani: command2 helper frame {} unavailable", state.frame)));
    };
    let mut next = state;
    next.repeat = next.repeat.wrapping_add(1);
    if next.repeat as i64 >= delay {
        next.repeat = 0;
        next.frame = next.frame.wrapping_add(1);
    }
    Ok(NativeCommand2HelperFrame { effect_frame: state.frame as i64, next, ..Default::default() })
}

pub fn build_native_command2_front_sequence(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<(Vec<NativeCommand2HelperFrame>, NativeCommand2HelperState), FiganiError> {
    let mut state = NativeCommand2HelperState::default();
    let mut frames = Vec::with_capacity(NATIVE_COMMAND2_FRONT_FRAMES as usize);
    for _ in 0..NATIVE_COMMAND2_FRONT_FRAMES {
        if state.frame == 10 {
            state.frame = 15;
        }
        let mut frame = advance_native_command2_helper(state, effect)?;
        frame.sample1 = raw_side == 0 && state.frame == 7;
        frames.push(frame);
        state = frame.next;
    }
    Ok((frames, state))
}

fn plan_native_command2_target_frame(state: NativeCommand2HelperState) -> NativeCommand2HelperFrame {
    let mut frame = NativeCommand2HelperFrame { effect_frame: state.frame as i64, ..Default::default() };
    let mut next = state;
    next.frame = next.frame.wrapping_add(1);
    if next.frame == 17 {
        frame.sample2 = true;
        frame.numeric_marker = true;
    } else if next.frame == 18 {
        next.frame = 16;
    }
    frame.next = next;
    frame
}

pub fn build_native_command2_target_sequence()
-> Result<(Vec<NativeCommand2HelperFrame>, NativeCommand2HelperState), FiganiError> {
    let mut state = NativeCommand2HelperState { frame: 16, repeat: 0 };
    let mut frames = Vec::with_capacity(NATIVE_COMMAND2_TARGET_FRAMES as usize);
    let mut hp_stage = 0;
    for _ in 0..NATIVE_COMMAND2_TARGET_FRAMES {
        let mut frame = plan_native_command2_target_frame(state);
        if frame.numeric_marker {
            hp_stage += 1;
            frame.hp_stage = hp_stage;
        }
        frames.push(frame);
        state = frame.next;
    }
    if hp_stage != NATIVE_COMMAND2_DAMAGE_STAGES {
        return Err(msg(format!("figani: command2 HP marker count={hp_stage}")));
    }
    Ok((frames, state))
}

pub fn build_native_command2_transition_sequence(
    mut state: NativeCommand2HelperState,
) -> Vec<NativeCommand2HelperFrame> {
    let mut frames = Vec::with_capacity(NATIVE_COMMAND2_TRANSITION_FRAMES as usize);
    for _ in 0..NATIVE_COMMAND2_TRANSITION_FRAMES {
        let mut frame = plan_native_command2_target_frame(state);
        frame.numeric_marker = false;
        frame.hp_stage = 0;
        frames.push(frame);
        state = frame.next;
    }
    frames
}

pub fn build_native_command2_tail_sequence(
    raw_side: u8,
    repeat: u8,
    effect: Option<&Animation>,
) -> Result<Vec<NativeCommand2HelperFrame>, FiganiError> {
    let mut state = NativeCommand2HelperState { frame: 10, repeat };
    let mut frames = Vec::with_capacity(NATIVE_COMMAND2_TAIL_FRAMES as usize);
    for _ in 0..NATIVE_COMMAND2_TAIL_FRAMES {
        let mut frame = advance_native_command2_helper(state, effect)?;
        frame.sample1 = raw_side == 0 && state.frame == 7;
        frames.push(frame);
        state = frame.next;
    }
    Ok(frames)
}

pub fn build_native_command2_presentation_schedule(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<NativeCommand2PresentationSchedule, FiganiError> {
    let resource = if raw_side == 0 { 27 } else { 26 };
    let Some(effect) = effect.filter(|e| {
        e.header_byte1 == 0
            && e.header_byte2 as i64 == NATIVE_COMMAND2_EFFECT_FRAME_COUNT
            && e.header_byte4 == 0
            && e.frames.len() as i64 == NATIVE_COMMAND2_EFFECT_FRAME_COUNT
    }) else {
        return Err(msg(format!("figani: command2 FDOTHER #{resource} signature mismatch")));
    };
    for (index, frame) in effect.frames.iter().enumerate() {
        if !frame_shape_ok(frame) {
            return Err(msg(format!("figani: command2 FDOTHER #{resource} frame {index} malformed")));
        }
    }
    Ok(NativeCommand2PresentationSchedule {
        effect_resource: resource,
        sound_resource: NATIVE_COMMAND2_SOUND_RESOURCE,
        sound_indices: [
            NATIVE_COMMAND2_SOUND_MODE2_INDEX,
            NATIVE_COMMAND2_SOUND_MODE5_INDEX,
            NATIVE_COMMAND2_SOUND_MODE6_INDEX,
        ],
    })
}
