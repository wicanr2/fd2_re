package campaign

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// MaterializeNativeContinueInteractiveBoundary records the second, verified
// selector write performed by the native CONTINUE loader immediately before
// returning to the shared 0x117E7 controller.  The opening redraw owns mode 0;
// the controller owns the later interactive mode.  No opening redraw, clock
// sample, scenario setup, or unit construction is performed here.
func MaterializeNativeContinueInteractiveBoundary(
	state *battle.State,
	input fdsave.ContinueRuntimeInput,
) error {
	if state == nil {
		return fmt.Errorf("native CONTINUE interactive boundary: nil battle state")
	}
	if !input.ValidatedForRuntimeBridge() {
		return fmt.Errorf("native CONTINUE interactive boundary: input did not pass preflight")
	}
	if !state.HasNativeFieldControlState ||
		!state.HasNativeRuntimeUnitProjection ||
		!state.HasNativePendingGroupBinding ||
		!state.HasNativeMapBinaryTimingState ||
		!state.HasNativeMapViewState ||
		!state.HasNativeMapHUDState ||
		!state.HasNativeMapRangeModeState {
		return fmt.Errorf("native CONTINUE interactive boundary: prior runtime boundary is incomplete")
	}
	if state.NativeMapRangeMode != input.MapPresentation.OpeningRangeMode {
		return fmt.Errorf("native CONTINUE interactive boundary: opening selector is not installed")
	}

	candidate := *state
	if !candidate.MaterializeNativeMapRangeMode(
		input.MapPresentation.InteractiveRangeMode,
	) {
		return fmt.Errorf("native CONTINUE interactive boundary: interactive selector is invalid")
	}
	*state = candidate
	return nil
}

// ValidateNativeContinueBattleHandoff is the fail-closed boundary between
// the four typed CONTINUE adapters and the player-facing battle controller.
// It validates only fields already proven by the native save/resource traces;
// it does not infer a chapter, call Scenario.Setup, or consume the unresolved
// owner list as if it were a gameplay result.
func ValidateNativeContinueBattleHandoff(
	input fdsave.ContinueRuntimeInput,
	state *battle.State,
	scenario *battle.Scenario,
) error {
	if !input.ValidatedForRuntimeBridge() {
		return fmt.Errorf("native CONTINUE battle handoff: input did not pass preflight")
	}
	if state == nil || scenario == nil {
		return fmt.Errorf("native CONTINUE battle handoff: missing state or scenario")
	}
	if scenario.Chapter != input.Context.Chapter+1 ||
		scenario.Map != input.Context.Chapter ||
		state.W != input.Context.FieldWidth ||
		state.H != input.Context.FieldHeight {
		return fmt.Errorf("native CONTINUE battle handoff: chapter asset mismatch")
	}
	if !scenario.RuntimeAppendGroups {
		return fmt.Errorf("native CONTINUE battle handoff: scenario lacks native append ownership")
	}
	if !state.HasNativeFieldControlState ||
		!state.HasNativeRuntimeUnitProjection ||
		!state.HasNativePendingGroupBinding ||
		!state.HasNativeMapBinaryTimingState ||
		!state.HasNativeMapViewState ||
		!state.HasNativeMapHUDState ||
		!state.HasNativeMapRangeModeState ||
		!state.HasNativeMapCycleState ||
		!state.HasNativeTerrainPhaseState {
		return fmt.Errorf("native CONTINUE battle handoff: runtime adapters are incomplete")
	}
	if state.NativeMapRangeMode != input.MapPresentation.InteractiveRangeMode {
		return fmt.Errorf("native CONTINUE battle handoff: interactive selector is not installed")
	}
	view := state.NativeMapViewState
	if view.CameraX != int(input.Header.CameraX) ||
		view.CameraY != int(input.Header.CameraY) ||
		view.CursorX != int(input.Header.CursorX) ||
		view.CursorY != int(input.Header.CursorY) ||
		view.VisibleCursorX != int(input.Header.VisibleCursorX) ||
		view.VisibleCursorY != int(input.Header.VisibleCursorY) {
		return fmt.Errorf("native CONTINUE battle handoff: saved map view was not preserved")
	}
	if state.NativeRoundCounter != int(input.Header.TurnCounter) {
		return fmt.Errorf("native CONTINUE battle handoff: saved turn was not preserved")
	}
	if state.NativeMapHUDState.DisplayGateA != input.Header.HUDGateA ||
		state.NativeMapHUDState.DisplayGateB != input.MapPresentation.HUDGateB ||
		state.NativeMapHUDState.AnchorX != input.MapPresentation.HUDAnchorX {
		return fmt.Errorf("native CONTINUE battle handoff: saved HUD state was not preserved")
	}
	if len(state.NativeRuntimeRecords) != len(input.RuntimeRecords) ||
		len(state.Units) != len(input.RuntimeRecords) ||
		state.NativeMapSelectorCache == nil ||
		state.NativeMapSelectorError != nil ||
		state.Roster == nil ||
		state.PendingGroups == nil {
		return fmt.Errorf("native CONTINUE battle handoff: runtime roster is incomplete")
	}
	return nil
}
