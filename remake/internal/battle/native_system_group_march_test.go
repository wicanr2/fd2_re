package battle

import (
	"reflect"
	"testing"
)

func TestNativeSystemGroupMarchPlansInRecordOrderAndCommits(t *testing.T) {
	first := nativeAIRuntimeUnit(0, 0, 2, 0)
	second := nativeAIRuntimeUnit(0, 1, 2, 0)
	state := &State{
		W: 4, H: 2, Units: []*Unit{first, second},
		NativeCompositionEventBytes: make([]byte, 8),
		NativeTerrainMoveCodes:      make([]byte, 8),
		nativeMovementCostRows:      nativeAIRuntimeCostRows(),
		NativeFieldEventSlots:       []int{-1, -1, -1, -1, -1, -1, -1, -1},
		NativeFieldEvents:           make([]NativeFieldEvent, 16),
	}
	plan, err := state.PlanNativeSystemGroupMarch(Cell{X: 3, Y: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) != 2 || plan.Steps[0].UnitIndex != 0 || plan.Steps[1].UnitIndex != 1 {
		t.Fatalf("steps=%#v", plan.Steps)
	}
	if first.X != 0 || first.Y != 0 || first.Acted || second.X != 0 || second.Y != 1 || second.Acted {
		t.Fatal("planner mutated live units")
	}
	for _, step := range plan.Steps {
		if err := state.CommitNativeSystemGroupMarchStep(step); err != nil {
			t.Fatal(err)
		}
	}
	if !first.Acted || first.NativeRecordByte5&0x80 == 0 ||
		!second.Acted || second.NativeRecordByte5&0x80 == 0 {
		t.Fatalf("commit flags first=%#x second=%#x", first.NativeRecordByte5, second.NativeRecordByte5)
	}
}

func TestNativeSystemGroupMarchFailsAtomicallyOnEventOrMissingRaw(t *testing.T) {
	unit := nativeAIRuntimeUnit(1, 0, 2, 0)
	state := &State{
		W: 2, H: 1, Units: []*Unit{unit},
		NativeCompositionEventBytes: make([]byte, 2),
		NativeTerrainMoveCodes:      make([]byte, 2),
		nativeMovementCostRows:      nativeAIRuntimeCostRows(),
		NativeFieldEventSlots:       []int{0, -1},
		NativeFieldEvents:           make([]NativeFieldEvent, 16),
	}
	state.NativeFieldEvents[0] = NativeFieldEvent{EventID: 82, Selector: 1}
	if _, err := state.PlanNativeSystemGroupMarch(Cell{X: 0, Y: 0}); err == nil {
		t.Fatal("unowned global event was accepted")
	}
	if unit.X != 1 || unit.Y != 0 || unit.Acted || unit.NativeRecordByte5 != 0 {
		t.Fatalf("event failure mutated unit: %+v", unit)
	}
	unit.HasNativeRecordByte6 = false
	state.NativeFieldEventSlots[0] = -1
	if _, err := state.PlanNativeSystemGroupMarch(Cell{X: 0, Y: 0}); err == nil {
		t.Fatal("missing raw gate was accepted")
	}
}

func TestNativeSystemGroupMarchPreflightsOwnedEvent75WithoutLiveMutation(t *testing.T) {
	unit := nativeAIRuntimeUnit(1, 0, 2, 0)
	unit.NativeRecordByte8 = 9
	unit.HasNativeRecordByte8 = true
	state := &State{
		W: 2, H: 1, Units: []*Unit{unit}, NativeRoundCounter: 8,
		NativeCompositionEventBytes: make([]byte, 2),
		NativeTerrainMoveCodes:      make([]byte, 2),
		nativeMovementCostRows:      nativeAIRuntimeCostRows(),
		NativeFieldEventSlots:       []int{0, -1},
		NativeFieldEvents:           make([]NativeFieldEvent, 16),
		NativeFieldEventRules: []NativeFieldEventRule{{
			EventID: 75, Selector: 1, TriggerGate: "record_byte6_nonzero",
			TurnChain: &NativeFieldTurnChain{
				Handler: "0x35c79", TriggerRecordByte8: 9,
				MismatchTextIndex: 0, SuccessTextIndex: 1,
				StateWrites: []NativeEventStateWrite{{Index: 17, Value: 1}, {Index: 16, Value: 4}},
				TurnActivations: []NativeTurnActivation{
					{Slot: 1, EventID: 76, RawCamp: 2, TurnDelta: 1},
					{Slot: 0, EventID: 74, RawCamp: 0, TurnDelta: 0},
				},
			},
		}},
		HasNativeTurnEventControlState: true,
	}
	state.NativeFieldEvents[0] = NativeFieldEvent{EventID: 75, Selector: 1}
	state.NativeTurnEventControls[0] = NativeTurnEventControl{Turn: 0xff, EventID: 74, RawCamp: 0}
	state.NativeTurnEventControls[1] = NativeTurnEventControl{Turn: 0xff, EventID: 76, RawCamp: 2}
	plan, err := state.PlanNativeSystemGroupMarch(Cell{X: 0, Y: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) != 1 || len(plan.Steps[0].Events) != 1 ||
		plan.Steps[0].Events[0] != (NativeSystemGroupMarchEvent{
			PathIndex: 1, EventID: 75, TextIndex: 1,
		}) {
		t.Fatalf("plan=%#v", plan)
	}
	if unit.X != 1 || unit.Acted || state.NativeEventState[16] != 0 ||
		state.NativeEventState[17] != 0 || state.NativeTurnEventControls[0].Turn != 0xff {
		t.Fatalf("preflight mutated live state: unit=%+v state=%v rows=%#v", unit, state.NativeEventState[16:18], state.NativeTurnEventControls[:2])
	}
}

func TestNativeSystemGroupMarchEvent61ProjectionExtendsDynamicRecordBound(t *testing.T) {
	actor := nativeAIRuntimeUnit(2, 0, 2, 0)
	actor.Inventory = []int{0xD0}
	actor.Equipped = []bool{false}
	actor.InventorySlots = []int{0xD0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	actor.NativeInventoryFlags = []int{0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	joined := nativeAIRuntimeUnit(1, 0, 2, 0)
	joined.Group = 1
	joined.Fig = 31
	joined.Camp = Own
	joined.InventorySlots = []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	joined.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	once, item, group, character := 12, 0xD0, 1, 31
	texts := NativeFieldTextIndices{MissingItem: 2, Success: 3, Final: 4}
	presentation := NativeFieldPresentation{
		Archive: "FDOTHER.DAT", Resource: 45, Frames: 59,
		Helper: "0x2935b", DestinationOffset: 48356, Stride: 320,
		Transparent: -1, DelayHelper: "0x17aa9", DelayTicks: 2,
	}
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor}, Roster: []*Unit{joined},
		NativeCompositionEventBytes: make([]byte, 3),
		NativeTerrainMoveCodes:      make([]byte, 3),
		nativeMovementCostRows:      nativeAIRuntimeCostRows(),
		NativeFieldEventSlots:       []int{-1, 0, -1},
		NativeFieldEvents:           make([]NativeFieldEvent, 16),
		NativeFieldEventRules: []NativeFieldEventRule{{
			EventID: 61, Selector: 1, OnceState: &once, RequiredItem: &item,
			ConsumeItem: true, SpawnGroup: &group, JoinCharacter: &character,
			TextIndices: &texts, Presentation: &presentation,
		}},
	}
	state.NativeFieldEvents[0] = NativeFieldEvent{EventID: 61, Selector: 1}
	plan, err := state.PlanNativeSystemGroupMarch(Cell{X: 0, Y: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) != 2 || plan.Steps[0].UnitIndex != 0 ||
		len(plan.Steps[0].Events) != 1 || plan.Steps[0].Events[0].EventID != 61 ||
		plan.Steps[1].UnitIndex != 1 {
		t.Fatalf("dynamic event61 plan=%#v", plan)
	}
	if len(state.Units) != 1 || len(state.Roster) != 1 || state.NativeEventState[12] != 0 ||
		!reflect.DeepEqual(actor.Inventory, []int{0xD0}) || actor.X != 2 || actor.Acted {
		t.Fatalf("event61 preflight mutated live state: units=%d roster=%d state=%d actor=%+v", len(state.Units), len(state.Roster), state.NativeEventState[12], actor)
	}
}
