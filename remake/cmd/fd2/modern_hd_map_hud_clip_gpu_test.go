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
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type hudPanelClipProbe struct {
	g        *Game
	frames   []*modernHDMapFrame
	index    int
	previous []byte
	out      string
	err      error
}

func (p *hudPanelClipProbe) Update() error {
	if p.err != nil || p.index == len(p.frames) {
		return ebiten.Termination
	}
	return nil
}
func (p *hudPanelClipProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *hudPanelClipProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == len(p.frames) {
		return
	}
	f := p.frames[p.index]
	if !p.g.drawPreparedModernHDMapFrame(screen, f, false) {
		p.err = fmt.Errorf("HUD clip frame rejected")
		return
	}
	final := ebiten.NewImage(1920, 1200)
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	(&modernHDPresentation{Game: p.g}).DrawFinalScreen(final, screen, geometry)
	pixels := make([]byte, 1920*1200*4)
	final.ReadPixels(pixels)
	r := p.frames[1].HUDPanel.Rect
	if p.index == 0 {
		p.previous = append([]byte(nil), pixels...)
	} else {
		for y := 0; y < 1200; y++ {
			for x := 0; x < 1920; x++ {
				if image.Pt(x, y).In(r) {
					continue
				}
				i := (y*1920 + x) * 4
				if !bytes.Equal(pixels[i:i+4], p.previous[i:i+4]) {
					p.err = fmt.Errorf("HUD changed outside rectangle at %d,%d", x, y)
					return
				}
			}
		}
	}
	pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	copy(pic.Pix, pixels)
	file, err := os.Create(filepath.Join(p.out, []string{"native-control.png", "hd-panel.png"}[p.index]))
	if err != nil {
		p.err = err
		return
	}
	err = png.Encode(file, pic)
	closeErr := file.Close()
	final.Dispose()
	if err != nil {
		p.err = err
		return
	}
	if closeErr != nil {
		p.err = closeErr
		return
	}
	p.index++
}

func TestModernHDHUDPanelClipsFinalGPU(t *testing.T) {
	if os.Getenv("FD2_HD_HUD_CLIP_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	g, in := hudPanelTestSource(t)
	palette := g.nativeMapAssets.Palette
	unit, err := g.prepareModernHDMapHUDUnit(in, palette)
	if err != nil {
		t.Fatal(err)
	}
	panel, err := g.prepareModernHDMapHUDPanel(in, palette)
	if err != nil {
		t.Fatal(err)
	}
	base := image.NewRGBA(image.Rect(0, 0, 320, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			base.SetRGBA(x, y, color.RGBA{byte(x), byte(y), byte(x ^ y), 255})
		}
	}
	pixels, err := modernHDOverwriteLayer(456*200, palette, func(dst []byte) error {
		return indexedmap.BlitNativeMapHUD(in.Frames, in.HUDTerrain, in.HUDUnits, in.HUDCache, dst, in.HUD)
	})
	if err != nil {
		t.Fatal(err)
	}
	ui, err := modernHDNativeViewportPixels(pixels, 0)
	if err != nil {
		t.Fatal(err)
	}
	// The HUD uses absolute work row157; no terrain viewport offset is added.
	clearModernHDHUDUnitPixels(ui, unit)
	native := &modernHDMapFrame{Background: base, UI: ui, HUDUnit: unit}
	hd := *native
	hd.UI = image.NewRGBA(ui.Bounds())
	copy(hd.UI.Pix, ui.Pix)
	hd.HUDPanel = panel
	clearModernHDHUDPanelPixels(hd.UI, panel)
	out := os.Getenv("FD2_HD_HUD_OUT")
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &hudPanelClipProbe{g: g, frames: []*modernHDMapFrame{native, &hd}, out: out}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 2 {
		t.Fatal("HUD clip", p.index, p.err)
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-HUD-CLIP-GPU", "rect": panel.Rect, "unchanged_pixels": 1920*1200 - 414*204,
		"scope": "same-current-source synthetic geometry control, not original parity or normal player"}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "clip.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
