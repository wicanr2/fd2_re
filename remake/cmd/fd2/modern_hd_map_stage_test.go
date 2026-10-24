package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestModernHDCh23StageLoader(t *testing.T) {
	for _, kind := range []string{"valid", "archive", "resource", "contract", "descriptor", "path", "metadata", "missing", "sha", "geometry", "alpha", "source-pixels", "source-mask", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			asset := stage42TestAsset(t)
			root := os.Getenv("FD2_TEST_UNIT_PRESENT_PACK")
			set := &modernStoryPortraitSet{}
			switch kind {
			case "archive":
				asset.SourceArchive = "BG.DAT"
			case "resource":
				asset.SourceResource = 41
			case "contract":
				asset.ConsumerContract = "unreviewed"
			case "descriptor":
				asset.NativeFrames[0].X = 1
			case "path":
				asset.File = "../escape.png"
			case "metadata":
				asset.SourceMetadataSHA256 = "invalid"
			case "missing":
				root = t.TempDir()
			case "sha":
				asset.SHA256 = "invalid"
			case "geometry", "alpha":
				raw, err := os.ReadFile(filepath.Join(root, asset.File))
				if err != nil {
					t.Fatal(err)
				}
				_ = raw
				var picture *image.NRGBA
				if kind == "geometry" {
					picture = image.NewNRGBA(image.Rect(0, 0, 1, 1))
				} else {
					picture = image.NewNRGBA(image.Rect(0, 0, 1872, 1152))
					for i := 3; i < len(picture.Pix); i += 4 {
						picture.Pix[i] = 255
					}
					picture.Pix[3] = 128
				}
				root = t.TempDir()
				f, err := os.Create(filepath.Join(root, asset.File))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(f, picture); err != nil {
					t.Fatal(err)
				}
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
				raw, err = os.ReadFile(filepath.Join(root, asset.File))
				if err != nil {
					t.Fatal(err)
				}
				asset.SHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
			case "source-pixels", "source-mask":
				original := os.Getenv("FD2_ASSET_PACK")
				temp := t.TempDir()
				folder := filepath.Join(temp, "surfaces", "FDOTHER_042")
				if err := os.MkdirAll(folder, 0755); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"resource.json", "frame.png", "mask.png"} {
					raw, err := os.ReadFile(filepath.Join(original, "surfaces", "FDOTHER_042", name))
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(folder, name), raw, 0644); err != nil {
						t.Fatal(err)
					}
				}
				name := "frame.png"
				if kind == "source-mask" {
					name = "mask.png"
				}
				f, err := os.Open(filepath.Join(folder, name))
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := png.Decode(f)
				f.Close()
				if err != nil {
					t.Fatal(err)
				}
				if kind == "source-pixels" {
					p, ok := decoded.(*image.Paletted)
					if !ok {
						t.Fatal("indexed source required")
					}
					p.Pix[0] ^= 1
				} else {
					p, ok := decoded.(*image.Gray)
					if !ok {
						t.Fatal("gray mask required")
					}
					p.Pix[0] = 0
				}
				out, err := os.Create(filepath.Join(folder, name))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(out, decoded); err != nil {
					t.Fatal(err)
				}
				out.Close()
				t.Setenv("FD2_ASSET_PACK", temp)
			case "duplicate":
				if err := loadModernMapStageHD(set, asset, root); err != nil {
					t.Fatal(err)
				}
			}
			before := set.mapStages[42]
			err := loadModernMapStageHD(set, asset, root)
			if kind == "valid" {
				if err != nil || set.mapStages[42] == nil || !set.hasHD {
					t.Fatal("valid source rejected", err)
				}
			} else if err == nil || set.mapStages[42] != before {
				t.Fatal("invalid source published", kind, err)
			}
		})
	}
}
