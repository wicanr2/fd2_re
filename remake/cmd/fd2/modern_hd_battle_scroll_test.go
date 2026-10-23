package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// Explicit graphics fixture: original archer25 and bandit288/289, two shots
// and a counter. No rule, normal-player or original-executable claim.
func modernScrollSceneFixture(t *testing.T, side byte) (*Game, *nativePhysicalScene, *battle.Unit, *battle.Unit) {
	return modernScrollTargetSceneFixture(t, side, 96)
}

func modernScrollTargetSceneFixture(t *testing.T, side byte, targetFigure int) (*Game, *nativePhysicalScene, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, actor, target := physicalSceneTestGame(t)
	actorBG, targetBG := byte(55), byte(5)
	if pair := os.Getenv("FD2_HD_SCROLL_SCENERY"); pair != "" {
		switch pair {
		case "4:5":
			actorBG, targetBG = 4, 5
		case "5:6":
			actorBG, targetBG = 5, 6
		case "6:4":
			actorBG, targetBG = 6, 4
		case "7:10":
			actorBG, targetBG = 7, 10
		case "34:36":
			actorBG, targetBG = 34, 36
		case "36:37":
			actorBG, targetBG = 36, 37
		default:
			t.Fatal("unsupported explicit scenery pair", pair)
		}
	}
	g.m.NativeTerrainControl[2] = actorBG
	g.m.NativeTerrainControl = append(g.m.NativeTerrainControl, 0, 0, targetBG, 0)
	g.m.Tiles[1] = 1 // two real BG55/5 inputs, not merely a header toggle.
	actor.BattleFig, target.BattleFig = 8, targetFigure
	actor.NativeRecordByte6, target.NativeRecordByte6 = side, 2
	if side != 0 {
		target.NativeRecordByte6 = 0
	}
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	if !scene.selection.HasSeparateBackgrounds || scene.bodyResources.attack.HeaderByte1 != 1 {
		t.Fatal("fixture did not select original nonzero header")
	}
	scene.bodyResources.counter, err = figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3+1)
	if err != nil {
		t.Fatal(err)
	}
	scene.bodyResources.counterSounds, err = g.nativePhysicalSoundBank(scene.bodyResources.counter)
	if err != nil {
		t.Fatal(err)
	}
	result := battle.NativePhysicalAttackResult{AttackStrikes: []battle.NativePhysicalStrike{
		{Roll: battle.NativePhysicalRollResult{Damage: 7, Crit: true, Status: true}, DefenderHP: 73},
		{Roll: battle.NativePhysicalRollResult{Damage: 7}, DefenderHP: 66},
	}, Counter: &battle.AttackResult{}, CounterStrikes: []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 9}, DefenderHP: 71}}}
	if err := g.attachNativePhysicalBody(scene, result); err != nil {
		t.Fatal(err)
	}
	return g, scene, actor, target
}

func appendModernScrollGPUCases(t *testing.T, cases []modernHDBattleGPUCase) []modernHDBattleGPUCase {
	t.Helper()
	for _, side := range []byte{0, 2} {
		g, scene, _, _ := modernScrollSceneFixture(t, side)
		for _, frame := range scene.prelude {
			sources, err := g.modernHDBattlePreludeSources(scene, frame.Stage)
			if err != nil {
				t.Fatal(err)
			}
			palette, err := fdother.ParseVGAPalette(frame.DAC)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := g.modernHDBattlePlan(sources, palette, float32(frame.Stage*6)/63)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("scroll-prelude/%d/%d", side, frame.Stage), plan: plan, indexed: frame.Pixels})
		}
		for index, sources := range scene.leadSources {
			plan, err := g.modernHDBattlePlan(sources, g.nativeUIPalette, 0)
			if err != nil {
				t.Fatal(err)
			}
			pixels := scene.departure
			if index >= len(pixels) {
				pixels = scene.transition
				index -= len(scene.departure)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("scroll-lead/%d/%d", side, len(cases)), plan: plan, indexed: pixels[index]})
		}
		phases := make(map[string]int)
		for index, job := range scene.body.jobs {
			if job.hdSources == nil {
				continue
			}
			palette := append(color.Palette(nil), g.nativeUIPalette...)
			if job.palette0 != nil {
				palette[0] = *job.palette0
			}
			plan, err := g.modernHDBattlePlan(job.hdSources, palette, 0)
			if err != nil {
				t.Fatal(err)
			}
			pixels, err := job.indexed()
			if err != nil {
				t.Fatal(err)
			}
			phases[job.hdPhase]++
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("scroll-body/%d/%s/%d", side, job.hdPhase, index), plan: plan, indexed: pixels})
		}
		for _, phase := range []string{"departure", "scroll-out", "scroll-in", "body"} {
			if phases[phase] == 0 {
				t.Fatalf("missing %s", phase)
			}
		}
	}
	// Wide synthetic target crosses a work half and wraps a linear base; BG
	// transparent holes must retain the entire earlier write history.
	_, paletteForEdges, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	var layers [3]fdother.Frame
	for i := range layers {
		layers[i] = fdother.Frame{Width: 320, Height: 100, Indexed: make([]byte, 32000), Mask: make([]byte, 32000)}
		for j := range layers[i].Indexed {
			layers[i].Indexed[j] = byte(20 + i*10)
			if (j+i)%4 != 0 {
				layers[i].Mask[j] = 255
			}
		}
	}
	base, targetBase := make([]byte, 64000), make([]byte, 64000)
	for i := range base {
		base[i] = 70
		targetBase[i] = 80
	}
	f := figani.Frame{X: 300, Y: 154, Width: 40, Height: 3, Pixels: bytes.Repeat([]byte{99}, 120), Mask: bytes.Repeat([]byte{255}, 120)}
	sourceRaw := append([]byte(nil), base...)
	if err := f.BlitTranslated(sourceRaw, 320, 0, 0, nil); err != nil {
		t.Fatal(err)
	}
	source := &modernHDBattleSources{Base: base, Layers: []modernHDBattleSource{{Frame: f, Resource: -1, Fill: -1}}}
	idle := figani.Frame{X: 300, Y: 60, Width: 80, Height: 3, Pixels: make([]byte, 240), Mask: bytes.Repeat([]byte{255}, 240)}
	for _, right := range []bool{false, true} {
		presents, err := nativePhysicalBackgroundPresents(battlepresent.NativeCommand24BackgroundInputs{Layers: layers, Source: sourceRaw, Target: targetBase, TargetIdle: idle}, right)
		if err != nil {
			t.Fatal(err)
		}
		bound, err := modernHDBattleScrollSources(presents, source, targetBase, idle, -1, layers)
		if err != nil {
			t.Fatal(err)
		}
		for i, present := range presents {
			plan, err := (&Game{}).modernHDBattlePlan(bound[i], paletteForEdges, 0)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("scroll-mask-wrap/%v/%d", right, i), plan: plan, indexed: present.Pixels})
		}
	}
	// Independent HD CPU composite for full, partial and empty destination clips.
	_, palette, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	hd := image.NewNRGBA(image.Rect(0, 0, 12, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 12; x++ {
			hd.SetNRGBA(x, y, color.NRGBA{R: 180, G: 80, B: 20, A: 128})
		}
	}
	partial, empty := image.Rect(3, 3, 9, 9), image.Rectangle{}
	for index, clip := range []*image.Rectangle{nil, &partial, &empty} {
		want := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		draw.Draw(want, want.Bounds(), image.NewUniform(palette[0]), image.Point{}, draw.Src)
		bounds := hd.Bounds()
		if clip != nil {
			bounds = bounds.Intersect(*clip)
		}
		if !bounds.Empty() {
			draw.Draw(want, bounds, hd, bounds.Min, draw.Over)
		}
		plan := modernHDBattlePlan{Base: make([]byte, 64000), Palette: palette, Layers: []modernHDBattleLayer{{HD: hd, HDKey: "scroll-clip", Rect: hd.Bounds(), Fill: -1, Clip: clip}}}
		cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("HD-scroll-clip/%d", index), plan: plan, want: want, tolerance: 1})
	}
	return cases
}

func TestModernScrollSurfaceKeepsOpaqueZeroAndTransparentMask(t *testing.T) {
	for _, input := range []fdother.Frame{
		{Width: 2, Height: 1, Indexed: []byte{0, 99}, Mask: []byte{255, 0}},
		{Width: 2, Height: 1, Pixels: []byte{2, 0, 1, 0, 0, 0, 192}}, // literal0 then skip1
	} {
		f, err := modernHDBattleSurfaceFrame(input)
		if err != nil || !bytes.Equal(f.Mask, []byte{255, 0}) || f.Pixels[0] != 0 {
			t.Fatalf("surface mask: %v %v", f, err)
		}
	}
}

func TestModernScrollMetadataRejectsBeforePartialSources(t *testing.T) {
	var layers [3]fdother.Frame
	for i := range layers {
		layers[i] = fdother.Frame{Width: 320, Height: 100, Indexed: make([]byte, 32000), Mask: make([]byte, 32000)}
	}
	source := &modernHDBattleSources{Base: make([]byte, 64000)}
	presents := []battlepresent.NativeCommand24BackgroundPresent{{Pixels: make([]byte, 64000), ContentX: 0, BGX: 320, ViewportX: 32, BGLayers: []int{1}}, {Pixels: make([]byte, 64000), ContentX: 0, BGX: 320, ViewportX: 64, BGLayers: []int{9}}}
	if partial, err := modernHDBattleScrollSources(presents, source, make([]byte, 64000), figani.Frame{}, 0, layers); err == nil || partial != nil {
		t.Fatal("invalid later metadata published partial sources")
	}
}

func TestModernScrollSourceClipRejectsBeforeScaling(t *testing.T) {
	clip := image.Rectangle{Min: image.Pt(-1<<30, 0), Max: image.Pt(1<<30, 200)}
	sources := &modernHDBattleSources{Base: make([]byte, 64000), Layers: []modernHDBattleSource{{Resource: -1, Clip: &clip}}}
	if _, err := (&Game{}).modernHDBattlePlan(sources, nil, 0); err == nil {
		t.Fatal("unbounded logical clip accepted")
	}
}

func TestModernScrollSceneryNativeAndAtomic(t *testing.T) {
	requirePhysicalScenePack(t)
	catalog, pack := os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK")
	set := modernHDBattleFixtureSet(t, catalog, pack, nil)
	canvases, rejections := 0, 0
	for _, pair := range []string{"4:5", "5:6", "6:4"} {
		t.Setenv("FD2_HD_SCROLL_SCENERY", pair)
		for _, side := range []byte{0, 2} {
			g, scene, actor, target := modernScrollSceneFixture(t, side)
			check := func(name string, sources *modernHDBattleSources, native []byte) {
				t.Helper()
				if sources == nil || !bytes.Equal(surfaceIndexedReference(sources), native) {
					t.Fatalf("%s side%d %s native reconstruction differs", pair, side, name)
				}
				for _, source := range append(append([]modernHDBattleSource(nil), sources.BaseScenery...), sources.Layers...) {
					if source.Archive != "BG.DAT" && source.Archive != "TAI.DAT" {
						continue
					}
					picture, _, err := set.battleSurfaceHD(source)
					if source.Archive == "BG.DAT" && source.Resource >= 0 && source.Resource < 3 && set.battleSurfaces["BG.DAT"][source.Resource] == nil {
						if err != nil || picture != nil {
							t.Fatal("undeclared transition source did not use native fallback", err)
						}
						continue
					}
					if err != nil || picture == nil {
						t.Fatalf("scenery source lost provenance: %v", err)
					}
				}
				canvases++
			}
			for _, f := range scene.prelude {
				sources, err := g.modernHDBattlePreludeSources(scene, f.Stage)
				if err != nil {
					t.Fatal(err)
				}
				check("prelude", sources, f.Pixels)
			}
			for i, sources := range scene.leadSources {
				pixels := scene.departure
				j := i
				if j >= len(pixels) {
					j -= len(pixels)
					pixels = scene.transition
				}
				check("lead", sources, pixels[j])
			}
			for _, job := range scene.body.jobs {
				if job.hdSources == nil {
					continue
				}
				pixels, err := job.indexed()
				if err != nil {
					t.Fatal(err)
				}
				check(job.hdPhase, job.hdSources, pixels)
			}
			wanted := []struct {
				archive string
				index   byte
			}{
				{"BG.DAT", scene.selection.BaseBG}, {"BG.DAT", modernPhysicalTargetBGIndex(scene)}, {"TAI.DAT", scene.selection.TAI},
			}
			for index := 0; index < 3; index++ {
				if set.battleSurfaces["BG.DAT"][index] != nil {
					wanted = append(wanted, struct {
						archive string
						index   byte
					}{"BG.DAT", byte(index)})
				}
			}
			for _, source := range wanted {
				bad := *set
				bad.battleSurfaces = map[string]map[int]*modernBattleSurfaceHD{}
				for archive, entries := range set.battleSurfaces {
					bad.battleSurfaces[archive] = map[int]*modernBattleSurfaceHD{}
					for index, entry := range entries {
						bad.battleSurfaces[archive][index] = entry
					}
				}
				entry := *bad.battleSurfaces[source.archive][int(source.index)]
				entry.picture = nil
				bad.battleSurfaces[source.archive][int(source.index)] = &entry
				g.modernStoryPortraits = &bad
				beforeActor, beforeTarget, beforeRNG := *actor, *target, g.nativeRNGState
				if _, err := g.prepareNativePhysicalScene(actor, target); err == nil || !reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) || g.nativeRNGState != beforeRNG {
					t.Fatal("bad scenery did not stop before settlement", source)
				}
				rejections++
			}
		}
	}
	t.Logf("%d full native canvases across three pairs and both sides; %d scenery preflight rejections without state changes", canvases, rejections)
}

func TestModernScrollSceneryGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SCROLL_SCENERY_GPU") != "1" {
		t.Skip("dedicated scenery GPU process required")
	}
	requirePhysicalScenePack(t)
	set := modernHDBattleFixtureSet(t, os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK"), nil)
	var cases []modernHDBattleGPUCase
	for _, pair := range []string{"4:5", "5:6", "6:4"} {
		t.Setenv("FD2_HD_SCROLL_SCENERY", pair)
		for _, side := range []byte{0, 2} {
			g, scene, _, _ := modernScrollSceneFixture(t, side)
			g.modernStoryPortraits = set
			appendCase := func(name string, sources *modernHDBattleSources, subtract float32, palette color.Palette) {
				t.Helper()
				plan, err := g.modernHDBattlePlan(sources, palette, subtract)
				if err != nil {
					t.Fatal(err)
				}
				scenery := 0
				for _, layer := range plan.Layers {
					if layer.HD != nil {
						scenery++
					}
				}
				if scenery == 0 {
					t.Fatal("vacuous scenery GPU case", name)
				}
				cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("scenery/%s/%d/%s", pair, side, name), plan: plan, want: surfaceCPUWorld(plan), tolerance: 1})
			}
			for _, f := range scene.prelude {
				s, err := g.modernHDBattlePreludeSources(scene, f.Stage)
				if err != nil {
					t.Fatal(err)
				}
				palette, err := fdother.ParseVGAPalette(f.DAC)
				if err != nil {
					t.Fatal(err)
				}
				appendCase(fmt.Sprintf("prelude%d", f.Stage), s, float32(f.Stage*6)/63, palette)
			}
			counts := map[string]int{}
			var body, incoming *modernHDBattleSources
			for _, job := range scene.body.jobs {
				if job.hdSources == nil {
					continue
				}
				counts[job.hdPhase]++
				n := counts[job.hdPhase]
				if job.hdPhase == "body" && body == nil {
					body = job.hdSources
				}
				if job.hdPhase == "scroll-in" && incoming == nil {
					incoming = job.hdSources
				}
				if n != 1 && !((job.hdPhase == "scroll-out" || job.hdPhase == "scroll-in") && (n == 5 || n == 9)) {
					continue
				}
				palette := append(color.Palette(nil), g.nativeUIPalette...)
				if job.palette0 != nil {
					palette[0] = *job.palette0
				}
				appendCase(fmt.Sprintf("%s%d", job.hdPhase, n), job.hdSources, 0, palette)
			}
			if body == nil || incoming == nil {
				t.Fatal("missing body or incoming")
			}
			for _, archive := range []string{"BG.DAT", "TAI.DAT"} {
				only := &modernStoryPortraitSet{hasHD: true, battleSurfaces: map[string]map[int]*modernBattleSurfaceHD{archive: set.battleSurfaces[archive]}}
				g.modernStoryPortraits = only
				if !g.modernHDBattleWanted(scene) {
					t.Fatal("surface-only did not enable HD", archive)
				}
				// TAI occurs on the stationary right half: side0 target, side2 actor.
				s := body
				if side != 0 {
					s = scene.body.jobs[0].hdSources
				}
				appendCase(archive+"-only", s, 0, g.nativeUIPalette)
			}
		}
	}
	probe := &modernHDBattleGPUProbe{cases: cases}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	defer probe.gpu.close()
	if probe.err != nil || probe.index != len(cases) {
		t.Fatalf("scroll scenery GPU %d/%d: %v", probe.index, len(cases), probe.err)
	}
	t.Logf("%d real BG/TAI GPU worlds across three pairs, both sides, nine prelude shades, departure/scroll/body/counter and BG-only/TAI-only; independent RGBA8 Over tolerance1", len(cases))
}

func TestModernHDTransitionOnlyGPU(t *testing.T) {
	if os.Getenv("FD2_HD_TRANSITION_GPU") != "1" {
		t.Skip("dedicated transition-only GPU process required")
	}
	requirePhysicalScenePack(t)
	full := modernHDBattleFixtureSet(t, os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK"), nil)
	entries := map[int]*modernBattleSurfaceHD{}
	for index := 0; index < 3; index++ {
		if full.battleSurfaces["BG.DAT"][index] == nil {
			t.Fatal("required real transition missing", index)
		}
		entries[index] = full.battleSurfaces["BG.DAT"][index]
	}
	only := &modernStoryPortraitSet{hasHD: true, battleSurfaces: map[string]map[int]*modernBattleSurfaceHD{"BG.DAT": entries}}
	var cases []modernHDBattleGPUCase
	seen := map[int]bool{}
	t.Setenv("FD2_HD_SCROLL_SCENERY", "4:5")
	for _, side := range []byte{0, 2} {
		g, scene, _, _ := modernScrollSceneFixture(t, side)
		g.modernStoryPortraits = only
		if !g.modernHDBattleWanted(scene) {
			t.Fatal("transition-only theme did not enable physical owner")
		}
		add := func(name string, s *modernHDBattleSources, palette color.Palette, subtract float32) {
			t.Helper()
			plan, err := g.modernHDBattlePlan(s, palette, subtract)
			if err != nil {
				t.Fatal(err)
			}
			ordered := append(append([]modernHDBattleSource(nil), s.BaseScenery...), s.Layers...)
			for i, layer := range plan.Layers {
				if layer.HD != nil {
					source := ordered[i]
					if source.Archive != "BG.DAT" || source.Resource < 0 || source.Resource > 2 || layer.HDKey != entries[source.Resource].hash {
						t.Fatal("transition-only invented another HD source")
					}
					if !layer.Rect.Intersect(image.Rect(0, 0, 1920, 1200)).Empty() {
						seen[source.Resource] = true
					}
				}
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("transition-only/%d/%s", side, name), plan: plan, want: surfaceCPUWorld(plan), tolerance: 1})
		}
		for _, f := range scene.prelude {
			s, err := g.modernHDBattlePreludeSources(scene, f.Stage)
			if err != nil {
				t.Fatal(err)
			}
			palette, err := fdother.ParseVGAPalette(f.DAC)
			if err != nil {
				t.Fatal(err)
			}
			add(fmt.Sprintf("prelude%d", f.Stage), s, palette, float32(f.Stage*6)/63)
		}
		for i, s := range scene.leadSources {
			add(fmt.Sprintf("lead%d", i), s, g.nativeUIPalette, 0)
		}
		phases := map[string]bool{}
		for _, job := range scene.body.jobs {
			if job.hdSources == nil || phases[job.hdPhase] {
				continue
			}
			phases[job.hdPhase] = true
			add(job.hdPhase, job.hdSources, g.nativeUIPalette, 0)
		}
	}
	if len(seen) != 3 {
		t.Fatal("missing visible transition planes", seen)
	}
	probe := &modernHDBattleGPUProbe{cases: cases}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	defer probe.gpu.close()
	if probe.err != nil || probe.index != len(cases) {
		t.Fatalf("transition-only GPU %d/%d: %v", probe.index, len(cases), probe.err)
	}
	t.Logf("%d transition-only/native-fallback GPU worlds; both sides, all lead frames, native prelude/body/counter, three visible real HD backgrounds; independent Over tolerance1", len(cases))
}
