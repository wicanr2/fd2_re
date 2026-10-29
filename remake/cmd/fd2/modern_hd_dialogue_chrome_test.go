package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// This oracle reads the actual raw cells independently of the frozen renderer.
type dialogueChromeSource struct {
	NativeSourceSHA  string        `json:"native_source_sha256,omitempty"`
	GoldReferenceSHA string        `json:"gold_reference_sha256,omitempty"`
	Gold             bool          `json:"gold,omitempty"`
	Brightness       float64       `json:"brightness,omitempty"`
	Background       [4]byte       `json:"background"`
	Edges            [4][3][4]byte `json:"edges"`
	Corners          [4][9][4]byte `json:"corners"`
}

var goldReferenceOnce sync.Once
var goldReference image.Image

func independentGoldReference() image.Image {
	goldReferenceOnce.Do(func() {
		file, err := os.Open("../../../docs/figures/hd1-gold-dialogue-frame-reference.png")
		if err != nil {
			panic(err)
		}
		defer file.Close()
		goldReference, err = png.Decode(file)
		if err != nil {
			panic(err)
		}
	})
	return goldReference
}

func goldDialogueChromeSource(cells []fdother.RawCell, palette color.Palette) dialogueChromeSource {
	s := nativeDialogueChromeSource(cells, palette)
	data, _ := json.Marshal(s)
	s.NativeSourceSHA = fmt.Sprintf("%x", sha256.Sum256(data))
	s.GoldReferenceSHA = "6084bbe6a90b9ae9a81ca0f4dcacfc3ed4167e67ce85f48140d63c2d744f76c8"
	s.Gold, s.Brightness = true, 1
	return s
}

func nativeDialogueChromeSource(cells []fdother.RawCell, palette color.Palette) dialogueChromeSource {
	rgba := func(index byte) [4]byte {
		r, g, b, a := palette[index].RGBA()
		return [4]byte{byte(r >> 8), byte(g >> 8), byte(b >> 8), byte(a >> 8)}
	}
	source := dialogueChromeSource{Background: rgba(cells[13].Pixels[0])}
	for side, index := range []int{9, 11, 12, 10} {
		for band := 0; band < 3; band++ {
			at := band*16 + 8
			if cells[index].Width == 3 {
				at = 8*3 + band
			}
			source.Edges[side][band] = rgba(cells[index].Pixels[at])
		}
	}
	for corner := 0; corner < 4; corner++ {
		for i, index := range cells[corner+1].Pixels {
			source.Corners[corner][i] = rgba(index)
		}
	}
	return source
}

func (s dialogueChromeSource) pixel(x, y int) ([4]byte, bool) {
	const w, h = 1860, 516
	if x >= 18 && x < w-18 && y >= 18 && y < h-18 {
		return [4]byte{}, false
	}
	if s.Gold {
		r, g, b, a := independentGoldReference().At(x, y).RGBA()
		return [4]byte{byte(float64(r>>8) * s.Brightness), byte(float64(g>>8) * s.Brightness), byte(float64(b>>8) * s.Brightness), byte(a >> 8)}, true
	}
	if (x < 3 || x >= w-3) && (y < 3 || y >= h-3) {
		corner, dx, dy := 0, x, y
		if x >= w-3 {
			corner, dx = 1, x-w+3
		}
		if y >= h-3 {
			corner, dy = corner+2, y-h+3
		}
		return s.Corners[corner][dy*3+dx], true
	}
	switch {
	case y < 3:
		return s.Edges[0][y], true
	case x >= w-3:
		return s.Edges[1][x-w+3], true
	case y >= h-3:
		return s.Edges[2][y-h+3], true
	case x < 3:
		return s.Edges[3][x], true
	default:
		return s.Background, true
	}
}

func checkDialogueChromePixels(pixels []byte, source dialogueChromeSource, nativeRect image.Rectangle) (int, error) {
	if nativeRect.Empty() {
		return 0, nil
	}
	visible := nativeRect.Intersect(image.Rect(0, 0, 320, 200))
	count := 0
	for y := visible.Min.Y * 6; y < visible.Max.Y*6; y++ {
		for x := visible.Min.X * 6; x < visible.Max.X*6; x++ {
			want, painted := source.pixel(x-nativeRect.Min.X*6, y-nativeRect.Min.Y*6)
			if !painted {
				continue
			}
			at := (y*1920 + x) * 4
			for c := 0; c < 4; c++ {
				d := int(pixels[at+c]) - int(want[c])
				if d < -1 || d > 1 {
					return count, fmt.Errorf("HD dialogue chrome differs at%d,%d channel%d got%d want%d", x, y, c, pixels[at+c], want[c])
				}
			}
			count++
		}
	}
	return count, nil
}

func TestModernHDDialogueChromeSourceAndPatch(t *testing.T) {
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), assets.dialogue[13].Pixels...)
	palette := append(color.Palette(nil), assets.palette...)
	chrome, err := freezeModernHDDialogueChrome(assets.dialogue, palette)
	if err != nil {
		t.Fatal(err)
	}
	reference := nativeDialogueChromeSource(assets.dialogue, assets.palette)
	patch := chrome.patch()
	points := 0
	for y := 0; y < 516; y++ {
		for x := 0; x < 1860; x++ {
			want, opaque := reference.pixel(x, y)
			got := patch.RGBAAt(x, y)
			if opaque && got != (color.RGBA{want[0], want[1], want[2], want[3]}) || !opaque && got != (color.RGBA{}) {
				t.Fatalf("patch differs at%d,%d", x, y)
			}
			points++
		}
	}
	if !bytes.Equal(before, assets.dialogue[13].Pixels) {
		t.Fatal("native source mutated")
	}
	// Source colors are frozen, not live references to a mutable palette.
	palette[74] = color.RGBA{255, 0, 0, 255}
	if !bytes.Equal(patch.Pix, chrome.patch().Pix) {
		t.Fatal("published chrome followed a later palette")
	}
	t.Log("independent patch pixels", points)
}

func TestModernHDDialogueChromeRejectsSourcesBeforeCountdown(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g, unit := expiryHDTestGame(t, nil, "zh-Hant")
	assets := g.nativeClassUI
	for _, index := range []int{1, 9, 10, 11, 12, 13} {
		cells := append([]fdother.RawCell(nil), assets.dialogue...)
		cells[index].Pixels = nil
		if _, err := freezeModernHDDialogueChrome(cells, assets.palette); err == nil {
			t.Fatal("invalid raw cell accepted", index)
		}
	}
	cells := append([]fdother.RawCell(nil), assets.dialogue...)
	cells[13].Pixels = append([]byte(nil), cells[13].Pixels...)
	cells[13].Pixels[0] ^= 1
	g.nativeClassUI = &nativeClassUIAssets{}
	*g.nativeClassUI = *assets
	g.nativeClassUI.dialogue = cells
	g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
	before, state := *unit, g.st
	if err := g.beginNativeTransientPhases([]byte{2}, nil); err == nil || !strings.Contains(err.Error(), "background is not uniform") {
		t.Fatal("bad background did not reach chrome guard", err)
	}
	if g.st != state || !reflect.DeepEqual(before, *unit) || g.nativeClassUIJob != nil || g.transientUI {
		t.Fatal("bad chrome published countdown or HP")
	}
	palette := append(color.Palette(nil), assets.palette...)
	palette[74] = nil
	if _, err := freezeModernHDDialogueChrome(assets.dialogue, palette); err == nil {
		t.Fatal("missing source color accepted")
	}
}
