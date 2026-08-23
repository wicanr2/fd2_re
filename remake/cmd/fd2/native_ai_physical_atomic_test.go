package main

import (
	"math/rand"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestNativeAIMode11PhysicalMissingFIGANIFailsBeforeStateMutation(t *testing.T) {
	actor := &battle.Unit{
		Camp: battle.Enemy, OnField: true, X: 0, Y: 0, HP: 80, MaxHP: 80,
		AP: 40, DP: 10, BattleFig: 40,
	}
	target := &battle.Unit{
		Camp: battle.Own, OnField: true, X: 1, Y: 0, HP: 60, MaxHP: 60,
		AP: 20, DP: 5, BattleFig: 4,
	}
	g := &Game{
		st:  &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}},
		rng: rand.New(rand.NewSource(1)), aiBusy: true,
	}
	g.executeNativeAIMode11Physical(&battle.AIPlan{
		U: actor, Target: target, NativeActionKind: battle.NativeAIActionPhysical,
	}, nil)
	if target.HP != 60 || actor.Acted || g.atk != nil || g.aiBusy {
		t.Fatalf("missing FIGANI crossed state boundary actor=%#v target=%#v atk=%v busy=%v", actor, target, g.atk, g.aiBusy)
	}
	if g.loadErr == "" {
		t.Fatal("missing FIGANI did not report the fail-closed boundary")
	}
}

func TestNativeAICommand0MissingPresentationFailsBeforeStateMutation(t *testing.T) {
	actor := &battle.Unit{
		Camp: battle.Enemy, OnField: true, X: 0, Y: 0, HP: 80, MaxHP: 80, MP: 8,
		BattleFig: 40, HasBattleFig: true, NativeRecordByte6: 0, HasNativeRecordByte6: true,
	}
	target := &battle.Unit{
		Camp: battle.Own, OnField: true, X: 1, Y: 0, HP: 60, MaxHP: 60, ClassID: 5,
		BattleFig: 4, HasBattleFig: true, NativeRecordByte6: 1, HasNativeRecordByte6: true,
	}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[0] = battle.NativeCommandRecord{ID: 0, Damage: 50, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 2, TargetCode: 3}
	g := &Game{
		st: &battle.State{
			W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCommandBook: book,
			NativeCommandResistances: map[int]int{5: 10}, NativeCompositionEventBytes: make([]byte, 2),
		},
		rng: rand.New(rand.NewSource(1)), nativeRNGState: 1, aiBusy: true,
	}
	err := g.executeNativeAIAction(&battle.AIPlan{
		U: actor, Target: target, NativeActionKind: battle.NativeAIActionCommand, NativeCommandID: 0,
	})
	if err == nil {
		t.Fatal("enemy command0 accepted a missing indexed presentation")
	}
	if actor.MP != 8 || actor.Acted || target.HP != 60 || g.nativeCmd0Presentation != nil {
		t.Fatalf("missing command0 presentation crossed state boundary actor=%#v target=%#v job=%v", actor, target, g.nativeCmd0Presentation != nil)
	}
}
