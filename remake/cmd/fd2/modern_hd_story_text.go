package main

import (
	"image"
	"unicode/utf8"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"golang.org/x/image/font/sfnt"
)

// Replaces only the existing native text window. Portrait, arrow, frame and
// outside pixels remain from the actual frame selected by the game clock.
func (g *Game) modernHDStoryTextFrame(frame []byte, progress int) ([]byte, []modernHDTextRun) {
	if g.modernHDText == nil || g.font == nil || g.nativeDialogueLayout == nil || len(g.nativeDialoguePortraits) == 0 {
		return frame, nil
	}
	layout := g.nativeDialogueLayout
	x, y, width, step, rows, err := campaign.NativeStoryDialogueTextGeometry(layout.Control)
	if err != nil {
		return frame, nil
	}
	base, err := campaign.ComposeNativeStoryDialogueBaseFrame(g.nativeMapVGA, g.nativeClassUI.dialogue, g.nativeDialoguePortraits[0], layout)
	if err != nil {
		return frame, nil
	}
	window := image.Rect(x-1, y, x+width, y+72).Intersect(image.Rect(0, 0, 320, 200))
	var glyphs []campaign.NativeStoryVisibleGlyph
	if g.localeID == "zh-Hant" || g.localeID == "" {
		glyphs, err = campaign.NativeStoryDialogueVisibleGlyphs(layout, g.dlgPage, progress)
		if err != nil {
			return frame, nil
		}
	} else {
		remaining := progress
		for row, line := range layout.Pages[g.dlgPage] {
			runes := []rune(line)
			n := len(runes)
			if n > remaining {
				n = remaining
			}
			if n > 0 {
				glyphs = append(glyphs, campaign.NativeStoryVisibleGlyph{Token: string(runes[:n]), X: x, Y: y + row*step})
			}
			remaining -= n
			if remaining <= 0 {
				break
			}
		}
		window.Max.Y = y + step*rows
	}
	result := append([]byte(nil), frame...)
	for py := window.Min.Y; py < window.Max.Y; py++ {
		copy(result[py*320+window.Min.X:py*320+window.Max.X], base[py*320+window.Min.X:py*320+window.Max.X])
	}
	clip := image.Rect(window.Min.X*2, window.Min.Y*2, window.Max.X*2, window.Max.Y*2)
	var runs []modernHDTextRun
	var buffer sfnt.Buffer
	for _, glyph := range glyphs {
		known := true
		for _, r := range glyph.Token {
			id, e := g.font.sf.GlyphIndex(&buffer, r)
			if e != nil || id == 0 {
				known = false
			}
		}
		if (g.localeID == "zh-Hant" || g.localeID == "") && utf8.RuneCountInString(glyph.Token) != 1 {
			known = false
		}
		if (g.localeID == "zh-Hant" || g.localeID == "") && glyph.GlyphID >= 0 {
			if canonical, ok := g.nativeBattleGlyphs[glyph.Token]; !ok || canonical != glyph.GlyphID {
				known = false
			}
		}
		if !known {
			// Explicit fallback for original custom glyph tokens; no guessed Unicode.
			r := image.Rect(glyph.X-1, glyph.Y, glyph.X+16, glyph.Y+17).Intersect(window)
			for py := r.Min.Y; py < r.Max.Y; py++ {
				copy(result[py*320+r.Min.X:py*320+r.Max.X], frame[py*320+r.Min.X:py*320+r.Max.X])
			}
			continue
		}
		for _, layer := range []struct {
			offset int
			index  byte
		}{{1, 0x4c}, {0, 0xcd}} {
			runs = append(runs, modernHDTextRun{font: g.font, text: glyph.Token, x: float64(glyph.X*2) + float64(layer.offset)/3, y: float64(glyph.Y*2) + float64(layer.offset)/3, pixels: 28,
				color: modernHDNativeTextColor(g.nativeClassUI.palette, layer.index), clip: clip})
		}
	}
	return result, runs
}
