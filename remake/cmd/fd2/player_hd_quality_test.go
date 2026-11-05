package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"testing"
)

func TestHDTextureQualityKeepsSourceAndAlpha(t *testing.T) {
	source := mapReusePicture(144, 144, color.RGBA{0, 0, 0, 0})
	for y := 30; y < 114; y++ {
		for x := 30; x < 114; x++ {
			source.SetRGBA(x, y, color.RGBA{180, 100, 20, 255})
		}
	}
	before := append([]byte(nil), source.Pix...)
	for _, tc := range []struct {
		quality hdTextureQuality
		size    int
	}{{hdTextureFull, 144}, {hdTextureBalanced, 96}, {hdTextureLight, 72}} {
		got := tc.quality.textureSource(source)
		if got.Bounds() != image.Rect(0, 0, tc.size, tc.size) {
			t.Fatal(tc.quality, got.Bounds())
		}
		r, g, b, a := got.At(0, 0).RGBA()
		if r|g|b|a != 0 {
			t.Fatal("transparent border gained colour")
		}
		r, g, b, a = got.At(tc.size/2, tc.size/2).RGBA()
		if r != 180*257 || g != 100*257 || b != 20*257 || a != 65535 {
			t.Fatal("opaque core changed", r, g, b, a)
		}
		if !bytes.Equal(before, source.Pix) {
			t.Fatal("source art was modified")
		}
	}
	if hdTextureFull.textureSource(source) != source {
		t.Fatal("full quality no longer uses exact source")
	}
}

func TestHDQualityRasterCoordinatesAndSceneryCache(t *testing.T) {
	for _, q := range []hdTextureQuality{hdTextureFull, hdTextureBalanced, hdTextureLight} {
		want := []image.Point{{1920, 1200}, {1280, 800}, {960, 600}}[q]
		if q.rasterBounds().Size() != want {
			t.Fatal(q, q.rasterBounds())
		}
		r := image.Rect(-6, 24, 1896, 1176)
		got := q.rasterRect(r)
		if got.Min.X != -q.numerator() || got.Min.Y != 4*q.numerator() || got.Max.X != 316*q.numerator() {
			t.Fatal("native coordinate lost", q, got)
		}
		if q.rasterRect(image.Rect(-1, -1, 1, 1)).Min.X != -1 {
			t.Fatal("negative clip truncated")
		}
	}
	f, _ := mapReuseFrame()
	f.TerrainSeed = []byte{1, 2, 3}
	f.TerrainPalette[1] = color.RGBA{20, 30, 40, 255}
	want := modernHDSceneryFrameKey(f)
	f.Units[0].Frame++
	f.UI.Pix[0]++
	if modernHDSceneryFrameKey(f) != want {
		t.Fatal("actors/UI invalidate scenery-only cache")
	}
	f.TerrainSeed[0]++
	if modernHDSceneryFrameKey(f) == want {
		t.Fatal("native terrain change reused scenery")
	}
	f.TerrainSeed[0]--
	f.TerrainPalette[1].R++
	if modernHDSceneryFrameKey(f) == want {
		t.Fatal("palette change reused scenery")
	}
}

func TestHDTextureQualitySettingsAndToolbar(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	old := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = old })
	if got := loadSettings().HDTextureQuality; got != "balanced" {
		t.Fatal("new settings", got)
	}
	for _, raw := range []string{`{}`, `{"hd_texture_quality":"unknown"}`} {
		if err := os.WriteFile(settingsPath(), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if got := loadSettings().HDTextureQuality; got != "balanced" {
			t.Fatal("old/invalid settings", got)
		}
	}
	g := &Game{playerHDQuality: hdTextureBalanced}
	p := &modernHDPresentation{Game: g}
	p.poster.menuOpen = true
	before, _ := json.Marshal(saveData{})
	for i, rect := range playerHDQualityChoices {
		if !p.queuePosterPoint(rect.Min.Add(image.Pt(2, 2))) || !p.poster.consumeInput {
			t.Fatal("quality click was not consumed")
		}
		if g.playerHDQuality != hdTextureQuality(i) || loadSettings().HDTextureQuality != hdTextureQuality(i).name() {
			t.Fatal("quality did not persist", i)
		}
		if g.playerThemeRequested || g.playerInterfaceRequested || g.playerSpeedRequested {
			t.Fatal("quality click changed another control")
		}
	}
	after, _ := json.Marshal(saveData{})
	if !bytes.Equal(before, after) || bytes.Contains(after, []byte("hd_texture_quality")) {
		t.Fatal("display preference entered campaign save")
	}
}
