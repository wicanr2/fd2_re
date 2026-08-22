package battle

import "testing"

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
	unit := nativeAIRuntimeUnit(0, 0, 2, 0)
	state := &State{
		W: 2, H: 1, Units: []*Unit{unit},
		NativeCompositionEventBytes: make([]byte, 2),
		NativeTerrainMoveCodes:      make([]byte, 2),
		nativeMovementCostRows:      nativeAIRuntimeCostRows(),
		NativeFieldEventSlots:       []int{-1, 0},
		NativeFieldEvents:           make([]NativeFieldEvent, 16),
	}
	state.NativeFieldEvents[0] = NativeFieldEvent{EventID: 82, Selector: 1}
	if _, err := state.PlanNativeSystemGroupMarch(Cell{X: 1, Y: 0}); err == nil {
		t.Fatal("unowned global event was accepted")
	}
	if unit.X != 0 || unit.Y != 0 || unit.Acted || unit.NativeRecordByte5 != 0 {
		t.Fatalf("event failure mutated unit: %+v", unit)
	}
	unit.HasNativeRecordByte6 = false
	state.NativeFieldEventSlots[1] = -1
	if _, err := state.PlanNativeSystemGroupMarch(Cell{X: 1, Y: 0}); err == nil {
		t.Fatal("missing raw gate was accepted")
	}
}
