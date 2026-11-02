package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func titleInterfaceTestAssets(t *testing.T) *titleAssets {
	t.Helper()
	a := &titleAssets{}
	if err := loadSeparatedTitleFDOTHER(separatedAssetPath(""), a); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestModernHDTitleInterfaceSources(t *testing.T) {
	a := titleInterfaceTestAssets(t)
	f := loadFont()
	if f == nil {
		t.Fatal("font unavailable")
	}
	count := 0
	glyphPixels := 0
	for item := 0; item < 3; item++ {
		for state := 0; state < 2; state++ {
			h := a.hdItems[item][state]
			if !h.valid(item, state, a.items[item][state]) {
				t.Fatal("source not bound", item, state)
			}
			before := titleMenuHDSeal(h)
			source, err := fdother.LoadSeparatedNestedSingleFrame(separatedAssetPath("surfaces"), 7, 1+item*2+state)
			if err != nil {
				t.Fatal(err)
			}
			_, palette, err := fdother.LoadSeparatedFDOTHERPalette(separatedAssetPath("palette"), 8)
			if err != nil {
				t.Fatal(err)
			}
			want, _, row := titlePrototypeItem(t, source, palette, f, item)
			if !bytes.Equal(want.Indexed, h.blank) || !bytes.Equal(source.Mask, h.source.Mask) {
				t.Fatal("blank differs from independent prototype")
			}
			for y := 0; y < source.Height; y++ {
				for x := 0; x < source.Width; x++ {
					at := y*source.Width + x
					if x < 7 || x >= source.Width-7 {
						if h.blank[at] != source.Indexed[at] {
							t.Fatal("jewel or outside text changed")
						}
					}
					if source.Indexed[at] != h.blank[at] {
						glyphPixels++
					}
				}
			}
			if row["label"] != []string{"START", "LOAD", "CONTINUE"}[item] {
				t.Fatal("source label changed")
			}
			for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
				for _, scale := range []float64{0.6, 1, 1.6} {
					g := &Game{font: f, fontScale: scale, localeID: locale}
					runs, ok := g.titleMenuHDRuns(h)
					if !ok || len(runs) != 2 || runs[1].text != row["label"] {
						t.Fatal("text admission")
					}
					run := runs[1]
					font := f
					if regional := f.hdFonts[locale]; regional != nil {
						font = regional
					}
					face, baseline := modernHDTextFace(font, run, scale)
					bounds := text.BoundString(face, run.text)
					moved := bounds.Add(image.Pt(int(run.x*3+0.5), int(baseline+0.5)))
					clip := image.Rectangle{Min: run.clip.Min.Mul(3), Max: run.clip.Max.Mul(3)}
					if !moved.In(clip) {
						t.Fatal("text escaped original clip", locale, scale, moved, clip)
					}
					count++
				}
			}
			if before != titleMenuHDSeal(h) {
				t.Fatal("layout mutated frozen source")
			}
		}
	}
	t.Logf("six title sources; %d locale/size layouts; %d replaced ink pixels; jewels/masks kept", count, glyphPixels)
}

func TestModernHDTitleInterfaceRejectsSource(t *testing.T) {
	a := titleInterfaceTestAssets(t)
	base := a.hdItems[0][0]
	f := loadFont()
	faults := []struct {
		name string
		edit func(*modernHDTitleMenuItem, *titleAssets)
	}{
		{"missing", func(h *modernHDTitleMenuItem, a *titleAssets) { a.hdItems[0][0] = nil }},
		{"owner", func(h *modernHDTitleMenuItem, a *titleAssets) { h.item = 1 }},
		{"selected", func(h *modernHDTitleMenuItem, a *titleAssets) { h.state = 1 }},
		{"source_pixels", func(h *modernHDTitleMenuItem, a *titleAssets) { h.source.Indexed[0] ^= 1 }},
		{"source_mask", func(h *modernHDTitleMenuItem, a *titleAssets) { h.source.Mask[0] ^= 255 }},
		{"blank", func(h *modernHDTitleMenuItem, a *titleAssets) { h.blank[0] ^= 1 }},
		{"palette", func(h *modernHDTitleMenuItem, a *titleAssets) { h.palette[0] = color.RGBA{R: 1, A: 255} }},
		{"foreground", func(h *modernHDTitleMenuItem, a *titleAssets) { h.foreground.R ^= 1 }},
		{"picture", func(h *modernHDTitleMenuItem, a *titleAssets) { h.picture = ebiten.NewImage(61, 7) }},
		{"original", func(h *modernHDTitleMenuItem, a *titleAssets) { a.items[0][0] = ebiten.NewImage(61, 7) }},
	}
	for _, fault := range faults {
		t.Run(fault.name, func(t *testing.T) {
			copyAssets := *a
			h := *base
			h.source.Indexed = append([]byte(nil), base.source.Indexed...)
			h.source.Mask = append([]byte(nil), base.source.Mask...)
			h.blank = append([]byte(nil), base.blank...)
			h.palette = append(color.Palette(nil), base.palette...)
			copyAssets.hdItems[0][0] = &h
			fault.edit(&h, &copyAssets)
			g := &Game{titlePhase: "menu", titleAssets: &copyAssets, font: f, fontScale: 1, interfaceMode: playerInterfaceOriginal, titleSel: 2, titleFlash: 3}
			if err := g.setInterfaceMode(playerInterfaceHD); err == nil {
				t.Fatal("bad source admitted")
			}
			if g.interfaceMode != playerInterfaceOriginal || g.titleSel != 2 || g.titleFlash != 3 {
				t.Fatal("failure changed controls")
			}
			g.modernHDText = &modernHDTextFrame{}
			if g.drawModernHDTitleMenuItem(ebiten.NewImage(640, 400), &copyAssets, 0, 0, copyAssets.items[0][0]) || len(g.modernHDText.runs) != 0 {
				t.Fatal("bad source retained HD")
			}
		})
	}
	dac, palette, err := fdother.LoadSeparatedFDOTHERPalette(separatedAssetPath("palette"), 8)
	if err != nil {
		t.Fatal(err)
	}
	source := base.source
	badShape := source
	badShape.Width, badShape.Height = 7, 61
	if _, err := freezeModernHDTitleMenuItem(badShape, dac, palette, 0, 0, base.original); err == nil {
		t.Fatal("same count wrong geometry admitted")
	}
	for _, field := range []string{"pixels", "mask", "dac", "palette"} {
		s := source
		s.Indexed = append([]byte(nil), source.Indexed...)
		s.Mask = append([]byte(nil), source.Mask...)
		d := append([]byte(nil), dac...)
		p := append(color.Palette(nil), palette...)
		switch field {
		case "pixels":
			s.Indexed[0] ^= 1
		case "mask":
			s.Mask[0] ^= 255
		case "dac":
			d[0] ^= 1
		case "palette":
			p[0] = color.RGBA{R: 1, A: 255}
		}
		if _, err := freezeModernHDTitleMenuItem(s, d, p, 0, 0, base.original); err == nil {
			t.Fatal("factory admitted", field)
		}
	}
	t.Log("ten live source faults rejected atomically; five factory faults rejected")
}

func TestModernHDTitleLoadSlotsChromeSource(t *testing.T) {
	t.Setenv("FD2_NATIVE_SAVE", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	a, err := loadNativeLoadSlotsUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	if !a.titleChromeHDValid() {
		t.Fatal("source chrome unavailable")
	}
	if fmt.Sprintf("%x", sha256.Sum256(a.dialogueBox.Pixels)) != "c009ad5ed72466295e3c60d4cb5117c3331444bcbe23d5d1befd2f413b15ac00" {
		t.Fatal("panel changed")
	}
	_, glyphs, err := loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{font: loadFont(), nativeLoadSlotsUI: a, nativeBattleGlyphs: glyphs, modernHDText: &modernHDTextFrame{}}
	screen := ebiten.NewImage(640, 400)
	seal := a.hd.seal
	for selection := 0; selection < 4; selection++ {
		g.titleSlotSel = selection
		if !g.drawNativeLoadSlots(screen) || len(g.modernHDText.panels) != 1 || g.modernHDText.panels[0].gold || g.modernHDText.panels[0].rect != image.Rect(5, 112, 315, 198) {
			t.Fatal("slot border/clear", selection)
		}
		if a.hd.seal != seal || !a.titleChromeHDValid() {
			t.Fatal("Draw changed source")
		}
	}
	faults := []struct {
		name string
		edit func(*nativeLoadSlotsUIAssets)
	}{
		{"missing", func(a *nativeLoadSlotsUIAssets) { a.hd = nil }},
		{"panel", func(a *nativeLoadSlotsUIAssets) { a.dialogueBox.Pixels[0] ^= 1 }},
		{"palette", func(a *nativeLoadSlotsUIAssets) { a.palette[0] = color.RGBA{R: 1, A: 255} }},
		{"cell", func(a *nativeLoadSlotsUIAssets) { a.hd.cells[1].Pixels[0] ^= 1 }},
		{"chrome", func(a *nativeLoadSlotsUIAssets) { a.hd.chrome = &modernHDDialogueChrome{} }},
		{"edge", func(a *nativeLoadSlotsUIAssets) { a.hd.chrome.edges[0][0].R ^= 1 }},
	}
	for _, fault := range faults {
		t.Run(fault.name, func(t *testing.T) {
			copyAssets := *a
			copyAssets.dialogueBox.Pixels = append([]byte(nil), a.dialogueBox.Pixels...)
			copyAssets.palette = append(color.Palette(nil), a.palette...)
			h := *a.hd
			h.cells = append([]fdother.RawCell(nil), h.cells...)
			for i := range h.cells {
				h.cells[i].Pixels = append([]byte(nil), h.cells[i].Pixels...)
			}
			chrome := *h.chrome
			h.chrome = &chrome
			h.chromeIdentity = h.chrome
			copyAssets.hd = &h
			fault.edit(&copyAssets)
			game := &Game{font: g.font, nativeLoadSlotsUI: &copyAssets, nativeBattleGlyphs: glyphs, modernHDText: &modernHDTextFrame{}, titlePhase: "loadslots", interfaceMode: playerInterfaceOriginal, titleSlotSel: 2}
			if copyAssets.titleChromeHDValid() || game.setInterfaceMode(playerInterfaceHD) == nil || game.interfaceMode != playerInterfaceOriginal || game.titleSlotSel != 2 {
				t.Fatal("slot fault admitted or controls changed")
			}
			if !game.drawNativeLoadSlots(screen) || len(game.modernHDText.panels) != 0 {
				t.Fatal("bad source retained panel")
			}
		})
	}
	t.Log("four slot selections; six source faults rejected; native fallback kept without stale panel")
}
