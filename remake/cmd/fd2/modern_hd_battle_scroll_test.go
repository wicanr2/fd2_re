package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// Explicit graphics fixture: original archer25 and bandit288/289, two shots
// and a counter. No rule, normal-player or original-executable claim.
func modernScrollSceneFixture(t *testing.T, side byte) (*Game, *nativePhysicalScene, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, actor, target := physicalSceneTestGame(t)
	g.m.NativeTerrainControl = append(g.m.NativeTerrainControl, 0, 0, 5, 0)
	g.m.Tiles[1] = 1 // two real BG55/5 inputs, not merely a header toggle.
	actor.BattleFig, target.BattleFig = 8, 96
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
