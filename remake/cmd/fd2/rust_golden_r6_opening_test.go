package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
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
