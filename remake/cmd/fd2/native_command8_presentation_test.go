package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
)

func nativeCommand8PresentationTestPlan(t *testing.T) (*battle.NativeCommandDamagePlan, *battle.Unit, *battle.Unit) {
	t.Helper()
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[8] = battle.NativeCommandRecord{ID: 8, Damage: 440, Hit: 100, SelectionMode: 8, EffectMode: 0, MPCost: 24, TargetCode: 0}
	actor := &battle.Unit{Camp: battle.Own, X: 0, Y: 0, HP: 20, MP: 30, OnField: true}
	target := &battle.Unit{Camp: battle.Enemy, ClassID: 5, X: 1, Y: 0, HP: 103, OnField: true}
	state := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	plan, err := state.PlanNativeCommandDamage(actor, target, 8, map[int]int{5: 10}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Results) != 1 || plan.DamageStages != 16 {
		t.Fatalf("plan results=%d stages=%d", len(plan.Results), plan.DamageStages)
	}
	return plan, actor, target
}

func TestNativeCommand8PresentationPublishesOnlyAfterDraw(t *testing.T) {
	plan, actor, target := nativeCommand8PresentationTestPlan(t)
	callback := 0
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd8Presentation = &nativeCommand8PresentationJob{actor: actor, plan: plan, prelude: make([]*ebiten.Image, 1), actorBlack: make([]*ebiten.Image, 1), actorPulse: make([]*ebiten.Image, 1), actorSpecs: []battlepresent.NativeCommand0ActorFrame{{PublishMP: true}}, actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: target.HP, then: func(results []battle.NativeCommandDamageResult) { callback = len(results) }}
	for stage := 1; stage <= plan.DamageStages; stage++ {
		g.nativeCmd8Presentation.handler = append(g.nativeCmd8Presentation.handler, nativeCommand8HandlerFrame{hpStage: stage})
	}
	g.stepNativeCommand8Presentation()
	if actor.MP != plan.MPBefore || actor.Acted || target.HP != plan.Results[0].HPBefore {
		t.Fatal("未繪製的第 8 指令改變了戰鬥狀態")
	}
	for g.nativeCmd8Presentation != nil {
		g.nativeCmd8Presentation.drawn = true
		g.stepNativeCommand8Presentation()
	}
	if actor.MP != plan.MPAfter || !actor.Acted || target.HP != plan.Results[0].HPAfter || g.nativeRNGState != plan.RNGAfter || callback != 1 {
		t.Fatalf("第 8 指令交易未完成：mp=%d acted=%v hp=%d rng=%#x callback=%d", actor.MP, actor.Acted, target.HP, g.nativeRNGState, callback)
	}
}

func TestNativeCommand8PresentationRollsBackOutOfOrderStage(t *testing.T) {
	plan, actor, target := nativeCommand8PresentationTestPlan(t)
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd8Presentation = &nativeCommand8PresentationJob{actor: actor, plan: plan, phase: nativeCommand8Actor, actorBlack: make([]*ebiten.Image, 1), actorPulse: make([]*ebiten.Image, 1), actorSpecs: []battlepresent.NativeCommand0ActorFrame{{PublishMP: true}}, handler: []nativeCommand8HandlerFrame{{hpStage: 2}}, actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: target.HP, drawn: true}
	g.stepNativeCommand8Presentation()
	if actor.MP != plan.MPAfter {
		t.Fatalf("MP 標記未發布：%d", actor.MP)
	}
	g.nativeCmd8Presentation.drawn = true
	g.stepNativeCommand8Presentation()
	if g.nativeCmd8Presentation != nil || actor.MP != plan.MPBefore || actor.Acted || target.HP != plan.Results[0].HPBefore || g.nativeRNGState != plan.RNGBefore || g.loadErr == "" {
		t.Fatalf("失敗回復不完整：mp=%d acted=%v hp=%d rng=%#x err=%q", actor.MP, actor.Acted, target.HP, g.nativeRNGState, g.loadErr)
	}
}
