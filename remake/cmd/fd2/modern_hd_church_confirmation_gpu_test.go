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
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// Independent control: existing original producers, literal markers and HD
// text/chrome/button renderers. No R60 frozen recipe or frame composer.
func churchConfirmationStaticReference(t *testing.T, g *Game, stage, kind string, index int) []byte {
	t.Helper()
	job, mode := g.nativeClassUIJob, g.churchMode
	g.nativeClassUIJob = nil
	defer func() { g.nativeClassUIJob, g.churchMode = job, mode }()
	compose := churchConfirmationPrototypeQuestion(g)
	if stage == "source" {
		compose = g.composeNativeChurchSourceFrame
	} else if stage == "dialogue" {
		compose = g.composeNativeChurchDialogueBase
	}
	choices := stage == "question"
	if choices {
		question := compose
		compose = func() ([]byte, bool) {
			frame, ok := question()
			if !ok {
				return nil, false
			}
			var frames [][]byte
			var err error
			if kind == "opening" {
				frames, err = campaign.NativeClassConfirmationOpeningFrames(frame, g.nativeClassUI.choices)
			} else {
				frames, err = campaign.NativeClassConfirmationClosingFrames(frame, g.nativeClassUI.choices)
			}
			if err != nil || index < 0 || index >= len(frames) {
				return nil, false
			}
			return frames[index], true
		}
	}
	a, okA := g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 0, compose)
	b, okB := g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 255, compose)
	if !okA || !okB {
		t.Fatal("independent confirmation writer absent")
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
		frame, runs, ok = churchListPrototypeCapture(g, compose)
	}
	if !ok {
		t.Fatal("independent confirmation frame absent")
	}
	if g.modernStoryPortraits != nil {
		scene := g.modernFacility(14)
		if scene == nil || !g.presentModernFacilityFrame(logical, frame, mask, g.nativeClassUI.palette, &modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 1920, 1200)}) {
			t.Fatal("independent confirmation scene absent")
		}
	} else {
		g.presentNativeClassFrame(logical, frame)
	}
	if g.modernHDText != nil {
		g.modernHDText.runs = append(g.modernHDText.runs, runs...)
		g.churchMode = "menu" // Existing gold frame policy without stable choices.
		g.appendModernHDFacilityDialogue(frame)
		g.churchMode = mode
		if len(g.modernHDText.runs) == 0 || len(g.modernHDText.panels) == 0 {
			t.Fatal("independent source lost text or gold frame", stage)
		}
		if choices {
			h, err := g.prepareModernHDFacilityDialogue()
			if err != nil {
				t.Fatal(err)
			}
			phase, at := fullRewardOpening, index+6
			if kind != "opening" {
				phase, at = fullRewardChoiceClosing, index
			}
			if err := g.appendModernHDConfirmation(h.buttons, phase, at, 0, 0); err != nil {
				t.Fatal(err)
			}
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

func TestModernHDChurchConfirmationRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_CHURCH_CONFIRMATION_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	mode := os.Getenv("FD2_TEST_CHURCH_CONFIRMATION_MODE")
	if mode != "class_confirm" && mode != "revive_confirm" {
		t.Fatal("explicit confirmation mode required")
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
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		job := g.nativeClassUIJob
		d := job.churchConfirmation
		var want []byte
		if job.frame < 4 {
			want = churchConfirmationStaticReference(t, g, "question", d.kind, job.frame)
		} else {
			key := fmt.Sprintf("%s/%s/%t", g.localeID, g.interfaceMode, g.modernStoryPortraits != nil)
			controls, known := cache[key]
			if !known {
				controls = [2][]byte{churchConfirmationStaticReference(t, g, "source", "", 0), churchConfirmationStaticReference(t, g, "dialogue", "", 0)}
				cache[key] = controls
			}
			want = append([]byte(nil), controls[0]...)
			if job.frame < 9 {
				shift := (job.frame - 3) * 13
				for y := (112 + shift) * 6; y < min(198+shift, 200)*6; y++ {
					from, to := ((y-shift*6)*1920+30)*4, (y*1920+30)*4
					copy(want[to:to+1860*4], controls[1][from:from+1860*4])
				}
			}
		}
		// Reference presentation owns temporary plans; restore this actual Draw.
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
					return fmt.Errorf("case%d %s/%s frame%d %s art=%t pixel(%d,%d) channel%d: %d != independent %d", p.index, mode, d.kind, job.frame, g.interfaceMode, g.modernStoryPortraits != nil, i/4%1920, i/4/1920, i%4, got[i], value)
				}
			}
		}
		phase := "choices"
		if job.frame >= 4 {
			phase = "dialogue"
		}
		if job.frame == 9 {
			phase = "restore"
		}
		hdDraw := g.interfaceMode == playerInterfaceHD || g.modernStoryPortraits != nil
		if hdDraw {
			if d.lastDraw == nil || d.lastDraw.frame != job.frame || d.lastDraw.phase != phase || d.lastDraw.interfaceHD != (g.interfaceMode == playerInterfaceHD) {
				return fmt.Errorf("case%d expected actual HD confirmation Draw", p.index)
			}
		} else if d.lastDraw != nil {
			return fmt.Errorf("case%d original controls published an HD Draw", p.index)
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "mode": mode, "kind": d.kind, "frame": job.frame, "phase": phase, "hd_draw_expected": hdDraw, "locale": g.localeID, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "whole_game_rgba_bytes": len(want), "rgba_tolerance": 0, "independent_static_composite_equal": true})
		return nil
	}
	add := func(kind, locale string, index, style int) {
		if locale == "zh-Hant" && style == 3 {
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
			job := churchConfirmationTestJob(t, g, mode, kind)
			job.frame = index
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
			if style >= 2 {
				g.modernStoryPortraits = set
			}
		})
	}
	kinds := []string{"opening", "closing"}
	if mode == "revive_confirm" {
		kinds = append(kinds, "choice_only")
	}
	for _, kind := range kinds {
		count := 4
		if kind == "closing" {
			count = 10
		}
		for index := 0; index < count; index++ {
			for style := 0; style < 4; style++ {
				add(kind, "zh-Hant", index, style)
			}
		}
		for _, locale := range []string{"zh-Hans", "en", "ja"} {
			indices := []int{2}
			if kind == "closing" {
				indices = append(indices, 6)
			}
			for _, index := range indices {
				for style := 0; style < 4; style++ {
					add(kind, locale, index, style)
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
	wanted := 92
	if mode == "revive_confirm" {
		wanted = 120
	}
	if len(p.checks) != wanted {
		t.Fatal("incomplete confirmation matrix", len(p.checks), wanted)
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-CHURCH-CONFIRMATION-GPU", "mode": mode, "scope": "complete Game.Draw against existing independent native producers plus HD text, gold frame and YES/NO renderers; native steps and restore, four styles and four locale states; no new original oracle claim", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
