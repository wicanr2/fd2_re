package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func TestModernHDShopComparisonSourceAndFallback(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	for _, variant := range []int{1, 3, 5} {
		shopDialogueTestState(g, variant, "recipient_equipment", 0, 0)
		frame, ok := shopDialogueCompose(g)
		if !ok {
			t.Fatal("actual equipment recipient source absent")
		}
		before := append([]byte(nil), frame...)
		party, _ := json.Marshal(g.partyRoster)
		g.modernHDText = &modernHDTextFrame{}
		g.appendModernHDShopComparison(frame)
		if len(g.modernHDText.serviceIcons) != 24 {
			t.Fatal("complete actual comparison rejected", variant)
		}
		h := g.shopComparisonHD
		assets, _, _, _ := g.nativeShopState()
		if g.prepareModernHDShopComparison(assets.CompareCells) != h {
			t.Fatal("identical material missed cache")
		}
		// Independent CPU font drawing at the known base sizes checks the ink,
		// placement and opaque source erasure before exercising GPU placement.
		for i, label := range shopComparisonLabels {
			want := image.NewRGBA(h.icons[i].picture.Bounds())
			pal, _ := preparationCursorPalette(g.nativeClassUI.palette)
			draw.Draw(want, want.Bounds(), image.NewUniform(pal[73]), image.Point{}, draw.Src)
			face, _ := g.font.faceFor(label.height * 6)
			bounds, _ := font.BoundString(face, label.text)
			box := image.Rect(bounds.Min.X.Floor(), bounds.Min.Y.Floor(), bounds.Max.X.Ceil(), bounds.Max.Y.Ceil())
			if box.Dx() > want.Bounds().Dx() || box.Dy() > want.Bounds().Dy() {
				t.Fatal("base font no longer fits original rectangle")
			}
			d := font.Drawer{Dst: want, Src: image.NewUniform(pal[169]), Face: face, Dot: fixed.P(-box.Min.X, -box.Min.Y)}
			d.DrawString(label.text)
			if !bytes.Equal(want.Pix, h.icons[i].picture.Pix) {
				t.Fatal("label differs from independent base-font drawing", label.text)
			}
		}
		for _, p := range shopComparisonPlacements(3) {
			covered := append([]byte(nil), frame...)
			covered[p.origin.Y*320+p.origin.X] ^= 1
			g.modernHDText = &modernHDTextFrame{}
			g.appendModernHDShopComparison(covered)
			if len(g.modernHDText.serviceIcons) != 0 {
				t.Fatal("partially covered source published labels", p)
			}
		}
		for i := range assets.CompareCells {
			bad := assets.CompareCells
			bad[i].Pixels = append([]byte(nil), bad[i].Pixels...)
			bad[i].Pixels[0] ^= 1
			if g.prepareModernHDShopComparison(bad) != nil {
				t.Fatal("different source reused cached material", i)
			}
			bad = assets.CompareCells
			bad[i].Width++
			if g.prepareModernHDShopComparison(bad) != nil {
				t.Fatal("different geometry admitted", i)
			}
		}
		for _, flag := range []string{"job", "help", "transient", "black", "white", "mode", "owner"} {
			g.modernHDText = &modernHDTextFrame{}
			g.helpVisible, g.transientUI = flag == "help", flag == "transient"
			g.nativeFullDACBlack, g.nativeFullDACWhite = flag == "black", flag == "white"
			if flag == "job" {
				g.nativeShopUIJob = &nativeClassUIJob{}
			}
			if flag == "mode" {
				g.nativeShopMode = "recipient_consumable"
			}
			if flag == "owner" {
				g.camp.Node().NativeHubVariant = 0
			}
			g.appendModernHDShopComparison(frame)
			if len(g.modernHDText.serviceIcons) != 0 {
				t.Fatal("covered owner retained labels", flag)
			}
			g.nativeShopUIJob = nil
			g.nativeShopMode, g.camp.Node().NativeHubVariant = "recipient_equipment", variant
		}
		g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
		g.modernHDText = nil
		g.appendModernHDShopComparison(frame)
		after, _ := json.Marshal(g.partyRoster)
		if !bytes.Equal(before, frame) || !bytes.Equal(party, after) {
			t.Fatal("presentation changed original frame or party")
		}
	}
	assets, _, _, _ := g.nativeShopState()
	palette := g.nativeClassUI.palette
	g.nativeClassUI.palette = append(color.Palette(nil), palette...)
	g.nativeClassUI.palette[73] = color.RGBA{1, 2, 3, 255}
	old := g.shopComparisonHD
	if h := g.prepareModernHDShopComparison(assets.CompareCells); h == nil || h == old {
		t.Fatal("palette change reused old material")
	}
	g.nativeClassUI.palette[73] = nil
	if g.prepareModernHDShopComparison(assets.CompareCells) != nil {
		t.Fatal("incomplete palette admitted")
	}
	g.nativeClassUI.palette = palette
	old = g.prepareModernHDShopComparison(assets.CompareCells)
	g.font.userScale = 1.5
	if h := g.prepareModernHDShopComparison(assets.CompareCells); h == nil || h == old {
		t.Fatal("font scale reused old material")
	}
	g.font.userScale = 1
	originalFont := g.font
	g.font = &Font{}
	if g.prepareModernHDShopComparison(assets.CompareCells) != nil {
		t.Fatal("missing font admitted")
	}
	g.font = originalFont
}

func TestModernHDShopComparisonRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SHOP_COMPARISON_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	set := facilityHDTestSet(t)
	g := facilityDialogueTestGame(t, set)
	facilityRosterTestEightParty(g)
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}, captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		hd := g.interfaceMode == playerInterfaceHD
		var placements []modernHDServiceIconPlacement
		if g.modernHDText != nil {
			placements = g.modernHDText.serviceIcons
		}
		if (len(placements) == 24) != hd {
			return fmt.Errorf("case%d wrong comparison count %d", p.index, len(placements))
		}
		source, ok := g.modernFacilityRosterSource()
		if !ok {
			return fmt.Errorf("case%d native owner absent", p.index)
		}
		native, ok := shopDialogueCompose(g)
		if !ok {
			return fmt.Errorf("case%d native composition absent", p.index)
		}
		checked := 0
		for _, at := range shopComparisonPlacements(len(source.ids)) {
			label := shopComparisonLabels[at.cell]
			for y := 0; y < label.height*6; y++ {
				for x := 0; x < label.width*6; x++ {
					value := native[(at.origin.Y+y/6)*320+at.origin.X+x/6]
					want := color.RGBAModel.Convert(g.nativeClassUI.palette[value]).(color.RGBA)
					if hd {
						want = g.shopComparisonHD.icons[at.cell].picture.RGBAAt(x, y)
					}
					i := ((at.origin.Y*6+y+72)*1920 + at.origin.X*6 + x) * 4
					if !bytes.Equal(pixels[i:i+4], []byte{want.R, want.G, want.B, 255}) {
						return fmt.Errorf("case%d label%d pixel%d,%d", p.index, at.cell, x, y)
					}
					checked++
				}
			}
		}
		if g.modernHDText != nil {
			g.modernHDText.serviceIcons = nil
		}
		reference := ebiten.NewImage(1920, 1272)
		defer reference.Dispose()
		var geometry ebiten.GeoM
		geometry.Scale(3, 3)
		p.presentation.DrawFinalScreen(reference, offscreen, geometry)
		control := make([]byte, len(pixels))
		reference.ReadPixels(control)
		if g.modernHDText != nil {
			g.modernHDText.serviceIcons = placements
		}
		outside := append([]byte(nil), pixels...)
		for _, placement := range placements {
			size := placement.icon.picture.Bounds().Size()
			zero := make([]byte, size.X*4)
			for y := 0; y < size.Y; y++ {
				i := ((placement.origin.Y*6+y+72)*1920 + placement.origin.X*6) * 4
				copy(outside[i:i+len(zero)], zero)
				copy(control[i:i+len(zero)], zero)
			}
		}
		if !bytes.Equal(outside, control) {
			return fmt.Errorf("case%d comparison layer changed pixels outside complete source rectangles", p.index)
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "variant": g.nativeShopVariant, "locale": g.localeID, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "font_scale": g.fontScale, "source_start": source.start, "comparison_pixels": checked, "all_outside_bytes_equal": true})
		return nil
	}
	add := func(variant, style int, locale string, scroll bool, scale float64) {
		if style == 3 {
			p.captureCases[len(p.cases)] = true
		}
		p.cases = append(p.cases, func() {
			shopDialogueTestState(g, variant, "recipient_equipment", 0, 0)
			if scroll {
				facilityRosterTestScroll(g, "shop", "recipient_equipment")
			}
			var err error
			g.localeCatalog, err = loadOfficialLocale(locale)
			if err != nil {
				t.Fatal(err)
			}
			g.localeContent, err = loadOfficialLocaleContent(locale)
			if err != nil {
				t.Fatal(err)
			}
			g.localeEntities, err = loadOfficialLocaleEntities(locale)
			if err != nil {
				t.Fatal(err)
			}
			g.localeID, g.fontScale = locale, scale
			g.interfaceMode, g.modernStoryPortraits = playerInterfaceOriginal, nil
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
			if style >= 2 {
				g.modernStoryPortraits = set
			}
		})
	}
	for _, variant := range []int{1, 3, 5} {
		for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
			for style := 0; style < 4; style++ {
				add(variant, style, locale, false, 1)
			}
		}
		for style := 0; style < 4; style++ {
			add(variant, style, "zh-Hant", true, 1)
		}
	}
	for style := 0; style < 4; style++ {
		add(1, style, "zh-Hant", true, 1.5)
	}
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != 64 {
		t.Fatal("incomplete comparison matrix", len(p.checks))
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-SHOP-COMPARISON-GPU", "scope": "three shops/four independent styles/four locales/scroll/font scale; complete label and arrow RGBA against source and existing CPU raster, all outside bytes equal with only this layer disabled; no new oracle claim", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
