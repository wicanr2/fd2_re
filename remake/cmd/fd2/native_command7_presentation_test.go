package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func nativeCommand7PresentationTestPlan(t *testing.T) (*battle.NativeCommandDamagePlan, *battle.Unit, *battle.Unit) {
	t.Helper()
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[7] = battle.NativeCommandRecord{ID: 7, Damage: 90, Hit: 100, SelectionMode: 1, MPCost: 2}
	actor := &battle.Unit{Camp: battle.Own, X: 0, Y: 0, HP: 20, MP: 5, OnField: true}
	target := &battle.Unit{Camp: battle.Enemy, ClassID: 5, X: 1, Y: 0, HP: 103, OnField: true}
	state := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	plan, err := state.PlanNativeCommandDamage(actor, target, 7, map[int]int{5: 10}, 3)
	if err != nil {
		t.Fatal(err)
	}
	return plan, actor, target
}

func TestNativeCommand7PresentationPublishesOnlyAfterDraw(t *testing.T) {
	plan, actor, target := nativeCommand7PresentationTestPlan(t)
	callback := 0
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd7Presentation = &nativeCommand7PresentationJob{
		actor: actor, plan: plan, prelude: make([]*ebiten.Image, 1),
		actorBlack: make([]*ebiten.Image, 1), actorPulse: make([]*ebiten.Image, 1),
		actorSpecs:    []battlepresent.NativeCommand0ActorFrame{{PublishMP: true}},
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: []int{target.HP},
		then: func(results []battle.NativeCommandDamageResult) { callback = len(results) },
	}
	for stage := 1; stage <= plan.DamageStages; stage++ {
		g.nativeCmd7Presentation.handler = append(g.nativeCmd7Presentation.handler, nativeCommand7HandlerFrame{targetIndex: 0, hpStage: stage})
	}

	g.stepNativeCommand7Presentation()
	if actor.MP != plan.MPBefore || actor.Acted || target.HP != plan.Results[0].HPBefore {
		t.Fatal("未繪製的第 7 指令改變了戰鬥狀態")
	}
	for g.nativeCmd7Presentation != nil {
		g.nativeCmd7Presentation.drawn = true
		g.stepNativeCommand7Presentation()
	}
	if actor.MP != plan.MPAfter || !actor.Acted || target.HP != plan.Results[0].HPAfter ||
		g.nativeRNGState != plan.RNGAfter || callback != 1 {
		t.Fatalf("第 7 指令交易未完成：mp=%d acted=%v hp=%d rng=%#x callback=%d", actor.MP, actor.Acted, target.HP, g.nativeRNGState, callback)
	}
}

func TestNativeCommand7PresentationRollsBackRuntimeFailure(t *testing.T) {
	plan, actor, target := nativeCommand7PresentationTestPlan(t)
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd7Presentation = &nativeCommand7PresentationJob{
		actor: actor, plan: plan, phase: nativeCommand7Actor,
		actorBlack: make([]*ebiten.Image, 1), actorPulse: make([]*ebiten.Image, 1),
		actorSpecs:    []battlepresent.NativeCommand0ActorFrame{{PublishMP: true}},
		handler:       []nativeCommand7HandlerFrame{{targetIndex: 0, hpStage: 2}},
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: []int{target.HP}, drawn: true,
	}
	g.stepNativeCommand7Presentation()
	if actor.MP != plan.MPAfter {
		t.Fatalf("MP 標記未發布：%d", actor.MP)
	}
	g.nativeCmd7Presentation.drawn = true
	g.stepNativeCommand7Presentation()
	if g.nativeCmd7Presentation != nil || actor.MP != plan.MPBefore || actor.Acted ||
		target.HP != plan.Results[0].HPBefore || g.nativeRNGState != plan.RNGBefore || g.loadErr == "" {
		t.Fatalf("失敗回復不完整：mp=%d acted=%v hp=%d rng=%#x err=%q", actor.MP, actor.Acted, target.HP, g.nativeRNGState, g.loadErr)
	}
}

func TestNativeCommand7RNGMatchesOriginalChapter24Targets(t *testing.T) {
	// #152: canonical dosgolem command7-rng-original-r1, 0x1C75E entries
	// and 0x2AF40 markers. The third target remains alive for the second cast.
	book, err := battle.LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, side := range []byte{0, 1} {
		resource := 37
		if side == 0 {
			resource = 38
		}
		schedule := figani.NativeCommand7PresentationSchedule{EffectResource: resource}
		for _, tc := range []struct {
			seed, final uint16
			before      []uint16
			damage      []int
			hit         []bool
		}{
			{10947, 61757, []uint16{10947, 16895, 42561}, []int{433, 416, 409}, []bool{true, true, true}},
			{61757, 6897, []uint16{61757, 37706}, []int{419, 0}, []bool{true, false}},
		} {
			calls := 0
			final, err := figani.WalkNativeCommand7RNG(tc.seed, schedule, side, len(tc.before),
				func(index int, rng uint16) (uint16, bool, error) {
					if index != calls || rng != tc.before[index] {
						t.Fatalf("side=%d target=%d entry RNG=%d want=%d", side, index, rng, tc.before[index])
					}
					calls++
					result, next, err := battle.ResolveNativeCommandDamage(book[7].Damage, book[7].Hit, 10, rng)
					if result.Damage != tc.damage[index] || result.Hit != tc.hit[index] {
						t.Fatalf("side=%d target=%d result=%+v want damage=%d hit=%v", side, index, result, tc.damage[index], tc.hit[index])
					}
					return next, result.Hit, err
				})
			if err != nil || final != tc.final || calls != len(tc.before) {
				t.Fatalf("side=%d final=%d want=%d calls=%d err=%v", side, final, tc.final, calls, err)
			}
		}
	}
}
