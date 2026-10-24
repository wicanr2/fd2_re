package main

import (
	"bytes"
	"image"
	"math"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestModernHDStoryMotionControllerAndIsolation(t *testing.T) {
	for _, destination := range []image.Point{{8, 5}, {4, 5}, {6, 3}, {6, 7}, {9, 6}, {7, 8}} {
		g := storyHDTestGame(t)
		u := &g.storyActors[0]
		u.OnField = true
		u.SetMapPlacement(6, 5, 0)
		continued := 0
		job := &storyWalkJob{actor: 0, fromX: 6, fromY: 5, toX: destination.X, toY: destination.Y,
			frames: 24, finalDir: 2, then: func() { continued++ }}
		g.storyWalks = []*storyWalkJob{job}
		for step := 0; step <= 24; step++ {
			before := storyHDNativeSnapshot(g)
			frame, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
			if err != nil || frame == nil || len(frame.Units) == 0 {
				t.Fatalf("destination%v step%d: %v", destination, step, err)
			}
			if !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) || job.t != step || continued != step/24 {
				t.Fatal("HD preparation advanced controller or source")
			}
			want := image.Pt(24+int(math.Round((float64(u.X*24)+u.OffX-g.camX)*6)),
				24+int(math.Round((float64(u.Y*24)+u.OffY-g.camY-6)*6)))
			if frame.Units[0].Rect.Min != want {
				t.Fatalf("step%d got%v want%v", step, frame.Units[0].Rect.Min, want)
			}
			if step < 24 {
				if !frame.TransientTerrain || frame.Units[0].Frame != u.Dir*3+(g.frame/8)%3 {
					t.Fatal("walking phase or transient terrain missing")
				}
				g.frame++
				g.stepStoryWalks()
			}
		}
		g.stepStoryWalks()
		if continued != 1 || len(g.storyWalks) != 0 || u.X != destination.X || u.Y != destination.Y || u.OffX != 0 || u.OffY != 0 || u.Dir != 2 {
			t.Fatal("walking completion changed")
		}
	}
}

func TestModernHDStoryMotionCameraOverscanAndParallax(t *testing.T) {
	g := storyHDTestGame(t)
	g.camX += 23.5
	g.camY += 23.25
	// A row/column pattern makes independent expected viewport samples clear.
	p := &fdother.NativeMapParallaxSurface{Pixels: make([]byte, fdother.NativeMapParallaxWidth*fdother.NativeMapParallaxHeight)}
	for y := 0; y < fdother.NativeMapParallaxHeight; y++ {
		for x := 0; x < fdother.NativeMapParallaxWidth; x++ {
			p.Pixels[y*fdother.NativeMapParallaxWidth+x] = byte((x+3*y)%255 + 1)
		}
	}
	g.nativeMapAssets.Parallax = p
	before := storyHDNativeSnapshot(g)
	f, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) || len(f.Terrain) == 0 || !f.TransientTerrain {
		t.Fatal("camera composition mutated source or missed terrain")
	}
	layer := f.Terrain[0]
	if layer.Image.Bounds().Dx() != 2016 || layer.Image.Bounds().Dy() != 1296 ||
		layer.Rect.Min != image.Pt(24-141, 24-140) || !modernHDMapViewport().In(layer.Rect) {
		t.Fatal("incoming right/bottom terrain is uncovered", layer.Rect, layer.Image.Bounds())
	}
	// The aligned case must match the independent native consumer byte for
	// byte. Half-tile camera motion must move the backdrop by 1.5/1 pixels,
	// rather than the terrain's twelve pixels.
	in := indexedmap.FrameInput{CameraX: 4, CameraY: 3, Parallax: p}
	base := 0x8088
	native, projected := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeUnitPresentWorkSize)
	if err := fdother.BlitNativeMapParallaxViewport(native[base:], 456, p, 4, 3, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := seedModernHDStoryParallax(projected, base, 14, 9, in, 0, 0); err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 192; y++ {
		if !bytes.Equal(native[base+y*456:base+y*456+312], projected[base+y*456:base+y*456+312]) {
			t.Fatal("aligned parallax diverged from native contract")
		}
	}
	if err := seedModernHDStoryParallax(projected, base, 14, 9, in, 12, 12); err != nil ||
		projected[base+12*456+12] != p.Pixels[7*fdother.NativeMapParallaxWidth+13] {
		t.Fatal("parallax followed terrain speed", err)
	}
	// Exercise finite camera boundaries and no partial publication.
	for _, bad := range []float64{-1, math.NaN(), math.Inf(1), float64(g.m.W * 24)} {
		g.camX = bad
		if _, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits); err == nil {
			t.Fatal("invalid camera accepted", bad)
		}
	}
}

func TestModernHDStoryMotionTerrainRemapsOnce(t *testing.T) {
	g := storyHDTestGame(t)
	for i := 0; i < 256; i++ {
		for channel := 0; channel < 3; channel++ {
			g.nativeMapAssets.PaletteDAC[i*3+channel] = byte(i % 64)
		}
	}
	for i := range g.nativeMapAssets.Terrain.Sprites {
		sprite := &g.nativeMapAssets.Terrain.Sprites[i]
		sprite.RemapMask = make([]byte, 576)
		sprite.Mask[0], sprite.RemapMask[0] = 0, 1
	}
	for i := range g.storyNativeMapState.NativeTileBlitModes {
		g.storyNativeMapState.NativeTileBlitModes[i] = 0
	}
	static, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	g.storyWalks = []*storyWalkJob{{actor: 0}}
	moving, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 192; y++ {
		for x := 0; x < 312; x++ {
			if static.Background.RGBAAt(x+4, y+4) != moving.Terrain[0].Image.RGBAAt(x, y) {
				t.Fatalf("terrain remapped again at %d,%d", x, y)
			}
		}
	}
	if static.Background.RGBAAt(4, 4).R == 0 {
		t.Fatal("fixture did not exercise a destination remap")
	}
}

func TestModernHDStoryMotionParallaxSubpixels(t *testing.T) {
	g := storyHDTestGame(t)
	p := &fdother.NativeMapParallaxSurface{Pixels: make([]byte, fdother.NativeMapParallaxWidth*fdother.NativeMapParallaxHeight)}
	for i := range p.Pixels {
		p.Pixels[i] = byte(i%fdother.NativeMapParallaxWidth%200 + 1)
	}
	for i := range g.nativeMapAssets.Terrain.Sprites {
		s := &g.nativeMapAssets.Terrain.Sprites[i]
		s.Mask = make([]byte, 576)
		s.RemapMask = make([]byte, 576)
		s.Mask[1], s.Pixels[1], s.RemapMask[0] = 1, 42, 1
	}
	cells, err := indexedmap.BuildNativeTerrainCells(g.m.Tiles, g.storyNativeMapState.NativeTileBlitModes)
	if err != nil {
		t.Fatal(err)
	}
	for i := range cells {
		cells[i].BlitMode = 0
	}
	in := indexedmap.FrameInput{TerrainBank: g.nativeMapAssets.Terrain, Controls: g.nativeMapAssets.Controls,
		MapWidth: g.m.W, Cells: cells, LUT: g.nativeMapAssets.LUTs[0], CameraX: 4, CameraY: 3, Parallax: p}
	aligned, err := modernHDStoryParallaxBackdrop(in, 14, 9, 0, 0, g.nativeMapAssets.Palette)
	if err != nil {
		t.Fatal(err)
	}
	native := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	if err := fdother.BlitNativeMapParallaxViewport(native[0x8088:], 456, p, 4, 3, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := in.TerrainBank.BlitNativeTerrainRegion(native, 456, 72, 72, in.MapWidth, cells, in.Controls, 4, 3, 13, 8, 0, 0, in.LUT); err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 192; y++ {
		for x := 0; x < 312; x++ {
			if aligned.At(x*6, y*6) != g.nativeMapAssets.Palette[native[0x8088+y*456+x]] {
				t.Fatalf("aligned mask or remap differs at %d,%d", x, y)
			}
		}
	}
	half, err := modernHDStoryParallaxBackdrop(in, 14, 9, .5, .5, g.nativeMapAssets.Palette)
	if err != nil || half.At(123, 123) != g.nativeMapAssets.Palette[p.Pixels[26*fdother.NativeMapParallaxWidth+32]] {
		t.Fatal("subpixel parallax wobbled with the terrain grid", err)
	}
}

func TestModernHDStoryMotionKeepsIdleNativePlacement(t *testing.T) {
	g := storyHDTestGame(t)
	g.storyActors[0].SetMapPlacement(6, 5, 0)
	u := &g.storyActors[1]
	u.SetMapPlacement(8, 5, 1)
	u.NativeTransient[4] = 1
	u.NativeRecordByte5 = 0x80
	u.Acted = false // Raw palette-band provenance wins for the stationary unit.
	base, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	var idle modernHDMapUnitDraw
	for _, unit := range base.Units {
		if unit.Rect.Min.X == 24+(8-4)*144 {
			idle = unit
		}
	}
	if idle.Rect.Empty() || !idle.PaletteBand {
		t.Fatal("idle control absent")
	}
	g.storyWalks = []*storyWalkJob{{actor: 0}}
	g.camX += 0.5
	g.camY += 0.25
	moving, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	idle.Rect = idle.Rect.Add(image.Pt(-3, -2))
	for _, unit := range moving.Units {
		if unit.Rect == idle.Rect && unit.Group == idle.Group && unit.Frame == idle.Frame && unit.PaletteBand {
			return
		}
	}
	t.Fatal("stationary raw origin, force-base frame or palette band changed during motion")
}

func TestModernHDStoryMotionF2RejectsWithoutProgress(t *testing.T) {
	g := storyHDTestGame(t)
	g.storyActors[0].OnField = true
	g.storyActors[0].SetMapPlacement(6, 5, 0)
	g.storyWalks = []*storyWalkJob{{actor: 0, fromX: 6, fromY: 5, toX: 10, toY: 5, frames: 24, finalDir: 0}}
	g.stepStoryWalks()
	g.camX += 0.5
	f, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	g.modernHDMapFrame = f
	set := *g.modernStoryPortraits
	set.mapSpritesHD = make(map[int][]image.Image)
	for group, frames := range g.modernStoryPortraits.mapSpritesHD {
		set.mapSpritesHD[group] = append([]image.Image(nil), frames...)
	}
	selected := f.Units[0]
	set.mapSpritesHD[selected.Group][selected.Frame] = nil
	before, theme, job := storyHDNativeSnapshot(g), g.modernStoryPortraits, g.storyWalks[0]
	if err := g.applyTheme(&set); err == nil || g.modernStoryPortraits != theme || g.modernHDMapFrame != f ||
		!reflect.DeepEqual(before, storyHDNativeSnapshot(g)) || g.storyWalks[0] != job || job.t != 1 {
		t.Fatal("failed F2 changed an active walk")
	}
}
