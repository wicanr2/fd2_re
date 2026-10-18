package main

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

func modernHDMapRuntimeFixture(t *testing.T) *Game {
	a, m, st := completeNativeMapFrameFixture(t)
	set := &modernStoryPortraitSet{hasHD: true, mapSpritesHD: make(map[int][]image.Image)}
	picture := image.NewRGBA(image.Rect(0, 0, 144, 144))
	for group := 0; group < fdicon.NativeSourceGroupCount; group++ {
		if !fdicon.NativeSourceGroupHasPixels(group) {
			continue
		}
		set.mapSpritesHD[group] = make([]image.Image, 12)
		for frame := range set.mapSpritesHD[group] {
			set.mapSpritesHD[group][frame] = picture
		}
	}
	return &Game{nativeMapAssets: a, m: m, st: st, modernStoryPortraits: set, modernHDMapComposing: true}
}

func TestModernHDMapOverwriteCoverageAndDestinationDependency(t *testing.T) {
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{uint8(i), 0, 0, 255}
	}
	pixels, err := modernHDOverwriteLayer(5, palette, func(dst []byte) error { dst[0] = 0; dst[1] = 255; dst[3] = 30; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if pixels[0].A != 255 || pixels[1].A != 255 || pixels[2].A != 0 || pixels[3].R != 30 || pixels[4].A != 0 {
		t.Fatal("same-color/zero writes or transparent skip lost")
	}
	if _, err := modernHDOverwriteLayer(5, palette, func(dst []byte) error { dst[0]++; return nil }); err == nil {
		t.Fatal("destination-dependent blitter admitted as overwrite-only UI")
	}
}

func TestModernHDMapTransactionUsesOneNativeClockAndIndependentTerrain(t *testing.T) {
	hd := modernHDMapRuntimeFixture(t)
	native := modernHDMapRuntimeFixture(t)
	native.modernHDMapComposing = false
	for _, now := range []time.Time{time.Unix(500, 0), time.Unix(500, 300000000)} {
		if err := native.composeNativeMapFrameAt(now); err != nil {
			t.Fatal(err)
		}
		if err := hd.composeNativeMapFrameAt(now); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(native.nativeMapWork, hd.nativeMapWork) || !bytes.Equal(native.nativeMapVGA, hd.nativeMapVGA) ||
			!reflect.DeepEqual(native.nativeMapClock, hd.nativeMapClock) || native.st.NativeMapCycleState != hd.st.NativeMapCycleState ||
			!bytes.Equal(native.nativeMapDAC, hd.nativeMapDAC) {
			t.Fatal("HD advanced or changed the original transaction")
		}
		frame := hd.modernHDMapFrame
		if frame == nil || len(frame.Units) != 1 || len(frame.TerrainSeed) == 0 {
			t.Fatal("complete HD frame absent")
		}
		base := 0x8088
		for y := 0; y < 192; y++ {
			for x := 0; x < 312; x++ {
				if frame.TerrainSeed[base+y*456+x] != 1 {
					t.Fatal("terrain seed contains unit/range/HUD writes")
				}
			}
		}
	}
}

func TestModernHDMapRejectedArtDoesNotPublishNativeState(t *testing.T) {
	g := modernHDMapRuntimeFixture(t)
	if err := g.composeNativeMapFrameAt(time.Unix(500, 0)); err != nil {
		t.Fatal(err)
	}
	work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	clock, cycles, frame, seed := g.nativeMapClock, g.st.NativeMapCycleState, g.modernHDMapFrame, append([]byte(nil), g.modernHDMapTerrain...)
	// Remove the actual raw selector-cache result, not a guessed character ID.
	for _, unit := range frame.Units {
		delete(g.modernStoryPortraits.mapSpritesHD, unit.Group)
	}
	if err := g.composeNativeMapFrameAt(time.Unix(501, 0)); err == nil {
		t.Fatal("missing actual group accepted")
	}
	if !bytes.Equal(work, g.nativeMapWork) || !bytes.Equal(vga, g.nativeMapVGA) || !reflect.DeepEqual(clock, g.nativeMapClock) ||
		cycles != g.st.NativeMapCycleState || frame != g.modernHDMapFrame || !bytes.Equal(seed, g.modernHDMapTerrain) {
		t.Fatal("rejected HD frame published partial original state or pixels")
	}
}

func TestModernHDMapMosaicShaderCompiles(t *testing.T) {
	if _, err := ebiten.NewShader([]byte(modernHDMapMosaicSource)); err != nil {
		t.Fatal(err)
	}
}
