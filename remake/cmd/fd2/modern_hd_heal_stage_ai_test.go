package main

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestModernHDHealStageAIEntryControl(t *testing.T) {
	set := itemHDSet(t)
	makeGame := func(hd bool) (*Game, *battle.Unit, *battle.Unit) {
		g, _, _ := player9StageGame(t, false, false)
		a, b := g.st.Units[4], g.st.Units[5]
		// The ordinary fixture's normalized player rows are not AI raw
		// records. Keep the two original enemy rows, as in modifier tests.
		g.st.Units = []*battle.Unit{a, b}
		for i, u := range []*battle.Unit{a, b} {
			u.Camp, u.OnField, u.Acted = battle.Enemy, true, false
			u.NativeRecordByte6, u.HasNativeRecordByte6 = 1, true
			u.MP, u.HP, u.MaxHP, u.NativeRecordWord42, u.HasNativeRecordWord42 = 100, 100, 100, 100, true
			u.SetMapPlacement(26+i, 16, 3)
			if err := u.MaterializeNativeMapPresentation(); err != nil {
				t.Fatal(err)
			}
		}
		b.HP = 20
		g.st.NativeCommandBook[13] = battle.NativeCommandRecord{ID: 13, Damage: 70, SelectionMode: 1, EffectMode: 1, MPCost: 3, TargetCode: 1}
		if hd {
			g.modernStoryPortraits = set
		}
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		return g, a, b
	}
	g, a, b := makeGame(true)
	control, ca, cb := makeGame(false)
	done, controlDone := 0, 0
	for i, game := range []*Game{g, control} {
		actor, target, count := a, b, &done
		if i == 1 {
			actor, target, count = ca, cb, &controlDone
		}
		plan := &battle.AIPlan{U: actor, Target: target, NativeActionKind: battle.NativeAIActionCommand, NativeCommandID: 13, NativeActionDestination: battle.Cell{X: actor.X, Y: actor.Y}}
		if err := game.executeNativeAIActionWithContinuation(plan, func() { *count++ }); err != nil {
			t.Fatal(err)
		}
	}
	if g.nativeHealPresentation == nil || g.nativeHealPresentation.hdFrames == nil || len(g.nativeHealPresentation.targets) == 0 {
		t.Fatal("AI heal did not enter full owner")
	}
	for _, f := range g.nativeHealPresentation.hdFrames.Front {
		if !f.TransientTerrain || len(f.Terrain) == 0 || !f.Terrain[0].Stage {
			t.Fatal("AI radial front lost HD stage")
		}
	}
	for step := 0; g.nativeHealPresentation != nil && step < 500; step++ {
		if control.nativeHealPresentation == nil || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
			t.Fatal("AI native control diverged", step)
		}
		g.nativeHealPresentation.drawn = true
		control.nativeHealPresentation.drawn = true
		g.stepNativeCommandHealPresentation()
		control.stepNativeCommandHealPresentation()
	}
	if g.nativeHealPresentation != nil || control.nativeHealPresentation != nil || done != 1 || controlDone != 1 || b.HP <= 20 || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
		t.Fatal("AI heal continuation/state differs", done, controlDone, b.HP)
	}
}
