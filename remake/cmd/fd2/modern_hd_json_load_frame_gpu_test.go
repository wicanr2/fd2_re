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

type jsonLoadFrameProbe struct {
	t     *testing.T
	g     *Game
	set   *modernStoryPortraitSet
	index int
	err   error
	out   string
	final *ebiten.Image
	rows  []map[string]any
}

func (p *jsonLoadFrameProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *jsonLoadFrameProbe) Update() error {
	if p.err != nil || p.index == 16 {
		return ebiten.Termination
	}
	return nil
}
func (p *jsonLoadFrameProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == 16 {
		return
	}
	g := p.g
	selection := p.index % 4
	ui := (p.index/4)%2 != 0
	art := p.index/8 != 0
	var set *modernStoryPortraitSet
	if art {
		set = p.set
	}
	if g.modernStoryPortraits != set {
		if p.err = g.applyTheme(set); p.err != nil {
			return
		}
	}
	mode := playerInterfaceOriginal
	if ui {
		mode = playerInterfaceHD
	}
	if p.err = g.setInterfaceMode(mode); p.err != nil {
		return
	}
	g.titleSlotSel = selection
	before := movementPanelGameplay(g)
	saved, err := os.ReadFile(saveSlotPath(0))
	if err != nil {
		p.err = err
		return
	}
	if _, ok := nativeLoadSlotMetadata(); ok {
		p.err = fmt.Errorf("JSON0 entered native metadata")
		return
	}
	g.Draw(screen)
	frame := g.modernHDText
	if ui {
		if frame == nil || len(frame.panels) != 1 || frame.panels[0].chrome != g.nativeLoadSlotsUI.hd.jsonChrome || frame.panels[0].rect != image.Rect(70, 35, 250, 160) {
			p.err = fmt.Errorf("JSON border missing/wrong")
			return
		}
	} else if frame != nil {
		p.err = fmt.Errorf("native interface retained HD")
		return
	}
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	presentation := &modernHDPresentation{Game: g}
	// Keep the actual existing native background and text; independently add
	// only the verified source edges/corners to a border-free final reference.
	if ui {
		frame.panels = nil
	}
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	want := make([]byte, 1920*1200*4)
	p.final.ReadPixels(want)
	if ui {
		frame.panels = []modernHDDialoguePanel{{chrome: g.nativeLoadSlotsUI.hd.jsonChrome, rect: image.Rect(70, 35, 250, 160)}}
		c := g.nativeLoadSlotsUI.hd.chrome
		for y := 0; y < 750; y++ {
			for x := 0; x < 1080; x++ {
				var ink color.RGBA
				switch {
				case x < 3 && y < 3:
					ink = c.corners[0][y*3+x]
				case x >= 1077 && y < 3:
					ink = c.corners[1][y*3+x-1077]
				case x < 3 && y >= 747:
					ink = c.corners[2][(y-747)*3+x]
				case x >= 1077 && y >= 747:
					ink = c.corners[3][(y-747)*3+x-1077]
				case y < 3:
					ink = c.edges[0][y]
				case x >= 1077:
					ink = c.edges[1][x-1077]
				case y >= 747:
					ink = c.edges[2][y-747]
				case x < 3:
					ink = c.edges[3][x]
				default:
					continue
				}
				at := ((210+y)*1920 + 420 + x) * 4
				copy(want[at:at+4], []byte{ink.R, ink.G, ink.B, ink.A})
			}
		}
	}
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	actual := make([]byte, len(want))
	p.final.ReadPixels(actual)
	if !bytes.Equal(actual, want) {
		for i, v := range actual {
			if v != want[i] {
				p.err = fmt.Errorf("case%d composition differs byte%d: %d/%d", p.index, i, v, want[i])
				break
			}
		}
		return
	}
	after, err := os.ReadFile(saveSlotPath(0))
	if err != nil || !bytes.Equal(saved, after) || !bytes.Equal(before, movementPanelGameplay(g)) {
		p.err = fmt.Errorf("Draw changed source/gameplay: %v", err)
		return
	}
	name := ""
	if selection == 0 {
		name = fmt.Sprintf("json-art%t-ui%t.png", art, ui)
		helper := &movementGPUProbe{out: p.out}
		helper.writePNG(name, actual)
		if helper.err != nil {
			p.err = helper.err
			return
		}
	}
	p.rows = append(p.rows, map[string]any{"selection": selection, "interface_hd": ui, "art_hd": art, "rgba_tolerance": 0, "interior_and_outside_unchanged": true, "save_bytes_equal": true, "file": name})
	p.index++
}
func TestModernHDJSONLoadFrameRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_JSON_LOAD_GPU") != "1" {
		t.Skip("dedicated JSON fallback GPU process")
	}
	g, _ := movementPanelTestGame(t, 160)
	g.clearNativePlayerMovement()
	g.sel = nil
	g.titlePhase = "loadslots"
	g.titleAssets = titleInterfaceTestAssets(t)
	a, err := loadNativeLoadSlotsUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeLoadSlotsUI = a
	out := os.Getenv("FD2_HD_JSON_LOAD_GPU_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	set := modifierHDTestTheme(t)
	g.modernThemeCache = set
	p := &jsonLoadFrameProbe{t: t, g: g, set: set, out: out, final: ebiten.NewImage(1920, 1200)}
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if p.index != 16 {
		t.Fatal("incomplete matrix")
	}
	code, count := hdConsumerSourceTree(t)
	report := map[string]any{"status": "PASS-JSON-LOAD-FRAME-GPU", "cases": p.index, "game_draws": p.index, "rows": p.rows, "source_tree_sha256": code, "source_file_count": count, "scope": "actual production JSON0 fallback Draw and final composition; independently add verified source edges/corners to border-free ordinary background/text; no new original oracle or keyboard route"}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "json-load-gpu-report.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Log("16 actual JSON fallback Draw; four slots/four controls; only thin border changes final pixels, tolerance0")
}
