package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// parallelThemeFixture 寫一份只有一張合格故事頭像的最小清冊與素材目錄。
func parallelThemeFixture(t *testing.T) (string, modernThemeAsset) {
	t.Helper()
	root := t.TempDir()
	picture := image.NewNRGBA(image.Rect(0, 0, 80, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			picture.SetNRGBA(x, y, color.NRGBA{R: 40, G: 90, B: 160, A: 255})
		}
	}
	raw := encodeBattleHDFixturePNG(t, picture)
	if err := os.WriteFile(filepath.Join(root, "portrait.png"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	return root, modernThemeAsset{Role: "story_portrait_frame", Status: "runtime_ready",
		ConsumerContract: "native_story_dialogue_rgba_overlay_v1", Width: 80, Height: 80, SpeakerID: 1,
		MouthState: "closed", File: "portrait.png", SHA256: hex.EncodeToString(digest[:])}
}

func writeParallelThemeCatalog(t *testing.T, root string, assets []modernThemeAsset) string {
	t.Helper()
	raw, err := json.Marshal(modernThemeCatalog{SchemaVersion: 1, ThemeID: modernHandpaintedThemeID, Assets: assets})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "catalog.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// 分組並行載入回報的錯誤必須和逐項依序載入相同：清冊中第一個失敗的項目；任一項失敗整包
// 拒收。單一載入流（一個處理器）時退化成依序載入，結果相同。
func TestModernThemeGroupedLoadReportsFirstCatalogFailure(t *testing.T) {
	badTerrain := modernThemeAsset{Role: "native_terrain_hd_set", Status: "runtime_ready"}
	badSurface := modernThemeAsset{Role: "battle_surface_hd", SourceArchive: "XX.DAT"}
	badHUD := modernThemeAsset{Role: "battle_hud_panel", Status: "runtime_ready"}
	scaledTerrain := badTerrain
	scaledTerrain.VisualScalePercent = 115
	cases := []struct {
		name   string
		assets []modernThemeAsset
		want   string
	}{
		{"grouped-first", []modernThemeAsset{badTerrain, badHUD}, "modern HD terrain"},
		{"main-first", []modernThemeAsset{badHUD, badTerrain}, "modern battle HUD"},
		{"two-groups", []modernThemeAsset{badSurface, badTerrain}, "modern HD battle surface"},
		{"two-groups-reversed", []modernThemeAsset{badTerrain, badSurface}, "modern HD terrain"},
		{"grouped-visual-scale", []modernThemeAsset{scaledTerrain, badHUD}, "visual scale requires"},
	}
	for _, procs := range []int{1, runtime.GOMAXPROCS(0)} {
		old := runtime.GOMAXPROCS(procs)
		for _, c := range cases {
			root, portrait := parallelThemeFixture(t)
			path := writeParallelThemeCatalog(t, root, append([]modernThemeAsset{portrait}, c.assets...))
			set, err := loadModernStoryPortraitSet(path, root)
			if err == nil || set != nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("procs=%d %s: got %v, want error containing %q", procs, c.name, err, c.want)
			}
		}
		root, portrait := parallelThemeFixture(t)
		set, err := loadModernStoryPortraitSet(writeParallelThemeCatalog(t, root, []modernThemeAsset{portrait}), root)
		if err != nil || set == nil || set.portraits[1] == nil {
			t.Errorf("procs=%d valid minimal catalog rejected: %v", procs, err)
		}
		runtime.GOMAXPROCS(old)
	}
}

// 載入進度單調遞增，最後等於清冊項目數。
func TestModernThemeLoadProgressCountsEveryAsset(t *testing.T) {
	root, portrait := parallelThemeFixture(t)
	concept := modernThemeAsset{Role: "portrait_concept"}
	path := writeParallelThemeCatalog(t, root, []modernThemeAsset{portrait, concept, concept})
	last, total := 0, 0
	if _, err := loadModernStoryPortraitSetWithProgress(path, root, func(done, n int) {
		if done < last {
			t.Errorf("progress went backwards: %d after %d", done, last)
		}
		last, total = done, n
	}); err != nil {
		t.Fatal(err)
	}
	if total != 3 || last != 3 {
		t.Fatalf("progress ended at %d/%d", last, total)
	}
}
