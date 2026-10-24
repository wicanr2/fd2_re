package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func appendModernCommand0GPUCases(t *testing.T, cases []modernHDBattleGPUCase) []modernHDBattleGPUCase {
	t.Helper()
	dac, palette, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	pulse := append(color.Palette(nil), palette...)
	pulse[0] = color.RGBA{255, 0, 0, 255}
	idle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 270)
	if err != nil {
		t.Fatal(err)
	}
	actor, actual, err := figani.LoadSeparatedResourceWithZeroHeaderFallbackSource(separatedAssetPath("animations"), 272)
	if err != nil || actual != 272 {
		t.Fatalf("real cast source: %d %v", actual, err)
	}
	target, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 0)
	if err != nil {
		t.Fatal(err)
	}
	bg, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", 0)
	if err != nil {
		t.Fatal(err)
	}
	tai, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", 0)
	if err != nil {
		t.Fatal(err)
	}
	g, _, _ := nativeCommand32PresentationFixture(t)
	base := make([]byte, 64000)
	for i := range base {
		base[i] = byte(i % 251)
	}
	bases := make([][]byte, 8)
	for stage := range bases {
		bases[stage] = append([]byte(nil), base...)
		for i := 0; i < 6400; i++ {
			bases[stage][i] = byte((i + stage*3) % 251)
		}
	}
	plan := &battle.NativeCommandDamagePlan{Results: []battle.NativeCommandDamageResult{{HPBefore: 100, HPAfter: 30}}}
	for _, side := range []byte{0, 2} {
		effectID := 18
		if side == 0 {
			effectID = 20
		}
		effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", effectID)
		if err != nil {
			t.Fatal(err)
		}
		schedule, err := figani.BuildNativeCommand0PresentationSchedule(side, effect)
		if err != nil {
			t.Fatal(err)
		}
		preludeIn := battlepresent.NativeCommandPreludeInput{Base: base, ActorIdle: idle.Frames[0], FirstTargetIdle: target.Frames[0], Platform: &tai, RawSide: side, Mode: 0, BaselineDAC: dac}
		prelude, err := battlepresent.BuildNativeCommandPreludeFrames(preludeIn)
		if err != nil {
			t.Fatal(err)
		}
		actorIn := battlepresent.NativeCommand0ActorInput{BaseBefore: base, BaseAfter: bases[0], ActorEffect: actor, FirstTargetIdle: target, RawSide: side, Background: bg, Platform: tai, LUT: g.nativeMapAssets.LUTs[11]}
		presents, err := battlepresent.BuildNativeCommand0ActorFrames(actorIn)
		if err != nil {
			t.Fatal(err)
		}
		targets, _, tail, err := nativeCommand0TargetAndTailPixels(plan, schedule, effect, actor, target, bases, bg, tai, g.nativeMapAssets.LUTs)
		if err != nil {
			t.Fatal(err)
		}
		frames, err := bindModernHDCommand0(modernHDCommand0Input{prelude: preludeIn, preludeFrames: prelude, actor: actorIn, actorFrames: presents,
			actorIdleID: 270, actorEffectID: 272, targetIdleID: 0, effectID: effectID, effect: effect, schedule: schedule, bases: bases, luts: g.nativeMapAssets.LUTs, palette: palette, pulse: pulse})
		if err != nil {
			t.Fatal(err)
		}
		var originals [][]byte
		for _, f := range prelude {
			originals = append(originals, f.Pixels)
		}
		for _, f := range presents {
			originals = append(originals, f.Pixels)
		}
		for _, f := range presents {
			originals = append(originals, f.Pixels)
		}
		originals = append(originals, targets...)
		originals = append(originals, tail...)
		all := append([]nativeCompoundPresentedFrame(nil), frames.prelude...)
		all = append(all, frames.actor...)
		all = append(all, frames.pulse...)
		all = append(all, frames.target...)
		all = append(all, frames.tail...)
		if len(all) != len(originals) {
			t.Fatal("incomplete ordinary command0 phases")
		}
		for i, f := range all {
			p, err := (&Game{}).modernHDBattlePlan(f.hdSources, f.hdPalette, f.hdSubtract)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("ordinary0/%d/%s/%d", side, f.hdPhase, i), plan: p, indexed: originals[i]})
		}
	}
	return cases
}

func ordinaryCommand0HDTestGame(t *testing.T, clocks ...func() time.Time) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, actor, target := nativeCommand32PresentationFixture(t, clocks...)
	actor.BattleFig, target.BattleFig = 9, 0
	if value := os.Getenv("FD2_TEST_COMMAND_ACTOR_FIGURE"); value != "" {
		figure, err := strconv.Atoi(value)
		if err != nil || figure < 0 || figure > 135 {
			t.Fatal("invalid explicit command actor figure")
		}
		actor.BattleFig = figure
	}
	g.st.NativeCommandBook[0] = battle.NativeCommandRecord{ID: 0, Damage: 50, Hit: 100, SelectionMode: 5, EffectMode: 0, MPCost: 2, TargetCode: 0}
	var err error
	g.nativeCommandScene, err = battle.LoadNativeCommandSceneTable("../../assets/data/native_command_scene.json")
	if err != nil {
		t.Fatal(err)
	}
	g.nativeCommandPaletteFlash, err = battle.LoadNativeCommandPaletteFlashTable("../../assets/data/native_command_palette_flash.json")
	if err != nil {
		t.Fatal(err)
	}
	return g, actor, target
}

func TestModernHDCommand0PreflightAndF2Atomic(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit full theme required")
	}
	t.Setenv("FD2_MUTE", "1")
	template, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	modernHDCommand0PreflightAndF2Atomic(t, template)
}

// 每次交易仍從獨立paths與全nil Frames開始；template只提供唯讀來源資料。
func modernHDCommand0PreflightAndF2Atomic(t *testing.T, template *modernStoryPortraitSet) {
	t.Helper()
	g, actor, target := ordinaryCommand0HDTestGame(t)
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
	if err := g.startNativeCommand0Presentation(actor, target, nil); err == nil || !strings.Contains(err.Error(), "missing.png") || g.nativeCmd0Presentation != nil || entry.Frames[0] != nil ||
		!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
		t.Fatal("failed cast preflight published state or bank")
	}
	entry.paths[last] = saved
	if err := g.startNativeCommand0Presentation(actor, target, nil); err != nil {
		t.Fatal(err)
	}
	j := g.nativeCmd0Presentation
	if !reflect.DeepEqual(j.hd.resources(), []int{0, actor.BattleFig * 3, actual}) {
		t.Fatalf("wrong fallback sources: %v", j.hd.resources())
	}
	if err := g.applyTheme(nil); err != nil {
		t.Fatal(err)
	}
	reload := cloneModernHDBanksForTest(template)
	reload.battleAnimations[actual].paths[last] = filepath.Join(t.TempDir(), "missing-f2.png")
	if err := g.applyTheme(reload); err == nil || g.modernStoryPortraits != nil || g.nativeCmd0Presentation != j ||
		reload.battleAnimations[actor.BattleFig*3].Frames[0] != nil || reload.battleAnimations[0].Frames[0] != nil ||
		!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
		t.Fatal("failed F2 changed owner, state or partial bank")
	}
	if err := g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	g.stepNativeCommand0Presentation()
	if j.frame != 0 || actor.MP != a.MP || target.HP != b.HP {
		t.Fatal("unacknowledged frame changed state")
	}
}

func TestModernHDCasters91And92Command0PreflightAndF2Atomic(t *testing.T) {
	if os.Getenv("FD2_HD_CASTERS_91_92_GPU") != "1" {
		t.Skip("dedicated casters 91 and 92 preflight process required")
	}
	for _, figure := range []int{91, 92} {
		t.Run(fmt.Sprint(figure), func(t *testing.T) {
			t.Setenv("FD2_TEST_COMMAND_ACTOR_FIGURE", fmt.Sprint(figure))
			TestModernHDCommand0PreflightAndF2Atomic(t)
		})
	}
}

func TestModernHDActors104To109Command0PreflightAndF2Atomic(t *testing.T) {
	if os.Getenv("FD2_HD_ACTORS_104_109_GPU") != "1" {
		t.Skip("dedicated actors 104, 105 and 107 preflight process required")
	}
	for _, figure := range []int{104, 105, 107} {
		t.Run(fmt.Sprint(figure), func(t *testing.T) {
			t.Setenv("FD2_TEST_COMMAND_ACTOR_FIGURE", fmt.Sprint(figure))
			TestModernHDCommand0PreflightAndF2Atomic(t)
		})
	}
}

func TestModernHDRemainingFIGANICommand0PreflightAndF2Atomic(t *testing.T) {
	if os.Getenv("FD2_HD_REMAINING_FIGANI_GPU") != "1" {
		t.Skip("dedicated remaining FIGANI command0 preflight process required")
	}
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Fatal("explicit full theme required")
	}
	t.Setenv("FD2_MUTE", "1")
	template, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	assertCold := func() {
		for _, banks := range []map[int]*modernBattleHDResource{template.battleAnimations, template.battleEffects} {
			for resource, bank := range banks {
				for _, frame := range bank.Frames {
					if frame != nil {
						t.Fatalf("shared template preloaded bank %d", resource)
					}
				}
			}
		}
	}
	assertCold()
	for _, figure := range []int{124, 125, 126, 127} {
		t.Run(fmt.Sprint(figure), func(t *testing.T) {
			t.Setenv("FD2_TEST_COMMAND_ACTOR_FIGURE", fmt.Sprint(figure))
			modernHDCommand0PreflightAndF2Atomic(t, template)
		})
		assertCold()
	}
}

func TestModernHDCommand0RejectsMultiMarkerThirdBanks(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit full theme required")
	}
	t.Setenv("FD2_MUTE", "1")
	template, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	for _, figure := range []int{32, 34} {
		t.Run(fmt.Sprint(figure), func(t *testing.T) {
			g, actor, target := ordinaryCommand0HDTestGame(t)
			actor.BattleFig = figure
			set := cloneModernHDBanksForTest(template)
			g.modernStoryPortraits = set
			a, b, seed := *actor, *target, g.nativeRNGState
			if err := g.startNativeCommand0Presentation(actor, target, nil); err == nil ||
				!strings.Contains(err.Error(), "command0 actor MP markers=2") {
				t.Fatalf("multi-marker source must reject command0: %v", err)
			}
			if g.nativeCmd0Presentation != nil || !reflect.DeepEqual(a, *actor) ||
				!reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
				t.Fatal("rejected source published an owner, unit state or RNG")
			}
			for _, resource := range []int{0, figure * 3, figure*3 + 2} {
				bank := set.battleAnimations[resource]
				if bank == nil {
					t.Fatalf("registered source required: %d", resource)
				}
				for _, frame := range bank.Frames {
					if frame != nil {
						t.Fatalf("rejected source partially loaded HD bank %d", resource)
					}
				}
			}
		})
	}
}
