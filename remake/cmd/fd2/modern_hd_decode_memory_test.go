package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func decodedSharingPNG(t *testing.T, value color.RGBA) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, 12, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 12; x++ {
			picture.SetRGBA(x, y, value)
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func decodedSharingFixture(t *testing.T) *modernSkyKeyHDBank {
	t.Helper()
	encoded := decodedSharingPNG(t, color.RGBA{R: 34, G: 86, B: 140, A: 255})
	root := t.TempDir()
	sum := sha256.Sum256(encoded)
	hash := hex.EncodeToString(sum[:])
	b := &modernSkyKeyHDBank{root: root, asset: modernThemeAsset{
		Files: []string{"one.png", "two.png", "three.png"}, FrameSHA256: []string{hash, hash, hash},
		NativeFrames: []modernBattleHDDescriptor{{Width: 2, Height: 2}, {Width: 2, Height: 2}, {Width: 2, Height: 2}},
	}, plan: modernSkyKeyPlan{Frames: make([]modernSkyKeyRow, 3)}}
	for _, name := range b.asset.Files {
		if err := os.WriteFile(filepath.Join(root, name), encoded, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return b
}

func distinctSharingSource(t *testing.T, b *modernSkyKeyHDBank) {
	t.Helper()
	raw := decodedSharingPNG(t, color.RGBA{R: 120, G: 30, B: 80, A: 255})
	if err := os.WriteFile(filepath.Join(b.root, b.asset.Files[2]), raw, 0644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	b.asset.FrameSHA256[2] = hex.EncodeToString(sum[:])
}

func TestModernHDSkyKeyDecodedSharing(t *testing.T) {
	t.Run("identical-pixels-and-storage", func(t *testing.T) {
		b := decodedSharingFixture(t)
		pool := map[string]*image.RGBA{}
		a, err := b.pictureWithSharedStorage(0, pool)
		if err != nil {
			t.Fatal(err)
		}
		z, err := b.pictureWithSharedStorage(1, pool)
		if err != nil || a != z || len(pool) != 1 || a.RGBAAt(0, 0) != (color.RGBA{R: 34, G: 86, B: 140, A: 255}) {
			t.Fatal("verified duplicate was not shared intact", err)
		}
		distinctSharingSource(t, b)
		other, err := b.pictureWithSharedStorage(2, pool)
		if err != nil || other == a || other.RGBAAt(0, 0) != (color.RGBA{R: 120, G: 30, B: 80, A: 255}) {
			t.Fatal("distinct verified source was coalesced", err)
		}
		b.pictures = []*image.RGBA{a, z, other}
		b.pictureSHA = []string{skyKeySHA(a.Pix), skyKeySHA(z.Pix), skyKeySHA(other.Pix)}
		a.Pix[0] ^= 1 // Deliberate violation of the immutable-picture contract.
		if _, err := b.picture(1); err == nil {
			t.Fatal("shared decoded pixel corruption accepted")
		}
		if _, err := b.picture(2); err != nil {
			t.Fatal("unmodified independent source was rejected", err)
		}
	})
	for _, fault := range []string{"missing-path", "changed-bytes", "wrong-shape", "wrong-black-frame", "wrong-coverage"} {
		t.Run(fault, func(t *testing.T) {
			b := decodedSharingFixture(t)
			pool := map[string]*image.RGBA{}
			if _, err := b.pictureWithSharedStorage(0, pool); err != nil {
				t.Fatal(err)
			}
			index := 1
			switch fault {
			case "missing-path":
				if err := os.Remove(filepath.Join(b.root, b.asset.Files[1])); err != nil {
					t.Fatal(err)
				}
			case "changed-bytes":
				if err := os.WriteFile(filepath.Join(b.root, b.asset.Files[1]), []byte("changed"), 0644); err != nil {
					t.Fatal(err)
				}
			case "wrong-shape":
				b.asset.NativeFrames[1].Width = 3
			case "wrong-black-frame":
				b.plan.Frames[1].Blank = true
			case "wrong-coverage":
				b.offset, index = 1, 2 // FDOTHER coverage needs transparent and opaque endpoints.
			}
			if _, err := b.pictureWithSharedStorage(index, pool); err == nil {
				t.Fatal("duplicate hash bypassed per-frame validation")
			}
		})
	}
}

func TestModernHDBattleDecodedSharing(t *testing.T) {
	for _, fault := range []string{"none", "distinct-source", "missing-path", "changed-bytes", "wrong-shape"} {
		t.Run(fault, func(t *testing.T) {
			b := decodedSharingFixture(t)
			bounds := image.Rect(0, 0, 12, 12)
			old := image.NewRGBA(bounds)
			entries := map[int]*modernBattleHDResource{
				1: {paths: []string{filepath.Join(b.root, b.asset.Files[0]), filepath.Join(b.root, b.asset.Files[1])},
					hashes: b.asset.FrameSHA256[:2], sizes: []image.Rectangle{bounds, bounds}, Frames: make([]image.Image, 2)},
				2: {paths: []string{filepath.Join(b.root, b.asset.Files[2])}, hashes: b.asset.FrameSHA256[2:],
					sizes: []image.Rectangle{bounds}, Frames: make([]image.Image, 1)},
				9: {paths: []string{"old.png"}, Frames: []image.Image{old}},
			}
			switch fault {
			case "distinct-source":
				distinctSharingSource(t, b)
			case "missing-path":
				if err := os.Remove(entries[2].paths[0]); err != nil {
					t.Fatal(err)
				}
			case "changed-bytes":
				if err := os.WriteFile(entries[2].paths[0], []byte("changed"), 0644); err != nil {
					t.Fatal(err)
				}
			case "wrong-shape":
				entries[2].sizes[0] = image.Rect(0, 0, 18, 12)
			}
			pending, err := stageModernBattleHDResources(entries, 1, 2)
			if entries[1].Frames[0] != nil || entries[2].Frames[0] != nil || entries[9].Frames[0] != old {
				t.Fatal("staging published or evicted before complete validation")
			}
			if fault != "none" && fault != "distinct-source" {
				if err == nil {
					t.Fatal("cached hash bypassed independent path or geometry check")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			a, z := pending.frames[1][0], pending.frames[2][0]
			if a != pending.frames[1][1] || (fault == "none" && a != z) || (fault == "distinct-source" && a == z) {
				t.Fatal("newly decoded identical sources have separate storage")
			}
			if fault == "distinct-source" && color.RGBAModel.Convert(z.At(0, 0)) != (color.RGBA{R: 120, G: 30, B: 80, A: 255}) {
				t.Fatal("distinct source pixels changed")
			}
			pending.publish()
			if entries[1].Frames[0] != a || entries[2].Frames[0] != z || entries[9].Frames[0] != nil {
				t.Fatal("publication or old-bank eviction changed")
			}
			empty, err := stageModernBattleHDResources(entries)
			if err != nil {
				t.Fatal(err)
			}
			empty.publish()
			next, err := stageModernBattleHDResources(entries, 1, 2)
			if err != nil || next.frames[1][0] == a {
				t.Fatal("sharing pool retained frames across eviction/transactions", err)
			}
		})
	}
}

// Compare the same real PNG sources before/after sharing. This is CPU storage
// evidence, not RSS, GPU memory, normal reachability or platform acceptance.
func TestModernHDDecodedImageMemory(t *testing.T) {
	if os.Getenv("FD2_HD_DECODE_MEMORY") != "1" {
		t.Skip("dedicated real-source decoder inventory required")
	}
	catalogPath, pack := os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK")
	out := os.Getenv("FD2_HD_DECODE_MEMORY_OUT")
	if catalogPath == "" || pack == "" || out == "" {
		t.Fatal("explicit catalog, pack and new report required")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("report exists or cannot be checked")
	}
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	set := &modernStoryPortraitSet{}
	resources := map[int]bool{0: true, 1: true, 288: true, 289: true}
	for _, asset := range catalog.Assets {
		switch {
		case asset.Role == "story_animation_hd_set":
			err = loadModernSkyKeyHDAsset(set, asset, pack)
		case asset.Role == "battle_animation_hd_set" && resources[asset.SourceResource]:
			err = loadModernBattleHDAsset(set, asset, pack)
		default:
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if set.skyKeyANI == nil || set.skyKeyFD34 == nil || len(set.battleAnimations) != 4 {
		t.Fatal("actual source banks missing")
	}
	if err := set.prepareBattleHDResources(0, 1, 288, 289); err != nil {
		t.Fatal(err)
	}
	groups := map[string][]image.Image{}
	hashes := map[string][]string{}
	for name, bank := range map[string]*modernSkyKeyHDBank{"sky-fdother34": set.skyKeyFD34, "sky-ani0": set.skyKeyANI} {
		if err := bank.recheckFiles(); err != nil {
			t.Fatal(err)
		}
		for _, picture := range bank.pictures {
			groups[name] = append(groups[name], picture)
		}
		hashes[name] = bank.asset.FrameSHA256
	}
	for _, resource := range []int{0, 1, 288, 289} {
		entry := set.battleAnimations[resource]
		groups["physical"] = append(groups["physical"], entry.Frames...)
		hashes["physical"] = append(hashes["physical"], entry.hashes...)
	}
	rows := map[string]any{}
	for name, frames := range groups {
		pointers := map[image.Image]bool{}
		identities := map[string]image.Image{}
		logicalBytes, retainedBytes, uniqueSourceBytes := int64(0), int64(0), int64(0)
		sharedRequired := os.Getenv("FD2_HD_DECODE_SHARE_REQUIRED") == "1"
		for index, picture := range frames {
			if picture == nil {
				t.Fatal("unpublished frame", name, index)
			}
			size := int64(picture.Bounds().Dx()) * int64(picture.Bounds().Dy()) * 4
			logicalBytes += size
			if !pointers[picture] {
				retainedBytes += size
				pointers[picture] = true
			}
			hash := hashes[name][index]
			if prior := identities[hash]; prior == nil {
				identities[hash] = picture
				uniqueSourceBytes += size
			} else if sharedRequired && prior != picture {
				t.Fatal("identical verified source has separate decoded storage", name, index)
			}
		}
		if sharedRequired && (len(pointers) != len(identities) || retainedBytes != uniqueSourceBytes) {
			t.Fatal("sharing inventory differs from source identity", name)
		}
		rows[name] = map[string]any{"frames": len(frames), "distinct_decoded_images": len(pointers),
			"unique_png_sha256": len(identities), "logical_rgba_bytes": logicalBytes,
			"retained_rgba_equivalent_bytes": retainedBytes, "unique_source_rgba_bytes": uniqueSourceBytes,
			"saved_rgba_equivalent_bytes": logicalBytes - retainedBytes}
	}
	runtime.GC()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	sha := sha256.Sum256(raw)
	result := map[string]any{"status": "CPU-DECODE-STORAGE-OBSERVATION", "sharing_required": os.Getenv("FD2_HD_DECODE_SHARE_REQUIRED") == "1",
		"catalog_sha256": hex.EncodeToString(sha[:]), "banks": rows, "go_version": runtime.Version(),
		"post_gc_heap_alloc_bytes": memory.HeapAlloc,
		"scope":                    "同一正式PNG與loader；4-byte RGBA等價容量及實際image指標去重。不是全遊戲RSS、GPU、正常玩家或平台驗收。"}
	runtime.KeepAlive(set)
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("decoded source storage: %s", encoded)
}
