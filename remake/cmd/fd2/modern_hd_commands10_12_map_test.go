package main

import (
	"image"
	"image/color"
	"math"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func command1012HDTestGame(t *testing.T, id int, hit bool) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, actor, target := player9MapTestGame(t, hit)
	record := g.st.NativeCommandBook[9]
	record.ID = id
	g.st.NativeCommandBook[id] = record
	actor.SetMapPlacement(1, 6, 3)
	target.SetMapPlacement(2, 6, 1)
	if err := actor.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	if err := target.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 2, CursorY: 6, VisibleCursorX: 2, VisibleCursorY: 6}); err != nil {
		t.Fatal(err)
	}
	g.curX, g.curY = 2, 6
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g, actor, target
}

func command1012HDTestTheme(t *testing.T) *modernStoryPortraitSet {
	t.Helper()
	catalog, pack := os.Getenv("FD2_TEST_COMMAND1012_CATALOG"), os.Getenv("FD2_TEST_COMMAND1012_PACK")
	if catalog == "" || pack == "" {
		t.Fatal("explicit complete catalog and pack required")
	}
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func command1012HDTestMultiGame(t *testing.T, id int) (*Game, *battle.Unit, *battle.Unit) {
	g, actor, target := command1012HDTestGame(t, id, true)
	extra := g.st.Units[5]
	extra.Camp, extra.HP, extra.MaxHP, extra.OnField = target.Camp, 100, 100, true
	extra.ClassID, extra.NativeRecordClass, extra.HasNativeRecordClass = target.ClassID, target.NativeRecordClass, target.HasNativeRecordClass
	extra.SetMapPlacement(3, 6, 1)
	if err := extra.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	record := g.st.NativeCommandBook[id]
	record.EffectMode = 1
	g.st.NativeCommandBook[id] = record
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	plan, err := g.st.PlanNativeCommandDamage(actor, target, id, g.st.NativeCommandResistances, g.nativeRNGState)
	if err != nil || len(plan.Results) != 2 {
		t.Fatalf("two-target fixture: %+v %v", plan, err)
	}
	return g, actor, target
}

// The independent reference works in global HD texels using floating floor;
// it does not call the production quotient/remainder or frame planner.
func TestModernHDCommand1012SamplingIndependentCoordinates(t *testing.T) {
	palette := make(color.Palette, 256)
	identity := make([]byte, 256)
	for i := range palette {
		palette[i] = color.RGBA{uint8(i), uint8(i), uint8(i), 255}
		identity[i] = byte(i)
	}
	tile := image.NewRGBA(image.Rect(0, 0, 144, 144))
	for y := 0; y < 144; y++ {
		for x := 0; x < 144; x++ {
			tile.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), 99, 255})
		}
	}
	set := &modernStoryPortraitSet{terrainHD: map[int]map[int]*modernTerrainHDAsset{0: {0: {Image: tile, Role: "terrain", Colours: []byte{0}}}}}
	g := &Game{nativeMapAssets: &nativeMapAssets{MapIndex: 0, Palette: palette}}
	in := indexedmap.NativeFrameInput{Frame: indexedmap.FrameInput{MapWidth: 20, Cells: make([]fdicon.NativeTerrainCell, 20*16), Controls: make([]byte, 4), LUT: identity}}
	for _, spec := range [][3]int{{128, 128, 131}, {0, 0, 128}, {-128, 128, 125}, {-32768, -32768, 131}} {
		got, err := g.commandHDSampledTerrain(in, spec[0], spec[1], spec[2], palette, set)
		if err != nil {
			t.Fatal(err)
		}
		startX, startY := 13*1536+spec[0]-156*spec[2], 8*1536+spec[1]-96*spec[2]
		for y := 0; y < 1152; y += 11 {
			for x := 0; x < 1872; x += 13 {
				wx := int(math.Floor(float64(startX*6+x*spec[2]) / 128))
				wy := int(math.Floor(float64(startY*6+y*spec[2]) / 128))
				tx, ty := int(math.Floor(float64(wx)/144)), int(math.Floor(float64(wy)/144))
				want := color.RGBA{}
				if tx >= 0 && tx < 20 && ty >= 0 && ty < 16 {
					want = tile.RGBAAt(wx-tx*144, wy-ty*144)
				}
				if value := got.RGBAAt(x, y); value != want {
					t.Fatalf("sample %v (%d,%d): got%v want%v", spec, x, y, value, want)
				}
			}
		}
		for y := 0; y < 192; y++ {
			for x := 0; x < 312; x++ {
				wx := int(math.Floor(float64(startX+x*spec[2]) / 128))
				wy := int(math.Floor(float64(startY+y*spec[2]) / 128))
				if wx < 0 || wx >= 20*24 || wy < 0 || wy >= 16*24 {
					continue
				}
				value := got.RGBAAt(x*6, y*6)
				if int(value.R)/6 != wx%24 || int(value.G)/6 != wy%24 {
					t.Fatal("HD sample crossed native pixel anchor")
				}
			}
		}
	}
}

func TestModernHDCommand1012FullThemeAndAtomicF2(t *testing.T) {
	if os.Getenv("FD2_HD_COMMAND1012_UNIT") != "1" {
		t.Skip("dedicated complete theme process required")
	}
	t.Setenv("FD2_MUTE", "1")
	set := command1012HDTestTheme(t)
	for _, id := range []int{10, 11, 12} {
		g, actor, target := command1012HDTestGame(t, id, true)
		g.modernStoryPortraits = set
		if err := g.startNativeCommand1012Presentation(actor, target, id, nil); err != nil {
			t.Fatal(id, err)
		}
		j := g.nativeCmd1012
		if j.hdFrames == nil || len(j.hdFrames.Results) != 22 {
			t.Fatal("missing complete HD bundle")
		}
		if id != 10 && (g.native2189A == nil || len(g.native2189A.hdFrames) != 11) {
			t.Fatal("missing HD prelude")
		}
		if actor.MP != 80 || target.HP != 100 || actor.Acted || g.nativeRNGState != 7 {
			t.Fatal("preflight published transaction")
		}
		if err := g.applyTheme(nil); err != nil {
			t.Fatal(err)
		}
		bad := *set
		bank := *set.mapOverlays
		bank.frames = make(map[int]*image.RGBA, len(set.mapOverlays.frames))
		for k, v := range set.mapOverlays.frames {
			bank.frames[k] = v
		}
		delete(bank.frames, 103)
		bad.mapOverlays = &bank
		phase, frame, hold, drawn := j.phase, j.frame, j.hold, j.drawn
		if err := g.applyTheme(&bad); err == nil {
			t.Fatal("unused final digit missing did not reject F2")
		}
		bank.frames[103] = image.NewRGBA(image.Rect(0, 0, 1, 1))
		if err := g.applyTheme(&bad); err == nil {
			t.Fatal("wrong final digit geometry did not reject F2")
		}
		delete(bank.frames, 103)
		if g.modernStoryPortraits != nil || g.nativeCmd1012 != j || j.phase != phase || j.frame != frame || j.hold != hold || j.drawn != drawn || actor.MP != 80 || target.HP != 100 || g.nativeRNGState != 7 {
			t.Fatal("failed F2 changed state")
		}
		if err := g.applyTheme(set); err != nil {
			t.Fatal(err)
		}
		if j.hdFrames == nil || j.hdTheme != set {
			t.Fatal("F2 did not restore full HD bundle")
		}
		other, oa, ot := command1012HDTestGame(t, id, true)
		other.modernStoryPortraits = &bad
		if err := other.startNativeCommand1012Presentation(oa, ot, id, nil); err == nil || other.nativeCmd1012 != nil || other.native2189A != nil || oa.MP != 80 || ot.HP != 100 || other.nativeRNGState != 7 {
			t.Fatal("missing digit start was not atomic")
		}
	}
	// Move the source beside registered HD terrain. The left-edge complete
	// Game.Draw fixture alone does not prove a radial pass recolours HD art.
	for _, id := range []int{11, 12} {
		g, actor, target := command1012HDTestGame(t, id, true)
		actor.SetMapPlacement(8, 5, 3)
		target.SetMapPlacement(9, 5, 1)
		if err := actor.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
		if err := target.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
		if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 9, CursorY: 5, VisibleCursorX: 9, VisibleCursorY: 5}); err != nil {
			t.Fatal(err)
		}
		g.curX, g.curY = 9, 5
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		g.modernStoryPortraits = set
		if err := g.startNativeCommand1012Presentation(actor, target, id, nil); err != nil {
			t.Fatal(err)
		}
		j := g.native2189A
		normal := j.hdFrames[10]
		inside, changed := 0, 0
		for pass, spec := range j.hdRadial {
			if spec.CenterX != 204 || spec.CenterY != 138 || spec.Radius != g.nativeCmd1012.schedule.Prelude.InitialRadius+pass*g.nativeCmd1012.schedule.Prelude.RadiusStep {
				t.Fatal("radial producer geometry differs")
			}
			plane := j.hdFrames[pass].Terrain[0].Image
			for _, terrain := range normal.Terrain {
				rect := terrain.Rect.Add(image.Pt(-24, -24))
				col, row := rect.Min.X/144, rect.Min.Y/144
				cell := j.hdInput.Frame.Cells[row*j.hdInput.Frame.MapWidth+col]
				lut := make([]byte, 256)
				for index := range lut {
					nativeIndex := byte(index)
					if cell.BlitMode != 0xff {
						nativeIndex = j.hdInput.Frame.LUT[index]
					}
					lut[index] = j.hdLUTs[pass][nativeIndex]
				}
				colours, err := terrainColourContext(0, g.nativeMapAssets.Palette, j.palette, lut)
				if err != nil {
					t.Fatal(err)
				}
				mapped := set.terrainHD[0][terrain.Frame].shade(colours, false)
				for sy := 3; sy < 144; sy += 6 {
					for sx := 3; sx < 144; sx += 6 {
						x, y := rect.Min.X+sx, rect.Min.Y+sy
						if x < 0 || x >= 1872 || y < 0 || y >= 1152 {
							continue
						}
						dy := math.Abs(float64(138 - y/6))
						covered := false
						if dy < float64(spec.Radius) {
							span := int(math.Sqrt(float64(spec.Radius*spec.Radius)-dy*dy) * 1.2)
							left, right := 204-span, 204+span
							if left < 0 {
								left = 0
							}
							if right > 312 {
								right = 312
							}
							covered = x/6 >= left && x/6 < right
						}
						want := terrain.Image.RGBAAt(sx, sy)
						if covered {
							inside++
							want = mapped.RGBAAt(sx, sy)
							if want != terrain.Image.RGBAAt(sx, sy) {
								changed++
							}
						}
						if plane.RGBAAt(x, y) != want {
							t.Fatalf("command%d pass%d radial HD point (%d,%d)", id, pass, x, y)
						}
					}
				}
			}
		}
		if inside < 100 || changed < 100 {
			t.Fatalf("vacuous radial HD check id%d inside%d changed%d", id, inside, changed)
		}
		t.Logf("command%d independent radial HD points inside=%d changed=%d", id, inside, changed)
	}
}
