package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestModernHDFacilitySuccessRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FACILITY_SUCCESS_GPU") != "1" {
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
	if len(set.facilitySuccess) != 4 {
		t.Fatal("four reaction banks required")
	}
	g := facilityDialogueTestGame(t, set)
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}, captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	composites := map[[2]int]*modernFacilityScene{}
	var job *nativeClassUIJob
	var selected, style int
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		step := job.timeline[selected]
		s := step.facilitySuccess
		if s == nil {
			return fmt.Errorf("step unbound")
		}
		d := s.presentation
		ui, art := style%2 == 1, style >= 2
		if style == 0 {
			if d.lastDraw != nil {
				return fmt.Errorf("native control retained HD observation")
			}
		} else if d.lastDraw == nil || d.lastDraw.frame != selected || d.lastDraw.interfaceHD != ui || d.lastDraw.artHD != art {
			return fmt.Errorf("actual successful Draw absent resource%d step%d style%d", d.resource, selected, style)
		}
		count, _, _, offset, _ := facilitySuccessShape(d.resource)
		pos := 0
		if s.stage == -2 {
			pos = count + 1
		} else if s.stage >= 0 {
			pos = s.stage + 1
		}
		frame := d.raw[pos]
		var runs []modernHDTextRun
		if ui {
			frame = d.hd[pos]
			runs = append(runs, d.source.runs...)
		}
		if job.goldText != nil {
			if ui {
				filtered := runs[:0]
				for _, run := range runs {
					if run.x >= 32 && run.x < 128 && run.y >= 190 && run.y < 216 {
						continue
					}
					filtered = append(filtered, run)
				}
				runs = filtered
				var roll []modernHDTextRun
				var ok bool
				frame, roll, ok = job.goldText.compose(frame, selected, g.font, step.palette)
				if !ok {
					return fmt.Errorf("reference gold window absent")
				}
				runs = append(runs, roll...)
			} else {
				frame = step.frame
			}
		}
		for i := range runs {
			// Independently apply the published additive RGB approximation.
			ink := runs[i].color
			add := s.delta * 255 / 63
			channel := func(v uint8) uint8 {
				n := int(v) + add
				if n > 255 {
					n = 255
				}
				return uint8(n)
			}
			ink.R, ink.G, ink.B = channel(ink.R), channel(ink.G), channel(ink.B)
			runs[i].color = ink
		}
		if ui && !reflect.DeepEqual(g.modernHDText.runs, runs) {
			return fmt.Errorf("HD text/window or flash differs resource%d step%d style%d", d.resource, selected, style)
		}
		oldText, oldScene := g.modernHDText, g.modernHDScene
		defer func() { g.modernHDText, g.modernHDScene = oldText, oldScene }()
		referenceGame := ebiten.NewImage(640, 400)
		defer referenceGame.Dispose()
		g.modernHDText = nil
		if ui {
			g.modernHDText = &modernHDTextFrame{scene: referenceGame}
		}
		g.modernHDScene = nil
		if art {
			key := [2]int{d.resource, s.stage}
			scene := composites[key]
			if scene == nil {
				// Independent 1920x1200 source assembly: no production patch painter.
				canvas := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
				draw.Draw(canvas, canvas.Bounds(), set.facilityScenes[d.resource].image, image.Point{}, draw.Src)
				if s.stage >= 0 {
					x, y := offset%320*6, offset/320*6
					draw.Draw(canvas, image.Rect(x, y, x+480, y+480), set.facilitySuccess[d.resource].frames[s.stage].hdImage, image.Point{}, draw.Src)
				}
				scene = &modernFacilityScene{image: canvas}
				composites[key] = scene
			}
			if !g.presentModernFacilityFrame(referenceGame, frame, d.source.mask, step.palette, &modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 1920, 1200), whiteDelta: s.delta}) {
				return fmt.Errorf("reference source absent")
			}
		} else {
			g.presentNativeClassFrameWithPalette(referenceGame, frame, step.palette)
		}
		if ui {
			g.modernHDText.runs = runs
			if d.resource == 14 {
				g.modernHDText.panels = []modernHDDialoguePanel{{chrome: d.chrome, rect: image.Rect(5, 112, 315, 198), gold: true, whiteDelta: s.delta}}
			}
		}
		referenceOffscreen := ebiten.NewImage(640, 424)
		defer referenceOffscreen.Dispose()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, 24)
		referenceOffscreen.DrawImage(referenceGame, op)
		final := ebiten.NewImage(1920, 1272)
		defer final.Dispose()
		var geometry ebiten.GeoM
		geometry.Scale(3, 3)
		p.presentation.DrawFinalScreen(final, referenceOffscreen, geometry)
		want := make([]byte, 1920*1272*4)
		final.ReadPixels(want)
		got, want := pixels[72*1920*4:], want[72*1920*4:]
		if !bytes.Equal(got, want) {
			for name, data := range map[string][]byte{"actual": got, "expected": want} {
				f, err := os.Create(filepath.Join(out, "mismatch-"+name+".png"))
				if err == nil {
					_ = png.Encode(f, &image.RGBA{Pix: data, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
					_ = f.Close()
				}
			}
			for i, value := range want {
				if value != got[i] {
					return fmt.Errorf("resource%d step%d stage%d delta%d style%d pixel(%d,%d)c%d %d != %d", d.resource, selected, s.stage, s.delta, style, i/4%1920, i/4/1920, i%4, got[i], value)
				}
			}
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "resource": d.resource, "timeline_step": selected, "animation_entry": s.stage, "white_delta": s.delta, "style": style, "rolling": job.goldText != nil, "whole_game_rgba_equal": true, "rgba_tolerance": 0, "whole_game_rgba_bytes": len(want)})
		return nil
	}
	add := func(resource, index, control int, kind string, capture bool) {
		if capture {
			p.captureCases[len(p.cases)] = true
		}
		p.cases = append(p.cases, func() {
			if kind == "success" {
				job = facilitySuccessTestJob(t, g, resource)
			} else {
				job = facilitySuccessGoldTestJob(t, g, resource, kind == "credit")
			}
			selected = index
			if selected < 0 {
				selected = len(job.timeline) - 1
			}
			style = control
			job.elapsed = 0
			for i := 0; i < selected; i++ {
				job.elapsed += job.timeline[i].duration
			}
			if job.timeline[selected].duration > 0 {
				job.elapsed += time.Nanosecond
			}
			g.interfaceMode, g.modernStoryPortraits = playerInterfaceOriginal, nil
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
			if style >= 2 {
				g.modernStoryPortraits = set
			}
		})
	}
	for _, resource := range []int{12, 29, 63, 14} {
		counts := map[int]int{12: 6, 29: 3, 63: 8, 14: 76}
		for i := 0; i < counts[resource]; i++ {
			add(resource, i, 3, "success", i == 1 || i == 2 || i == 8 || i == 24 || i == 40 || i == counts[resource]-1)
		}
		indices := []int{0, 1, 2, counts[resource] - 1}
		if resource == 29 {
			indices = []int{0, 1, 2}
		}
		if resource == 14 {
			indices = []int{0, 8, 9, 24, 40, 41, 72, 74, 75}
		}
		for control := 0; control < 3; control++ {
			for _, i := range indices {
				add(resource, i, control, "success", i == 1 || resource == 14 && i == 24)
			}
		}
	}
	for _, resource := range []int{12, 29, 63} {
		for _, kind := range []string{"debit", "credit"} {
			for control := 0; control < 4; control++ {
				for _, i := range []int{1, -1} {
					add(resource, i, control, kind, control == 3)
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
	if len(p.checks) != expected {
		t.Fatal("incomplete matrix", len(p.checks), expected)
	}
	report := map[string]any{"status": "PASS", "scope": "typed fixture; actual Draw and DrawFinalScreen; not a new original oracle or normal campaign entry", "checks": p.checks, "rgba_tolerance": 0, "frames": len(p.checks)}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "success-gpu-report.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("actual complete Draw frames=%d; full scene source assembly identical", len(p.checks))
}
