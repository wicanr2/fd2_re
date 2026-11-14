//! Go 來源：`remake/internal/campaign/native_continue_field_boundary.go`（108 行）。

use super::*;

/// MaterializeNativeContinueFieldBoundary installs only the live FDFIELD
/// control/view/HUD boundary which is already closed for title CONTINUE.
///
/// It deliberately does not materialize saved runtime 0x50 records, schedule
/// the two opening redraws, or start the battle driver. The caller must select
/// the chapter asset before calling and pass its raw chapter explicitly; a
/// mismatched asset fails before State is changed.
///
/// Go：`native_continue_field_boundary.go:17`，`func MaterializeNativeContinueFieldBoundary( state *battle.State, input fdsave.ContinueRuntimeInput, assetChapter int, ) error`
pub fn materialize_native_continue_field_boundary(
    state: Option<&mut battle::State>,
    input: &save::ContinueRuntimeInput,
    asset_chapter: i64,
) -> Result<()> {
    let Some(state) = state else {
        return Err(campaign_err!("native CONTINUE field boundary: nil battle state"));
    };
    if !input.validated_for_runtime_bridge() {
        return Err(campaign_err!("native CONTINUE field boundary: input did not pass preflight"));
    }
    if asset_chapter != input.context.chapter
        || state.w != input.context.field_width
        || state.h != input.context.field_height
    {
        return Err(campaign_err!("native CONTINUE field boundary: chapter asset mismatch"));
    }
    if input.field_control.units.len() != input.field_control.raw_unit_count as usize {
        return Err(campaign_err!("native CONTINUE field boundary: unit control count mismatch"));
    }
    if state.native_field_event_slots.len() as i64 != state.w.wrapping_mul(state.h)
        || state.native_field_events.len() != 16
    {
        return Err(campaign_err!("native CONTINUE field boundary: chapter field events are incomplete"));
    }

    // Go `candidate := *state` 是淺複製；之後只寫回陣列元素或整個換掉切片，且所有可能失敗的步驟都
    // 在寫入之前，所以 Rust 的深複製結果相同。
    let mut candidate = state.clone();
    let header = &input.header;
    candidate
        .materialize_native_map_view_state(battle::NativeMapViewState {
            camera_x: i64::from(header.camera_x),
            camera_y: i64::from(header.camera_y),
            cursor_x: i64::from(header.cursor_x),
            cursor_y: i64::from(header.cursor_y),
            visible_cursor_x: i64::from(header.visible_cursor_x),
            visible_cursor_y: i64::from(header.visible_cursor_y),
        })
        .map_err(|err| CampaignError::from(err).wrap("native CONTINUE field boundary: "))?;
    if !candidate.materialize_native_map_hud_state(
        header.hud_gate_a,
        input.map_presentation.hud_gate_b,
        input.map_presentation.hud_anchor_x,
    ) {
        return Err(campaign_err!("native CONTINUE field boundary: invalid HUD state"));
    }
    if !candidate.materialize_native_map_range_mode(input.map_presentation.opening_range_mode) {
        return Err(campaign_err!("native CONTINUE field boundary: invalid opening range mode"));
    }

    candidate.native_round_counter = i64::from(header.turn_counter);
    candidate.native_event_state = input.native_event_state;
    candidate.native_field_control_raw = input.native_field_control.to_vec();
    for (index, event) in input.field_control.turn_events.iter().enumerate() {
        candidate.native_turn_event_controls[index] =
            battle::NativeTurnEventControl { turn: event.turn, event_id: event.event_id, raw_camp: event.raw_camp };
    }
    candidate.has_native_turn_event_control_state = true;
    for (index, event) in input.field_control.field_events.iter().enumerate() {
        candidate.native_field_events[index] =
            battle::NativeFieldEvent { event_id: event.event_id, selector: event.selector };
    }
    for (index, chest) in input.field_control.chests.iter().enumerate() {
        candidate.native_chest_controls[index] =
            battle::NativeChestControl { raw_type: chest.raw_type, value: chest.value };
    }
    candidate.has_native_chest_control_state = true;
    candidate.native_field_unit_controls = input
        .field_control
        .units
        .iter()
        .map(|unit| {
            let mut control = battle::NativeFieldUnitControl::default();
            control.raw.copy_from_slice(&unit.raw);
            control
        })
        .collect();
    candidate.native_runtime_records = input
        .runtime_records
        .iter()
        .map(|record| battle::NativeRuntimeRecordState {
            raw: record.raw.raw,
            selector_key: record.selector_key,
            selector_slot: record.selector_slot,
        })
        .collect();
    candidate.has_native_field_control_state = true;

    *state = candidate;
    Ok(())
}

#[cfg(test)]
pub(crate) mod native_continue_field_boundary_test;
