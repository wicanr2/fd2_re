//! 移植自 `remake/internal/battle/native_ai_mode5_test.go`。
//!
//! 其中兩個測試經由 E 組 `State::next_ai_plan`（`combat.go:NextAIPlan`）。

use super::*;
use crate::battle::native_ai_runtime::native_ai_runtime_test::{native_ai_runtime_cost_rows, native_ai_runtime_unit};

/// Go `nativeAIMode5Grid`。
fn native_ai_mode5_grid(w: i64, h: i64, event_cell: Cell, event_id: u8) -> Vec<u8> {
    let mut grid = vec![0u8; (4 + 4 * w * h) as usize];
    (grid[0], grid[2]) = (w as u8, h as u8);
    let offset = (4 + 4 * (event_cell.x + w * event_cell.y)) as usize;
    grid[offset..offset + 2].copy_from_slice(&0u16.to_le_bytes());
    grid[offset + 2] = event_id;
    reset_native_map_event_grid(&mut grid);
    grid
}

fn mode5_actor() -> Unit {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 5);
    actor.native_record_byte3_d = 1;
    actor.has_native_record_byte3_d = true;
    actor
}

/// Go：`TestNativeAIMode5EventCellAndStateTailPreserveRawRows`
#[test]
fn native_ai_mode5_event_cell_and_state_tail_preserve_raw_rows() {
    let mut actor = mode5_actor();
    actor.native_record_byte34 = 0xa5;
    actor.native_record_death_effect = [0xff, 0xff, 0xff];
    actor.has_native_record_death_effect = true;
    let mut state = State {
        w: 3,
        h: 1,
        units: vec![actor],
        native_event_state: [0; 0x20],
        native_composition_event_bytes: vec![0, 0, 0],
        native_terrain_move_codes: vec![0, 0, 0],
        native_terrain_control: vec![0x20, 0, 0, 0],
        native_map_event_grid: native_ai_mode5_grid(3, 1, Cell { x: 2, y: 0 }, 1),
        has_native_map_event_grid: true,
        native_field_control_raw: vec![0; 0x56 + 3],
        has_native_field_control_state: true,
        ..Default::default()
    };
    let row_offset = 0x53 + 3;
    state.native_field_control_raw[row_offset] = 0;
    state.native_field_control_raw[row_offset + 1..row_offset + 3].copy_from_slice(&7u16.to_le_bytes());
    let got = state.native_ai_mode5_event_cell(1);
    assert!(matches!(got, Ok(c) if c == Cell { x: 2, y: 0 }), "event cell={got:?}, want (2,0)");
    state.units[0].x = 2;
    if let Err(err) = state.apply_native_ai_mode5_event(Some(0), 1, Cell { x: 2, y: 0 }) {
        panic!("ApplyNativeAIMode5Event() error = {err}");
    }
    let actor = &state.units[0];
    assert!(
        state.native_event_state[1] == 1
            && actor.native_record_byte34 == 7
            && actor.native_record_death_effect == [0, 7, 0]
            && actor.inventory == [7]
            && actor.inventory_slots[0] == 7,
        "mode5 state mutation lost raw fields: state={} mode={} effect={:?} inventory={:?} slots={:?}",
        state.native_event_state[1],
        actor.native_record_byte34 & 0x0f,
        actor.native_record_death_effect,
        actor.inventory,
        actor.inventory_slots
    );
    let got = state.native_map_event_grid[4 + 4 * 2 + 2];
    assert!(got == 0, "mode5 event grid low byte={got}, want cleared");
}

/// Go：`TestNativeAIMode5EventEmitsProvenRawAudioCueBeforeStateCompletion`
///
/// Go 的 callback 讀外層 `state` 確認 cue 在事件狀態寫成 1 之後才發出；Rust 的 `state` 在呼叫期間
/// 被可變借用，callback 讀不到它。改以 `mark_treasure_event`（寫事件狀態）在 `emit` 之前的
/// 呼叫順序為準：callback 收到 cue 時記下次數，呼叫後檢查事件狀態與模式位元組。
#[test]
fn native_ai_mode5_event_emits_proven_raw_audio_cue_before_state_completion() {
    let mut actor = mode5_actor();
    actor.native_record_death_effect = [0xff, 0xff, 0xff];
    actor.has_native_record_death_effect = true;
    let mut raw = vec![0u8; 0x56 + 3];
    raw[0x56] = 1;
    raw[0x57..0x59].copy_from_slice(&9u16.to_le_bytes());
    let mut state = State {
        w: 1,
        h: 1,
        units: vec![actor],
        native_event_state: [0; 0x20],
        native_terrain_control: vec![0x20, 0, 0, 0],
        native_map_event_grid: native_ai_mode5_grid(1, 1, Cell { x: 0, y: 0 }, 1),
        has_native_map_event_grid: true,
        native_field_control_raw: raw,
        has_native_field_control_state: true,
        ..Default::default()
    };
    let mut cues = Vec::new();
    state.units[0].x = 0;
    let mut emit = |cue: NativeAIMode5AudioCue| cues.push(cue);
    if let Err(err) = state.apply_native_ai_mode5_event_with_audio_cue(Some(0), 1, Cell { x: 0, y: 0 }, Some(&mut emit))
    {
        panic!("ApplyNativeAIMode5EventWithAudioCue() error = {err}");
    }
    let want = NativeAIMode5AudioCue {
        handle_linear_address: NATIVE_AI_MODE5_AUDIO_HANDLE,
        resource_id: NATIVE_AI_MODE5_AUDIO_RESOURCE,
        index: NATIVE_AI_MODE5_AUDIO_INDEX,
        loop_count: NATIVE_AI_MODE5_AUDIO_LOOP_COUNT,
    };
    assert!(cues.len() == 1 && cues[0] == want, "raw mode5 audio cues={cues:?}");
    assert!(
        state.units[0].native_record_byte34 == 7 && state.native_event_state[1] == 1,
        "mode5 state completion lost after audio cue: mode={} state={}",
        state.units[0].native_record_byte34,
        state.native_event_state[1]
    );
}

/// Go：`TestNextAIPlanMode5UsesRawEventCellAndFailsClosedWithoutRow`
#[test]
fn next_ai_plan_mode5_uses_raw_event_cell_and_fails_closed_without_row() {
    let mut state = State {
        w: 2,
        h: 1,
        units: vec![mode5_actor()],
        native_composition_event_bytes: vec![0, 0],
        native_terrain_move_codes: vec![0, 0],
        native_terrain_control: vec![0x20, 0, 0, 0],
        native_map_event_grid: native_ai_mode5_grid(2, 1, Cell { x: 1, y: 0 }, 1),
        has_native_map_event_grid: true,
        ..Default::default()
    };
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.native_error.is_some()),
        "mode5 missing field-control row plan={plan:?}, want fail-closed error"
    );
    state.native_field_control_raw = vec![0; 0x56 + 3];
    state.native_field_control_raw[0x56] = 0;
    state.has_native_field_control_state = true;
    let plan = state.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.native_error.is_none()
            && p.native_mode_event_active
            && p.native_mode_event_id == 1
            && p.native_mode_event_destination == Cell { x: 1, y: 0 }),
        "mode5 raw event plan={plan:?}"
    );
}

/// Go：`TestNativeAIMode5RejectsMalformedLaterCellWithoutPartialMutation`
#[test]
fn native_ai_mode5_rejects_malformed_later_cell_without_partial_mutation() {
    let mut actor = mode5_actor();
    actor.native_record_death_effect = [0xff, 0xff, 0xff];
    actor.has_native_record_death_effect = true;
    let mut grid = native_ai_mode5_grid(2, 1, Cell { x: 0, y: 0 }, 1);
    grid[8..10].copy_from_slice(&0x03ffu16.to_le_bytes()); // later tile has no terrain row
    let mut raw = vec![0u8; 0x56 + 3];
    raw[0x56] = 0;
    raw[0x57..0x59].copy_from_slice(&7u16.to_le_bytes());
    let mut state = State {
        w: 2,
        h: 1,
        units: vec![actor],
        native_event_state: [0; 0x20],
        native_terrain_control: vec![0x20, 0, 0, 0],
        native_map_event_grid: grid,
        has_native_map_event_grid: true,
        native_field_control_raw: raw,
        has_native_field_control_state: true,
        ..Default::default()
    };
    let before_grid = state.native_map_event_grid.clone();
    let before_byte34 = state.units[0].native_record_byte34;
    assert!(
        state.apply_native_ai_mode5_event(Some(0), 1, Cell { x: 0, y: 0 }).is_err(),
        "malformed later event cell unexpectedly committed"
    );
    let actor = &state.units[0];
    assert!(
        state.native_event_state[1] == 0
            && actor.native_record_death_effect == [0xff, 0xff, 0xff]
            && actor.inventory.is_empty()
            && actor.native_record_byte34 == before_byte34,
        "mode5 partial state mutation: state={} effect={:?} inventory={:?} byte34={}",
        state.native_event_state[1],
        actor.native_record_death_effect,
        actor.inventory,
        actor.native_record_byte34
    );
    for (i, (&got, &want)) in state.native_map_event_grid.iter().zip(&before_grid).enumerate() {
        assert!(got == want, "mode5 partial grid mutation at {i}: got={got} want={want}");
    }
}

fn chest_state(raw_type: u8, value: u16) -> State {
    let mut actor = mode5_actor();
    actor.native_record_death_effect = [0xff, 0, 0];
    actor.has_native_record_death_effect = true;
    let mut state = State {
        w: 2,
        h: 1,
        units: vec![actor],
        native_composition_event_bytes: vec![0, 0],
        native_terrain_move_codes: vec![0, 0],
        native_terrain_control: vec![0x20, 0, 0, 0],
        native_map_event_grid: native_ai_mode5_grid(2, 1, Cell { x: 1, y: 0 }, 1),
        has_native_map_event_grid: true,
        has_native_chest_control_state: true,
        ..Default::default()
    };
    state.native_chest_controls[1] = NativeChestControl { raw_type, value };
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    state
}

/// Go：`TestNativeMapDrawTilesFollowsTheMutableBuffer`
///
/// 0x12263 就地把事件格的 tile word +1；繪圖端讀的是同一份緩衝，所以撿走之後
/// 那一格要換成打開的箱子（第十一章 (7,20) 事件 5：原版 seq 5525 起箱子是開的）。
#[test]
fn native_map_draw_tiles_follows_the_mutable_buffer() {
    let mut state = chest_state(1, 3000);
    let (before, ok) = state.native_map_draw_tiles();
    assert!(ok, "可變緩衝沒有材料化");
    state.units[0].set_map_placement(1, 0, 0);
    state.apply_native_ai_mode5_event(Some(0), 1, Cell { x: 1, y: 0 }).unwrap();
    let (after, ok) = state.native_map_draw_tiles();
    assert!(ok, "事件之後讀不到可變緩衝");
    assert!(after[1] == before[1] + 1 && after[0] == before[0], "繪圖圖塊 {before:?} → {after:?}，事件格應該只 +1");
    let empty = State { w: 2, h: 1, ..Default::default() };
    assert!(!empty.native_map_draw_tiles().1, "沒有可變緩衝時不該回報可用");
}

/// Go：`TestNativeAIMode5PickupBecomesExecutableDeathReward`
///
/// 0x13CCE／0x13CD6 的寶箱列寫進 +0x31..+0x33；重製端的掉落走 DeathEffect／
/// DeathReward，兩份要一致，撿到的金錢才會在被擊倒時進玩家金庫。
#[test]
fn native_ai_mode5_pickup_becomes_executable_death_reward() {
    let mut state = chest_state(1, 10000);
    state.units[0].set_map_placement(1, 0, 0);
    state.apply_native_ai_mode5_event(Some(0), 1, Cell { x: 1, y: 0 }).unwrap();
    let actor = &state.units[0];
    assert!(
        actor.native_record_death_effect == [1, 0x10, 0x27],
        "raw +0x31..+0x33={:?}",
        actor.native_record_death_effect
    );
    assert!(
        actor.death_reward.is_some_and(|r| r.r#type == 1 && r.value == 10000)
            && actor.death_effect == actor.death_reward,
        "typed 掉落沒有跟著寫：effect={:?} reward={:?}",
        actor.death_effect,
        actor.death_reward
    );
}

/// Go：`TestNativeAIMode5FallsBackToModeZeroTailWhenEventUnavailable`
///
/// 0x13C42：[0x53AD5][+0x3D] 非零跳 0x13B05；0x13C59：0x15DF3 回 −1 也跳 0x13B05。
/// 兩種都不是失敗即關閉，是走 mode 0 的 0x14121／0x13E9C 尾段。
#[test]
fn native_ai_mode5_falls_back_to_mode_zero_tail_when_event_unavailable() {
    let build = || {
        let actor = mode5_actor();
        let mut target = native_ai_runtime_unit(1, 0, 0, 0);
        target.camp = OWN;
        let mut state = State {
            w: 2,
            h: 1,
            units: vec![actor, target],
            native_composition_event_bytes: vec![0, 0],
            native_terrain_move_codes: vec![0, 0],
            native_terrain_control: vec![0x20, 0, 0, 0],
            native_map_event_grid: native_ai_mode5_grid(2, 1, Cell { x: 1, y: 0 }, 1),
            has_native_map_event_grid: true,
            has_native_chest_control_state: true,
            ..Default::default()
        };
        state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
        state
    };
    let mut state = build();
    state.native_event_state[1] = 1;
    let plan = state.next_ai_plan();
    let plan = match plan {
        Some(p) if p.native_error.is_none() => p,
        other => panic!("事件已觸發時的計畫={other:?}"),
    };
    assert!(
        !plan.native_mode_event_active && plan.native_mode_fallback_active && plan.native_mode_fallback == 5,
        "事件已觸發仍走事件分支：{plan:?}"
    );
    let mut state = build();
    state.native_map_event_grid = native_ai_mode5_grid(2, 1, Cell { x: 1, y: 0 }, 9); // 沒有 +0x3D=1 的格子
    let plan = state.next_ai_plan();
    let plan = match plan {
        Some(p) if p.native_error.is_none() => p,
        other => panic!("找不到事件格時的計畫={other:?}"),
    };
    assert!(!plan.native_mode_event_active, "找不到事件格仍走事件分支：{plan:?}");
}
