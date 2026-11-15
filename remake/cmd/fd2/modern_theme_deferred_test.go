package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rewriteBattleHDFixtureFrame 換掉 fixture 的一格並更新清冊 SHA-256，讓雜湊仍相符。
func rewriteBattleHDFixtureFrame(t *testing.T, a *modernThemeAsset, root string, index int, raw []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, a.Files[index]), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	a.FrameSHA256[index] = hex.EncodeToString(digest[:])
}

func encodeBattleHDFixturePNG(t *testing.T, picture image.Image) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, picture); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// 載入只核對檔案、SHA-256 與 PNG 標頭；像素層的缺陷（透明度、壓縮資料損壞）到第一次預備才
// 發現。發現時那筆資源整組退回原版畫格並留下訊息，其他資源照常發布，交易不失敗（#317）。
func TestModernHDBattleDeferredChecksFallBackAtPrepare(t *testing.T) {
	for _, fault := range []string{"opaque-background", "no-opaque-core", "corrupt-pixels"} {
		t.Run(fault, func(t *testing.T) {
			bad, badNative, badRoot := battleHDAssetFixture(t)
			good, goodNative, goodRoot := battleHDAssetFixture(t)
			good.SourceResource = 90
			switch fault {
			case "opaque-background":
				picture := image.NewNRGBA(image.Rect(0, 0, 12, 12))
				for i := 3; i < len(picture.Pix); i += 4 {
					picture.Pix[i] = 255
				}
				rewriteBattleHDFixtureFrame(t, &bad, badRoot, 1, encodeBattleHDFixturePNG(t, picture))
			case "no-opaque-core":
				picture := image.NewNRGBA(image.Rect(0, 0, 12, 12))
				picture.SetNRGBA(2, 2, color.NRGBA{R: 200, A: 128})
				rewriteBattleHDFixtureFrame(t, &bad, badRoot, 1, encodeBattleHDFixturePNG(t, picture))
			case "corrupt-pixels":
				raw, err := os.ReadFile(filepath.Join(badRoot, bad.Files[1]))
				if err != nil {
					t.Fatal(err)
				}
				at := bytes.Index(raw, []byte("IDAT"))
				if at < 0 {
					t.Fatal("fixture has no IDAT")
				}
				raw = append([]byte(nil), raw...)
				raw[at+6] ^= 0xff // 壓縮資料與 CRC 不再相符；標頭仍完整
				rewriteBattleHDFixtureFrame(t, &bad, badRoot, 1, raw)
			}
			set := &modernStoryPortraitSet{}
			if err := loadModernBattleHDAssetWithNative(set, bad, badRoot, badNative); err != nil {
				t.Fatalf("pixel-level fault rejected at load (load must not decode): %v", err)
			}
			if err := loadModernBattleHDAssetWithNative(set, good, goodRoot, goodNative); err != nil {
				t.Fatal(err)
			}
			if set.battleAnimations[91].Frames[0] != nil || set.battleAnimations[90].Frames[0] != nil {
				t.Fatal("load decoded battle frames")
			}
			if err := set.prepareBattleHDBanks([]int{91, 90}, nil); err != nil {
				t.Fatalf("deferred fault failed the whole transaction: %v", err)
			}
			if set.battleAnimations[91] != nil {
				t.Fatal("rejected resource still registered")
			}
			if picture, err := set.battleHDFrame(91, 0); picture != nil || err != nil {
				t.Fatal("rejected resource did not fall back to native frames", picture, err)
			}
			for index := 0; index < 2; index++ {
				if picture, err := set.battleHDFrame(90, index); picture == nil || err != nil {
					t.Fatal("healthy resource was not published", err)
				}
			}
			g := &Game{modernStoryPortraits: set}
			g.surfaceModernThemeNotices()
			if !strings.HasPrefix(g.msg, modernThemeDeferredMessage) || !strings.Contains(g.msg, "resource 91 frame 1") {
				t.Fatalf("player was not told about the fallback: %q", g.msg)
			}
			g.msg = ""
			if err := set.prepareBattleHDBanks([]int{91, 90}, nil); err != nil {
				t.Fatal(err)
			}
			g.surfaceModernThemeNotices()
			if g.msg != "" {
				t.Fatalf("fallback reported twice: %q", g.msg)
			}
		})
	}
}

// 缺件、雜湊不符與 PNG 標頭壞掉仍在載入時整筆拒收，不延後。
func TestModernHDBattleLoadStillRejectsFileFaults(t *testing.T) {
	for _, fault := range []string{"missing", "sha", "truncated-header", "wrong-size"} {
		t.Run(fault, func(t *testing.T) {
			a, native, root := battleHDAssetFixture(t)
			switch fault {
			case "missing":
				if err := os.Remove(filepath.Join(root, a.Files[1])); err != nil {
					t.Fatal(err)
				}
			case "sha":
				a.FrameSHA256[1] = strings.Repeat("0", 64)
			case "truncated-header":
				rewriteBattleHDFixtureFrame(t, &a, root, 1, []byte("\x89PNG\r\n\x1a\n"))
			case "wrong-size":
				rewriteBattleHDFixtureFrame(t, &a, root, 1, encodeBattleHDFixturePNG(t, image.NewNRGBA(image.Rect(0, 0, 13, 12))))
			}
			set := &modernStoryPortraitSet{}
			if err := loadModernBattleHDAssetWithNative(set, a, root, native); err == nil || len(set.battleAnimations) != 0 {
				t.Fatalf("file-level fault accepted: %v", err)
			}
		})
	}
}

// 快速透明度摘要必須和逐點 At().RGBA() 的判斷一致（含非零原點與 stride 大於寬度的子圖）。
func TestModernHDAlphaCoverageMatchesPointwise(t *testing.T) {
	pointwise := func(picture image.Image) modernHDAlphaCoverage {
		var c modernHDAlphaCoverage
		for y := picture.Bounds().Min.Y; y < picture.Bounds().Max.Y; y++ {
			for x := picture.Bounds().Min.X; x < picture.Bounds().Max.X; x++ {
				_, _, _, a := picture.At(x, y).RGBA()
				c.transparent = c.transparent || a == 0
				c.opaque = c.opaque || a == 0xffff
				c.visible = c.visible || a != 0
				if c.transparent && c.opaque {
					return c
				}
			}
		}
		return c
	}
	big := image.NewNRGBA(image.Rect(0, 0, 9, 7))
	for i := 3; i < len(big.Pix); i += 4 {
		big.Pix[i] = 255
	}
	big.SetNRGBA(8, 0, color.NRGBA{A: 0}) // 只在子圖外
	sub := big.SubImage(image.Rect(2, 1, 7, 6))
	rgba := image.NewRGBA(image.Rect(-3, -2, 4, 5))
	rgba.SetRGBA(3, 4, color.RGBA{R: 9, A: 9})
	gray := image.NewGray(image.Rect(0, 0, 4, 4))
	paletted := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{color.NRGBA{A: 0}, color.NRGBA{R: 1, A: 255}})
	paletted.SetColorIndex(1, 1, 1)
	soft := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	soft.SetNRGBA(1, 1, color.NRGBA{R: 5, A: 7})
	for name, picture := range map[string]image.Image{"sub-opaque": sub, "rgba-offset": rgba, "gray": gray, "paletted": paletted,
		"soft": soft, "full": big, "empty": image.NewNRGBA(image.Rect(0, 0, 5, 5))} {
		if got, want := modernHDAlphaCoverageOf(picture), pointwise(picture); got != want {
			t.Errorf("%s: fast %+v, pointwise %+v", name, got, want)
		}
	}
}
