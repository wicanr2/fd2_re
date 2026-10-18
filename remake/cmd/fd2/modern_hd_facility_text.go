package main

import (
	"image"
	"unicode/utf8"

	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
	"golang.org/x/image/font/sfnt"
)

// A caller uses this only around a fresh composition, then publishes the
// returned runs alongside that exact frame. Cached animation frames need their
// own explicit descriptors and must not use this adapter.
func (g *Game) modernHDFacilityFont(source *fdtxt.Font, capture *modernHDIndexedText) *fdtxt.Font {
	if g.modernHDText == nil || g.font == nil || source == nil || capture == nil {
		return source
	}
	keys := make(map[int]string)
	ambiguous := make(map[int]bool)
	for key, id := range g.nativeBattleGlyphs {
		if utf8.RuneCountInString(key) != 1 {
			continue
		}
		if prior, ok := keys[id]; ok && prior != key {
			ambiguous[id] = true
		}
		keys[id] = key
	}
	var buffer sfnt.Buffer
	return source.WithNativePresenter(func(stride, base, index int, style fdtxt.NativeGlyphStyle) bool {
		key, ok := keys[index]
		x, y := base%stride, base/stride
		if !ok || ambiguous[index] || stride != 320 || x < 1 || x+16 > 320 || y+17 > 200 {
			return false
		}
		for _, r := range key {
			id, err := g.font.sf.GlyphIndex(&buffer, r)
			if err != nil || id == 0 {
				return false
			}
		}
		clip := image.Rect((x-1)*2, y*2, (x+16)*2, (y+17)*2)
		for _, layer := range []struct {
			offset int
			index  byte
		}{{1, style.Shadow}, {0, style.Foreground}} {
			capture.runs = append(capture.runs, modernHDTextRun{font: g.font, text: key,
				x: float64(x*2) + float64(layer.offset)/3, y: float64(y*2) + float64(layer.offset)/3,
				pixels: 28, color: modernHDNativeTextColor(capture.palette, layer.index), clip: clip})
		}
		return true
	})
}

func (g *Game) beginModernHDHotelText() (*modernHDIndexedText, func()) {
	if g.modernHDText == nil || g.nativeClassUI == nil {
		return nil, func() {}
	}
	capture := &modernHDIndexedText{palette: g.nativeClassUI.palette}
	original := g.nativeClassUI
	copy := *original
	copy.font = g.modernHDFacilityFont(original.font, capture)
	g.nativeClassUI = &copy
	return capture, func() { g.nativeClassUI = original }
}
