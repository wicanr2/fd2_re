package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// Actual map23 geometry and separated native banks, with two controlled actors.
// This is a direct fixture, not a normal campaign-path or original-EXE receipt.
func stage42ActualGame(t *testing.T, now *time.Time) *Game {
	t.Helper()
	g, actor, target := ordinaryCommand0HDTestGame(t, func() time.Time { return *now })
	if err := g.loadMap("assets/maps/map23"); err != nil {
		t.Fatal(err)
	}
	if g.nativeMapAssets == nil || g.nativeMapAssets.MapIndex != 23 || len(g.nativeMapAssets.LUTs) < 11 {
		actual, err := loadNativeMapAssets(assetPath("assets/maps/map23"))
		if err != nil {
			t.Fatal("real map23 bundle", err)
		}
		t.Fatalf("real map23 bundle absent: index=%d LUTs=%d", actual.MapIndex, len(actual.LUTs))
	}
	mapState, err := battle.Load(assetPath("assets/maps/map23/map23_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Keep the controlled actors, but import all geometry from the real map.
	g.st.W, g.st.H, g.st.Cost = mapState.W, mapState.H, mapState.Cost
	g.st.NativeCompositionEventBytes = mapState.NativeCompositionEventBytes
	g.st.NativeMapEventGrid, g.st.HasNativeMapEventGrid = mapState.NativeMapEventGrid, mapState.HasNativeMapEventGrid
	g.st.NativeTileBlitModes = mapState.NativeTileBlitModes
	g.st.NativeTerrainControl, g.st.NativeTerrainMoveCodes = mapState.NativeTerrainControl, mapState.NativeTerrainMoveCodes
	for _, u := range g.st.Units {
		u.OnField = u == actor || u == target
	}
	actor.SetMapPlacement(20, 18, 0)
	target.SetMapPlacement(21, 18, 0)
	for _, u := range []*battle.Unit{actor, target} {
		if err := u.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
	}
	view := battle.NativeMapViewState{CameraX: 16, CameraY: 14, CursorX: 20, CursorY: 18, VisibleCursorX: 4, VisibleCursorY: 4}
	if err := g.st.MaterializeNativeMapViewState(view); err != nil {
		t.Fatal(err)
	}
	g.st.MaterializeNativeMapHUDState(1, 1, 1)
	g.st.MaterializeNativeMapRangeMode(1)
	g.curX, g.curY = 20, 18
	g.nativeMapWork, g.nativeMapVGA = nil, nil
	g.nativeMapClock.Reset()
	g.nativeFDOTHERPaletteTick, g.nativeFDOTHERPalettePhase = 0, 0
	if err := g.composeNativeMapFrameAt(*now); err != nil {
		t.Fatal(err)
	}
	return g
}

type stage42GPUProbe struct {
	checker                                      unitPresentGPUProbe
	g, control                                   *Game
	now                                          time.Time
	phase                                        string
	draws, switches, continued, controlContinued int
	out                                          string
	controlScreen                                *ebiten.Image
	rows, returns                                []map[string]any
}

func (p *stage42GPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *stage42GPUProbe) Update() error {
	if p.checker.err != nil || p.phase == "done" {
		return ebiten.Termination
	}
	return nil
}
func (p *stage42GPUProbe) save(name string, pixels []byte) error {
	return (&rowShakeGPUProbe{out: p.out}).save(name, pixels)
}
func (p *stage42GPUProbe) start(phase string) error {
	p.phase, p.draws = phase, 0
	if err := p.g.startNativeCh23Loop(nativeCh23LoopForTest(phase), func() { p.continued++ }); err != nil {
		return err
	}
	return p.control.startNativeCh23Loop(nativeCh23LoopForTest(phase), func() { p.controlContinued++ })
}
func (p *stage42GPUProbe) Draw(screen *ebiten.Image) {
	if p.checker.err != nil || p.phase == "done" {
		return
	}
	g, j := p.g, p.g.nativeCh23Loop
	if j == nil {
		p.checker.err = fmt.Errorf("lost stage owner: %s", g.loadErr)
		return
	}
	before := stage42NativeState(g)
	g.stepNativeCh23LoopAt(p.now)
	if !reflect.DeepEqual(before, stage42NativeState(g)) {
		p.checker.err = fmt.Errorf("advanced without Draw")
		return
	}
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || !j.drawn || !reflect.DeepEqual(before, stage42NativeState(g)) {
		p.checker.err = fmt.Errorf("stage Draw changed native producer: %s", g.loadErr)
		return
	}
	f := j.hdFrame
	if f == nil || len(f.Terrain) == 0 || len(f.Units) != 2 || f.HUDUnit == nil || f.Units[0].ColourImage == nil || f.HUDUnit.Unit.ColourImage == nil {
		p.checker.err = fmt.Errorf("real HD layers absent")
		return
	}
	pixels, ok := p.checker.checkWorld(g, f, healHDCPUWorld(f, p.checker.set), screen, p.phase)
	if !ok {
		return
	}
	digest := sha256.Sum256(pixels)
	limit := 240
	if p.phase == "palette" {
		limit = 60
	}
	if p.draws == 0 || p.draws == limit/2 || p.draws == limit-1 {
		if err := g.applyTheme(nil); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.checker.err = fmt.Errorf("native theme retained stage HD")
			return
		}
		if err := g.applyTheme(p.checker.set); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		again, ok := p.checker.checkWorld(g, j.hdFrame, healHDCPUWorld(j.hdFrame, p.checker.set), screen, "F2")
		if !ok {
			return
		}
		if sha256.Sum256(again) != digest || !reflect.DeepEqual(before, stage42NativeState(g)) {
			p.checker.err = fmt.Errorf("F2 changed pixels or source")
			return
		}
		p.switches++
		if err := p.save(fmt.Sprintf("%s-%03d.png", p.phase, p.draws), pixels); err != nil {
			p.checker.err = err
			return
		}
	}
	if p.controlScreen == nil {
		p.controlScreen = ebiten.NewImage(640, 400)
	}
	p.control.Draw(p.controlScreen)
	if !p.control.nativeCh23Loop.drawn || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(p.control)) {
		p.checker.err = fmt.Errorf("native control diverged at %s/%d", p.phase, p.draws)
		return
	}
	p.rows = append(p.rows, map[string]any{"phase": p.phase, "draw": p.draws, "row_shift": g.nativeCh23State.rowShift, "raw_esi": j.rawESI, "rng": g.nativeRNGState, "final_sha256": fmt.Sprintf("%x", digest), "native_vga_sha256": fmt.Sprintf("%x", sha256.Sum256(g.nativeMapVGA)), "dac_sha256": fmt.Sprintf("%x", sha256.Sum256(g.nativeMapDAC))})
	p.draws++
	for tick := 0; tick < 3; tick++ {
		p.now = p.now.Add(time.Second / 60)
		g.stepNativeCh23LoopAt(p.now)
		p.control.stepNativeCh23LoopAt(p.now)
		if tick < 2 && (g.nativeCh23Loop != j || j.waitFrames != 2-tick || !j.drawn) {
			p.checker.err = fmt.Errorf("three Update wait changed")
			return
		}
	}
	g.frame++
	p.control.frame++
	if g.nativeCh23Loop != nil {
		return
	}
	if p.draws != limit || p.control.nativeCh23Loop != nil || p.continued != p.controlContinued || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(p.control)) {
		p.checker.err = fmt.Errorf("stage completion differs: %s", g.loadErr)
		return
	}
	// Ordinary Game.Draw must resume the same native stage while retaining HD.
	g.Draw(screen)
	p.control.Draw(p.controlScreen)
	if !g.modernHDWorldVisible || g.modernHDMapPresented == nil || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(p.control)) {
		p.checker.err = fmt.Errorf("normal HD return differs: %s / %s", g.loadErr, p.control.loadErr)
		return
	}
	pixels, ok = p.checker.checkWorld(g, g.modernHDMapPresented, healHDCPUWorld(g.modernHDMapPresented, p.checker.set), screen, "return")
	if !ok {
		return
	}
	p.returns = append(p.returns, map[string]any{"phase": p.phase, "draws": p.draws, "continuations": p.continued, "final_sha256": fmt.Sprintf("%x", sha256.Sum256(pixels))})
	if err := p.save(p.phase+"-return.png", pixels); err != nil {
		p.checker.err = err
		return
	}
	if p.phase == "initial" {
		p.checker.err = p.start("palette")
	} else {
		p.phase = "done"
	}
}

func TestModernHDCh23StageGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_CH23_STAGE_GPU") != "1" {
		t.Skip("dedicated stage GPU process")
	}
	out := os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &stage42GPUProbe{out: out, now: time.Unix(1000, 0)}
	p.g, p.control = stage42ActualGame(t, &p.now), stage42ActualGame(t, &p.now)
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_UNIT_PRESENT_CATALOG"), os.Getenv("FD2_TEST_UNIT_PRESENT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	p.checker.set = set
	p.g.modernStoryPortraits = set
	if err := p.start("initial"); err != nil {
		t.Fatal(err)
	}
	ebiten.SetWindowSize(1920, 1200)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.checker.err != nil {
		t.Fatal(p.checker.err)
	}
	if len(p.rows) != 300 || p.switches != 6 || p.continued != 2 {
		t.Fatal("incomplete sequence")
	}
	raw, err := json.MarshalIndent(map[string]any{"status": "PASS-CH23-STAGE-HD-GAME", "fixed_clock_unix": 1000, "native_seed": 7, "rgba_tolerance": 2, "f2_round_trips": p.switches, "continuations": p.continued, "native_control_unchanged": true, "map_index": 23, "rows": p.rows, "returns": p.returns, "scope": "actual map23 separated banks, two controlled actors; RUNTIME-E1, remake native-theme control, no original-EXE or normal-player parity"}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
