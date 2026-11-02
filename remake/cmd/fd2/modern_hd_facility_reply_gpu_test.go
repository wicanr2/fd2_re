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

// Render complete, unmoved source/target using pre-R62 text and gold controls.
// The expected result then copies native rectangles, never the new composer.
func facilityReplyStaticReference(t *testing.T, g *Game, owner string, target bool) []byte {
	t.Helper()
	shopJob, classJob := g.nativeShopUIJob, g.nativeClassUIJob
	g.nativeShopUIJob, g.nativeClassUIJob = nil, nil
	defer func() { g.nativeShopUIJob, g.nativeClassUIJob = shopJob, classJob }()
	source, compose := facilityReplyTestProducers(g, owner)
	if !target {
		compose = source
	}
	var a, b []byte
	var oka, okb bool
	if owner == "shop" {
		a, oka = prototypeShopTransitionMarker(g, 0, compose)
		b, okb = prototypeShopTransitionMarker(g, 255, compose)
	} else {
		a, oka = g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 0, compose)
		b, okb = g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 255, compose)
	}
	if !oka || !okb {
		t.Fatal("static reply writer absent")
	}
	mask := make([]byte, 64000)
	for i := range mask {
		if a[i] == b[i] {
			mask[i] = 255
		}
	}
	logical := ebiten.NewImage(640, 400)
	defer logical.Dispose()
	final := ebiten.NewImage(1920, 1200)
	defer final.Dispose()
	p := &modernHDPresentation{Game: g}
	defer func() {
		if p.textSurface != nil {
			p.textSurface.Dispose()
		}
	}()
	g.modernHDText, g.modernHDScene, g.modernHDOverlay = nil, nil, nil
	g.preparationRosterHD, g.facilityRosterHD, g.facilityListRosterHD = nil, nil, nil
	g.modernHDPanelPortraits = nil
	finish := g.beginModernHDText(logical)
	frame, ok := compose()
	var runs []modernHDTextRun
	if g.modernHDText != nil {
		if owner == "shop" {
			frame, runs, ok = prototypeShopTransitionCapture(g, compose)
		} else {
			frame, runs, ok = churchListPrototypeCapture(g, compose)
		}
	}
	if !ok {
		t.Fatal("static reply producer absent")
	}
	if g.modernStoryPortraits != nil {
		resource := 14
		if owner == "shop" {
			a, _, _, valid := g.nativeShopState()
			if !valid {
				t.Fatal("static shop absent")
			}
			resource = a.ResourceID
		}
		scene := g.modernFacility(resource)
		if scene == nil || !g.presentModernFacilityFrame(logical, frame, mask, g.nativeClassUI.palette, &modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 1920, 1200)}) {
			t.Fatal("static scene absent")
		}
	} else {
		g.presentNativeClassFrame(logical, frame)
	}
	if g.modernHDText != nil {
		g.modernHDText.runs = append(g.modernHDText.runs, runs...)
		g.appendModernHDFacilityDialogue(frame)
		if len(g.modernHDText.panels) != 1 || !g.modernHDText.panels[0].gold {
			t.Fatal("static gold frame absent", owner, target)
		}
	}
	finish()
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.DrawFinalScreen(final, logical, geometry)
	pixels := make([]byte, 1920*1200*4)
	final.ReadPixels(pixels)
	return pixels
}
func TestModernHDFacilityReplyRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FACILITY_REPLY_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	owner := os.Getenv("FD2_HD_FACILITY_REPLY_OWNER")
	if owner != "shop" && owner != "church" {
		t.Fatal("explicit owner required")
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
	cache := make(map[string][2][]byte)
	group := ""
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		job := g.activeFacilityListJob(owner)
		d := job.churchConfirmation
		cacheGroup := fmt.Sprintf("%d/%s/%s", d.variant, d.mode, g.localeID)
		if group != cacheGroup {
			group = cacheGroup
			cache = make(map[string][2][]byte)
		}
		key := fmt.Sprintf("%s/%t", g.interfaceMode, g.modernStoryPortraits != nil)
		controls, known := cache[key]
		if !known {
			controls = [2][]byte{facilityReplyStaticReference(t, g, owner, false), facilityReplyStaticReference(t, g, owner, true)}
			cache[key] = controls
		}
		want := append([]byte(nil), controls[0]...)
		phase := "restore"
		if job.frame < len(job.frames) {
			phase = "dialogue"
			shift := (job.frame + 1) * 13
			if d.paneOpening {
				shift = (5 - job.frame) * 13
			}
			for y := (112 + shift) * 6; y < min(198+shift, 200)*6; y++ {
				from, to := ((y-shift*6)*1920+30)*4, (y*1920+30)*4
				copy(want[to:to+1860*4], controls[1][from:from+1860*4])
			}
		}
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
			for i, value := range want {
				if got[i] != value {
					return fmt.Errorf("case%d %s/%s/%s frame%d %s art=%t pixel(%d,%d)c%d %d != %d", p.index, owner, d.mode, d.kind, job.frame, g.interfaceMode, g.modernStoryPortraits != nil, i/4%1920, i/4/1920, i%4, got[i], value)
				}
			}
		}
		hdDraw := g.interfaceMode == playerInterfaceHD || g.modernStoryPortraits != nil
		if hdDraw {
			if d.lastDraw == nil || d.lastDraw.frame != job.frame || d.lastDraw.phase != phase || d.lastDraw.interfaceHD != (g.interfaceMode == playerInterfaceHD) {
				return fmt.Errorf("case%d actual reply Draw absent", p.index)
			}
		} else if d.lastDraw != nil {
			return fmt.Errorf("original published HD Draw")
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "owner": owner, "mode": d.mode, "variant": d.variant, "kind": d.kind, "frame": job.frame, "phase": phase, "locale": g.localeID, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "whole_game_rgba_bytes": len(want), "rgba_tolerance": 0, "independent_static_composite_equal": true})
		return nil
	}
	add := func(variant int, mode string, closing bool, locale string, index, style int, capture bool) {
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
			job := facilityReplyTestJob(t, g, owner, variant, mode, closing)
			job.frame = index
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
			if style >= 2 {
				g.modernStoryPortraits = set
			}
		})
	}
	modes, variant := facilityReplyShopModes, 1
	if owner == "church" {
		modes, variant = facilityReplyChurchModes, 0
	}
	for _, mode := range modes {
		for _, closing := range []bool{false, true} {
			if !closing && !facilityReplyTestOpening(mode) {
				continue
			}
			for index := 0; index < 6; index++ {
				for style := 0; style < 4; style++ {
					add(variant, mode, closing, "zh-Hant", index, style, style == 3 && (mode == "transfer_intro" || mode == "revive_empty" || index == 2))
				}
			}
		}
	}
	if owner == "shop" {
		for _, other := range []int{3, 5} {
			for _, closing := range []bool{false, true} {
				for index := 0; index < 6; index++ {
					for style := 0; style < 4; style++ {
						add(other, "transfer_intro", closing, "zh-Hant", index, style, style == 3 && index == 2)
					}
				}
			}
		}
	}
	for _, locale := range []string{"zh-Hans", "en", "ja"} {
		for _, mode := range modes {
			for _, closing := range []bool{false, true} {
				if !closing && !facilityReplyTestOpening(mode) {
					continue
				}
				add(variant, mode, closing, locale, 2, 3, false)
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
	if len(p.checks) != expected {
		t.Fatal("incomplete reply matrix", len(p.checks), expected)
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-FACILITY-REPLY-GPU", "owner": owner, "scope": "full Game.Draw, independent static source/target and literal native rectangle translation; all modes/all steps/four controls on first shop or church; other two shop variants transfer_intro full; other three locales representative HD steps; no new original oracle", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
