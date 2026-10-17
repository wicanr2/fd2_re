package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func changeModernHDCatalog(t *testing.T, path string, change func(map[string]any)) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	change(document)
	raw, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func addModernHDFixture(t *testing.T, path, root string, speaker, size int, alpha uint8) {
	t.Helper()
	picture := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			picture.SetNRGBA(x, y, color.NRGBA{R: 200, G: 40, B: 80, A: alpha})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("hd-%d.png", speaker)
	if err := os.WriteFile(filepath.Join(root, name), buffer.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(buffer.Bytes())
	changeModernHDCatalog(t, path, func(document map[string]any) {
		document["assets"] = append(document["assets"].([]any), map[string]any{
			"role": "story_portrait_hd_frame", "status": "runtime_candidate", "file": name,
			"width": 480, "height": 480, "sha256": hex.EncodeToString(digest[:]),
			"consumer_contract": "native_story_dialogue_rgba_overlay_hd6_v1",
			"speaker_id":        speaker, "frame": 0, "mouth_state": "closed",
		})
	})
}

func TestModernHDGroupPreservesLegacyCompatibility(t *testing.T) {
	path, root := writeModernPortraitFixture(t, true)
	legacy, err := loadModernStoryPortraitSet(path, root)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.hasHD || legacy.portraits[0].hdImage != nil {
		t.Fatal("legacy pack enabled HD")
	}
	addModernHDFixture(t, path, root, 0, 480, 255)
	hd, err := loadModernStoryPortraitSet(path, root)
	if err != nil {
		t.Fatal(err)
	}
	if !hd.hasHD || hd.portraits[0].hdImage.Bounds() != image.Rect(0, 0, 480, 480) ||
		hd.portraits[0].image.Bounds() != image.Rect(0, 0, 80, 80) {
		t.Fatal("HD or compatible base missing")
	}
}

func TestModernHDRejectsInvalidGroupsAtomically(t *testing.T) {
	for _, name := range []string{"partial", "orphan", "duplicate", "geometry", "alpha", "hash"} {
		t.Run(name, func(t *testing.T) {
			path, root := writeModernPortraitFixture(t, true)
			speaker, size, alpha := 0, 480, uint8(255)
			if name == "orphan" {
				speaker = 99
			}
			if name == "geometry" {
				size = 479
			}
			if name == "alpha" {
				alpha = 128
			}
			addModernHDFixture(t, path, root, speaker, size, alpha)
			changeModernHDCatalog(t, path, func(document map[string]any) {
				assets := document["assets"].([]any)
				switch name {
				case "partial":
					second := map[string]any{}
					for key, value := range assets[0].(map[string]any) {
						second[key] = value
					}
					second["speaker_id"] = 1
					document["assets"] = append(assets, second)
				case "duplicate":
					document["assets"] = append(assets, assets[1])
				case "hash":
					assets[1].(map[string]any)["sha256"] = string(make([]byte, 64))
				}
			})
			got, err := loadModernStoryPortraitSet(path, root)
			if err == nil || got != nil {
				t.Fatalf("invalid group published: %v %v", got, err)
			}
		})
	}
}

func TestModernHDPrivateCatalogIncludesEveryPortrait(t *testing.T) {
	pack := os.Getenv("FD2_TEST_MODERN_PACK")
	if pack == "" {
		t.Skip("set FD2_TEST_MODERN_PACK")
	}
	set, err := loadModernStoryPortraitSet(assetPath("assets/themes/modern/catalog.json"), pack)
	if err != nil {
		t.Fatal(err)
	}
	if !set.hasHD {
		t.Fatal("private HD pack was not activated")
	}
	for speaker, picture := range set.portraits {
		if picture.hdImage == nil || picture.hdImage.Bounds() != image.Rect(0, 0, 480, 480) {
			t.Fatalf("speaker %d missing HD", speaker)
		}
	}
}

type modernHDDrawCapture struct {
	images  []*ebiten.Image
	options []ebiten.DrawImageOptions
}

func (c *modernHDDrawCapture) DrawImage(image *ebiten.Image, options *ebiten.DrawImageOptions) {
	c.images = append(c.images, image)
	c.options = append(c.options, *options)
}
func TestModernHDFinalPreservesDetailScaleAndAnchors(t *testing.T) {
	base := ebiten.NewImage(640, 400)
	portrait := &modernStoryPortraitFrame{hdImage: image.NewNRGBA(image.Rect(0, 0, 480, 480))}
	for _, rect := range []image.Rectangle{image.Rect(464, 10, 624, 170), image.Rect(16, 230, 176, 390)} {
		target := &modernHDDrawCapture{}
		var geometry ebiten.GeoM
		geometry.Scale(3, 3)
		geometry.Translate(11, 7)
		drawModernHDFinal(target, base, geometry, &modernHDOverlay{portrait: portrait, rect: rect}, 1)
		if len(target.images) != 2 || target.images[0] != base || target.images[1].Bounds() != image.Rect(0, 0, 480, 480) {
			t.Fatal("final layer did not submit the original 480-pixel image")
		}
		transform := target.options[1].GeoM
		x, y := transform.Apply(0, 0)
		if x != float64(rect.Min.X*3+11) || y != float64(rect.Min.Y*3+7) {
			t.Fatalf("incorrect final anchor %v %v", x, y)
		}
		edgeX, edgeY := transform.Apply(480, 480)
		if math.Abs(edgeX-x-480) > 1e-6 || math.Abs(edgeY-y-480) > 1e-6 {
			t.Fatal("portrait rectangle changed")
		}
		nextX, _ := transform.Apply(1, 0)
		if math.Abs(nextX-x-1) > 1e-6 {
			t.Fatal("source detail was downsampled before final display")
		}
		cached := portrait.hdGPU
		drawModernHDFinal(target, base, geometry, &modernHDOverlay{portrait: portrait, rect: rect}, 0)
		if cached != portrait.hdGPU {
			t.Fatal("GPU portrait recreated each draw")
		}
	}
}

func TestModernHDDrawClearsPreviousDialogueOverlay(t *testing.T) {
	g := &Game{startupBlocked: true, modernHDOverlay: &modernHDOverlay{}}
	g.Draw(ebiten.NewImage(640, 400))
	if g.modernHDOverlay != nil {
		t.Fatal("old dialogue portrait leaked into a later frame")
	}
}

func TestModernHDFinalSuppressesPortraitUnderModalAndDAC(t *testing.T) {
	for _, name := range []string{"normal", "help", "status", "transient", "white", "black", "dac"} {
		t.Run(name, func(t *testing.T) {
			portrait := &modernStoryPortraitFrame{hdImage: image.NewNRGBA(image.Rect(0, 0, 480, 480))}
			g := &Game{modernHDOverlay: &modernHDOverlay{portrait: portrait, rect: image.Rect(464, 10, 624, 170)}}
			switch name {
			case "help":
				g.helpVisible = true
			case "status":
				g.nativePlayerStatus = &nativePlayerStatusState{}
			case "transient":
				g.transientUI = true
			case "white":
				g.nativeFullDACWhite = true
			case "black":
				g.nativeFullDACBlack = true
			case "dac":
				g.nativeMapDAC = make([]byte, 768)
			}
			p := &modernHDPresentation{Game: g}
			var geometry ebiten.GeoM
			geometry.Scale(3, 3)
			p.DrawFinalScreen(ebiten.NewImage(1920, 1200), ebiten.NewImage(640, 400), geometry)
			if (portrait.hdGPU != nil) != (name == "normal") {
				t.Fatal("portrait visibility bypassed top-level effect")
			}
		})
	}
}
