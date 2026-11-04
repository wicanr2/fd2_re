package main

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
)

func TestNativePlayerFormalCommandEntryAndPresentation(t *testing.T) {
	for _, command := range []int{3, 4, 5, 7} {
		for _, side := range []byte{0, 2} {
			t.Run(fmt.Sprintf("command%d-side%d", command, side), func(t *testing.T) {
				g, actor, target := ordinaryCommand0HDTestGame(t)
				book, err := battle.LoadNativeCommandRecords("../../assets/spells.json")
				if err != nil {
					t.Fatal(err)
				}
				g.st.NativeCommandBook = book
				g.st.NativeCommandResistances, err = battle.LoadNativeCommandResistances("../../assets/data/native_command_resistances.json")
				if err != nil {
					t.Fatal(err)
				}
				actor.MP, actor.MaxMP, actor.NativeRecordByte6 = 200, 200, side
				if command == 4 || command == 5 {
					actor.BattleFig, actor.NativeRecordByte8 = 14, 14
					actor.NativeIdentity, actor.HasNativeIdentity = 14, true
				}
				target.HP, target.MaxHP = 900, 900
				g.sel, g.curX, g.curY = actor, target.X, target.Y
				if err := g.beginNativeCommandTargetSelection(command); err != nil {
					t.Fatal(err)
				}
				g.confirm()
				active := func() bool {
					return g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd7Presentation != nil
				}
				if !active() {
					t.Fatalf("formal command did not enter presentation: %s", g.msg)
				}
				screen := ebiten.NewImage(640, 400)
				for steps := 0; active() && steps < 5000; steps++ {
					switch command {
					case 3:
						g.drawNativeCommand3Presentation(screen)
						g.stepNativeCommand3Presentation()
					case 4, 5:
						g.drawNativeCommand5Presentation(screen)
						g.stepNativeCommand5Presentation()
					case 7:
						g.drawNativeCommand7Presentation(screen)
						g.stepNativeCommand7Presentation()
					}
				}
				if active() || actor.MP != 200-book[command].MPCost || !actor.Acted || g.nativeCommand0Targeting || g.sel != nil || g.loadErr != "" {
					t.Fatalf("formal command did not complete and restore cursor: MP%d acted%v error%s", actor.MP, actor.Acted, g.loadErr)
				}
			})
		}
	}
}

func nativeCommand3PresentationTestPlan(t *testing.T) (*battle.NativeCommandDamagePlan, *battle.Unit, *battle.Unit) {
	t.Helper()
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[3] = battle.NativeCommandRecord{ID: 3, Damage: 90, Hit: 100, SelectionMode: 1, MPCost: 2}
	actor := &battle.Unit{Camp: battle.Own, X: 0, Y: 0, HP: 20, MP: 5, OnField: true}
	target := &battle.Unit{Camp: battle.Enemy, ClassID: 5, X: 1, Y: 0, HP: 103, OnField: true}
	state := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	plan, err := state.PlanNativeCommandDamage(actor, target, 3, map[int]int{5: 10}, 3)
	if err != nil {
		t.Fatal(err)
	}
	return plan, actor, target
}

func TestNativeCommand3PresentationPublishesOnlyAfterDraw(t *testing.T) {
	plan, actor, target := nativeCommand3PresentationTestPlan(t)
	callback := 0
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd3Presentation = &nativeCommand3PresentationJob{
		actor: actor, plan: plan, prelude: make([]*ebiten.Image, 1),
		actorBlack: make([]*ebiten.Image, 1), actorPulse: make([]*ebiten.Image, 1),
		actorSpecs:    []battlepresent.NativeCommand0ActorFrame{{PublishMP: true}},
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: []int{target.HP},
		then: func(results []battle.NativeCommandDamageResult) { callback = len(results) },
	}
	for stage := 1; stage <= plan.DamageStages; stage++ {
		g.nativeCmd3Presentation.handler = append(g.nativeCmd3Presentation.handler, nativeCommand3HandlerFrame{targetIndex: 0, hpStage: stage})
	}

	g.stepNativeCommand3Presentation()
	if actor.MP != plan.MPBefore || actor.Acted || target.HP != plan.Results[0].HPBefore {
		t.Fatal("未繪製的第 3 指令改變了戰鬥狀態")
	}
	for g.nativeCmd3Presentation != nil {
		g.nativeCmd3Presentation.drawn = true
		g.stepNativeCommand3Presentation()
	}
	if actor.MP != plan.MPAfter || !actor.Acted || target.HP != plan.Results[0].HPAfter ||
		g.nativeRNGState != plan.RNGAfter || callback != 1 {
		t.Fatalf("第 3 指令交易未完成：mp=%d acted=%v hp=%d rng=%#x callback=%d", actor.MP, actor.Acted, target.HP, g.nativeRNGState, callback)
	}
}

func TestNativeCommand3PresentationRollsBackRuntimeFailure(t *testing.T) {
	plan, actor, target := nativeCommand3PresentationTestPlan(t)
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd3Presentation = &nativeCommand3PresentationJob{
		actor: actor, plan: plan, phase: nativeCommand3Actor,
		actorBlack: make([]*ebiten.Image, 1), actorPulse: make([]*ebiten.Image, 1),
		actorSpecs:    []battlepresent.NativeCommand0ActorFrame{{PublishMP: true}},
		handler:       []nativeCommand3HandlerFrame{{targetIndex: 0, hpStage: 2}},
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: []int{target.HP}, drawn: true,
	}
	g.stepNativeCommand3Presentation()
	if actor.MP != plan.MPAfter {
		t.Fatalf("MP 標記未發布：%d", actor.MP)
	}
	g.nativeCmd3Presentation.drawn = true
	g.stepNativeCommand3Presentation()
	if g.nativeCmd3Presentation != nil || actor.MP != plan.MPBefore || actor.Acted ||
		target.HP != plan.Results[0].HPBefore || g.nativeRNGState != plan.RNGBefore || g.loadErr == "" {
		t.Fatalf("失敗回復不完整：mp=%d acted=%v hp=%d rng=%#x err=%q", actor.MP, actor.Acted, target.HP, g.nativeRNGState, g.loadErr)
	}
}
