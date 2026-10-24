package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type storyMotionGPUCase struct {
	name             string
	g, control       *Game
	continued        *int
	controlContinued *int
	draws, switches  int
}

type storyMotionGPUProbe struct {
	checker unitPresentGPUProbe
	cases   []storyMotionGPUCase
	index   int
	step    int
	out     string
	rows    []map[string]any
}

func (p *storyMotionGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *storyMotionGPUProbe) Update() error {
	if p.checker.err != nil || p.index >= len(p.cases) {
		return ebiten.Termination
	}
	return nil
}

func (p *storyMotionGPUProbe) Draw(screen *ebiten.Image) {
	if p.checker.err != nil || p.index >= len(p.cases) {
		return
	}
	c := &p.cases[p.index]
	g := c.g
	before := storyHDNativeSnapshot(g)
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || g.modernHDMapPresented == nil || !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
		p.checker.err = fmt.Errorf("%s/%d HD Draw or isolation failed: %s", c.name, p.step, g.loadErr)
		return
	}
	frame := g.modernHDMapPresented
	want := healHDCPUWorld(frame, p.checker.set)
	pixels, ok := p.checker.checkWorld(g, frame, want, screen, fmt.Sprintf("%s/%d", c.name, p.step))
	if !ok {
		return
	}
	digest := sha256.Sum256(pixels)
	if p.step == 0 || p.step == 6 || p.step == 13 {
		if err := g.applyTheme(nil); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.checker.err = fmt.Errorf("native theme retained HD")
			return
		}
		if err := g.applyTheme(p.checker.set); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		again, ok := p.checker.checkWorld(g, g.modernHDMapPresented, want, screen, c.name+" F2")
		if !ok {
			return
		}
		if digest != sha256.Sum256(again) || !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
			p.checker.err = fmt.Errorf("%s F2 changed story source or pixels", c.name)
			return
		}
		c.switches++
	}
	// Native-control controllers receive identical steps, with no HD resources.
	if !reflect.DeepEqual(storyHDNativeSnapshot(g), storyHDNativeSnapshot(c.control)) || *c.continued != *c.controlContinued {
		p.checker.err = fmt.Errorf("%s/%d native controller diverged", c.name, p.step)
		return
	}
	textureLimit := 1 + len(frame.Terrain)
	for _, fg := range frame.Foreground {
		if fg.HDImage != nil {
			textureLimit++
		}
	}
	if len(g.modernHDMapGPU.TerrainImages) > textureLimit {
		p.checker.err = fmt.Errorf("story terrain textures accumulated")
		return
	}
	p.rows = append(p.rows, map[string]any{"case": c.name, "step": p.step, "final_sha256": fmt.Sprintf("%x", digest),
		"camera_x": g.camX, "camera_y": g.camY, "actor_x": float64(g.storyActors[0].X*24) + g.storyActors[0].OffX,
		"actor_y": float64(g.storyActors[0].Y*24) + g.storyActors[0].OffY, "units": len(frame.Units), "foreground": len(frame.Foreground)})
	if p.step == 6 {
		path := filepath.Join(p.out, c.name+".png")
		f, err := os.Create(path)
		if err == nil {
			pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
			copy(pic.Pix, pixels)
			err = png.Encode(f, pic)
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
		}
		if err != nil {
			p.checker.err = err
			return
		}
	}
	c.draws++
	if p.step == 13 {
		if *c.continued != 1 || c.switches != 3 || len(g.storyWalks) != 0 || g.camPan != nil {
			p.checker.err = fmt.Errorf("%s controller did not finish exactly once", c.name)
			return
		}
		p.index++
		p.step = 0
		return
	}
	for _, game := range []*Game{g, c.control} {
		game.frame++
		game.stepStoryWalks()
		game.stepCamPan()
	}
	p.step++
}

func storyMotionActualGame(t *testing.T, set *modernStoryPortraitSet, name string, continued *int) *Game {
	t.Helper()
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	g.nativeRNGState = 7
	source := g.st
	g.storyActors = make([]battle.Unit, len(source.Units))
	for i, u := range source.Units {
		g.storyActors[i] = *u
	}
	g.storyBG, g.st = true, nil
	g.storyNativeMapView = battle.NativeMapViewState{CameraX: 2, CameraY: 3, CursorX: 6, CursorY: 5, VisibleCursorX: 4, VisibleCursorY: 2}
	g.hasStoryNativeMapView = true
	g.camX, g.camY, g.curX, g.curY = 48, 72, 6, 5
	g.storyActors[0].OnField = true
	g.storyActors[0].SetMapPlacement(6, 5, 0)
	if err := g.materializeNativeStoryMapState(source); err != nil {
		t.Fatal(err)
	}
	if name == "pan" {
		// This controlled fixture adds the real FDOTHER 16/17 surface to the
		// map-zero carrier. It tests the renderer, not chapter-18 reachability.
		var err error
		g.nativeMapAssets.Parallax, err = fdother.LoadSeparatedNativeMapParallax(separatedAssetPath("surfaces"))
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := g.composeNativeStoryMapFrame(); err != nil {
		t.Fatal(err)
	}
	g.modernStoryPortraits = set
	finish := func() { *continued++ }
	if name == "pan" {
		g.camPan = &camPanJob{fromX: 48, fromY: 72, toX: 96, toY: 96, frames: 12, then: finish}
		return g
	}
	dest := map[string]image.Point{"east": {9, 5}, "west": {3, 5}, "north": {6, 3}, "south": {6, 8},
		"turn-x": {9, 6}, "turn-y": {7, 8}, "follow": {6, 2}}[name]
	free := 0
	if name == "follow" {
		free = 1 // unitY-cameraY starts at 2: the first up-step stays in the safe band.
	}
	g.storyWalks = []*storyWalkJob{{actor: 0, fromX: 6, fromY: 5, toX: dest.X, toY: dest.Y, frames: 12,
		finalDir: 0, then: finish, scrollFollow: name == "follow", scrollFree: free, fromCamY: 72}}
	return g
}

func TestModernHDStoryMotionGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_STORY_MOTION_GPU") != "1" {
		t.Skip("dedicated story-motion GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	out := os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh capture output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_UNIT_PRESENT_CATALOG"), os.Getenv("FD2_TEST_UNIT_PRESENT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	p := &storyMotionGPUProbe{out: out, checker: unitPresentGPUProbe{set: set}}
	for _, name := range []string{"east", "west", "north", "south", "turn-x", "turn-y", "pan", "follow"} {
		a, b := new(int), new(int)
		p.cases = append(p.cases, storyMotionGPUCase{name: name, continued: a, controlContinued: b,
			g: storyMotionActualGame(t, set, name, a), control: storyMotionActualGame(t, nil, name, b)})
	}
	ebiten.SetWindowSize(1920, 1200)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.checker.err != nil {
		t.Fatal(p.checker.err)
	}
	cases := make([]map[string]any, len(p.cases))
	for i, c := range p.cases {
		cases[i] = map[string]any{"name": c.name, "draws": c.draws, "continuations": *c.continued,
			"f2_round_trips": c.switches, "native_control_unchanged": true}
	}
	raw, err := json.MarshalIndent(map[string]any{"status": "PASS-STORY-MOTION-HD-GAME", "fixed_clock_unix": 1000,
		"native_seed": 7, "rgba_tolerance": 2, "cases": cases, "rows": p.rows,
		"parallax_fixture": "pan: original FDOTHER 16/17 on map-zero carrier; controlled RUNTIME-E1"}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
