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
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type sharedGroundGPUCase struct {
	g             *Game
	frame         *modernHDMapFrame
	name          string
	mapID, source int
	keepPNG       bool
	skipF2        bool
}
type sharedGroundGPUProbe struct {
	set   *modernStoryPortraitSet
	cases []sharedGroundGPUCase
	index int
	err   error
	out   string
	rows  []map[string]any
}

func (p *sharedGroundGPUProbe) Update() error {
	if p.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *sharedGroundGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *sharedGroundGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == len(p.cases) {
		return
	}
	c := p.cases[p.index]
	g := c.g
	if c.frame == nil {
		g.frame++
		g.Draw(screen)
	} else if !g.drawPreparedModernHDMapFrame(screen, c.frame, false) {
		p.err = fmt.Errorf("%s refused: %s", c.name, g.loadErr)
		return
	}
	f := g.modernHDMapPresented
	if f == nil || !g.modernHDWorldVisible {
		p.err = fmt.Errorf("%s world missing", c.name)
		return
	}
	want := healHDCPUWorld(f, p.set)
	got := make([]byte, 1920*1200*4)
	g.modernHDMapGPU.World.ReadPixels(got)
	check := func() bool {
		for i, v := range want.Pix {
			d := int(got[i]) - int(v)
			if d < -2 || d > 2 {
				p.err = fmt.Errorf("%s GPU byte%d got%d want%d", c.name, i, got[i], v)
				return false
			}
		}
		return true
	}
	if !check() {
		return
	}
	f2 := false
	if c.frame == nil && !c.skipF2 {
		work, vga, rng := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...), g.nativeRNGState
		if e := g.applyTheme(nil); e != nil {
			p.err = e
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.err = fmt.Errorf("native retained HD")
			return
		}
		if e := g.applyTheme(p.set); e != nil {
			p.err = e
			return
		}
		g.Draw(screen)
		if !bytes.Equal(work, g.nativeMapWork) || !bytes.Equal(vga, g.nativeMapVGA) || rng != g.nativeRNGState {
			p.err = fmt.Errorf("F2 changed native state")
			return
		}
		g.modernHDMapGPU.World.ReadPixels(got)
		if !check() {
			return
		}
		f2 = true
	}
	row := map[string]any{"case": c.name, "map": c.mapID, "source_frame": c.source, "terrain_draws": len(f.Terrain), "world_rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(got)), "rgba_tolerance": 2, "game_draw": c.frame == nil, "f2": f2}
	if c.keepPNG {
		final := ebiten.NewImage(1920, 1200)
		var geom ebiten.GeoM
		geom.Scale(3, 3)
		(&modernHDPresentation{Game: g}).DrawFinalScreen(final, screen, geom)
		final.ReadPixels(got)
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, got)
		path := filepath.Join(p.out, c.name+".png")
		file, e := os.Create(path)
		if e != nil {
			p.err = e
			return
		}
		e = png.Encode(file, pic)
		ce := file.Close()
		if e != nil {
			p.err = e
			return
		}
		if ce != nil {
			p.err = ce
			return
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			p.err = e
			return
		}
		row["file"] = filepath.Base(path)
		row["sha256"] = fmt.Sprintf("%x", sha256.Sum256(raw))
		final.Dispose()
	}
	p.rows = append(p.rows, row)
	p.index++
}

func TestModernHDSharedGroundRealBanksAndGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SHARED_GROUND_GPU") != "1" {
		t.Skip("dedicated real-bank shared-ground GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	catalogPath := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG")
	catalogInput, e := os.ReadFile(catalogPath)
	if e != nil {
		t.Fatal(e)
	}
	out := os.Getenv("FD2_HD_SHARED_GROUND_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if _, e := os.Stat(out); !os.IsNotExist(e) {
		t.Fatal("fresh output required")
	}
	if e := os.Mkdir(out, 0755); e != nil {
		t.Fatal(e)
	}
	set := modifierHDTestTheme(t)
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	palette := g.nativeMapAssets.Palette
	targets := [][2]int{{0, 144}, {2, 0}, {3, 112}, {6, 48}, {10, 48}, {11, 144}, {13, 0}, {14, 0}, {17, 144}, {30, 48}, {31, 0}}
	p := &sharedGroundGPUProbe{set: set, out: out}
	sourceGame := &Game{modernStoryPortraits: set}
	for _, target := range targets {
		mapID, frame := target[0], target[1]
		bank, controls, e := fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), mapID)
		if e != nil {
			t.Fatal(e)
		}
		if e = set.validateTerrainHD(mapID, bank); e != nil {
			t.Fatal(mapID, e)
		}
		asset := set.terrainHD[mapID][frame]
		if asset == nil {
			t.Fatal("new real grass missing", mapID)
		}
		bad := *bank
		bad.Sprites = append(bank.Sprites[:0:0], bank.Sprites...)
		bad.Sprites[frame].Mask = append([]byte(nil), bank.Sprites[frame].Mask...)
		bad.Sprites[frame].Mask[0] = 0
		if e = set.validateTerrainHD(mapID, &bad); e == nil {
			t.Fatal("wrong real-bank alpha admitted", mapID)
		}
		var data struct {
			Tiles []uint16 `json:"tiles"`
		}
		raw, e := os.ReadFile(fmt.Sprintf("../../assets/maps/map%d/map.json", mapID))
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(raw, &data); e != nil {
			t.Fatal(e)
		}
		var selected uint16
		flip, cycle, found := 0, 0, false
		for _, word := range data.Tiles {
			for a := 0; a < 2 && !found; a++ {
				for b := 0; b < 4; b++ {
					idx, e := fdicon.NativeTerrainFrameIndex(int(word&0x3ff), controls[int(word&0x3ff)*4], a, b)
					if e == nil && idx == frame {
						selected, flip, cycle, found = word, a, b, true
						break
					}
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Fatal("source has no real map selector", mapID, frame)
		}
		in := indexedmap.FrameInput{MapWidth: 13, Cells: make([]fdicon.NativeTerrainCell, 13*12), Controls: controls, Flip: flip, TerrainCycle: cycle}
		for i := range in.Cells {
			in.Cells[i].Tile = selected
		}
		for _, mode := range []string{"steady", "up1", "up6", "lut", "black", "white"} {
			lut := make([]byte, 256)
			current := append(color.Palette(nil), palette...)
			for i := range lut {
				lut[i] = byte(i)
				if mode == "lut" {
					lut[i] = byte((i + 1) % 256)
				}
				if mode == "black" {
					current[i] = color.RGBA{0, 0, 0, 255}
				}
				if mode == "white" {
					current[i] = color.RGBA{255, 255, 255, 255}
				}
			}
			ctx, e := terrainColourContext(mapID, palette, current, lut)
			if e != nil {
				t.Fatal(e)
			}
			base, camY, rows, view := 0x8088, 0, 8, 0x8088
			if mode == "up1" || mode == "up6" {
				step := 1
				if mode == "up6" {
					step = 6
				}
				base, camY, rows, view, e = indexedmap.NativeUpStepLayout(in, 2, step)
				if e != nil {
					t.Fatal(e)
				}
			}
			plan, e := buildModernHDTerrainPlan(in, base, camY, rows, view, set.terrainHD[mapID], ctx)
			if e != nil || len(plan) == 0 {
				t.Fatal("missing real selector plan", mapID, mode, e)
			}
			f := &modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200)), UI: image.NewRGBA(image.Rect(0, 0, 320, 200)), Terrain: plan, Viewport: modernHDMapViewport(), TerrainColourKey: ctx.Key, TransientTerrain: true}
			p.cases = append(p.cases, sharedGroundGPUCase{g: sourceGame, frame: f, name: fmt.Sprintf("map%02d-%s", mapID, mode), mapID: mapID, source: frame, keepPNG: mapID == 0 && (mode == "steady" || mode == "up1" || mode == "lut")})
		}
	}
	// A real map0 Game.Draw retains the original topology, units, range and HUD.
	nativeIn, e := g.nativeCh28PostFrameInput(g.st)
	if e != nil {
		t.Fatal(e)
	}
	in := nativeIn.Frame
	grassCell := -1
	for i, c := range in.Cells {
		n, e := fdicon.NativeTerrainFrameIndex(int(c.Tile&0x3ff), in.Controls[int(c.Tile&0x3ff)*4], in.Flip, in.TerrainCycle)
		if e == nil && n == 144 {
			grassCell = i
			break
		}
	}
	if grassCell < 0 {
		t.Fatal("normal map0 has no target grass")
	}
	v := g.st.NativeMapViewState
	v.CameraX = max(0, min(g.st.W-13, grassCell%g.st.W-6))
	v.CameraY = max(0, min(g.st.H-8, grassCell/g.st.W-3))
	v.CursorX, v.VisibleCursorX = v.CameraX, v.CameraX
	v.CursorY, v.VisibleCursorY = v.CameraY, v.CameraY
	if e = g.st.MaterializeNativeMapViewState(v); e != nil {
		t.Fatal(e)
	}
	g.curX, g.curY = v.CursorX, v.CursorY
	if e = g.applyTheme(set); e != nil {
		t.Fatal(e)
	}
	g.modernHDMapComposing = true
	if e = g.composeNativeMapFrame(); e != nil {
		t.Fatal(e)
	}
	visible := false
	for _, tile := range g.modernHDMapFrame.Terrain {
		visible = visible || tile.Frame == 144
	}
	if !visible {
		t.Fatal("new grass absent from actual world")
	}
	p.cases = append(p.cases, sharedGroundGPUCase{g: g, name: "game-map00", mapID: 0, source: 144, keepPNG: true})
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if e = ebiten.RunGame(p); e != nil {
		t.Fatal(e)
	}
	if p.err != nil || p.index != 67 {
		t.Fatalf("grass GPU %d/67: %v", p.index, p.err)
	}
	currentCatalog, e := os.ReadFile(catalogPath)
	if e != nil || !bytes.Equal(currentCatalog, catalogInput) {
		t.Fatal("catalog changed while GPU was running", e)
	}
	report := map[string]any{"status": "PASS-SHARED-GROUND-GPU", "maps": 11, "source_worlds": 66, "game_draws": 1, "f2": 1, "native_rng_seed": 7, "fixed_clock_unix": 1000, "wrong_bank_masks_rejected": 11, "cases": p.rows,
		"catalog_sha256":      fmt.Sprintf("%x", sha256.Sum256(catalogInput)),
		"painted_rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(set.terrainHD[0][144].Image.Pix)),
		"scope":               "十一實際bank及正式selector的重複草地測具、原樣map0 Game.Draw與F2；非十一章正常玩家或原版RGB對拍。"}
	raw, e := json.MarshalIndent(report, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(out, "gpu.json"), append(raw, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
}
