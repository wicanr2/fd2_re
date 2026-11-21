package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

// TestRustGoldenR6Terrain 輸出 Go 正式地形取用與原版來源核對的結果。
func TestRustGoldenR6Terrain(t *testing.T) {
	out := os.Getenv("FD2_R6_OPENING_OUT")
	if out == "" {
		t.Skip("set FD2_R6_OPENING_OUT to export the Rust golden")
	}
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_OPENING_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	set := &modernStoryPortraitSet{}
	for _, asset := range catalog.Assets {
		if asset.Role == "native_terrain_hd_set" {
			if err := loadModernTerrainHDAsset(set, asset, os.Getenv("FD2_TEST_OPENING_PACK")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(set.terrainHD) != 33 {
		t.Fatalf("terrain map count %d, want 33", len(set.terrainHD))
	}
	var report strings.Builder
	total := 0
	for id := 0; id < 33; id++ {
		bank, _, err := fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), id)
		if err != nil {
			t.Fatal(err)
		}
		if err := set.validateTerrainHD(id, bank); err != nil {
			t.Fatalf("map %d: %v", id, err)
		}
		frames, err := set.terrainHDMap(id)
		if err != nil || len(frames) == 0 {
			t.Fatalf("map %d has no checked frames: %v", id, err)
		}
		var indexes []int
		for index := range frames {
			indexes = append(indexes, index)
		}
		sort.Ints(indexes)
		for _, index := range indexes {
			frame := frames[index]
			fmt.Fprintf(&report, "%d/%d %s %x %x\n", id, index, frame.Role,
				sha256.Sum256(frame.Image.Pix), sha256.Sum256(frame.Colours))
			total++
		}
	}
	if total != 3983 {
		t.Fatalf("terrain frame count %d, want 3983", total)
	}
	if err := os.WriteFile(filepath.Join(out, "go-terrain.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
}
