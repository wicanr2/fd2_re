package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

func TestCoverageIncludesFormalEventWriterAndPreservesRawInput(t *testing.T) {
	data := mapData{Width: 1, Height: 1, Tiles: []uint16{0xc000}, Controls: make([]byte, 16), Modes: []byte{0xff}, Events: []byte{0}}
	data.Controls[0], data.Controls[4] = 0x28, 0x84
	raw, _ := json.Marshal(data)
	states, err := coverageStates(data)
	if err != nil || len(states) != 2 || states[1][0] != 0xc001 {
		t.Fatal(states, err)
	}
	selected, passes, checks, err := coverageRequired(data, 4)
	if err != nil || passes != 1 || checks != 16 || len(selected) != 4 {
		t.Fatal(selected, passes, checks, err)
	}
	if selected[[2]int{1, 0}].Pass != 1 || selected[[2]int{2, 1}].Pass != 1 || selected[[2]int{2, 0}].Flip != 1 {
		t.Fatal(selected)
	}
	after, _ := json.Marshal(data)
	if !bytes.Equal(raw, after) {
		t.Fatal("coverage mutated source map")
	}
	data.Controls[4], data.Controls[8] = 0x20, 0
	states, err = coverageStates(data)
	if err != nil || len(states) != 3 || states[2][0] != 0xc002 {
		t.Fatal(states, err)
	}
	data.Events = nil
	if _, err = coverageStates(data); err == nil {
		t.Fatal("missing composition admitted")
	}
	data.Events = []byte{0}
	data.Controls[0] = 0x28
	if _, _, _, err = coverageRequired(data, 2); err == nil {
		t.Fatal("selector beyond actual bank admitted")
	}
}

func coverageFixture(t *testing.T) (artManifest, []loadedMap, string, string, map[string]any) {
	t.Helper()
	root := t.TempDir()
	manifest := artManifest{}
	loaded := []loadedMap{}
	for mapID := 0; mapID < 3; mapID++ {
		s := fdicon.Sprite{Pixels: make([]byte, 576), Mask: bytes.Repeat([]byte{1}, 576), RemapMask: make([]byte, 576)}
		content := digest(append(append(append([]byte{}, s.Pixels...), s.Mask...), s.RemapMask...))
		manifest.Maps = append(manifest.Maps, mapSource{Map: mapID, FrameCount: 1, Frames: []frameSource{{Frame: 0, ContentHash: content}}, TerrainFrames: []int{0}})
		loaded = append(loaded, loadedMap{Bank: &fdicon.Bank{Sprites: []fdicon.Sprite{s}}, Data: mapData{Width: 1, Height: 1, Tiles: []uint16{0}, Controls: make([]byte, 4), Events: []byte{0}, Modes: []byte{0}}})
	}
	picture := image.NewRGBA(image.Rect(0, 0, 144, 144))
	for y := 0; y < 144; y++ {
		for x := 0; x < 144; x++ {
			picture.SetRGBA(x, y, color.RGBA{20, 40, 60, 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hd.png"), encoded.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	asset := map[string]any{"asset_id": "test", "role": "native_terrain_hd_set", "status": "runtime_candidate", "map_id": 0, "frame_count": 1, "width": 144, "height": 144, "consumer_contract": "fdshap_native_hd_sparse_v1", "alpha_contract": "native_mask_nearest_6", "cycle_policy": "native_selector", "frame_ids": []int{0}, "semantics": []string{"terrain"}, "files": []string{"hd.png"}, "frame_sha256": []string{digest(encoded.Bytes())}}
	return manifest, loaded, filepath.Join(root, "catalog.json"), root, asset
}

func writeCoverageCatalog(t *testing.T, path string, assets ...map[string]any) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"schema_version": 1, "theme_id": "modern-handpainted-a", "assets": assets})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCoverageOnlyReusesIdenticalNativeContentAndUniqueHD(t *testing.T) {
	manifest, loaded, path, root, asset := coverageFixture(t)
	writeCoverageCatalog(t, path, asset)
	a, err := buildCoverage(manifest, loaded, path, root)
	if err != nil || a.Summary["reusable_slots"] != 2 || a.Summary["registered-hd"] != 1 {
		t.Fatal(a.Summary, err)
	}
	if a.Maps[1].Frames[0].Reuse == nil || a.Maps[1].Frames[0].Reuse.Map != 0 {
		t.Fatal("missing exact source proof")
	}
	b, err := buildCoverage(manifest, loaded, path, root)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("nondeterministic coverage", err)
	}
	manifest.Maps[2].Frames[0].ContentHash = "different"
	b, err = buildCoverage(manifest, loaded, path, root)
	if err != nil || b.Summary["reusable_slots"] != 1 || b.Maps[2].Frames[0].Reuse != nil {
		t.Fatal("guessed source reuse", err)
	}
	manifest.Maps[2].Frames[0].ContentHash = manifest.Maps[0].Frames[0].ContentHash
	raw, _ := json.Marshal(asset)
	var second map[string]any
	json.Unmarshal(raw, &second)
	second["asset_id"], second["map_id"] = "other", 1
	picture := image.NewRGBA(image.Rect(0, 0, 144, 144))
	for y := 0; y < 144; y++ {
		for x := 0; x < 144; x++ {
			picture.SetRGBA(x, y, color.RGBA{90, 80, 70, 255})
		}
	}
	var out bytes.Buffer
	if err = png.Encode(&out, picture); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "other.png"), out.Bytes(), 0644)
	second["files"], second["frame_sha256"] = []string{"other.png"}, []string{digest(out.Bytes())}
	writeCoverageCatalog(t, path, asset, second)
	b, err = buildCoverage(manifest, loaded, path, root)
	if err != nil || b.Summary["reusable_slots"] != 0 || b.Maps[2].Frames[0].Reuse != nil || len(b.Maps[2].Frames[0].Conflicts) != 2 {
		t.Fatal("ambiguous HD silently selected", b.Summary, err)
	}
}

func TestCoverageRejectsWrongCatalogSHAAlphaAndDuplicates(t *testing.T) {
	for _, fault := range []string{"sha", "file", "duplicate", "role", "native-mask", "geometry"} {
		t.Run(fault, func(t *testing.T) {
			manifest, loaded, path, root, asset := coverageFixture(t)
			switch fault {
			case "sha":
				asset["frame_sha256"] = []string{"wrong"}
			case "file":
				asset["files"] = []string{"../hd.png"}
			case "role":
				asset["semantics"] = []string{"guessed"}
			case "native-mask":
				loaded[0].Bank.Sprites[0].Mask[0] = 0
			case "geometry":
				asset["width"] = 24
			}
			writeCoverageCatalog(t, path, asset)
			if fault == "duplicate" {
				writeCoverageCatalog(t, path, asset, asset)
			}
			if _, err := buildCoverage(manifest, loaded, path, root); err == nil {
				t.Fatal("accepted", fault)
			}
		})
	}
}

func TestCoverageSeparatesCompositeRequirementsFromMissingArt(t *testing.T) {
	manifest, loaded, path, root, _ := coverageFixture(t)
	writeCoverageCatalog(t, path)
	loaded[0].Bank.Sprites[0].Mask[0] = 0
	loaded[1].Bank.Sprites[0].Mask = make([]byte, 576)
	loaded[1].Bank.Sprites[0].RemapMask[0] = 1
	loaded[2].Bank.Sprites[0].Mask = make([]byte, 576)
	for i := range loaded {
		s := loaded[i].Bank.Sprites[0]
		manifest.Maps[i].Frames[0].ContentHash = digest(append(append(append([]byte{}, s.Pixels...), s.Mask...), s.RemapMask...))
	}
	r, err := buildCoverage(manifest, loaded, path, root)
	if err != nil {
		t.Fatal(err)
	}
	for key, count := range map[string]int{"missing-composite": 0, "native-remap-only": 1, "no-source-write": 1, "missing-art": 1} {
		if r.Summary[key] != count {
			t.Fatal(key, r.Summary)
		}
	}
	if r.Maps[0].Frames[0].Backdrop != "already-composed-native-or-registered-hd-stage" {
		t.Fatal("partial source lacks explicit backdrop policy")
	}
}
