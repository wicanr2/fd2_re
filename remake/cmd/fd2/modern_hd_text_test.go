package main

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestModernHDTextRequiresCompleteHDAndKeepsLogicalMetrics(t *testing.T) {
	f := loadFont()
	if f == nil {
		t.Fatal("maintained test image requires Noto CJK")
	}
	for _, locale := range []string{"zh-Hant", "zh-Hans", "ja", "en"} {
		if f.hdFonts[locale] == nil {
			t.Fatalf("maintained Noto TTC lacks verified regional face for %s", locale)
		}
	}
	screen := ebiten.NewImage(logicalW, logicalH)
	g := &Game{font: f}
	done := g.beginModernHDText(screen)
	if f.hd != nil {
		t.Fatal("faithful mode enabled HD text")
	}
	done()
	g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
	before := f.Width("索爾 Thor 日本語", 1)
	done = g.beginModernHDText(screen)
	f.Draw(screen, "索爾 Thor 日本語", 40, 50, 1, color.RGBA{255, 255, 255, 255})
	if len(g.modernHDText.runs) != 1 || g.modernHDText.runs[0].pixels != 18 {
		t.Fatal("HD run lost logical font size")
	}
	if f.Width("索爾 Thor 日本語", 1) != before {
		t.Fatal("HD changed wrapping metrics")
	}
	g.clearModernHDText()
	if len(g.modernHDText.runs) != 0 {
		t.Fatal("opaque native panel left previous text")
	}
	done()
	if f.hd != nil {
		t.Fatal("HD recording leaked outside Draw")
	}
}
