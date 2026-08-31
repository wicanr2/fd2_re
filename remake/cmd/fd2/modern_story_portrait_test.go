package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeModernPortraitFixture(t *testing.T, opaque bool) (catalogPath, root string) {
	t.Helper()
	root = t.TempDir()
	portrait := image.NewNRGBA(image.Rect(0, 0, 80, 80))
	alpha := uint8(0xff)
	if !opaque {
		alpha = 0x80
	}
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			portrait.SetNRGBA(x, y, color.NRGBA{R: 20, G: 40, B: 90, A: alpha})
		}
	}
	pngPath := filepath.Join(root, "sol.png")
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, portrait); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	document := map[string]any{
		"schema_version": 1,
		"theme_id":       modernHandpaintedThemeID,
		"assets": []map[string]any{{
			"role": "story_portrait_frame", "status": "runtime_candidate",
			"file": "sol.png", "width": 80, "height": 80,
			"sha256":            hex.EncodeToString(digest[:]),
			"consumer_contract": "native_story_dialogue_rgba_overlay_v1",
			"speaker_id":        0, "frame": 0, "mouth_state": "closed",
		}},
	}
	catalogRaw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	catalogPath = filepath.Join(root, "catalog.json")
	if err := os.WriteFile(catalogPath, catalogRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	return catalogPath, root
}

func TestLoadModernStoryPortraitSetValidatesIdentityGeometryAndOpacity(t *testing.T) {
	catalogPath, root := writeModernPortraitFixture(t, true)
	set, err := loadModernStoryPortraitSet(catalogPath, root)
	if err != nil {
		t.Fatal(err)
	}
	portrait := set.portraits[0]
	if portrait == nil || portrait.image.Bounds().Dx() != 80 || portrait.image.Bounds().Dy() != 80 {
		t.Fatalf("portrait=%v", portrait)
	}
}

func TestLoadModernStoryPortraitSetRejectsTransparentFrame(t *testing.T) {
	catalogPath, root := writeModernPortraitFixture(t, false)
	if _, err := loadModernStoryPortraitSet(catalogPath, root); err == nil {
		t.Fatal("transparent modern portrait accepted")
	}
}

func TestLoadModernStoryPortraitSetRejectsDigestMismatch(t *testing.T) {
	catalogPath, root := writeModernPortraitFixture(t, true)
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	assets := document["assets"].([]any)
	assets[0].(map[string]any)["sha256"] = string(make([]byte, 64))
	raw, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(catalogPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadModernStoryPortraitSet(catalogPath, root); err == nil {
		t.Fatal("modern portrait digest mismatch accepted")
	}
}
