package main

import (
	"fmt"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func appendModernCommand1GPUCases(t *testing.T, cases []modernHDBattleGPUCase) []modernHDBattleGPUCase {
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
		for stage := 0; stage <= 8; stage++ {
			base := make([]byte, 64000)
			for i := range base {
				base[i] = byte((i + stage*3 + target*13) % 251)
			}
			bases[target] = append(bases[target], base)
		}
	}
	for _, side := range []byte{0, 2} {
		effectID := 19
		if side == 0 {
			effectID = 21
		}
		effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", effectID)
		if err != nil {
			t.Fatal(err)
		}
		schedule, err := figani.BuildNativeCommand1PresentationSchedule(side, effect)
		if err != nil {
			t.Fatal(err)
		}
		for target := range targets {
			sequence, err := battlepresent.BuildNativeCommand1TargetSequence(bases[target], targets[target], effect, schedule)
			if err != nil {
				t.Fatal(err)
			}
			frames, sources := sequence.Frames, sequence.Sources
			anims, ids := targets[target:target+1], []int{[]int{0, 24}[target]}
			if target == 0 {
				transition, err := battlepresent.BuildNativeCommand1TransitionSequence(bases[target][8], actor, targets[0], targets[1], side)
				if err != nil {
					t.Fatal(err)
				}
				frames = append(frames, transition.Frames...)
				sources = append(sources, transition.Sources...)
				anims, ids = targets, []int{0, 24}
			}
			bound, err := bindModernHDCommand1Sources(sources, actor, 272, anims, ids, effect, effectID, palette, "command1-target")
			if err != nil || len(bound) != len(frames) {
				t.Fatalf("source binding: %v", err)
			}
			for i, f := range bound {
				plan, err := (&Game{}).modernHDBattlePlan(f.hdSources, f.hdPalette, 0)
				if err != nil {
					t.Fatal(err)
				}
				cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("ordinary1/%d/%d/%d", side, target, i), plan: plan, indexed: frames[i]})
			}
			bad := append([]battlepresent.NativeCommand1FrameSources(nil), sources...)
			bad[len(bad)-1].TargetFrame = 999
			if result, err := bindModernHDCommand1Sources(bad, actor, 272, anims, ids, effect, effectID, palette, "command1-target"); err == nil || result != nil {
				t.Fatal("late source failure returned a partial batch")
			}
		}
	}
	return cases
}

func TestModernHDCommand1PreflightAndF2Atomic(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit full theme required")
	}
	t.Setenv("FD2_MUTE", "1")
	template, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	g, actor, target := ordinaryCommand1HDTestGame(t)
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
	if err := g.startNativeCommand1Presentation(actor, target, nil); err == nil || !strings.Contains(err.Error(), "missing.png") || g.nativeCmd1Presentation != nil || entry.Frames[0] != nil ||
		!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
		t.Fatal("failed cast preflight published state or bank")
	}
	entry.paths[last] = saved
	if err := g.startNativeCommand1Presentation(actor, target, nil); err != nil {
		t.Fatal(err)
	}
	j := g.nativeCmd1Presentation
	if !reflect.DeepEqual(j.hd.resources(), []int{0, actor.BattleFig * 3, actual}) {
		t.Fatalf("wrong fallback sources: %v", j.hd.resources())
	}
	if err := g.applyTheme(nil); err != nil {
		t.Fatal(err)
	}
	reload := cloneModernHDBanksForTest(template)
	reload.battleAnimations[actual].paths[last] = filepath.Join(t.TempDir(), "missing-f2.png")
	if err := g.applyTheme(reload); err == nil || g.modernStoryPortraits != nil || g.nativeCmd1Presentation != j ||
		reload.battleAnimations[actor.BattleFig*3].Frames[0] != nil || reload.battleAnimations[0].Frames[0] != nil ||
		!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
		t.Fatal("failed F2 changed owner, state or partial bank")
	}
	if err := g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	g.stepNativeCommand1Presentation()
	if j.frame != 0 || actor.MP != a.MP || target.HP != b.HP {
		t.Fatal("unacknowledged frame changed state")
	}
}
