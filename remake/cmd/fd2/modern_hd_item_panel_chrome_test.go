package main

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func itemPanelChromeTestSource(t *testing.T) (*nativeUnitStatusData, color.Palette, dialogueChromeSource) {
	t.Helper()
	ui, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	d := &nativeUnitStatusData{base: make([]byte, 64000), assets: assets}
	if err := battle.RenderNativeItemPanelBaseResources(separatedAssetPath(""), separatedAssetPath("portraits"), 0, d.base); err != nil {
		t.Fatal(err)
	}
	return d, ui.palette, nativeDialogueChromeSource(ui.dialogue, ui.palette)
}

func itemPanelChromePixel(source dialogueChromeSource, x, y, w, h int) ([4]byte, bool) {
	if x < 0 || y < 0 || x >= w || y >= h || x >= 18 && x < w-18 && y >= 18 && y < h-18 {
		return [4]byte{}, false
	}
	if (x < 3 || x >= w-3) && (y < 3 || y >= h-3) {
		corner, dx, dy := 0, x, y
		if x >= w-3 {
			corner, dx = 1, x-w+3
		}
		if y >= h-3 {
			corner, dy = corner+2, y-h+3
		}
		return source.Corners[corner][dy*3+dx], true
	}
	switch {
	case y < 3:
		return source.Edges[0][y], true
	case x >= w-3:
		return source.Edges[1][x-w+3], true
	case y >= h-3:
		return source.Edges[2][y-h+3], true
	case x < 3:
		return source.Edges[3][x], true
	default:
		return source.Background, true
	}
}

func TestModernHDItemPanelChromeSourceAndGeometry(t *testing.T) {
	d, palette, source := itemPanelChromeTestSource(t)
	before := append([]byte(nil), d.base...)
	c, err := prepareModernHDItemPanelChrome(d, palette)
	if err != nil {
		t.Fatal(err)
	}
	points := 0
	for i, size := range []image.Point{{86, 86}, {223, 86}, {310, 99}} {
		patch := c.frames[i].patchFor(size.X*6, size.Y*6)
		for y := 0; y < size.Y*6; y++ {
			for x := 0; x < size.X*6; x++ {
				want, opaque := itemPanelChromePixel(source, x, y, size.X*6, size.Y*6)
				got := patch.RGBAAt(x, y)
				if opaque && got != (color.RGBA{want[0], want[1], want[2], want[3]}) || !opaque && got != (color.RGBA{}) {
					t.Fatalf("panel%d differs at%d,%d", i, x, y)
				}
				points++
			}
		}
	}
	for frame := 0; frame < 12; frame++ {
		pass, err := battle.NativeItemPanelFrameFor(frame)
		if err != nil {
			t.Fatal(err)
		}
		regions := []battle.NativeItemPanelRegion{pass.Left, pass.Upper, pass.Bottom}
		placements := modernHDItemPanelPlacements(c, regions)
		coverage := make([]byte, 64000)
		for _, p := range placements {
			if !p.status || p.clip.Empty() {
				t.Fatal("missing owner or clipping")
			}
			visible := p.clip.Intersect(image.Rect(0, 0, 320, 200))
			for y := visible.Min.Y; y < visible.Max.Y; y++ {
				for x := visible.Min.X; x < visible.Max.X; x++ {
					coverage[y*320+x] = 37
				}
			}
		}
		panel := bytes.Repeat([]byte{37}, 64000)
		native, err := nativeChurchPanelFrame(make([]byte, 64000), panel, frame)
		if err != nil || !bytes.Equal(native, coverage) {
			t.Fatal("native clipped coverage differs", frame, err)
		}
		for _, p := range placements {
			if p.rect.Min.Y >= 94 && p.rect.Dy() != 99 {
				t.Fatal("copy padding was treated as artwork")
			}
		}
	}
	if !bytes.Equal(before, d.base) {
		t.Fatal("native source changed")
	}
	t.Log("independent full patch points", points, "native clipped frames", 12)
}

func TestModernHDItemPanelChromeFreezeAndStaging(t *testing.T) {
	d, palette, _ := itemPanelChromeTestSource(t)
	c, err := prepareModernHDItemPanelChrome(d, palette)
	if err != nil {
		t.Fatal(err)
	}
	d.itemChrome = c
	check := func() {
		t.Helper()
		if _, err := prepareModernHDItemPanelChrome(d, palette); err == nil || d.itemChrome != c {
			t.Fatal("changed source accepted or replaced frozen owner", err)
		}
	}
	d.base[0] ^= 1
	check()
	d.base[0] ^= 1
	original := d.assets.RawCells[13]
	changed := original
	changed.Pixels = append([]byte(nil), original.Pixels...)
	changed.Pixels[0] ^= 1
	d.assets.RawCells[13] = changed
	check()
	d.assets.RawCells[13] = original
	width := d.assets.ItemPanelBottom.Height
	d.assets.ItemPanelBottom.Height = 102
	check()
	d.assets.ItemPanelBottom.Height = width
	if restored, err := prepareModernHDItemPanelChrome(d, palette); err != nil || restored != c {
		t.Fatal("restored source", err)
	}
	bad := *d
	bad.base = nil
	g := &Game{nativeItemPanelText: &nativeItemPanelTextData{data: *d}, nativeUIPalette: palette, nativePlayerStatus: &nativePlayerStatusState{data: &bad}}
	if publish, err := g.stageModernHDItemPanels(&modernStoryPortraitSet{hasHD: true}); err == nil || publish != nil || g.nativeItemPanelText.data.itemChrome != c {
		t.Fatal("failed theme preflight partly published", err)
	}
	if publish, err := g.stageModernHDItemPanels(nil); err != nil || publish == nil {
		t.Fatal("pixel mode blocked", err)
	}
	g.nativePlayerStatus = nil
	g.nativeItemPanelText.data.itemChrome = nil
	publish, err := g.stageModernHDItemPanels(&modernStoryPortraitSet{hasHD: true})
	if err != nil || publish == nil || g.nativeItemPanelText.data.itemChrome != nil {
		t.Fatal("staging published early", err)
	}
	publish()
	if g.nativeItemPanelText.data.itemChrome == nil {
		t.Fatal("staging did not publish")
	}
	for _, broken := range []*nativeUnitStatusData{nil, {base: make([]byte, 20)}, {base: make([]byte, 64000), assets: battle.NativeItemPanelDataAssets{RawCells: map[int]fdother.RawCell{}}}} {
		if _, err := prepareModernHDItemPanelChrome(broken, palette); err == nil {
			t.Fatal("missing source accepted")
		}
	}
}
