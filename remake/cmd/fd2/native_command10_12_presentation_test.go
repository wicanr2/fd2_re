package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func nativeCommand10TestPlan(t *testing.T) (*battle.NativeCommandDamagePlan, *battle.Unit, *battle.Unit) {
	t.Helper()
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[10] = battle.NativeCommandRecord{ID: 10, Damage: 999, Hit: 50, SelectionMode: 3, EffectMode: 0, MPCost: 30, TargetCode: 0}
	actor := &battle.Unit{Camp: battle.Own, X: 0, Y: 0, HP: 20, MP: 40, OnField: true}
	target := &battle.Unit{Camp: battle.Enemy, ClassID: 5, X: 1, Y: 0, HP: 403, OnField: true}
	state := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	plan, err := state.PlanNativeCommandDamage(actor, target, 10, map[int]int{5: 10}, 3)
	if err != nil {
		t.Fatal(err)
	}
	return plan, actor, target
}

func TestNativeCommand1012PublishesOnlyAtRecoveredBoundaries(t *testing.T) {
	plan, actor, target := nativeCommand10TestPlan(t)
	callback := 0
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd1012 = &nativeCommand1012Job{
		actor: actor, plan: plan, schedule: fdother.NativeCommand10To12Schedule{ResultHoldMS: 0},
		phase: nativeCommand1012Main, mainFrames: [][]byte{{0}}, resultFrames: [][]byte{{0}},
		baselineWork: make([]byte, indexedmap.NativeUnitPresentWorkSize), baselineVGA: make([]byte, indexedmap.NativeMapVGASize),
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: []int{target.HP}, rngBefore: plan.RNGBefore,
		then: func(results []battle.NativeCommandDamageResult) { callback = len(results) },
	}
	g.stepNativeCommand1012Presentation()
	if actor.MP != plan.MPBefore || target.HP != plan.Results[0].HPBefore {
		t.Fatal("未 Draw 即發布交易")
	}
	g.nativeCmd1012.drawn = true
	g.stepNativeCommand1012Presentation()
	if actor.MP != plan.MPAfter || target.HP != plan.Results[0].HPAfter || actor.Acted {
		t.Fatal("主演出邊界發布錯誤")
	}
	g.nativeCmd1012.drawn = true
	g.stepNativeCommand1012Presentation()
	g.stepNativeCommand1012Presentation()
	if g.nativeCmd1012 != nil || !actor.Acted || g.nativeRNGState != plan.RNGAfter || callback != 1 {
		t.Fatal("結果 hold 後未完成交易")
	}
}

func TestNativeCommand1012RollsBackLatePublishFailure(t *testing.T) {
	plan, actor, target := nativeCommand10TestPlan(t)
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd1012 = &nativeCommand1012Job{
		actor: actor, plan: plan, phase: nativeCommand1012Main, mainFrames: [][]byte{{0}},
		baselineWork: make([]byte, indexedmap.NativeUnitPresentWorkSize), baselineVGA: make([]byte, indexedmap.NativeMapVGASize),
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: []int{target.HP}, rngBefore: plan.RNGBefore, drawn: true,
	}
	target.HP--
	g.stepNativeCommand1012Presentation()
	if g.nativeCmd1012 != nil || actor.MP != plan.MPBefore || actor.Acted || target.HP != plan.Results[0].HPBefore || g.nativeRNGState != plan.RNGBefore || g.loadErr == "" {
		t.Fatal("晚期失敗未完整回復")
	}
}

func TestNativeCommand1012PreludeCannotDisappearSilently(t *testing.T) {
	plan, actor, target := nativeCommand10TestPlan(t)
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd1012 = &nativeCommand1012Job{
		actor: actor, plan: plan, phase: nativeCommand1012Prelude,
		baselineWork: make([]byte, indexedmap.NativeUnitPresentWorkSize), baselineVGA: make([]byte, indexedmap.NativeMapVGASize),
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: []int{target.HP}, rngBefore: plan.RNGBefore,
	}
	g.stepNativeCommand1012Presentation()
	if g.nativeCmd1012 != nil || actor.MP != plan.MPBefore || target.HP != plan.Results[0].HPBefore || g.loadErr == "" {
		t.Fatal("前奏消失未失敗即關閉")
	}
}
