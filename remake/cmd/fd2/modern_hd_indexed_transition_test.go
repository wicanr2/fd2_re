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

func indexedHDTestGame(t *testing.T, modern bool) *Game {
	t.Helper()
	g := storyHDTestGame(t)
	g.baseMapTileset = image.NewRGBA(image.Rect(0, 0, 24, 24))
	g.storyActors[0].OnField = true
	g.storyActors[0].SetMapPlacement(6, 5, 0)
	if !modern {
		g.modernStoryPortraits = nil
	}
	return g
}

func TestModernHDIndexedTransitionFrozenFramesAndNativeControl(t *testing.T) {
	g, control := indexedHDTestGame(t, true), indexedHDTestGame(t, false)
	before := storyHDNativeSnapshot(g)
	for _, game := range []*Game{g, control} {
		if err := game.startNativeIndexedTransition(nativeIndexedTransitionSpecForTest(), "", nil); err != nil {
			t.Fatal(err)
		}
	}
	j, original := g.indexedTransition, control.indexedTransition
	if !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) || len(j.hdFrames) != 41 || len(j.frameWork) != 9 ||
		!reflect.DeepEqual(j.frameWork, original.frameWork) || !reflect.DeepEqual(j.frameVGA, original.frameVGA) || !bytes.Equal(j.work, original.work) {
		t.Fatal("HD start changed native producer or failed to stage all frames")
	}
	for i, frame := range j.hdFrames {
		if frame == nil || frame.Viewport != image.Rect(0, 0, 1872, 1152) || frame.HUDUnit != nil || frame.Range != nil || !frame.OverlaysAboveUI {
			t.Fatal("event frame lost its UI-free viewport", i)
		}
		want := 0
		if i >= 9 {
			want = (i - 9) * 2
			if frame.Background != j.hdFrames[8].Background {
				t.Fatal("DAC phase recomposed indexed pixels")
			}
		}
		if frame.PaletteAdd6 != want {
			t.Fatal("wrong baseline palette addition", i)
		}
	}
	if len(j.hdFrames[0].Foreground) == 0 {
		t.Fatal("HD world objects were not included")
	}
	frames, err := g.prepareModernHDIndexedTransition(j, g.modernStoryPortraits)
	if err != nil || frames[0] != j.hdFrames[0] || !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
		t.Fatal("cached preparation advanced source or missed the same frame", err)
	}
}

func TestModernHDIndexedTransitionUsesPreviousNativeWork(t *testing.T) {
	g := indexedHDTestGame(t, false)
	for i := range g.nativeMapAssets.Terrain.Sprites {
		sprite := &g.nativeMapAssets.Terrain.Sprites[i]
		sprite.Mask[0] = 0
		sprite.RemapMask = make([]byte, 576)
		sprite.RemapMask[0] = 1
	}
	for i := range g.m.NativeTileBlitModes {
		g.m.NativeTileBlitModes[i] = 0
	}
	if err := g.startNativeIndexedTransition(nativeIndexedTransitionSpecForTest(), "", nil); err != nil {
		t.Fatal(err)
	}
	j := g.indexedTransition
	ground := nativeTransitionInput(j.hdSource.Input)
	ground.Units, ground.ForegroundUnits = nil, nil
	for i, pass := range j.hdSource.Passes {
		work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
		if i > 0 {
			copy(work, j.frameWork[i-1])
		}
		vga := make([]byte, indexedmap.NativeMapVGASize)
		if err := indexedmap.ComposeNativeTransitionFrame(work, vga, ground, pass, j.hdSource.LUTs[i]); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(vga, j.hdSource.Backgrounds[i]) {
			t.Fatal("background lost the previous complete work", i)
		}
	}
	coldWork, coldVGA := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeMapVGASize)
	if err := indexedmap.ComposeNativeTransitionFrame(coldWork, coldVGA, ground, j.hdSource.Passes[1], j.hdSource.LUTs[1]); err != nil {
		t.Fatal(err)
	}
	if coldVGA[0] == j.hdSource.Backgrounds[1][0] {
		t.Fatal("fixture did not expose destination-dependent history")
	}
}

func TestModernHDIndexedTransitionMaskKeepsFullWorkOrigin(t *testing.T) {
	inside := func(x, y int, radial fdother.RadialLUTRemap) bool {
		dy := math.Abs(float64(y - radial.CenterY))
		if y < radial.StartY || y >= radial.EndY || dy >= float64(radial.Radius) || x < 0 || x >= radial.ClipWidth {
			return false
		}
		span := int(math.Sqrt(float64(radial.Radius*radial.Radius)-dy*dy) * float64(radial.Scale) / 10)
		return x >= radial.CenterX-span && x < radial.CenterX+span
	}
	for _, center := range []image.Point{{6, 6}, {150, 120}, {300, 190}} {
		for step := 0; step < 9; step++ {
			pass, err := fdother.BuildNativeIndexedTransitionPass(center.X, center.Y, 10+step*8, 0, 192)
			if err != nil {
				t.Fatal(err)
			}
			terrain, objects, err := modernHDRadialCounts(pass, 0)
			if err != nil {
				t.Fatal(err)
			}
			for y := 0; y < 192; y++ {
				for x := 0; x < 312; x++ {
					xx, yy := 72+x, 72+y
					a, b := inside(xx, yy, pass.FirstRadial), inside(xx, yy, pass.SecondRadial)
					r := pass.FinalRect
					c := xx < r.ClipWidth && xx >= r.CenterX-r.HorizontalRadius && xx < r.CenterX+r.HorizontalRadius && yy >= r.StartY && yy < r.EndY
					var tg, ob byte
					for _, yes := range []bool{a, b, c} {
						if yes {
							tg++
						}
					}
					for _, yes := range []bool{b, c} {
						if yes {
							ob++
						}
					}
					if healHDCountAt(terrain, x, y) != tg || healHDCountAt(objects, x, y) != ob {
						t.Fatalf("work origin mismatch center%v step%d (%d,%d)", center, step, x, y)
					}
				}
			}
		}
	}
}

func TestModernHDIndexedTransitionRejectsBeforePublication(t *testing.T) {
	for _, failure := range []string{"world", "image", "size", "late-vga", "late-work", "lut", "pass", "baseline", "input", "bank"} {
		g := indexedHDTestGame(t, true)
		set := g.modernStoryPortraits
		g.modernStoryPortraits = nil
		if err := g.startNativeIndexedTransition(nativeIndexedTransitionSpecForTest(), "", nil); err != nil {
			t.Fatal(err)
		}
		j := g.indexedTransition
		counts := make(map[int]int)
		for group, images := range set.mapSpritesHD {
			counts[group] = len(images)
		}
		units, err := buildModernHDMapUnitPlan(j.hdSource.Input.Frame, 0x8088, counts)
		if err != nil || len(units) == 0 {
			t.Fatal("missing unit control", err)
		}
		unit := units[0]
		switch failure {
		case "world":
			delete(set.mapSpritesHD, unit.Group)
		case "image":
			set.mapSpritesHD[unit.Group][unit.Frame] = nil
		case "size":
			set.mapSpritesHD[unit.Group][unit.Frame] = image.NewRGBA(image.Rect(0, 0, 143, 144))
		case "late-vga":
			j.frameVGA[8][0]++
		case "late-work":
			j.frameWork[8][0]++
		case "lut":
			j.hdSource.LUTs[8][0]++
		case "pass":
			j.hdSource.Passes[8].FirstRadial.Radius++
		case "baseline":
			j.baseline[0]++
		case "input":
			j.input.Units[0].Pose = 3
		case "bank":
			j.hdSource.Input.Frame.UnitBank = g.nativeMapAssets.Terrain
		}
		before := storyHDNativeSnapshot(g)
		if err := g.applyTheme(set); err == nil || g.modernStoryPortraits != nil || g.indexedTransition != j || j.hdFrames != nil || j.hdCached != nil ||
			!reflect.DeepEqual(before, storyHDNativeSnapshot(g)) || j.frame != 0 || j.drawn {
			t.Fatal("failed F2 partially published or advanced", failure, err)
		}
	}
}

func TestModernHDIndexedTransitionStagesAllOwnersAtomically(t *testing.T) {
	g := indexedHDTestGame(t, true)
	set := g.modernStoryPortraits
	g.modernStoryPortraits = nil
	if err := g.startNativeIndexedTransition(nativeIndexedTransitionSpecForTest(), "", nil); err != nil {
		t.Fatal(err)
	}
	j := g.indexedTransition
	g.storyNativeMapState.NativeTileBlitModes = g.storyNativeMapState.NativeTileBlitModes[:1]
	before := storyHDNativeSnapshot(g)
	if err := g.applyTheme(set); err == nil || g.modernStoryPortraits != nil || j.hdFrames != nil || j.hdCached != nil ||
		!reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
		t.Fatal("later story failure published an earlier indexed-transition stage")
	}
}

func TestModernHDIndexedTransitionCachedF2Revalidates(t *testing.T) {
	g := indexedHDTestGame(t, true)
	if err := g.startNativeIndexedTransition(nativeIndexedTransitionSpecForTest(), "", nil); err != nil {
		t.Fatal(err)
	}
	j, set := g.indexedTransition, g.modernStoryPortraits
	if err := g.applyTheme(nil); err != nil {
		t.Fatal(err)
	}
	counts := make(map[int]int)
	for group, images := range set.mapSpritesHD {
		counts[group] = len(images)
	}
	units, err := buildModernHDMapUnitPlan(j.hdSource.Input.Frame, 0x8088, counts)
	if err != nil || len(units) == 0 {
		t.Fatal(err)
	}
	unit := units[0]
	set.mapSpritesHD[unit.Group][unit.Frame] = nil
	if err := g.applyTheme(set); err == nil || g.modernStoryPortraits != nil || j.hdFrames != nil || j.hdTheme != nil || len(j.hdCached) != 41 {
		t.Fatal("cached theme bypassed cold source validation")
	}
}

func TestModernHDIndexedTransitionColdStartRejects(t *testing.T) {
	for _, failure := range []string{"world", "image", "size"} {
		g := indexedHDTestGame(t, true)
		in, err := g.buildNativeIndexedTransitionInput()
		if err != nil {
			t.Fatal(err)
		}
		frozen, err := g.unitPresentHDInput(in)
		if err != nil {
			t.Fatal(err)
		}
		counts := make(map[int]int)
		for group, images := range g.modernStoryPortraits.mapSpritesHD {
			counts[group] = len(images)
		}
		units, err := buildModernHDMapUnitPlan(frozen.Frame, 0x8088, counts)
		if err != nil || len(units) == 0 {
			t.Fatal("missing cold control", err)
		}
		unit := units[0]
		switch failure {
		case "world":
			delete(g.modernStoryPortraits.mapSpritesHD, unit.Group)
		case "image":
			g.modernStoryPortraits.mapSpritesHD[unit.Group][unit.Frame] = nil
		case "size":
			g.modernStoryPortraits.mapSpritesHD[unit.Group][unit.Frame] = image.NewRGBA(image.Rect(0, 0, 144, 143))
		}
		before, theme := storyHDNativeSnapshot(g), g.modernStoryPortraits
		if err := g.startNativeIndexedTransition(nativeIndexedTransitionSpecForTest(), "", nil); err == nil || g.indexedTransition != nil ||
			g.modernStoryPortraits != theme || !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
			t.Fatal("cold failure published an owner or native state", failure, err)
		}
	}
}
