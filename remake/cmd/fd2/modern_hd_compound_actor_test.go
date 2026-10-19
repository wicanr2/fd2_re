package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func appendModernCompoundActorGPUCases(t *testing.T, cases []modernHDBattleGPUCase) []modernHDBattleGPUCase {
	t.Helper()
	dac, palette, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	actor, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 13)
	if err != nil {
		t.Fatal(err)
	}
	idle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 12)
	if err != nil {
		t.Fatal(err)
	}
	target, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 288)
	if err != nil {
		t.Fatal(err)
	}
	platform, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", 0)
	if err != nil {
		t.Fatal(err)
	}
	base := make([]byte, 64000)
	for i := range base {
		base[i] = byte(i % 251)
	}
	g := &Game{}
	for _, side := range []byte{0, 2} {
		in := battlepresent.NativeCommandPreludeInput{Base: base, ActorIdle: idle.Frames[0], Platform: &platform, RawSide: side, Mode: 1, BaselineDAC: dac}
		prelude, err := battlepresent.BuildNativeCommandPreludeFrames(in)
		if err != nil {
			t.Fatal(err)
		}
		frames := make([]nativeCompoundPresentedFrame, len(prelude))
		if err := bindModernHDCompoundPrelude(frames, prelude, in, 12); err != nil {
			t.Fatal(err)
		}
		for i, f := range frames {
			plan, err := g.modernHDBattlePlan(f.hdSources, f.hdPalette, f.hdSubtract)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("compound-prelude/%d/%d", side, i), plan: plan, indexed: prelude[i].Pixels})
		}
		presents, err := battlepresent.BuildNativeCompoundActorPresents(base, actor, target, side)
		if err != nil {
			t.Fatal(err)
		}
		frames = make([]nativeCompoundPresentedFrame, len(presents))
		if err := bindModernHDCompoundActor(frames, presents, base, actor, target, side, 13, 288, dac); err != nil {
			t.Fatal(err)
		}
		for i, f := range frames {
			plan, err := g.modernHDBattlePlan(f.hdSources, f.hdPalette, 0)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("compound-actor/%d/%d", side, i), plan: plan, indexed: presents[i].Pixels})
		}
	}
	for command := 32; command <= 35; command++ {
		effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", command+33)
		if err != nil {
			t.Fatal(err)
		}
		schedule, err := figani.BuildNativeCompoundPresentationSchedule(command, effect)
		if err != nil {
			t.Fatal(err)
		}
		original, err := battlepresent.BuildNativeCompoundCommonFrames(base, actor.Frames[0], effect, schedule)
		if err != nil {
			t.Fatal(err)
		}
		frames := make([]nativeCompoundPresentedFrame, len(original.ActorSlide))
		if err := bindModernHDCompoundActorSlide(frames, base, actor.Frames[0], 13, schedule, dac); err != nil {
			t.Fatal(err)
		}
		for i, f := range frames {
			plan, err := g.modernHDBattlePlan(f.hdSources, palette, 0)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("compound-actor-slide/%d/%d", command, i), plan: plan, indexed: original.ActorSlide[i]})
		}
	}
	return cases
}

func cloneModernHDBanksForTest(set *modernStoryPortraitSet) *modernStoryPortraitSet {
	copy := *set
	clone := func(entries map[int]*modernBattleHDResource) map[int]*modernBattleHDResource {
		out := make(map[int]*modernBattleHDResource, len(entries))
		for id, entry := range entries {
			value := *entry
			value.paths = append([]string(nil), entry.paths...)
			value.Frames = make([]image.Image, len(entry.Frames))
			out[id] = &value
		}
		return out
	}
	copy.battleAnimations, copy.battleEffects = clone(set.battleAnimations), clone(set.battleEffects)
	return &copy
}

func TestModernHDCompoundActorPreflightAndF2Atomic(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit full theme required")
	}
	t.Setenv("FD2_MUTE", "1")
	template, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	for command := 32; command <= 35; command++ {
		t.Run(fmt.Sprint(command), func(t *testing.T) {
			var g *Game
			var actor, target *battle.Unit
			switch command {
			case 32:
				g, actor, target = nativeCommand32PresentationFixture(t)
			case 33:
				g, actor, target = nativeCommand33PresentationFixture(t)
			case 34:
				g, actor, target = nativeCommand34PresentationFixture(t)
			case 35:
				g, actor, target = nativeCommand35PresentationFixture(t)
			}
			set := cloneModernHDBanksForTest(template)
			g.modernStoryPortraits = set
			beforeActor, beforeTarget, seed := *actor, *target, g.nativeRNGState
			start := func() error {
				switch command {
				case 32:
					return g.startNativeCommand32Presentation(actor, target, nil)
				case 33:
					return g.startNativeCommand33Presentation(actor, target, nil)
				case 34:
					return g.startNativeCommand34Presentation(actor, target, nil)
				default:
					return g.startNativeCommand35Presentation(actor, target, nil)
				}
			}
			entry := set.battleAnimations[actor.BattleFig*3+1]
			if entry == nil {
				t.Fatal("registered caster required")
			}
			last := len(entry.paths) - 1
			saved := entry.paths[last]
			entry.paths[last] = filepath.Join(t.TempDir(), "missing-caster.png")
			if err := start(); err == nil || g.modernCompoundEffectResource() != -1 || !reflect.DeepEqual(beforeActor, *actor) || !reflect.DeepEqual(beforeTarget, *target) || g.nativeRNGState != seed || entry.Frames[0] != nil || set.battleEffects[command+33].Frames[0] != nil {
				t.Fatal("caster preflight partially published")
			}
			entry.paths[last] = saved
			fdother6 := g.nativeMapAssets.FDOTHER6
			g.nativeMapAssets.FDOTHER6 = nil
			if err := start(); err == nil || g.modernCompoundEffectResource() != -1 || entry.Frames[0] != nil || set.battleEffects[command+33].Frames[0] != nil {
				t.Fatal("native validation failure published staged HD banks")
			}
			g.nativeMapAssets.FDOTHER6 = fdother6
			if err := start(); err != nil {
				t.Fatal(err)
			}
			if err := g.applyTheme(nil); err != nil {
				t.Fatal(err)
			}
			reload := cloneModernHDBanksForTest(template)
			reload.battleAnimations[actor.BattleFig*3+1].paths[last] = filepath.Join(t.TempDir(), "missing-on-f2.png")
			if err := g.applyTheme(reload); err == nil || g.modernStoryPortraits != nil || g.modernCompoundEffectResource() != command+33 || !reflect.DeepEqual(beforeActor, *actor) || !reflect.DeepEqual(beforeTarget, *target) || g.nativeRNGState != seed || reload.battleEffects[command+33].Frames[0] != nil || set.battleEffects[command+33].Frames[0] == nil {
				t.Fatal("F2 source failure changed owner, state or bank")
			}
		})
	}
}

func TestModernHDCompoundBanksCommitTogether(t *testing.T) {
	a, native, root := battleHDAssetFixture(t)
	set := &modernStoryPortraitSet{}
	if err := loadModernBattleHDAssetWithNative(set, a, root, native); err != nil {
		t.Fatal(err)
	}
	a.SourceResource = 65
	a.Role, a.SourceArchive = "battle_effect_hd_set", "FDOTHER.DAT"
	a.ConsumerContract, a.CyclePolicy, a.LayoutPolicy = "native_fdother_compound_rgba_hd6_v1", "native_compound_frames", "native_descriptor_hd6"
	if err := loadModernBattleHDAssetWithNative(set, a, root, native); err != nil {
		t.Fatal(err)
	}
	figure := set.battleAnimations[90]
	// Fixture resource is authoritative; do not assume its numeric selector.
	var id int
	for key, entry := range set.battleAnimations {
		id, figure = key, entry
	}
	held := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	held.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	set.battleAnimations[400] = &modernBattleHDResource{Frames: []image.Image{held}, paths: []string{"old-figure"}}
	set.battleEffects[66] = &modernBattleHDResource{Frames: []image.Image{held}, paths: []string{"old-effect"}}
	effect := set.battleEffects[65]
	saved := effect.paths[1]
	effect.paths[1] = filepath.Join(root, "absent.png")
	if err := set.prepareBattleHDBanks([]int{id}, []int{65}); err == nil || figure.Frames[0] != nil || effect.Frames[0] != nil || set.battleAnimations[400].Frames[0] != held || set.battleEffects[66].Frames[0] != held {
		t.Fatal("effect failure partially committed or evicted a bank")
	}
	effect.paths[1] = saved
	saved = figure.paths[1]
	figure.paths[1] = filepath.Join(root, "absent-figure.png")
	if err := set.prepareBattleHDBanks([]int{id}, []int{65}); err == nil || figure.Frames[0] != nil || effect.Frames[0] != nil || set.battleAnimations[400].Frames[0] != held || set.battleEffects[66].Frames[0] != held {
		t.Fatal("figure failure partially committed or evicted a bank")
	}
	figure.paths[1] = saved
	if err := set.prepareBattleHDBanks([]int{id}, []int{65}); err != nil {
		t.Fatal(err)
	}
	if figure.Frames[0] == nil || effect.Frames[0] == nil || set.battleAnimations[400].Frames[0] != nil || set.battleEffects[66].Frames[0] != nil {
		t.Fatal("successful two-bank preload did not publish together")
	}
}
