package main

import (
	"image/color"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func nativeCommand9PlayerTestPlan(t *testing.T) (*battle.NativeCommandDamagePlan, *battle.Unit, *battle.Unit) {
	t.Helper()
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[9] = battle.NativeCommandRecord{ID: 9, Damage: 999, Hit: 50, SelectionMode: 3, EffectMode: 0, MPCost: 30, TargetCode: 0}
	actor := &battle.Unit{Camp: battle.Own, X: 0, Y: 0, HP: 20, MP: 40, OnField: true}
	target := &battle.Unit{Camp: battle.Enemy, ClassID: 5, X: 1, Y: 0, HP: 403, OnField: true}
	state := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	plan, err := state.PlanNativeCommandDamage(actor, target, 9, map[int]int{5: 10}, 3)
	if err != nil {
		t.Fatal(err)
	}
	return plan, actor, target
}

func TestNativeCommand9PlayerPublishesOnlyAtRecoveredBoundaries(t *testing.T) {
	plan, actor, target := nativeCommand9PlayerTestPlan(t)
	callback := 0
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd9Player = &nativeCommand9PlayerJob{
		actor: actor, target: target, plan: plan, palettes: make([]color.Palette, 1),
		effectFrames: make([][]byte, 1), resultFrames: make([][]byte, 1),
		baselineWork: make([]byte, indexedmap.NativeUnitPresentWorkSize), baselineVGA: make([]byte, indexedmap.NativeMapVGASize),
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: target.HP, rngBefore: plan.RNGBefore,
		then: func(results []battle.NativeCommandDamageResult) { callback = len(results) },
	}
	g.stepNativeCommand9PlayerPresentation()
	if actor.MP != plan.MPBefore || target.HP != plan.Results[0].HPBefore {
		t.Fatal("未 Draw 即發布交易")
	}
	g.nativeCmd9Player.drawn = true
	g.stepNativeCommand9PlayerPresentation()
	if actor.MP != plan.MPBefore || target.HP != plan.Results[0].HPBefore {
		t.Fatal("palette 後過早發布交易")
	}
	g.nativeCmd9Player.drawn = true
	g.stepNativeCommand9PlayerPresentation()
	if actor.MP != plan.MPAfter || target.HP != plan.Results[0].HPAfter || actor.Acted {
		t.Fatal("effect 邊界發布錯誤")
	}
	g.nativeCmd9Player.drawn = true
	g.stepNativeCommand9PlayerPresentation()
	g.nativeCmd9Player.hold = 0
	g.stepNativeCommand9PlayerPresentation()
	if g.nativeCmd9Player != nil || !actor.Acted || g.nativeRNGState != plan.RNGAfter || callback != 1 {
		t.Fatal("result hold 後未完成交易")
	}
}

func TestNativeCommand9PlayerRollsBackLatePublishFailure(t *testing.T) {
	plan, actor, target := nativeCommand9PlayerTestPlan(t)
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd9Player = &nativeCommand9PlayerJob{actor: actor, target: target, plan: plan, phase: nativeCommand9PlayerEffect,
		effectFrames: make([][]byte, 1), baselineWork: make([]byte, indexedmap.NativeUnitPresentWorkSize), baselineVGA: make([]byte, indexedmap.NativeMapVGASize),
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: target.HP, rngBefore: plan.RNGBefore, drawn: true}
	target.HP--
	g.stepNativeCommand9PlayerPresentation()
	if g.nativeCmd9Player != nil || actor.MP != plan.MPBefore || actor.Acted || target.HP != plan.Results[0].HPBefore || g.nativeRNGState != plan.RNGBefore || g.loadErr == "" {
		t.Fatal("晚期失敗未完整回復")
	}
}
