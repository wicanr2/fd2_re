package main

import (
	"image"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
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

func (g *Game) beginModernHDFacilityText() (*modernHDIndexedText, func()) {
	if g.modernHDText == nil || g.font == nil || g.nativeClassUI == nil {
		return nil, func() {}
	}
	capture := &modernHDIndexedText{palette: g.nativeClassUI.palette}
	original := g.nativeClassUI
	copy := *original
	copy.font = g.modernHDFacilityFont(original.font, capture)
	g.nativeClassUI = &copy
	return capture, func() { g.nativeClassUI = original }
}

func (g *Game) drawModernHDFacilityFrame(screen *ebiten.Image, compose func() ([]byte, bool), digits bool) bool {
	capture, restore := g.beginModernHDFacilityText()
	defer restore()
	if capture != nil {
		previous := g.font.indexedHD
		g.font.indexedHD = capture
		defer func() { g.font.indexedHD = previous }()
		if digits {
			g.nativeClassUI.digits = append([]fdother.Frame(nil), g.nativeClassUI.digits...)
			for digit, frame := range g.nativeClassUI.digits {
				g.nativeClassUI.digits[digit] = g.modernHDDecimalSprite(frame, digit)
			}
		}
	}
	frame, ok := compose()
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	if capture != nil {
		g.modernHDText.runs = append(g.modernHDText.runs, capture.runs...)
	}
	return true
}

func (g *Game) modernHDChurchItemAssets(assets battle.NativeItemPanelDataAssets) battle.NativeItemPanelDataAssets {
	if g.modernHDText != nil && g.modernHDShopText() {
		assets.Font = g.modernHDFacilityFont(assets.Font, g.font.indexedHD)
		assets.Frames = g.modernHDDecimalFrames(assets.Frames)
	}
	return assets
}

// Preparation owns a separate status font and decimal bank. Adapt value copies
// only during a stable Draw; its prebuilt transition frames keep native ink.
func (g *Game) beginModernHDPreparationText() (*modernHDIndexedText, func()) {
	if g.nativePreparationUI == nil {
		return nil, func() {}
	}
	capture, restoreClass := g.beginModernHDFacilityText()
	if capture == nil {
		return nil, restoreClass
	}
	previous := g.font.indexedHD
	g.font.indexedHD = capture
	original := g.nativePreparationUI
	copy := *original
	copy.status = g.modernHDChurchItemAssets(original.status)
	g.nativePreparationUI = &copy
	return capture, func() {
		g.nativePreparationUI = original
		g.font.indexedHD = previous
		restoreClass()
	}
}
