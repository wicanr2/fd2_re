package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func indexedHDNativeState(g *Game) any {
	state := storyHDNativeSnapshot(g)
	j := g.indexedTransition
	if j == nil {
		return []any{state, nil}
	}
	identity, err := indexedHDInputIdentity(j.input)
	if err != nil {
		panic(err)
	}
	raw, err := json.Marshal([]any{identity, j.spec, j.schedule, j.palette, j.phase, j.frame, j.paletteStep, j.tailTicks})
	if err != nil {
		panic(err)
	}
	h := sha256.New()
	h.Write(raw)
	for _, part := range [][]byte{j.work, j.vga, j.dac, j.baseline} {
		h.Write(part)
	}
	for _, series := range [][][]byte{j.frameWork, j.frameVGA} {
		for _, part := range series {
			h.Write(part)
		}
	}
	return []any{state, fmt.Sprintf("%x", h.Sum(nil))}
}

// Independent RGBA8 channel arithmetic checks the GPU's premultiplied float
// addition, including alpha. It does not call its shader or production helper.
func indexedHDCPUWorld(frame *modernHDMapFrame, set *modernStoryPortraitSet) *image.RGBA {
	out := healHDCPUWorld(frame, set)
	for i := 0; i < len(out.Pix); i += 4 {
		a := int(out.Pix[i+3])
		for c := 0; c < 3; c++ {
			value := int(out.Pix[i+c]) + int(math.Round(float64(frame.PaletteAdd6*a)/63))
			out.Pix[i+c] = byte(min(a, value))
		}
	}
	return out
}

func indexedHDActualGame(t *testing.T, set *modernStoryPortraitSet) *Game {
	t.Helper()
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	g.nativeRNGState = 7
	source := g.st
	g.storyActors = make([]battle.Unit, len(source.Units))
	for i, unit := range source.Units {
		g.storyActors[i] = *unit
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
	if err := g.composeNativeStoryMapFrame(); err != nil {
		t.Fatal(err)
	}
	g.modernStoryPortraits = set
	return g
}

type indexedHDGPUCase struct {
	name, source                 string
	g, control                   *Game
	continued, controlContinued  *int
	draws, switches, tailUpdates int
	returnSHA                    string
}

type indexedHDGPUProbe struct {
	checker       unitPresentGPUProbe
	cases         []indexedHDGPUCase
	index, alpha  int
	out           string
	controlScreen *ebiten.Image
	cachedFrame   *modernHDMapFrame
	cachedWant    *image.RGBA
	rows, sources []map[string]any
}

func (p *indexedHDGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *indexedHDGPUProbe) Update() error {
	if p.checker.err != nil || p.index >= len(p.cases) {
		return ebiten.Termination
	}
	return nil
}

func (p *indexedHDGPUProbe) alphaWorld(screen *ebiten.Image) {
	delta := []int{0, 16, 62}[p.alpha]
	picture := image.NewRGBA(image.Rect(0, 0, 12, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 12; x++ {
			alpha := []uint8{0, 128, 255}[x/4]
			picture.Set(x, y, color.NRGBA{100, 70, 30, alpha})
		}
	}
	f := &modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200)), UI: image.NewRGBA(image.Rect(0, 0, 320, 200)),
		OverlaysAboveUI: true, PaletteAdd6: delta, Terrain: []modernHDTerrainDraw{{Image: picture, Rect: picture.Bounds().Add(image.Pt(100, 100))}}}
	g := p.cases[0].g
	if !g.drawPreparedModernHDMapFrame(screen, f, false) {
		p.checker.err = fmt.Errorf("alpha source absent: %s", g.loadErr)
		return
	}
	want := indexedHDCPUWorld(f, p.checker.set)
	pixels := make([]byte, len(want.Pix))
	g.modernHDMapGPU.World.ReadPixels(pixels)
	if !p.checker.compare(pixels, want.Pix, "six-bit addition alpha") {
		return
	}
	p.sources = append(p.sources, map[string]any{"delta": delta, "alphas": []int{0, 128, 255}, "world_sha256": fmt.Sprintf("%x", sha256.Sum256(pixels))})
	p.alpha++
}

func (p *indexedHDGPUProbe) Draw(screen *ebiten.Image) {
	if p.checker.err != nil || p.index >= len(p.cases) {
		return
	}
	if p.alpha < 3 {
		p.alphaWorld(screen)
		return
	}
	c := &p.cases[p.index]
	g, j := c.g, c.g.indexedTransition
	if j == nil {
		p.checker.err = fmt.Errorf("%s lost owner before completion", c.name)
		return
	}
	phase, number := "pass", j.frame
	if j.phase == nativeTransitionTail {
		phase, number = "tail", j.tailTicks
	} else if j.phase == nativeTransitionPalette {
		phase, number = "palette", j.paletteStep
	}
	key := fmt.Sprintf("%s/%d", phase, number)
	before := indexedHDNativeState(g)
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || g.modernHDMapPresented == nil || !j.drawn || !reflect.DeepEqual(before, indexedHDNativeState(g)) {
		p.checker.err = fmt.Errorf("%s/%s draw or native isolation failed: %s", c.name, key, g.loadErr)
		return
	}
	frame := g.modernHDMapPresented
	if p.cachedFrame != frame {
		p.cachedFrame, p.cachedWant = frame, indexedHDCPUWorld(frame, p.checker.set)
	}
	pixels, ok := p.checker.checkWorld(g, frame, p.cachedWant, screen, c.name+"/"+key)
	if !ok {
		return
	}
	digest := sha256.Sum256(pixels)
	f2 := phase == "pass" && (number == 0 || number == 4 || number == 8) ||
		phase == "tail" && (number == 30 || number == 0) || phase == "palette" && (number == 0 || number == 15 || number == 31)
	if f2 {
		if err := g.applyTheme(nil); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.checker.err = fmt.Errorf("native theme retained indexed HD")
			return
		}
		if err := g.applyTheme(p.checker.set); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		again, ok := p.checker.checkWorld(g, frame, p.cachedWant, screen, c.name+" F2/"+key)
		if !ok {
			return
		}
		if digest != sha256.Sum256(again) || !reflect.DeepEqual(before, indexedHDNativeState(g)) {
			p.checker.err = fmt.Errorf("F2 changed indexed source, progress or pixels")
			return
		}
		c.switches++
	}
	if p.controlScreen == nil {
		p.controlScreen = ebiten.NewImage(640, 400)
	}
	p.controlScreen.Clear()
	c.control.Draw(p.controlScreen)
	if !reflect.DeepEqual(indexedHDNativeState(g), indexedHDNativeState(c.control)) || !c.control.indexedTransition.drawn || *c.continued != *c.controlContinued {
		p.checker.err = fmt.Errorf("%s/%s native-theme control diverged", c.name, key)
		return
	}
	if phase == "palette" {
		for i, value := range j.dac {
			if value != byte(min(63, int(j.baseline[i])+number*2)) {
				p.checker.err = fmt.Errorf("baseline DAC addition changed at %d", i)
				return
			}
		}
	}
	p.rows = append(p.rows, map[string]any{"case": c.name, "key": key, "phase": phase, "number": number,
		"final_sha256": fmt.Sprintf("%x", digest), "dac_sha256": fmt.Sprintf("%x", sha256.Sum256(j.dac)),
		"native_vga_sha256": fmt.Sprintf("%x", sha256.Sum256(j.vga)), "palette_add6": frame.PaletteAdd6, "rng": g.nativeRNGState})
	if phase == "pass" && (number == 0 || number == 8) || phase == "palette" && number == 15 {
		if err := p.save(c.name+"-"+phase+fmt.Sprintf("-%02d.png", number), pixels); err != nil {
			p.checker.err = err
			return
		}
	}
	c.draws++
	if phase == "tail" {
		c.tailUpdates++
	}
	for _, game := range []*Game{g, c.control} {
		game.stepNativeIndexedTransition()
		game.frame++
	}
	if g.indexedTransition == nil {
		if c.control.indexedTransition != nil || *c.continued != 1 || *c.controlContinued != 1 || c.switches != 8 || c.tailUpdates != 31 || c.draws != 72 {
			p.checker.err = fmt.Errorf("%s ended at a different boundary", c.name)
			return
		}
		before := indexedHDNativeState(g)
		g.Draw(screen)
		if !g.modernHDWorldVisible || g.modernHDMapPresented == nil || !reflect.DeepEqual(before, indexedHDNativeState(g)) {
			p.checker.err = fmt.Errorf("return world missing: %s", g.loadErr)
			return
		}
		pixels, ok := p.checker.checkWorld(g, g.modernHDMapPresented, healHDCPUWorld(g.modernHDMapPresented, p.checker.set), screen, c.name+" return")
		if !ok {
			return
		}
		c.returnSHA = fmt.Sprintf("%x", sha256.Sum256(pixels))
		if err := p.save(c.name+"-return.png", pixels); err != nil {
			p.checker.err = err
			return
		}
		p.index++
		p.cachedFrame, p.cachedWant = nil, nil
	}
}

func (p *indexedHDGPUProbe) save(name string, pixels []byte) error {
	f, err := os.Create(filepath.Join(p.out, name))
	if err != nil {
		return err
	}
	picture := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	copy(picture.Pix, pixels)
	err = png.Encode(f, picture)
	closed := f.Close()
	if err == nil {
		err = closed
	}
	return err
}

func TestModernHDIndexedTransitionGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_INDEXED_TRANSITION_GPU") != "1" {
		t.Skip("dedicated indexed-transition GPU process")
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
	p := &indexedHDGPUProbe{out: out, checker: unitPresentGPUProbe{set: set}}
	for _, name := range []string{"ch21", "ch22", "ch27", "geometry"} {
		g, control := indexedHDActualGame(t, set), indexedHDActualGame(t, nil)
		a, b := new(int), new(int)
		spec := nativeIndexedTransitionSpecForTest()
		source := map[string]string{"ch21": "0x245ce", "ch22": "0x336e5", "ch27": "0x33ce2"}[name]
		if source != "" {
			spec.CursorSource = "native_relative_cursor"
			spec.CursorYOffset = map[string]int{"ch21": 3, "ch22": 5, "ch27": 5}[name]
			if name == "ch27" {
				spec.CursorXOffset = 6
			}
		} else {
			// Direct geometry fixture only; it does not claim a new caller.
			spec.TileX, spec.TileY = 156, 120
		}
		if err := g.startNativeIndexedTransition(spec, source, func() { *a++ }); err != nil {
			t.Fatal(name, err)
		}
		if err := control.startNativeIndexedTransition(spec, source, func() { *b++ }); err != nil {
			t.Fatal(name, err)
		}
		p.cases = append(p.cases, indexedHDGPUCase{name: name, source: source, g: g, control: control, continued: a, controlContinued: b})
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
		cases[i] = map[string]any{"name": c.name, "source": c.source, "draws": c.draws, "f2_round_trips": c.switches,
			"tail_updates": c.tailUpdates, "continuations": *c.continued, "native_control_unchanged": true, "return_final_sha256": c.returnSHA}
	}
	raw, err := json.MarshalIndent(map[string]any{"status": "PASS-INDEXED-TRANSITION-HD-GAME", "fixed_clock_unix": 1000,
		"native_seed": 7, "rgba_tolerance": 2, "cases": cases, "rows": p.rows, "alpha_sources": p.sources,
		"scope": "three existing caller bindings plus one direct geometry fixture; controlled remake native-theme control, no new original-executable parity"}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
