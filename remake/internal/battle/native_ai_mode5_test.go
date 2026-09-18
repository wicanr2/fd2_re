package battle

import (
	"encoding/binary"
	"testing"
)

func nativeAIMode5Grid(w, h int, eventCell Cell, eventID byte) []byte {
	grid := make([]byte, 4+4*w*h)
	grid[0], grid[2] = byte(w), byte(h)
	offset := 4 + 4*(eventCell.X+w*eventCell.Y)
	binary.LittleEndian.PutUint16(grid[offset:offset+2], 0)
	grid[offset+2] = eventID
	resetNativeMapEventGrid(grid)
	return grid
}

func TestNativeAIMode5EventCellAndStateTailPreserveRawRows(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 5)
	actor.NativeRecordByte3D = 1
	actor.HasNativeRecordByte3D = true
	actor.NativeRecordByte34 = 0xa5
	actor.NativeRecordDeathEffect = [3]byte{0xff, 0xff, 0xff}
	actor.HasNativeRecordDeathEffect = true
	state := &State{
		W: 3, H: 1, Units: []*Unit{actor},
		NativeEventState:            [0x20]byte{},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
		NativeTerrainControl:        []byte{0x20, 0, 0, 0},
		NativeMapEventGrid:          nativeAIMode5Grid(3, 1, Cell{X: 2, Y: 0}, 1),
		HasNativeMapEventGrid:       true,
		NativeFieldControlRaw:       make([]byte, 0x56+3),
		HasNativeFieldControlState:  true,
	}
	rowOffset := 0x53 + 3*1
	state.NativeFieldControlRaw[rowOffset] = 0
	binary.LittleEndian.PutUint16(state.NativeFieldControlRaw[rowOffset+1:rowOffset+3], 7)
	if got, err := state.NativeAIMode5EventCell(1); err != nil || got != (Cell{X: 2, Y: 0}) {
		t.Fatalf("event cell=(%v,%v), want (2,0)", got, err)
	}
	actor.X = 2
	if err := state.ApplyNativeAIMode5Event(actor, 1, Cell{X: 2, Y: 0}); err != nil {
		t.Fatalf("ApplyNativeAIMode5Event() error = %v", err)
	}
	if state.NativeEventState[1] != 1 || actor.NativeRecordByte34 != 7 ||
		actor.NativeRecordDeathEffect != [3]byte{0, 7, 0} ||
		len(actor.Inventory) != 1 || actor.Inventory[0] != 7 ||
		actor.InventorySlots[0] != 7 {
		t.Fatalf("mode5 state mutation lost raw fields: state=%d mode=%d effect=%v inventory=%v slots=%v",
			state.NativeEventState[1], actor.NativeRecordByte34&0x0f,
			actor.NativeRecordDeathEffect, actor.Inventory, actor.InventorySlots)
	}
	if got := state.NativeMapEventGrid[4+4*2+2]; got != 0 {
		t.Fatalf("mode5 event grid low byte=%d, want cleared", got)
	}
}

func TestNativeAIMode5EventEmitsProvenRawAudioCueBeforeStateCompletion(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 5)
	actor.NativeRecordByte3D = 1
	actor.HasNativeRecordByte3D = true
	actor.NativeRecordDeathEffect = [3]byte{0xff, 0xff, 0xff}
	actor.HasNativeRecordDeathEffect = true
	state := &State{
		W: 1, H: 1, Units: []*Unit{actor},
		NativeEventState:      [0x20]byte{},
		NativeTerrainControl:  []byte{0x20, 0, 0, 0},
		NativeMapEventGrid:    nativeAIMode5Grid(1, 1, Cell{X: 0, Y: 0}, 1),
		HasNativeMapEventGrid: true,
		NativeFieldControlRaw: func() []byte {
			raw := make([]byte, 0x56+3)
			raw[0x56] = 1
			binary.LittleEndian.PutUint16(raw[0x57:0x59], 9)
			return raw
		}(),
		HasNativeFieldControlState: true,
	}
	var cues []NativeAIMode5AudioCue
	actor.X = 0
	if err := state.ApplyNativeAIMode5EventWithAudioCue(
		actor, 1, Cell{X: 0, Y: 0}, func(cue NativeAIMode5AudioCue) {
			cues = append(cues, cue)
			if state.NativeEventState[1] != 1 {
				t.Fatalf("audio cue emitted before raw event state=1: %d", state.NativeEventState[1])
			}
		},
	); err != nil {
		t.Fatalf("ApplyNativeAIMode5EventWithAudioCue() error = %v", err)
	}
	if len(cues) != 1 || cues[0] != (NativeAIMode5AudioCue{
		HandleLinearAddress: NativeAIMode5AudioHandle,
		ResourceID:          NativeAIMode5AudioResource,
		Index:               NativeAIMode5AudioIndex,
		LoopCount:           NativeAIMode5AudioLoopCount,
	}) {
		t.Fatalf("raw mode5 audio cues=%v", cues)
	}
	if actor.NativeRecordByte34 != 7 || state.NativeEventState[1] != 1 {
		t.Fatalf("mode5 state completion lost after audio cue: mode=%d state=%d", actor.NativeRecordByte34, state.NativeEventState[1])
	}
}

func TestNextAIPlanMode5UsesRawEventCellAndFailsClosedWithoutRow(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 5)
	actor.NativeRecordByte3D = 1
	actor.HasNativeRecordByte3D = true
	state := &State{
		W: 2, H: 1, Units: []*Unit{actor},
		NativeCompositionEventBytes: []byte{0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0},
		NativeTerrainControl:        []byte{0x20, 0, 0, 0},
		NativeMapEventGrid:          nativeAIMode5Grid(2, 1, Cell{X: 1, Y: 0}, 1),
		HasNativeMapEventGrid:       true,
	}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError == nil {
		t.Fatalf("mode5 missing field-control row plan=%+v, want fail-closed error", plan)
	}
	state.NativeFieldControlRaw = make([]byte, 0x56+3)
	state.NativeFieldControlRaw[0x56] = 0
	state.HasNativeFieldControlState = true
	plan = state.NextAIPlan()
	if plan == nil || plan.NativeError != nil || !plan.NativeModeEventActive ||
		plan.NativeModeEventID != 1 || plan.NativeModeEventDestination != (Cell{X: 1, Y: 0}) {
		t.Fatalf("mode5 raw event plan=%+v", plan)
	}
}

func TestNativeAIMode5RejectsMalformedLaterCellWithoutPartialMutation(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 5)
	actor.NativeRecordByte3D = 1
	actor.HasNativeRecordByte3D = true
	actor.NativeRecordDeathEffect = [3]byte{0xff, 0xff, 0xff}
	actor.HasNativeRecordDeathEffect = true
	grid := nativeAIMode5Grid(2, 1, Cell{X: 0, Y: 0}, 1)
	binary.LittleEndian.PutUint16(grid[8:10], 0x03ff) // later tile has no terrain row
	state := &State{
		W: 2, H: 1, Units: []*Unit{actor},
		NativeEventState:      [0x20]byte{},
		NativeTerrainControl:  []byte{0x20, 0, 0, 0},
		NativeMapEventGrid:    grid,
		HasNativeMapEventGrid: true,
		NativeFieldControlRaw: func() []byte {
			raw := make([]byte, 0x56+3)
			raw[0x56] = 0
			binary.LittleEndian.PutUint16(raw[0x57:0x59], 7)
			return raw
		}(),
		HasNativeFieldControlState: true,
	}
	beforeGrid := append([]byte(nil), state.NativeMapEventGrid...)
	beforeByte34 := actor.NativeRecordByte34
	if err := state.ApplyNativeAIMode5Event(actor, 1, Cell{X: 0, Y: 0}); err == nil {
		t.Fatal("malformed later event cell unexpectedly committed")
	}
	if state.NativeEventState[1] != 0 || actor.NativeRecordDeathEffect != [3]byte{0xff, 0xff, 0xff} ||
		len(actor.Inventory) != 0 || actor.NativeRecordByte34 != beforeByte34 {
		t.Fatalf("mode5 partial state mutation: state=%d effect=%v inventory=%v byte34=%d",
			state.NativeEventState[1], actor.NativeRecordDeathEffect, actor.Inventory, actor.NativeRecordByte34)
	}
	for i := range beforeGrid {
		if state.NativeMapEventGrid[i] != beforeGrid[i] {
			t.Fatalf("mode5 partial grid mutation at %d: got=%d want=%d", i, state.NativeMapEventGrid[i], beforeGrid[i])
		}
	}
}

func TestNativeMapDrawTilesFollowsTheMutableBuffer(t *testing.T) {
	// 0x12263 就地把事件格的 tile word +1；繪圖端讀的是同一份緩衝，所以撿走之後
	// 那一格要換成打開的箱子（第十一章 (7,20) 事件 5：原版 seq 5525 起箱子是開的）。
	actor := nativeAIRuntimeUnit(0, 0, 1, 5)
	actor.NativeRecordByte3D = 1
	actor.HasNativeRecordByte3D = true
	actor.NativeRecordDeathEffect = [3]byte{0xff, 0, 0}
	actor.HasNativeRecordDeathEffect = true
	state := &State{
		W: 2, H: 1, Units: []*Unit{actor},
		NativeCompositionEventBytes: []byte{0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0},
		NativeTerrainControl:        []byte{0x20, 0, 0, 0},
		NativeMapEventGrid:          nativeAIMode5Grid(2, 1, Cell{X: 1, Y: 0}, 1),
		HasNativeMapEventGrid:       true,
		HasNativeChestControlState:  true,
	}
	state.NativeChestControls[1] = NativeChestControl{RawType: 1, Value: 3000}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	before, ok := state.NativeMapDrawTiles()
	if !ok {
		t.Fatal("可變緩衝沒有材料化")
	}
	actor.SetMapPlacement(1, 0, 0)
	if err := state.ApplyNativeAIMode5Event(actor, 1, Cell{X: 1, Y: 0}); err != nil {
		t.Fatal(err)
	}
	after, ok := state.NativeMapDrawTiles()
	if !ok {
		t.Fatal("事件之後讀不到可變緩衝")
	}
	if after[1] != before[1]+1 || after[0] != before[0] {
		t.Fatalf("繪圖圖塊 %v → %v，事件格應該只 +1", before, after)
	}
	empty := &State{W: 2, H: 1}
	if _, ok := empty.NativeMapDrawTiles(); ok {
		t.Fatal("沒有可變緩衝時不該回報可用")
	}
}

func TestNativeAIMode5PickupBecomesExecutableDeathReward(t *testing.T) {
	// 0x13CCE／0x13CD6 的寶箱列寫進 +0x31..+0x33；重製端的掉落走 DeathEffect／
	// DeathReward，兩份要一致，撿到的金錢才會在被擊倒時進玩家金庫。
	actor := nativeAIRuntimeUnit(0, 0, 1, 5)
	actor.NativeRecordByte3D = 1
	actor.HasNativeRecordByte3D = true
	actor.NativeRecordDeathEffect = [3]byte{0xff, 0, 0}
	actor.HasNativeRecordDeathEffect = true
	state := &State{
		W: 2, H: 1, Units: []*Unit{actor},
		NativeCompositionEventBytes: []byte{0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0},
		NativeTerrainControl:        []byte{0x20, 0, 0, 0},
		NativeMapEventGrid:          nativeAIMode5Grid(2, 1, Cell{X: 1, Y: 0}, 1),
		HasNativeMapEventGrid:       true,
		HasNativeChestControlState:  true,
	}
	state.NativeChestControls[1] = NativeChestControl{RawType: 1, Value: 10000}
	if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
		t.Fatal(err)
	}
	actor.SetMapPlacement(1, 0, 0)
	if err := state.ApplyNativeAIMode5Event(actor, 1, Cell{X: 1, Y: 0}); err != nil {
		t.Fatal(err)
	}
	if actor.NativeRecordDeathEffect != [3]byte{1, 0x10, 0x27} {
		t.Fatalf("raw +0x31..+0x33=%v", actor.NativeRecordDeathEffect)
	}
	if actor.DeathReward == nil || actor.DeathReward.Type != 1 || actor.DeathReward.Value != 10000 ||
		actor.DeathEffect == nil || *actor.DeathEffect != *actor.DeathReward {
		t.Fatalf("typed 掉落沒有跟著寫：effect=%+v reward=%+v", actor.DeathEffect, actor.DeathReward)
	}
}

func TestNativeAIMode5FallsBackToModeZeroTailWhenEventUnavailable(t *testing.T) {
	// 0x13C42：[0x53AD5][+0x3D] 非零跳 0x13B05；0x13C59：0x15DF3 回 −1 也跳 0x13B05。
	// 兩種都不是失敗即關閉，是走 mode 0 的 0x14121／0x13E9C 尾段。
	build := func() (*State, *Unit) {
		actor := nativeAIRuntimeUnit(0, 0, 1, 5)
		actor.NativeRecordByte3D = 1
		actor.HasNativeRecordByte3D = true
		target := nativeAIRuntimeUnit(1, 0, 0, 0)
		target.Camp = Own
		state := &State{
			W: 2, H: 1, Units: []*Unit{actor, target},
			NativeCompositionEventBytes: []byte{0, 0},
			NativeTerrainMoveCodes:      []byte{0, 0},
			NativeTerrainControl:        []byte{0x20, 0, 0, 0},
			NativeMapEventGrid:          nativeAIMode5Grid(2, 1, Cell{X: 1, Y: 0}, 1),
			HasNativeMapEventGrid:       true,
			HasNativeChestControlState:  true,
		}
		if err := state.BindNativeMovementCostRows(nativeAIRuntimeCostRows()); err != nil {
			t.Fatal(err)
		}
		return state, actor
	}
	state, _ := build()
	state.NativeEventState[1] = 1
	plan := state.NextAIPlan()
	if plan == nil || plan.NativeError != nil {
		t.Fatalf("事件已觸發時的計畫=%+v", plan)
	}
	if plan.NativeModeEventActive || !plan.NativeModeFallbackActive || plan.NativeModeFallback != 5 {
		t.Fatalf("事件已觸發仍走事件分支：%+v", plan)
	}
	state, _ = build()
	state.NativeMapEventGrid = nativeAIMode5Grid(2, 1, Cell{X: 1, Y: 0}, 9) // 沒有 +0x3D=1 的格子
	plan = state.NextAIPlan()
	if plan == nil || plan.NativeError != nil {
		t.Fatalf("找不到事件格時的計畫=%+v", plan)
	}
	if plan.NativeModeEventActive {
		t.Fatalf("找不到事件格仍走事件分支：%+v", plan)
	}
}
