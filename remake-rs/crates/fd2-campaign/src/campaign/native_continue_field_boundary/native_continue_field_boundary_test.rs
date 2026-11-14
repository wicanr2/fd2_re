//! 移植自 `remake/internal/campaign/native_continue_field_boundary_test.go`。
//!
//! Go 的 `reflect.DeepEqual(state, before)` 在 Rust 以 `Debug` 文字比較（`battle::State` 含 callback，
//! 不 derive `PartialEq`；callback 的 `Debug` 是固定字串）。

use super::*;

/// Go `continueFieldBoundaryInput`。
pub(crate) fn continue_field_boundary_input() -> save::ContinueRuntimeInput {
    let mut snapshot = save::CurrentSnapshot {
        header: save::CurrentRuntimeHeader {
            turn_counter: 7,
            runtime_count: 1,
            chapter: 1,
            camera_x: 1,
            camera_y: 2,
            cursor_x: 8,
            cursor_y: 7,
            visible_cursor_x: 7,
            visible_cursor_y: 5,
            persistent_count: 1,
            hud_gate_a: 0xa1,
            ..Default::default()
        },
        ..Default::default()
    };
    snapshot.runtime_records = vec![save::PersistentRecord::default()];
    snapshot.runtime_records[0].raw[0] = 8;
    snapshot.runtime_records[0].raw[1] = 7;
    snapshot.runtime_records[0].raw[7] = 5;
    snapshot.native_field_control[2] = 1;
    snapshot.native_field_control[3..6].copy_from_slice(&[9, 10, 11]);
    snapshot.native_field_control[0x33..0x35].copy_from_slice(&[12, 13]);
    snapshot.native_field_control[0x53] = 14;
    snapshot.native_field_control[0x54..0x56].copy_from_slice(&0x1234u16.to_le_bytes());
    snapshot.native_field_control[save::CURRENT_FIELD_CONTROL_UNIT_OFFSET] = 15;
    snapshot.native_event_state[12] = 16;

    save::build_continue_runtime_input(
        &snapshot,
        save::ContinueRuntimeContext {
            chapter: 1,
            field_width: 30,
            field_height: 20,
            selector_group_count: 16,
            title_timer_tick: -123,
            has_title_timer_tick: true,
        },
    )
    .unwrap()
}

/// Go `continueFieldBoundaryState`。
pub(crate) fn continue_field_boundary_state() -> battle::State {
    let mut state = battle::State::default();
    (state.w, state.h) = (30, 20);
    state.native_field_event_slots = vec![0; 30 * 20];
    state.native_field_events = vec![battle::NativeFieldEvent::default(); 16];
    let mut sentinel = Unit::default();
    sentinel.name = "sentinel".into();
    state.units = vec![sentinel];
    state
}

#[test]
fn test_materialize_native_continue_field_boundary_installs_live_state_atomically() {
    let mut input = continue_field_boundary_input();
    let mut state = continue_field_boundary_state();
    materialize_native_continue_field_boundary(Some(&mut state), &input, 1).unwrap();
    assert!(
        state.has_native_field_control_state
            && state.native_round_counter == 7
            && state.native_event_state[12] == 16
            && state.native_turn_event_controls[0]
                == battle::NativeTurnEventControl { turn: 9, event_id: 10, raw_camp: 11 }
            && state.native_field_events[0] == battle::NativeFieldEvent { event_id: 12, selector: 13 }
            && state.native_chest_controls[0] == battle::NativeChestControl { raw_type: 14, value: 0x1234 }
            && state.native_field_unit_controls.len() == 1
            && state.native_field_unit_controls[0].raw[0] == 15
            && state.native_runtime_records.len() == 1
            && state.native_runtime_records[0].raw[7] == 5
            && state.native_runtime_records[0].selector_key == 5
            && state.native_runtime_records[0].selector_slot == 0,
        "materialized field boundary={state:?}"
    );
    assert!(
        state.has_native_map_view_state
            && state.has_native_map_hud_state
            && state.has_native_map_range_mode_state
            && state.native_map_range_mode == 0
            && state.units.len() == 1
            && state.units[0].name == "sentinel",
        "materialized view/roster boundary={state:?}"
    );
    assert!(
        !state.has_native_map_cycle_state
            && !state.has_native_terrain_phase_state
            && !state.has_native_map_binary_timing_state,
        "field boundary guessed unresolved timing state"
    );

    input.native_field_control[0] ^= 0xff;
    input.field_control.units[0].raw[0] ^= 0xff;
    input.runtime_records[0].raw.raw[7] ^= 0xff;
    assert!(
        state.native_field_control_raw[0] != input.native_field_control[0]
            && state.native_field_unit_controls[0].raw[0] != input.field_control.units[0].raw[0]
            && state.native_runtime_records[0].raw[7] != input.runtime_records[0].raw.raw[7],
        "field boundary aliases CONTINUE input"
    );
}

#[test]
fn test_materialize_native_continue_field_boundary_rejects_before_mutation() {
    let input = continue_field_boundary_input();
    // Go 的 "unvalidated input"（`*input = fdsave.ContinueRuntimeInput{}`）在 Rust 無法建構：
    // `ContinueRuntimeInput` 的 `validated` 欄位是 fd2-formats 私有，也沒有 `Default`。
    // 同一道閘門由 "tampered validated input" 覆蓋。
    type Mutate = fn(&mut battle::State, &mut save::ContinueRuntimeInput) -> i64;
    let tests: [(&str, Mutate); 4] = [
        ("tampered validated input", |_, input| {
            input.native_field_control[3] += 1;
            1
        }),
        ("chapter mismatch", |_, _| 2),
        ("dimension mismatch", |state, _| {
            state.w += 1;
            1
        }),
        ("missing field events", |state, _| {
            state.native_field_events = Vec::new();
            1
        }),
    ];
    for (name, mutate) in tests {
        let mut state = continue_field_boundary_state();
        let mut candidate = input.clone();
        let asset_chapter = mutate(&mut state, &mut candidate);
        let before = state.clone();
        assert!(
            materialize_native_continue_field_boundary(Some(&mut state), &candidate, asset_chapter).is_err(),
            "{name}: malformed field boundary was accepted"
        );
        assert!(
            state.units == before.units
                && state.native_field_event_slots == before.native_field_event_slots
                && state.native_field_events == before.native_field_events
                && !state.has_native_field_control_state,
            "{name}: rejected field boundary mutated state={state:?}"
        );
    }
}
