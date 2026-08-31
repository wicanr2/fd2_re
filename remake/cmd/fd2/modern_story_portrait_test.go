package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func TestLoadModernStoryPortraitSetAdmitsMultipleSpeakersAtomically(t *testing.T) {
	catalogPath, root := writeModernPortraitFixture(t, true)
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	first := document["assets"].([]any)[0].(map[string]any)
	second := make(map[string]any, len(first))
	for key, value := range first {
		second[key] = value
	}
	second["asset_id"] = "modern.ares.portrait.style_a.frame0"
	second["speaker_id"] = float64(1)
	document["assets"] = append(document["assets"].([]any), second)
	raw, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(catalogPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := loadModernStoryPortraitSet(catalogPath, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.portraits) != 2 || set.portraits[0] == nil || set.portraits[1] == nil {
		t.Fatalf("portraits=%v", set.portraits)
	}
}

func writeModernMapSpriteFixture(t *testing.T, alpha uint8) (catalogPath, root string) {
	t.Helper()
	catalogPath, root = writeModernPortraitFixture(t, true)
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	files := make([]string, 12)
	hashes := make([]string, 12)
	for frame := range files {
		files[frame] = fmt.Sprintf("sol-map-%02d.png", frame)
		img := image.NewNRGBA(image.Rect(0, 0, 24, 24))
		img.SetNRGBA(frame%24, frame/24, color.NRGBA{R: uint8(frame + 1), G: 40, B: 90, A: alpha})
		path := filepath.Join(root, files[frame])
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		pngRaw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(pngRaw)
		hashes[frame] = hex.EncodeToString(digest[:])
	}
	assets := document["assets"].([]any)
	document["assets"] = append(assets, map[string]any{
		"role": "map_sprite_set", "status": "runtime_candidate",
		"files": files, "width": 24, "height": 24, "frame_count": 12,
		"frame_sha256": hashes, "source_group": 68,
		"consumer_contract": "fdicon_map_sprite_12x24_v1",
		"alpha_contract":    "binary", "cycle_policy": "three_distinct_cycles",
	})
	raw, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(catalogPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return catalogPath, root
}

func TestLoadModernStoryPortraitSetAdmitsCompleteMapSpriteSet(t *testing.T) {
	catalogPath, root := writeModernMapSpriteFixture(t, 0xff)
	set, err := loadModernStoryPortraitSet(catalogPath, root)
	if err != nil {
		t.Fatal(err)
	}
	if frames := set.mapSprites[68]; len(frames) != 12 || frames[11].Bounds().Dx() != 24 {
		t.Fatalf("map sprite frames=%d", len(frames))
	}
}

func TestLoadModernStoryPortraitSetRejectsMapSpriteSoftAlpha(t *testing.T) {
	catalogPath, root := writeModernMapSpriteFixture(t, 0x80)
	if _, err := loadModernStoryPortraitSet(catalogPath, root); err == nil {
		t.Fatal("modern map sprite with soft alpha accepted")
	}
}
