package main

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func nativeCommand6PresentationTestPlan(t *testing.T) (*battle.NativeCommandDamagePlan, *battle.Unit, *battle.Unit) {
	t.Helper()
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[6] = battle.NativeCommandRecord{
		ID: 6, Damage: 90, Hit: 100, SelectionMode: 1,
		EffectMode: 0, MPCost: 2, TargetCode: 0,
	}
	actor := &battle.Unit{Camp: battle.Own, X: 0, Y: 0, HP: 20, MP: 5, OnField: true}
	target := &battle.Unit{Camp: battle.Enemy, ClassID: 5, X: 1, Y: 0, HP: 103, OnField: true}
	state := &battle.State{
		W: 2, H: 1, Units: []*battle.Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book,
	}
	schedule := figani.NativeCommand6PresentationSchedule{EffectResource: 33, BaseByte: 90, DwordTable: [5]int{-10, -8, -3, 0, 0}}
	walk := func(count int, resolve func(int, uint16) (uint16, bool, error)) (uint16, error) {
		return figani.WalkNativeCommand6RNG(3, schedule, 0, count, resolve)
	}
	plan, err := state.PlanNativeCommandDamageWalk(actor, target, 6, map[int]int{5: 10}, 3, walk)
	if err != nil {
		t.Fatal(err)
	}
	return plan, actor, target
}

func TestNativeCommand6PresentationPublishesOnlyAfterDraw(t *testing.T) {
	plan, actor, target := nativeCommand6PresentationTestPlan(t)
	callback := 0
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd6Presentation = &nativeCommand6PresentationJob{
		actor: actor, plan: plan, prelude: make([]*ebiten.Image, 1),
		actorBlack: make([]*ebiten.Image, 1), actorPulse: make([]*ebiten.Image, 1),
		actorSpecs:    []battlepresent.NativeCommand0ActorFrame{{PublishMP: true}},
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted,
		targetHPBefore: []int{target.HP},
		then:           func(results []battle.NativeCommandDamageResult) { callback = len(results) },
	}
	for stage := 1; stage <= plan.DamageStages; stage++ {
		g.nativeCmd6Presentation.handler = append(g.nativeCmd6Presentation.handler,
			nativeCommand6HandlerFrame{targetIndex: 0, hpStage: stage})
	}

	g.stepNativeCommand6Presentation()
	if actor.MP != plan.MPBefore || actor.Acted || target.HP != plan.Results[0].HPBefore {
		t.Fatal("未繪製的第 6 指令改變了戰鬥狀態")
	}
	for g.nativeCmd6Presentation != nil {
		g.nativeCmd6Presentation.drawn = true
		g.stepNativeCommand6Presentation()
	}
	if actor.MP != plan.MPAfter || !actor.Acted || target.HP != plan.Results[0].HPAfter ||
		g.nativeRNGState != plan.RNGAfter || callback != 1 {
		t.Fatalf("第 6 指令交易未完成：mp=%d acted=%v hp=%d rng=%#x callback=%d",
			actor.MP, actor.Acted, target.HP, g.nativeRNGState, callback)
	}
}

func TestNativeCommand6PresentationRollsBackRuntimeFailure(t *testing.T) {
	plan, actor, target := nativeCommand6PresentationTestPlan(t)
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd6Presentation = &nativeCommand6PresentationJob{
		actor: actor, plan: plan, phase: nativeCommand6Actor,
		actorBlack: make([]*ebiten.Image, 1), actorPulse: make([]*ebiten.Image, 1),
		actorSpecs:    []battlepresent.NativeCommand0ActorFrame{{PublishMP: true}},
		handler:       []nativeCommand6HandlerFrame{{targetIndex: 0, hpStage: 2}},
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted,
		targetHPBefore: []int{target.HP}, drawn: true,
	}
	g.stepNativeCommand6Presentation()
	if actor.MP != plan.MPAfter {
		t.Fatalf("MP 標記未發布：%d", actor.MP)
	}
	g.nativeCmd6Presentation.drawn = true
	g.stepNativeCommand6Presentation()
	if g.nativeCmd6Presentation != nil || actor.MP != plan.MPBefore || actor.Acted ||
		target.HP != plan.Results[0].HPBefore || g.nativeRNGState != plan.RNGBefore || g.loadErr == "" {
		t.Fatalf("失敗回復不完整：mp=%d acted=%v hp=%d rng=%#x err=%q",
			actor.MP, actor.Acted, target.HP, g.nativeRNGState, g.loadErr)
	}
}

// #157 的正常原版收據記錄了各目標 entry、命中、傷害與施法末端 RNG。
// side0 有正常原版樣本；side1 在相同控制條件下驗證共用規則。
func TestNativeCommand6RNGMatchesOriginalChapter24Targets(t *testing.T) {
	book, err := battle.LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, side := range []byte{0, 1} {
		frames := make([]figani.Frame, figani.NativeCommand6EffectFrameCount)
		for i := range frames {
			frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}}
		}
		schedule, err := figani.BuildNativeCommand6PresentationSchedule(side, &figani.Animation{HeaderByte2: figani.NativeCommand6EffectFrameCount, Frames: frames})
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name           string
			seed, final    uint16
			before         []uint16
			resist, damage []int
			hit            []bool
		}{
			{"original-actor20", 14047, 32891, []uint16{14047, 57339}, []int{10, 7}, []int{212, 0}, []bool{true, false}},
			{"original-actor21", 32891, 61045, []uint16{32891}, []int{10}, []int{215}, []bool{true}},
			{"original-actor25", 2762, 59025, []uint16{2762}, []int{8}, []int{158}, []bool{true}},
			// 以下是明示的受控規則樣本，驗證後續命中的12個 marker 與先落空。
			{"controlled-hit-miss-hit", 14047, 62539, []uint16{14047, 57339, 32891}, []int{10, 8, 7}, []int{212, 0, 150}, []bool{true, false, true}},
			{"controlled-miss-hit-miss", 63768, 57895, []uint16{63768, 18788, 27696}, []int{10, 8, 7}, []int{0, 165, 0}, []bool{false, true, false}},
		} {
			t.Run(fmt.Sprintf("side%d/%s", side, tc.name), func(t *testing.T) {
				calls := 0
				final, err := figani.WalkNativeCommand6RNG(tc.seed, schedule, side, len(tc.before), func(index int, rng uint16) (uint16, bool, error) {
					if index != calls || rng != tc.before[index] {
						t.Fatalf("target=%d entry=%d want=%d calls=%d", index, rng, tc.before[index], calls)
					}
					calls++
					result, next, err := battle.ResolveNativeCommandDamage(book[6].Damage, book[6].Hit, tc.resist[index], rng)
					if result.Damage != tc.damage[index] || result.Hit != tc.hit[index] {
						t.Fatalf("target=%d result=%+v want damage=%d hit=%v", index, result, tc.damage[index], tc.hit[index])
					}
					return next, result.Hit, err
				})
				if err != nil || final != tc.final || calls != len(tc.before) {
					t.Fatalf("final=%d want=%d calls=%d err=%v", final, tc.final, calls, err)
				}
			})
		}
	}
}
