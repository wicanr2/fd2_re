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

func TestNextAIPlanPhysicalHelperUsesFullTargetItemTable(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 2)
	actor.InventorySlots[0] = 1
	actor.NativeInventoryFlags[0] = 0x40
	target := nativeAIRuntimeUnit(2, 0, 0, 0)
	target.Camp = Own
	target.AP, target.DP = 1, 1
	target.InventorySlots[0] = 0x1f
	target.NativeInventoryFlags[0] = 0x40
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
	}
	rows := make([]byte, 0x20*NativeItemEffectRowSize)
	rows[NativeItemEffectRowSize+0x0c] = 1
	rows[0x1f*NativeItemEffectRowSize+0x0b] = 1
	if err := state.BindNativeFutureItemRows(rows); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil {
		if plan == nil {
			t.Fatal("mode-2 plan is nil")
		}
		t.Fatalf("mode-2 target item 0x1f plan error: %v", plan.NativeError)
	}
	if plan.Target != target {
		t.Fatalf("plan target=%p want %p", plan.Target, target)
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
	if got := plan.NativeMode11Stages[0].Action.Path; len(got) != 1 || got[0] != (Cell{X: 0, Y: 0}) {
		t.Fatalf("first mode11 command path=%v, want stationary actor origin", got)
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

// 0x1428B 0x1B83D(actor,0) 回 -1 → 0x14296 je 0x145C3 → xor eax,eax：0x14237 回傳 0，
// 與沒有物理候選同一條 0x13C06→0x13FD4 收尾，不是失敗即關閉。
func TestNextAIPlanMode2WithoutEquippedLowItemTakesIdleRecoveryTail(t *testing.T) {
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
	if plan == nil || plan.NativeError != nil || !plan.NativeMode2Physical || !plan.NativeModeFallbackActive ||
		plan.NativeModeFallback != 2 || plan.Target != nil || len(plan.Path) != 0 {
		t.Fatalf("plan=%+v want 0x13C06 idle-recovery tail", plan)
	}
}

func TestNextAIPlanMode2NoPhysicalCandidateUsesAccepted13FD4Decision(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 0, 2)
	actor.HP = 10
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
	if plan == nil || plan.NativeError != nil || !plan.NativeModeFallbackActive ||
		plan.NativeModeFallback != 2 || !plan.NativeMode2Physical || plan.NativeIdleRecovery == nil {
		t.Fatalf("plan=%+v want accepted mode-2 0x13fd4 fallback", plan)
	}
	if plan.NativeIdleRecovery.CurrentHP != 10 || plan.NativeIdleRecovery.MaximumHP != 20 ||
		plan.NativeIdleRecovery.NextHP != 14 {
		t.Fatalf("recovery=%+v want 10/20 -> 14", plan.NativeIdleRecovery)
	}
}

func TestNextAIPlanMode2NoPhysicalCandidateCompletesRejected13FD4Tail(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 0, 2)
	actor.InventorySlots[0] = 1
	actor.NativeInventoryFlags[0] = 0x40
	state := &State{
		W: 1, H: 1, Units: []*Unit{actor},
		NativeCompositionEventBytes: []byte{0}, NativeTerrainMoveCodes: []byte{0},
	}
	if err := state.BindNativeFutureItemRows(make([]byte, 2*NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	state.nativeFutureItemRows[NativeItemEffectRowSize+0x0c] = 1
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil || !plan.NativeModeFallbackActive ||
		plan.NativeModeFallback != 2 || plan.NativeIdleRecovery != nil || plan.Target != nil {
		t.Fatalf("plan=%+v want rejected 0x13fd4 common-tail plan", plan)
	}
}

func TestNextAIPlanMode2RoutesFailed14EF0ProducerTo13FD4(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 0, 2)
	actor.HP = 10
	actor.InventorySlots[0] = 1
	actor.NativeInventoryFlags[0] = 0x40
	state := &State{
		W: 1, H: 1, Units: []*Unit{actor},
		NativeCompositionEventBytes: []byte{0}, NativeTerrainMoveCodes: []byte{0},
		NativeCommandBook: make([]NativeCommandRecord, NativeCommandRecordCount),
	}
	if err := state.BindNativeFutureItemRows(make([]byte, 2*NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	state.nativeFutureItemRows[NativeItemEffectRowSize+0x0c] = 1
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil || !plan.NativeModeFallbackActive ||
		plan.NativeModeFallback != 2 || !plan.NativeMode2Physical || plan.NativeIdleRecovery == nil {
		t.Fatalf("plan=%+v want failed 0x14ef0 producer routed to mode-2 0x13fd4", plan)
	}
}

func TestNativeAIPhysicalHelper1DEBEKeepsRawPredicateBoundaries(t *testing.T) {
	target := make([]byte, nativeRecordSize)
	target[0], target[1] = 1, 0
	target[0x26] = 0
	target[0x0a] = 0x40
	target[0x0b] = 1
	itemRows := make([]byte, 2*NativeItemEffectRowSize)
	itemRows[NativeItemEffectRowSize+0x0b] = 1

	if got, err := nativeAIPhysicalHelper1DEBE(target, Cell{X: 0, Y: 0}, itemRows); err != nil || got != 1 {
		t.Fatalf("verified adjacent raw predicate = %d, err=%v; want 1", got, err)
	}

	target[0x26] = 1
	if got, err := nativeAIPhysicalHelper1DEBE(target, Cell{X: 0, Y: 0}, itemRows); err != nil || got != -1 {
		t.Fatalf("nonzero target +0x26 predicate = %d, err=%v; want -1", got, err)
	}
	target[0x26] = 0

	if got, err := nativeAIPhysicalHelper1DEBE(target, Cell{X: 0, Y: 2}, itemRows); err != nil || got != -1 {
		t.Fatalf("non-adjacent raw predicate = %d, err=%v; want -1", got, err)
	}

	itemRows[NativeItemEffectRowSize+0x0b] = 2
	if got, err := nativeAIPhysicalHelper1DEBE(target, Cell{X: 0, Y: 0}, itemRows); err != nil || got != -1 {
		t.Fatalf("raw item geometry >1 predicate = %d, err=%v; want -1", got, err)
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
	if len(plan.Path) != 1 || plan.Path[0] != (Cell{X: 0, Y: 0}) ||
		plan.NativeActionDestination != (Cell{X: 1, Y: 0}) {
		t.Fatalf("command path=%v destination=%v want stationary origin and effect (1,0)",
			plan.Path, plan.NativeActionDestination)
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
	if len(plan.NativeItemTargetIndices) != 1 || plan.NativeItemTargetIndices[0] != 1 {
		t.Fatalf("item target list=%v want detached raw [1]", plan.NativeItemTargetIndices)
	}
	// 0x15055 不呼叫 0x14B78：原地使用，(1,0) 只是效果格。
	if len(plan.Path) != 1 || plan.Path[0] != (Cell{X: 0, Y: 0}) ||
		plan.NativeActionDestination != (Cell{X: 1, Y: 0}) {
		t.Fatalf("item path=%v destination=%v want stationary origin and effect (1,0)",
			plan.Path, plan.NativeActionDestination)
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

func TestNextAIPlanMode4AtDestinationFocusesActorAndTakesIdleRecovery(t *testing.T) {
	// 0x13BE1..0x13C0F：mode 4 無條件 0x12D7B 聚焦自己，0x14B78 沒走成回 0 → 0x13FD4。
	// 第十章 r2 每回合 record 12／13 已在 (8,10)／(22,10) 仍各聚焦一次。
	actor := nativeAIRuntimeUnit(1, 0, 1, 4)
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
	if plan == nil || plan.NativeError != nil || plan.NativeModeFallback != 4 {
		t.Fatalf("mode4 plan=%+v", plan)
	}
	if len(plan.Path) > 1 || !plan.NativeModeFocusActor || !plan.NativeModeWriteRangeZero || plan.Target != nil {
		t.Fatalf("mode4 at destination plan=%+v", plan)
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

// 0x14121 只在 0x14B78 真的走了才回 1；阻擋格的規劃落在原格時，0x13A9F 的 mode 0
// 分派（0x13B0F）接著走 0x13E9C 以最近的對立單位再規劃。第七章 r6 第 7 回合記錄 23
// 的形狀：mode 2 搜尋最後接受的阻擋格是遠處的 T2，往它的長路徑第一步就被同組單位
// 佔住的零預算格截住；最近的 T1 才走得到。
func TestNextAIPlanMode0FallsBackToNearestWhenBlockedPlanStays(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 0)
	actor.MV = 1
	ally := nativeAIRuntimeUnit(0, 1, 1, 0)
	ally.MV = 1
	near := nativeAIRuntimeUnit(2, 0, 0, 0)
	near.Camp = Own
	far := nativeAIRuntimeUnit(0, 3, 0, 0)
	far.Camp = Own
	state := &State{
		W: 4, H: 4, Units: []*Unit{actor, ally, near, far},
		NativeCompositionEventBytes: make([]byte, 16),
		NativeTerrainMoveCodes:      make([]byte, 16),
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	blocked, found, err := NativePathBlockedCoordinate(
		4, 4, Cell{}, 28, mustNativeAIModeBlockedSearchFlags(t, state, 1), state.NativeTerrainMoveCodes,
		nativeAIRuntimeCostRows()[0],
	)
	if err != nil || !found || blocked != (Cell{X: 0, Y: 3}) {
		t.Fatalf("0x14121 blocked cell=%v found=%v err=%v want (0,3)", blocked, found, err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil || plan.NativeModeFallback != 0 || plan.Target != nil {
		t.Fatalf("mode0 plan=%+v", plan)
	}
	if len(plan.Path) != 2 || plan.Path[1] != (Cell{X: 1, Y: 0}) {
		t.Fatalf("mode0 path=%v want the 0x13E9C route toward (2,0) stopping at (1,0)", plan.Path)
	}
	if !plan.NativeModeBlockedFound || plan.NativeModeBlockedCell != (Cell{X: 0, Y: 3}) {
		t.Fatalf("mode0 plan should record the abandoned 0x14121 cell: %+v", plan)
	}
}

// 0x13E9C 聚焦後 0x14B78 沒走成就回 0，0x13B21 jmp 0x13C06 → 0x13FD4：第八章 r1 第 7 回合
// 記錄 36 被圍住，原版 HP 7→43（max 180 的五分之一）。計畫不移動，但要帶回復決策。
func TestNextAIPlanMode0StuckFallbackCarriesIdleRecovery(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 0)
	actor.MV = 1
	actor.HP = 7
	actor.MaxHP = 180
	blocker := nativeAIRuntimeUnit(1, 0, 1, 0)
	target := nativeAIRuntimeUnit(4, 0, 0, 0)
	target.Camp = Own
	state := &State{
		W: 5, H: 1, Units: []*Unit{actor, blocker, target},
		NativeCompositionEventBytes: make([]byte, 5),
		NativeTerrainMoveCodes:      make([]byte, 5),
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil || plan.NativeModeFallback != 0 || plan.Target != nil {
		t.Fatalf("mode0 plan=%+v", plan)
	}
	moved := len(plan.Path) > 1
	recovery := plan.NativeFallbackIdleRecovery
	if plan.NativeIdleRecovery != nil {
		recovery = plan.NativeIdleRecovery
	}
	if moved || recovery == nil {
		t.Fatalf("被圍住的 mode 0 應該不動並帶 0x13FD4 回復：path=%v recovery=%+v", plan.Path, recovery)
	}
}

func mustNativeAIModeBlockedSearchFlags(t *testing.T, state *State, selector int) []byte {
	t.Helper()
	records, err := NativeAIScoringRecords(state.Units)
	if err != nil {
		t.Fatal(err)
	}
	flags, err := nativeAIModeBlockedSearchFlags(
		state.W, state.H, records, len(state.Units), selector, state.NativeCompositionEventBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	return flags
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
