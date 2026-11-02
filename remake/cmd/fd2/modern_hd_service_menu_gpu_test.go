package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func serviceMenuStaticReference(t *testing.T, g *Game, owner string, closing bool, index int, set *modernStoryPortraitSet) []byte {
	t.Helper()
	churchJob := g.nativeChurchUIJob
	g.nativeChurchUIJob = nil
	want := facilityReplyStaticReference(t, g, owner, false)
	g.nativeChurchUIJob = churchJob
	if index == 4 {
		return want
	}
	resource, cells := serviceMenuTestCells(t, g, owner)
	divisor := 4 - index
	if closing {
		divisor = index + 1
	}
	hd := g.interfaceMode == playerInterfaceHD
	for option, offset := range []int{-39, -13, 13, 39} {
		c := cells[option]
		x0 := 240 + offset/divisor
		icon := set.uiServiceIcons[[2]int{resource, 3 + 2*option}]
		for y := 0; y < 20; y++ {
			for x := 0; x < 24; x++ {
				value := c.Pixels[y*24+x]
				if value == 0 {
					continue
				}
				rgba := nativeServiceMenuReferenceColor(g, value)
				for sy := 0; sy < 6; sy++ {
					for sx := 0; sx < 6; sx++ {
						at := ((169*6+y*6+sy)*1920 + x0*6 + x*6 + sx) * 4
						if hd {
							from := icon.picture.PixOffset(x*6+sx, y*6+sy)
							copy(want[at:at+4], icon.picture.Pix[from:from+4])
						} else {
							copy(want[at:at+4], rgba)
						}
					}
				}
			}
		}
	}
	return want
}
func nativeServiceMenuReferenceColor(g *Game, index byte) []byte {
	r, b, c, a := g.nativeClassUI.palette[index].RGBA()
	return []byte{byte(r >> 8), byte(b >> 8), byte(c >> 8), byte(a >> 8)}
}
func TestModernHDServiceMenuRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SERVICE_MENU_GPU") != "1" {
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
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}, captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		owner := g.camp.Node().Type
		d := serviceMenuTestDescriptor(g, owner)
		index := 0
		if owner == "shop" {
			index = g.nativeShopUIJob.frame
		} else {
			index = g.nativeChurchUIJob.frame
		}
		want := serviceMenuStaticReference(t, g, owner, d.closing, index, set)
		p.presentation.Draw(offscreen)
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
					return fmt.Errorf("case%d %s/%d closing=%t frame%d %s art=%t pixel(%d,%d)c%d %d != %d", p.index, owner, d.variant, d.closing, index, g.interfaceMode, g.modernStoryPortraits != nil, i/4%1920, i/4/1920, i%4, got[i], v)
				}
			}
		}
		phase := "menu"
		if index == 4 {
			phase = "restore"
		}
		hdDraw := g.interfaceMode == playerInterfaceHD || g.modernStoryPortraits != nil
		if hdDraw {
			if d.lastDraw == nil || d.lastDraw.frame != index || d.lastDraw.phase != phase {
				return fmt.Errorf("actual menu Draw absent")
			}
		} else if d.lastDraw != nil {
			return fmt.Errorf("original controls retained HD Draw")
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "owner": owner, "variant": d.variant, "resource": d.resource, "closing": d.closing, "frame": index, "phase": phase, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "locale": g.localeID, "whole_game_rgba_bytes": len(want), "rgba_tolerance": 0, "independent_static_composite_equal": true})
		return nil
	}
	add := func(owner string, variant int, closing bool, locale string, index, style int, capture bool) {
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
			serviceMenuBoundTestJob(t, g, owner, variant, closing)
			if owner == "shop" {
				g.nativeShopUIJob.frame = index
			} else {
				g.nativeChurchUIJob.frame = index
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
			for _, closing := range []bool{false, true} {
				count := 4
				if closing {
					count = 5
				}
				for index := 0; index < count; index++ {
					for style := 0; style < 4; style++ {
						add(owner, variant, closing, "zh-Hant", index, style, style == 3)
					}
				}
			}
		}
	}
	for _, locale := range []string{"zh-Hans", "en", "ja"} {
		for _, owner := range []string{"shop", "church"} {
			variant := 1
			if owner == "church" {
				variant = 0
			}
			for _, closing := range []bool{false, true} {
				for style := 0; style < 4; style++ {
					add(owner, variant, closing, locale, 1, style, false)
				}
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
	if len(p.checks) != 192 {
		t.Fatal("incomplete menu matrix", len(p.checks))
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-SERVICE-MENU-GPU", "scope": "all four native steps and restores, three shops and church, four controls; other three locales representative shop/church steps; independent static source and literal per-pixel writes, no new original oracle", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
