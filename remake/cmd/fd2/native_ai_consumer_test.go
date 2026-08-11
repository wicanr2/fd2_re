package main

import (
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
