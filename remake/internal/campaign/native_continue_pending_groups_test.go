package campaign

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func TestNativeContinueSpawnScheduleAcceptsAllNativeAppendScenarios(t *testing.T) {
	wantKeys := map[int]int{1: 4, 2: 1, 3: 1, 26: 9}
	for chapter, want := range wantKeys {
		t.Run(fmt.Sprintf("ch%02d", chapter), func(t *testing.T) {
			scenario, err := battle.LoadScenario(fmt.Sprintf(
				"../../assets/scenarios/ch%02d.json", chapter,
			))
			if err != nil {
				t.Fatal(err)
			}
			if !scenario.RuntimeAppendGroups {
				t.Fatal("scenario lost native append ownership")
			}
			schedule, err := nativeContinueSpawnSchedule(scenario)
			if err != nil {
				t.Fatal(err)
			}
			if len(schedule) != want {
				t.Fatalf("schedule keys=%d, want %d", len(schedule), want)
			}
		})
	}
}

func TestMaterializeNativeContinuePendingGroupsBindsCurrentAndFutureSchedule(t *testing.T) {
	input := nativePendingGroupsInput(t, 1, nil)
	state, assetState, scenario, itemRows := nativePendingGroupsFixture(t, input)
	beforeUnits := append([]*battle.Unit(nil), state.Units...)
	beforeControls := state.NativeTurnEventControls

	if err := MaterializeNativeContinuePendingGroups(
		state, input, 0, assetState, scenario, itemRows,
	); err != nil {
		t.Fatal(err)
	}
	if !state.HasNativePendingGroupBinding || len(state.Roster) != 15 ||
		!reflect.DeepEqual(state.Units, beforeUnits) ||
		state.NativeTurnEventControls != beforeControls {
		t.Fatalf("pending-group state was not isolated: %#v", state)
	}
	wantCounts := map[int]int{3: 1, 4: 4, 5: 5, 6: 4, 7: 1}
	gotCounts := map[int]int{}
	for _, unit := range state.Roster {
		gotCounts[unit.Group]++
	}
	if !reflect.DeepEqual(gotCounts, wantCounts) ||
		!reflect.DeepEqual(state.PendingGroups, map[int]bool{
			3: true, 4: true, 5: true, 6: true, 7: true,
		}) {
		t.Fatalf("roster groups=%v pending=%v", gotCounts, state.PendingGroups)
	}
	for _, forbidden := range []int{1, 2, 10, 11} {
		if state.PendingGroups[forbidden] {
			t.Fatalf("already-present/unscheduled group %d was bound", forbidden)
		}
	}
	// 綁定必須深複製；後續 constructor 不可污染資產 loader 的來源列。
	state.Roster[0].InventorySlots[0] ^= 0xff
	state.Roster[0].NativeConstructor.Record[0] ^= 0xff
	var source *battle.Unit
	for _, unit := range assetState.Units {
		if unit.Group == state.Roster[0].Group {
			source = unit
			break
		}
	}
	if source == nil || source.InventorySlots[0] == state.Roster[0].InventorySlots[0] ||
		source.NativeConstructor.Record[0] == state.Roster[0].NativeConstructor.Record[0] {
		t.Fatal("pending roster shares mutable storage with chapter asset")
	}
}

func TestMaterializeNativeContinuePendingGroupsAcceptsProvenStaticScenario(t *testing.T) {
	input := nativePendingGroupsInput(t, 1, nil)
	state, assetState, scenario, itemRows := nativePendingGroupsFixture(t, input)
	scenario.RuntimeAppendGroups = false
	scenario.Events = nil
	assetState.NativeFieldEventRules = nil
	state.NativeFieldEventRules = nil
	if err := MaterializeNativeContinuePendingGroups(
		state, input, 0, assetState, scenario, itemRows,
	); err != nil {
		t.Fatal(err)
	}
	if !state.HasNativePendingGroupBinding || state.Roster == nil ||
		len(state.Roster) != 0 || state.PendingGroups == nil || len(state.PendingGroups) != 0 {
		t.Fatalf("靜態 pending binding 不完整：roster=%v pending=%v bound=%v",
			state.Roster, state.PendingGroups, state.HasNativePendingGroupBinding)
	}
}

func TestMaterializeNativeContinuePendingGroupsRejectsStaticFutureSchedule(t *testing.T) {
	input := nativePendingGroupsInput(t, 1, nil)
	state, assetState, scenario, itemRows := nativePendingGroupsFixture(t, input)
	scenario.RuntimeAppendGroups = false
	if err := MaterializeNativeContinuePendingGroups(
		state, input, 0, assetState, scenario, itemRows,
	); err == nil {
		t.Fatal("缺少 runtime append owner 的 future schedule 被接受")
	}
	if state.HasNativePendingGroupBinding || state.Roster != nil || state.PendingGroups != nil {
		t.Fatalf("失敗的靜態 future schedule 污染 state：roster=%v pending=%v bound=%v",
			state.Roster, state.PendingGroups, state.HasNativePendingGroupBinding)
	}
}

func TestMaterializeNativeContinuePendingGroupsKeepsSavedTurnPending(t *testing.T) {
	input := nativePendingGroupsInput(t, 4, nil)
	state, assetState, scenario, itemRows := nativePendingGroupsFixture(t, input)
	if err := MaterializeNativeContinuePendingGroups(
		state, input, 0, assetState, scenario, itemRows,
	); err != nil {
		t.Fatal(err)
	}
	want := map[int]bool{4: true, 5: true, 6: true}
	if !reflect.DeepEqual(state.PendingGroups, want) {
		t.Fatalf("saved-turn pending groups=%v, want %v", state.PendingGroups, want)
	}
}

func TestMaterializeNativeContinuePendingGroupsExcludesSavedTurnSelector2(t *testing.T) {
	input := nativePendingGroupsInput(t, 3, func(raw *[fdsave.CurrentFieldControlSize]byte) {
		raw[5] = 2 // turn3/event0 已在上一輪增加到3後的 selector2 checkpoint 掃描。
	})
	state, assetState, scenario, itemRows := nativePendingGroupsFixture(t, input)
	if err := MaterializeNativeContinuePendingGroups(
		state, input, 0, assetState, scenario, itemRows,
	); err != nil {
		t.Fatal(err)
	}
	want := map[int]bool{4: true, 5: true, 6: true}
	if !reflect.DeepEqual(state.PendingGroups, want) {
		t.Fatalf("selector2 pending groups=%v, want %v", state.PendingGroups, want)
	}
}

func TestMaterializeNativeContinuePendingGroupsRejectsMutatedScheduleAtomically(t *testing.T) {
	input := nativePendingGroupsInput(t, 1, func(raw *[fdsave.CurrentFieldControlSize]byte) {
		raw[3] = 2 // event 0 的 live turn 已被改寫，不能用 scenario turn 3 猜回。
	})
	state, assetState, scenario, itemRows := nativePendingGroupsFixture(t, input)
	beforeRoster := state.Roster
	beforePending := state.PendingGroups
	beforeUnits := append([]*battle.Unit(nil), state.Units...)
	if err := MaterializeNativeContinuePendingGroups(
		state, input, 0, assetState, scenario, itemRows,
	); err == nil {
		t.Fatal("mutated live turn schedule was accepted")
	}
	if state.HasNativePendingGroupBinding || state.Roster != nil ||
		!reflect.DeepEqual(state.Roster, beforeRoster) ||
		!reflect.DeepEqual(state.PendingGroups, beforePending) ||
		!reflect.DeepEqual(state.Units, beforeUnits) {
		t.Fatalf("failed pending binding mutated state=%#v", state)
	}
}

func TestMaterializeNativeContinuePendingGroupsRejectsPastRowMovedToFuture(t *testing.T) {
	input := nativePendingGroupsInput(t, 4, func(raw *[fdsave.CurrentFieldControlSize]byte) {
		raw[3] = 5 // 原 turn3/event0 已過，但 handler 將 live row 延到未來。
	})
	state, assetState, scenario, itemRows := nativePendingGroupsFixture(t, input)
	beforeControls := state.NativeTurnEventControls
	if err := MaterializeNativeContinuePendingGroups(
		state, input, 0, assetState, scenario, itemRows,
	); err == nil {
		t.Fatal("past schedule row moved to the future was accepted")
	}
	if state.HasNativePendingGroupBinding || state.Roster != nil ||
		state.PendingGroups != nil || state.NativeTurnEventControls != beforeControls {
		t.Fatalf("failed shifted-row binding mutated state=%#v", state)
	}
}

func TestMaterializeNativeContinuePendingGroupsRejectsWrongSameSizeMapAtomically(t *testing.T) {
	input := nativePendingGroupsInput(t, 1, nil)
	state, assetState, scenario, itemRows := nativePendingGroupsFixture(t, input)
	state.NativeCompositionEventBytes[0] ^= 0xff
	beforeComposition := append([]byte(nil), state.NativeCompositionEventBytes...)
	if err := MaterializeNativeContinuePendingGroups(
		state, input, 0, assetState, scenario, itemRows,
	); err == nil {
		t.Fatal("same-size state with different map composition was accepted")
	}
	if state.HasNativePendingGroupBinding || state.Roster != nil ||
		state.PendingGroups != nil ||
		!reflect.DeepEqual(state.NativeCompositionEventBytes, beforeComposition) {
		t.Fatalf("failed map-identity check mutated state=%#v", state)
	}
}

func TestMaterializeNativeContinuePendingGroupsUsesMap25Event61OnceState(t *testing.T) {
	assetState, err := battle.Load("../../assets/maps/map25/map25_units.json")
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := battle.LoadScenario("../../assets/scenarios/ch26.json")
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(
		"../../assets/data/native_item_effect_rows.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	build := func(once byte, mutateLiveSelector bool) (fdsave.ContinueRuntimeInput, *battle.State) {
		t.Helper()
		snapshot := fdsave.CurrentSnapshot{Header: fdsave.CurrentRuntimeHeader{
			TurnCounter: 1, Chapter: 25,
			CameraX: 0, CameraY: 0, CursorX: 7, CursorY: 4,
			VisibleCursorX: 7, VisibleCursorY: 4,
		}}
		snapshot.NativeFieldControl[2] = byte(len(assetState.Units))
		for index, turn := range []byte{2, 4, 6, 8, 10, 12, 15, 16, 17} {
			copy(snapshot.NativeFieldControl[3+index*3:], []byte{turn, 57, 0})
		}
		const fieldEventOffset = 3 + 16*3
		for index, event := range assetState.NativeFieldEvents {
			snapshot.NativeFieldControl[fieldEventOffset+index*2] = event.EventID
			snapshot.NativeFieldControl[fieldEventOffset+index*2+1] = event.Selector
		}
		if mutateLiveSelector {
			snapshot.NativeFieldControl[fieldEventOffset+2*2+1] = 0
		}
		snapshot.NativeEventState[12] = once
		input, inputErr := fdsave.BuildContinueRuntimeInput(
			snapshot,
			fdsave.ContinueRuntimeContext{
				Chapter: 25, FieldWidth: assetState.W, FieldHeight: assetState.H,
				SelectorGroupCount: 256,
				TitleTimerTick:     0, HasTitleTimerTick: true,
			},
		)
		if inputErr != nil {
			t.Fatal(inputErr)
		}
		state := &battle.State{
			W: assetState.W, H: assetState.H,
			NativeCompositionEventBytes: append(
				[]byte(nil), assetState.NativeCompositionEventBytes...,
			),
			NativeFieldEventSlots: append(
				[]int(nil), assetState.NativeFieldEventSlots...,
			),
			NativeFieldEvents: append(
				[]battle.NativeFieldEvent(nil), assetState.NativeFieldEvents...,
			),
			NativeFieldEventRules: append(
				[]battle.NativeFieldEventRule(nil), assetState.NativeFieldEventRules...,
			),
		}
		if boundaryErr := MaterializeNativeContinueFieldBoundary(
			state, input, 25,
		); boundaryErr != nil {
			t.Fatal(boundaryErr)
		}
		state.HasNativeRuntimeUnitProjection = true
		return input, state
	}

	input, state := build(0, false)
	if err := MaterializeNativeContinuePendingGroups(
		state, input, 25, assetState, scenario, itemRows,
	); err != nil {
		t.Fatal(err)
	}
	if !state.PendingGroups[1] {
		t.Fatalf("unconsumed map25/event61 group1 was not bound: %v", state.PendingGroups)
	}

	consumedInput, consumed := build(1, false)
	if err := MaterializeNativeContinuePendingGroups(
		consumed, consumedInput, 25, assetState, scenario, itemRows,
	); err != nil {
		t.Fatal(err)
	}
	if consumed.PendingGroups[1] {
		t.Fatalf("consumed map25/event61 group1 was rebound: %v", consumed.PendingGroups)
	}

	wrongInput, wrong := build(0, true)
	if err := MaterializeNativeContinuePendingGroups(
		wrong, wrongInput, 25, assetState, scenario, itemRows,
	); err == nil {
		t.Fatal("map25/event61 with the wrong live selector was accepted")
	}
	if wrong.HasNativePendingGroupBinding || wrong.Roster != nil || wrong.PendingGroups != nil {
		t.Fatalf("rejected live field-event binding mutated state=%#v", wrong)
	}
}

func nativePendingGroupsInput(
	t *testing.T,
	turn byte,
	mutate func(*[fdsave.CurrentFieldControlSize]byte),
) fdsave.ContinueRuntimeInput {
	t.Helper()
	snapshot := fdsave.CurrentSnapshot{Header: fdsave.CurrentRuntimeHeader{
		TurnCounter: turn, Chapter: 0,
		CameraX: 0, CameraY: 0, CursorX: 7, CursorY: 4,
		VisibleCursorX: 7, VisibleCursorY: 4,
	}}
	snapshot.NativeFieldControl[2] = 30
	for index, row := range [][3]byte{
		{3, 0, 1}, {4, 1, 0}, {5, 2, 0}, {6, 3, 1},
	} {
		copy(snapshot.NativeFieldControl[3+index*3:], row[:])
	}
	if mutate != nil {
		mutate(&snapshot.NativeFieldControl)
	}
	input, err := fdsave.BuildContinueRuntimeInput(
		snapshot,
		fdsave.ContinueRuntimeContext{
			Chapter: 0, FieldWidth: 24, FieldHeight: 24,
			SelectorGroupCount: 256,
			TitleTimerTick:     0, HasTitleTimerTick: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func nativePendingGroupsFixture(
	t *testing.T,
	input fdsave.ContinueRuntimeInput,
) (*battle.State, *battle.State, *battle.Scenario, []byte) {
	t.Helper()
	assetState, err := battle.Load("../../assets/maps/map0/map0_units.json")
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := battle.LoadScenario("../../assets/scenarios/ch01.json")
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(
		"../../assets/data/native_item_effect_rows.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	state := &battle.State{
		W: assetState.W, H: assetState.H,
		NativeCompositionEventBytes: append(
			[]byte(nil), assetState.NativeCompositionEventBytes...,
		),
		NativeFieldEventSlots: append([]int(nil), assetState.NativeFieldEventSlots...),
		NativeFieldEvents: append(
			[]battle.NativeFieldEvent(nil), assetState.NativeFieldEvents...,
		),
		NativeFieldEventRules: append(
			[]battle.NativeFieldEventRule(nil), assetState.NativeFieldEventRules...,
		),
	}
	if err := MaterializeNativeContinueFieldBoundary(state, input, 0); err != nil {
		t.Fatal(err)
	}
	state.HasNativeRuntimeUnitProjection = true
	return state, assetState, scenario, itemRows
}
