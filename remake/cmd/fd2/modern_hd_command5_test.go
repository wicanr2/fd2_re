package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func appendModernCommand5GPUCases(t *testing.T, cases []modernHDBattleGPUCase) []modernHDBattleGPUCase {
	t.Helper()
	_, palette, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	actor, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 272)
	if err != nil {
		t.Fatal(err)
	}
	var targets []*figani.Animation
	for _, id := range []int{0, 24} {
		idle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), id)
		if err != nil {
			t.Fatal(err)
		}
		targets = append(targets, idle)
	}
	bases := make([][][]byte, 2)
	for target := range bases {
		for stage := 0; stage <= figani.NativeCommand5DamageStages; stage++ {
			base := make([]byte, 64000)
			for i := range base {
				base[i] = byte((i + stage*3 + target*13) % 251)
			}
			bases[target] = append(bases[target], base)
		}
	}
	for _, side := range []byte{0, 2} {
		for _, command := range []int{4, 5} {
			effectID := 22 + (command-4)*2
			if side == 0 {
				effectID++
			}
			effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", effectID)
			if err != nil {
				t.Fatal(err)
			}
			var schedule figani.NativeCommand5PresentationSchedule
			if command == 4 {
				schedule, err = figani.BuildNativeCommand4PresentationSchedule(side, effect)
			} else {
				schedule, err = figani.BuildNativeCommand5PresentationSchedule(side, effect)
			}
			if err != nil {
				t.Fatal(err)
			}
			sequence, err := battlepresent.BuildNativeCommand5EffectSequence(battlepresent.NativeCommand5EffectInput{
				FrontBase: bases[0][0], TailBase: bases[1][6], TargetBases: bases, TransitionBases: [][]byte{bases[0][6]},
				ActorEffect: actor, TargetIdle: targets, Effect: effect, Schedule: schedule, RawSide: side, VisualRNG: 7})
			if err != nil {
				t.Fatal(err)
			}
			originals := append([]battlepresent.NativeCommand5RenderedFrame(nil), sequence.Front...)
			for i, frames := range sequence.Targets {
				originals = append(originals, frames...)
				if i < len(sequence.Transitions) {
					originals = append(originals, sequence.Transitions[i]...)
				}
			}
			originals = append(originals, sequence.Tail...)
			bound, err := bindModernHDCommand5Handler(originals, actor, 272, targets, []int{0, 24}, effect, effectID, palette)
			if err != nil || len(bound) != len(originals) {
				t.Fatalf("handler binding: %v", err)
			}
			for i, f := range bound {
				plan, err := (&Game{}).modernHDBattlePlan(f.hdSources, f.hdPalette, 0)
				if err != nil {
					t.Fatal(err)
				}
				cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("ordinary%d/%d/%s/%d", command, side, f.hdPhase, i), plan: plan, indexed: originals[i].Pixels})
			}
			bad := append([]battlepresent.NativeCommand5RenderedFrame(nil), originals...)
			bad[len(bad)-1].Sources.TargetIndex = 3
			if result, err := bindModernHDCommand5Handler(bad, actor, 272, targets, []int{0, 24}, effect, effectID, palette); err == nil || result != nil {
				t.Fatal("late source failure returned a partial batch")
			}
		}
	}
	return cases
}

func TestModernHDCommand5PreflightAndF2Atomic(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit full theme required")
	}
	t.Setenv("FD2_MUTE", "1")
	template, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	g, actor, target := ordinaryCommand5HDTestGame(t)
	set := cloneModernHDBanksForTest(template)
	g.modernStoryPortraits = set
	a, b, seed := *actor, *target, g.nativeRNGState
	_, actual, err := figani.LoadSeparatedResourceWithZeroHeaderFallbackSource(separatedAssetPath("animations"), actor.BattleFig*3+2)
	if err != nil {
		t.Fatal(err)
	}
	entry := set.battleAnimations[actual]
	if entry == nil {
		t.Fatal("registered actual cast source required")
	}
	last := len(entry.paths) - 1
	saved := entry.paths[last]
	entry.paths[last] = filepath.Join(t.TempDir(), "missing.png")
	if err := g.startNativeCommand5Presentation(actor, target, nil); err == nil || !strings.Contains(err.Error(), "missing.png") || g.nativeCmd5Presentation != nil || entry.Frames[0] != nil ||
		!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
		t.Fatal("failed cast preflight published state or bank")
	}
	entry.paths[last] = saved
	if err := g.startNativeCommand5Presentation(actor, target, nil); err != nil {
		t.Fatal(err)
	}
	j := g.nativeCmd5Presentation
	if !reflect.DeepEqual(j.hd.resources(), []int{0, actor.BattleFig * 3, actual}) {
		t.Fatalf("wrong fallback sources: %v", j.hd.resources())
	}
	if err := g.applyTheme(nil); err != nil {
		t.Fatal(err)
	}
	reload := cloneModernHDBanksForTest(template)
	reload.battleAnimations[actual].paths[last] = filepath.Join(t.TempDir(), "missing-f2.png")
	if err := g.applyTheme(reload); err == nil || g.modernStoryPortraits != nil || g.nativeCmd5Presentation != j ||
		reload.battleAnimations[actor.BattleFig*3].Frames[0] != nil || reload.battleAnimations[0].Frames[0] != nil ||
		!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
		t.Fatal("failed F2 changed owner, state or partial bank")
	}
	if err := g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	g.stepNativeCommand5Presentation()
	if j.frame != 0 || actor.MP != a.MP || target.HP != b.HP {
		t.Fatal("unacknowledged frame changed state")
	}
}
