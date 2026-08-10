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
		NativeTerrainControl:        []byte{0, 0, 0, 0x20},
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

func TestNextAIPlanMode5UsesRawEventCellAndFailsClosedWithoutRow(t *testing.T) {
	actor := nativeAIRuntimeUnit(0, 0, 1, 5)
	actor.NativeRecordByte3D = 1
	actor.HasNativeRecordByte3D = true
	state := &State{
		W: 2, H: 1, Units: []*Unit{actor},
		NativeCompositionEventBytes: []byte{0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0},
		NativeTerrainControl:        []byte{0, 0, 0, 0x20},
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
		NativeTerrainControl:  []byte{0, 0, 0, 0x20},
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
