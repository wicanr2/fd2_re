//! 指令 6（`0x26E39`）的五通道軌道、目標序列與 caller 顯示狀態。移植自 Go `native_command6.go`。

use super::{Animation, FiganiError, Resolver, RngWalkError, fail, frame_shape_ok, msg};
use crate::go_math;
use crate::native_rng::native_rng_step;

pub const NATIVE_COMMAND6_EFFECT_FRAME_COUNT: i64 = 10;
pub const NATIVE_COMMAND6_CHANNEL_COUNT: usize = 5;
pub const NATIVE_COMMAND6_SOUND_RESOURCE: i64 = 87;
pub const NATIVE_COMMAND6_TARGET_FRAMES: i64 = 12;
pub const NATIVE_COMMAND6_DAMAGE_STAGES: i64 = 5;

const NATIVE_COMMAND6_DWORD_TABLE: [i64; NATIVE_COMMAND6_CHANNEL_COUNT] = [10, 8, 3, 0, 0];
const NATIVE_COMMAND6_BYTE_TABLE: [u8; NATIVE_COMMAND6_CHANNEL_COUNT] = [10, 8, 3, 0, 0];

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand6Point {
    pub x: i64,
    pub y: i64,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand6TargetState {
    pub counters: [i64; NATIVE_COMMAND6_CHANNEL_COUNT],
    pub secondary: [u8; NATIVE_COMMAND6_CHANNEL_COUNT],
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand6Layer {
    pub mode: i64,
    pub channel: i64,
    pub frame: i64,
    pub x: i64,
    pub y: i64,
    pub secondary: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand6TargetFrame {
    pub mode4: Vec<NativeCommand6Layer>,
    pub mode5: Vec<NativeCommand6Layer>,
    pub next: NativeCommand6TargetState,
    pub numeric_marker: bool,
    pub hp_stage: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand6OrbitFrame {
    pub first: Vec<NativeCommand6Layer>,
    pub second: Vec<NativeCommand6Layer>,
    pub next_radius: i64,
    pub draw_target: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand6TransitionFrame {
    pub mode4: Vec<NativeCommand6Layer>,
    pub mode5: Vec<NativeCommand6Layer>,
    pub use_next_target: bool,
    pub target_offset_x: i64,
    pub next: NativeCommand6TargetState,
}

/// 保存 `0x26E39` 已由直接指令閉合的區域表格與資源邊界。浮點座標與 mode4/5 compositor 不在此
/// 型別中猜測。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand6PresentationSchedule {
    pub effect_resource: i64,
    pub sound_resource: i64,
    pub sound_indices: [i64; 3],
    pub base_byte: u8,
    pub dword_table: [i64; NATIVE_COMMAND6_CHANNEL_COUNT],
    pub byte_table: [u8; NATIVE_COMMAND6_CHANNEL_COUNT],
}

/// Go 在 amd64 把超出 int64 範圍或 NaN 的浮點轉整數得到 `0x8000000000000000`（CVTTSD2SI）；
/// Rust 的 `as` 會飽和，所以另外處理。
fn go_f64_to_int(v: f64) -> i64 {
    if v.is_nan() || v >= 9_223_372_036_854_775_808.0 || v < -9_223_372_036_854_775_808.0 { i64::MIN } else { v as i64 }
}

/// 重現 `0x26F99..0x2704C`。角度換算沿用原版單精度常數；`__CHP` 控制字 helper（`0x377A4`）在
/// fistp 之前向零截斷。證據：`fd2_ch24_command6_work_bounds_20261004.json`、issue162。
/// 三角函數用 [`go_math`]（Go `math.Sin`／`math.Cos` 的逐行移植），避免平台 libm 末位差異改變截斷結果。
pub fn native_command6_coordinates(radius: i64, base_byte: u8) -> [NativeCommand6Point; NATIVE_COMMAND6_CHANNEL_COUNT] {
    // 照抄 Go 的 `float64(float32(0.017453292519943295))`。
    #[allow(clippy::excessive_precision)]
    let degree = 0.017453292519943295_f32 as f64;
    let mut points = [NativeCommand6Point::default(); NATIVE_COMMAND6_CHANNEL_COUNT];
    for (channel, point) in points.iter_mut().enumerate() {
        let angle = (channel as i64 * 72) as f64 * degree;
        *point = NativeCommand6Point {
            x: go_f64_to_int(go_math::cos(angle) * radius as f64 + base_byte as f64),
            y: go_f64_to_int(go_math::sin(angle) * radius as f64 * 1.2 + 30.0),
        };
    }
    points
}

/// 保存 `0x26F13..0x26F3A` 的第一次 mode3 寫入。之後的目標經 `0x54096` gate 保留這些座標。
pub fn native_command6_target_coordinates(
    mut points: [NativeCommand6Point; NATIVE_COMMAND6_CHANNEL_COUNT],
    next_radius: i64,
) -> [NativeCommand6Point; NATIVE_COMMAND6_CHANNEL_COUNT] {
    let (x2, x4) = (points[2].x, points[4].x);
    points[2].x = x4;
    points[4].x = x2;
    points[2].y = points[4].y;
    points[4].y = next_radius;
    points
}

fn native_command6_orbit_layers(
    mode: i64,
    radius: i64,
    raw_side: u8,
    schedule: NativeCommand6PresentationSchedule,
) -> Result<Vec<NativeCommand6Layer>, FiganiError> {
    if schedule.effect_resource != 32 && schedule.effect_resource != 33 {
        return Err(FiganiError::Static("figani: command6 orbit schedule unavailable"));
    }
    let points = native_command6_coordinates(radius, schedule.base_byte);
    let mut layers = Vec::with_capacity(NATIVE_COMMAND6_CHANNEL_COUNT);
    for (channel, point) in points.iter().enumerate() {
        let draw = if raw_side == 0 {
            mode == 2 || mode == 8
        } else if mode == 1 || mode == 7 {
            channel < 2
        } else if mode == 2 || mode == 8 {
            channel > 1
        } else {
            false
        };
        if draw {
            layers.push(NativeCommand6Layer {
                mode,
                channel: channel as i64,
                frame: 4,
                x: point.x,
                y: point.y,
                secondary: false,
            });
        }
    }
    Ok(layers)
}

pub fn plan_native_command6_prelude_frame(
    radius: i64,
    raw_side: u8,
    schedule: NativeCommand6PresentationSchedule,
) -> Result<NativeCommand6OrbitFrame, FiganiError> {
    let first = native_command6_orbit_layers(1, radius, raw_side, schedule)?;
    let second = native_command6_orbit_layers(2, radius, raw_side, schedule)?;
    Ok(NativeCommand6OrbitFrame { first, second, next_radius: radius.wrapping_add(6), draw_target: false })
}

pub fn plan_native_command6_tail_frame(
    radius: i64,
    raw_side: u8,
    schedule: NativeCommand6PresentationSchedule,
) -> Result<NativeCommand6OrbitFrame, FiganiError> {
    let first = native_command6_orbit_layers(7, radius, raw_side, schedule)?;
    let next = radius.wrapping_sub(6);
    let second = native_command6_orbit_layers(8, radius, raw_side, schedule)?;
    Ok(NativeCommand6OrbitFrame { first, second, next_radius: next, draw_target: true })
}

pub fn new_native_command6_target_state() -> NativeCommand6TargetState {
    let mut state = NativeCommand6TargetState::default();
    for (channel, counter) in state.counters.iter_mut().enumerate() {
        *counter = -(channel as i64);
    }
    state
}

/// 重現一次 mode4 → 目標 → mode5 迭代。回傳的疊層不帶音效或數值發布語意。
pub fn plan_native_command6_target_frame(
    state: NativeCommand6TargetState,
    schedule: NativeCommand6PresentationSchedule,
    points: [NativeCommand6Point; NATIVE_COMMAND6_CHANNEL_COUNT],
    raw_side: u8,
) -> Result<NativeCommand6TargetFrame, FiganiError> {
    if schedule.effect_resource != 32 && schedule.effect_resource != 33 {
        return Err(FiganiError::Static("figani: command6 target schedule unavailable"));
    }
    let want_resource = if raw_side == 0 { 33 } else { 32 };
    if schedule.effect_resource != want_resource {
        return Err(FiganiError::Static("figani: command6 target side mismatch"));
    }
    let mut next = state;
    let main_layers = |mode: i64,
                       next: &mut NativeCommand6TargetState|
     -> Result<Vec<NativeCommand6Layer>, FiganiError> {
        let mut layers = Vec::with_capacity(NATIVE_COMMAND6_CHANNEL_COUNT);
        for (channel, point) in points.iter().enumerate() {
            let counter = next.counters[channel];
            let frame = if counter < 0 { 4 } else { counter };
            if frame >= NATIVE_COMMAND6_EFFECT_FRAME_COUNT {
                return Err(msg(format!("figani: command6 main frame {frame} unavailable")));
            }
            if frame == 1 {
                next.secondary[channel] = 5;
            }
            let draw = if raw_side == 0 { mode == 5 } else { (mode == 4 && channel < 2) || (mode == 5 && channel > 1) };
            if draw {
                // Go 以 frame 索引五筆的表格；frame 落在 5..9 時 Go 會 index out of range panic。
                layers.push(NativeCommand6Layer {
                    mode,
                    channel: channel as i64,
                    frame,
                    x: point.x.wrapping_add(schedule.dword_table[frame as usize]),
                    y: point.y.wrapping_add(schedule.byte_table[frame as usize] as i64),
                    secondary: false,
                });
            }
        }
        Ok(layers)
    };
    let mode4 = main_layers(4, &mut next)?;
    let mut mode5 = main_layers(5, &mut next)?;
    let mut numeric_marker = false;
    for (channel, point) in points.iter().enumerate() {
        next.counters[channel] = next.counters[channel].wrapping_add(1);
        if next.counters[channel] == 5 {
            next.counters[channel] = 0;
        }
        if next.counters[channel] == 2 {
            numeric_marker = true;
        }
        let secondary = next.secondary[channel];
        if secondary != 0 {
            if secondary as i64 >= NATIVE_COMMAND6_EFFECT_FRAME_COUNT {
                return Err(msg(format!("figani: command6 secondary frame {secondary} unavailable")));
            }
            mode5.push(NativeCommand6Layer {
                mode: 5,
                channel: channel as i64,
                frame: secondary as i64,
                secondary: true,
                x: point.x.wrapping_sub(60),
                y: point.y.wrapping_sub(20),
            });
            next.secondary[channel] = next.secondary[channel].wrapping_add(1);
            if next.secondary[channel] == 10 {
                next.secondary[channel] = 0;
            }
        }
    }
    Ok(NativeCommand6TargetFrame { mode4, mode5, next, numeric_marker, hp_stage: 0 })
}

/// 把十二張 mode3 幀綁到第一個目標的外層 `0x2A6BD` 消費端。因 `0x525AF[6]` 為 5，只有前五個
/// 原始 marker 發布 HP。
pub fn build_native_command6_target_sequence(
    schedule: NativeCommand6PresentationSchedule,
    points: [NativeCommand6Point; NATIVE_COMMAND6_CHANNEL_COUNT],
    raw_side: u8,
) -> Result<Vec<NativeCommand6TargetFrame>, FiganiError> {
    build_native_command6_target_sequence_from_state(new_native_command6_target_state(), schedule, points, raw_side)
}

/// 保留經 `sub_2BA22` 傳到下一個目標 gated mode3 呼叫的 handler 狀態。
pub fn build_native_command6_target_sequence_from_state(
    mut state: NativeCommand6TargetState,
    schedule: NativeCommand6PresentationSchedule,
    points: [NativeCommand6Point; NATIVE_COMMAND6_CHANNEL_COUNT],
    raw_side: u8,
) -> Result<Vec<NativeCommand6TargetFrame>, FiganiError> {
    let mut frames = Vec::with_capacity(NATIVE_COMMAND6_TARGET_FRAMES as usize);
    let mut hp_stage = 0;
    for _ in 0..NATIVE_COMMAND6_TARGET_FRAMES {
        let mut frame = plan_native_command6_target_frame(state, schedule, points, raw_side)?;
        if frame.numeric_marker && hp_stage < NATIVE_COMMAND6_DAMAGE_STAGES {
            hp_stage += 1;
            frame.hp_stage = hp_stage;
        }
        state = frame.next;
        frames.push(frame);
    }
    if hp_stage != NATIVE_COMMAND6_DAMAGE_STAGES {
        return Err(msg(format!("figani: command6 HP marker count incomplete: {hp_stage}")));
    }
    Ok(frames)
}

/// 重現 `sub_2BA22` 的指令 6 分支。保留目前目標留下的效果 counter 狀態，產生四張離場與五張進場幀。
pub fn build_native_command6_transition_sequence(
    mut state: NativeCommand6TargetState,
    schedule: NativeCommand6PresentationSchedule,
    points: [NativeCommand6Point; NATIVE_COMMAND6_CHANNEL_COUNT],
    raw_side: u8,
) -> Result<Vec<NativeCommand6TransitionFrame>, FiganiError> {
    let direction: i64 = if raw_side == 0 { 1 } else { -1 };
    let mut frames = Vec::with_capacity(9);
    let steps = (1..=4).map(|s| (false, s)).chain((0..=4).rev().map(|s| (true, s)));
    for (use_next, step) in steps {
        let planned = plan_native_command6_target_frame(state, schedule, points, raw_side)?;
        state = planned.next;
        frames.push(NativeCommand6TransitionFrame {
            mode4: planned.mode4,
            mode5: planned.mode5,
            use_next_target: use_next,
            target_offset_x: direction * 35 * step,
            next: planned.next,
        });
    }
    Ok(frames)
}

pub fn build_native_command6_presentation_schedule(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<NativeCommand6PresentationSchedule, FiganiError> {
    let (mut resource, mut base) = (32, 30u8);
    let (mut dwords, mut bytes) = (NATIVE_COMMAND6_DWORD_TABLE, NATIVE_COMMAND6_BYTE_TABLE);
    if raw_side == 0 {
        (resource, base) = (33, 90);
        for index in 0..3 {
            dwords[index] = -dwords[index];
            bytes[index] = 0;
        }
    }
    let Some(effect) = effect.filter(|e| {
        e.header_byte1 == 0
            && e.header_byte2 as i64 == NATIVE_COMMAND6_EFFECT_FRAME_COUNT
            && e.header_byte4 == 0
            && e.frames.len() as i64 == NATIVE_COMMAND6_EFFECT_FRAME_COUNT
    }) else {
        return Err(msg(format!("figani: command6 FDOTHER #{resource} signature mismatch")));
    };
    for (index, frame) in effect.frames.iter().enumerate() {
        if !frame_shape_ok(frame) {
            return Err(msg(format!("figani: command6 FDOTHER #{resource} frame {index} malformed")));
        }
    }
    Ok(NativeCommand6PresentationSchedule {
        effect_resource: resource,
        sound_resource: NATIVE_COMMAND6_SOUND_RESOURCE,
        sound_indices: [1, 2, 3],
        base_byte: base,
        dword_table: dwords,
        byte_table: bytes,
    })
}

/// 依 `sub_2A6BD`：先 resolve 每個目標，再在每個命中 marker 消耗數值抖動亂數（包含第五個 HP 分段
/// 之後的 marker）。未命中與過場保留動畫狀態，不擲數值亂數。
pub fn walk_native_command6_rng(
    mut rng: u16,
    schedule: NativeCommand6PresentationSchedule,
    raw_side: u8,
    target_count: i64,
    resolve: Resolver<'_>,
) -> Result<u16, RngWalkError> {
    let Some(resolve) = resolve.filter(|_| target_count > 0) else {
        return Err(fail(rng, FiganiError::Static("figani: command6 RNG walk needs targets and a resolver")));
    };
    let points = native_command6_target_coordinates(native_command6_coordinates(36, schedule.base_byte), 42);
    let mut state = new_native_command6_target_state();
    for index in 0..target_count {
        let frames = build_native_command6_target_sequence_from_state(state, schedule, points, raw_side)
            .map_err(|e| fail(rng, e))?;
        let (next, hit) = resolve(index, rng).map_err(|e| fail(rng, FiganiError::Resolver(e)))?;
        rng = next;
        if hit {
            for frame in &frames {
                if frame.numeric_marker {
                    rng = native_rng_step(rng);
                }
            }
        }
        state = frames[frames.len() - 1].next;
        if index + 1 < target_count {
            let transition = build_native_command6_transition_sequence(state, schedule, points, raw_side)
                .map_err(|e| fail(rng, e))?;
            state = transition[transition.len() - 1].next;
        }
    }
    Ok(rng)
}

/// `sub_2A6BD` 的 caller 局部命中狀態。不隨目標或 `sub_2BA22` 重設；未命中保留全部狀態。
/// 主契約 followup161。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand6DisplayState {
    pub shade: i64,
    pub pose: i64,
    pub jitter: i64,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand6TargetDisplayFrame {
    pub shader: i64,
    pub offset_x: i64,
    pub offset_y: i64,
    pub rng_after: u16,
    pub next: NativeCommand6DisplayState,
}

pub fn new_native_command6_display_state() -> NativeCommand6DisplayState {
    NativeCommand6DisplayState { shade: 8, pose: 3, jitter: -1 }
}

/// 消費已規劃的 marker 與傷害後亂數，只產生繪圖參數，不重新 resolve 傷害或發布遊戲狀態。
pub fn build_native_command6_target_display_frames(
    mut state: NativeCommand6DisplayState,
    planned: &[NativeCommand6TargetFrame],
    hit: bool,
    mut numeric_rng: u16,
) -> Result<Vec<NativeCommand6TargetDisplayFrame>, FiganiError> {
    if planned.len() as i64 != NATIVE_COMMAND6_TARGET_FRAMES
        || !(2..=8).contains(&state.shade)
        || !(0..=3).contains(&state.pose)
        || !(-1..=1).contains(&state.jitter)
    {
        return Err(FiganiError::Static("figani: command6 caller display state unavailable"));
    }
    let (x_pose, y_pose) = ([6i64, 4, 2, 0], [-3i64, -2, -1, 0]);
    let mut out = Vec::with_capacity(planned.len());
    for frame in planned {
        let mut draw = NativeCommand6TargetDisplayFrame { shader: -1, ..Default::default() };
        if hit {
            draw.shader = state.shade << 8 | 0xb0;
            draw.offset_x = x_pose[state.pose as usize] * state.jitter;
            draw.offset_y = y_pose[state.pose as usize];
            state.shade -= 1;
            if state.shade == 1 {
                state.shade = 8;
            }
            if state.pose != 3 {
                state.pose += 1;
            }
            if frame.numeric_marker {
                state.pose = 0;
                numeric_rng = native_rng_step(numeric_rng);
                state.jitter = 1 - (numeric_rng % 3) as i64;
            }
        }
        draw.next = state;
        draw.rng_after = numeric_rng;
        out.push(draw);
    }
    Ok(out)
}
