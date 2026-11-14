//! 移植自 `remake/internal/campaign/native_continue_map_timing_test.go`（含 handoff 的兩個測試）。
//!
//! Go 的 `reflect.DeepEqual(state, before)` 以 `Debug` 文字比較（見 field boundary 測試的說明）。

use super::*;
use crate::campaign::native_continue_field_boundary::native_continue_field_boundary_test::{
    continue_field_boundary_input, continue_field_boundary_state,
};
use crate::campaign::native_continue_pending_groups::native_continue_pending_groups_test::{
    native_pending_groups_fixture, native_pending_groups_input,
};
use crate::campaign::native_continue_runtime_units::native_continue_runtime_units_test::native_runtime_catalog;

fn dbg(state: &battle::State) -> String {
    format!("{state:?}")
}

#[test]
fn test_materialize_native_continue_map_timing_installs_exact_seed() {
    let Some(mut catalog) = native_runtime_catalog() else { return };
    let input = continue_field_boundary_input();
    let mut state = continue_field_boundary_state();
    materialize_native_continue_field_boundary(Some(&mut state), &input, 1).unwrap();
    materialize_native_continue_runtime_units(Some(&mut state), &input, &mut catalog).unwrap();
    materialize_native_continue_map_timing(Some(&mut state), &input).unwrap();
    assert_eq!(
        state.native_map_cycle_state,
        fdicon::NativeMapSpriteCycleState { last_timer_tick: -123, ..Default::default() },
        "sprite seed"
    );
    assert_eq!(state.native_terrain_phase_state, fdother::NativeTerrainPhaseState::default(), "terrain seed");
    assert_eq!(state.native_terrain_flip_state, fdicon::NativeBinaryTickState::default(), "terrain flip seed");
    assert_eq!(state.native_unit_pixel_shift_state, fdicon::NativeBinaryTickState::default(), "unit shift seed");
    assert!(
        state.has_native_map_cycle_state
            && state.has_native_terrain_phase_state
            && state.has_native_map_binary_timing_state,
        "timing provenance flags were not installed"
    );
}

#[test]
fn test_materialize_native_continue_map_timing_rejects_atomically() {
    let Some(mut catalog) = native_runtime_catalog() else { return };
    let input = continue_field_boundary_input();
    let mut base = continue_field_boundary_state();
    materialize_native_continue_field_boundary(Some(&mut base), &input, 1).unwrap();
    materialize_native_continue_runtime_units(Some(&mut base), &input, &mut catalog).unwrap();

    type Mutate = fn(&mut battle::State);
    let tests: [(&str, Mutate); 3] = [
        ("missing field boundary", |state| state.has_native_field_control_state = false),
        ("missing runtime units", |state| state.has_native_runtime_unit_projection = false),
        ("wrong opening selector", |state| state.native_map_range_mode = 1),
    ];
    for (name, mutate) in tests {
        let mut state = base.clone();
        mutate(&mut state);
        let before = dbg(&state);
        assert!(
            materialize_native_continue_map_timing(Some(&mut state), &input).is_err(),
            "{name}: incomplete predecessor was accepted"
        );
        assert_eq!(dbg(&state), before, "{name}: rejected timing seed mutated state");
    }

    let mut tampered = input.clone();
    tampered.map_timing_seed.sprite_idle_cycle = 1;
    let mut state = base.clone();
    let before = dbg(&state);
    assert!(
        materialize_native_continue_map_timing(Some(&mut state), &tampered).is_err(),
        "tampered validated input was accepted"
    );
    assert_eq!(dbg(&state), before, "tampered input changed state");
}

#[test]
fn test_materialize_native_continue_interactive_boundary_installs_controller_mode() {
    let Some(mut catalog) = native_runtime_catalog() else { return };
    let input = native_pending_groups_input(1, None);
    let Some((mut state, asset_state, scenario, item_rows)) = native_pending_groups_fixture(&input) else { return };
    materialize_native_continue_runtime_units(Some(&mut state), &input, &mut catalog).unwrap();
    materialize_native_continue_map_timing(Some(&mut state), &input).unwrap();
    materialize_native_continue_pending_groups(
        Some(&mut state),
        &input,
        0,
        Some(&asset_state),
        Some(&scenario),
        &item_rows,
    )
    .unwrap();
    assert_eq!(state.native_map_range_mode, input.map_presentation.opening_range_mode, "opening selector");
    materialize_native_continue_interactive_boundary(Some(&mut state), &input).unwrap();
    assert_eq!(state.native_map_range_mode, input.map_presentation.interactive_range_mode, "interactive selector");
    validate_native_continue_battle_handoff(&input, Some(&state), Some(&scenario)).unwrap();
}

#[test]
fn test_validate_native_continue_battle_handoff_rejects_incomplete_adapters() {
    let input = native_pending_groups_input(1, None);
    let Some((mut state, _, scenario, _)) = native_pending_groups_fixture(&input) else { return };
    state.has_native_runtime_unit_projection = false;
    assert!(
        validate_native_continue_battle_handoff(&input, Some(&state), Some(&scenario)).is_err(),
        "incomplete CONTINUE adapters were accepted"
    );
}
