package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type modifierMapGPUProbe struct {
	base                                 *modernHDCompoundProbe
	theme                                *modernStoryPortraitSet
	reference                            *Game
	referenceDone, done                  *int
	ticks, faithfulPixels, changedPixels int
	err                                  error
	nativeScreen                         *ebiten.Image
	nativePixels                         []byte
	caseInfo                             modifierHDCase
	seed                                 uint16
}

func (p *modifierMapGPUProbe) Layout(int, int) (int, int) { return 640, 400 }

func modifierMapFrame(g *Game) *nativeCompoundPresentedFrame {
	if j := g.nativeModifierPresentation; j != nil && j.phase >= 0 && j.phase < len(j.hdFrames) {
		return &j.hdFrames[j.phase]
	}
	if j := g.nativeAICommandModifier; j != nil && j.frame >= 0 && j.frame < len(j.frames) {
		return &j.frames[j.frame]
	}
	return nil
}

func stepModifierMap(g *Game) {
	if g.nativeModifierPresentation != nil {
		g.stepNativeCommandModifierPresentation()
	} else {
		g.stepNativeAICommandModifierPresentation()
	}
}

func (p *modifierMapGPUProbe) Update() error {
	if p.err != nil || p.base.err != nil {
		return ebiten.Termination
	}
	stepModifierMap(p.base.g)
	stepModifierMap(p.reference)
	if !reflect.DeepEqual(compoundMapUnitSnapshot(p.base.g), compoundMapUnitSnapshot(p.reference)) || p.base.g.nativeRNGState != p.reference.nativeRNGState || *p.done != *p.referenceDone {
		p.err = fmt.Errorf("HD changed native transaction at tick%d", p.ticks)
		return ebiten.Termination
	}
	if modifierMapFrame(p.base.g) == nil {
		if modifierMapFrame(p.reference) != nil || *p.done != 1 {
			p.err = fmt.Errorf("modifier returned without exactly one continuation")
		}
		return ebiten.Termination
	}
	p.ticks++
	if p.ticks > 1024 {
		p.err = fmt.Errorf("modifier owner did not finish")
		return ebiten.Termination
	}
	return nil
}

func (p *modifierMapGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.base.err != nil {
		return
	}
	g := p.base.g
	f := modifierMapFrame(g)
	native := modifierMapFrame(p.reference)
	if f == nil || native == nil || f.hdPhase != native.hdPhase || !bytes.Equal(f.hdMap.pixels, native.hdMap.pixels) {
		phase, referencePhase := "nil", "nil"
		if f != nil {
			phase = f.hdPhase
		}
		if native != nil {
			referencePhase = native.hdPhase
		}
		p.err = fmt.Errorf("native/HD frame schedules differ case%v tick%d phases%s/%s", p.caseInfo, p.ticks, phase, referencePhase)
		return
	}
	key := f.hdPhase
	if key == "map-mask" && len(f.hdMap.maskTargets) > 0 {
		key += "/filled"
	}
	if !p.base.phaseSwitches[key] {
		before, rng := compoundMapUnitSnapshot(g), g.nativeRNGState
		if err := g.applyTheme(nil); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible || g.modernHDBattleVisible {
			p.err = fmt.Errorf("faithful retained HD world")
			return
		}
		screen.ReadPixels(p.nativePixels)
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				r, green, b, a := f.hdMap.scene.palette[f.hdMap.pixels[y*320+x]].RGBA()
				at := (y*2*640 + x*2) * 4
				for c, v := range []byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), byte(a >> 8)} {
					diff := int(p.nativePixels[at+c]) - int(v)
					if diff < -1 || diff > 1 {
						p.err = fmt.Errorf("faithful pixel(%d,%d) channel%d differs", x, y, c)
						return
					}
				}
				p.faithfulPixels++
			}
		}
		if err := g.applyTheme(p.theme); err != nil {
			p.err = err
			return
		}
		if !reflect.DeepEqual(before, compoundMapUnitSnapshot(g)) || rng != g.nativeRNGState || modifierMapFrame(g) != f {
			p.err = fmt.Errorf("F2 changed active modifier")
			return
		}
		p.base.phaseSwitches[key] = true
	}
	g.Draw(screen)
	p.reference.Draw(p.nativeScreen)
	p.base.drawCompoundMapTail(f, p.ticks, screen)
	if p.base.err != nil {
		return
	}
	// Compare actual native and HD canvases on an independent full-world lattice.
	// A prepared source alone cannot prove that HD appeared in Game.Draw.
	p.nativeScreen.ReadPixels(p.nativePixels)
	for y := 0; y < 1200; y += 11 {
		for x := 0; x < 1920; x += 13 {
			at := (y*1920 + x) * 4
			nativeAt := ((y/3)*640 + x/3) * 4
			if !bytes.Equal(compoundMapGPUStats.Readback[at:at+4], p.nativePixels[nativeAt:nativeAt+4]) {
				p.changedPixels++
			}
		}
	}
}

func newModifierMapGPUProbe(t *testing.T, c modifierHDCase, set *modernStoryPortraitSet, out string, seeds ...uint16) *modifierMapGPUProbe {
	t.Helper()
	g, _, _ := modifierHDTestGame(t, c)
	reference, _, _ := modifierHDTestGame(t, c)
	seed := uint16(7)
	if len(seeds) > 1 {
		t.Fatal("one fixed seed required")
	}
	if len(seeds) == 1 {
		seed = seeds[0]
	}
	g.nativeRNGState, reference.nativeRNGState = seed, seed
	initCompoundMapFixture(t, g)
	done, referenceDone := new(int), new(int)
	g.modernStoryPortraits = set
	startModifierHDTest(t, g, c, done)
	startModifierHDTest(t, reference, c, referenceDone)
	left, right := g.modernCompoundMapFrames(), reference.modernCompoundMapFrames()
	if len(left) != len(right) {
		t.Fatal("initial control frame count differs")
	}
	for i := range left {
		if left[i].hdPhase != right[i].hdPhase || !bytes.Equal(left[i].hdMap.pixels, right[i].hdMap.pixels) {
			t.Fatal(c, "initial control snapshot differs at frame", i)
		}
	}
	base := &modernHDCompoundProbe{g: g, out: out, phaseSwitches: make(map[string]bool), shots: make(map[string]bool), phaseDraws: make(map[string]int)}
	base.presentation = &modernHDPresentation{Game: g}
	if base.out != "" {
		if err := os.MkdirAll(base.out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	return &modifierMapGPUProbe{base: base, theme: set, reference: reference, done: done, referenceDone: referenceDone, caseInfo: c, seed: seed, nativeScreen: ebiten.NewImage(640, 400), nativePixels: make([]byte, 640*400*4)}
}

func (p *modifierMapGPUProbe) result() (map[string]any, error) {
	base := p.base
	if p.err != nil || base.err != nil {
		return nil, fmt.Errorf("modifier GPU: %v %v", p.err, base.err)
	}
	stats := compoundMapGPUStats
	if os.Getenv("FD2_HD_MODIFIER_REQUIRE_EFFECTS") == "1" && stats.NativeLayers != 0 {
		return nil, fmt.Errorf("modifier still has %d native LMI layers", stats.NativeLayers)
	}
	if stats.Draws < 8 || stats.Pixels < 10000 || p.changedPixels == 0 || *p.done != 1 || *p.referenceDone != 1 {
		return nil, fmt.Errorf("missing actual HD/native/return coverage")
	}
	for phase := range stats.Phases {
		if !base.phaseSwitches[phase] {
			return nil, fmt.Errorf("missing F2 phase %s", phase)
		}
	}
	return map[string]any{"status": "PASS-MODIFIER-MAP-GAME", "case": p.caseInfo, "seed": p.seed, "native_control_seed": p.seed, "clock_unix_seconds": 1000, "native_control_units_rng_equal_after_every_step": true, "continuations": *p.done,
		"audited_draws": stats.Draws, "matched_pixels": stats.Pixels, "registered_overlay_pixels": stats.OverlayPixels, "changed_overlay_pixels": stats.ChangedOverlayPixels, "registered_layers": stats.HDLayers, "native_fallback_layers": stats.NativeLayers, "phases": stats.Phases, "phase_f2_switches": base.phaseSwitches, "faithful_palette_pixels": p.faithfulPixels, "different_native_hd_lattice_pixels": p.changedPixels, "source_shots": base.sourceShots, "capture_directory": base.out, "scope": "actual Game.Draw and final GPU of controlled fixtures; command23 palette only; unregistered effects retain native fallback; not ordinary player or original RGB parity"}, nil
}

type modifierMapGPUBatch struct {
	cases []*modifierMapGPUProbe
	index int
	rows  []map[string]any
	err   error
}

func (b *modifierMapGPUBatch) Layout(int, int) (int, int) { return 640, 400 }
func (b *modifierMapGPUBatch) Draw(screen *ebiten.Image) {
	if b.err == nil && b.index < len(b.cases) {
		b.cases[b.index].Draw(screen)
	}
}
func (b *modifierMapGPUBatch) Update() error {
	if b.err != nil || b.index >= len(b.cases) {
		return ebiten.Termination
	}
	p := b.cases[b.index]
	if err := p.Update(); err != nil {
		row, resultErr := p.result()
		if resultErr != nil {
			b.err = fmt.Errorf("case%v: %w", p.caseInfo, resultErr)
			return ebiten.Termination
		}
		b.rows = append(b.rows, row)
		p.base.g, p.base.presentation, p.reference = nil, nil, nil
		b.cases[b.index] = nil
		b.index++
		if b.index == len(b.cases) {
			return ebiten.Termination
		}
		compoundMapGPUStats.Draws, compoundMapGPUStats.Pixels = 0, 0
		compoundMapGPUStats.HDLayers, compoundMapGPUStats.NativeLayers = 0, 0
		compoundMapGPUStats.OverlayPixels, compoundMapGPUStats.ChangedOverlayPixels = 0, 0
		compoundMapGPUStats.Phases = make(map[string]int)
		compoundMapGPUStats.Worlds = make(map[*modernHDMapFrame]*image.RGBA)
		compoundMapGPUStats.Captures = make(map[string]bool)
		runtime.GC()
	}
	return nil
}

func TestModernModifierMapBatchGPU(t *testing.T) {
	if os.Getenv("FD2_HD_MODIFIER_MAP_BATCH") != "1" {
		t.Skip("explicit bounded modifier GPU slice")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_HD_COMPOUND_MAP", "1")
	start, e1 := strconv.Atoi(os.Getenv("FD2_HD_MODIFIER_START"))
	limit, e2 := strconv.Atoi(os.Getenv("FD2_HD_MODIFIER_LIMIT"))
	cases := modifierHDCases()
	if e1 != nil || e2 != nil || start < 0 || limit < 1 || limit > 6 || start+limit > len(cases) {
		t.Fatal("bounded source slice required")
	}
	set := modifierHDTestTheme(t)
	batch := &modifierMapGPUBatch{}
	for _, c := range cases[start : start+limit] {
		out := filepath.Join(os.Getenv("FD2_HD_MODIFIER_CAPTURE"), fmt.Sprintf("command%d-player%t-failed%t", c.ID, c.Player, c.Failed))
		batch.cases = append(batch.cases, newModifierMapGPUProbe(t, c, set, out))
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(batch); err != nil {
		t.Fatal(err)
	}
	if batch.err != nil || len(batch.rows) != limit {
		t.Fatal("modifier GPU slice", batch.err, len(batch.rows))
	}
	modifierHDReport(t, ".batch.json", map[string]any{"status": "PASS-MODIFIER-MAP-GAME-SLICE", "start": start, "limit": limit, "cases": batch.rows, "scope": "bounded controlled Game.Draw/native pairs, fixed clock from initialization; command23 palette only"})
}

func TestModernModifierMapZeroSeedApplicationGPU(t *testing.T) {
	if os.Getenv("FD2_HD_MODIFIER_MAP_ZERO") != "1" {
		t.Skip("explicit existing seed0 application control")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_HD_COMPOUND_MAP", "1")
	c := modifierHDCase{ID: 26}
	p := newModifierMapGPUProbe(t, c, modifierHDTestTheme(t), os.Getenv("FD2_HD_MODIFIER_CAPTURE"), 0)
	applied := 0
	for _, r := range p.base.g.nativeAICommandModifier.plan2022.Results {
		if r.Apply != nil && r.Apply.Applied {
			applied++
		}
	}
	if applied == 0 {
		t.Fatal("existing original-data seed0 control did not exercise application")
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	result, err := p.result()
	if err != nil {
		t.Fatal(err)
	}
	result["application_successes"] = applied
	result["seed_policy"] = "separate fixed seed0 control from TestExecuteNativeAICommand26UsesOriginalIndexedTail; seed7 matrix retained, no rerolls"
	modifierHDReport(t, ".game.json", result)
}

func TestModernModifierMapRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_MODIFIER_MAP_GPU") != "1" {
		t.Skip("dedicated modifier runtime GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_HD_COMPOUND_MAP", "1")
	id, err := strconv.Atoi(os.Getenv("FD2_HD_MODIFIER_COMMAND"))
	if err != nil {
		t.Fatal(err)
	}
	c := modifierHDCase{ID: id, Player: os.Getenv("FD2_HD_MODIFIER_PLAYER") == "1", Failed: os.Getenv("FD2_HD_MODIFIER_FAILED") == "1"}
	legal := false
	for _, candidate := range modifierHDCases() {
		if candidate == c {
			legal = true
		}
	}
	if !legal {
		t.Fatal("explicit legal modifier fixture required", c)
	}
	p := newModifierMapGPUProbe(t, c, modifierHDTestTheme(t), os.Getenv("FD2_HD_MODIFIER_CAPTURE"))
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	result, err := p.result()
	if err != nil {
		t.Fatal(err)
	}
	modifierHDReport(t, ".game.json", result)
}
