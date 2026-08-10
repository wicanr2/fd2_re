package battle

import "testing"

func nativeAIRuntimeUnit(x, y int, selector, mode byte) *Unit {
	u := &Unit{
		Camp:                     Enemy,
		X:                        x,
		Y:                        y,
		OnField:                  true,
		HP:                       20,
		MaxHP:                    20,
		MP:                       4,
		MaxMP:                    4,
		AP:                       20,
		DP:                       1,
		MV:                       2,
		BattleFig:                1,
		HasBattleFig:             true,
		NativeRecordByte5:        0,
		HasNativeRecordByte5:     true,
		NativeRecordByte6:        selector,
		HasNativeRecordByte6:     true,
		NativeRecordByte34:       mode,
		HasNativeRecordByte34:    true,
		NativeRecordByte35:       0,
		HasNativeRecordByte35:    true,
		NativeRecordByte36:       0,
		HasNativeRecordByte36:    true,
		NativeRecordByte8:        1,
		HasNativeRecordByte8:     true,
		NativeRecordRace:         0,
		HasNativeRecordRace:      true,
		NativeRecordClass:        0,
		HasNativeRecordClass:     true,
		NativeRecordWord42:       20,
		HasNativeRecordWord42:    true,
		NativeRecordWord46:       4,
		HasNativeRecordWord46:    true,
		NativeMapPresentation:    NativeMapPresentationState{X: byte(x), Y: byte(y)},
		HasNativeMapPresentation: true,
		InventorySlots:           []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags:     []int{0, 0, 0, 0, 0, 0, 0, 0},
	}
	return u
}

func nativeAIRuntimeCostRows() [][]byte {
	rows := make([][]byte, NativeMovementCostRowCount)
	for index := range rows {
		rows[index] = make([]byte, NativeMovementCostRowSize)
		for cell := range rows[index] {
			rows[index][cell] = 1
		}
	}
	return rows
}

func TestNextAIPlanUsesVerifiedMode2PhysicalCandidate(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 0, 2)
	actor.InventorySlots[0] = 1
	actor.NativeInventoryFlags[0] = 0x40
	target := nativeAIRuntimeUnit(2, 0, 1, 0)
	target.Camp = Own
	target.AP, target.DP = 1, 1
	state := &State{
		W:                           3,
		H:                           1,
		Units:                       []*Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
	}
	if err := state.BindNativeFutureItemRows(make([]byte, 2*NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	itemRows := state.nativeFutureItemRows
	itemRows[NativeItemEffectRowSize+0x0b] = 0
	itemRows[NativeItemEffectRowSize+0x0c] = 1
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil {
		t.Fatal("mode-2 plan is nil")
	}
	if plan.NativeError != nil {
		t.Fatalf("mode-2 plan failed: %v", plan.NativeError)
	}
	if !plan.NativeMode2Physical {
		t.Fatalf("plan did not retain native mode-2 provenance: %+v", plan)
	}
	if plan.Target != target {
		t.Fatalf("plan=%+v target=%p want %p", plan, plan.Target, target)
	}
	if len(plan.Path) < 2 || plan.Path[0] != (Cell{X: 0, Y: 0}) || plan.Path[len(plan.Path)-1] != (Cell{X: 1, Y: 0}) {
		t.Fatalf("path=%v want (0,0) to selected destination (1,0)", plan.Path)
	}
}

func TestNextAIPlanMode2FailsClosedWithoutMovementRows(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 0, 2)
	state := &State{W: 1, H: 1, Units: []*Unit{actor}, NativeCompositionEventBytes: []byte{0}, NativeTerrainMoveCodes: []byte{0}}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError == nil {
		t.Fatalf("plan=%+v want fail-closed native error", plan)
	}
}

func TestNextAIPlanMode2FailsClosedWithoutEquippedLowItem(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 0, 2)
	state := &State{
		W:                           1,
		H:                           1,
		Units:                       []*Unit{actor},
		NativeCompositionEventBytes: []byte{0},
		NativeTerrainMoveCodes:      []byte{0},
	}
	if err := state.BindNativeFutureItemRows(make([]byte, 2*NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError == nil {
		t.Fatalf("plan=%+v want missing-item fail-closed error", plan)
	}
}

func TestNextAIPlanMode2FailsClosedWithoutPhysicalCandidate(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 0, 2)
	actor.InventorySlots[0] = 1
	actor.NativeInventoryFlags[0] = 0x40
	state := &State{
		W:                           1,
		H:                           1,
		Units:                       []*Unit{actor},
		NativeCompositionEventBytes: []byte{0},
		NativeTerrainMoveCodes:      []byte{0},
	}
	if err := state.BindNativeFutureItemRows(make([]byte, 2*NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	itemRows := state.nativeFutureItemRows
	itemRows[NativeItemEffectRowSize+0x0b] = 0
	itemRows[NativeItemEffectRowSize+0x0c] = 1
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError == nil {
		t.Fatalf("plan=%+v want no-candidate fail-closed error", plan)
	}
}
