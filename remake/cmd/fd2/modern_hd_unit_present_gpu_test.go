package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	xdraw "golang.org/x/image/draw"
)

// The native target/destination confirmation handlers own the transaction.
// Fixture records only make the item and its legal destinations reproducible.
func unitPresentHDRelocationGame(t *testing.T, set *modernStoryPortraitSet) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	actor, target, enemy := g.st.Units[0], g.st.Units[1], g.st.Units[4]
	actor.Camp, actor.OnField, actor.NativeIdentity, actor.HasNativeIdentity = battle.Own, true, 24, true
	actor.NativeRecordByte5, actor.HasNativeRecordByte5 = 0, true
	actor.MP, actor.MaxMP = 30, 40
	actor.Inventory = []int{101}
	actor.Equipped = []bool{false}
	actor.InventorySlots = []int{101, 255, 255, 255, 255, 255, 255, 255}
	actor.NativeInventoryFlags = []int{0, 128, 128, 128, 128, 128, 128, 128}
	target.Camp, target.OnField, target.NativeIdentity, target.HasNativeIdentity = battle.Own, true, 1, true
	target.NativeRecordClass, target.HasNativeRecordClass = 1, true
	target.NativeRecordByte5, target.HasNativeRecordByte5 = 0, true
	target.InventorySlots = []int{255, 255, 255, 255, 255, 255, 255, 255}
	target.NativeInventoryFlags = []int{128, 128, 128, 128, 128, 128, 128, 128}
	actor.SetMapPlacement(0, 0, 3)
	target.SetMapPlacement(1, 0, 1)
	enemy.SetMapPlacement(12, 7, 1)
	for _, u := range []*battle.Unit{actor, target, enemy} {
		if err := u.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
	}
	g.st.Units = []*battle.Unit{actor, target, enemy}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for i := range book {
		book[i].ID = i
	}
	book[23].MPCost = 20
	g.nativeCommandBook, g.st.NativeCommandBook = book, book
	g.sel, g.moved, g.itemOpen = actor, true, true
	g.st.NativeCompositionEventBytes = make([]byte, g.st.W*g.st.H)
	g.st.NativeTerrainMoveCodes = make([]byte, g.st.W*g.st.H)
	for i := range g.st.NativeTerrainMoveCodes {
		g.st.NativeTerrainMoveCodes[i] = 2
	}
	g.st.NativeTileBlitModes[target.Y*g.st.W+target.X] = 0
	g.m.NativeTileBlitModes[target.Y*g.st.W+target.X] = 0
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CursorX: 1, VisibleCursorX: 1}); err != nil {
		t.Fatal(err)
	}
	g.camX, g.camY, g.curX, g.curY = 0, 0, 1, 0
	g.modernStoryPortraits = set
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if targeting, err := g.beginNativeTargetItem(0, 101); err != nil || !targeting {
		t.Fatalf("target input: %t %v", targeting, err)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if applied, err := g.applyNativeTargetItem(target); err != nil || applied || !g.nativeItemRelocating {
		t.Fatalf("target confirm: %t %v", applied, err)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if applied, err := g.applyNativeRelocationDestination(8, 7); err != nil || !applied {
		t.Fatalf("destination confirm: %t %v", applied, err)
	}
	return g, actor, target
}

type unitPresentGPUCase struct {
	name                        string
	g, control                  *Game
	actor, target               *battle.Unit
	continued, controlContinued *int
	legs                        map[*nativeUnitPresentJob]int
	after                       map[*nativeUnitPresentJob][]byte
	f2                          map[string]bool
	shots                       map[string]bool
	draws                       int
	focusSwitched               bool
	returnSHA                   string
	finalDestination            *image.Point
}

type unitPresentGPUProbe struct {
	set           *modernStoryPortraitSet
	cases         []unitPresentGPUCase
	index, source int
	out           string
	err           error
	rows, sources []map[string]any
	cachedFrame   *modernHDMapFrame
	cachedWant    *image.RGBA
	final         *ebiten.Image
	controlScreen *ebiten.Image
	presentation  *modernHDPresentation
}

func unitPresentGPUState(g *Game) any {
	if g.st == nil {
		return storyHDNativeSnapshot(g)
	}
	return compoundMapUnitSnapshot(g)
}

func unitPresentGPUExpected(name string) (int, int) {
	switch name {
	case "command23":
		return 20, 2
	case "staging", "story-staging":
		return 10, 1
	default:
		return 9, 1
	}
}

func (p *unitPresentGPUProbe) Update() error {
	if p.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *unitPresentGPUProbe) Layout(int, int) (int, int) { return 640, 400 }

func (p *unitPresentGPUProbe) compare(pixels, want []byte, label string) bool {
	for i, v := range want {
		d := int(pixels[i]) - int(v)
		if d < -2 || d > 2 {
			p.err = fmt.Errorf("%s byte%d got%d want%d", label, i, pixels[i], v)
			return false
		}
	}
	return true
}

func (p *unitPresentGPUProbe) checkWorld(g *Game, frame *modernHDMapFrame, want *image.RGBA, screen *ebiten.Image, label string) ([]byte, bool) {
	pixels := make([]byte, 1920*1200*4)
	g.modernHDMapGPU.World.ReadPixels(pixels)
	if !p.compare(pixels, want.Pix, label+" world") {
		return nil, false
	}
	finalWant := image.NewRGBA(want.Bounds())
	copy(finalWant.Pix, want.Pix)
	if !frame.OverlaysAboveUI {
		xdraw.NearestNeighbor.Scale(finalWant, finalWant.Bounds(), frame.UI, frame.UI.Bounds(), draw.Over, nil)
	}
	if p.final == nil {
		p.final = ebiten.NewImage(1920, 1200)
		p.presentation = &modernHDPresentation{}
	}
	p.presentation.Game = g
	p.final.Clear()
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(pixels)
	return pixels, p.compare(pixels, finalWant.Pix, label+" final")
}

func (p *unitPresentGPUProbe) f2(c *unitPresentGPUCase, screen *ebiten.Image, key string, frame *modernHDMapFrame, want *image.RGBA, digest [32]byte) bool {
	units, rng := unitPresentGPUState(c.g), c.g.nativeRNGState
	work, vga := append([]byte(nil), c.g.nativeMapWork...), append([]byte(nil), c.g.nativeMapVGA...)
	job, palette, focus := c.g.nativeUnitPresent, c.g.nativeModifierPresentation, c.g.focusJob
	if err := c.g.applyTheme(nil); err != nil {
		p.err = err
		return false
	}
	c.g.Draw(screen)
	if c.g.modernHDWorldVisible {
		p.err = fmt.Errorf("native F2 retained HD")
		return false
	}
	if err := c.g.applyTheme(p.set); err != nil {
		p.err = err
		return false
	}
	c.g.Draw(screen)
	if c.g.nativeUnitPresent != job || c.g.nativeModifierPresentation != palette || c.g.focusJob != focus || rng != c.g.nativeRNGState || !reflect.DeepEqual(units, unitPresentGPUState(c.g)) || !bytes.Equal(work, c.g.nativeMapWork) || !bytes.Equal(vga, c.g.nativeMapVGA) {
		p.err = fmt.Errorf("F2 changed transaction/owner")
		return false
	}
	pixels, ok := p.checkWorld(c.g, frame, want, screen, "F2/"+key)
	if !ok {
		return false
	}
	if sha256.Sum256(pixels) != digest {
		p.err = fmt.Errorf("F2 changed final pixels")
		return false
	}
	c.f2[key] = true
	return true
}

// Independent byte-address projection uses the actual full after-world GPU
// result as the source. It never reads production strip patches or their CPU
// rasterizer, so an incorrect patch origin/blend cannot validate itself.
func (p *unitPresentGPUProbe) bridgeWant(c *unitPresentGPUCase, j *nativeUnitPresentJob, screen *ebiten.Image) *image.RGBA {
	g := c.g
	after := c.after[j]
	if after == nil {
		if j.hdCachedTheme != p.set || j.hdCachedBridge == nil || !g.drawPreparedModernHDMapFrame(screen, j.hdCachedBridge, false) {
			p.err = fmt.Errorf("after-world GPU source absent")
			return nil
		}
		after = make([]byte, 1920*1200*4)
		g.modernHDMapGPU.World.ReadPixels(after)
		c.after[j] = after
	}
	want := healHDCPUWorld(j.hdFrames[16], p.set)
	s := j.hdSource
	camX, camY := s.Before.Frame.CameraX, s.Before.Frame.CameraY
	x, y := int(s.Call.VisualX)-camX, int(s.Call.VisualY)-camY
	src := 0x8088 + x*24 + y*24*456
	dst := x*24 + y*24*320
	if int(s.Call.VisualY) != camY {
		src -= 6 * 456
		dst -= 6 * 320
	}
	for row := 0; row <= j.frame-17; row++ {
		for col := 0; col < 24; col++ {
			si, di := src+row*456+col, dst+row*320+col
			sx, sy := si%456-0x8088%456, si/456-0x8088/456
			dx, dy := di%320, di/320
			for yy := 0; yy < 6; yy++ {
				for xx := 0; xx < 6; xx++ {
					var value color.Color = j.palette[j.workFrames[j.frame][si]]
					if sx >= 0 && sx < 312 && sy >= 0 && sy < 192 && dx < 312 && dy < 192 {
						at := ((sy*6+yy)*1920 + sx*6 + xx) * 4
						value = color.RGBA{after[at], after[at+1], after[at+2], after[at+3]}
					}
					want.Set(dx*6+xx, dy*6+yy, value)
				}
			}
		}
	}
	return want
}

func (p *unitPresentGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == len(p.cases) {
		return
	}
	c := &p.cases[p.index]
	g, control := c.g, c.control
	if p.controlScreen == nil {
		p.controlScreen = ebiten.NewImage(640, 400)
	}
	if p.source < 22 {
		entry, pos := 114+p.source/2, p.source%2
		pic := p.set.mapEffects.frames[entry]
		point := image.Pt(600, 480)
		if pos == 1 {
			point = image.Pt(1820, 800)
		}
		f := &modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200)), UI: image.NewRGBA(image.Rect(0, 0, 320, 200)), Viewport: image.Rect(0, 0, 1872, 1152), OverlaysAboveUI: true, Terrain: []modernHDTerrainDraw{{Image: pic, Rect: pic.Bounds().Add(point)}}}
		draw.Draw(f.Background, f.Background.Bounds(), image.NewUniform(color.RGBA{19, 27, 39, 255}), image.Point{}, draw.Src)
		if !g.drawPreparedModernHDMapFrame(screen, f, false) {
			p.err = fmt.Errorf("source draw rejected")
			return
		}
		pixels, ok := p.checkWorld(g, f, healHDCPUWorld(f, p.set), screen, fmt.Sprintf("source/%d/%d", entry, pos))
		if !ok {
			return
		}
		p.sources = append(p.sources, map[string]any{"entry": entry, "placement": pos, "gpu_sha256": fmt.Sprintf("%x", sha256.Sum256(pixels))})
		p.source++
		return
	}
	if g.focusJob != nil {
		if !c.focusSwitched {
			focus, job := g.focusJob, g.focusJob.unitPresent
			units, rng := unitPresentGPUState(g), g.nativeRNGState
			if err := g.applyTheme(nil); err != nil {
				p.err = err
				return
			}
			if job.hdFrames != nil {
				p.err = fmt.Errorf("pending focus retained HD")
				return
			}
			if err := g.applyTheme(p.set); err != nil {
				p.err = err
				return
			}
			if g.focusJob != focus || len(job.hdFrames) == 0 || rng != g.nativeRNGState || !reflect.DeepEqual(units, unitPresentGPUState(g)) {
				p.err = fmt.Errorf("pending focus F2 mutated state")
				return
			}
			c.f2["focus-preflight"] = true
			c.focusSwitched = true
		}
		g.frame++
		control.frame++
		g.Draw(screen)
		p.controlScreen.Clear()
		control.Draw(p.controlScreen)
		g.stepFocusUnit()
		control.stepFocusUnit()
		return
	}
	job, palette := g.nativeUnitPresent, g.nativeModifierPresentation
	if job == nil && palette == nil {
		p.err = fmt.Errorf("missing owner before completion")
		return
	}
	phase, key := "palette", ""
	f2point, shot := false, false
	if job != nil {
		leg, ok := c.legs[job]
		if !ok {
			leg = len(c.legs)
			c.legs[job] = leg
		}
		rows := len(job.vgaFrames) - 27
		i := job.frame
		phase = "intro"
		if i >= 11 {
			phase = "contract"
		}
		if i >= 17 {
			phase = "bridge"
		}
		if i >= 17+rows {
			phase = "release"
		}
		key = fmt.Sprintf("leg%d/%s/%d", leg, phase, i)
		f2point = i == 0 || i == 10 || i == 11 || i == 16 || i == 17 || i == 17+rows/2 || i == 16+rows || i == 17+rows || i == len(job.vgaFrames)-1
		shot = i == 0 || i == 10 || i == 16 || i == 17+rows/2 || i == len(job.vgaFrames)-1
	} else {
		key = fmt.Sprintf("palette/%d", palette.phase)
		f2point = palette.phase == 0 || palette.phase == 7
		shot = palette.phase == 0
	}
	var want *image.RGBA
	if job != nil && phase == "bridge" {
		want = p.bridgeWant(c, job, screen)
		if p.err != nil {
			return
		}
	}
	g.frame++
	control.frame++
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || g.modernHDMapPresented == nil {
		p.err = fmt.Errorf("%s Game.Draw absent: %s", key, g.loadErr)
		return
	}
	f := g.modernHDMapPresented
	if want == nil {
		if p.cachedFrame == f {
			want = p.cachedWant
		} else {
			want = healHDCPUWorld(f, p.set)
			p.cachedFrame, p.cachedWant = f, want
		}
	}
	pixels, ok := p.checkWorld(g, f, want, screen, c.name+"/"+key)
	if !ok {
		return
	}
	digest := sha256.Sum256(pixels)
	if f2point && !c.f2[key] {
		if !p.f2(c, screen, key, f, want, digest) {
			return
		}
	}
	if shot && !c.shots[key] {
		name := fmt.Sprintf("%s-%03d-%s.png", c.name, len(c.shots), phase)
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, pixels)
		file, err := os.Create(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		err = png.Encode(file, pic)
		closeErr := file.Close()
		if err != nil {
			p.err = err
			return
		}
		if closeErr != nil {
			p.err = closeErr
			return
		}
		c.shots[key] = true
	}
	p.controlScreen.Clear()
	control.Draw(p.controlScreen)
	if palette != nil {
		g.stepNativeCommandModifierPresentation()
		control.stepNativeCommandModifierPresentation()
	} else {
		g.stepNativeUnitPresent()
		control.stepNativeUnitPresent()
	}
	if !reflect.DeepEqual(unitPresentGPUState(g), unitPresentGPUState(control)) || g.nativeRNGState != control.nativeRNGState || !bytes.Equal(g.nativeMapWork, control.nativeMapWork) || !bytes.Equal(g.nativeMapVGA, control.nativeMapVGA) {
		p.err = fmt.Errorf("%s native control diverged", key)
		return
	}
	c.draws++
	p.rows = append(p.rows, map[string]any{"case": c.name, "phase": phase, "key": key, "rng": g.nativeRNGState, "final_sha256": fmt.Sprintf("%x", digest)})
	if c.draws > 900 {
		p.err = fmt.Errorf("unit present exceeded bounded scheduler")
		return
	}
	if g.nativeUnitPresent == nil && g.nativeModifierPresentation == nil && g.focusJob == nil {
		wantF2, wantLegs := unitPresentGPUExpected(c.name)
		if len(c.f2) != wantF2 || len(c.legs) != wantLegs || *c.continued != 1 || *c.controlContinued != 1 {
			p.err = fmt.Errorf("%s incomplete F2/legs/continuation: %d/%d/%d/%d", c.name, len(c.f2), len(c.legs), *c.continued, *c.controlContinued)
			return
		}
		destination := image.Pt(8, 7)
		if c.finalDestination != nil {
			destination = *c.finalDestination
		}
		if c.name == "command23" && (c.actor.MP != 10 || !c.actor.Acted || c.target.X != destination.X || c.target.Y != destination.Y || g.sel != nil || g.nativeItemRelocating) {
			p.err = fmt.Errorf("relocation final transaction differs")
			return
		}
		beforeReturn := unitPresentGPUState(g)
		g.frame++
		g.Draw(screen)
		if !g.modernHDWorldVisible || g.loadErr != "" {
			p.err = fmt.Errorf("HD map did not return: %s", g.loadErr)
			return
		}
		if c.name == "story-staging" || c.name == "battle-present" || c.finalDestination != nil {
			if c.name == "story-staging" && !reflect.DeepEqual(beforeReturn, unitPresentGPUState(g)) {
				p.err = fmt.Errorf("story return Draw changed canonical state")
				return
			}
			frame := g.modernHDMapPresented
			pixels, ok := p.checkWorld(g, frame, healHDCPUWorld(frame, p.set), screen, c.name+"/return")
			if !ok {
				return
			}
			c.returnSHA = fmt.Sprintf("%x", sha256.Sum256(pixels))
			pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
			copy(pic.Pix, pixels)
			file, err := os.Create(filepath.Join(p.out, c.name+"-return.png"))
			if err != nil {
				p.err = err
				return
			}
			err = png.Encode(file, pic)
			closeErr := file.Close()
			if err != nil || closeErr != nil {
				p.err = fmt.Errorf("return PNG: %v %v", err, closeErr)
				return
			}
		}
		fmt.Printf("unit present %s complete draws=%d f2=%d legs=%d continuation=1\n", c.name, c.draws, len(c.f2), len(c.legs))
		p.index++
		p.cachedFrame, p.cachedWant = nil, nil
	}
}

func TestModernHDUnitPresentCompleteGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_UNIT_PRESENT_GPU") != "1" {
		t.Skip("dedicated complete unit-present GPU process")
	}
	runModernHDUnitPresentGPU(t, false)
}

func TestModernHDUnitPresentStoryGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_UNIT_PRESENT_STORY_GPU") != "1" {
		t.Skip("dedicated story unit-present GPU process")
	}
	runModernHDUnitPresentGPU(t, true)
}

func runModernHDUnitPresentGPU(t *testing.T, story bool) {
	t.Helper()
	t.Setenv("FD2_MUTE", "1")
	catalog, pack, out := os.Getenv("FD2_TEST_UNIT_PRESENT_CATALOG"), os.Getenv("FD2_TEST_UNIT_PRESENT_PACK"), os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh capture output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	if !hasHealHDBank(set.mapEffects, 114, 124) {
		t.Fatal("all eleven actual HD effects required")
	}
	p := &unitPresentGPUProbe{set: set, out: out}
	names := []string{"command23", "staging"}
	if story {
		names = []string{"story-staging", "battle-present"}
	}
	for _, name := range names {
		g, actor, target := unitPresentHDRelocationGame(t, set)
		control, _, _ := unitPresentHDRelocationGame(t, nil)
		continued, controlContinued := new(int), new(int)
		if name != "command23" {
			// Discard only these preflighted fixture jobs, then use the real
			// battle staging/focus wrapper with the same original map assets.
			for _, game := range []*Game{g, control} {
				game.nativeModifierPresentation = nil
				game.sel = nil
				game.moved = false
				game.itemOpen = false
				game.nativeItemRelocating = false
				if !game.st.Units[1].SetNativeMapCoordinatesRaw(255, 255) {
					t.Fatal("staging source failed")
				}
				if err := game.composeNativeMapFrame(); err != nil {
					t.Fatal(err)
				}
			}
			for i, game := range []*Game{g, control} {
				count := continued
				if i == 1 {
					count = controlContinued
				}
				switch name {
				case "story-staging":
					source := game.st
					game.storyActors = make([]battle.Unit, len(source.Units))
					for n, unit := range source.Units {
						game.storyActors[n] = *unit
					}
					game.storyNativeMapView, game.hasStoryNativeMapView = source.NativeMapViewState, true
					game.storyBG, game.st = true, nil
					if err := game.materializeNativeStoryMapState(source); err != nil {
						t.Fatal(err)
					}
					if err := game.composeNativeStoryMapFrame(); err != nil {
						t.Fatal(err)
					}
					if err := game.startNativeStagingPresent(campaign.NativeStagingPresent{Slot: 1, X: 6, Y: 3}); err != nil {
						t.Fatal(err)
					}
					game.focusJob.unitPresent.then = func() { *count++ }
				case "battle-present":
					spec := campaign.NativeUnitPresent{LastRuntimeSlot: true, NewX: 8, NewY: 5, VisualX: 8, VisualY: 5}
					if err := game.startNativeUnitPresent(spec, func() { *count++ }); err != nil {
						t.Fatal(err)
					}
				default:
					if err := game.startNativeBattleStagingPresent(&battle.NativeStagingPlacement{Slot: 1, X: 6, Y: 3}, func() { *count++ }); err != nil {
						t.Fatal(err)
					}
				}
			}
		} else {
			j := g.nativeModifierPresentation.unitPresent.hdNext
			next := j.then
			j.then = func() { *continued++; next() }
			j = control.nativeModifierPresentation.unitPresent.hdNext
			nextControl := j.then
			j.then = func() { *controlContinued++; nextControl() }
		}
		p.cases = append(p.cases, unitPresentGPUCase{name: name, g: g, control: control, actor: actor, target: target, continued: continued, controlContinued: controlContinued, legs: make(map[*nativeUnitPresentJob]int), after: make(map[*nativeUnitPresentJob][]byte), f2: make(map[string]bool), shots: make(map[string]bool)})
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 2 || p.source != 22 {
		t.Fatalf("unit present GPU cases%d source%d: %v", p.index, p.source, p.err)
	}
	var cases []map[string]any
	for _, c := range p.cases {
		cases = append(cases, map[string]any{"name": c.name, "draws": c.draws, "f2_round_trips": len(c.f2), "legs": len(c.legs), "continuations": *c.continued, "native_control_unchanged": true, "shots": len(c.shots), "return_final_sha256": c.returnSHA})
	}
	status := "PASS-UNIT-PRESENT-HD-GAME"
	if story {
		status = "PASS-UNIT-PRESENT-HD-STORY-GAME"
	}
	report := map[string]any{"status": status, "cases": cases, "sources": p.sources, "rows": p.rows, "fixed_clock_unix": 1000, "native_seed": 7, "rgba_tolerance": 2, "scope": "actual target/destination or story staging/direct-unit handlers and Game.Draw, controlled records/placement; no new original oracle or normal-player proof"}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
