package main

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func TestModernHDFacilityUnknownAndAmbiguousGlyphsKeepNativeInk(t *testing.T) {
	source, err := fdtxt.ParseFont(bytes.Repeat([]byte{255}, 4*fdtxt.GlyphBytes))
	if err != nil {
		t.Fatal(err)
	}
	f := loadFont()
	if f == nil {
		t.Fatal("maintained image requires Noto")
	}
	g := &Game{font: f, modernHDText: &modernHDTextFrame{}, nativeBattleGlyphs: map[string]int{"索": 0, "甲": 1, "乙": 1, "[special]": 2}}
	capture := &modernHDIndexedText{}
	adapter := g.modernHDFacilityFont(source, capture)
	style := fdtxt.NativeGlyphStyle{Foreground: 205, Shadow: 76}
	for index := 0; index < 4; index++ {
		a, b := make([]byte, 320*200), make([]byte, 320*200)
		if err := source.BlitNativeGlyph(a, 320, 119*320+12, index, style); err != nil {
			t.Fatal(err)
		}
		if err := adapter.BlitNativeGlyph(b, 320, 119*320+12, index, style); err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			if bytes.Equal(a, b) || len(capture.runs) != 2 {
				t.Fatal("known glyph was not replaced")
			}
		} else if !bytes.Equal(a, b) {
			t.Fatalf("unknown/ambiguous glyph %d changed", index)
		}
	}
	g.modernHDText = nil
	if g.modernHDFacilityFont(source, capture) != source {
		t.Fatal("faithful font was wrapped")
	}
}

func TestModernHDHotelSlotsCoverPromptAndRestoreSharedAssets(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	a, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	h, err := loadNativeHotelUIAssets(a)
	if err != nil {
		t.Fatal(err)
	}
	_, glyphs, err := loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{font: loadFont(), nativeClassUI: a, nativeHotelUI: h, nativeBattleGlyphs: glyphs, nativeHotelMode: "menu", modernStoryPortraits: &modernStoryPortraitSet{hasHD: true}}
	screen := ebiten.NewImage(logicalW, logicalH)
	legacy, ok := g.composeNativeHotelFrame(0)
	if !ok {
		t.Fatal("legacy hotel failed")
	}
	done := g.beginModernHDText(screen)
	if !g.drawNativeHotel(screen) || len(g.modernHDText.runs) == 0 {
		t.Fatal("hotel text missing")
	}
	if g.nativeClassUI != a || g.font.indexedHD != nil {
		t.Fatal("draw adapter leaked into state")
	}
	after, ok := g.composeNativeHotelFrame(0)
	if !ok || !bytes.Equal(legacy, after) {
		t.Fatal("HD changed faithful frame")
	}
	for _, mode := range []string{"slots", "loadslots", "saved", "loaded", "load_blocked"} {
		g.nativeHotelMode = mode
		if !g.drawNativeHotel(screen) || len(g.modernHDText.runs) == 0 {
			t.Fatalf("%s has no text", mode)
		}
		positions := map[[2]float64]bool{}
		for i := 1; i < len(g.modernHDText.runs); i += 2 {
			run := g.modernHDText.runs[i]
			position := [2]float64{run.x, run.y}
			if positions[position] {
				t.Fatalf("%s retained covered text at %v", mode, position)
			}
			positions[position] = true
		}
		if mode == "slots" || mode == "loadslots" {
			for _, run := range g.modernHDText.runs {
				if run.x == 24 {
					t.Fatal("slot overlay retained greeting")
				}
			}
		}
	}
	done()
}
