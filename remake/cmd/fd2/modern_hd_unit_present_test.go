package main

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func unitPresentHDTestGame(t *testing.T, top, appear, declared bool) (*Game, campaign.NativeUnitPresent) {
	t.Helper()
	g := completeNativeUnitPresentGame(t, func() time.Time { return time.Unix(1000, 0) })
	g.nativeRNGState = 7
	if appear {
		if !g.st.Units[1].SetNativeMapCoordinatesRaw(255, 255) {
			t.Fatal("off-map fixture failed")
		}
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	g.nativeMapDAC = nil // The synthetic fixture's declared grayscale palette.
	set := modernHDMapRuntimeFixture(t).modernStoryPortraits
	picture := image.NewRGBA(image.Rect(0, 0, 144, 144))
	for y := 18; y < 144; y++ {
		for x := 24; x < 120; x++ {
			picture.Set(x, y, color.NRGBA{90, 180, 230, 255})
		}
	}
	for group, frames := range set.mapSpritesHD {
		for i := range frames {
			set.mapSpritesHD[group][i] = picture
		}
	}
	if declared {
		set.mapEffects = &modernMapOverlayHDResource{frames: make(map[int]*image.RGBA), native: make(map[int]modernLMIHDDescriptor)}
		for i := 114; i <= 124; i++ {
			entry := g.nativeMapAssets.FDOTHER6[i]
			pic := image.NewRGBA(image.Rect(0, 0, entry.Width*6, entry.Height*6))
			for y := 0; y < pic.Rect.Dy(); y++ {
				for x := 0; x < pic.Rect.Dx(); x++ {
					pic.SetRGBA(x, y, color.RGBA{230, 90, 40, 255})
				}
			}
			set.mapEffects.frames[i] = pic
			set.mapEffects.native[i] = modernLMIHDDescription(i, entry)
		}
	}
	g.modernStoryPortraits = set
	y := 5
	if top {
		y = 3
	}
	spec := campaign.NativeUnitPresent{Slot: 1, NewX: 255, NewY: 255, VisualX: 8, VisualY: y}
	if appear {
		spec.NewX, spec.NewY = 8, y
	}
	return g, spec
}

func TestModernHDUnitPresentCountsUseExplicitWorkOffset(t *testing.T) {
	pass, err := fdother.NativeUnitPresentLUTPass(156, 111, 37)
	if err != nil {
		t.Fatal(err)
	}
	inside := func(x, y, cx, cy, radius, scale, first, last, width int) bool {
		dy := math.Abs(float64(cy - y))
		if y < first || y >= last || dy >= float64(radius) {
			return false
		}
		span := int(math.Sqrt(float64(radius*radius)-dy*dy) * float64(scale) / 10)
		return x >= 0 && x < width && x >= cx-span && x < cx+span
	}
	for _, offset := range []int{0, unitPresentHDBase} {
		tc, oc, err := modernHDRadialCounts(pass, offset)
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < 192; y++ {
			for x := 0; x < 312; x++ {
				index := unitPresentHDBase + y*456 + x - offset
				xx, yy := index%456, index/456
				a, b, c := pass.FirstRadial, pass.SecondRadial, pass.FinalRect
				one := inside(xx, yy, a.CenterX, a.CenterY, a.Radius, a.Scale, a.StartY, a.EndY, a.ClipWidth)
				two := inside(xx, yy, b.CenterX, b.CenterY, b.Radius, b.Scale, b.StartY, b.EndY, b.ClipWidth)
				three := xx >= 0 && xx < c.ClipWidth && xx >= c.CenterX-c.HorizontalRadius && xx < c.CenterX+c.HorizontalRadius && yy >= c.StartY && yy < c.EndY
				terrain, objects := byte(0), byte(0)
				if one {
					terrain++
				}
				if two {
					terrain++
					objects++
				}
				if three {
					terrain++
					objects++
				}
				if healHDCountAt(tc, x, y) != terrain || healHDCountAt(oc, x, y) != objects {
					t.Fatalf("offset%x mask at%d,%d", offset, x, y)
				}
			}
		}
	}
}

func TestModernHDUnitPresentFrozenFramesAndNativeControl(t *testing.T) {
	for _, top := range []bool{false, true} {
		for _, appear := range []bool{false, true} {
			g, spec := unitPresentHDTestGame(t, top, appear, true)
			control := *g
			control.modernStoryPortraits = nil
			var err error
			control.st, err = cloneNativeUnitPresentState(g.st)
			if err != nil {
				t.Fatal(err)
			}
			control.nativeMapWork = append([]byte(nil), g.nativeMapWork...)
			control.nativeMapVGA = append([]byte(nil), g.nativeMapVGA...)
			units, rng := compoundMapUnitSnapshot(g), g.nativeRNGState
			if err := g.startNativeUnitPresent(spec, nil); err != nil {
				t.Fatal(err)
			}
			if err := control.startNativeUnitPresent(spec, nil); err != nil {
				t.Fatal(err)
			}
			j, n := g.nativeUnitPresent, control.nativeUnitPresent
			count := 51
			if top {
				count = 45
			}
			if len(j.hdFrames) != count || j.mutationAt != 17 || !reflect.DeepEqual(j.workFrames, n.workFrames) || !reflect.DeepEqual(j.vgaFrames, n.vgaFrames) || !reflect.DeepEqual(j.waits, n.waits) {
				t.Fatal("HD changed native producer")
			}
			if !reflect.DeepEqual(units, compoundMapUnitSnapshot(g)) || rng != g.nativeRNGState {
				t.Fatal("preflight changed units/RNG")
			}
			origin := j.hdFrames[0].Terrain[len(j.hdFrames[0].Terrain)-1].Rect.Min
			if origin != image.Pt((spec.VisualX-4)*144, ((spec.VisualY-3)*24+1)*6) || j.hdSource.Cursor != image.Pt(6, 4) {
				t.Fatal("visual anchor and LUT cursor conflated")
			}
			for i := range j.hdFrames {
				f := j.hdFrames[i]
				if f.HUDUnit != nil || f.Range != nil || f.Viewport != image.Rect(0, 0, 1872, 1152) {
					t.Fatal("unit event acquired steady HUD/inset")
				}
				if i >= 17 && i < count-10 && len(f.Overlays) != i-16 {
					t.Fatal("strip prefix lost a completed row")
				}
			}
			for i := 0; i < count; i++ {
				g.nativeUnitPresent.drawn = true
				g.nativeUnitPresent.wait = 0
				g.stepNativeUnitPresent()
				control.nativeUnitPresent.drawn = true
				control.nativeUnitPresent.wait = 0
				control.stepNativeUnitPresent()
				if !reflect.DeepEqual(compoundMapUnitSnapshot(g), compoundMapUnitSnapshot(&control)) || g.nativeRNGState != control.nativeRNGState || !bytes.Equal(g.nativeMapWork, control.nativeMapWork) || !bytes.Equal(g.nativeMapVGA, control.nativeMapVGA) {
					t.Fatalf("native state diverged after frame%d", i)
				}
			}
			if g.nativeUnitPresent != nil || control.nativeUnitPresent != nil {
				t.Fatal("owner did not return")
			}
		}
	}
}

func TestModernHDUnitPresentRejectsBeforePublication(t *testing.T) {
	for _, kind := range []string{"partial-bank", "wrong-size", "native-source", "work-source", "typed-source", "world-size"} {
		t.Run(kind, func(t *testing.T) {
			g, spec := unitPresentHDTestGame(t, false, false, true)
			if err := g.startNativeUnitPresent(spec, nil); err != nil {
				t.Fatal(err)
			}
			j := g.nativeUnitPresent
			beforeFrames, beforeTheme := j.hdFrames, j.hdTheme
			units, rng := compoundMapUnitSnapshot(g), g.nativeRNGState
			work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
			bad := *g.modernStoryPortraits
			switch kind {
			case "partial-bank":
				delete(bad.mapEffects.frames, 117)
			case "wrong-size":
				bad.mapEffects.frames[117] = image.NewRGBA(image.Rect(0, 0, 1, 1))
			case "native-source":
				j.vgaFrames[20][7] ^= 1
			case "work-source":
				j.workFrames[20][10] ^= 1
			case "typed-source":
				j.hdSource.Cursor.X++
			case "world-size":
				for _, u := range j.hdFrames[0].Units {
					bad.mapSpritesHD[u.Group][u.Frame] = image.NewRGBA(image.Rect(0, 0, 24, 24))
					break
				}
			}
			if _, err := g.stageModernHDUnitPresent(&bad); err == nil {
				t.Fatal("invalid source admitted", kind)
			}
			if j.hdTheme != beforeTheme || len(j.hdFrames) != len(beforeFrames) || j.hdFrames[0] != beforeFrames[0] || rng != g.nativeRNGState || !reflect.DeepEqual(units, compoundMapUnitSnapshot(g)) || !bytes.Equal(work, g.nativeMapWork) || !bytes.Equal(vga, g.nativeMapVGA) {
				t.Fatal("rejection published/mutated live state")
			}
		})
	}
}

func TestModernHDUnitPresentStagesPendingLegsAtomically(t *testing.T) {
	g, spec := unitPresentHDTestGame(t, false, false, true)
	a, err := g.buildNativeUnitPresentJob(spec, g.st, g.nativeMapWork, g.nativeMapVGA, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := g.buildNativeUnitPresentJob(spec, g.st, g.nativeMapWork, g.nativeMapVGA, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.hdNext = b
	g.nativeModifierPresentation = &nativeCommandModifierPresentationJob{unitPresent: a}
	g.focusJob = &focusUnitJob{unitPresent: a}
	bad := *g.modernStoryPortraits
	b.vgaFrames[10][40] ^= 1
	if _, err := g.stageModernHDUnitPresent(&bad); err == nil {
		t.Fatal("bad next leg was not preflighted")
	}
	if a.hdTheme != g.modernStoryPortraits || b.hdTheme != g.modernStoryPortraits {
		t.Fatal("partial pending publication")
	}
	b.vgaFrames[10][40] ^= 1
	publish, err := g.stageModernHDUnitPresent(nil)
	if err != nil {
		t.Fatal(err)
	}
	publish()
	if a.hdFrames != nil || b.hdFrames != nil {
		t.Fatal("native switch retained pending HD")
	}
	publish, err = g.stageModernHDUnitPresent(g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	publish()
	if len(a.hdFrames) != 51 || len(b.hdFrames) != 51 {
		t.Fatal("pending legs did not restore together")
	}
}

func TestModernHDUnitPresentLegacyEffectAndLinearRowWrap(t *testing.T) {
	g, spec := unitPresentHDTestGame(t, true, false, false)
	if err := g.startNativeUnitPresent(spec, nil); err != nil {
		t.Fatal(err)
	}
	if len(g.nativeUnitPresent.hdFrames) != 45 {
		t.Fatal("old theme lost HD world")
	}
	// Exercise byte-address wrapping with a coloured source world. The native
	// strip primitive permits a row to cross the 320-byte destination stride.
	f := &modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200)), UI: image.NewRGBA(image.Rect(0, 0, 320, 200))}
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			f.Background.SetRGBA(x, y, color.RGBA{byte(x), byte(y), 90, 255})
		}
	}
	layout := indexedmap.NativeUnitPresentStripLayout{WorkOffset: unitPresentHDBase + 20*456 + 10, VGAOffset: 30*320 + 312, Rows: 2}
	rows, err := unitPresentHDStripRows(f, layout, make([]byte, indexedmap.NativeUnitPresentWorkSize), g.nativeMapAssets.Palette)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || len(rows[0]) != 2 || rows[0][0].Rect != image.Rect(1872, 180, 1920, 186) || rows[0][1].Rect != image.Rect(0, 186, 96, 192) {
		t.Fatal("VGA byte wrap was clipped or widened")
	}
	if rows[0][1].Image.RGBAAt(0, 0) != (color.RGBA{18, 20, 90, 255}) {
		t.Fatal("wrapped segment used wrong source offset")
	}
}
