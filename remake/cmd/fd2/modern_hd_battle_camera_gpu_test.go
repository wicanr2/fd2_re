package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type modernCameraGPUProbe struct {
	gpu    modernHDBattleGPU
	plan   modernHDBattlePlan
	index  int
	err    error
	pixels []byte
}

func (p *modernCameraGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *modernCameraGPUProbe) Update() error {
	if p.err != nil || p.index == 4 {
		return ebiten.Termination
	}
	return nil
}
func (p *modernCameraGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == 4 {
		return
	}
	q := []hdTextureQuality{hdTextureFull, hdTextureBalanced, hdTextureLight, hdTextureFull}[p.index]
	p.gpu.setQuality(q)
	if p.err = p.gpu.draw(p.plan); p.err != nil {
		return
	}
	readHDQualityWorldPixels(p.gpu.World, p.pixels)
	panels := []image.Rectangle{q.rasterRect(image.Rect(1026, 24, 1920, 276)), q.rasterRect(image.Rect(0, 924, 894, 1176))}
	bounds := p.gpu.World.Bounds()
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			i := (y*bounds.Dx() + x) * 4
			blue := p.pixels[i] < 10 && p.pixels[i+1] < 10 && p.pixels[i+2] > 240
			fixed := image.Pt(x, y).In(panels[0]) || image.Pt(x, y).In(panels[1])
			if blue != fixed {
				p.err = fmt.Errorf("%s moved panel or left ghost at %d,%d", q.name(), x, y)
				return
			}
		}
	}
	before := p.gpu.rendered
	if p.err = p.gpu.draw(p.plan); p.err != nil {
		return
	}
	if p.gpu.rendered != before || p.gpu.reused == 0 {
		p.err = fmt.Errorf("same camera failed reuse")
		return
	}
	changed := p.plan
	changed.CameraZoom = 1.035
	if p.err = p.gpu.draw(changed); p.err != nil {
		return
	}
	if p.gpu.rendered != before+1 {
		p.err = fmt.Errorf("camera matrix missing from reuse key")
		return
	}
	readHDQualityWorldPixels(p.gpu.World, p.pixels)
	snapshot := append([]byte(nil), p.pixels...)
	for _, zoom := range []float64{1.08, math.NaN(), math.Inf(1)} {
		bad := p.plan
		bad.CameraZoom = zoom
		if p.gpu.draw(bad) == nil {
			p.err = fmt.Errorf("invalid camera accepted")
			return
		}
		readHDQualityWorldPixels(p.gpu.World, p.pixels)
		if !bytes.Equal(snapshot, p.pixels) {
			p.err = fmt.Errorf("rejected camera changed published pixels")
			return
		}
	}
	p.index++
}

func TestModernBattleCameraGPUFixedPanelsAndCache(t *testing.T) {
	if os.Getenv("FD2_HD_CAMERA_GPU") != "1" {
		t.Skip("dedicated camera GPU process")
	}
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{0, 0, 0, 255}
	}
	palette[1] = color.RGBA{0, 0, 255, 255}
	base := make([]byte, 64000)
	sources := &modernHDBattleSources{Base: base, BaseScenery: []modernHDBattleSource{{Archive: "BG.DAT"}}}
	for _, at := range []image.Point{{171, 4}, {0, 154}} {
		f := figani.Frame{X: at.X, Y: at.Y, Width: 149, Height: 42}
		for y := f.Y; y < f.Y+f.Height; y++ {
			for x := f.X; x < f.X+f.Width; x++ {
				base[y*320+x] = 1
			}
		}
		sources.BaseScenery = append(sources.BaseScenery, modernHDBattleSource{Frame: f, PanelText: &modernBattlePanelText{native: f}})
	}
	clean, ready := modernBattleCameraBase(sources)
	if !ready {
		t.Fatal("camera fixture rejected")
	}
	plan := modernHDBattlePlan{Base: clean, Palette: palette, CameraZoom: 1.07}
	for _, tc := range []struct {
		key   string
		rect  image.Rectangle
		ink   color.RGBA
		fixed bool
	}{
		{"scene", image.Rect(0, 300, 1920, 1200), color.RGBA{0, 255, 0, 255}, false},
		{"actor", image.Rect(840, 500, 1040, 700), color.RGBA{255, 255, 0, 255}, false},
		{"panel-upper", image.Rect(1026, 24, 1920, 276), color.RGBA{0, 0, 255, 255}, true},
		{"panel-lower", image.Rect(0, 924, 894, 1176), color.RGBA{0, 0, 255, 255}, true},
	} {
		plan.Layers = append(plan.Layers, modernHDBattleLayer{Native: figani.Frame{Width: 2, Height: 2}, HD: mapReusePicture(12, 12, tc.ink), HDKey: tc.key, Rect: tc.rect, Fill: -1, ScreenFixed: tc.fixed})
	}
	p := &modernCameraGPUProbe{plan: plan, pixels: make([]byte, 1920*1200*4)}
	defer p.gpu.close()
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 4 {
		t.Fatal(p.index, p.err)
	}
}
