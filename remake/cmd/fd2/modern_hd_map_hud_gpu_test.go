package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type hudHDGPUCase struct {
	g     *Game
	frame *modernHDMapFrame
	name  string
}
type hudHDGPUProbe struct {
	set   *modernStoryPortraitSet
	cases []hudHDGPUCase
	index int
	err   error
	out   string
	rows  []map[string]any
}

func (p *hudHDGPUProbe) Update() error {
	if p.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *hudHDGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *hudHDGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == len(p.cases) {
		return
	}
	c := p.cases[p.index]
	g := c.g
	if c.frame != nil {
		if !g.drawPreparedModernHDMapFrame(screen, c.frame, false) {
			p.err = fmt.Errorf("frame rejected: %s", g.loadErr)
			return
		}
	} else {
		g.frame++
		g.Draw(screen)
	}
	f := g.modernHDMapPresented
	if f == nil || f.HUDUnit == nil && f.HUDPanel == nil || !g.modernHDWorldVisible {
		p.err = fmt.Errorf("%s missing actual HUD: %s", c.name, g.loadErr)
		return
	}
	want := healHDCPUWorld(f, p.set)
	pixels := make([]byte, 1920*1200*4)
	g.modernHDMapGPU.World.ReadPixels(pixels)
	for i, v := range want.Pix {
		d := int(pixels[i]) - int(v)
		if d < -2 || d > 2 {
			p.err = fmt.Errorf("%s full GPU byte%d got%d want%d", c.name, i, pixels[i], v)
			return
		}
	}
	// A second opaque restore must erase a ring crossing this panel without
	// doubling translucent edges. Compare against the independently rastered patch.
	if c.frame != nil && !f.OverlaysAboveUI {
		var restore image.Rectangle
		if f.HUDPanel != nil {
			restore = f.HUDPanel.Rect
		} else {
			restore = f.HUDUnit.Unit.Rect
		}
		g.modernHDMapGPU.World.SubImage(restore).(*ebiten.Image).Fill(color.RGBA{0, 255, 0, 255})
		g.modernHDMapGPU.drawHUD(f, modernHDMapViewport())
		g.modernHDMapGPU.drawHUD(f, modernHDMapViewport())
		g.modernHDMapGPU.World.ReadPixels(pixels)
		for y := restore.Min.Y; y < restore.Max.Y; y++ {
			for x := restore.Min.X; x < restore.Max.X; x++ {
				for ch := 0; ch < 4; ch++ {
					i := (y*1920+x)*4 + ch
					d := int(pixels[i]) - int(want.Pix[i])
					if d < -2 || d > 2 {
						p.err = fmt.Errorf("ring restoration changed HUD alpha")
						return
					}
				}
			}
		}
	}
	if c.frame == nil {
		work, vga, rng := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...), g.nativeRNGState
		if e := g.applyTheme(nil); e != nil {
			p.err = e
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.err = fmt.Errorf("F2 native retained HD")
			return
		}
		if e := g.applyTheme(p.set); e != nil {
			p.err = e
			return
		}
		g.Draw(screen)
		if !bytes.Equal(work, g.nativeMapWork) || !bytes.Equal(vga, g.nativeMapVGA) || rng != g.nativeRNGState {
			p.err = fmt.Errorf("F2 changed native transaction")
			return
		}
		g.modernHDMapGPU.World.ReadPixels(pixels)
		for i, v := range want.Pix {
			d := int(pixels[i]) - int(v)
			if d < -2 || d > 2 {
				p.err = fmt.Errorf("F2 changed GPU world")
				return
			}
		}
	}
	final := ebiten.NewImage(1920, 1200)
	presentation := &modernHDPresentation{Game: g}
	var geom ebiten.GeoM
	geom.Scale(3, 3)
	presentation.DrawFinalScreen(final, screen, geom)
	final.ReadPixels(pixels)
	pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	copy(pic.Pix, pixels)
	file, e := os.Create(filepath.Join(p.out, c.name+".png"))
	if e != nil {
		p.err = e
		return
	}
	e = png.Encode(file, pic)
	closeErr := file.Close()
	if e != nil {
		p.err = e
		return
	}
	if closeErr != nil {
		p.err = closeErr
		return
	}
	// Logical UI and other menus may add pixels elsewhere. The actual 144px
	// icon footprint must survive the final screen unchanged, including HP.
	var r image.Rectangle
	if f.HUDPanel != nil {
		r = f.HUDPanel.Rect
	} else {
		r = f.HUDUnit.Unit.Rect
	}
	if f.OverlaysAboveUI {
		r = r.Intersect(f.Viewport)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			for ch := 0; ch < 4; ch++ {
				i := (y*1920+x)*4 + ch
				d := int(pixels[i]) - int(want.Pix[i])
				if d < -2 || d > 2 {
					p.err = fmt.Errorf("%s final HUD mismatch at %d,%d channel%d got%d want%d logical%v msg%q text%+v", c.name, x, y, ch, pixels[i], want.Pix[i], screen.At(x/3, y/3), g.msg, g.modernHDText)
					return
				}
			}
		}
	}
	group, pose := -1, -1
	if f.HUDUnit != nil {
		group, pose = f.HUDUnit.Unit.Group, f.HUDUnit.Unit.Frame
	}
	p.rows = append(p.rows, map[string]any{"case": c.name, "group": group, "frame": pose, "rect": r, "panel_hd": f.HUDPanel != nil, "terrain_hd": f.HUDPanel != nil && f.HUDPanel.TerrainHD, "game_draw": c.frame == nil, "f2": c.frame == nil, "final_sha256": fmt.Sprintf("%x", sha256.Sum256(pixels)), "rgba_tolerance": 2})
	final.Dispose()
	p.index++
}

func TestModernHDHUDUnitSourcesAndGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_HUD_GPU") != "1" {
		t.Skip("dedicated actual-source HUD GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	set := modifierHDTestTheme(t)
	out := os.Getenv("FD2_HD_HUD_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if _, e := os.Stat(out); !os.IsNotExist(e) {
		t.Fatal("fresh output required")
	}
	if e := os.Mkdir(out, 0755); e != nil {
		t.Fatal(e)
	}
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	g.modernStoryPortraits = set
	in, e := g.nativeCh28PostFrameInput(g.st)
	if e != nil {
		t.Fatal(e)
	}
	groups, states := 0, 0
	for group := 0; group < fdicon.NativeSourceGroupCount; group++ {
		if !fdicon.NativeSourceGroupHasPixels(group) {
			continue
		}
		cache := &fdicon.NativeSelectorCache{}
		slot, e := cache.SlotFor(group)
		if e != nil {
			t.Fatal(e)
		}
		in.HUDCache = cache
		in.HUD.DisplayGateA, in.HUD.DisplayGateB = true, true
		for state := 0; state < 4; state++ {
			in.HUD.OptionalUnit = &indexedmap.NativeMapHUDOptionalUnit{SelectorSlot: slot, RawState: state, Current: 5, Maximum: 10}
			patch, e := g.prepareModernHDMapHUDUnit(in, g.nativeMapAssets.Palette)
			if e != nil || patch == nil || patch.Unit.Group != group {
				t.Fatalf("raw group%d state%d: %v", group, state, e)
			}
			states++
		}
		groups++
	}
	if groups != 125 || states != 500 {
		t.Fatal("incomplete real HUD sources")
	}
	p := &hudHDGPUProbe{set: set, out: out}
	for _, anchor := range []int{1, 242} {
		for _, cursor := range []int{0, 1} {
			g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
			g.font = loadFont()
			if g.font == nil {
				t.Fatal("Noto font required")
			}
			// This controlled fixture starts at a neutral map, after the chapter
			// dialogue. Loading a font must not expose its leftover opening line.
			g.dialog = nil
			view := g.st.NativeMapViewState
			view.CursorX, view.VisibleCursorX = cursor, cursor
			if e := g.st.MaterializeNativeMapViewState(view); e != nil {
				t.Fatal(e)
			}
			g.curX = cursor
			if !g.st.MaterializeNativeMapHUDState(1, 1, anchor) {
				t.Fatal("HUD anchor rejected")
			}
			if e := g.applyTheme(set); e != nil {
				t.Fatal(e)
			}
			g.modernHDMapComposing = true
			if e := g.composeNativeMapFrame(); e != nil {
				t.Fatal(e)
			}
			if g.modernHDMapFrame == nil || g.modernHDMapFrame.HUDUnit == nil {
				t.Fatalf("cursor%d HUD absent", cursor)
			}
			p.cases = append(p.cases, hudHDGPUCase{g: g, name: fmt.Sprintf("game-anchor%d-cursor%d", anchor, cursor)})
			if anchor == 242 && cursor == 1 {
				f := *g.modernHDMapFrame
				p.cases = append(p.cases, hudHDGPUCase{g: g, frame: &f, name: "ring-restore"})
				event := modernPlayer9EventFrame(&f, g.nativeMapVGA, g.nativeMapAssets.Palette)
				overlay := image.NewRGBA(image.Rect(0, 0, 31, 29))
				for y := 0; y < 29; y++ {
					for x := 0; x < 31; x++ {
						overlay.SetRGBA(x, y, color.RGBA{51, 0, 0, 85})
					}
				}
				event.Overlays = []modernHDMapOverlayDraw{{Image: overlay, Rect: overlay.Bounds().Add(event.HUDUnit.Unit.Rect.Min.Add(image.Pt(50, 70)))}}
				p.cases = append(p.cases, hudHDGPUCase{g: g, frame: event, name: "event-effect-above-hud"})
				in, err := g.nativeCh28PostFrameInput(g.st)
				if err != nil {
					t.Fatal(err)
				}
				for control := byte(0); control < 6; control++ {
					input := in
					input.HUD.TerrainControl = control
					input.HUD.OptionalUnit = nil
					panel, err := g.prepareModernHDMapHUDPanel(input, g.nativeMapAssets.Palette)
					if err != nil {
						t.Fatal(err)
					}
					terrain := f
					terrain.HUDUnit, terrain.HUDPanel = nil, panel
					p.cases = append(p.cases, hudHDGPUCase{g: g, frame: &terrain, name: fmt.Sprintf("terrain-control%d", control)})
				}
				var registered []int
				for tile, asset := range set.terrainHD[0] {
					if asset.supports("terrain") {
						registered = append(registered, tile)
					}
				}
				sort.Ints(registered)
				if len(registered) == 0 {
					t.Fatal("registered terrain required")
				}
				input := in
				input.HUD.OptionalUnit = nil
				input.HUD.TerrainDescriptor = registered[0]
				panel, err := g.prepareModernHDMapHUDPanel(input, g.nativeMapAssets.Palette)
				if err != nil || !panel.TerrainHD || panel.Terrain != set.terrainHD[0][registered[0]].Image {
					t.Fatal("actual registered HUD terrain", err)
				}
				terrain := f
				terrain.HUDUnit, terrain.HUDPanel = nil, panel
				p.cases = append(p.cases, hudHDGPUCase{g: g, frame: &terrain, name: fmt.Sprintf("registered-terrain-%d", registered[0])})
				for _, hp := range [][2]uint16{{0, 0}, {5, 10}, {999, 999}, {1000, 1000}, {65535, 65534}} {
					input := in
					u := *in.HUD.OptionalUnit
					u.Current, u.Maximum = hp[0], hp[1]
					input.HUD.OptionalUnit = &u
					panel, err := g.prepareModernHDMapHUDPanel(input, g.nativeMapAssets.Palette)
					if err != nil {
						t.Fatal(err)
					}
					frozen := f
					frozen.HUDPanel = panel
					p.cases = append(p.cases, hudHDGPUCase{g: g, frame: &frozen, name: fmt.Sprintf("hp-%d-%d", hp[0], hp[1])})
				}
			}
		}
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if e := ebiten.RunGame(p); e != nil {
		t.Fatal(e)
	}
	if p.err != nil || p.index != len(p.cases) {
		t.Fatalf("HUD GPU %d/%d: %v", p.index, len(p.cases), p.err)
	}
	report := map[string]any{"status": "PASS-HUD-PANEL-HD-RUNTIME-E1", "groups": groups, "source_states": states, "fixed_clock_unix": 1000, "native_rng_seed": 7, "cases": p.rows, "scope": "controlled Game.Draw, not a new original parity receipt"}
	raw, e := json.MarshalIndent(report, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(out, "gpu.json"), append(raw, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
}
