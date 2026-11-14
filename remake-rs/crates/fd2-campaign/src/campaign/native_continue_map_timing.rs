//! Go 來源：`remake/internal/campaign/native_continue_map_timing.go`（73 行）。

use super::*;

/// MaterializeNativeContinueMapTiming installs the exact process-global seed
/// observed immediately before sub_10010's first 0x11CAC redraw. It does not
/// sample a clock or schedule either redraw; those are caller-owned operations.
///
/// Field and runtime-unit adapters must already have succeeded. This preserves
/// the original order and prevents a timing-only partial CONTINUE state.
///
/// Go：`native_continue_map_timing.go:18`，`func MaterializeNativeContinueMapTiming( state *battle.State, input fdsave.ContinueRuntimeInput, ) error`
pub fn materialize_native_continue_map_timing(
    state: Option<&mut battle::State>,
    input: &save::ContinueRuntimeInput,
) -> Result<()> {
    let Some(state) = state else {
        return Err(campaign_err!("native CONTINUE map timing: nil battle state"));
    };
    if !input.validated_for_runtime_bridge() {
        return Err(campaign_err!("native CONTINUE map timing: input did not pass preflight"));
    }
    if !state.has_native_field_control_state
        || !state.has_native_runtime_unit_projection
        || !state.has_native_map_view_state
        || !state.has_native_map_range_mode_state
        || state.native_map_range_mode != input.map_presentation.opening_range_mode
    {
        return Err(campaign_err!("native CONTINUE map timing: prior runtime boundary is incomplete"));
    }

    let seed = &input.map_timing_seed;
    let word = |v: i64| (-0x8000..=0x7fff).contains(&v);
    if !(0..=3).contains(&seed.sprite_idle_cycle)
        || !(0..=3).contains(&seed.sprite_moving_cycle)
        || !word(seed.sprite_last_timer_tick)
        || !(0..20).contains(&seed.terrain_phase)
        || !word(seed.terrain_phase_last_timer_tick)
        || seed.terrain_phase_override != -1
        || (seed.terrain_flip != 0 && seed.terrain_flip != 1)
        || !word(seed.terrain_flip_last_timer_tick)
        || (seed.unit_pixel_shift != 0 && seed.unit_pixel_shift != 1)
        || !word(seed.unit_pixel_shift_last_timer_tick)
    {
        return Err(campaign_err!("native CONTINUE map timing: seed is outside raw bounds"));
    }

    let mut candidate = state.clone();
    candidate.native_map_cycle_state = fdicon::NativeMapSpriteCycleState {
        idle: seed.sprite_idle_cycle,
        moving: seed.sprite_moving_cycle,
        last_timer_tick: seed.sprite_last_timer_tick,
    };
    candidate.native_terrain_phase_state = fdother::NativeTerrainPhaseState {
        phase: seed.terrain_phase,
        last_timer_tick: seed.terrain_phase_last_timer_tick,
    };
    candidate.native_terrain_flip_state =
        fdicon::NativeBinaryTickState { value: seed.terrain_flip, last_timer_tick: seed.terrain_flip_last_timer_tick };
    candidate.native_unit_pixel_shift_state = fdicon::NativeBinaryTickState {
        value: seed.unit_pixel_shift,
        last_timer_tick: seed.unit_pixel_shift_last_timer_tick,
    };
    candidate.has_native_map_cycle_state = true;
    candidate.has_native_terrain_phase_state = true;
    candidate.has_native_map_binary_timing_state = true;
    *state = candidate;
    Ok(())
}

#[cfg(test)]
mod native_continue_map_timing_test;
