package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// Literal native source and the published HD border recipe, independent of
// the production picture builder and currencyRun.
func moneyReferencePixel(c []byte, palette color.Palette, x, y int) color.RGBA {
	index := byte(73)
	switch {
	case (x < 3 || x >= 375) && (y < 3 || y >= 87):
		sx, sy := x, y
		if x >= 375 {
			sx = x - 375 + 60
		}
		if y >= 87 {
			sy = y - 87 + 12
		}
		index = c[sy*63+sx]
	case y < 3:
		index = c[y*63+20]
	case x >= 375:
		index = c[4*63+60+x-375]
	case y >= 87:
		index = c[(12+y-87)*63+20]
	case x < 3:
		index = c[4*63+x]
	}
	return color.RGBAModel.Convert(palette[index]).(color.RGBA)
}
func moneyGoldRollTestJob(t *testing.T, g *Game, variant int, credit bool, which int) {
	t.Helper()
	shopDialogueTestState(g, variant, "success", 0, 0)
	a, _, _, ok := g.nativeShopState()
	if !ok {
		t.Fatal("roll assets missing")
	}
	oldGold, amount := 1000, 10
	compose := campaign.ComposeNativeGoldDebitFrames
	if credit {
		oldGold, amount = 990, 7
		compose = campaign.ComposeNativeGoldCreditFrames
		g.nativeShopMode = "sell_success"
	}
	base, err := campaign.ComposeNativeShopBareScene(a, g.nativeClassUI.digits, oldGold)
	if err != nil {
		t.Fatal(err)
	}
	frames, next, err := compose(base, a.GoldRollStrip, oldGold, amount)
	if err != nil {
		t.Fatal(err)
	}
	plan := g.snapshotNativeGoldRoll(a, oldGold, next, credit)
	if plan == nil {
		t.Fatal("roll snapshot missing")
	}
	index := 0
	if which == 1 {
		index = len(frames) / 2
	} else if which == 2 {
		index = len(frames) - 1
	}
	job := &nativeClassUIJob{goldText: plan}
	for _, raw := range frames {
		job.timeline = append(job.timeline, nativeClassUITimelineStep{frame: raw, palette: g.nativeClassUI.palette, duration: 10 * time.Millisecond})
	}
	job.elapsed = time.Duration(index)*10*time.Millisecond + time.Millisecond
	g.nativeShopUIJob = job
	g.gold = oldGold
}
func TestModernHDFacilityMoneyRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FACILITY_MONEY_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	set := facilityHDTestSet(t)
	g := facilityDialogueTestGame(t, set)
	facilityRosterTestEightParty(g)
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}, captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		owner := g.camp.Node().Type
		ui := g.interfaceMode == playerInterfaceHD
		d := (*modernHDFacilityMoneyPanel)(nil)
		if g.modernHDText != nil {
			d = g.modernHDText.money
		}
		if ui && (d == nil || !d.presented) {
			return fmt.Errorf("case%d actual money Draw absent", p.index)
		}
		if !ui && d != nil {
			return fmt.Errorf("original interface admitted money")
		}
		c, ok := moneyPrototypeCell(g, owner)
		if !ok {
			t.Fatal("reference decoration absent")
		}
		// Baseline is the same formal Game.Draw with only the new decoration omitted.
		p.presentation.Draw(offscreen)
		if g.modernHDText != nil {
			g.modernHDText.money = nil
		}
		baseline := ebiten.NewImage(1920, 1272)
		defer baseline.Dispose()
		var geometry ebiten.GeoM
		geometry.Scale(3, 3)
		p.presentation.DrawFinalScreen(baseline, offscreen, geometry)
		raw := make([]byte, 1920*1272*4)
		baseline.ReadPixels(raw)
		want := append([]byte(nil), raw[72*1920*4:]...)
		if ui {
			for y := 0; y < 90; y++ {
				for x := 0; x < 378; x++ {
					if image.Pt(5+x/6, 95+y/6).In(moneyPrototypeDigits) {
						continue
					}
					ink := moneyReferencePixel(c.Pixels, g.nativeClassUI.palette, x, y)
					at := ((570+y)*1920 + 30 + x) * 4
					copy(want[at:at+4], []byte{ink.R, ink.G, ink.B, ink.A})
				}
			}
			surface := ebiten.NewImageFromImage(&image.RGBA{Pix: want, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
			defer surface.Dispose()
			g.modernHDText = &modernHDTextFrame{runs: []modernHDTextRun{{font: g.font, text: "$", x: 18, y: 198, pixels: 20, color: nativePaletteColor(g.nativeClassUI.palette, 169, color.RGBA{}), clip: image.Rect(18, 198, 30, 214), inkTop: true}}}
			p.presentation.drawModernHDText(surface, geometry, 1)
			surface.ReadPixels(want)
		}
		p.presentation.Draw(offscreen)
		p.presentation.DrawFinalScreen(baseline, offscreen, geometry)
		got := pixels[72*1920*4:]
		if !bytes.Equal(got, want) {
			for name, data := range map[string][]byte{"actual": got, "expected": want} {
				f, err := os.Create(filepath.Join(out, "mismatch-"+name+".png"))
				if err == nil {
					_ = png.Encode(f, &image.RGBA{Pix: data, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
					_ = f.Close()
				}
			}
			for i, v := range want {
				if got[i] != v {
					return fmt.Errorf("case%d %s/%d mode=%s/%s %s art=%t pixel(%d,%d)c%d %d != %d", p.index, owner, g.nativeShopVariant, g.nativeShopMode, g.churchMode, g.interfaceMode, g.modernStoryPortraits != nil, i/4%1920, i/4/1920, i%4, got[i], v)
				}
			}
		}
		rolling := d != nil && d.rolling
		variant := g.nativeShopVariant
		if owner == "church" {
			variant = 0
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "owner": owner, "variant": variant, "shop_mode": g.nativeShopMode, "church_mode": g.churchMode, "rolling": rolling, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "locale": g.localeID, "whole_game_rgba_bytes": len(want), "rgba_tolerance": 0, "independent_static_composite_equal": true})
		return nil
	}
	add := func(owner string, variant int, mode, kind, locale string, index, style int, capture bool) {
		if capture {
			p.captureCases[len(p.cases)] = true
		}
		p.cases = append(p.cases, func() {
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
			if kind == "debit" || kind == "credit" {
				moneyGoldRollTestJob(t, g, variant, kind == "credit", index)
			} else {
				if owner == "shop" {
					shopDialogueTestState(g, variant, mode, 0, 0)
				} else {
					facilityDialogueTestOwner(g, owner, mode, 0, 0)
				}
				if kind == "menu_open" || kind == "menu_close" {
					serviceMenuBoundTestJob(t, g, owner, variant, kind == "menu_close")
					if owner == "shop" {
						g.nativeShopUIJob.frame = index
					} else {
						g.nativeChurchUIJob.frame = index
					}
				}
			}
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
			if style >= 2 {
				g.modernStoryPortraits = set
			}
		})
	}
	for _, owner := range []string{"shop", "church"} {
		variants := []int{1, 3, 5}
		if owner == "church" {
			variants = []int{0}
		}
		for _, variant := range variants {
			modes := facilityDialogueChurchModes
			if owner == "shop" {
				modes = nil
				for _, m := range shopDialogueModes {
					modes = append(modes, m.mode)
				}
			}
			for _, mode := range modes {
				styles := []int{3}
				if mode == "menu" {
					styles = []int{0, 1, 2, 3}
				}
				for _, style := range styles {
					add(owner, variant, mode, "stable", "zh-Hant", 0, style, style == 3 && (mode == "menu" || mode == "insufficient" || mode == "revive_insufficient"))
				}
			}
			for _, kind := range []string{"menu_open", "menu_close"} {
				for style := 0; style < 4; style++ {
					add(owner, variant, "menu", kind, "zh-Hant", 1, style, style == 3)
				}
			}
		}
	}
	for _, variant := range []int{1, 3, 5} {
		for _, kind := range []string{"debit", "credit"} {
			indices := []int{1}
			if variant == 1 {
				indices = []int{0, 1, 2}
			}
			for _, index := range indices {
				for style := 0; style < 4; style++ {
					add("shop", variant, "", kind, "zh-Hant", index, style, style == 3)
				}
			}
		}
	}
	for _, locale := range []string{"zh-Hans", "en", "ja"} {
		for _, owner := range []string{"shop", "church"} {
			variants := []int{1, 3, 5}
			if owner == "church" {
				variants = []int{0}
			}
			for _, variant := range variants {
				for style := 0; style < 4; style++ {
					add(owner, variant, "menu", "stable", locale, 0, style, false)
				}
			}
		}
	}
	expected := len(p.cases)
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != 207 || expected != 207 {
		t.Fatal("incomplete money matrix", len(p.checks), expected)
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-FACILITY-MONEY-GPU", "scope": "75 stable modes HD/HD; four controls for four menus, menu transitions, three-shop debit/credit windows and representative other locales; independent baseline plus literal HD border recipe and independent currency glyph; no new original oracle", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
