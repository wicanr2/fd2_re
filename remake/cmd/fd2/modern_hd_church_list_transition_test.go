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

	"github.com/hajimehoshi/ebiten/v2"
)

func churchListTransitionTestJob(t *testing.T, g *Game, mode string, closing, scroll bool) *nativeClassUIJob {
	t.Helper()
	churchListPrototypeState(t, g, mode, scroll)
	g.nativeShopUI = nil
	if !churchListPrototypeBegin(g, closing) || g.nativeClassUIJob == nil || g.nativeClassUIJob.facilityList == nil {
		t.Fatal("church job did not bind", mode, closing, scroll)
	}
	return g.nativeClassUIJob
}

func TestModernHDChurchListTransitionSourceAndFallback(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	shopUI := g.nativeShopUI
	cases, frames, guards := 0, 0, 0
	for _, mode := range churchListPrototypeModes {
		for _, closing := range []bool{false, true} {
			for _, scroll := range []bool{false, true} {
				g.nativeShopUI = shopUI
				previous := &modernHDTextFrame{runs: []modernHDTextRun{{text: "previous-frame"}}}
				g.modernHDText = previous
				job := churchListTransitionTestJob(t, g, mode, closing, scroll)
				d := job.facilityList
				if d.owner != "church" || d.resource != 14 || d.mode != mode || d.closing != closing || g.activeFacilityListJob("church") != job || g.modernHDText != previous || len(previous.runs) != 1 || g.nativeShopUI != nil {
					t.Fatal("wrong church owner or leaked capture")
				}
				g.modernHDText = nil
				source, runs, ok := churchListPrototypeCapture(g, g.composeNativeChurchSourceFrame)
				if !ok || !bytes.Equal(source, d.sourceHD) || len(runs) != len(d.sourceRuns) || len(runs) == 0 {
					t.Fatal("source capture differs")
				}
				final, runs, ok := churchListPrototypeCapture(g, churchListPrototypeProducer(g))
				if !ok || !bytes.Equal(final, d.finalHD) || len(runs) != len(d.finalRuns) || len(runs) == 0 {
					t.Fatal("final capture differs")
				}
				want := 6
				if closing {
					want = 5
					if !bytes.Equal(job.restore, d.source) {
						t.Fatal("restore changed")
					}
				}
				if len(job.frames) != want || len(d.rawFrames) != want {
					t.Fatal("original phase count changed")
				}
				if mode == "transfer_item" {
					if d.itemRows == nil || d.roster != nil {
						t.Fatal("item list invented roster")
					}
				} else if d.roster == nil || d.itemRows != nil {
					t.Fatal("roster lost proof")
				}
				for i := range job.frames {
					if !bytes.Equal(job.frames[i], d.rawFrames[i]) {
						t.Fatal("original frame changed")
					}
					shift, r, ok := d.placement(i)
					step := 5 - i
					if closing {
						step = i + 1
					}
					if !ok || shift != 13*step || r.Min != image.Pt(5, 112+shift) || r.Max.Y > 200 {
						t.Fatal("geometry changed")
					}
					frames++
				}
				if _, r, ok := d.placement(want); ok != closing || ok && !r.Empty() {
					t.Fatal("restore retained list")
				}
				for _, fault := range []string{"mode", "owner", "job", "shop-job", "church-job", "locale", "font", "palette", "help", "transient", "black", "white", "missing-hd", "raw"} {
					font, palette := g.font, g.nativeClassUI.palette
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
					if fault == "locale" {
						g.localeID = "unbound"
					}
					if fault == "font" {
						g.font = nil
					}
					if fault == "palette" {
						g.nativeClassUI.palette = append(color.Palette(nil), palette...)
						g.nativeClassUI.palette[73] = nil
					}
					g.helpVisible, g.transientUI = fault == "help", fault == "transient"
					g.nativeFullDACBlack, g.nativeFullDACWhite = fault == "black", fault == "white"
					if fault == "missing-hd" {
						g.modernStoryPortraits = &modernStoryPortraitSet{}
					}
					if fault == "raw" {
						job.frames[0][0] ^= 1
					}
					g.facilityListRosterHD = &modernFacilityListRoster{job: job}
					if g.drawModernHDFacilityList(nil, job) || g.facilityListRosterHD != nil {
						t.Fatal("invalid church source retained presentation", mode, fault)
					}
					if fault == "raw" {
						job.frames[0][0] ^= 1
					}
					g.font, g.nativeClassUI.palette = font, palette
					g.churchMode = mode
					g.camp.Node().Type = "church"
					g.nativeClassUIJob = job
					g.nativeShopUIJob = nil
					g.nativeChurchUIJob = nil
					g.localeID = d.locale
					g.modernStoryPortraits = nil
					guards++
				}
				g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
				g.nativeClassUIJob = nil
				cases++
			}
		}
	}
	t.Logf("church source cases=%d original_frames=%d fault_guards=%d", cases, frames, guards)
}

func churchListStaticReference(t *testing.T, g *Game, sourceOnly bool) []byte {
	t.Helper()
	job, mode := g.nativeClassUIJob, g.churchMode
	g.nativeClassUIJob = nil
	defer func() { g.nativeClassUIJob, g.churchMode = job, mode }()
	p := &modernHDPresentation{Game: g}
	logical := ebiten.NewImage(640, 400)
	defer logical.Dispose()
	final := ebiten.NewImage(1920, 1200)
	defer final.Dispose()
	defer func() {
		if p.textSurface != nil {
			p.textSurface.Dispose()
		}
	}()
	if !sourceOnly {
		p.Draw(logical)
	} else {
		g.modernHDText = nil
		g.modernHDScene = nil
		g.modernHDOverlay = nil
		g.preparationRosterHD = nil
		g.facilityRosterHD = nil
		g.facilityListRosterHD = nil
		g.modernHDPanelPortraits = nil
		a, ok := g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 0, g.composeNativeChurchSourceFrame)
		if !ok {
			t.Fatal("independent source marker0 failed")
		}
		b, ok := g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 255, g.composeNativeChurchSourceFrame)
		if !ok {
			t.Fatal("independent source marker255 failed")
		}
		mask := make([]byte, 64000)
		for i := range mask {
			if a[i] == b[i] {
				mask[i] = 255
			}
		}
		finish := g.beginModernHDText(logical)
		var frame []byte
		var runs []modernHDTextRun
		if g.modernHDText != nil {
			frame, runs, ok = churchListPrototypeCapture(g, g.composeNativeChurchSourceFrame)
		} else {
			frame, ok = g.composeNativeChurchSourceFrame()
		}
		if !ok {
			t.Fatal("independent source absent")
		}
		if g.modernStoryPortraits != nil {
			scene := g.modernFacility(14)
			if scene == nil || !g.presentModernFacilityFrame(logical, frame, mask, g.nativeClassUI.palette, &modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 1920, 1200)}) {
				t.Fatal("independent HD source absent")
			}
		} else {
			g.presentNativeClassFrame(logical, frame)
		}
		if g.modernHDText != nil {
			g.modernHDText.runs = append(g.modernHDText.runs, runs...)
			if len(g.modernHDText.runs) == 0 {
				t.Fatal("independent source lost its captured text")
			}
		}
		g.churchMode = "menu" // Only select the existing gold chrome policy.
		g.appendModernHDFacilityDialogue(frame)
		g.churchMode = mode
		finish()
	}
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.DrawFinalScreen(final, logical, geometry)
	pixels := make([]byte, 1920*1200*4)
	final.ReadPixels(pixels)
	return pixels
}

func TestModernHDChurchListTransitionRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_CHURCH_LIST_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	mode := os.Getenv("FD2_TEST_FACILITY_LIST_KIND")
	switch mode {
	case "class", "status_roster", "transfer_source", "transfer_dest", "transfer_item", "revive":
	default:
		t.Fatal("explicit list kind required")
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
	fixtureShopUI := g.nativeShopUI
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}, captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	type controls struct{ source, final []byte }
	group := ""
	cache := make(map[string]controls)
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		job := g.nativeClassUIJob
		d := job.facilityList
		if d == nil {
			return fmt.Errorf("case%d missing source recipe", p.index)
		}
		start, cycle := facilityListTestWindow(d)
		cacheGroup := fmt.Sprintf("%d/%s/%d", d.variant, d.locale, start)
		if cacheGroup != group {
			group = cacheGroup
			cache = make(map[string]controls)
		}
		style := fmt.Sprintf("%s/%t", g.interfaceMode, g.modernStoryPortraits != nil)
		control, known := cache[style]
		if !known {
			control = controls{churchListStaticReference(t, g, true), churchListStaticReference(t, g, false)}
			cache[style] = control
			// Reference Draws own their temporary plans; restore this actual job.
			p.presentation.Draw(offscreen)
		}
		want := append([]byte(nil), control.source...)
		step := 5 - job.frame
		if d.closing {
			step = job.frame + 1
		}
		restore := d.closing && job.frame == 5
		if !restore {
			y0 := 112 + 13*step
			bottom := y0 + 86
			if bottom > 200 {
				bottom = 200
			}
			for y := y0 * 6; y < bottom*6; y++ {
				src := ((y-13*step*6)*1920 + 5*6) * 4
				dst := (y*1920 + 5*6) * 4
				copy(want[dst:dst+310*6*4], control.final[src:src+310*6*4])
			}
		}
		got := pixels[72*1920*4:]
		if !bytes.Equal(got, want) {
			for name, data := range map[string][]byte{"actual": got, "expected": want, "source-control": control.source, "final-control": control.final} {
				f, err := os.Create(filepath.Join(out, "mismatch-"+name+".png"))
				if err == nil {
					err = png.Encode(f, &image.RGBA{Pix: data, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
					f.Close()
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			for label, runs := range map[string][]modernHDTextRun{"frozen-source": d.sourceRuns, "frozen-final": d.finalRuns, "published": g.modernHDText.runs} {
				for _, run := range runs {
					if run.y < 224 {
						t.Logf("%s: text=%q xy=%.3f,%.3f pixels=%.3f clip=%v layout=%v offset=%v inkTop=%t userScaled=%t color=%v", label, run.text, run.x, run.y, run.pixels, run.clip, run.layoutSize, run.layoutOffset, run.inkTop, run.userScaled, run.color)
					}
				}
			}
			for i, value := range want {
				if got[i] != value {
					return fmt.Errorf("case%d %s closing=%t frame%d style%s whole-view pixel(%d,%d) channel%d: %d != independent stable composition %d", p.index, mode, d.closing, job.frame, style, (i/4)%1920, (i/4)/1920, i%4, got[i], value)
				}
			}
		}
		if plan := g.facilityListRosterHD; restore && plan != nil {
			return fmt.Errorf("restore retained moving Q layer")
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "variant": d.variant, "mode": mode, "closing": d.closing, "frame": job.frame, "restore": restore, "source_start": start, "source_cycle": cycle, "locale": d.locale, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "whole_game_rgba_bytes": len(want), "rgba_tolerance": 0, "independent_static_composite_equal": true})
		return nil
	}
	add := func(variant, style, index int, closing, scroll bool, locale string) {
		if style == 3 && (variant == 1 || index == 5) {
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
			// Bind while original art/UI are active, then change both controls
			// for this individual animation frame, including first HD activation.
			g.modernStoryPortraits, g.interfaceMode = nil, playerInterfaceOriginal
			g.nativeShopUI = fixtureShopUI
			job := churchListTransitionTestJob(t, g, mode, closing, scroll)
			job.frame = index
			job.drawn = false
			if style >= 2 {
				g.modernStoryPortraits = set
			}
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
		})
	}
	smoke := os.Getenv("FD2_TEST_FACILITY_LIST_SMOKE") == "1"
	if smoke {
		for _, state := range []struct {
			index   int
			closing bool
		}{{2, false}, {5, false}, {3, true}, {5, true}} {
			for style := 0; style < 4; style++ {
				add(1, style, state.index, state.closing, false, "zh-Hant")
			}
		}
	} else {
		for _, variant := range []int{1} {
			for _, closing := range []bool{false, true} {
				for index := 0; index < 6; index++ {
					for style := 0; style < 4; style++ {
						add(variant, style, index, closing, false, "zh-Hant")
					}
				}
			}
		}
		for _, closing := range []bool{false, true} {
			for index := 0; index < 6; index++ {
				for style := 0; style < 4; style++ {
					add(1, style, index, closing, true, "zh-Hant")
				}
			}
		}
		for _, locale := range []string{"zh-Hans", "en", "ja"} {
			for _, closing := range []bool{false, true} {
				for style := 0; style < 4; style++ {
					add(1, style, 2, closing, false, locale)
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
	wanted := 120
	if smoke {
		wanted = 16
	}
	if len(p.checks) != wanted {
		t.Fatal("incomplete transition matrix", len(p.checks), wanted)
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-CHURCH-LIST-TRANSITION-GPU", "mode": mode, "smoke_only": smoke, "scope": "complete Game.Draw compared byte-for-byte to independent untouched source plus translated ordinary stable list; all game pixels, four styles, all original opening/closing/restore steps; no new oracle claim", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
