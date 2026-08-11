package campaign

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestMaterializeNativeContinueMapTimingInstallsExactSeed(t *testing.T) {
	input := continueFieldBoundaryInput(t)
	state := continueFieldBoundaryState()
	if err := MaterializeNativeContinueFieldBoundary(state, input, 1); err != nil {
		t.Fatal(err)
	}
	if err := MaterializeNativeContinueRuntimeUnits(
		state, input, nativeRuntimeCatalog(t),
	); err != nil {
		t.Fatal(err)
	}
	if err := MaterializeNativeContinueMapTiming(state, input); err != nil {
		t.Fatal(err)
	}
	if got := state.NativeMapCycleState; got != (fdicon.NativeMapSpriteCycleState{
		LastTimerTick: -123,
	}) {
		t.Fatalf("sprite seed=%+v", got)
	}
	if got := state.NativeTerrainPhaseState; got != (fdother.NativeTerrainPhaseState{}) {
		t.Fatalf("terrain seed=%+v", got)
	}
	if got := state.NativeTerrainFlipState; got != (fdicon.NativeBinaryTickState{}) {
		t.Fatalf("terrain flip seed=%+v", got)
	}
	if got := state.NativeUnitPixelShiftState; got != (fdicon.NativeBinaryTickState{}) {
		t.Fatalf("unit shift seed=%+v", got)
	}
	if !state.HasNativeMapCycleState || !state.HasNativeTerrainPhaseState ||
		!state.HasNativeMapBinaryTimingState {
		t.Fatal("timing provenance flags were not installed")
	}
}

func TestMaterializeNativeContinueMapTimingRejectsAtomically(t *testing.T) {
	input := continueFieldBoundaryInput(t)
	base := continueFieldBoundaryState()
	if err := MaterializeNativeContinueFieldBoundary(base, input, 1); err != nil {
		t.Fatal(err)
	}
	if err := MaterializeNativeContinueRuntimeUnits(
		base, input, nativeRuntimeCatalog(t),
	); err != nil {
		t.Fatal(err)
	}

	tests := map[string]func(*battle.State){
		"missing field boundary": func(state *battle.State) {
			state.HasNativeFieldControlState = false
		},
		"missing runtime units": func(state *battle.State) {
			state.HasNativeRuntimeUnitProjection = false
		},
		"wrong opening selector": func(state *battle.State) {
			state.NativeMapRangeMode = 1
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			state := *base
			mutate(&state)
			before := state
			if err := MaterializeNativeContinueMapTiming(&state, input); err == nil {
				t.Fatal("incomplete predecessor was accepted")
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected timing seed mutated state")
			}
		})
	}

	tampered := input
	tampered.MapTimingSeed.SpriteIdleCycle = 1
	state := *base
	before := state
	if err := MaterializeNativeContinueMapTiming(&state, tampered); err == nil {
		t.Fatal("tampered validated input was accepted")
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("tampered input changed state")
	}
}

func TestMaterializeNativeContinueInteractiveBoundaryInstallsControllerMode(t *testing.T) {
	input := nativePendingGroupsInput(t, 1, nil)
	state, assetState, scenario, itemRows := nativePendingGroupsFixture(t, input)
	if err := MaterializeNativeContinueRuntimeUnits(
		state, input, nativeRuntimeCatalog(t),
	); err != nil {
		t.Fatal(err)
	}
	if err := MaterializeNativeContinueMapTiming(state, input); err != nil {
		t.Fatal(err)
	}
	if err := MaterializeNativeContinuePendingGroups(
		state, input, 0, assetState, scenario, itemRows,
	); err != nil {
		t.Fatal(err)
	}
	if state.NativeMapRangeMode != input.MapPresentation.OpeningRangeMode {
		t.Fatalf("opening selector=%d, want %d", state.NativeMapRangeMode, input.MapPresentation.OpeningRangeMode)
	}
	if err := MaterializeNativeContinueInteractiveBoundary(state, input); err != nil {
		t.Fatal(err)
	}
	if state.NativeMapRangeMode != input.MapPresentation.InteractiveRangeMode {
		t.Fatalf("interactive selector=%d, want %d", state.NativeMapRangeMode, input.MapPresentation.InteractiveRangeMode)
	}
	if err := ValidateNativeContinueBattleHandoff(input, state, scenario); err != nil {
		t.Fatal(err)
	}
}

func TestValidateNativeContinueBattleHandoffRejectsIncompleteAdapters(t *testing.T) {
	input := nativePendingGroupsInput(t, 1, nil)
	state, _, scenario, _ := nativePendingGroupsFixture(t, input)
	state.HasNativeRuntimeUnitProjection = false
	if err := ValidateNativeContinueBattleHandoff(input, state, scenario); err == nil {
		t.Fatal("incomplete CONTINUE adapters were accepted")
	}
}
