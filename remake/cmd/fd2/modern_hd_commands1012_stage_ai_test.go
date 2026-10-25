package main

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestModernHDCommand1012StageAIEntryControl(t *testing.T) {
	set := itemHDSet(t)
	for _, id := range []int{11, 12} {
		makeGame := func(hd bool) (*Game, *battle.Unit, *battle.Unit) {
			g, _, _ := player9StageGame(t, false, true)
			a, b := g.st.Units[4], g.st.Units[0]
			for _, u := range g.st.Units {
				u.OnField = u == a || u == b
			}
			a.Camp, a.MP, a.Acted = battle.Enemy, 100, false
			b.Camp, b.HP, b.MaxHP = battle.Own, 10000, 10000
			// Match the established native AI command10 fixture's explicit
			// target class/resistance, rather than invent a class19 table.
			b.ClassID, b.NativeRecordClass, b.HasNativeRecordClass = 5, 5, true
			g.st.NativeCommandResistances = map[int]int{5: 10}
			b.NativeRecordWord42, b.HasNativeRecordWord42 = 10000, true
			a.SetMapPlacement(26, 16, 3)
			b.SetMapPlacement(27, 16, 0)
			if err := a.MaterializeNativeMapPresentation(); err != nil {
				t.Fatal(err)
			}
			if err := b.MaterializeNativeMapPresentation(); err != nil {
				t.Fatal(err)
			}
			g.st.NativeCommandBook[id] = battle.NativeCommandRecord{ID: id, Damage: 70, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 3, TargetCode: 1}
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
		done, cdone := 0, 0
		for i, game := range []*Game{g, control} {
			actor, target, count := a, b, &done
			if i == 1 {
				actor, target, count = ca, cb, &cdone
			}
			plan := &battle.AIPlan{U: actor, Target: target, NativeActionKind: battle.NativeAIActionCommand, NativeCommandID: id, NativeActionDestination: battle.Cell{X: actor.X, Y: actor.Y}}
			if err := game.executeNativeAIActionWithContinuation(plan, func() { *count++ }); err != nil {
				t.Fatal(id, err)
			}
		}
		if g.nativeCmd1012 == nil || g.native2189A == nil || g.native2189A.hdFrames == nil {
			t.Fatal("AI did not enter full prelude owner")
		}
		p := g.native2189A
		t.Logf("AI%d source boundary units_equal=%t hud_equal=%t", id, reflect.DeepEqual(p.hdScene.input.Frame.Units, p.hdSteady.input.Frame.Units), reflect.DeepEqual(p.hdScene.input.HUD, p.hdSteady.input.HUD))
		for step := 0; g.nativeCmd1012 != nil && step < 200; step++ {
			if control.nativeCmd1012 == nil || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
				t.Fatal("AI native control diverged", id, step)
			}
			for _, game := range []*Game{g, control} {
				if game.native2189A != nil {
					game.native2189A.drawn = true
					game.stepNative2189A()
				} else {
					game.nativeCmd1012.drawn = true
					game.stepNativeCommand1012Presentation()
				}
			}
		}
		if g.nativeCmd1012 != nil || control.nativeCmd1012 != nil || done != 1 || cdone != 1 || b.HP >= 10000 || b.HP <= 0 || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
			t.Fatal("AI complete state differs", id, done, cdone, b.HP, g.loadErr)
		}
	}
}
