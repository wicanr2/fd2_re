package main

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func nativeAIConsumerUnit(x, y int, selector, mode byte) *battle.Unit {
	return &battle.Unit{
		Camp:                     battle.Enemy,
		X:                        x,
		Y:                        y,
		OnField:                  true,
		HP:                       80,
		MaxHP:                    80,
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
		NativeMapPresentation:    battle.NativeMapPresentationState{X: byte(x), Y: byte(y)},
		HasNativeMapPresentation: true,
		InventorySlots:           []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags:     []int{0, 0, 0, 0, 0, 0, 0, 0},
	}
}

func nativeAIConsumerCostRows() [][]byte {
	rows := make([][]byte, battle.NativeMovementCostRowCount)
	for index := range rows {
		rows[index] = make([]byte, battle.NativeMovementCostRowSize)
		for cell := range rows[index] {
			rows[index][cell] = 1
		}
	}
	return rows
}

func nativeAIConsumerCommandBook() []battle.NativeCommandRecord {
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	// This is the same raw command tuple used by the battle-package mode-11
	// producer fixture.  It is intentionally kept local to the consumer test:
	// the executable layer must prove it can consume a verified book, not reach
	// into an unexported test helper or invent a normalized command.
	book[0] = battle.NativeCommandRecord{
		ID: 0, Damage: 50, Hit: 100, SelectionMode: 5, EffectMode: 1,
		MPCost: 2, TargetCode: 0,
	}
	return book
}

func nativeAIConsumerMode5Grid(w, h int, eventCell battle.Cell, eventID byte) []byte {
	grid := make([]byte, 4+4*w*h)
	grid[0], grid[2] = byte(w), byte(h)
	offset := 4 + 4*(eventCell.X+w*eventCell.Y)
	binary.LittleEndian.PutUint16(grid[offset:offset+2], 0)
	grid[offset+2] = eventID
	return grid
}

func TestAIStepConsumesVerifiedMode2PhysicalPlan(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 1, 2)
	actor.InventorySlots[0] = 1
	actor.NativeInventoryFlags[0] = 0x40
	target := nativeAIConsumerUnit(2, 0, 0, 0)
	target.Camp = battle.Own
	target.AP, target.DP = 1, 1
	state := &battle.State{
		W:                           3,
		H:                           1,
		Units:                       []*battle.Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
	}
	itemRows := make([]byte, 2*battle.NativeItemEffectRowSize)
	itemRows[battle.NativeItemEffectRowSize+0x0b] = 0
	itemRows[battle.NativeItemEffectRowSize+0x0c] = 1
	if err := state.BindNativeFutureItemRows(itemRows); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}

	frames := []*ebiten.Image{ebiten.NewImage(1, 1), ebiten.NewImage(1, 1), ebiten.NewImage(1, 1)}
	g := &Game{
		m:            &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st:           state,
		aiBusy:       true,
		rng:          rand.New(rand.NewSource(1)),
		figani:       map[int][]*ebiten.Image{3: frames, 4: frames},
		figaniDelays: map[int][]int{3: {1, 1, 1}, 4: {1, 1, 1}},
	}

	g.aiStep()
	if g.loadErr != "" {
		t.Fatalf("mode-2 aiStep rejected complete raw plan: %s", g.loadErr)
	}
	if g.walk == nil || g.walk.u != actor || len(g.walk.path) < 2 || g.walk.path[len(g.walk.path)-1] != (battle.Cell{X: 1, Y: 0}) {
		t.Fatalf("aiStep walk=%#v, want raw-selected destination (1,0)", g.walk)
	}

	for step := 0; g.walk != nil && step < 8; step++ {
		g.stepBattleWalk()
	}
	if g.loadErr != "" || g.walk != nil || g.atk == nil {
		t.Fatalf("mode-2 walk/attack owner walk=%v atk=%v err=%q", g.walk != nil, g.atk != nil, g.loadErr)
	}
	for step := 0; (g.aiBusy || g.atk != nil || g.nativeFieldEvent61 != nil) && step < 96; step++ {
		if err := g.Update(); err != nil {
			t.Fatalf("mode-2 Update returned %v", err)
		}
	}
	if g.loadErr != "" || g.aiBusy || g.atk != nil || g.nativeFieldEvent61 != nil || state.Turn != 1 {
		t.Fatalf("mode-2 attack completion ai=%v atk=%v event61=%v turn=%d err=%q", g.aiBusy, g.atk != nil, g.nativeFieldEvent61 != nil, state.Turn, g.loadErr)
	}
}

func TestAIStepStopsMode2WithoutMovementProvenance(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 0, 2)
	state := &battle.State{
		W:                           1,
		H:                           1,
		Units:                       []*battle.Unit{actor},
		NativeCompositionEventBytes: []byte{0},
		NativeTerrainMoveCodes:      []byte{0},
	}
	g := &Game{st: state, aiBusy: true}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || actor.Acted {
		t.Fatalf("incomplete mode-2 AI was consumed: err=%q ai=%v acted=%v", g.loadErr, g.aiBusy, actor.Acted)
	}
}

func TestAIStepMode2NoCandidateCompletesRejected13FD4Tail(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 0, 2)
	actor.HP = 20 // 原始 +0x40 等於此 fixture 已固定的原始 +0x42 值。
	actor.InventorySlots[0] = 1
	actor.NativeInventoryFlags[0] = 0x40
	state := &battle.State{
		W: 1, H: 1, Units: []*battle.Unit{actor},
		NativeCompositionEventBytes: []byte{0}, NativeTerrainMoveCodes: []byte{0},
	}
	itemRows := make([]byte, 2*battle.NativeItemEffectRowSize)
	itemRows[battle.NativeItemEffectRowSize+0x0c] = 1
	if err := state.BindNativeFutureItemRows(itemRows); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}
	g := &Game{st: state, aiBusy: true}
	g.aiStep()
	if g.loadErr != "" || !actor.Acted {
		t.Fatalf("mode-2 rejected 0x13fd4 tail err=%q acted=%v", g.loadErr, actor.Acted)
	}
}

func TestAIStepMode2NoCandidateRecoveryFailsClosedWithoutPresentation(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 0, 2)
	actor.HP = 10
	actor.InventorySlots[0] = 1
	actor.NativeInventoryFlags[0] = 0x40
	state := &battle.State{
		W: 1, H: 1, Units: []*battle.Unit{actor},
		NativeCompositionEventBytes: []byte{0}, NativeTerrainMoveCodes: []byte{0},
	}
	itemRows := make([]byte, 2*battle.NativeItemEffectRowSize)
	itemRows[battle.NativeItemEffectRowSize+0x0c] = 1
	if err := state.BindNativeFutureItemRows(itemRows); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}
	g := &Game{st: state, aiBusy: true}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || actor.Acted || actor.HP != 10 {
		t.Fatalf("mode-2 recovery failure err=%q ai=%v acted=%v hp=%d", g.loadErr, g.aiBusy, actor.Acted, actor.HP)
	}
}

func TestAIStep14EF0CommandRejectsMissingPresentationWhileStationary(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 1, 2)
	actor.NativeCommandMask[0] = 1
	actor.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	target := nativeAIConsumerUnit(2, 0, 0, 0)
	target.Camp = battle.Enemy
	target.ClassID = 0
	target.AP, target.DP = 1, 1
	commandBook := nativeAIConsumerCommandBook()
	state := &battle.State{
		W:                           3,
		H:                           1,
		Units:                       []*battle.Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
		NativeCommandBook:           commandBook,
		NativeCommandResistances:    map[int]int{0: 10},
	}
	if err := state.BindNativeFutureItemRows(make([]byte, 2*battle.NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st:     state,
		aiBusy: true,
		rng:    rand.New(rand.NewSource(1)),
	}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || g.walk != nil || g.atk != nil || state.Turn != 0 {
		t.Fatalf("0x14ef0 stationary command did not stop atomically: ai=%v walk=%v atk=%v turn=%d err=%q", g.aiBusy, g.walk != nil, g.atk != nil, state.Turn, g.loadErr)
	}
	if actor.X != 0 || actor.Y != 0 || actor.MP != 4 || target.HP != target.MaxHP || actor.Acted {
		t.Fatalf("0x14ef0 missing presentation mutated command transaction: actor=(%d,%d) mp=%d acted=%v targetHP=%d/%d", actor.X, actor.Y, actor.MP, actor.Acted, target.HP, target.MaxHP)
	}
}

func TestAIStepStops14EF0ItemRouteWithoutIndexedPresentation(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 1, 2)
	actor.InventorySlots[0] = 192 // raw row 192: type-5 route, 0x211A4
	actor.NativeInventoryFlags = []int{0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	target := nativeAIConsumerUnit(1, 0, 1, 0)
	target.Camp = battle.Own
	target.HP = 1
	target.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	state := &battle.State{
		W:                           2,
		H:                           1,
		Units:                       []*battle.Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0},
		NativeTileBlitModes:         make([]byte, 2),
		NativeCommandBook:           nativeAIConsumerCommandBook(),
		NativeCommandResistances:    map[int]int{0: 10},
	}
	// Use the checked-in raw table rather than replacing item 192 with a
	// synthetic row.  This keeps the game-layer proof tied to the recovered
	// type-5/0x211A4 asset record and its target tuple.
	itemRows, err := battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeFutureItemRows(itemRows); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		m:      &MapData{W: 2, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0}},
		st:     state,
		aiBusy: true,
		rng:    rand.New(rand.NewSource(1)),
	}

	beforeItem, beforeHP := actor.InventorySlots[0], target.HP
	g.aiStep()
	for step := 0; step < 96 && g.loadErr == "" && (g.aiBusy || g.walk != nil || g.atk != nil); step++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.loadErr == "" || g.aiBusy || g.walk != nil || g.atk != nil || state.Turn != 0 {
		t.Fatalf("0x14ef0 missing indexed owner did not stop: ai=%v walk=%v atk=%v turn=%d err=%q", g.aiBusy, g.walk != nil, g.atk != nil, state.Turn, g.loadErr)
	}
	if actor.InventorySlots[0] != beforeItem || actor.NativeInventoryFlags[0]&0x80 != 0 ||
		target.HP != beforeHP || actor.Acted {
		t.Fatalf("missing indexed owner mutated transaction: slot=%d flags=%#x targetHP=%d acted=%v", actor.InventorySlots[0], actor.NativeInventoryFlags[0], target.HP, actor.Acted)
	}
}

func TestAIStepStops14EF0ItemRouteWithoutItemRows(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 1, 2)
	actor.InventorySlots[0] = 192
	actor.NativeInventoryFlags = []int{0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	target := nativeAIConsumerUnit(1, 0, 0, 0)
	target.Camp = battle.Own
	target.HP = 1
	target.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	state := &battle.State{
		W:                           2,
		H:                           1,
		Units:                       []*battle.Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0},
		NativeCommandBook:           nativeAIConsumerCommandBook(),
	}
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}
	g := &Game{st: state, aiBusy: true}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || actor.InventorySlots[0] != 192 || target.HP != 1 || actor.Acted {
		t.Fatalf("incomplete 0x14ef0 item route was consumed: err=%q ai=%v slot=%d targetHP=%d acted=%v", g.loadErr, g.aiBusy, actor.InventorySlots[0], target.HP, actor.Acted)
	}
}

func TestAIStepConsumesVerifiedMode5EventPlan(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	actor := nativeAIConsumerUnit(0, 0, 1, 5)
	actor.NativeRecordByte3D = 1
	actor.HasNativeRecordByte3D = true
	actor.NativeRecordDeathEffect = [3]byte{0xff, 0xff, 0xff}
	actor.HasNativeRecordDeathEffect = true
	state := &battle.State{
		W:                           3,
		H:                           1,
		Units:                       []*battle.Unit{actor},
		NativeEventState:            [0x20]byte{},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
		NativeTerrainControl:        []byte{0, 0, 0, 0x20},
		NativeMapEventGrid:          nativeAIConsumerMode5Grid(3, 1, battle.Cell{X: 2, Y: 0}, 1),
		HasNativeMapEventGrid:       true,
		NativeFieldControlRaw:       make([]byte, 0x56+3),
		HasNativeFieldControlState:  true,
	}
	state.NativeFieldControlRaw[0x56] = 1
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}

	g := &Game{m: &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}}, st: state, aiBusy: true}
	g.aiStep()
	if g.loadErr != "" || g.walk == nil {
		t.Fatalf("mode-5 event plan did not start: walk=%v err=%q", g.walk != nil, g.loadErr)
	}
	for step := 0; step < 96 && (g.aiBusy || g.walk != nil); step++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.loadErr != "" || g.aiBusy || g.walk != nil || state.Turn != 1 {
		t.Fatalf("mode-5 event completion ai=%v walk=%v turn=%d err=%q", g.aiBusy, g.walk != nil, state.Turn, g.loadErr)
	}
	if actor.X != 2 || actor.Y != 0 || state.NativeEventState[1] != 1 || actor.NativeRecordByte34 != 7 {
		t.Fatalf("mode-5 event owner did not commit raw tail: pos=(%d,%d) state=%d mode=%d", actor.X, actor.Y, state.NativeEventState[1], actor.NativeRecordByte34)
	}
}

func TestAIStepStopsMode5WithoutMovementProvenance(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 1, 5)
	actor.NativeRecordByte3D = 1
	actor.HasNativeRecordByte3D = true
	actor.NativeRecordDeathEffect = [3]byte{0xff, 0xff, 0xff}
	actor.HasNativeRecordDeathEffect = true
	state := &battle.State{
		W:                           3,
		H:                           1,
		Units:                       []*battle.Unit{actor},
		NativeEventState:            [0x20]byte{},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
		NativeTerrainControl:        []byte{0, 0, 0, 0x20},
		NativeMapEventGrid:          nativeAIConsumerMode5Grid(3, 1, battle.Cell{X: 2, Y: 0}, 1),
		HasNativeMapEventGrid:       true,
		NativeFieldControlRaw:       make([]byte, 0x56+3),
		HasNativeFieldControlState:  true,
	}
	state.NativeFieldControlRaw[0x56] = 1
	beforeGrid := append([]byte(nil), state.NativeMapEventGrid...)
	g := &Game{
		m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st:     state,
		aiBusy: true,
	}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || g.walk != nil || g.atk != nil || actor.Acted {
		t.Fatalf("incomplete mode-5 AI was consumed: err=%q ai=%v walk=%v atk=%v acted=%v", g.loadErr, g.aiBusy, g.walk != nil, g.atk != nil, actor.Acted)
	}
	if actor.X != 0 || actor.Y != 0 || state.Turn != 0 || state.NativeEventState[1] != 0 || actor.NativeRecordByte34 != 5 || state.HasNativeMapRangeModeState {
		t.Fatalf("mode-5 failure partially changed runtime: pos=(%d,%d) turn=%d state=%d mode=%d range=%v", actor.X, actor.Y, state.Turn, state.NativeEventState[1], actor.NativeRecordByte34, state.HasNativeMapRangeModeState)
	}
	for i := range beforeGrid {
		if state.NativeMapEventGrid[i] != beforeGrid[i] {
			t.Fatalf("mode-5 failure changed event grid at %d: got=%d want=%d", i, state.NativeMapEventGrid[i], beforeGrid[i])
		}
	}
}

func TestAIStepConsumesVerifiedMode7DestinationPlan(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 1, 7)
	actor.NativeRecordByte35 = 1
	actor.HasNativeRecordByte35 = true
	actor.NativeRecordByte36 = 0
	actor.HasNativeRecordByte36 = true
	state := &battle.State{
		W:                           3,
		H:                           1,
		Units:                       []*battle.Unit{actor},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
		NativeTerrainControl:        []byte{0, 0, 0, 0x20},
	}
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st:     state,
		aiBusy: true,
	}
	g.aiStep()
	if g.loadErr != "" || g.walk == nil || g.walk.u != actor || g.atk != nil {
		t.Fatalf("mode-7 destination plan did not start as movement-only: walk=%v atk=%v err=%q", g.walk != nil, g.atk != nil, g.loadErr)
	}
	for step := 0; step < 96 && (g.aiBusy || g.walk != nil); step++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.loadErr != "" || g.aiBusy || g.walk != nil || g.atk != nil || state.Turn != 1 {
		t.Fatalf("mode-7 movement completion ai=%v walk=%v atk=%v turn=%d err=%q", g.aiBusy, g.walk != nil, g.atk != nil, state.Turn, g.loadErr)
	}
	if actor.X != 1 || actor.Y != 0 || actor.NativeRecordByte5 != 1 || !actor.HasNativeRecordByte5 ||
		!state.HasNativeMapRangeModeState || state.NativeMapRangeMode != 0 {
		t.Fatalf("mode-7 raw owner did not commit: pos=(%d,%d) byte5=%d range=%d/%v", actor.X, actor.Y, actor.NativeRecordByte5, state.NativeMapRangeMode, state.HasNativeMapRangeModeState)
	}
}

func TestAIStepStopsMode7WithoutMovementProvenance(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 1, 7)
	actor.NativeRecordByte35 = 1
	actor.HasNativeRecordByte35 = true
	actor.NativeRecordByte36 = 0
	actor.HasNativeRecordByte36 = true
	state := &battle.State{
		W:                           3,
		H:                           1,
		Units:                       []*battle.Unit{actor},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
		NativeTerrainControl:        []byte{0, 0, 0, 0x20},
	}
	g := &Game{
		m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st:     state,
		aiBusy: true,
	}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || g.walk != nil || g.atk != nil || actor.Acted {
		t.Fatalf("incomplete mode-7 AI was consumed: err=%q ai=%v walk=%v atk=%v acted=%v", g.loadErr, g.aiBusy, g.walk != nil, g.atk != nil, actor.Acted)
	}
	if actor.X != 0 || actor.Y != 0 || actor.NativeRecordByte5 != 0 || state.Turn != 0 || state.HasNativeMapRangeModeState {
		t.Fatalf("mode-7 failure partially changed runtime: pos=(%d,%d) byte5=%d turn=%d range=%v", actor.X, actor.Y, actor.NativeRecordByte5, state.Turn, state.HasNativeMapRangeModeState)
	}
}

func TestAIStepConsumesTwoVerifiedMode7ActorsBeforeFinishingTurn(t *testing.T) {
	first := nativeAIConsumerUnit(0, 0, 1, 7)
	first.NativeRecordByte35, first.NativeRecordByte36 = 1, 0
	first.HasNativeRecordByte35, first.HasNativeRecordByte36 = true, true
	second := nativeAIConsumerUnit(2, 0, 1, 7)
	second.NativeRecordByte35, second.NativeRecordByte36 = 3, 0
	second.HasNativeRecordByte35, second.HasNativeRecordByte36 = true, true
	state := &battle.State{
		W: 4, H: 1, Units: []*battle.Unit{first, second},
		NativeCompositionEventBytes: []byte{0, 0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0, 0},
		NativeTerrainControl:        []byte{0, 0, 0, 0x20},
	}
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		m:      &MapData{W: 4, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0, 0}},
		st:     state,
		aiBusy: true,
	}
	g.aiStep()
	if g.loadErr != "" || g.walk == nil || g.walk.u != first || g.atk != nil {
		t.Fatalf("first mode-7 actor did not start: walk=%v actor=%p atk=%v err=%q", g.walk != nil, func() *battle.Unit {
			if g.walk == nil {
				return nil
			}
			return g.walk.u
		}(), g.atk != nil, g.loadErr)
	}
	firstCommitted := false
	for step := 0; step < 96 && (g.aiBusy || g.walk != nil); step++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if first.NativeRecordByte5 == 1 && second.NativeRecordByte5 == 0 {
			firstCommitted = true
		}
	}
	if g.loadErr != "" || g.aiBusy || g.walk != nil || g.atk != nil || state.Turn != 1 {
		t.Fatalf("two mode-7 actors did not finish one enemy turn: ai=%v walk=%v atk=%v turn=%d err=%q", g.aiBusy, g.walk != nil, g.atk != nil, state.Turn, g.loadErr)
	}
	if !firstCommitted || first.X != 1 || first.Y != 0 || second.X != 3 || second.Y != 0 ||
		first.NativeRecordByte5 != 1 || second.NativeRecordByte5 != 1 ||
		!state.HasNativeMapRangeModeState || state.NativeMapRangeMode != 0 {
		t.Fatalf("two mode-7 raw owners committed unexpected state: first=(%d,%d,%d) second=(%d,%d,%d) range=%d/%v", first.X, first.Y, first.NativeRecordByte5, second.X, second.Y, second.NativeRecordByte5, state.NativeMapRangeMode, state.HasNativeMapRangeModeState)
	}
}

func TestAIStepStopsTwoMode7ActorsWithoutMovementProvenance(t *testing.T) {
	first := nativeAIConsumerUnit(0, 0, 1, 7)
	first.NativeRecordByte35, first.NativeRecordByte36 = 1, 0
	first.HasNativeRecordByte35, first.HasNativeRecordByte36 = true, true
	second := nativeAIConsumerUnit(2, 0, 1, 7)
	second.NativeRecordByte35, second.NativeRecordByte36 = 3, 0
	second.HasNativeRecordByte35, second.HasNativeRecordByte36 = true, true
	state := &battle.State{
		W: 4, H: 1, Units: []*battle.Unit{first, second},
		NativeCompositionEventBytes: []byte{0, 0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0, 0},
		NativeTerrainControl:        []byte{0, 0, 0, 0x20},
	}
	g := &Game{
		m:      &MapData{W: 4, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0, 0}},
		st:     state,
		aiBusy: true,
	}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || g.walk != nil || g.atk != nil || first.Acted || second.Acted {
		t.Fatalf("incomplete two-actor mode-7 AI was consumed: err=%q ai=%v walk=%v atk=%v acted=%v/%v", g.loadErr, g.aiBusy, g.walk != nil, g.atk != nil, first.Acted, second.Acted)
	}
	if first.X != 0 || first.Y != 0 || second.X != 2 || second.Y != 0 ||
		first.NativeRecordByte5 != 0 || second.NativeRecordByte5 != 0 || state.Turn != 0 || state.HasNativeMapRangeModeState {
		t.Fatalf("incomplete two-actor mode-7 AI partially changed runtime: first=(%d,%d,%d) second=(%d,%d,%d) turn=%d range=%v", first.X, first.Y, first.NativeRecordByte5, second.X, second.Y, second.NativeRecordByte5, state.Turn, state.HasNativeMapRangeModeState)
	}
}

func nativeAIConsumerModeTargetState(mode byte) (*battle.State, *battle.Unit, *battle.Unit) {
	actor := nativeAIConsumerUnit(0, 0, 1, mode)
	actor.NativeRecordByte35 = 7
	actor.HasNativeRecordByte35 = true
	target := nativeAIConsumerUnit(2, 0, 0, 0)
	target.Camp = battle.Own
	target.NativeRecordByte8 = 7
	target.HasNativeRecordByte8 = true
	state := &battle.State{
		W:                           3,
		H:                           1,
		Units:                       []*battle.Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
	}
	return state, actor, target
}

func TestAIStepConsumesVerifiedMode3AndMode9RawTargetPlans(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mode      byte
		wantRange bool
	}{
		{name: "mode3", mode: 3, wantRange: true},
		{name: "mode9", mode: 9, wantRange: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, actor, target := nativeAIConsumerModeTargetState(tc.mode)
			if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
				t.Fatal(err)
			}
			g := &Game{
				m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
				st:     state,
				aiBusy: true,
			}
			g.aiStep()
			if g.loadErr != "" || g.walk == nil || g.walk.u != actor || g.atk != nil || len(g.walk.path) < 2 ||
				g.walk.path[len(g.walk.path)-1] != (battle.Cell{X: 1, Y: 0}) {
				t.Fatalf("mode-%d raw target plan did not start movement-only: walk=%v atk=%v path=%v err=%q", tc.mode, g.walk != nil, g.atk != nil, func() []battle.Cell {
					if g.walk == nil {
						return nil
					}
					return g.walk.path
				}(), g.loadErr)
			}
			for step := 0; step < 96 && (g.aiBusy || g.walk != nil); step++ {
				if err := g.Update(); err != nil {
					t.Fatal(err)
				}
			}
			if g.loadErr != "" || g.aiBusy || g.walk != nil || g.atk != nil || state.Turn != 1 {
				t.Fatalf("mode-%d movement completion ai=%v walk=%v atk=%v turn=%d err=%q", tc.mode, g.aiBusy, g.walk != nil, g.atk != nil, state.Turn, g.loadErr)
			}
			if actor.X != 1 || actor.Y != 0 || actor.NativeRecordByte5 != 0 || target.X != 2 || target.Y != 0 ||
				state.HasNativeMapRangeModeState != tc.wantRange || (tc.wantRange && state.NativeMapRangeMode != 0) {
				t.Fatalf("mode-%d raw target owner changed unexpected state: actor=(%d,%d) byte5=%d target=(%d,%d) range=%d/%v", tc.mode, actor.X, actor.Y, actor.NativeRecordByte5, target.X, target.Y, state.NativeMapRangeMode, state.HasNativeMapRangeModeState)
			}
		})
	}
}

func TestAIStepStopsMode3AndMode9WithoutMovementProvenance(t *testing.T) {
	for _, mode := range []byte{3, 9} {
		t.Run(fmt.Sprintf("mode%d", mode), func(t *testing.T) {
			state, actor, _ := nativeAIConsumerModeTargetState(mode)
			g := &Game{
				m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
				st:     state,
				aiBusy: true,
			}
			g.aiStep()
			if g.loadErr == "" || g.aiBusy || g.walk != nil || g.atk != nil || actor.Acted {
				t.Fatalf("incomplete mode-%d AI was consumed: err=%q ai=%v walk=%v atk=%v acted=%v", mode, g.loadErr, g.aiBusy, g.walk != nil, g.atk != nil, actor.Acted)
			}
			if actor.X != 0 || actor.Y != 0 || state.Turn != 0 || state.HasNativeMapRangeModeState {
				t.Fatalf("mode-%d failure partially changed runtime: pos=(%d,%d) turn=%d range=%v", mode, actor.X, actor.Y, state.Turn, state.HasNativeMapRangeModeState)
			}
		})
	}
}

func TestAIStepConsumesVerifiedMode4AndMode10DestinationPlans(t *testing.T) {
	for _, mode := range []byte{4, 10} {
		t.Run(fmt.Sprintf("mode%d", mode), func(t *testing.T) {
			actor := nativeAIConsumerUnit(0, 0, 1, mode)
			actor.NativeRecordByte35 = 1
			actor.HasNativeRecordByte35 = true
			actor.NativeRecordByte36 = 0
			actor.HasNativeRecordByte36 = true
			state := &battle.State{
				W:                           3,
				H:                           1,
				Units:                       []*battle.Unit{actor},
				NativeCompositionEventBytes: []byte{0, 0, 0},
				NativeTerrainMoveCodes:      []byte{0, 0, 0},
			}
			if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
				t.Fatal(err)
			}
			g := &Game{
				m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
				st:     state,
				aiBusy: true,
			}
			g.aiStep()
			if g.loadErr != "" || g.walk == nil || g.walk.u != actor || g.atk != nil || len(g.walk.path) < 2 ||
				g.walk.path[len(g.walk.path)-1] != (battle.Cell{X: 1, Y: 0}) {
				t.Fatalf("mode-%d destination plan did not start movement-only: walk=%v atk=%v path=%v err=%q", mode, g.walk != nil, g.atk != nil, func() []battle.Cell {
					if g.walk == nil {
						return nil
					}
					return g.walk.path
				}(), g.loadErr)
			}
			for step := 0; step < 96 && (g.aiBusy || g.walk != nil); step++ {
				if err := g.Update(); err != nil {
					t.Fatal(err)
				}
			}
			if g.loadErr != "" || g.aiBusy || g.walk != nil || g.atk != nil || state.Turn != 1 {
				t.Fatalf("mode-%d movement completion ai=%v walk=%v atk=%v turn=%d err=%q", mode, g.aiBusy, g.walk != nil, g.atk != nil, state.Turn, g.loadErr)
			}
			if actor.X != 1 || actor.Y != 0 || actor.NativeRecordByte5 != 0 ||
				!state.HasNativeMapRangeModeState || state.NativeMapRangeMode != 0 {
				t.Fatalf("mode-%d raw owner did not commit: pos=(%d,%d) byte5=%d range=%d/%v", mode, actor.X, actor.Y, actor.NativeRecordByte5, state.NativeMapRangeMode, state.HasNativeMapRangeModeState)
			}
		})
	}
}

func TestAIStepStopsMode4AndMode10WithoutMovementProvenance(t *testing.T) {
	for _, mode := range []byte{4, 10} {
		t.Run(fmt.Sprintf("mode%d", mode), func(t *testing.T) {
			actor := nativeAIConsumerUnit(0, 0, 1, mode)
			actor.NativeRecordByte35 = 1
			actor.HasNativeRecordByte35 = true
			actor.NativeRecordByte36 = 0
			actor.HasNativeRecordByte36 = true
			state := &battle.State{
				W:                           3,
				H:                           1,
				Units:                       []*battle.Unit{actor},
				NativeCompositionEventBytes: []byte{0, 0, 0},
				NativeTerrainMoveCodes:      []byte{0, 0, 0},
			}
			g := &Game{
				m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
				st:     state,
				aiBusy: true,
			}
			g.aiStep()
			if g.loadErr == "" || g.aiBusy || g.walk != nil || g.atk != nil || actor.Acted {
				t.Fatalf("incomplete mode-%d AI was consumed: err=%q ai=%v walk=%v atk=%v acted=%v", mode, g.loadErr, g.aiBusy, g.walk != nil, g.atk != nil, actor.Acted)
			}
			if actor.X != 0 || actor.Y != 0 || state.Turn != 0 || state.HasNativeMapRangeModeState {
				t.Fatalf("mode-%d failure partially changed runtime: pos=(%d,%d) turn=%d range=%v", mode, actor.X, actor.Y, state.Turn, state.HasNativeMapRangeModeState)
			}
		})
	}
}

func TestAIStepConsumesVerifiedMode0NearestFallback(t *testing.T) {
	state, actor, target := nativeAIConsumerModeTargetState(0)
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st:     state,
		aiBusy: true,
	}
	g.aiStep()
	if g.loadErr != "" || g.walk == nil || g.walk.u != actor || g.atk != nil || len(g.walk.path) < 2 ||
		g.walk.path[len(g.walk.path)-1] != (battle.Cell{X: 1, Y: 0}) {
		t.Fatalf("mode-0 nearest fallback did not start movement-only: walk=%v atk=%v path=%v err=%q", g.walk != nil, g.atk != nil, func() []battle.Cell {
			if g.walk == nil {
				return nil
			}
			return g.walk.path
		}(), g.loadErr)
	}
	for step := 0; step < 96 && (g.aiBusy || g.walk != nil); step++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.loadErr != "" || g.aiBusy || g.walk != nil || g.atk != nil || state.Turn != 1 {
		t.Fatalf("mode-0 movement completion ai=%v walk=%v atk=%v turn=%d err=%q", g.aiBusy, g.walk != nil, g.atk != nil, state.Turn, g.loadErr)
	}
	if actor.X != 1 || actor.Y != 0 || target.X != 2 || target.Y != 0 || actor.NativeRecordByte5 != 0 || state.HasNativeMapRangeModeState {
		t.Fatalf("mode-0 raw fallback changed unexpected state: actor=(%d,%d) target=(%d,%d) byte5=%d range=%v", actor.X, actor.Y, target.X, target.Y, actor.NativeRecordByte5, state.HasNativeMapRangeModeState)
	}
}

func TestAIStepStopsMode0WithoutMovementProvenance(t *testing.T) {
	state, actor, _ := nativeAIConsumerModeTargetState(0)
	g := &Game{
		m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st:     state,
		aiBusy: true,
	}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || g.walk != nil || g.atk != nil || actor.Acted {
		t.Fatalf("incomplete mode-0 AI was consumed: err=%q ai=%v walk=%v atk=%v acted=%v", g.loadErr, g.aiBusy, g.walk != nil, g.atk != nil, actor.Acted)
	}
	if actor.X != 0 || actor.Y != 0 || state.Turn != 0 || state.HasNativeMapRangeModeState {
		t.Fatalf("mode-0 failure partially changed runtime: pos=(%d,%d) turn=%d range=%v", actor.X, actor.Y, state.Turn, state.HasNativeMapRangeModeState)
	}
}

func TestAIStepConsumesVerifiedMode1BlockedCoordinate(t *testing.T) {
	state, actor, target := nativeAIConsumerModeTargetState(1)
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st:     state,
		aiBusy: true,
	}
	g.aiStep()
	if g.loadErr != "" || g.walk == nil || g.walk.u != actor || g.atk != nil || len(g.walk.path) < 2 ||
		g.walk.path[len(g.walk.path)-1] != (battle.Cell{X: 1, Y: 0}) {
		t.Fatalf("mode-1 blocked-coordinate owner did not start movement-only: walk=%v atk=%v path=%v err=%q", g.walk != nil, g.atk != nil, func() []battle.Cell {
			if g.walk == nil {
				return nil
			}
			return g.walk.path
		}(), g.loadErr)
	}
	for step := 0; step < 96 && (g.aiBusy || g.walk != nil); step++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.loadErr != "" || g.aiBusy || g.walk != nil || g.atk != nil || state.Turn != 1 {
		t.Fatalf("mode-1 movement completion ai=%v walk=%v atk=%v turn=%d err=%q", g.aiBusy, g.walk != nil, g.atk != nil, state.Turn, g.loadErr)
	}
	if actor.X != 1 || actor.Y != 0 || target.X != 2 || target.Y != 0 || actor.NativeRecordByte5 != 0 || state.HasNativeMapRangeModeState {
		t.Fatalf("mode-1 raw blocked-coordinate owner changed unexpected state: actor=(%d,%d) target=(%d,%d) byte5=%d range=%v", actor.X, actor.Y, target.X, target.Y, actor.NativeRecordByte5, state.HasNativeMapRangeModeState)
	}
}

func TestAIStepStopsMode1WithoutMovementProvenance(t *testing.T) {
	state, actor, _ := nativeAIConsumerModeTargetState(1)
	g := &Game{
		m:      &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st:     state,
		aiBusy: true,
	}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || g.walk != nil || g.atk != nil || actor.Acted {
		t.Fatalf("incomplete mode-1 AI was consumed: err=%q ai=%v walk=%v atk=%v acted=%v", g.loadErr, g.aiBusy, g.walk != nil, g.atk != nil, actor.Acted)
	}
	if actor.X != 0 || actor.Y != 0 || state.Turn != 0 || state.HasNativeMapRangeModeState {
		t.Fatalf("mode-1 failure partially changed runtime: pos=(%d,%d) turn=%d range=%v", actor.X, actor.Y, state.Turn, state.HasNativeMapRangeModeState)
	}
}

func TestAIStepConsumesVerifiedMode8Completion(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 1, 8)
	state := &battle.State{W: 1, H: 1, Units: []*battle.Unit{actor}}
	g := &Game{st: state, aiBusy: true}
	g.aiStep()
	for step := 0; step < 4 && g.aiBusy; step++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.loadErr != "" || g.aiBusy || g.walk != nil || g.atk != nil || state.Turn != 1 {
		t.Fatalf("mode-8 completion failed: ai=%v walk=%v atk=%v turn=%d err=%q", g.aiBusy, g.walk != nil, g.atk != nil, state.Turn, g.loadErr)
	}
	if actor.X != 0 || actor.Y != 0 || actor.NativeRecordByte34 != 8 || state.HasNativeMapRangeModeState {
		t.Fatalf("mode-8 completion changed raw state: pos=(%d,%d) mode=%d range=%v", actor.X, actor.Y, actor.NativeRecordByte34, state.HasNativeMapRangeModeState)
	}
}

func TestAIStepMode11RejectsMissingCommandPresentationWhileStationary(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 1, 11)
	actor.NativeCommandMask[0] = 1
	actor.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	actor.MP = 255
	target := nativeAIConsumerUnit(2, 0, 0, 0)
	// The raw command target code is authoritative for this fixture.  Both
	// records are deliberately in the native enemy group so the command
	// consumer's Camp predicate and the producer's raw +6 predicate agree;
	// this test does not rename that group as an allegiance rule.
	target.Camp = battle.Enemy
	target.ClassID = 0
	target.AP, target.DP = 1, 1
	target.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	state := &battle.State{
		W:                           3,
		H:                           1,
		Units:                       []*battle.Unit{actor, target},
		NativeCompositionEventBytes: []byte{0, 0, 0},
		NativeTerrainMoveCodes:      []byte{0, 0, 0},
		NativeCommandBook:           nativeAIConsumerCommandBook(),
		NativeCommandResistances:    map[int]int{0: 10},
	}
	if err := state.BindNativeFutureItemRows(make([]byte, 2*battle.NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(nativeAIConsumerCostRows()); err != nil {
		t.Fatal(err)
	}

	frames := []*ebiten.Image{ebiten.NewImage(1, 1), ebiten.NewImage(1, 1), ebiten.NewImage(1, 1)}
	g := &Game{
		m:            &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st:           state,
		aiBusy:       true,
		rng:          rand.New(rand.NewSource(1)),
		figani:       map[int][]*ebiten.Image{3: frames, 4: frames},
		figaniDelays: map[int][]int{3: {1, 1, 1}, 4: {1, 1, 1}},
	}

	// 0x15311 owns command target/effect geometry, not an actor movement path.
	// Missing presentation therefore rejects the first stage in place.
	g.aiStep()
	if g.loadErr == "" {
		t.Fatal("mode-11 accepted command0 without its formal presentation context")
	}
	if g.aiBusy || g.walk != nil || g.atk != nil || state.Turn != 0 || actor.MP != 255 ||
		actor.Acted || target.HP != target.MaxHP {
		t.Fatalf("mode-11 missing presentation mutated transaction: ai=%v walk=%v atk=%v turn=%d mp=%d acted=%v targetHP=%d/%d",
			g.aiBusy, g.walk != nil, g.atk != nil, state.Turn, actor.MP, actor.Acted, target.HP, target.MaxHP)
	}
	if actor.X != 0 || actor.Y != 0 {
		t.Fatalf("mode-11 0x15311 moved actor to (%d,%d)", actor.X, actor.Y)
	}
}

func TestAIStepStopsMode11WithoutVerifiedProducerTables(t *testing.T) {
	actor := nativeAIConsumerUnit(0, 0, 1, 11)
	state := &battle.State{
		W:                           1,
		H:                           1,
		Units:                       []*battle.Unit{actor},
		NativeCompositionEventBytes: []byte{0},
		NativeTerrainMoveCodes:      []byte{0},
	}
	g := &Game{st: state, aiBusy: true}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || actor.Acted {
		t.Fatalf("incomplete mode-11 AI was consumed or normalized: err=%q ai=%v acted=%v", g.loadErr, g.aiBusy, actor.Acted)
	}
}
