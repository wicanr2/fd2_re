package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestModernCompoundTailArtAtomic(t *testing.T) {
	if os.Getenv("FD2_HD_COMPOUND_TAIL_ART_UNIT") != "1" {
		t.Skip("explicit healing catalog/pack")
	}
	checkModernTailArtGroups(t, []int{12, 65}, 77)
}

func checkModernTailArtGroups(t *testing.T, counts []int, sources int) {
	t.Helper()
	pack := os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	groups := 0
	for _, asset := range catalog.Assets {
		selected := false
		for _, count := range counts {
			selected = selected || asset.FrameCount == count
		}
		if asset.Role != "map_overlay_hd_set" || asset.SourceResource != 6 || !selected {
			continue
		}
		groups++
		if err := loadModernMapOverlayHDAsset(&modernStoryPortraitSet{}, asset, pack); err != nil {
			t.Fatal(err)
		}
		for _, fault := range []string{"metadata", "source", "contract", "count", "descriptor", "missing", "sha", "path", "png-size"} {
			bad := asset
			bad.Files = append([]string(nil), asset.Files...)
			bad.FrameSHA256 = append([]string(nil), asset.FrameSHA256...)
			bad.NativeLMIFrames = append([]modernLMIHDDescriptor(nil), asset.NativeLMIFrames...)
			root := pack
			switch fault {
			case "metadata":
				bad.SourceMetadataSHA256 = "bad"
			case "source":
				bad.SourceResource = 4
			case "contract":
				bad.ConsumerContract = "unknown"
			case "count":
				bad.FrameCount++
			case "descriptor":
				bad.NativeLMIFrames[0].PixelsSHA256 = "bad"
			case "missing":
				bad.Files[len(bad.Files)-1] = "absent-heal.png"
			case "sha":
				bad.FrameSHA256[0] = "bad"
			case "path":
				bad.Files[0] = "../escape.png"
			case "png-size":
				root = t.TempDir()
				for _, n := range asset.Files {
					if err := os.Symlink(filepath.Join(pack, n), filepath.Join(root, n)); err != nil {
						t.Fatal(err)
					}
				}
				n := asset.Files[len(asset.Files)-1]
				if err := os.Remove(filepath.Join(root, n)); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(filepath.Join(root, n))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 6, 6))); err != nil {
					t.Fatal(err)
				}
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
				b, err := os.ReadFile(filepath.Join(root, n))
				if err != nil {
					t.Fatal(err)
				}
				bad.FrameSHA256[len(bad.FrameSHA256)-1] = fmt.Sprintf("%x", sha256.Sum256(b))
			}
			marker := &modernMapOverlayHDResource{frames: map[int]*image.RGBA{999: image.NewRGBA(image.Rect(0, 0, 1, 1))}}
			set := &modernStoryPortraitSet{mapEffects: marker, mapOverlays: marker}
			if err := loadModernMapOverlayHDAsset(set, bad, root); err == nil || set.mapEffects != marker || set.mapOverlays != marker {
				t.Fatal("invalid compound tail group published", asset.SourceResource, fault)
			}
		}
	}
	if groups != len(counts) {
		t.Fatal("missing declared LMI source groups", groups, counts)
	}
	t.Logf("all%d real LMI PNGs plus%d invalid cold-group cases passed atomically", sources, groups*9)
}
