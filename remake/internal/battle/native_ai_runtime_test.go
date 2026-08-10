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
	actor := nativeAIRuntimeUnit(0, 0, 1, 2)
	actor.InventorySlots[0] = 1
	actor.NativeInventoryFlags[0] = 0x40
	// Physical selector one targets the opposite raw +6 group (zero); Camp is
	// intentionally not used as a substitute for that provenance.
	target := nativeAIRuntimeUnit(2, 0, 0, 0)
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

func TestNextAIPlanMode11BuildsOrderedDirectStages(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 11)
	actor.NativeCommandMask[0] = 1
	actor.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	actor.MP = 255
	target := nativeAIRuntimeUnit(2, 0, 0, 0)
	target.Camp = Own
	target.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
		NativeCommandBook:           nativeAIActionCommandBook(),
	}
	if err := state.BindNativeFutureItemRows(make([]byte, 2*NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil {
		if plan == nil {
			t.Fatal("mode11 plan is nil")
		}
		t.Fatalf("mode11 plan=%+v err=%v", plan, plan.NativeError)
	}
	if len(plan.NativeMode11Stages) != 2 {
		t.Fatalf("mode11 stages=%+v, want two direct stages", plan.NativeMode11Stages)
	}
	if plan.NativeMode11Stages[0].Stage != (NativeAIMode11Stage{Ordinal: 1, Route: NativeAIMode11Call15311}) {
		t.Fatalf("first mode11 stage=%+v", plan.NativeMode11Stages[0].Stage)
	}
	if plan.NativeMode11Stages[0].Action == nil ||
		plan.NativeMode11Stages[0].Action.NativeActionKind != NativeAIActionCommand {
		t.Fatalf("first mode11 action=%+v, want command owner", plan.NativeMode11Stages[0].Action)
	}
	if plan.NativeMode11Stages[1].Stage.Ordinal != 2 {
		t.Fatalf("second mode11 stage=%+v", plan.NativeMode11Stages[1].Stage)
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

func nativeAIActionCommandBook() []NativeCommandRecord {
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id] = NativeCommandRecord{ID: id, SelectionMode: 0, EffectMode: 0, TargetCode: 0}
	}
	// Synthetic raw tuple: the command producer can reach an adjacent target
	// from the selected destination, while its score remains visibly positive.
	book[0] = NativeCommandRecord{
		ID: 0, Damage: 50, Hit: 90, SelectionMode: 5, EffectMode: 1,
		MPCost: 2, TargetCode: 0,
	}
	return book
}

func TestNextAIPlanUses14EF0CommandWinnerAndRetainsRawTarget(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 2)
	actor.NativeCommandMask[0] = 1
	actor.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	target := nativeAIRuntimeUnit(2, 0, 0, 0)
	target.Camp = Own
	target.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
		NativeCommandBook:           nativeAIActionCommandBook(),
	}
	if err := state.BindNativeFutureItemRows(make([]byte, 2*NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil {
		t.Fatal("command plan is nil")
	}
	if plan.NativeError != nil {
		t.Fatalf("command plan=%+v err=%v", plan, plan.NativeError)
	}
	if plan.NativeActionKind != NativeAIActionCommand || plan.NativeCommandID != 0 ||
		plan.NativeAI14EF0Route != NativeAI14EF0Call15311 || plan.Target != target {
		t.Fatalf("command plan=%+v", plan)
	}
	if len(plan.Path) < 2 || plan.Path[len(plan.Path)-1] != (Cell{X: 1, Y: 0}) {
		t.Fatalf("command path=%v want destination (1,0)", plan.Path)
	}
}

func TestNextAIPlanUses14EF0ItemWinnerAndRetainsRawTarget(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 0, 2)
	actor.NativeInventoryFlags = []int{0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	actor.InventorySlots[0] = 1
	target := nativeAIRuntimeUnit(2, 0, 1, 0)
	target.Camp = Own
	target.HP = 1
	target.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
		NativeCommandBook:           nativeAIActionCommandBook(),
	}
	state.NativeCommandBook[0].MPCost = 99 // make the command unavailable
	rows := make([]byte, 2*NativeItemEffectRowSize)
	row := rows[NativeItemEffectRowSize:]
	row[0x0d] = 5 // one of the recovered positive score families
	row[0x10] = 5 // destination selection mode
	row[0x11] = 0 // selector zero flips raw target code to 1
	row[0x12] = 1 // adjacent target stage
	if err := state.BindNativeFutureItemRows(rows); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil {
		t.Fatal("item plan is nil")
	}
	if plan.NativeError != nil {
		t.Fatalf("item plan=%+v err=%v", plan, plan.NativeError)
	}
	if plan.NativeActionKind != NativeAIActionItem || plan.NativeItemSlot != 0 ||
		plan.NativeItemID != 1 || plan.NativeAI14EF0Route != NativeAI14EF0Call15055 ||
		plan.Target != target {
		t.Fatalf("item plan=%+v", plan)
	}
	if len(plan.Path) < 2 || plan.Path[len(plan.Path)-1] != (Cell{X: 1, Y: 0}) {
		t.Fatalf("item path=%v want destination (1,0)", plan.Path)
	}
}

func TestNextAIPlanRejectsRawModeWithout14EF0Tables(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 0, 1)
	target := nativeAIRuntimeUnit(1, 0, 1, 0)
	target.Camp = Own
	state := &State{W: 2, H: 1, Units: []*Unit{actor, target}}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError == nil {
		t.Fatalf("plan=%+v want fail-closed raw mode error", plan)
	}
}

func TestNextAIPlanUsesMode4RawDestinationFallback(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 4)
	actor.NativeRecordByte35, actor.NativeRecordByte36 = 1, 0
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil {
		t.Fatalf("mode4 plan=%+v", plan)
	}
	if plan.NativeModeFallback != 4 || plan.Target != nil || plan.NativeActionDestination != (Cell{X: 1, Y: 0}) {
		t.Fatalf("mode4 raw fallback plan=%+v", plan)
	}
	if len(plan.Path) != 2 || plan.Path[1] != (Cell{X: 1, Y: 0}) {
		t.Fatalf("mode4 path=%v", plan.Path)
	}
}

func TestNextAIPlanMode7RetainsRawByte5ArrivalWrite(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 7)
	actor.NativeRecordByte35, actor.NativeRecordByte36 = 1, 0
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil || plan.NativeModeFallback != 7 || !plan.NativeModeWriteByte5 {
		t.Fatalf("mode7 plan=%+v", plan)
	}
}

func TestNextAIPlanMode0UsesRawBlockedCoordinateBeforeNearestFallback(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 0)
	target := nativeAIRuntimeUnit(2, 0, 0, 0)
	target.Camp = Own
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil || plan.NativeModeFallback != 0 || plan.Target != nil {
		t.Fatalf("mode0 plan=%+v", plan)
	}
	if len(plan.Path) != 2 || plan.Path[1] != (Cell{X: 1, Y: 0}) {
		t.Fatalf("mode0 path=%v", plan.Path)
	}
}

func TestNextAIPlanMode3Uses12C60RawRecord8Lookup(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 3)
	actor.NativeRecordByte35 = 7
	target := nativeAIRuntimeUnit(2, 0, 0, 0)
	target.NativeRecordByte8 = 7
	target.Camp = Own
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil || !plan.NativeModeFallbackActive ||
		plan.NativeModeFallback != 3 || !plan.NativeModeWriteRangeZero {
		t.Fatalf("mode3 plan=%+v", plan)
	}
	if len(plan.Path) != 2 || plan.Path[1] != (Cell{X: 1, Y: 0}) || plan.Target != nil {
		t.Fatalf("mode3 path=%v target=%v", plan.Path, plan.Target)
	}
}

func TestNextAIPlanMode9Uses12C60RawRecord8Lookup(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 9)
	actor.NativeRecordByte35 = 7
	target := nativeAIRuntimeUnit(2, 0, 0, 0)
	target.NativeRecordByte8 = 7
	target.Camp = Own
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil || !plan.NativeModeFallbackActive || plan.NativeModeFallback != 9 {
		t.Fatalf("mode9 plan=%+v", plan)
	}
	if plan.NativeModeWriteRangeZero || len(plan.Path) != 2 || plan.Path[1] != (Cell{X: 1, Y: 0}) {
		t.Fatalf("mode9 path/range=%v/%v", plan.Path, plan.NativeModeWriteRangeZero)
	}
}
