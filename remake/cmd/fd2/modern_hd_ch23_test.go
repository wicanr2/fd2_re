package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func stage42TestAsset(t *testing.T) modernThemeAsset {
	t.Helper()
	if os.Getenv("FD2_TEST_UNIT_PRESENT_CATALOG") == "" || os.Getenv("FD2_TEST_UNIT_PRESENT_PACK") == "" {
		t.Skip("dedicated stage verifier supplies catalog and private pack")
	}
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_UNIT_PRESENT_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	for _, asset := range catalog.Assets {
		if asset.Role == "map_stage_hd" {
			return asset
		}
	}
	t.Fatal("dedicated candidate with FDOTHER42 required")
	return modernThemeAsset{}
}

func stage42TestGame(t *testing.T, hd bool) *Game {
	t.Helper()
	g := rowShakeHDTestGame(t, false, true)
	if err := loadModernMapStageHD(g.modernStoryPortraits, stage42TestAsset(t), os.Getenv("FD2_TEST_UNIT_PRESENT_PACK")); err != nil {
		t.Fatal(err)
	}
	g.nativeMapAssets.MapIndex = 23
	// NativeTerrainLUTIndex maps 20 phases to indices 0..10. The compact
	// unit-present fixture supplies only entries 0..9.
	for len(g.nativeMapAssets.LUTs) < 11 {
		lut := make([]byte, 256)
		for i := range lut {
			lut[i] = byte(i)
		}
		g.nativeMapAssets.LUTs = append(g.nativeMapAssets.LUTs, lut)
	}
	g.nativeMapAssets.Terrain = nativeFrameTestBank(12, 80)
	for i := range g.nativeMapAssets.Units.Sprites {
		for p := range g.nativeMapAssets.Units.Sprites[i].Pixels {
			g.nativeMapAssets.Units.Sprites[i].Pixels[p] = 200
		}
	}
	for i := range g.nativeMapAssets.Terrain.Sprites {
		s := &g.nativeMapAssets.Terrain.Sprites[i]
		for p := range s.Mask {
			s.Mask[p] = 0
			s.RemapMask[p] = 0
			if p%3 == 0 {
				s.Mask[p] = 1
			} else if p%3 == 1 {
				s.RemapMask[p] = 1
			}
		}
	}
	for i := 0; i < 256; i++ {
		for c := 0; c < 3; c++ {
			g.nativeMapAssets.PaletteDAC[i*3+c] = byte(i / 4)
		}
	}
	var err error
	g.nativeMapAssets.Palette, err = fdother.VGAPaletteFromDAC(g.nativeMapAssets.PaletteDAC)
	if err != nil {
		t.Fatal(err)
	}
	g.nativeMapDAC = append([]byte(nil), g.nativeMapAssets.PaletteDAC...)
	g.nativeCh23State = nil
	if !hd {
		g.modernStoryPortraits = nil
	}
	if err := g.composeNativeMapFrameAt(time.Unix(1000, 0)); err != nil {
		t.Fatal(err)
	}
	return g
}

func stage42NativeState(g *Game) any {
	var job any
	if j := g.nativeCh23Loop; j != nil {
		job = []any{j.spec, j.stage, j.draw, j.rawESI, j.waitFrames, append(color.Palette(nil), j.palette...)}
	}
	return []any{rowShakeNativeState(g), cloneNativeCh23AdapterState(g.nativeCh23State), job}
}

func TestModernHDCh23StageRowsAndTerrainMasks(t *testing.T) {
	g := stage42TestGame(t, true)
	s := g.modernStoryPortraits.mapStages[42]
	in := *g.nativeMapFrameSource
	for _, shift := range []int{0, 1, 191} {
		state := cloneNativeCh23AdapterState(g.nativeCh23State)
		state.rowShift = shift
		for y := 0; y < 192; y++ {
			copy(state.staging[y*312:(y+1)*312], s.native.Pixels[((y-shift+192)%192)*312:((y-shift+192)%192+1)*312])
		}
		for _, mode := range []byte{0xff, 0} {
			for i := range in.Frame.Cells {
				in.Frame.Cells[i].BlitMode = mode
			}
			out, err := g.prepareModernHDMapStage(in.Frame, g.nativeMapAssets.Palette, -1, state)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range []image.Point{{0, 0}, {1, 0}, {2, 0}, {12, 23}, {310, 191}, {311, 191}} {
				x, y := p.X, p.Y
				literal := (y%24*24+x%24)%3 == 0
				if literal {
					if out.RGBAAt(x*6, y*6).A != 0 {
						t.Fatal("HD stage covered native literal")
					}
					continue
				}
				if mode == 0xff || (y%24*24+x%24)%3 == 2 {
					want := s.art.Image.RGBAAt(x*6, ((y-shift+192)%192)*6)
					if out.RGBAAt(x*6, y*6) != want {
						t.Fatal("HD stage row or transparent span differs", shift, p)
					}
				}
			}
		}
	}
	// Independent endpoint: a zero LUT makes only destination-remap spans
	// opaque black. Literal writes and skipped spans keep different ownership.
	in.Frame.LUT = make([]byte, 256)
	for i := range in.Frame.Cells {
		in.Frame.Cells[i].BlitMode = 0
	}
	state := cloneNativeCh23AdapterState(g.nativeCh23State)
	out, err := g.prepareModernHDMapStage(in.Frame, g.nativeMapAssets.Palette, -1, state)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 192; y++ {
		for x := 0; x < 312; x++ {
			want := color.RGBA{}
			switch (y%24*24 + x%24) % 3 {
			case 1:
				want = color.RGBA{0, 0, 0, 255}
			case 2:
				want = s.art.Image.RGBAAt(x*6, ((y-state.rowShift+192)%192)*6)
			}
			if out.RGBAAt(x*6, y*6) != want {
				t.Fatal("literal/remap/skip ownership differs", x, y)
			}
		}
	}
	bad := cloneNativeCh23AdapterState(g.nativeCh23State)
	bad.rowShift++
	if _, err := g.prepareModernHDMapStage(in.Frame, g.nativeMapAssets.Palette, -1, bad); err == nil {
		t.Fatal("unproven row shift admitted")
	}
}

func TestModernHDCh23FigurePalettePreservesAlpha(t *testing.T) {
	native := nativeFrameTestSprite(20)
	var ctx modernTerrainColourContext
	ctx.Base[20] = [3]float64{60, 120, 180}
	ctx.Raw[20] = [3]float64{20, 80, 140}
	raw := image.NewNRGBA(image.Rect(0, 0, 144, 144))
	raw.SetNRGBA(0, 0, color.NRGBA{100, 150, 200, 128})
	raw.SetNRGBA(1, 0, color.NRGBA{20, 30, 40, 0})
	before := append([]byte(nil), raw.Pix...)
	out, err := shadeModernHDMapFigure(raw, native, ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := color.RGBAModel.Convert(color.NRGBA{60, 110, 160, 128}).(color.RGBA)
	if out.RGBAAt(0, 0) != want || out.RGBAAt(1, 0) != (color.RGBA{}) || !bytes.Equal(before, raw.Pix) {
		t.Fatal("palette mapping lost alpha or edited source")
	}
	ctx.Raw[20] = [3]float64{}
	out, err = shadeModernHDMapFigure(raw, native, ctx)
	if err != nil || out.RGBAAt(0, 0) != (color.RGBA{0, 0, 0, 128}) {
		t.Fatal("black endpoint lost alpha", err)
	}
}

func TestModernHDCh23FullLoopsMatchNativeControl(t *testing.T) {
	g, control := stage42TestGame(t, true), stage42TestGame(t, false)
	now := time.Unix(1000, 0)
	for _, v := range []*Game{g, control} {
		v.nativeMapFrozenNow = func() time.Time { return now }
	}
	a, b := 0, 0
	for _, phase := range []string{"initial", "palette"} {
		if err := g.startNativeCh23Loop(nativeCh23LoopForTest(phase), func() { a++ }); err != nil {
			t.Fatal(err)
		}
		if err := control.startNativeCh23Loop(nativeCh23LoopForTest(phase), func() { b++ }); err != nil {
			t.Fatal(err)
		}
		count := 0
		for g.nativeCh23Loop != nil && count < 301 {
			if !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) || g.nativeCh23Loop.hdFrame == nil {
				t.Fatal("native producer diverged", phase, count)
			}
			before := stage42NativeState(g)
			g.stepNativeCh23LoopAt(now)
			if !reflect.DeepEqual(before, stage42NativeState(g)) {
				t.Fatal("loop advanced without Draw")
			}
			g.nativeCh23Loop.drawn = true
			control.nativeCh23Loop.drawn = true
			for tick := 0; tick < 3; tick++ {
				now = now.Add(time.Second / 60)
				g.stepNativeCh23LoopAt(now)
				control.stepNativeCh23LoopAt(now)
			}
			count++
		}
		want := 240
		if phase == "palette" {
			want = 60
		}
		if count != want || control.nativeCh23Loop != nil || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
			t.Fatal("wrong loop boundary", phase, count, g.loadErr, control.loadErr)
		}
	}
	if a != 2 || b != 2 || g.nativeRNGState != 7 {
		t.Fatal("continuation or RNG changed")
	}
}

func TestModernHDCh23F2RejectsAndStagesAllOwners(t *testing.T) {
	for _, kind := range []string{"native-source", "row", "image", "size", "partial", "later-owner"} {
		g := stage42TestGame(t, true)
		set := g.modernStoryPortraits
		g.modernStoryPortraits = nil
		if err := g.startNativeCh23Loop(nativeCh23LoopForTest("initial"), nil); err != nil {
			t.Fatal(err)
		}
		j := g.nativeCh23Loop
		switch kind {
		case "native-source":
			j.hdSource.work[0]++
		case "row":
			j.hdSource.stage.rowShift++
		case "image":
			set.mapStages[42].art.Image = nil
		case "size":
			set.mapStages[42].art.Image = image.NewRGBA(image.Rect(0, 0, 4, 4))
		case "partial":
			delete(set.mapSpritesHD, 0)
		case "later-owner":
			g.nativeUnitPresent = &nativeUnitPresentJob{}
		}
		before := stage42NativeState(g)
		if err := g.applyTheme(set); err == nil || g.modernStoryPortraits != nil || j.hdFrame != nil || j.hdTheme != nil || !reflect.DeepEqual(before, stage42NativeState(g)) {
			t.Fatal("failed F2 partially published", kind, err)
		}
	}
}

func TestModernHDCh23ColdFailureRestoresNativeBoundary(t *testing.T) {
	g := stage42TestGame(t, true)
	g.modernStoryPortraits.mapStages[42].art.Image = nil
	before := stage42NativeState(g)
	continued := 0
	if err := g.startNativeCh23Loop(nativeCh23LoopForTest("initial"), func() { continued++ }); err == nil || g.nativeCh23Loop != nil || continued != 0 || !reflect.DeepEqual(before, stage42NativeState(g)) {
		t.Fatal("cold HD failure leaked native initialization", err)
	}
}

func TestModernHDCh23LegacyThemeKeepsNativeStage(t *testing.T) {
	g := stage42TestGame(t, true)
	delete(g.modernStoryPortraits.mapStages, 42)
	if err := g.startNativeCh23Loop(nativeCh23LoopForTest("initial"), nil); err != nil {
		t.Fatal(err)
	}
	f := g.nativeCh23Loop.hdFrame
	if f == nil || len(f.Terrain) != 0 || len(f.Units) == 0 || f.Units[0].ColourImage == nil {
		t.Fatal("legacy stage lost HD figures or invented a surface")
	}
}

func TestModernHDCh23StoryPrivateStage(t *testing.T) {
	g := stage42TestGame(t, true)
	g.nativeMapFrozenNow = func() time.Time { return time.Unix(1000, 0) }
	source := g.st
	g.storyActors = make([]battle.Unit, len(source.Units))
	for i, u := range source.Units {
		g.storyActors[i] = *u
	}
	g.storyBG, g.st = true, nil
	g.storyNativeMapView, g.hasStoryNativeMapView = source.NativeMapViewState, true
	g.camX, g.camY = float64(g.storyNativeMapView.CameraX*24), float64(g.storyNativeMapView.CameraY*24)
	if err := g.materializeNativeStoryMapState(source); err != nil {
		t.Fatal(err)
	}
	if err := g.composeNativeStoryMapFrame(); err != nil {
		t.Fatal(err)
	}
	before := stage42NativeState(g)
	for i := 0; i < 2; i++ {
		f, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
		if err != nil {
			t.Fatal(err)
		}
		if f == nil || len(f.Terrain) == 0 || !f.TransientTerrain || len(f.Units) == 0 || f.Units[0].ColourImage == nil {
			t.Fatal("private story lost same-frame stage or coloured figures")
		}
		if !reflect.DeepEqual(before, stage42NativeState(g)) {
			t.Fatal("private story HD published native stage")
		}
	}
}
