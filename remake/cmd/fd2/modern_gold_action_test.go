package main

import (
	"bytes"
	"image"
	"testing"
)

func TestGoldActionAllNativeCellsKeepSourceAndState(t *testing.T) {
	palette := loadNativeUIPalette()
	_, cells := loadNativeActionCellsWithRaw(palette)
	if len(cells) != 78 {
		t.Fatal("full separated source pack required", len(cells))
	}
	for i := range cells {
		before := append([]byte(nil), cells[i].Pixels...)
		frame, err := buildGoldActionFrame(cells, palette, i, i%3 != 2)
		if err != nil {
			t.Fatalf("cell%d: %v", i, err)
		}
		if frame.Bounds() != image.Rect(0, 0, 144, 120) || frame.RGBAAt(0, 0).A != 0 {
			t.Fatal("bounds/corner changed", i)
		}
		if !bytes.Equal(before, cells[i].Pixels) {
			t.Fatal("original pixels changed", i)
		}
		face := frame.RGBAAt(20, 20)
		if i%3 == 1 {
			if face.R <= face.B {
				t.Fatal("selected cell lost red", i)
			}
		} else if face.B <= face.R {
			t.Fatal("normal/disabled cell lost navy", i)
		}
		rim := frame.RGBAAt(7, 60)
		if rim.R <= rim.B || rim.A != 255 {
			t.Fatal("gold rim unavailable", i)
		}
		if i < 2 {
			highlight := frame.RGBAAt(30, 13)
			if highlight.R < 80 || highlight.R <= highlight.B || highlight.G <= highlight.B {
				t.Fatal("translucent gold highlight lost warm channels", i, highlight)
			}
		}
	}
	g := &Game{nativeUIPalette: palette, nativeActionCellsRaw: cells}
	a, err := g.goldActionFrame(0, true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := g.goldActionFrame(0, true)
	if err != nil || a != b || len(g.goldActionFrames) != 1 {
		t.Fatal("steady redraw rebuilt texture")
	}
	if _, err := g.goldActionFrame(1, true); err != nil || len(g.goldActionFrames) != 2 {
		t.Fatal("selection cache mismatch")
	}
	g.resetGoldActionFrames()
	if len(g.goldActionFrames) != 0 {
		t.Fatal("theme change retained old texture")
	}
}

func TestGoldActionRejectsMissingGlyphSource(t *testing.T) {
	if _, err := buildGoldActionFrame(nil, nil, 0, true); err == nil {
		t.Fatal("missing source accepted")
	}
	palette := loadNativeUIPalette()
	_, cells := loadNativeActionCellsWithRaw(palette)
	if len(cells) != 78 {
		t.Fatal("source required")
	}
	cells[0].Pixels = nil
	if _, err := buildGoldActionFrame(cells, palette, 0, true); err == nil {
		t.Fatal("truncated glyph accepted")
	}
}
