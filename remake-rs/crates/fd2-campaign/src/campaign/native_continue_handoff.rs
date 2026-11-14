//! Go 來源：`remake/internal/campaign/native_continue_handoff.go`（110 行）。

use super::*;

/// MaterializeNativeContinueInteractiveBoundary records the second, verified
/// selector write performed by the native CONTINUE loader immediately before
/// returning to the shared 0x117E7 controller.  The opening redraw owns mode 0;
/// the controller owns the later interactive mode.  No opening redraw, clock
/// sample, scenario setup, or unit construction is performed here.
///
/// Go：`native_continue_handoff.go:15`，`func MaterializeNativeContinueInteractiveBoundary( state *battle.State, input fdsave.ContinueRuntimeInput, ) error`
pub fn materialize_native_continue_interactive_boundary(
    state: Option<&mut battle::State>,
    input: &save::ContinueRuntimeInput,
) -> Result<()> {
    let Some(state) = state else {
        return Err(campaign_err!("native CONTINUE interactive boundary: nil battle state"));
    };
    if !input.validated_for_runtime_bridge() {
        return Err(campaign_err!("native CONTINUE interactive boundary: input did not pass preflight"));
    }
    if !state.has_native_field_control_state
        || !state.has_native_runtime_unit_projection
        || !state.has_native_pending_group_binding
        || !state.has_native_map_binary_timing_state
        || !state.has_native_map_view_state
        || !state.has_native_map_hud_state
        || !state.has_native_map_range_mode_state
    {
        return Err(campaign_err!("native CONTINUE interactive boundary: prior runtime boundary is incomplete"));
    }
    if state.native_map_range_mode != input.map_presentation.opening_range_mode {
        return Err(campaign_err!("native CONTINUE interactive boundary: opening selector is not installed"));
    }

    let mut candidate = state.clone();
    if !candidate.materialize_native_map_range_mode(input.map_presentation.interactive_range_mode) {
        return Err(campaign_err!("native CONTINUE interactive boundary: interactive selector is invalid"));
    }
    *state = candidate;
    Ok(())
}

/// ValidateNativeContinueBattleHandoff is the fail-closed boundary between
/// the four typed CONTINUE adapters and the player-facing battle controller.
/// It validates only fields already proven by the native save/resource traces;
/// it does not infer a chapter, call Scenario.Setup, or consume the unresolved
/// owner list as if it were a gameplay result.
///
/// Go：`native_continue_handoff.go:53`，`func ValidateNativeContinueBattleHandoff( input fdsave.ContinueRuntimeInput, state *battle.State, scenario *battle.Scenario, ) error`
pub fn validate_native_continue_battle_handoff(
    input: &save::ContinueRuntimeInput,
    state: Option<&battle::State>,
    scenario: Option<&battle::Scenario>,
) -> Result<()> {
    if !input.validated_for_runtime_bridge() {
        return Err(campaign_err!("native CONTINUE battle handoff: input did not pass preflight"));
    }
    let (Some(state), Some(scenario)) = (state, scenario) else {
        return Err(campaign_err!("native CONTINUE battle handoff: missing state or scenario"));
    };
    if scenario.chapter != input.context.chapter + 1
        || scenario.map != input.context.chapter
        || state.w != input.context.field_width
        || state.h != input.context.field_height
    {
        return Err(campaign_err!("native CONTINUE battle handoff: chapter asset mismatch"));
    }
    if !state.has_native_field_control_state
        || !state.has_native_runtime_unit_projection
        || !state.has_native_pending_group_binding
        || !state.has_native_map_binary_timing_state
        || !state.has_native_map_view_state
        || !state.has_native_map_hud_state
        || !state.has_native_map_range_mode_state
        || !state.has_native_map_cycle_state
        || !state.has_native_terrain_phase_state
    {
        return Err(campaign_err!("native CONTINUE battle handoff: runtime adapters are incomplete"));
    }
    if state.native_map_range_mode != input.map_presentation.interactive_range_mode {
        return Err(campaign_err!("native CONTINUE battle handoff: interactive selector is not installed"));
    }
    let view = &state.native_map_view_state;
    let header = &input.header;
    if view.camera_x != i64::from(header.camera_x)
        || view.camera_y != i64::from(header.camera_y)
        || view.cursor_x != i64::from(header.cursor_x)
        || view.cursor_y != i64::from(header.cursor_y)
        || view.visible_cursor_x != i64::from(header.visible_cursor_x)
        || view.visible_cursor_y != i64::from(header.visible_cursor_y)
    {
        return Err(campaign_err!("native CONTINUE battle handoff: saved map view was not preserved"));
    }
    if state.native_round_counter != i64::from(header.turn_counter) {
        return Err(campaign_err!("native CONTINUE battle handoff: saved turn was not preserved"));
    }
    if state.native_map_hud_state.display_gate_a != header.hud_gate_a
        || state.native_map_hud_state.display_gate_b != input.map_presentation.hud_gate_b
        || state.native_map_hud_state.anchor_x != input.map_presentation.hud_anchor_x
    {
        return Err(campaign_err!("native CONTINUE battle handoff: saved HUD state was not preserved"));
    }
    // Go 另檢查 `state.Roster == nil || state.PendingGroups == nil`。Rust 的 `Vec`／`BTreeMap`
    // 不分 nil 與空值；兩者只由 MaterializeNativeContinuePendingGroups 一起設定（Go 給非 nil 的空值），
    // 上面的 `has_native_pending_group_binding` 已涵蓋（見黃金測試 KNOWN_DIFFERENCES）。
    if state.native_runtime_records.len() != input.runtime_records.len()
        || state.units.len() != input.runtime_records.len()
        || state.native_map_selector_cache.is_none()
        || state.native_map_selector_error.is_some()
    {
        return Err(campaign_err!("native CONTINUE battle handoff: runtime roster is incomplete"));
    }
    Ok(())
}
