package campaign

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// MaterializeNativeContinueFieldBoundary installs only the live FDFIELD
// control/view/HUD boundary which is already closed for title CONTINUE.
//
// It deliberately does not materialize saved runtime 0x50 records, schedule
// the two opening redraws, or start the battle driver. The caller must select
// the chapter asset before calling and pass its raw chapter explicitly; a
// mismatched asset fails before State is changed.
func MaterializeNativeContinueFieldBoundary(
	state *battle.State,
	input fdsave.ContinueRuntimeInput,
	assetChapter int,
) error {
	if state == nil {
		return fmt.Errorf("native CONTINUE field boundary: nil battle state")
	}
	if !input.ValidatedForRuntimeBridge() {
		return fmt.Errorf("native CONTINUE field boundary: input did not pass preflight")
	}
	if assetChapter != input.Context.Chapter ||
		state.W != input.Context.FieldWidth ||
		state.H != input.Context.FieldHeight {
		return fmt.Errorf("native CONTINUE field boundary: chapter asset mismatch")
	}
	if len(input.FieldControl.Units) != int(input.FieldControl.RawUnitCount) {
		return fmt.Errorf("native CONTINUE field boundary: unit control count mismatch")
	}
	if len(state.NativeFieldEventSlots) != state.W*state.H ||
		len(state.NativeFieldEvents) != 16 {
		return fmt.Errorf("native CONTINUE field boundary: chapter field events are incomplete")
	}

	candidate := *state
	if err := candidate.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CameraX:        int(input.Header.CameraX),
		CameraY:        int(input.Header.CameraY),
		CursorX:        int(input.Header.CursorX),
		CursorY:        int(input.Header.CursorY),
		VisibleCursorX: int(input.Header.VisibleCursorX),
		VisibleCursorY: int(input.Header.VisibleCursorY),
	}); err != nil {
		return fmt.Errorf("native CONTINUE field boundary: %w", err)
	}
	if !candidate.MaterializeNativeMapHUDState(
		input.Header.HUDGateA,
		input.MapPresentation.HUDGateB,
		input.MapPresentation.HUDAnchorX,
	) {
		return fmt.Errorf("native CONTINUE field boundary: invalid HUD state")
	}
	if !candidate.MaterializeNativeMapRangeMode(
		input.MapPresentation.OpeningRangeMode,
	) {
		return fmt.Errorf("native CONTINUE field boundary: invalid opening range mode")
	}

	candidate.NativeRoundCounter = int(input.Header.TurnCounter)
	candidate.NativeEventState = input.NativeEventState
	candidate.NativeFieldControlRaw = append(
		[]byte(nil),
		input.NativeFieldControl[:]...,
	)
	for index, event := range input.FieldControl.TurnEvents {
		candidate.NativeTurnEventControls[index] = battle.NativeTurnEventControl{
			Turn: event.Turn, EventID: event.EventID, RawCamp: event.RawCamp,
		}
	}
	candidate.HasNativeTurnEventControlState = true
	for index, event := range input.FieldControl.FieldEvents {
		candidate.NativeFieldEvents[index] = battle.NativeFieldEvent{
			EventID: event.EventID, Selector: event.Selector,
		}
	}
	for index, chest := range input.FieldControl.Chests {
		candidate.NativeChestControls[index] = battle.NativeChestControl{
			RawType: chest.RawType, Value: chest.Value,
		}
	}
	candidate.HasNativeChestControlState = true
	candidate.NativeFieldUnitControls = make(
		[]battle.NativeFieldUnitControl,
		len(input.FieldControl.Units),
	)
	for index, unit := range input.FieldControl.Units {
		copy(candidate.NativeFieldUnitControls[index].Raw[:], unit.Raw[:])
	}
	candidate.NativeRuntimeRecords = make(
		[]battle.NativeRuntimeRecordState,
		len(input.RuntimeRecords),
	)
	for index, record := range input.RuntimeRecords {
		copy(candidate.NativeRuntimeRecords[index].Raw[:], record.Raw.Raw[:])
		candidate.NativeRuntimeRecords[index].SelectorKey = record.SelectorKey
		candidate.NativeRuntimeRecords[index].SelectorSlot = record.SelectorSlot
	}
	candidate.HasNativeFieldControlState = true

	*state = candidate
	return nil
}
