package main

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text"
)

func TestHD1ToolbarControlsStayOneRowAndHaveBoundedLabels(t *testing.T) {
	controls := []image.Rectangle{playerThemeButtons[0], playerThemeButtons[1], playerInterfaceButton, playerLocaleButton, playerFontButton, playerSpeedButton, playerPosterButton, playerFullscreenButton}
	for i, box := range controls {
		if !box.In(image.Rect(0, 0, 640, 24)) {
			t.Fatal("control outside one row", box)
		}
		for _, previous := range controls[:i] {
			if box.Overlaps(previous) {
				t.Fatal("controls overlap", box, previous)
			}
		}
	}
	f := loadFont()
	if f == nil {
		t.Fatal("Noto required")
	}
	for _, label := range []string{"UI: Original F11", "介面：原版 F11", "界面：原版 F11", "UI：原版 F11", "繁體中文 F4", "简体中文 F4", "日本語 F4", "English F4", "A+ 125%", "4× F8", "全螢幕", "全画面", "Full"} {
		for _, box := range controls {
			face, caption, at := playerToolbarLabelLayout(f, label, box)
			if face == nil {
				t.Fatal("missing font")
			}
			ink := text.BoundString(face, caption).Add(at)
			if !ink.In(image.Rect(box.Min.X*3+12, box.Min.Y*3+6, box.Max.X*3-12, box.Max.Y*3-6)) {
				t.Fatal("toolbar text escaped", label, ink, box)
			}
		}
	}
}

func TestHD1ToolbarQueuesOnlyTheSelectedControl(t *testing.T) {
	for _, item := range []struct {
		rect  image.Rectangle
		field string
	}{
		{playerInterfaceButton, "ui"}, {playerLocaleButton, "locale"}, {playerFontButton, "font"}, {playerSpeedButton, "speed"}, {playerFullscreenButton, "fullscreen"},
	} {
		g := &Game{}
		p := &modernHDPresentation{Game: g}
		p.queueToolbarPoint(image.Pt((item.rect.Min.X+item.rect.Max.X)/2, 12))
		got := map[string]bool{"ui": g.playerInterfaceRequested, "locale": g.playerLocaleRequested, "font": g.playerFontRequested, "speed": g.playerSpeedRequested, "fullscreen": g.playerFullscreenRequested}
		for name, value := range got {
			if value != (name == item.field) {
				t.Fatal("click affected another control", item.field, got)
			}
		}
		if g.playerThemeRequested || g.st != nil {
			t.Fatal("control click changed artwork or battle")
		}
	}
}

func TestHD1FontEnlargementKeepsMovingCropGeometry(t *testing.T) {
	f := loadFont()
	if f == nil {
		t.Fatal("Noto required")
	}
	run := modernHDTextRun{font: f, text: "索爾", x: 10, y: 20, pixels: 28, clip: image.Rect(10, 20, 74, 54)}.freezeLayout()
	face, baseline := modernHDTextFace(f, run, 1.25)
	partial := run
	partial.clip = partial.clip.Intersect(image.Rect(10, 20, 11, 54))
	other, otherBaseline := modernHDTextFace(f, partial, 1.25)
	if face != other || baseline != otherBaseline {
		t.Fatal("moving crop squeezed the text")
	}
	partial.x += 20
	partial.y += 30
	partial.clip = partial.clip.Add(image.Pt(20, 30))
	other, otherBaseline = modernHDTextFace(f, partial, 1.25)
	if face != other || otherBaseline-baseline != 90 {
		t.Fatal("moving text changed size or baseline")
	}
}

func TestHD1ConfirmationFontScalePreservesWholeLabels(t *testing.T) {
	ui, err := loadNativePreparationUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	class, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	c, err := freezeModernHDConfirmation(ui.choices, class.palette, loadFont())
	if err != nil {
		t.Fatal(err)
	}
	for _, scale := range fontScales {
		for locale, f := range c.fonts {
			f.SetUserScale(scale)
			for _, kind := range []fullRewardPhase{fullRewardOpening, fullRewardChoice, fullRewardChoiceClosing} {
				index := 6
				if kind == fullRewardChoiceClosing {
					index = 0
				}
				placements, err := c.placements(kind, index, 0, 1)
				if err != nil {
					t.Fatal(err)
				}
				for _, p := range placements {
					p.locale = locale
					face, label, at := c.labelLayout(p)
					ink := text.BoundString(face, label).Add(at)
					ink.Max = ink.Max.Add(image.Pt(1, 1))
					if label != c.labels[locale][p.option] || !ink.In(image.Rect(12, 12, 132, p.rect.Dy()*6-12)) {
						t.Fatal("scaled confirmation clipped a label or its shadow", locale, scale, ink)
					}
				}
			}
		}
	}
}
