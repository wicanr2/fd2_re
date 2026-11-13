//! 指令 9 AI 演出（`0x275D6`）。移植自 Go `native_command9.go`。

use super::{Animation, FiganiError, StateError, fail};

pub const NATIVE_COMMAND9_AI_COMMAND_ID: i64 = 9;
pub const NATIVE_COMMAND9_AI_EFFECT_RESOURCE: i64 = 44;
pub const NATIVE_COMMAND9_AI_SOUND_RESOURCE: i64 = 90;
pub const NATIVE_COMMAND9_AI_FRONT_FRAMES: i64 = 20;
pub const NATIVE_COMMAND9_AI_TARGET_FRAMES: i64 = 60;
pub const NATIVE_COMMAND9_AI_TAIL_FRAMES: i64 = 20;
pub const NATIVE_COMMAND9_AI_DAMAGE_STAGES: i64 = 20;
pub const NATIVE_COMMAND9_AI_ACTOR_SLIDE_IN: i64 = 11;
pub const NATIVE_COMMAND9_AI_ACTOR_SLIDE_OUT: i64 = 8;
pub const NATIVE_COMMAND9_AI_SLIDE_IN_HOLD_MS: i64 = 500;

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand9AISchedule {
    pub effect_resource: i64,
    pub sound_resource: i64,
    pub front_frames: i64,
    pub target_frames: i64,
    pub tail_frames: i64,
    pub damage_stages: i64,
    pub actor_slide_in: i64,
    pub actor_slide_out: i64,
    pub slide_in_hold_ms: i64,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand9AIState {
    pub counter: i64,
    pub toggle: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand9AIFrame {
    pub effect_frames: Vec<i64>,
    pub numeric_marker: bool,
    pub hp_stage: i64,
    pub play_sub1: bool,
    pub play_sub2: bool,
    pub next: NativeCommand9AIState,
}

/// 只接受 `0x2A6BD` 已複製完整十筆的 raw-side-zero 表。raw-side-nonzero 表只複製九筆，ID9 沒有
/// 可證實的值。
pub fn build_native_command9_ai_schedule(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<NativeCommand9AISchedule, FiganiError> {
    if raw_side != 0 {
        return Err(FiganiError::Static("figani: command9 AI nonzero-side effect selector unavailable"));
    }
    if !effect.is_some_and(|e| e.frames.len() >= 31) {
        return Err(FiganiError::Static("figani: command9 AI effect must contain at least 31 frames"));
    }
    Ok(NativeCommand9AISchedule {
        effect_resource: NATIVE_COMMAND9_AI_EFFECT_RESOURCE,
        sound_resource: NATIVE_COMMAND9_AI_SOUND_RESOURCE,
        front_frames: NATIVE_COMMAND9_AI_FRONT_FRAMES,
        target_frames: NATIVE_COMMAND9_AI_TARGET_FRAMES,
        tail_frames: NATIVE_COMMAND9_AI_TAIL_FRAMES,
        damage_stages: NATIVE_COMMAND9_AI_DAMAGE_STAGES,
        actor_slide_in: NATIVE_COMMAND9_AI_ACTOR_SLIDE_IN,
        actor_slide_out: NATIVE_COMMAND9_AI_ACTOR_SLIDE_OUT,
        slide_in_hold_ms: NATIVE_COMMAND9_AI_SLIDE_IN_HOLD_MS,
    })
}

pub fn new_native_command9_ai_state() -> NativeCommand9AIState {
    NativeCommand9AIState { counter: 1, toggle: false }
}

pub fn plan_native_command9_ai_orbit_frame(
    state: NativeCommand9AIState,
    schedule: NativeCommand9AISchedule,
) -> Result<NativeCommand9AIFrame, FiganiError> {
    if schedule.effect_resource != NATIVE_COMMAND9_AI_EFFECT_RESOURCE {
        return Err(FiganiError::Static("figani: command9 AI schedule unavailable"));
    }
    let mut frame = NativeCommand9AIFrame { next: state, ..Default::default() };
    if !state.toggle {
        frame.effect_frames = vec![0];
    }
    frame.next.toggle = !state.toggle;
    Ok(frame)
}

pub fn plan_native_command9_ai_target_frame(
    state: NativeCommand9AIState,
    schedule: NativeCommand9AISchedule,
) -> Result<NativeCommand9AIFrame, FiganiError> {
    if schedule.effect_resource != NATIVE_COMMAND9_AI_EFFECT_RESOURCE || !(1..=60).contains(&state.counter) {
        return Err(FiganiError::Static("figani: command9 AI target state unavailable"));
    }
    let mut frame =
        NativeCommand9AIFrame { effect_frames: vec![0, state.counter >> 1], next: state, ..Default::default() };
    frame.play_sub1 = state.counter == 6;
    frame.play_sub2 = state.counter == 36;
    frame.next.counter += 1;
    frame.numeric_marker = frame.next.counter > 16 && frame.next.counter < 44;
    Ok(frame)
}

/// `schedule.target_frames` 為負時 Go 的 `make` 會 panic，這裡視為零。
pub fn build_native_command9_ai_target_sequence(
    mut state: NativeCommand9AIState,
    schedule: NativeCommand9AISchedule,
) -> Result<(Vec<NativeCommand9AIFrame>, NativeCommand9AIState), StateError<NativeCommand9AIState>> {
    let mut frames = Vec::with_capacity(schedule.target_frames.clamp(0, 1 << 16) as usize);
    let mut stage = 0;
    for _ in 0..schedule.target_frames {
        let mut frame = plan_native_command9_ai_target_frame(state, schedule).map_err(|e| fail(state, e))?;
        if frame.numeric_marker && stage < schedule.damage_stages {
            stage += 1;
            frame.hp_stage = stage;
        }
        state = frame.next;
        frames.push(frame);
    }
    if stage != schedule.damage_stages {
        return Err(fail(state, FiganiError::Static("figani: command9 AI HP marker count incomplete")));
    }
    Ok((frames, state))
}
