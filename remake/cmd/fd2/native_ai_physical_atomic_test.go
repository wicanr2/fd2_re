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
