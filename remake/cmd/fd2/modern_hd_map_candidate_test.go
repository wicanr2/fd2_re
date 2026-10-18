package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// Explicit external candidates produce a local review artifact. This test
// restores a real save, but does not claim keyboard-path or oracle evidence.
func TestModernHDMapActualCandidateWorld(t *testing.T) {
	dir, source := os.Getenv("FD2_HD_CANDIDATE_DIR"), os.Getenv("FD2_HD_CANDIDATE_SOURCE")
	save, reportPath := os.Getenv("FD2_HD_WORLD_SAVE"), os.Getenv("FD2_HD_WORLD_REPORT")
	if dir == "" || source == "" || save == "" || reportPath == "" {
		t.Skip("未提供完整HD候選、來源、既有存檔與本機輸出路徑")
	}
	manifestPath := filepath.Join(dir, "hd-map-sprites.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		CatalogSHA256 string `json:"catalog_sha256"`
		LayoutPolicy  string `json:"layout_policy"`
		Groups        []struct {
			SourceGroup int    `json:"source_group"`
			FrameCount  int    `json:"frame_count"`
			CyclePolicy string `json:"cycle_policy"`
			Frames      []struct {
				File   string `json:"file"`
				SHA256 string `json:"sha256"`
			} `json:"frames"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	manifestHash := sha256.Sum256(raw)
	catalogPath := os.Getenv("FD2_HD_PARENT_CATALOG")
	if catalogPath == "" {
		catalogPath = "../../assets/themes/modern/catalog.json"
	}
	catalogRaw, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	catalogHash := sha256.Sum256(catalogRaw)
	if hex.EncodeToString(catalogHash[:]) != manifest.CatalogSHA256 || len(manifest.Groups) != 125 || manifest.LayoutPolicy != "contain_22px_cell_feet_v1" {
		t.Fatal("候選來源或完整原生可見圖組不符")
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(catalogRaw, &catalog); err != nil {
		t.Fatal(err)
	}
	parents := make([]modernThemeAsset, 0, len(catalog.Assets))
	for _, asset := range catalog.Assets {
		if asset.Role != "map_sprite_hd_set" {
			parents = append(parents, asset)
		}
	}
	catalog.Assets = parents
	pack := t.TempDir()
	linked := make(map[string]bool)
	link := func(root, name string) {
		t.Helper()
		if name == "" || linked[name] {
			return
		}
		if filepath.Base(name) != name {
			t.Fatal("unsafe candidate name")
		}
		if err := os.Symlink(filepath.Join(root, name), filepath.Join(pack, name)); err != nil {
			t.Fatal(err)
		}
		linked[name] = true
	}
	for _, asset := range catalog.Assets {
		link(source, asset.File)
		for _, file := range asset.Files {
			link(source, file)
		}
	}
	frameCount := 0
	for _, group := range manifest.Groups {
		contract := "fdicon_hd_map_sprite_12x144_v1"
		if group.FrameCount == 3 {
			contract = "fdicon_hd_static_sprite_3x144_v1"
		}
		asset := modernThemeAsset{Role: "map_sprite_hd_set", Status: "runtime_candidate", SourceGroup: group.SourceGroup,
			Width: 144, Height: 144, FrameCount: group.FrameCount, CyclePolicy: group.CyclePolicy,
			ConsumerContract: contract, AlphaContract: "coverage", LayoutPolicy: manifest.LayoutPolicy}
		for _, frame := range group.Frames {
			link(dir, frame.File)
			asset.Files = append(asset.Files, frame.File)
			asset.FrameSHA256 = append(asset.FrameSHA256, frame.SHA256)
			frameCount++
		}
		catalog.Assets = append(catalog.Assets, asset)
	}
	combinedRaw, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	combined := filepath.Join(pack, "catalog.json")
	if err := os.WriteFile(combined, combinedRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := loadModernStoryPortraitSet(combined, pack)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.mapSpritesHD) != 125 || frameCount != 1455 {
		t.Fatal("實際清冊未完整載入")
	}
	saveRaw, err := os.ReadFile(save)
	if err != nil {
		t.Fatal(err)
	}
	saveHash := sha256.Sum256(saveRaw)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_CAMPAIGN", "")
	t.Setenv("FD2_THEME", "")
	t.Setenv("FD2_SEED", "1")
	if err := os.MkdirAll(filepath.Dir(savePath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(savePath(), saveRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	g.camp = campaign.NewRunner(graph)
	if !g.loadJSONGameFromSlot(0, false) || g.st == nil || g.nativeMapAssets == nil {
		t.Fatalf("既有存檔未還原：%s %s", g.loadErr, g.msg)
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		t.Fatal(g.nativeMapHUDInputReason)
	}
	in, err := buildNativeMapFrameInput(g.nativeMapAssets, g.m, g.st, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		t.Fatal(err)
	}
	seed := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	world, err := composeModernHDMapWorld(in.Frame, seed, g.nativeMapAssets.Palette, set.mapSpritesHD)
	if err != nil {
		t.Fatal(err)
	}
	baselineInput := in.Frame
	baselineInput.Cells = append(baselineInput.Cells[:0:0], baselineInput.Cells...)
	baselineInput.SelectionOverlay = nil
	baselineVGA := make([]byte, indexedmap.NativeMapVGASize)
	if err := indexedmap.ComposeFrame(seed, baselineVGA, baselineInput, func([]byte) error { return nil }); err != nil {
		t.Fatal(err)
	}
	baseline := image.NewPaletted(image.Rect(0, 0, 320, 200), g.nativeMapAssets.Palette)
	copy(baseline.Pix, baselineVGA)
	outputs := make(map[string]string)
	for name, img := range map[string]image.Image{"hd-world-candidate.png": world, "native-remake-world.png": baseline} {
		path := filepath.Join(filepath.Dir(reportPath), name)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		encodeErr, closeErr := png.Encode(file, img), file.Close()
		if encodeErr != nil || closeErr != nil {
			t.Fatalf("candidate output: %v %v", encodeErr, closeErr)
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(bytes)
		outputs[name] = hex.EncodeToString(hash[:])
	}
	afterRaw, err := os.ReadFile(save)
	if err != nil || sha256.Sum256(afterRaw) != saveHash {
		t.Fatal("原始存檔被改寫")
	}
	report, err := json.MarshalIndent(map[string]any{
		"status": "PASS", "scope": "實際完整原生可見圖源載入與既有存檔還原的候選世界；不含HUD／正常鍵盤路徑／原版oracle／GPU呈現",
		"groups": len(set.mapSpritesHD), "frames": frameCount, "node": g.camp.NodeID(), "view": g.st.NativeMapViewState,
		"manifest_sha256": hex.EncodeToString(manifestHash[:]), "catalog_sha256": manifest.CatalogSHA256,
		"save_sha256": hex.EncodeToString(saveHash[:]), "original_save_unchanged": true, "images": outputs,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(reportPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.Write(append(report, '\n'))
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("candidate report: %v %v", writeErr, closeErr)
	}
}
