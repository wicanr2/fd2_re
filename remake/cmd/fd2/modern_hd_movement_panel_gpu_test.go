package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type movementGPUCase struct {
	x       int
	art, ui bool
	guard   string
}
type movementGPUProbe struct {
	t     *testing.T
	g     *Game
	set   *modernStoryPortraitSet
	cases []movementGPUCase
	index int
	err   error
	out   string
	final *ebiten.Image
	rows  []map[string]any
}

func (p *movementGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *movementGPUProbe) Update() error {
	if p.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *movementGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == len(p.cases) {
		return
	}
	c := p.cases[p.index]
	g := p.g
	var theme *modernStoryPortraitSet
	if c.art {
		theme = p.set
	}
	if g.modernStoryPortraits != theme {
		if p.err = g.applyTheme(theme); p.err != nil {
			return
		}
	}
	// Both source anchors come from the real native view and constructor.
	g.clearNativePlayerMovement()
	g.sel = g.st.Units[0]
	v := g.st.NativeMapViewState
	visible := 4
	if c.x == 9 {
		visible = 7
	}
	v.CameraX = v.CursorX - visible
	v.VisibleCursorX = visible
	if p.err = g.st.MaterializeNativeMapViewState(v); p.err != nil {
		return
	}
	g.syncNativeMapView()
	if p.err = g.prepareNativePlayerMovement(g.sel); p.err != nil {
		return
	}
	mode := playerInterfaceOriginal
	if c.ui {
		mode = playerInterfaceHD
	}
	if p.err = g.setInterfaceMode(mode); p.err != nil {
		return
	}
	g.helpVisible = c.guard == "help"
	g.nativeFullDACWhite = c.guard == "white"
	g.nativeFullDACBlack = c.guard == "black"
	if p.err = g.composeNativeMapFrame(); p.err != nil {
		return
	}
	before := movementPanelGameplay(g)
	plan, source, h := g.nativeMovePlan, g.nativeMovePanel, g.movementPanelHD
	saved, err := os.ReadFile(saveSlotPath(0))
	if err != nil {
		p.err = err
		return
	}
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	// Independent reference: ordinary world without the panel, then the
	// existing source-only CPU panel recipe. No new presenter builds this image.
	g.nativeMovePanel = nil
	g.Draw(screen)
	presentation := &modernHDPresentation{Game: g}
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	want := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	p.final.ReadPixels(want.Pix)
	g.nativeMovePanel = source
	g.Draw(screen)
	d := g.movementPanelDraw
	visiblePanel := c.guard == ""
	if visiblePanel && (d == nil || d.hd != c.ui || d.artHD != c.art || d.rect != image.Rect(c.x*2, 18, c.x*2+298, 102)) {
		p.err = fmt.Errorf("case%d wrong independent mode/anchor", p.index)
		return
	}
	if c.ui && visiblePanel {
		collector := []*modernBattlePanelText{}
		assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
		if err != nil {
			p.err = err
			return
		}
		if p.err = g.renderBattlePanelText(assets, h.record, make([]byte, 64000), 0, 0, &collector); p.err != nil {
			return
		}
		if len(collector) != 1 {
			p.err = fmt.Errorf("independent source collector unavailable")
			return
		}
		pic, _, err := collector[0].raster(g.nativeUIPalette)
		if err != nil {
			p.err = err
			return
		}
		draw.Draw(want, image.Rect(c.x*6, 54, c.x*6+894, 306), pic, image.Point{}, draw.Src)
	} else if !c.ui && visiblePanel {
		// Native UI stays byte-exact before the established final screen filter.
		pixels := make([]byte, 640*400*4)
		screen.ReadPixels(pixels)
		for y := 0; y < 84; y++ {
			for x := 0; x < 298; x++ {
				index := h.raw[(y/2)*149+x/2]
				r, gg, b, a := g.nativeUIPalette[index].RGBA()
				at := ((18+y)*640 + c.x*2 + x) * 4
				if !bytes.Equal(pixels[at:at+4], []byte{byte(r >> 8), byte(gg >> 8), byte(b >> 8), byte(a >> 8)}) {
					p.err = fmt.Errorf("case%d native pixels differ at %d,%d", p.index, x, y)
					return
				}
			}
		}
	}
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	actual := make([]byte, len(want.Pix))
	p.final.ReadPixels(actual)
	if c.ui || !visiblePanel {
		if !bytes.Equal(actual, want.Pix) {
			for i, v := range actual {
				if v != want.Pix[i] {
					p.err = fmt.Errorf("case%d final differs byte%d %d/%d guard=%s", p.index, i, v, want.Pix[i], c.guard)
					break
				}
			}
			p.writePNG("failure-actual.png", actual)
			p.writePNG("failure-expected.png", want.Pix)
			return
		}
	}
	after, err := os.ReadFile(saveSlotPath(0))
	if err != nil || !bytes.Equal(saved, after) || !bytes.Equal(before, movementPanelGameplay(g)) || g.nativeMovePlan != plan || g.nativeMovePanel != source || g.movementPanelHD != h {
		p.err = fmt.Errorf("Draw changed source, gameplay or save: %v", err)
		return
	}
	row := map[string]any{"x": c.x, "art_hd": c.art, "interface_hd": c.ui, "guard": c.guard, "rgba_tolerance": 0, "native_pixels_exact": !c.ui && visiblePanel, "full_final_exact": c.ui || !visiblePanel, "save_bytes_equal": true, "gameplay_unchanged": true}
	if visiblePanel {
		name := fmt.Sprintf("x%d-art%t-ui%t.png", c.x, c.art, c.ui)
		p.writePNG(name, actual)
		if p.err != nil {
			return
		}
		raw, _ := os.ReadFile(filepath.Join(p.out, name))
		row["file"], row["sha256"] = name, fmt.Sprintf("%x", sha256.Sum256(raw))
	}
	p.rows = append(p.rows, row)
	p.index++
}
func (p *movementGPUProbe) writePNG(name string, raw []byte) {
	f, err := os.Create(filepath.Join(p.out, name))
	if err != nil {
		p.err = err
		return
	}
	pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	copy(pic.Pix, raw)
	err = png.Encode(f, pic)
	f.Close()
	if err != nil {
		p.err = err
	}
}
func TestModernHDMovementPanelRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_MOVEMENT_PANEL_GPU") != "1" {
		t.Skip("dedicated actual movement GPU process")
	}
	g, _ := movementPanelTestGame(t, 160)
	out := os.Getenv("FD2_HD_MOVEMENT_PANEL_GPU_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	set := modifierHDTestTheme(t)
	g.modernThemeCache = set
	p := &movementGPUProbe{t: t, g: g, set: set, out: out, final: ebiten.NewImage(1920, 1200)}
	for _, x := range []int{9, 160} {
		for _, art := range []bool{false, true} {
			for _, ui := range []bool{false, true} {
				p.cases = append(p.cases, movementGPUCase{x: x, art: art, ui: ui})
			}
		}
	}
	for _, guard := range []string{"help", "white", "black"} {
		p.cases = append(p.cases, movementGPUCase{x: 160, art: true, ui: true, guard: guard})
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if p.index != len(p.cases) {
		t.Fatal("incomplete GPU matrix")
	}
	code, count := hdConsumerSourceTree(t)
	source, err := os.ReadFile(os.Getenv("FD2_HD_MOVEMENT_PANEL_SAVE"))
	if err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"status": "PASS-MOVEMENT-PANEL-GPU", "frames": p.index, "rows": p.rows, "source_tree_sha256": code, "source_file_count": count, "save_source_sha256": fmt.Sprintf("%x", sha256.Sum256(source)), "scope": "complete existing normal first-chapter save loaded by production loader; actual native movement constructor, Game.Draw and DrawFinalScreen; controlled view anchors and guards; no new original oracle or keyboard route"}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "movement-gpu-report.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("actual movement Draw=%d; two anchors/four controls; exact native pixels or full HD final; help/white/black kept hidden", p.index)
}
