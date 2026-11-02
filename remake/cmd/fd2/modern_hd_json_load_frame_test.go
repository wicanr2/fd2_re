package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// This source-only disposable prototype uses the already approved thin bevel.
// The existing semi-transparent JSON background remains untouched.
func TestModernHDJSONLoadFramePrototype(t *testing.T) {
	out := os.Getenv("FD2_HD_JSON_LOAD_PROTOTYPE_OUT")
	if out == "" {
		t.Skip("explicit prototype output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	a, err := loadNativeLoadSlotsUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	if !a.titleChromeHDValid() {
		t.Fatal("verified bevel source unavailable")
	}
	c := *a.hd.chrome
	c.background = color.RGBA{}
	c.gpu = nil
	pic := c.patchFor(1080, 750)
	covered := 0
	for y := 0; y < 750; y++ {
		for x := 0; x < 1080; x++ {
			pixel := pic.RGBAAt(x, y)
			edge := x < 3 || y < 3 || x >= 1077 || y >= 747
			if edge {
				if pixel.A != 255 {
					t.Fatal("thin edge missing")
				}
				covered++
			} else if pixel != (color.RGBA{}) {
				t.Fatal("prototype changed interior")
			}
		}
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 1080, 750))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{8, 20, 48, 240}), image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), pic, image.Point{}, draw.Over)
	f, err := os.Create(filepath.Join(out, "thin-frame-prototype.png"))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, canvas)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(out, "thin-frame-prototype.png"))
	report := map[string]any{"status": "PASS-SOURCE-PROTOTYPE", "width": 1080, "height": 750, "edge_pixels": covered, "border_width": 3, "interior_unchanged": true, "png_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "source": "verified existing R67 native load-slot thin bevel; no new palette or source art", "scope": "isolated border prototype only; no production hook or normal route"}
	raw, _ = json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "prototype.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestModernHDJSONLoadFrameSource(t *testing.T) {
	t.Setenv("FD2_NATIVE_SAVE", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	if err := os.MkdirAll(filepath.Dir(saveSlotPath(0)), 0755); err != nil {
		t.Fatal(err)
	}
	source := []byte(`{"chapter":0}`)
	if err := os.WriteFile(saveSlotPath(0), source, 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := nativeLoadSlotMetadata(); ok {
		t.Fatal("JSON0 entered native metadata")
	}
	a, err := loadNativeLoadSlotsUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	if !a.titleJSONChromeHDValid() || a.hd.jsonChrome == a.hd.chrome {
		t.Fatal("independent verified chrome unavailable")
	}
	g := &Game{font: loadFont(), titlePhase: "loadslots", nativeLoadSlotsUI: a, modernThemeCache: modifierHDTestTheme(t), interfaceMode: playerInterfaceOriginal, titleSlotSel: 2}
	if err := g.setInterfaceMode(playerInterfaceHD); err != nil {
		t.Fatal("healthy source rejected", err)
	}
	g.interfaceMode = playerInterfaceOriginal
	faults := []struct {
		name string
		edit func(*modernHDTitleLoadSlots)
	}{
		{"missing", func(h *modernHDTitleLoadSlots) { h.jsonChrome = nil }},
		{"alias", func(h *modernHDTitleLoadSlots) { h.jsonChrome = h.chrome }},
		{"identity", func(h *modernHDTitleLoadSlots) { h.jsonIdentity = nil }},
		{"background", func(h *modernHDTitleLoadSlots) { h.jsonChrome.background.A = 1 }},
		{"edge", func(h *modernHDTitleLoadSlots) { h.jsonChrome.edges[0][0].R ^= 1 }},
		{"corner", func(h *modernHDTitleLoadSlots) { h.jsonChrome.corners[0][0].R ^= 1 }},
	}
	for _, fault := range faults {
		t.Run(fault.name, func(t *testing.T) {
			assets := *a
			h := *a.hd
			c := *h.jsonChrome
			h.jsonChrome = &c
			h.jsonIdentity = &c
			assets.hd = &h
			game := *g
			game.nativeLoadSlotsUI = &assets
			fault.edit(&h)
			if assets.titleJSONChromeHDValid() || game.setInterfaceMode(playerInterfaceHD) == nil || game.interfaceMode != playerInterfaceOriginal || game.titleSlotSel != 2 {
				t.Fatal("bad JSON frame source published")
			}
			game.modernHDText = &modernHDTextFrame{}
			game.appendModernHDJSONLoadFrame()
			if len(game.modernHDText.panels) != 0 {
				t.Fatal("bad source retained panel")
			}
		})
	}
	g.modernHDText = &modernHDTextFrame{}
	g.appendModernHDJSONLoadFrame()
	if len(g.modernHDText.panels) != 1 || g.modernHDText.panels[0].rect != image.Rect(70, 35, 250, 160) || g.modernHDText.panels[0].gold {
		t.Fatal("wrong JSON fallback geometry")
	}
	// The native and JSON owners retain separate caches for different rectangles.
	if a.hd.chrome.gpu != nil || a.hd.jsonChrome.gpu != nil {
		t.Fatal("source collection eagerly shared GPU")
	}
	after, err := os.ReadFile(saveSlotPath(0))
	if err != nil || string(after) != string(source) {
		t.Fatal("source bytes changed", err)
	}
	t.Log("healthy F11, JSON0 boundary, six clone/source faults and source bytes kept")
}
