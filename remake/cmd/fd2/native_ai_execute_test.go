package main

import (
	"math/rand"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestExecuteNativeAICommand0RejectsMissingPresentationBeforeMutation(t *testing.T) {
	actor := &battle.Unit{
		Camp: battle.Enemy, HP: 20, MaxHP: 20, MP: 4, MaxMP: 4,
		AP: 30, DP: 1, HIT: 100, EV: 0, ClassID: 1,
		X: 0, Y: 0, OnField: true, HasNativeRecordByte5: true,
	}
	target := &battle.Unit{
		Camp: battle.Own, HP: 20, MaxHP: 20, MP: 0, MaxMP: 0,
		AP: 1, DP: 1, HIT: 0, EV: 0, ClassID: 1,
		X: 1, Y: 0, OnField: true, HasNativeRecordByte5: true,
	}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[0] = battle.NativeCommandRecord{
		ID: 0, Damage: 10, Hit: 100, SelectionMode: 1,
		EffectMode: 0, MPCost: 2, TargetCode: 1,
	}
	g := &Game{
		st: &battle.State{
			W: 2, H: 1, Units: []*battle.Unit{actor, target},
			NativeCompositionEventBytes: []byte{0, 0},
			NativeCommandBook:           book,
			NativeCommandResistances:    map[int]int{1: 10},
		},
		rng: rand.New(rand.NewSource(1)),
	}
	plan := &battle.AIPlan{
		U: actor, Target: target, NativeActionKind: battle.NativeAIActionCommand,
		NativeCommandID: 0,
	}
	if err := g.executeNativeAIAction(plan); err == nil {
		t.Fatal("AI command0 accepted a missing formal presentation context")
	}
	if actor.Acted || actor.MP != 4 || target.HP != 20 {
		t.Fatalf("actor=%+v target=%+v", actor, target)
	}
}

func TestExecuteNativeAICommand13WaitsForIndexedPresentation(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	for i := range assets.LUTs {
		for p := range assets.LUTs[i] {
			assets.LUTs[i][p] = byte(p)
		}
	}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[13] = battle.NativeCommandRecord{
		ID: 13, Damage: 70, SelectionMode: 1, EffectMode: 0,
		MPCost: 3, TargetCode: 1,
	}
	actor := state.Units[0]
	actor.Camp, actor.OnField = battle.Enemy, true
	actor.HP, actor.MaxHP, actor.MP = 40, 100, 5
	actor.NativeRecordByte5, actor.HasNativeRecordByte5 = 0, true
	actor.NativeRecordByte6, actor.HasNativeRecordByte6 = 0, true
	actor.NativeRecordByte34, actor.HasNativeRecordByte34 = 0x81, true
	actor.NativeRecordByte35, actor.HasNativeRecordByte35 = 0, true
	actor.NativeRecordByte36, actor.HasNativeRecordByte36 = 0, true
	actor.NativeRecordWord42, actor.HasNativeRecordWord42 = 100, true
	actor.NativeRecordWord46, actor.HasNativeRecordWord46 = 5, true
	actor.NativeRecordRace, actor.HasNativeRecordRace = 0, true
	actor.NativeRecordClass, actor.HasNativeRecordClass = 0, true
	actor.NativeRecordByte8, actor.HasNativeRecordByte8 = 0, true
	actor.InventorySlots = []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	actor.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	state.NativeCommandBook = book
	state.NativeCompositionEventBytes = make([]byte, state.W*state.H)
	state.NativeTileBlitModes[0], field.NativeTileBlitModes[0] = 0, 0
	state.MaterializeNativeMapRangeMode(3)
	g := &Game{
		nativeMapAssets: assets, nativeMapDAC: append([]byte(nil), assets.PaletteDAC...),
		m: field, st: state, aiBusy: true, rng: rand.New(rand.NewSource(2)),
	}
	plan := &battle.AIPlan{
		U: actor, Target: actor, NativeActionKind: battle.NativeAIActionCommand,
		NativeCommandID: 13,
	}
	if err := g.executeNativeAIAction(plan); err == nil {
		t.Fatal("AI command13 accepted a missing indexed baseline")
	}
	if actor.MP != 5 || actor.HP != 40 || actor.Acted {
		t.Fatalf("failed AI presentation mutated actor=%#v", actor)
	}
	if err := g.composeNativeMapFrameAt(time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	effect := assets.FDOTHER6
	assets.FDOTHER6 = nil
	if err := g.executeNativeAIAction(plan); err == nil {
		t.Fatal("AI command13 accepted missing FDOTHER#6")
	}
	if actor.MP != 5 || actor.HP != 40 || actor.Acted {
		t.Fatalf("missing AI FDOTHER#6 mutated actor=%#v", actor)
	}
	assets.FDOTHER6 = effect
	if err := g.executeNativeAIAction(plan); err != nil {
		t.Fatal(err)
	}
	if g.nativeHealPresentation == nil || actor.MP != 5 || actor.HP != 40 || actor.Acted {
		t.Fatalf("AI command13 crossed presentation boundary: job=%v actor=%#v",
			g.nativeHealPresentation != nil, actor)
	}
	g.aiStep()
	if g.nativeHealPresentation == nil || actor.MP != 5 || actor.HP != 40 || actor.Acted {
		t.Fatalf("aiStep replanned during presentation: job=%v actor=%#v",
			g.nativeHealPresentation != nil, actor)
	}
	for step := 0; g.nativeHealPresentation != nil && step < 256; step++ {
		if phase := g.nativeHealPresentation.phase; phase == nativeCommandHealFrames || phase == nativeCommandHealEffectFrames || phase == nativeCommandHealMaskFrames || phase == nativeCommandHealDigitFrames {
			g.nativeHealPresentation.drawn = true
		}
		g.stepNativeCommandHealPresentation()
	}
	if g.nativeHealPresentation != nil || actor.MP != 2 || actor.HP != 100 || !actor.Acted {
		t.Fatalf("AI command13 did not finish after presentation: job=%v actor=%#v",
			g.nativeHealPresentation != nil, actor)
	}
}

func completeNativeAIExecutorUnit() *battle.Unit {
	return &battle.Unit{
		OnField: true, HP: 100, MaxHP: 100, MP: 10, MaxMP: 10,
		AP: 10, DP: 20, HIT: 30, EV: 40, MV: 5, Lv: 1,
		NativeMapPresentation:    battle.NativeMapPresentationState{},
		HasNativeMapPresentation: true,
		BattleFig:                1, HasBattleFig: true,
		NativeRecordByte8: 1, HasNativeRecordByte8: true,
		NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 1, HasNativeRecordClass: true,
		NativeRecordByte5: 0, HasNativeRecordByte5: true,
		NativeRecordByte6: 0, HasNativeRecordByte6: true,
		NativeRecordByte34: 0x81, HasNativeRecordByte34: true,
		NativeRecordByte35: 0, HasNativeRecordByte35: true,
		NativeRecordByte36: 0, HasNativeRecordByte36: true,
		NativeRecordWord42: 100, HasNativeRecordWord42: true,
		NativeRecordWord46: 10, HasNativeRecordWord46: true,
		InventorySlots:       []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags: []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
	}
}

func TestExecuteNativeAIActionConsumesVerifiedCommand17Transaction(t *testing.T) {
	actor := completeNativeAIExecutorUnit()
	actor.Camp, actor.MP, actor.AP = battle.Enemy, 10, 100
	actor.NativeMapPresentation.X, actor.NativeMapPresentation.Y = 0, 0
	actor.NativeRecordByte5, actor.NativeRecordByte6 = 0, 0
	actor.NativeTransient = [6]byte{}
	target := completeNativeAIExecutorUnit()
	target.Camp, target.AP = battle.Enemy, 200
	target.NativeMapPresentation.X, target.NativeMapPresentation.Y = 1, 0
	target.NativeRecordByte5, target.NativeRecordByte6 = 0, 0
	target.NativeTransient = [6]byte{}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[17] = battle.NativeCommandRecord{ID: 17, SelectionMode: 1, EffectMode: 1, MPCost: 99, TargetCode: 1}
	book[18] = battle.NativeCommandRecord{ID: 18, SelectionMode: 1, EffectMode: 0, MPCost: 4, TargetCode: 1}
	g := &Game{
		st: &battle.State{
			W: 2, H: 1, Units: []*battle.Unit{actor, target},
			NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book,
		},
		rng: rand.New(rand.NewSource(1)),
	}
	plan := &battle.AIPlan{
		U: actor, Target: target, NativeActionKind: battle.NativeAIActionCommand,
		NativeCommandID: 17,
	}
	if err := g.executeNativeAIAction(plan); err == nil {
		t.Fatal("enemy modifier accepted missing indexed map assets")
	}
	if actor.AP != 100 || target.AP != 200 || actor.MP != 10 || actor.Acted || g.nativeRNGState != 0 {
		t.Fatalf("failed presentation mutated actor=%#v target=%#v rng=%#x", actor, target, g.nativeRNGState)
	}
}

func TestExecuteNativeAIActionRejectsUnknownItemRoute(t *testing.T) {
	actor := &battle.Unit{Camp: battle.Enemy, OnField: true, X: 0, Y: 0}
	target := &battle.Unit{Camp: battle.Own, OnField: true, X: 1, Y: 0}
	g := &Game{st: &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}}}
	plan := &battle.AIPlan{
		U: actor, Target: target, NativeActionKind: battle.NativeAIActionItem,
		NativeItemSlot: 0, NativeItemID: 0xff,
	}
	if err := g.executeNativeAIAction(plan); err == nil {
		t.Fatal("unknown item route unexpectedly executed")
	}
	if actor.Acted {
		t.Fatal("failed item route consumed actor action")
	}
}
