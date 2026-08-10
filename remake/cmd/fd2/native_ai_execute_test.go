package main

import (
	"math/rand"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestExecuteNativeAIActionReusesVerifiedCommandDamageRoute(t *testing.T) {
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
	if err := g.executeNativeAIAction(plan); err != nil {
		t.Fatal(err)
	}
	if !actor.Acted || actor.MP != 2 || target.HP >= 20 {
		t.Fatalf("actor=%+v target=%+v", actor, target)
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
