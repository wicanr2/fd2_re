package main

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestExecuteNativeAICommand10UsesOriginalIndexedOwner(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	paths := map[string]string{
		"FD2_ORIGINAL_FIGANI":  filepath.Join(base, "FIGANI.DAT"),
		"FD2_ORIGINAL_FDOTHER": filepath.Join(base, "FDOTHER.DAT"),
		"FD2_ORIGINAL_FDTXT":   filepath.Join(base, "FDTXT.DAT"),
		"FD2_ORIGINAL_BG":      filepath.Join(base, "BG.DAT"),
		"FD2_ORIGINAL_TAI":     filepath.Join(base, "TAI.DAT"),
	}
	for key, path := range paths {
		if !fileExists(path) {
			t.Skip("player-provided original archives unavailable")
		}
		t.Setenv(key, path)
	}
	g := &Game{rng: rand.New(rand.NewSource(10)), nativeUIPalette: loadNativeUIPalette()}
	g.sfxCommand1012Prelude = loadWav("assets/sfx/battle_80_02.wav")
	g.sfxCommand1012Main = loadWav("assets/sfx/battle_80_13.wav")
	if err := g.loadMap("assets/maps/map0"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json")
	if g.loadErr != "" || len(g.st.Units) < 5 {
		t.Fatalf("fixture err=%q units=%d", g.loadErr, len(g.st.Units))
	}
	actor, target := g.st.Units[4], g.st.Units[0]
	for index, unit := range g.st.Units {
		if index != 0 && index != 4 {
			unit.OnField = false
		}
	}
	actor.SetMapPlacement(0, 0, 3)
	target.SetMapPlacement(1, 0, 3)
	if err := actor.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	if err := target.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	actor.Camp, actor.HP, actor.MaxHP, actor.MP, actor.Acted = battle.Enemy, 100, 100, 40, false
	target.Camp, target.HP, target.MaxHP, target.ClassID = battle.Own, 403, 403, 5
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[10] = battle.NativeCommandRecord{ID: 10, Damage: 999, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 30, TargetCode: 1}
	g.st.NativeCommandBook = book
	g.st.NativeCommandResistances = map[int]int{5: 10}
	g.st.NativeCompositionEventBytes = make([]byte, g.st.W*g.st.H)
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 1, CursorY: 0, VisibleCursorX: 1, VisibleCursorY: 0}); err != nil {
		t.Fatal(err)
	}
	if !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatal("HUD/range unavailable")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if !nativeMapAssetsAvailable(g.nativeMapAssets) || !g.st.HasNativeMapViewState ||
		len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize || len(g.nativeMapVGA) != indexedmap.NativeMapVGASize ||
		len(g.nativeMapAssets.CommandHealDigits) <= 118 || len(g.sfxCommand1012Main) == 0 {
		t.Fatalf("command10 assets unavailable: map=%t view=%t work=%d vga=%d digits=%d sound=%d",
			nativeMapAssetsAvailable(g.nativeMapAssets), g.st.HasNativeMapViewState, len(g.nativeMapWork), len(g.nativeMapVGA),
			len(g.nativeMapAssets.CommandHealDigits), len(g.sfxCommand1012Main))
	}
	g.nativeRNGState = 3
	continued := 0
	plan := &battle.AIPlan{U: actor, Target: target, NativeActionKind: battle.NativeAIActionCommand, NativeCommandID: 10}
	if err := g.executeNativeAIActionWithContinuation(plan, func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	if g.nativeCmd1012 == nil || actor.MP != 40 || target.HP != 403 || actor.Acted {
		t.Fatalf("AI command10 crossed first Draw boundary: actor=%#v target=%#v", actor, target)
	}
	screen := ebiten.NewImage(640, 400)
	for steps := 0; g.nativeCmd1012 != nil && steps < 512; steps++ {
		if !g.drawNativeCommand1012Presentation(screen) {
			t.Fatalf("draw failed at %d", steps)
		}
		g.stepNativeCommand1012Presentation()
		if g.nativeCmd1012 != nil && g.nativeCmd1012.phase == nativeCommand1012Hold {
			g.nativeCmd1012.hold = 0
		}
	}
	if g.nativeCmd1012 != nil || actor.MP != 10 || !actor.Acted || target.HP >= 403 || continued != 1 {
		t.Fatalf("AI command10 owner incomplete: actor=%#v target=%#v continued=%d", actor, target, continued)
	}
}

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
