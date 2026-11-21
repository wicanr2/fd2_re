package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// TestRustGoldenR6Opening 只輸出凍結 Go 版開場取圖端的像素雜湊。
func TestRustGoldenR6Opening(t *testing.T) {
	out := os.Getenv("FD2_R6_OPENING_OUT")
	if out == "" {
		t.Skip("set FD2_R6_OPENING_OUT to export the Rust golden")
	}
	set, assets, _ := openingHDTestAssets(t)
	sort.Slice(assets, func(i, j int) bool {
		if assets[i].SourceArchive == assets[j].SourceArchive {
			return assets[i].SourceResource < assets[j].SourceResource
		}
		return assets[i].SourceArchive < assets[j].SourceArchive
	})
	var report strings.Builder
	count := 0
	for _, asset := range assets {
		key := modernOpeningKey{asset.SourceArchive, asset.SourceResource}
		bank := set.openingHD.banks[key]
		for i := range bank.encoded {
			picture, err := bank.decode(key, i)
			if err != nil {
				t.Fatal(err)
			}
			rgba := image.NewRGBA(picture.Bounds())
			draw.Draw(rgba, rgba.Bounds(), picture, picture.Bounds().Min, draw.Src)
			fmt.Fprintf(&report, "%s %d %d %x\n", key.archive, key.resource, i, sha256.Sum256(rgba.Pix))
			count++
		}
	}
	if count != 173 {
		t.Fatalf("opening golden has %d frames, want 173", count)
	}
	if err := os.WriteFile(filepath.Join(out, "go-opening.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
}

// TestRustGoldenR6SkyKey 只輸出 Go 取圖端的雜湊，輸入為同一份清冊。
func TestRustGoldenR6SkyKey(t *testing.T) {
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
		if asset.Role == "story_animation_hd_set" {
			if err := loadModernSkyKeyHDAsset(set, asset, os.Getenv("FD2_TEST_OPENING_PACK")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if set.skyKeyANI == nil || set.skyKeyFD34 == nil {
		t.Fatal("complete sky key pair required")
	}
	frames, err := fdother.LoadSeparatedCh20SkyKeyFrames(separatedAssetPath("animations/fdother_034_ch20_sky_key"))
	if err != nil {
		t.Fatal(err)
	}
	clip, err := afm.LoadSeparatedResource(separatedAssetPath("animations"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 101 || len(clip.IndexedFrames) != 96 || len(clip.Palettes) != 96 {
		t.Fatal("incomplete native banks")
	}
	var report strings.Builder
	for i, pixels := range clip.IndexedFrames {
		if err := set.skyKeyANI.validateANIFrame(i, pixels, clip.Palettes[i]); err != nil {
			t.Fatal(err)
		}
		picture, err := set.skyKeyANI.picture(i)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&report, "ANI.DAT 0 %d %x\n", i, sha256.Sum256(picture.Pix))
	}
	for i, frame := range frames {
		if err := set.skyKeyFD34.validateFDFrame(i, frame); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			continue
		}
		picture, err := set.skyKeyFD34.picture(i)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&report, "FDOTHER.DAT 34 %d %x\n", i, sha256.Sum256(picture.Pix))
	}
	if err := os.WriteFile(filepath.Join(out, "go-sky-key.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
}
