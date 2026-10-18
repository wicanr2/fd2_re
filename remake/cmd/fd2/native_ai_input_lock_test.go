package main

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestNativeAIPhaseRejectsMapConfirmReentry(t *testing.T) {
	u := &battle.Unit{Camp: battle.Own, OnField: true, HP: 20, MaxHP: 20}
	st := &battle.State{W: 2, H: 2, Units: []*battle.Unit{u}}
	g := &Game{st: st, m: &MapData{W: 2, H: 2}, aiBusy: true}
	before := *u
	g.confirm()
	if g.sel != nil || g.nativeMovePlan != nil || g.ring || !reflect.DeepEqual(before, *u) {
		t.Fatal("AI phase accepted a player selection or changed its actor")
	}
	g.aiBusy = false
	g.confirm()
	if g.sel != u {
		t.Fatal("player selection did not resume after AI completion")
	}
}
