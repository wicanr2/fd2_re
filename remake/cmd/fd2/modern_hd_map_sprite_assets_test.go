package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

func writeModernHDMapFixture(t *testing.T) (string, string, map[string]any) {
	t.Helper()
	path, root := writeModernPortraitFixture(t, true)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	files, hashes, hdFiles, hdHashes := make([]string, 12), make([]string, 12), make([]string, 12), make([]string, 12)
	for frame := 0; frame < 12; frame++ {
		for _, hd := range []bool{false, true} {
			size, label := 24, "parent"
			if hd {
				size, label = 144, "hd"
			}
			img := image.NewNRGBA(image.Rect(0, 0, size, size))
			rect := image.Rect(8, 1, 16, 23)
			if hd {
				rect = image.Rect(42, 18, 102, 138)
			}
			draw.Draw(img, rect, &image.Uniform{C: color.NRGBA{R: uint8(40 + frame), G: 80, B: 120, A: 255}}, image.Point{}, draw.Src)
			if hd {
				img.SetNRGBA(41, 137, color.NRGBA{R: 100, G: 120, B: 140, A: 128})
			}
			name := fmt.Sprintf("%s-%02d.png", label, frame)
			file, err := os.Create(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, img); err != nil {
				file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			bytes, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(bytes)
			if hd {
				hdFiles[frame], hdHashes[frame] = name, hex.EncodeToString(digest[:])
			} else {
				files[frame], hashes[frame] = name, hex.EncodeToString(digest[:])
			}
		}
	}
	assets := document["assets"].([]any)
	for group := 0; group < fdicon.NativeSourceGroupCount; group++ {
		if !fdicon.NativeSourceGroupHasPixels(group) {
			continue
		}
		count, cycle, contract, hdContract := 12, "three_distinct_cycles", "fdicon_map_sprite_12x24_v1", "fdicon_hd_map_sprite_12x144_v1"
		switch group {
		case 48, 66, 67, 74, 75:
			count, cycle, contract, hdContract = 3, "static_three_phase", "fdicon_static_sprite_3x24_v1", "fdicon_hd_static_sprite_3x144_v1"
		}
		parentFiles, parentHashes := append([]string(nil), files[:count]...), append([]string(nil), hashes[:count]...)
		childFiles, childHashes := append([]string(nil), hdFiles[:count]...), append([]string(nil), hdHashes[:count]...)
		if group == 52 {
			cycle = "source_exact_repeats"
			parentFiles[8], parentHashes[8], childFiles[8], childHashes[8] = parentFiles[6], parentHashes[6], childFiles[6], childHashes[6]
		}
		assets = append(assets, map[string]any{"role": "map_sprite_set", "status": "runtime_candidate", "consumer_contract": contract, "source_group": group, "frame_count": count, "width": 24, "height": 24, "alpha_contract": "binary", "cycle_policy": cycle, "files": parentFiles, "frame_sha256": parentHashes})
		assets = append(assets, map[string]any{"role": "map_sprite_hd_set", "status": "runtime_candidate", "consumer_contract": hdContract, "source_group": group, "frame_count": count, "width": 144, "height": 144, "alpha_contract": "coverage", "layout_policy": "contain_22px_cell_feet_v1", "cycle_policy": cycle, "files": childFiles, "frame_sha256": childHashes})
	}
	document["assets"] = assets
	writeModernHDCatalog(t, path, document)
	return path, root, document
}

func writeModernHDCatalog(t *testing.T, path string, document map[string]any) {
	t.Helper()
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadModernHDMapSpritesCompleteCoverageAndSoftEdges(t *testing.T) {
	path, root, _ := writeModernHDMapFixture(t)
	set, err := loadModernStoryPortraitSet(path, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.mapSpritesHD) != 125 || len(set.mapSpritesHD[48]) != 3 || len(set.mapSpritesHD[0]) != 12 ||
		len(set.mapSpritesHD[96]) != 12 || len(set.mapSpritesHD[135]) != 12 {
		t.Fatal("incomplete HD collection")
	}
	_, _, _, alpha := set.mapSpritesHD[0][0].At(41, 137).RGBA()
	if alpha != 0x8080 {
		t.Fatalf("soft edge changed: %x", alpha)
	}
	if set.mapSprites[0][0].Bounds().Dx() != 24 || set.hasHD {
		t.Fatal("HD map admission changed existing projection or portrait presentation")
	}
}

func TestLoadModernHDMapSpritesRejectsIncompleteOrMismatchedCollection(t *testing.T) {
	for _, name := range []string{"missing", "duplicate", "parent", "cycle", "hash", "path", "feet", "zero-source", "outside", "legacy94"} {
		t.Run(name, func(t *testing.T) {
			path, root, document := writeModernHDMapFixture(t)
			assets := document["assets"].([]any)
			first := assets[2].(map[string]any)
			switch name {
			case "missing":
				document["assets"] = assets[:len(assets)-1]
			case "duplicate":
				assets[len(assets)-1].(map[string]any)["source_group"] = 0
			case "zero-source":
				assets[len(assets)-1].(map[string]any)["source_group"] = 104
			case "outside":
				assets[len(assets)-1].(map[string]any)["source_group"] = fdicon.NativeSourceGroupCount
			case "legacy94":
				filtered := make([]any, 0, len(assets))
				for _, entry := range assets {
					a := entry.(map[string]any)
					if group, ok := a["source_group"].(int); ok && group >= 96 {
						continue
					}
					filtered = append(filtered, entry)
				}
				document["assets"] = filtered
			case "parent":
				assets[1].(map[string]any)["source_group"] = 39
			case "cycle":
				for _, entry := range assets {
					a := entry.(map[string]any)
					if a["role"] == "map_sprite_hd_set" && a["source_group"] == 52 {
						a["files"].([]string)[8] = "hd-08.png"
						a["frame_sha256"].([]string)[8] = first["frame_sha256"].([]string)[8]
					}
				}
			case "hash":
				first["frame_sha256"].([]string)[0] = "bad"
			case "path":
				first["files"].([]string)[0] = "../hd-00.png"
			case "feet":
				first["files"].([]string)[0] = "bad-feet.png"
				img := image.NewNRGBA(image.Rect(0, 0, 144, 144))
				img.SetNRGBA(72, 100, color.NRGBA{A: 255})
				f, err := os.Create(filepath.Join(root, "bad-feet.png"))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(f, img); err != nil {
					t.Fatal(err)
				}
				f.Close()
				raw, err := os.ReadFile(filepath.Join(root, "bad-feet.png"))
				if err != nil {
					t.Fatal(err)
				}
				digest := sha256.Sum256(raw)
				first["frame_sha256"].([]string)[0] = hex.EncodeToString(digest[:])
			}
			writeModernHDCatalog(t, path, document)
			set, err := loadModernStoryPortraitSet(path, root)
			if err == nil || set != nil {
				t.Fatalf("%s accepted or published partial collection", name)
			}
		})
	}
}

func TestLoadModernHDMapSpritesKeepsLegacyCatalogCompatibility(t *testing.T) {
	for _, limit := range []int{96, fdicon.NativeSourceGroupCount} {
		path, root, document := writeModernHDMapFixture(t)
		filtered := make([]any, 0)
		for _, entry := range document["assets"].([]any) {
			asset := entry.(map[string]any)
			if asset["role"] == "map_sprite_hd_set" {
				continue
			}
			if group, ok := asset["source_group"].(int); ok && group >= limit {
				continue
			}
			filtered = append(filtered, entry)
		}
		document["assets"] = filtered
		writeModernHDCatalog(t, path, document)
		set, err := loadModernStoryPortraitSet(path, root)
		if err != nil || len(set.mapSpritesHD) != 0 {
			t.Fatalf("legacy catalog limit=%d: %v", limit, err)
		}
		if limit == fdicon.NativeSourceGroupCount && len(set.mapSprites[135]) != 12 {
			t.Fatal("last visible native group did not load")
		}
	}
}

func TestLoadModernHDMapSpritesRejectsInvalidAlpha(t *testing.T) {
	for _, alpha := range []uint8{0, 128, 255} {
		t.Run(fmt.Sprint(alpha), func(t *testing.T) {
			path, root, document := writeModernHDMapFixture(t)
			first := document["assets"].([]any)[2].(map[string]any)
			img := image.NewNRGBA(image.Rect(0, 0, 144, 144))
			draw.Draw(img, img.Bounds(), &image.Uniform{C: color.NRGBA{R: 40, G: 80, B: 120, A: alpha}}, image.Point{}, draw.Src)
			f, err := os.Create(filepath.Join(root, "hd-00.png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, img); err != nil {
				t.Fatal(err)
			}
			f.Close()
			raw, err := os.ReadFile(filepath.Join(root, "hd-00.png"))
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(raw)
			first["frame_sha256"].([]string)[0] = hex.EncodeToString(digest[:])
			writeModernHDCatalog(t, path, document)
			if set, err := loadModernStoryPortraitSet(path, root); err == nil || set != nil {
				t.Fatal("invalid alpha accepted")
			}
		})
	}
}
