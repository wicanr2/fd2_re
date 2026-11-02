package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestModernHDChurchFeeCurrencySourceAndFallback(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	g.nativeShopUI = nil
	checked, rejected := 0, 0
	for count := 1; count <= 3; count++ {
		churchListPrototypeState(t, g, "revive", false)
		g.churchIDs = append([]int(nil), g.churchIDs[:count]...)
		frame, ok := g.composeNativeChurchReviveListFrame()
		if !ok {
			t.Fatal("native revive unavailable")
		}
		source := g.prepareNativeChurchFeeCurrency(frame)
		if source == nil || source.count != count {
			t.Fatal("typed row count lost", count)
		}
		positions, ok := source.placements(frame)
		if !ok || len(positions) != count {
			t.Fatal("valid source rejected")
		}
		for _, p := range positions {
			for y := 0; y < 8; y++ {
				for x := 0; x < 6; x++ {
					at := (p.Y+y)*320 + p.X + x
					frame[at] ^= 1
					if got, ok := source.placements(frame); ok || got != nil {
						t.Fatal("partial currency admitted")
					}
					frame[at] ^= 1
					rejected++
				}
			}
		}
		original := g.nativeClassUI.priceCell.Pixels[0]
		g.nativeClassUI.priceCell.Pixels[0] ^= 1
		if source.cell.Pixels[0] != original {
			t.Fatal("frozen source aliases live cell")
		}
		if _, ok := source.placements(frame); !ok {
			t.Fatal("frozen source changed")
		}
		if g.prepareNativeChurchFeeCurrency(frame) != nil {
			t.Fatal("unverified source cell admitted")
		}
		g.nativeClassUI.priceCell.Pixels[0] = original
		if got, ok := g.modernHDChurchFeeCurrencyPlacements(frame, source, nil); ok || got != nil {
			t.Fatal("missing theme admitted")
		}
		for _, fault := range []string{"mode", "owner", "job", "shop-job", "church-job", "help", "transient", "black", "white", "viewport"} {
			if fault == "mode" {
				g.churchMode = "menu"
			}
			if fault == "owner" {
				g.camp.Node().Type = "shop"
			}
			if fault == "job" {
				g.nativeClassUIJob = &nativeClassUIJob{}
			}
			if fault == "shop-job" {
				g.nativeShopUIJob = &nativeClassUIJob{}
			}
			if fault == "church-job" {
				g.nativeChurchUIJob = &nativeChurchUIJob{}
			}
			if fault == "viewport" {
				g.churchVerticalStart = 99
			}
			g.helpVisible, g.transientUI = fault == "help", fault == "transient"
			g.nativeFullDACBlack, g.nativeFullDACWhite = fault == "black", fault == "white"
			if g.prepareNativeChurchFeeCurrency(frame) != nil {
				t.Fatal("invalid owner admitted", fault)
			}
			g.churchMode = "revive"
			g.camp.Node().Type = "church"
			g.nativeClassUIJob = nil
			g.nativeShopUIJob = nil
			g.nativeChurchUIJob = nil
			g.churchVerticalStart = 0
			g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
		}
		checked++
	}
	for _, closing := range []bool{false, true} {
		for _, scroll := range []bool{false, true} {
			job := churchListTransitionTestJob(t, g, "revive", closing, scroll)
			d := job.facilityList
			if d.feeCurrency == nil || d.feeCurrency.count != 3 {
				t.Fatal("job lost frozen fee currency")
			}
			original := g.nativeClassUI.priceCell.Pixels[0]
			g.nativeClassUI.priceCell.Pixels[0] ^= 1
			if _, ok := d.feeCurrency.placements(d.final); !ok {
				t.Fatal("job currency aliases live source")
			}
			g.nativeClassUI.priceCell.Pixels[0] = original
			g.nativeClassUIJob = nil
		}
	}
	t.Logf("source row counts=%d covered-byte rejections=%d frozen jobs=4", checked, rejected)
}

func TestModernHDChurchFeeCurrencyRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_CHURCH_FEE_CURRENCY_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	set := facilityHDTestSet(t)
	g := facilityDialogueTestGame(t, set)
	facilityRosterTestEightParty(g)
	g.nativeShopUI = nil
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}, captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	currentScroll := false
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		positions := []facilityItemRowPlacement{{15, image.Pt(220, 125)}, {15, image.Pt(220, 151)}, {15, image.Pt(220, 177)}}
		var placements []modernHDServiceIconPlacement
		if g.modernHDText != nil {
			placements = g.modernHDText.serviceIcons
		}
		hd := g.interfaceMode == playerInterfaceHD
		if len(placements) != 0 && !hd || hd && len(placements) != len(positions) {
			return fmt.Errorf("case%d wrong icon count %d", p.index, len(placements))
		}
		native, ok := g.composeNativeChurchReviveListFrame()
		if !ok {
			return fmt.Errorf("case%d native source unavailable", p.index)
		}
		checked := 0
		for _, at := range positions {
			width, height := 24, 20
			if at.index >= 64 {
				width, height = 21, 8
			} else if at.index == 15 {
				width, height = 6, 8
			}
			for y := 0; y < height*6; y++ {
				for x := 0; x < width*6; x++ {
					value := native[(at.origin.Y+y/6)*320+at.origin.X+x/6]
					want := color.RGBAModel.Convert(g.nativeClassUI.palette[value]).(color.RGBA)
					if hd {
						want = g.facilityItemRowsHD.icons[at.index].picture.RGBAAt(x, y)
					}
					i := ((at.origin.Y*6+y+72)*1920 + at.origin.X*6 + x) * 4
					if !bytes.Equal(pixels[i:i+4], []byte{want.R, want.G, want.B, 255}) {
						return fmt.Errorf("case%d icon%d pixel%d,%d", p.index, at.index, x, y)
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
			return fmt.Errorf("case%d changed bytes outside item roles", p.index)
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "owner": g.camp.Node().Type, "variant": g.nativeShopVariant, "shop_mode": g.nativeShopMode, "church_mode": g.churchMode, "locale": g.localeID, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "scroll": currentScroll, "item_role_pixels": checked, "all_outside_bytes_equal": true})
		return nil
	}
	add := func(mode string, variant, style int, locale string, scroll bool) {
		if style == 3 && locale == "zh-Hant" {
			p.captureCases[len(p.cases)] = true
		}
		p.cases = append(p.cases, func() {
			churchListPrototypeState(t, g, "revive", scroll)
			currentScroll = scroll
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
			g.localeID = locale
			g.interfaceMode, g.modernStoryPortraits = playerInterfaceOriginal, nil
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
			if style >= 2 {
				g.modernStoryPortraits = set
			}
		})
	}
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		for _, scroll := range []bool{false, true} {
			for style := 0; style < 4; style++ {
				add("revive", 0, style, locale, scroll)
			}
		}
	}
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != 32 {
		t.Fatal("incomplete item-row matrix", len(p.checks))
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-CHURCH-FEE-CURRENCY-GPU", "scope": "church revive fee currency; four independent styles/four locale states/two viewports; complete 6x8 label RGBA and all outside bytes equal; existing R55 font contract; no new original oracle claim", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
